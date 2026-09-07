# SENTINEL Backend — Token Likidite Serisi Tasarım Spec'i

**Tarih:** 2026-09-07
**Dilim:** Trend/zaman-serisi işinin **B dilimi** (A = KPI trend, MERGED `68f05ae`). Backend Alt-proje 1c'de
ertelenen `series.liquidity`'yi gerçeğe döndürür.
**Durum:** Spec — implementasyon öncesi

## Amaç

Token Detail'in (`GET /api/token/{mint}`) boş dönen `series.liquidity` alanını **gerçek zaman-serisiyle**
doldurmak. Mevcut `internal/trend` worker'ı KPI snapshot'ının yanında **en yeni N token'ın likiditesini**
periyodik snapshot'lar; `detail.go` per-mint okur. **Entegrasyon-gerektirmez** — likidite zaten DB'de
(GeckoTerminal enricher yazıyor, keysiz); yeni harici bağımlılık YOK (kullanıcı kararı: Helius gibi
entegrasyonlar sona).

## Kapsam

**Dahil:**
- Migration `0015_create_token_liq_samples.sql`.
- Store: `InsertLiquiditySamples(ctx, ts, limit)` (set-based, en yeni N token) + `PruneLiquiditySamples(ctx, cutoff)` (yaş-tabanlı) + `LiquiditySeries(ctx, mint, limit)` (postgres + fake).
- `internal/trend` worker cycle'ı: KPI sample + **likidite sample + prune** (aynı worker/interval).
- `detail.go`: `Series.Liquidity`'yi `LiquiditySeries`'ten doldur (best-effort).
- Config: `TREND_LIQ_ENABLED`(true) / `TREND_LIQ_SAMPLE_LIMIT`(200) / `TREND_LIQ_KEEP_HOURS`(48) / `TREND_LIQ_SERIES_LIMIT`(read cap, 500).

**Kapsam dışı (bilinçli):**
- **Holders serisi** — Helius'a bağlı (sona); B'ye dahil değil.
- **C radar zaman-serisi** — ertelendi.
- **Eski/nadir token'lar** (en yeni N dışında) örneklenmez → `series.liquidity` boş (kabul edilebilir;
  bugünkü davranışla aynı, best-effort). Detay çoğunlukla yeni token'lar için açılır.
- `series.price`/`series.volume` zaten gerçek (OHLCV) — dokunulmaz.

## Mimari

**Global ilkeler:** SRP, DIP dar arayüz, saf/test edilebilir; best-effort (worker Health nil-güvenli, detail
okuma hatası → boş seri, 500 değil); modül yolu `.../apps/api-go/...`.

### Migration 0015

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS token_liq_samples (
    mint      TEXT NOT NULL,
    ts        BIGINT NOT NULL,
    liquidity DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (mint, ts)
);
-- (mint, ts) PK btree → WHERE mint=$1 ORDER BY ts hem yazma-conflict hem okuma için yeter; ek index yok.

-- +goose Down
DROP TABLE IF EXISTS token_liq_samples;
```

### Store (tokens.go + postgres + fake)

TokenStore'a eklenir:
```go
// InsertLiquiditySamples, en yeni `limit` token'ın likiditesini `ts` anında snapshot'lar (set-based,
// tek sorgu). liquidity>0 filtresi (0/enrich-edilmemiş token gürültüsünü ele). Idempotent (PK conflict).
InsertLiquiditySamples(ctx context.Context, ts int64, limit int) error
// PruneLiquiditySamples, ts < cutoff örnekleri siler (yaş-tabanlı retention).
PruneLiquiditySamples(ctx context.Context, cutoff int64) error
// LiquiditySeries, bir mint'in likidite serisini kronolojik (ts ASC) son `limit` nokta döndürür.
LiquiditySeries(ctx context.Context, mint string, limit int) ([]SeriesPoint, error)
```
postgres:
- Insert: `INSERT INTO token_liq_samples (mint, ts, liquidity) SELECT mint, $1, liquidity FROM tokens WHERE liquidity > 0 ORDER BY first_seen_ts DESC LIMIT $2 ON CONFLICT (mint, ts) DO NOTHING`.
- Prune: `DELETE FROM token_liq_samples WHERE ts < $1`.
- Series: `SELECT ts, liquidity FROM token_liq_samples WHERE mint=$1 ORDER BY ts ASC LIMIT $2` → `[]SeriesPoint{T,V}`. `limit<=0` → nil (guard, KPI trend deseni).

fake: `liqSamples map[string]map[int64]float64` (mint→ts→liq). Insert: fake'te "en yeni N token" için
`RecentTokens(limit)` sonuçlarını kullan (liquidity>0), her mint'e ts→liq yaz. Prune: ts<cutoff sil.
Series: mint'in ts'lerini ASC sırala, son `limit`, `SeriesPoint` döndür.

### Worker (internal/trend genişletme)

`Sampler` arayüzüne 2 metot eklenir: `InsertLiquiditySamples`, `PruneLiquiditySamples`. `WorkerDeps`'e
`LiqEnabled bool`, `LiqSampleLimit int`, `LiqKeepSeconds int64` eklenir. `sampleOnce` (mevcut KPI akışı
korunur) sonrasında, `LiqEnabled` ise: `InsertLiquiditySamples(now, LiqSampleLimit)` + `PruneLiquiditySamples(now - LiqKeepSeconds)`.
Hata izolasyonu: likidite örnekleme hatası KPI sample'ı geçersiz kılmaz — ama cycle `err` set eder (Report
ok=false). itemsProcessed = KPI(1) — likidite set-based olduğundan ayrı sayılmaz (dokümante).

**Alternatif (SRP):** likidite örnekleme ayrı bir metoda (`liqSampleOnce`) çıkarılır; cycle her ikisini
çağırır, ilk hata err'i taşır. (Uygulama: `cycle` KPI + Liq'i ayrı çağırır, err birleştirilir.)

### detail.go

`TokenDetailService`'in Store dep'i (market paketindeki detail Store arayüzü) `LiquiditySeries` kazanır.
OHLCV Price/Volume doldurulduktan sonra:
```go
if s.d.LiqSeriesLimit > 0 {
	if pts, err := s.d.Store.LiquiditySeries(ctx, mint, s.d.LiqSeriesLimit); err != nil {
		s.d.Logger.Warn("detail liquidity series", "mint", mint, "err", err)
	} else {
		d.Series.Liquidity = pts
	}
}
```
`TokenDetailDeps`'e `LiqSeriesLimit int` eklenir (0 → doldurma, geriye uyumlu).

### Wiring (config + main.go)

- config: 4 alan + `Load()` env.
- main.go: trend worker construction'a `LiqEnabled: cfg.TrendLiqEnabled, LiqSampleLimit: cfg.TrendLiqSampleLimit, LiqKeepSeconds: int64(cfg.TrendLiqKeepHours)*3600`; detail service construction'a `LiqSeriesLimit: cfg.TrendLiqSeriesLimit`.
- Not: `series` gate'i System Health'e ayrı worker olarak EKLENMEZ (trend worker zaten kayıtlı; likidite onun bir parçası). `TREND_LIQ_ENABLED` gates map'e eklenir (görünürlük).

## Veri akışı

trend worker (5dk) → KPI sample + en-yeni-200-token likidite snapshot + prune → `token_liq_samples`.
`/api/token/{mint}` isteğinde `LiquiditySeries(mint, 500)` → `Series.Liquidity`. Frontend seam değişmez
(TokenDetailSeries.liquidity zaten SeriesPoint[]).

## Hata yönetimi

- Worker: likidite Insert/Prune hatası → WARN + cycle Report(ok=false); KPI örnek yine yazılır.
- detail: `LiquiditySeries` hatası → WARN + `Series.Liquidity` boş; token detail yine döner.
- Örneksiz mint / eski token → boş seri (best-effort; bugünkü davranış).

## Test

Go `test -race`:
- **Store fake:** InsertLiquiditySamples (en yeni N + liquidity>0) + LiquiditySeries (mint filtre + ts ASC + limit) + Prune (cutoff).
- **Worker:** fake Sampler ile cycle → KPI insert + liq insert + liq prune çağrıldı; likidite hatası → Report(ok=false) ama KPI insert yine oldu.
- **detail:** stub Store (LiquiditySeries örnekli) → `Series.Liquidity` dolu; hata → boş + token yine döner; LiqSeriesLimit=0 → doldurma yok.
- **config:** default assertion.
- Mevcut testler kırılmaz (yeni interface metotları fake/stub'lara eklenir).
- postgres path DB-gated (deploy'da doğrulanır).

## Deploy sonrası

Railway trend worker likidite de örnekler; `/api/token/{mint}` (yeni/aktif token) birkaç örnek sonra
`series.liquidity` dolu döner → Token Detail likidite grafiği canlanır (deploy+push kullanıcı onayına bağlı).
