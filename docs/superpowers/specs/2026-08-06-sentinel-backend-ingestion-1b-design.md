# Slice 1b — REST Keşif + Token Enrichment (Tasarım)

**Tarih:** 2026-08-06
**Alt-proje:** 1 (Solana ingestion) · **Dilim:** 1b
**Önceki:** Slice 1a (gerçek-zaman ingestion + WS transport) CANLI + doğrulandı; sürekli akış Helius free-tier WS limitinde bloke (sağlayıcı, kod değil).
**Bağlam:** `docs/superpowers/specs/2026-08-04-...-platform-skeleton-design.md`, `docs/superpowers/plans/2026-08-05-sentinel-backend-ingestion-1a.md`, memory `sentinel-backend-program`.

---

## 1. Amaç ve gerekçe

1a, gerçek pump.fun/Raydium token'larını Helius WebSocket ile tespit edip DB'ye yazdı; ama Helius free-tier `logsSubscribe` sürekli teslimat yapmadığından liste **donuk** kaldı (~34 eski snapshot). Ayrıca `TokenRow`'un piyasa alanları (price/liquidity/vol5m/momentum/spark) 1a'da nötr bırakıldı.

1b iki sorunu birden çözer:

1. **Keşif tazeliği** — GeckoTerminal REST (`new_pools`) ile yeni token'ları WS'ten bağımsız keşfeder → liste sürekli tazelenir, WS blokörünü tamamen atlar.
2. **Enrichment** — GeckoTerminal piyasa verisiyle token'ların price/liquidity/vol5m/momentum/spark alanlarını gerçek doldurur.

**Dürüstlük ilkesi:** Yalnız ingestion/piyasa verisinin gerçekten bildiği alanlar doldurulur. Skorlar (creator/safety) Alt-proje 2'ye, holders + derin seri Token Detail'e (1c) bırakılır — sahte veri yok, "henüz yok" nötr kalır.

**Neden GeckoTerminal (tek birincil kaynak):** Keysiz, `new_pools` (birebir yeni-havuz keşfi) + çoklu-havuz batch (fiyat/likidite/hacim/priceChange) + OHLCV (1c serisi için) — keşif + enrichment + gelecekteki seriyi tek kaynaktan verir. Tek eksik: holders (→ Helius, 1c). DexScreener temiz yeni-pair keşfi ve zaman serisi vermediği için elendi; ikinci provider olarak OCP-hazır bırakılır.

## 2. Kapsam

### 2.1 Kapsam içi
- Yeni Go paketi `internal/market/`: `MarketProvider` arayüzü + `GeckoTerminalClient` + `Discoverer` + `Enricher`.
- Migration `0003`: `tokens` tablosuna `pool_address` + `spark` (JSON) kolonları.
- Store: `UpdateMarket` (yeni), `UpsertToken` (launchpad + pool_address yazacak şekilde genişletme), `RecentTokens` (spark oku + pool_address iç-döndür).
- `main.go` wiring: Discoverer + Enricher goroutine'leri (config-gated), graceful shutdown.
- Config: GeckoTerminal base URL + keşif/enrichment aralık ve limitleri (keysiz).
- Testler: client parse/filtre (fixture), Discoverer/Enricher (fake provider+store), store round-trip.

### 2.2 Kapsam dışı (ertelendi — sessiz düşürme yok)
- **Token Detail (`getToken`) + OHLCV serisi + Helius holders → Slice 1c.**
- **`getKpis`/`getRadar` (Overview) → Alt-proje 2** (skorlara bağlı; ingestion tek başına anlamlı KPI/radar üretemez — bkz §7).
- **DexScreener ikinci provider** — OCP-hazır, gerekince eklenir.
- **`signal` / `creatorScore` / `safetyScore`** → Alt-proje 2 (nötr kalır).
- Sürekli WS akışı için güvenilir sağlayıcı (Helius paid / Chainstack / QuickNode) — bağımsız operasyonel karar, ertelendi.

## 3. Dürüst alan eşlemesi (`TokenRow`)

| Alan | 1b durumu | Kaynak / gerekçe |
|---|---|---|
| `mint`, `name`, `symbol` | ✅ gerçek | GeckoTerminal `new_pools` base token |
| `launchpad` (tokens tablosu) | ✅ gerçek | GeckoTerminal `dex_id` → pump.fun/Raydium |
| `ageSeconds` | ✅ gerçek | pool `created_at` → `first_seen_ts` |
| `price`, `liquidity`, `vol5m` | ✅ gerçek | GeckoTerminal pool attributes |
| `momentum` | ✅ türetilmiş | h1 fiyat değişiminden 0-100 (saf fiyat aksiyonu, skor DEĞİL) |
| `spark` | ✅ birikimli | her enrichment döngüsünde fiyat örneği eklenir |
| `holders` | — boş (0 → "—") | Helius getTokenAccounts → 1c (Token Detail) |
| `creatorScore`, `safetyScore` | — nötr 0 | Alt-proje 2 |
| `signal` | — null | A2 / strateji |
| `watchlisted` | — false | kullanıcı durumu |

`FeedEvent` (keşif olayları): `symbol`/`mint`/`launchpad`/`dex`/`liquidity`/`tokenAgeSeconds`/`volume5m` gerçek; `type` = `new_mint` veya `pool_created`; `severity` = `info`; `creatorScore`/`riskLevel` nötr (A2). `first_swap`/`liquidity_added` sentezlenmez (on-chain detay gerekir → ertelendi).

## 4. Mimari (Yaklaşım A — iki odaklı poller, tek client)

### 4.1 `MarketProvider` arayüzü (DIP)
```go
type Pool struct {
    PoolAddr      string
    Mint          string   // base token adresi
    Name, Symbol  string
    Dex           string   // "pumpfun" | "raydium" | ...
    Price         float64  // USD
    LiquidityUSD  float64
    Vol5m         float64
    PriceChangeH1 float64  // yüzde
    CreatedAtUnix int64
}

type MarketProvider interface {
    NewPools(ctx context.Context) ([]Pool, error)
    PoolsByAddresses(ctx context.Context, poolAddrs []string) ([]Pool, error)
}
```
Tüketiciler (Discoverer/Enricher) yalnız bu arayüze bağımlı. Somut kaynak değiştirilebilir/genişletilebilir (OCP).

### 4.2 `GeckoTerminalClient` (somut impl)
- Keysiz HTTP. `GET /networks/solana/new_pools` ve `GET /networks/solana/pools/multi/{addr,addr,...}` (≤30 pool adresi/istek).
- JSON → `Pool` eşlemesi; `dex_id` pump.fun/Raydium değilse elenir (SENTINEL kaynak listesiyle uyum).
- Enjekte edilebilir base URL + `http.Client` (test/timeout).

### 4.3 `Discoverer` (SRP)
Ticker (`MARKET_DISCOVER_INTERVAL`, default 30s):
1. `NewPools` → pump.fun/Raydium filtrele.
2. Her yeni mint için `UpsertToken` (kimlik + launchpad + pool_address; `first_seen_ts` = pool created_at) + `InsertEvent` (`new_mint`/`pool_created`).
3. `tokens` snapshot + tekil event broadcast.
- `new_pools` zaten piyasa verisi taşıdığından keşifte ilk enrichment bedava uygulanır (aynı `UpdateMarket`).
- Dedup: `UpsertToken` mint-PK `ON CONFLICT` idempotent; event dedup mevcut `events.id` deseni.

### 4.4 `Enricher` (SRP)
Ticker (`MARKET_ENRICH_INTERVAL`, default 30s):
1. `RecentTokens(MARKET_ENRICH_LIMIT, default 60)` → pool adreslerini topla.
2. `PoolsByAddresses` batch (30'lu) → güncel piyasa verisi.
3. Her token: `UpdateMarket(price, liquidity, vol5m, momentum, spark')` — spark'a güncel fiyat eklenip son ~16 örnekle sınırlanır.
4. Güncelleme sonrası tam `RecentTokens` snapshot'ı `tokens` topic'ine broadcast (1a seam kontratı: `subscribeTokens` []TokenRow alır, listeyi DEĞİŞTİRİR).

### 4.5 Türetmeler
- **momentum** = `clamp(50 + priceChangeH1_pct × k, 0, 100)` (k ayarlanabilir sabit; 50 = yatay). Saf kısa-vade fiyat aksiyonu → dürüst, Alt-proje-2 skoru değil.
- **spark** = birikimli örnekleme: her Enricher döngüsünde güncel fiyat diziye eklenir, son ~16 ile sınırlı, DB'de JSON olarak saklanır (restart'ta korunur). Başta 1-2 nokta, zamanla dolar; sıfır ekstra API çağrısı. Gerçek tarihsel OHLCV serisi → 1c (tek-token, rate-limit uygun).

## 5. Veri katmanı

### 5.1 Migration `0003_add_token_market_columns.sql`
```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS pool_address TEXT NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS spark TEXT NOT NULL DEFAULT '';  -- JSON []float64
-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS spark;
ALTER TABLE tokens DROP COLUMN IF EXISTS pool_address;
```
`spark` Postgres array yerine JSON `TEXT` — `database/sql` + pgx stdlib sürücüsüyle sürücü-bağımsız, ekstra dependency yok.

### 5.2 Store (SRP: kimlik ≠ piyasa)
- **Yeni** `UpdateMarket(ctx, mint string, price, liquidity, vol5m, momentum float64, spark []float64) error` — yalnız piyasa sütunlarını `UPDATE ... WHERE mint=$1`; spark JSON marshal.
- **Genişletme** `UpsertToken` — `launchpad` ve `pool_address`'i gerçek yazacak (şu an `""` hardcoded); mint-PK `ON CONFLICT` kimlik alanlarını korur, piyasa alanlarına dokunmaz.
- **Genişletme** `RecentTokens` — `spark` kolonunu okuyup JSON parse eder (artık boş `[]` sabiti değil); pool_address'i iç kullanım (Enricher) için ayrı okur. `TokenRow` JSON şekli değişmez (frontend seam sabit).

## 6. Config, rate-limit, hata yönetimi

- **Env** (hepsi opsiyonel, makul default): `GECKOTERMINAL_BASE_URL`, `MARKET_DISCOVER_INTERVAL` (30s), `MARKET_ENRICH_INTERVAL` (30s), `MARKET_ENRICH_LIMIT` (60). API key YOK.
- **Rate-limit** (GeckoTerminal free ~30 req/dk): keşif ~2/dk (1 istek/döngü) + enrichment ~2-4/dk (60 token / 30'lu batch = 2 istek/döngü) → toplam <10/dk, güvenli marj. Aralık/limit config ile ayarlanır.
- **Etkinleştirme:** poller'lar config flag'iyle açılır; `main.go` mevcut Helius worker'ı bozmadan yanına ekler. GeckoTerminal keysiz olduğundan gate basit (varsayılan açık; kapatılabilir).
- **Hata:** sağlayıcı/parse hatası → logla + döngü devam (poller ölmez); kısmi/hatalı batch broadcast edilmez (1a deseni). Ticker-tabanlı olduğundan backoff gerekmez; art arda hata görünürlüğü için sayaç/log.

## 7. Neden Overview (getKpis/getRadar) 1b'de değil

- **getKpis** — 8 karttan yalnız `detected` (24s'te first_seen sayısı) ingestion'dan gerçek; `highconf`/`critical` safety/risk skoruna (A2), `signals` stratejiye (A4), `positions`/`realized`/`unrealized` trading'e (A5) bağlı. 1b'de 8 karttan 1-2'si dürüst dolabilir → yarım iş.
- **getRadar** — `radarFrom` eksenleri x=`creatorScore`, level=(creator+safety)/2 → A2 skorları 1b'de nötr 0 olduğundan radar dejenere (her token x=0, aynı risk). Yalnız y=momentum, z=liquidity gerçek.
- **Karar:** Overview, skorlar (A2) gelince gerçek yapılır; 1b'de mock kalır. Sessiz düşürme yok — bu doküman + memory + `progress.md`'de "A2'ye bağlı" işaretli.

## 8. Frontend etkisi

**Yok.** `getTokens`/`getEvents`/`subscribeTokens`/`subscribeEvents` 1a'da canlı; seam bu alanları zaten taşıyor. 1b yalnız veriyi zenginleştirir → kullanıcı canlıda daha zengin liste + akan feed görür. Değişiklik: yalnız backend + migration 0003 (deploy'da otomatik goose).

## 9. Test stratejisi

- **`GeckoTerminalClient`** — kayıtlı `new_pools` + `pools/multi` JSON fixture ile parse/dex-filtre/eşleme (ağsız).
- **`Discoverer`** — fake `MarketProvider` + fake store: yeni mint upsert (kimlik+pool+launchpad), event üretimi, dedup idempotency, broadcast.
- **`Enricher`** — fake provider+store: `UpdateMarket` çağrısı, momentum türetme, spark append+sınır, snapshot broadcast.
- **Store** — `UpdateMarket` + spark JSON round-trip; `RecentTokens` spark parse.
- **Canlı GeckoTerminal** — yalnız deploy'da doğrulanır (yerel key/DB yok — 1a deseni). Deploy sonrası `/api/tokens` gerçek price/liquidity/vol/momentum/spark + tazelenen liste beklenir.

## 10. Kabul kriterleri (deploy'da)

1. `GET /api/tokens` → token'larda gerçek `price`/`liquidity`/`vol5m` (0 değil) ve türetilmiş `momentum`; birkaç döngü sonra `spark` dolu.
2. Liste zamanla tazelenir — yeni pump.fun/Raydium token'ları `new_pools`'tan gelir (WS akmasa bile).
3. `GET /api/events` → keşfedilen token'lar için `new_mint`/`pool_created` feed olayları.
4. Skorlar dürüst nötr (creator/safety 0 → "—"), holders 0 → "—" (henüz yok).
5. Go build/vet/test + `-race` yeşil; frontend testleri değişmeden yeşil (frontend'e dokunulmadı).

## 11. Kullanıcı aksiyonları
- **Yok (yeni ücretli/harici bağımlılık gerektirmez)** — GeckoTerminal keysiz. Deploy Railway'e mevcut pipeline'la; migration 0003 goose ile otomatik uygulanır. (Helius key zaten Railway'de; 1b onu kullanmaz — holders 1c'de.)

---

**Yaşayan kayıt:** implementasyon sonrası `docs/progress.md` "Backend programı" + memory `sentinel-backend-program` güncellenecek. Sonraki: `writing-plans` → SDD.
