# Slice 1c — Token Detail (`getToken`) (Tasarım)

**Tarih:** 2026-08-06
**Alt-proje:** 1 (Solana ingestion) · **Dilim:** 1c (Alt-proje 1'in son kontrat dilimi)
**Önceki:** 1a (gerçek-zaman ingestion + WS) CANLI; 1b (REST keşif + enrichment) CANLI + doğrulandı (GeckoTerminal keysiz, `/api/tokens` gerçek price/liquidity/vol/momentum).
**Bağlam:** `docs/superpowers/specs/2026-08-06-...-1b-design.md`, memory `sentinel-backend-program`.

---

## 1. Amaç

Token Detay ekranını (`getToken` → `TokenDetail`) mock'tan gerçeğe çevirmek: piyasa başlığı (fiyat/değişim/marketCap/likidite/hacim) + fiyat & hacim grafiği (GeckoTerminal OHLCV) + holders (Helius). Bu, Alt-proje 1'in ingestion kontrat dilimini tamamlar.

**Dürüstlük ilkesi (1a/1b ile aynı):** Yalnız ingestion/market/holder verisinin gerçekten bildiği alanlar doldurulur. `TokenDetail`'in çoğu alanı (scores, davranış-metrikleri, risks) Alt-proje 2'ye (scoring) aittir → nötr placeholder ("henüz yok"), sahte veri yok.

## 2. Kapsam

### 2.1 Kapsam içi
- Yeni endpoint `GET /api/token/{mint}` → `store.TokenDetail` (frontend `TokenDetail` JSON'una birebir).
- `internal/market/`: `MarketProvider` arayüzü `OHLCV(...)` ile genişler; `Pool` struct'ı header alanlarıyla (`PriceChangeH24`, `MarketCapUSD`, `Vol24h`) genişler; `GeckoTerminalClient` bunları uygular.
- `HoldersProvider` arayüzü (DIP) + Helius implementasyonu (`getTokenAccounts`, sınırlı sayfalama).
- `TokenDetailService` (SRP): store + MarketProvider + HoldersProvider'ı birleştirip `TokenDetail` kurar (nötr placeholder'lar dahil).
- Store: `TokenDetailBase(ctx, mint)` (mint → kimlik + pool_address + first_seen).
- Frontend: `http.ts` `getToken` gerçek fetch + `LIVE_ENDPOINTS` += `getToken`.
- Config: OHLCV/holders/cache parametreleri (opsiyonel, makul default).

### 2.2 Kapsam dışı (ertelendi — sessiz düşürme yok)
- **`scores` (4 ScoreKey) + `risks` + davranış-metrikleri** (buyRatio/sellRatio/top10HolderPct/sniperPct/botActivityPct/uniqueBuyers/creatorHoldingPct) → **Alt-proje 2** (nötr placeholder kalır).
- **`series.liquidity` / `series.holders`** → tarihsel kaynak yok (OHLCV vermiyor, biz saklamıyoruz); ileride örnekleme birikimi. 1c'de boş `[]`.
- **Bilinmeyen-mint / pool_address'siz** için GeckoTerminal top-pool keşfi → YAGNI (dürüst 404).
- `getKpis`/`getRadar` (Overview) → Alt-proje 2 (1b'de zaten ertelendi).

## 3. Dürüst alan eşlemesi (`TokenDetail`)

| Alan | 1c durumu | Kaynak |
|---|---|---|
| `name`, `symbol`, `mint`, `ageSeconds` | ✅ gerçek | tokens tablosu (`TokenDetailBase`) |
| `price`, `liquidity` | ✅ gerçek | GeckoTerminal pool (mevcut) |
| `priceChange24h`, `marketCap`, `volume24h` | ✅ gerçek | GeckoTerminal pool (yeni map'lenen alanlar) |
| `series.price`, `series.volume` | ✅ gerçek | GeckoTerminal OHLCV (close, volume) |
| `metrics.holders` | ✅ gerçek | Helius `getTokenAccounts` (sınırlı, "N+") |
| `scores` (opportunity/creatorReputation/tokenSafety/manipulationRisk) | — nötr | Alt-proje 2 |
| `metrics` (holders hariç) | — 0 | Alt-proje 2 |
| `risks` (contract/market/creator) | — boş `[]` | Alt-proje 2 |
| `series.liquidity`, `series.holders` | — boş `[]` | kaynak yok → ertelendi |

**`marketCap` fallback:** GeckoTerminal `market_cap_usd` çoğu yeni token'da null → `fdv_usd`'ye düş.

## 4. Endpoint + akış

`GET /api/token/{mint}`:
1. `TokenDetailBase(mint)` → yoksa **404** (dürüst; liste dışı mint UI'da ulaşılmaz).
2. Cache kontrol (mint başına ~20s TTL) → varsa dön.
3. **Header:** `PoolsByAddresses([poolAddr])` (mevcut metot yeniden kullanılır; `Pool` artık h24 değişim/marketCap/vol24h taşır).
4. **Grafik:** yaşa-uyarlı `OHLCV`: `ageSeconds < 6h` → `minute` (agg 1, limit ~200); değilse `hour` (agg 1, limit ~168). close→`series.price`, volume→`series.volume`.
5. **Holders:** `HoldersCount(mint, cap)` — sınırlı sayfalama.
6. `TokenDetail` kur (nötr placeholder'lar §7), cache'e yaz, dön.
- Herhangi bir upstream (OHLCV/holders) hata verirse: o alt-alan boş/0 döner + loglanır; header başarılıysa detay yine döner (kısmi başarı, dürüst). Header başarısızsa 502/boş uygun.

## 5. Bileşenler (SOLID)

### 5.1 `MarketProvider` genişlemesi (OCP)
```go
type Candle struct { Ts int64; Close, Volume float64 }
type MarketProvider interface {
    NewPools(ctx) ([]Pool, error)                                   // mevcut
    PoolsByAddresses(ctx, poolAddrs []string) ([]Pool, error)       // mevcut (Pool genişledi)
    OHLCV(ctx, poolAddr, timeframe string, limit int) ([]Candle, error) // YENİ
}
```
`Pool` struct'ına: `PriceChangeH24 float64`, `MarketCapUSD float64`, `Vol24h float64` (GeckoTerminal `price_change_percentage.h24`, `market_cap_usd`|`fdv_usd`, `volume_usd.h24`). `GeckoTerminalClient.OHLCV` → `GET /networks/solana/pools/{pool}/ohlcv/{timeframe}?limit=...&aggregate=1`; yanıt `data.attributes.ohlcv_list` = `[[ts,o,h,l,c,v],...]`.

### 5.2 `HoldersProvider` (DIP)
```go
type HoldersProvider interface {
    HoldersCount(ctx, mint string, cap int) (count int, capped bool, err error)
}
```
Helius impl: `getTokenAccounts` (mint'e göre, sayfalı `cursor`/`page`), `cap` (default ~5000) sayfaya kadar say; taşarsa `capped=true`. `internal/ingest/helius.go`'daki mevcut Helius client desenini izler (aynı RPC URL).

### 5.3 `TokenDetailService` (SRP, yeni — `internal/market/detail.go`)
`Build(ctx, mint) (store.TokenDetail, bool, error)` — store + MarketProvider + HoldersProvider'ı birleştirir, yaşa-uyarlı tf seçer, nötr placeholder üretir, mint başına TTL cache uygular. Enjekte edilebilir clock (test).

### 5.4 Store
`TokenDetailBase(ctx, mint string) (base TokenDetailBase, ok bool, err error)` — `TokenDetailBase{Name, Symbol, PoolAddr string; FirstSeenTs int64}`. `SELECT ... WHERE mint=$1`.

### 5.5 API + Frontend
- `internal/api`: `GET /api/token/{mint}` handler (chi URL param), 404/502 yolları.
- `apps/web/lib/api/http.ts`: `getToken: (mint) => getJson<TokenDetail>('/api/token/'+encodeURIComponent(mint))`.
- `apps/web/lib/api/live-endpoints.ts`: `LIVE_ENDPOINTS` += `"getToken"`.
- Seam: Go `store.TokenDetail` JSON tag'leri = TS `TokenDetail` birebir (Go tarafında yeni struct'lar: `TokenDetail`/`ScoreDetail`/`TokenMetrics`/`SeriesPoint`/`RiskGroups`).

## 6. Nötr placeholder (dürüst "henüz yok")
- `scores`: 4 ScoreKey (`opportunity`,`creatorReputation`,`tokenSafety`,`manipulationRisk`) → her biri `{value:0, confidence:0, updatedAt:"—", breakdown:[]}`. Frontend tüm anahtarları bekler → hepsi mevcut olmalı.
- `metrics`: `holders` gerçek; `uniqueBuyers/buyRatio/sellRatio/creatorHoldingPct/top10HolderPct/sniperPct/botActivityPct` = 0.
- `risks`: `{contract:[], market:[], creator:[]}`.
- `series.liquidity`, `series.holders`: `[]`.
- Frontend zaten nötr skoru dürüst gösteriyor (0 → "—", 1a'da eklendi) — detay ekranında da bu geçerli.

## 7. Cache & rate-limit
- getToken 3 upstream çağrı (pool + OHLCV + holders). **Mint başına ~20s TTL in-memory cache** (`sync.Map` + zaman damgası) → tekrar görüntüleme/çok izleyici GeckoTerminal (~30/dk) + Helius limitlerini yormaz.
- Config: `TOKEN_DETAIL_CACHE_SEC`(20), `OHLCV_LIMIT`(200), `HOLDERS_CAP`(5000), `TOKEN_DETAIL_AGE_MINUTE_THRESHOLD_SEC`(21600=6h) — hepsi opsiyonel default.

## 8. Test
- `GeckoTerminalClient.OHLCV` — fixture (`ohlcv_list`) parse; `Pool` header 3-alan parse (h24/marketCap/fdv fallback/vol24h).
- `HoldersCount` — fake Helius: tek sayfa, çok sayfa, cap/"N+" (capped=true).
- `TokenDetailService.Build` — fake provider'lar + fake store: alan eşleme, yaşa-uyarlı tf seçimi (genç→minute, yaşlı→hour), nötr placeholder bütünlüğü (4 ScoreKey mevcut, risks boş gruplar, series.liq/holders boş), cache (2. çağrı upstream'e gitmez), 404 (mint yok).
- API handler — 200 (var), 404 (yok).
- Frontend: `http.ts` getToken fetch + `LIVE_ENDPOINTS`; `TokenDetail` seam tip uyumu (mevcut frontend testleri yeşil kalır).
- Canlı GeckoTerminal OHLCV + Helius holders yalnız DEPLOY'da doğrulanır (yerel key/DB yok — 1a/1b deseni).

## 9. Kabul kriterleri (deploy'da)
1. `GET /api/token/{gerçek-mint}` → gerçek price/priceChange24h/marketCap/liquidity/volume24h.
2. `series.price` + `series.volume` dolu (yaşa-uyarlı granularite).
3. `metrics.holders` gerçek (veya cap'te "N+" mantığı — count + capped).
4. `scores`/`risks`/diğer metrics nötr (0/boş → UI "—").
5. Bilinmeyen mint → 404.
6. Go build/vet/`test -race` yeşil; frontend testleri yeşil; Token Detay ekranı canlı gerçek veri gösterir.

## 10. Kullanıcı aksiyonları
- **Yeni ücretli/harici bağımlılık yok** — GeckoTerminal keysiz; holders için **mevcut Helius key** (Railway'de zaten var, 1a'dan) kullanılır. Deploy Railway'e mevcut pipeline'la. Frontend değişikliği (getToken canlı) Vercel'e otomatik deploy (yeni commit push → cache'siz build).

---

**Yaşayan kayıt:** implementasyon sonrası `docs/progress.md` + memory `sentinel-backend-program`/`sentinel-deployment` güncellenecek. Sonraki: `writing-plans` → SDD. Alt-proje 1 (ingestion) 1c ile tamamlanır → sonraki Alt-proje 2 (scoring) scores/risks/Overview'ı açar.
