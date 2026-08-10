# Slice 2b-2a — Token Outcome Detection + Peak Tracking (Tasarım)

- **Tarih:** 2026-08-10
- **Alt-proje:** 2 (Scoring & Graph) — dilim 2b (Creator Reputation), parça 2a (İtibar'ın alt-bölümü)
- **Branch (planlanan):** `feat/backend-scoring-2b-2a`
- **Önceki:** Slice 2b-1 (Creator Capture) merged + deploying (master `cbb126f`)
- **Sonraki:** Slice 2b-2b (Creator Reputation Score — bu dilimin outcome'larını agrega eder)

## 1. Amaç

Her tokenin piyasa trajektörisinden **akıbetini (outcome)** sınıflandırmak: `active | graduated | dumped | rug | dead`. Bu, iki değeri birlikte üretir: (1) Creator token geçmişinde **dürüst outcome/drawdown rozetleri** (2b-1'de nötr bırakılan `outcome`/`peakMarketCap`/`maxDrawdownPct`/`liquidityStatus` alanları gerçeğe döner), ve (2) **2b-2b itibar skoru için besleyici veri** (rug%/success% ancak per-token outcome'dan hesaplanabilir).

Outcome, drawdown gibi sinyaller **tepe (peak) gözlemi** ister — bu yüzden dilim aynı zamanda **peak takibi** getirir (enricher marketCap/likidite tepesini running-max saklar). Erken deploy edilir ki peak ve outcome verisi **zamanla biriksin** (2b-1'deki "erken deploy, biriksin" mantığı; taze token'da peak≈güncel → outcome dürüstçe "active", yaşlanınca sınıflanır).

**2b'nin bölünme gerekçesi:** tam 2b-2 (outcome + itibar + 2 Helius sinyali + tüm alan-doldurma) tek spec için 2a'dan belirgin büyük ve novel outcome-eşik kalibrasyonunu reputation agregasyonuyla aynı review'a sıkıştırırdı. 2b-2a (outcome/peak) bağımsız değerli + bağımsız test edilebilir + erken deploy; 2b-2b (reputation) bunun ürettiği outcome'lara dayanır.

## 2. Kapsam

### 2.1 Kapsam içi

1. **Şema (migration 0007):** `tokens`'a peak + outcome kolonları; mevcut değerlerden peak seed.
2. **Peak takibi:** Enricher `UpdateMarket` her tick'te `peak_market_cap`/`peak_liquidity`'yi `GREATEST(peak, güncel)` ile yükseltir (struct değişmez — `MarketUpdate` zaten `MarketCapUSD`+`Liquidity` taşır).
3. **Outcome sınıflandırıcı** (yeni `internal/outcome/`): saf sınıflandırıcı (SRP) + arka plan Worker (2a "Option A"/Enricher deseni; **Helius YOK — saf market verisi DB'den**) + store metotları. Outcome/maxDrawdownPct/liquidityStatus/outcome_scored_ts persist eder.
4. **Read wiring:** `CreatorDetail` token geçmişi bu alanları DB'den okur (2b-1'de nötr placeholder'dı). **Frontend'e sıfır dokunuş** (seam alanları zaten taşıyor).

### 2.2 Kapsam dışı (ertelendi — sessiz düşürme yok)

- **İtibar skoru** (`reputationScore`/`reputation: ScoreDetail`/`riskLevel`/`metrics.*`/`TokenRow.creatorScore`/`TokenDetail.scores.creatorReputation`) → **2b-2b** (bu dilimin outcome'larını agrega eder).
- **walletAgeDays + creatorHoldingPct** (Helius sinyalleri) → 2b-2b.
- **`creatorSellPct`** (per-token creator satış %) → trade-flow verisi gerektirir → davranış katmanıyla ertelendi (2c/2e).
- **`behavior.*`** (deployFrequency/avgFirstSellMinutes/repeatedFunders/similarMetadata/sameSocial/sameLiquidityPattern) → trade-flow + funding-graph → 2c/2e.
- **`liquidityStatus="locked"`** → gerçek LP-lock ancak on-chain LP hesabı doğrulamasıyla bilinir (bu dilimde yok) → "locked" **kullanılmaz**; dürüstçe "unlocked" (default) / "removed" (rug). İşaretli, gelecek.
- **Gerçek pump.fun mezuniyet (Raydium migration) tespiti** → marketCap-eşiği **proxy** ile yaklaşılır (aşağıda); tam dex-geçiş tespiti ertelendi.

## 3. Outcome modeli

Saf sınıflandırıcı; girdi tek tokenin anlık + tepe piyasa durumu, çıktı `{outcome, maxDrawdownPct, liquidityStatus}`. **Öncelikli-eşleşme (ilk eşleşen kazanır)** — sıra anlamlıdır (terminal/kötü sinyaller önce):

**Girdi:** `CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h, AgeSeconds float64/int64`

**maxDrawdownPct** (her zaman): `PeakMarketCap > 0` ise `clamp((PeakMarketCap−CurMarketCap)/PeakMarketCap×100, 0, 100)`, değilse `0`.

**Sınıflandırma sırası:**
1. **rug** — `PeakLiquidity ≥ minLiqFloor` **AND** `CurLiquidity ≤ PeakLiquidity × rugLiqRatio` (LP büyük ölçüde çekildi). → `liquidityStatus="removed"`. *(Terminal kötü sinyal; graduated-sonrası-rug bile rug sayılır — likidite çekişi baskındır.)*
2. **graduated** — `PeakMarketCap ≥ graduationMcap` **AND** `CurLiquidity ≥ minLiqFloor` (yüksek-cap'e ulaştı + likidite sağlıklı). *(pump.fun bonding-curve mezuniyet proxy'si; gerçek Raydium-migration tespiti ertelendi.)*
3. **dumped** — `maxDrawdownPct ≥ dumpedDrawdown` **AND** `CurLiquidity > PeakLiquidity × rugLiqRatio` (marketCap tepeden çöktü ama likidite duruyor → rug değil, fiyat çöküşü).
4. **dead** — `AgeSeconds ≥ deadAgeSec` **AND** `Vol24h ≤ deadVol` (yaşlı + işlemsiz; sert LP çekişi yok). 
5. **active** — varsayılan (taze/işlem gören/tepeye yakın).

**liquidityStatus:** rug ise `"removed"`, aksi `"unlocked"`. (`"locked"` kullanılmaz — bkz §2.2.)

**Eşikler (config, deploy'da kalibre edilir — gerçek pump.fun dağılımına göre):**
- `rugLiqRatio = 0.10` (tepe likiditenin ≤%10'u kaldı → çekildi)
- `graduationMcap = 69000` (USD; pump.fun mezuniyet ~$69k proxy)
- `dumpedDrawdown = 80` (%)
- `deadAgeSec = 86400` (24s; yaşlı sayılma eşiği)
- `deadVol = 100` (USD; 24s hacim ~0 eşiği)
- `minLiqFloor = 500` (USD; "anlamlı likidite vardı" tabanı — rug'ın 0'dan-0 false-positive'ini önler)

**Kalibrasyon riski (deploy'da, 1a/2a deseni):** eşikler gerçek veriyle ayarlanır; saf sınıflandırıcı + config sayesinde ayar kod değişikliği değil env değişikliğidir. Taze token peak≈cur → drawdown 0 → "active" (dürüst); yaşlanınca/hareket edince sınıflanır.

## 4. Veri modeli

### Migration 0007 (`0007_add_token_outcome.sql`)

```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS peak_market_cap  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS peak_liquidity   DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS outcome          TEXT NOT NULL DEFAULT 'active';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS max_drawdown_pct DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS liquidity_status TEXT NOT NULL DEFAULT 'unlocked';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS outcome_scored_ts BIGINT NOT NULL DEFAULT 0;
-- Peak seed: mevcut token'lar için tepeyi güncelden başlat (gerçek tarihsel tepe bilinmez;
-- peak ≥ güncel garantisi honest-conservative — drawdown migration'dan itibaren ölçülür).
UPDATE tokens SET peak_market_cap = market_cap_usd WHERE peak_market_cap = 0 AND market_cap_usd > 0;
UPDATE tokens SET peak_liquidity  = liquidity      WHERE peak_liquidity  = 0 AND liquidity > 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS outcome_scored_ts;
ALTER TABLE tokens DROP COLUMN IF EXISTS liquidity_status;
ALTER TABLE tokens DROP COLUMN IF EXISTS max_drawdown_pct;
ALTER TABLE tokens DROP COLUMN IF EXISTS outcome;
ALTER TABLE tokens DROP COLUMN IF EXISTS peak_liquidity;
ALTER TABLE tokens DROP COLUMN IF EXISTS peak_market_cap;
```

**Peak seed notu:** migration'daki tek-seferlik seed, mevcut token'ların tepesini o-anki değerden başlatır (gerçek tarihsel tepe kayıp — dürüstçe conservative). Enricher forward peak'leri gerçek zamanlı yükseltir.

## 5. Bileşenler (SOLID)

### 5.1 Peak takibi — Enricher (`internal/store/tokens.go` + `market/enricher.go`)

`UpdateMarket` postgres SQL'ine peak running-max eklenir (struct/çağrı değişmez — `MarketUpdate` zaten `MarketCapUSD`+`Liquidity` taşır):

```sql
UPDATE tokens SET price=$2, liquidity=$3, vol5m=$4, momentum=$5, spark=$6,
  price_change_h24=$7, market_cap_usd=$8, vol24h=$9,
  peak_market_cap = GREATEST(peak_market_cap, $8),
  peak_liquidity  = GREATEST(peak_liquidity, $3)
WHERE mint=$1
```

Fake `UpdateMarket` aynı max mantığını yansıtır (parity). Discoverer'ın ilk `UpdateMarket`'ı da peak'i seed eder (aynı yol).

### 5.2 `Classifier` (saf, SRP — `internal/outcome/classifier.go`)

```go
type Input struct {
	CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h float64
	AgeSeconds int64
}
type Result struct {
	Outcome         string  // active|graduated|dumped|rug|dead
	MaxDrawdownPct  float64
	LiquidityStatus string  // unlocked|removed
}
type Thresholds struct { RugLiqRatio, GraduationMcap, DumpedDrawdown, DeadVol, MinLiqFloor float64; DeadAgeSec int64 }

func Classify(in Input, t Thresholds) Result
```

Saf, yan-etkisiz, ağsız — tam birim-test edilebilir. Kural sırası §3. Eşikler dışarıdan (config'ten) → OCP (yeni eşik/kural env/registry ile).

### 5.3 `OutcomeStore` (DIP — `internal/store/`)

Dar arayüz (ISP):
```go
type OutcomeStore interface {
	OutcomeTargets(ctx, limit) ([]OutcomeTarget, error) // pool'lu token'lar; en eski outcome_scored_ts önce
	UpdateOutcome(ctx, OutcomeUpdate) error
}
type OutcomeTarget struct { Mint string; CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h float64; FirstSeenTs int64 }
type OutcomeUpdate struct { Mint, Outcome, LiquidityStatus string; MaxDrawdownPct float64; ScoredTs int64 }
```
`OutcomeTargets` SQL: `WHERE pool_address <> '' ORDER BY outcome_scored_ts ASC, first_seen_ts DESC LIMIT $1` (2a `SafetyScoreTargets` deseni). AgeSeconds = now − first_seen_ts (worker hesaplar). postgres + fake parity.

### 5.4 `Worker` (Enricher/2a deseni — `internal/outcome/worker.go`)

`WorkerDeps{Store OutcomeStore, Thresholds, Interval, Limit, Logger}` (Helius/rpcURL YOK — saf market). `Run(ctx)` ticker döngüsü: `OutcomeTargets(limit)` → her hedef `Classify` → `UpdateOutcome`. 2a `safety.Worker` birebir deseni ama daha basit (dış çağrı yok). `main.go`: config-gated goroutine (`OutcomeEnabled && Tokens`).

### 5.5 Read wiring — `CreatorDetail` (`internal/store/creators.go`)

2b-1'de `newHistoryItem` outcome/liquidityStatus/peak/drawdown'ı nötr set ediyordu. Şimdi `CreatorDetail` sorgusu bu kolonları SELECT eder ve `newHistoryItem` gerçek değerleri alır:
```go
func newHistoryItem(mint, symbol string, firstSeenTs int64, currentMcap, peakMcap, maxDrawdown float64, outcome, liquidityStatus string) CreatorTokenHistoryItem
```
`creatorSellPct` nötr kalır (0 → trade-flow, deferred). postgres + fake parity (fake `CreatorDetail` de yeni alanları taşır).

## 6. Config

`OUTCOME_ENABLED` (default true), `OUTCOME_INTERVAL_SEC` (60), `OUTCOME_LIMIT` (60), + eşikler: `OUTCOME_RUG_LIQ_RATIO`(0.10), `OUTCOME_GRADUATION_MCAP`(69000), `OUTCOME_DUMPED_DRAWDOWN`(80), `OUTCOME_DEAD_AGE_SEC`(86400), `OUTCOME_DEAD_VOL`(100), `OUTCOME_MIN_LIQ_FLOOR`(500). Hepsi default'lu (yeni zorunlu env yok). **Yeni key/ücret/harici bağımlılık YOK** (saf market verisi, mevcut GeckoTerminal enrichment'ından).

## 7. Test

- **Classifier (saf, ağsız):** her outcome için tablo-testi (rug: peak-likidite yüksek + cur ~0; graduated: peak-mcap ≥ eşik + likit; dumped: drawdown ≥ eşik + likit; dead: yaşlı + vol~0; active: taze/likit). Sınır durumları: peak=0 → drawdown 0 + active; peak≈cur → active; rug öncelik graduated'ı ezer (graduated-sonrası-rug).
- **Peak takibi:** `UpdateMarket` art arda çağrılarda peak düşmez, yalnız yükselir (fake ↔ postgres parity); güncel < peak sonrası peak korunur.
- **Store:** `OutcomeTargets` (pool'suz atla, en eski-skorlanan önce) + `UpdateOutcome` fake↔postgres parity.
- **Worker:** hedefleri sınıflandırıp persist eder; dış çağrı yok → deterministik.
- **CreatorDetail read:** geçmiş gerçek outcome/peak/drawdown/liquidityStatus taşır (fake↔postgres).
- Canlı peak birikimi + DB round-trip yalnız **deploy'da** (yerel Postgres yok — 0-2b-1 deseni).

## 8. Clean Code & SOLID

- **SRP:** Classifier yalnız sınıflar; Worker yalnız orkestra; store yalnız persist; enricher peak'i kendi UPDATE'inde tek satırla yükseltir.
- **DIP:** Worker `OutcomeStore` arayüzüne bağımlı, somut postgres'e değil; `main.go` wiring.
- **OCP:** eşikler config'ten (kod değişmeden kalibrasyon); yeni kural sınıflandırıcıya eklenir, çağıranlar değişmez.
- **ISP:** dar `OutcomeStore` (Targets+Update) — şişkin arayüz yok.
- **Clean code:** saf sınıflandırıcı (test edilebilir), `GREATEST` ile peak (yarış-yok, read-yok), nötr/deferred alanlar açık işaretli (`creatorSellPct`/behavior → sonraki), DRY (peak mantığı tek SQL).
- **Dürüstlük:** outcome gerçek piyasa trajektörisinden; peak seed conservative + işaretli; "locked" iddia edilmez; eşik-kalibrasyonu deploy'da açık.

## 9. Kabul kriterleri (deploy'da doğrulanır)

1. Migration 0007 goose ile uygulanır (yeni zorunlu env yok; peak seed mevcut token'lara uygulanır).
2. Enricher birkaç tick sonra `peak_market_cap`/`peak_liquidity` ≥ güncel (tepe birikiyor).
3. Outcome worker token'ları sınıflar: yaşlı+likidite-çökmüş token'lar `rug`/`dead`, taze/likit token'lar `active`; `max_drawdown_pct` gerçek.
4. `/api/creator/{address}` token geçmişinde `outcome`/`peakMarketCap`/`maxDrawdownPct`/`liquidityStatus` **gerçek** (nötr placeholder değil); `creatorSellPct` hâlâ 0 (dürüst, 2c).
5. Eşik kalibrasyonu gerçek dağılıma göre ayarlanır (env; kod değişmez) — false-positive rug/graduated gözlenirse `OUTCOME_*` güncellenir.
6. Frontend Creator geçmiş tablosu gerçek outcome rozetlerini gösterir (UI regresyonsuz — `OUTCOME_DEFS`/`LIQUIDITY_DEFS` anahtarları geçerli).

## 10. Kullanıcı aksiyonları

- **Merge → Railway deploy** (master push otomatik; migration 0007 goose ile).
- Yeni key / ücretli bağımlılık / harici panel **YOK** (saf market verisi mevcut enrichment'tan).
- Deploy sonrası: peak/outcome zamanla birikir; eşikler gerçek veriyle kalibre edilebilir (env). Not: **creator→outcome bağı yine Helius WS akışına bağlı** — creator'lı token'lar dolunca outcome+creator birlikte anlam kazanır (2b-2b reputation buna dayanır).
