# Slice 2b-1 — Creator Capture + `getCreators`/`getCreator` (kimlik + sayımlar) (Tasarım)

- **Tarih:** 2026-08-10
- **Alt-proje:** 2 (Scoring & Graph) — dilim 2b (Creator Reputation), parça 1/2
- **Branch (planlanan):** `feat/backend-scoring-2b-1`
- **Önceki:** Slice 2a (Token Safety) canlı + doğrulandı (master `c848d61`)
- **Sonraki:** Slice 2b-2 (İtibar skoru + outcome tespiti + davranış)

## 1. Amaç

Pump.fun `CreateEvent` payload'unda **zaten mevcut** olan creator (dev) cüzdan adresini yakalamaya başlamak ve `getCreators` / `getCreator` endpoint'lerini kimlik + sayım düzeyinde gerçeğe çevirmek. Böylece **creator → token geçmişi hemen birikmeye başlar** — itibar skoru (2b-2) ancak bu geçmiş üzerine kurulabilir ve geçmiş zaman-değerlidir (ne kadar erken yakalarsak o kadar zengin olur).

Bu dilim **skor hesaplamaz**; yalnız veriyi yakalar, saklar ve kimlik + `totalTokens` + token geçmişini dürüstçe sunar. İtibar skoru, outcome tespiti ve davranış paternleri bilinçli olarak 2b-2'ye ertelenir (aşağıda açık işaretli — sessiz düşürme yok).

## 2. Kapsam

### 2.1 Kapsam içi

1. **Ingest — creator yakalama:** pump.fun `CreateEvent` decoder'ı creator pubkey'ini de çıkarır (şu an mint'te duruyor). `createEvent` struct'ı ve `Decoded.Token` creator taşır.
2. **Store — persist + agrega:**
   - Migration `0006`: `tokens.creator TEXT NOT NULL DEFAULT ''`.
   - `UpsertToken` creator'ı persist eder; boş creator gerçek creator'ı **ezmez** (merge kuralı, §4.3).
   - `UpsertDiscovered` (GeckoTerminal keşfi) aynı merge korumasını uygular (boş creator ile gelir).
   - Yeni dar store metotları: `Creators(limit)` (agrega liste) + `CreatorDetail(address)` (kimlik + token listesi).
3. **API:** `GET /api/creators` + `GET /api/creator/{address}` handler'ları (nil-guard, 404/200/502).
4. **Frontend:** `getCreators` / `getCreator` gerçek `httpApi` fetch + `LIVE_ENDPOINTS`'e eklenir. **UI'a sıfır dokunuş** (seam zaten alanları taşıyor — 1c deseni).

### 2.2 Kapsam dışı (ertelendi — sessiz düşürme yok)

- **İtibar skoru hesabı** (`reputationScore`, `reputation: ScoreDetail`, `riskLevel`) → 2b-2 (kural-tabanlı Go scorer).
- **Outcome tespiti** (`ruggedTokens`, `activeTokens`, `successRatePct`, per-token `outcome`) → 2b-2 (rug/success sinyali için likidite-çekme / fiyat-çöküş analizi gerekir).
- **PnL** (`realizedPnlSol`) → 2b-2 (creator satış akışı persist edilmiyor).
- **Davranış paternleri** (`behavior`: deployFrequency / avgFirstSellMinutes / repeatedFunders / similarMetadata / sameSocial / sameLiquidityPattern) → 2b-2.
- **Cüzdan yaşı** (`walletAgeDays`) → 2b-2 (creator cüzdanının ilk-tx lookup'ı gerekir).
- **Peak/drawdown takibi** (per-token `peakMarketCap` / `maxDrawdownPct`) → 2b-2 (zaman-serisi tepe izleme).
- **Ayrı `creators` tablosu** → YAGNI. 2b-1'de `tokens`'tan agrega yeter; 2b-2 hesaplanmış skorlar için ekleyebilir.
- **Creator etiketleme** (`label?`) → tanımlanmadı; boş bırakılır (undefined/omit).

## 3. Veri modeli — dürüst alan split

Store, frontend `CreatorRow` / `CreatorProfile` şekline birebir eşlenen Go struct'ları döndürür (mevcut `TokenRow` / `TokenDetail` deseni). Gerçek alanlar SQL'den; nötr alanlar Go zero-value ile doldurulur.

### `CreatorRow` (liste — `getCreators`)

| Alan | 2b-1 |
|---|---|
| `address` | **Gerçek** (yakalanan creator pubkey) |
| `totalTokens` | **Gerçek** (`COUNT(*)`) |
| `label?` | omit (undefined) |
| `reputationScore` | 0 (nötr → 2b-2) |
| `riskLevel` | `"medium"` (nötr placeholder → 2b-2) |
| `activeTokens` | 0 (nötr → 2b-2) |
| `ruggedTokens` | 0 (nötr → 2b-2) |
| `successRatePct` | 0 (nötr → 2b-2) |
| `realizedPnlSol` | 0 (nötr → 2b-2) |

### `CreatorProfile` (profil — `getCreator`)

| Alan | 2b-1 |
|---|---|
| `address` | **Gerçek** |
| `firstSeen` | **Gerçek** (`MIN(first_seen_ts)`, ISO string) |
| `metrics.totalTokens` | **Gerçek** (`COUNT(*)`) |
| `history[]` → `id/symbol/mint/createdAt` | **Gerçek** (tokens'tan) |
| `history[]` → `currentMarketCap` | **Gerçek** (`market_cap_usd`, migration 0004 kolonu) |
| `label?` | omit |
| `walletAgeDays` | 0 (nötr → 2b-2) |
| `reputation: ScoreDetail` | nötr (score 0, boş breakdown → 2b-2) |
| `riskLevel` | `"medium"` (nötr → 2b-2) |
| `metrics.*` (totalTokens hariç) | 0 (nötr → 2b-2) |
| `behavior.*` | nötr (boş/false → 2b-2) |
| `history[]` → `peakMarketCap/maxDrawdownPct/creatorSellPct/outcome/liquidityStatus/riskFlags` | nötr placeholder → 2b-2 |

`riskLevel` için nötr değer `"medium"` seçilir (mevcut skorsuz-token deseni: 1a event'leri de `RiskLevel: "medium"` kullanır) — sahte "düşük risk" sinyali vermemek için.

## 4. Bileşenler (SOLID)

### 4.1 Ingest — creator parse (`internal/ingest/decode_pumpfun.go`)

pump.fun `CreateEvent` borsh layout'u: `name(str) · symbol(str) · uri(str) · mint(32) · bondingCurve(32) · user/creator(32) · …`. Mevcut `tryParseCreateAt` 3 string sonrası `mint = raw[p:p+32]` okuyup duruyor. Genişletme:

- `p += 32` (mint'ten sonra bondingCurve'ü atla → `p += 32`), ardından `creator = raw[p:p+32]` (yani mint başlangıcından **+64** offset).
- **Uzunluk-guard:** `p+64 > len(raw)` ise `creator = ""` (dürüst boş; mint yine döner — mevcut mint-guard'ı bozmadan).
- `createEvent` struct'ına `creator string` eklenir; `Decoded.Token.Creator` set edilir; worker `UpsertToken`'a taşır.
- **Kalibrasyon riski (deploy'da):** byte offset (bondingCurve gerçekten 32B mi, `creator` mı yoksa `user` mı canlıda dev cüzdanıdır) — 1a Raydium / 2a amount hotfix deseni: canlı şekil farklıysa fixture + parse birlikte düzeltilir. Guard sayesinde yanlış offset en kötü boş creator verir, pipeline'ı bozmaz.

### 4.2 Store — şema + persist (`internal/store/`)

- **Migration `0006_add_token_creator.sql`:** `ALTER TABLE tokens ADD COLUMN creator TEXT NOT NULL DEFAULT '';` (+ down: `DROP COLUMN`). Agrega için `CREATE INDEX idx_tokens_creator ON tokens (creator);` (WHERE creator<>'' partial index — GROUP BY / filtre hızı).
- `TokenRow`'a `Creator string` **eklenmez** (frontend kontratında yok); creator ayrı parametre/kolon olarak taşınır (`firstSeenTs`'in `UpsertToken` imzasında ayrı taşınması deseni).
- `UpsertToken(ctx, t TokenRow, firstSeenTs int64, creator string)` — imzaya creator eklenir; persist eder.

### 4.3 Merge kuralı — gerçek creator'ı koru

İki yazar aynı token'a dokunur: Helius ingest (**gerçek** creator) ve GeckoTerminal `UpsertDiscovered` (**boş** creator, aggregator creator vermez). Hangisi önce gelirse gelsin gerçek creator kaybolmamalı (launchpad çapraz-yazar deseni, 1b'de çözülen):

```sql
-- UpsertToken ON CONFLICT:
creator = COALESCE(NULLIF(EXCLUDED.creator, ''), tokens.creator)
-- UpsertDiscovered ON CONFLICT: creator'a HİÇ dokunma (INSERT'te DEFAULT '' kalır,
-- conflict'te mevcut değer korunur — set listesine creator eklenmez).
```

Böylece: boş creator asla dolu creator'ı ezmez; iki yol hangi sırada gelirse gelsin sonuç gerçek creator.

### 4.4 Store — agrega okuma (dar metotlar, ISP)

`CreatorStore` arayüzü (DIP; fake + postgres implementasyonu):

- `Creators(ctx, limit) ([]CreatorRow, error)`:
  ```sql
  SELECT creator, COUNT(*) AS total, MIN(first_seen_ts) AS first_seen
  FROM tokens WHERE creator <> '' GROUP BY creator
  ORDER BY total DESC, first_seen ASC LIMIT $1
  ```
  Nötr alanlar Go'da zero/placeholder set edilir.
- `CreatorDetail(ctx, address) (CreatorProfile, bool, error)`:
  - Kimlik + agrega: `COUNT(*)`, `MIN(first_seen_ts)` (creator=address).
  - Token geçmişi: `SELECT mint, symbol, first_seen_ts, market_cap_usd FROM tokens WHERE creator=$1 ORDER BY first_seen_ts DESC`.
  - Hiç token yoksa `ok=false` → handler 404.

`CreatorStore`, mevcut `Bundle`'a eklenir (postgres) + `fake_ingest.go` fake'ine parity implementasyonu (test için).

### 4.5 API (`internal/api/`)

- `GET /api/creators` → `store.Creators(limit)`; JSON `CreatorRow[]`; 200. Limit config'ten (default 100).
- `GET /api/creator/{address}` → `store.CreatorDetail(address)`; bulunmazsa 404; store hatası 502; aksi 200 `CreatorProfile`.
- Nil-store guard (store nil ise 503/notReady — mevcut handler deseni).

### 4.6 Frontend (`apps/web/lib/api/`)

- `http.ts`: `getCreators()` → `GET /api/creators`; `getCreator(address)` → `GET /api/creator/{address}` (mevcut `getToken` fetch deseni).
- `live-endpoints.ts`: `LIVE_ENDPOINTS`'e `getCreators` + `getCreator` eklenir.
- **UI dokunulmaz** — Increment 5 (Creators) ekranları seam üzerinden okur; nötr alanlar mevcut mock'taki gibi görünür (dürüst placeholder).

## 5. Config

- `CREATORS_LIST_LIMIT` (default 100) — `/api/creators` satır sınırı.

Yeni env/secret/ücretli bağımlılık **YOK**. Creator verisi mevcut Helius ingest akışından gelir (ek RPC çağrısı yok); agrega saf SQL.

## 6. Test

- **Decoder:** creator içeren gerçek-şekilli `CreateEvent` fixture → `creator` doğru base58; kısa buffer → creator `""` + mint hâlâ doğru (guard).
- **Store parity:** `Creators` / `CreatorDetail` fake ↔ postgres aynı sonuç; merge kuralı testi (boş creator dolu olanı ezmez — her iki sıra); `CreatorDetail` bulunamazsa `ok=false`.
- **API handler:** `/api/creators` 200 + şekil; `/api/creator/{addr}` 200 / 404 / 502.
- **Frontend:** `getCreators` / `getCreator` http fetch + `LIVE_ENDPOINTS` üyeliği (mevcut http test deseni).
- Canlı Helius creator yakalama + DB round-trip yalnız **deploy'da** doğrulanır (yerel Postgres/key yok — Alt-proje 0-2a deseni).

## 7. Clean Code & SOLID

- **SRP:** decoder yalnız parse; store yalnız persist/agrega; handler yalnız HTTP; scorer bu dilimde YOK.
- **DIP:** frontend `getApi()` seam'den okur (mock import etmez); API `CreatorStore` arayüzüne bağımlı, somut postgres'e değil.
- **OCP:** yeni endpoint'ler mevcut kodu değiştirmeden eklenir (`LIVE_ENDPOINTS` genişletme, yeni handler); 2b-2 skoru bu şemaya kolon/tablo ekleyerek genişletebilir.
- **ISP:** dar store metotları (`Creators` / `CreatorDetail`) — şişkin arayüz yok.
- **Clean code:** merge kuralı tek yerde (SQL COALESCE); nötr alanlar açıkça işaretli (sahte skor değil); DRY (mevcut `getToken` fetch + `TokenRow`→struct deseni reuse); TDD (fixture + parity testleri).
- **Dürüstlük:** nötr alanlar 0 / `"medium"` / boş — 2b-2'ye ait, "hesaplandı ama düşük" değil.

## 8. Kabul kriterleri (deploy'da doğrulanır)

1. Migration 0006 goose ile uygulanır (yeni env gerekmez).
2. Deploy sonrası yeni pump.fun token'larında `tokens.creator` **dolu** (gerçek base58 pubkey).
3. `GET /api/creators` → gerçek creator adresleri + `totalTokens` sayımları (aynı creator birden çok token deploy ettiyse >1).
4. `GET /api/creator/{address}` → kimlik + `firstSeen` + o creator'ın token geçmişi (symbol/mint/createdAt/currentMarketCap gerçek); nötr alanlar dürüst placeholder.
5. Frontend `/creators` + `/creators/[address]` ekranları gerçek API'den okur (hibrit); UI regresyon yok.
6. Byte-offset kalibrasyonu doğruysa creator base58 geçerli; değilse hotfix (fixture+parse) — guard sayesinde bozulma yok.

## 9. Kullanıcı aksiyonları

- **Merge → Railway deploy** (master push'ta otomatik; migration 0006 goose ile).
- Yeni key / ücretli bağımlılık / harici panel ayarı **YOK** (mevcut `HELIUS_API_KEY` yeterli; ek RPC çağrısı yok).
- Deploy sonrası bekle: creator geçmişi birikmeye başlar (zaman-değerli — 2b-2 skoru bu birikime dayanır).
