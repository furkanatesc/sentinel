# SENTINEL Backend — KPI Trend (spark + change) Tasarım Spec'i

**Tarih:** 2026-09-07
**Dilim:** Backend Alt-proje 2 fast-follow — KPI zaman-serisi (trend). "Trend/zaman-serisi" işinin **A dilimi** (yalnız KPI; B token-likidite serisi A bitince değerlendirilecek, C radar ertelendi).
**Durum:** Spec — implementasyon öncesi

## Amaç

`GET /api/kpis`'in şu an boş dönen `change`/`spark` alanlarını **gerçek zaman-serisiyle** doldurmak.
Overview ekranındaki 4 gerçek KPI kartı (Tespit Edilen / Yüksek Güvenli / Kritik / Aktif Sinyaller)
sparkline + değişim oku gösterebilsin. **Entegrasyon-gerektirmez:** yalnız mevcut Postgres agregaları
(`store.Kpis`) periyodik snapshot'lanır; yeni harici bağımlılık YOK (kullanıcı kararı 2026-09-07:
"Helius gibi entegrasyonları sona bak").

## Kapsam

**Dahil:**
- Migration `0014_create_kpi_samples.sql` — zaman-serisi snapshot tablosu.
- Örnekleme worker'ı `internal/trend` — periyodik `store.Kpis()` → `InsertKpiSample` + retention prune.
- Store: `KpiSample` tipi + `InsertKpiSample` + `RecentKpiSamples` (postgres + fake).
- `kpisHandler` — son N örnekten her gerçek KPI için `spark` (metrik değerleri, kronolojik) + `change`
  (pencere içi ilk→son yüzde değişim) türetir (saf fonksiyon).
- Health kaydı: yeni worker adı `trend` (System Health paneli + gates).
- Config: `TREND_ENABLED`(true) / `TREND_SAMPLE_INTERVAL_SEC`(300) / `TREND_SAMPLE_KEEP`(288 = 5dk×288 ≈ 24s).

**Kapsam dışı (bilinçli, sessiz düşürme yok):**
- **4 placeholder KPI** (Açık Pozisyonlar / Gerçekleşen K/Z / Gerçekleşmemiş K/Z / Sistem Gecikmesi) —
  trade/ops verisi gerektirir (Alt-proje 5); `spark: []`, `change: 0` kalır.
- **B) Token likidite serisi** (`series.liquidity`) — A bitince aynı desenle ayrı dilim.
- **C) Radar zaman-serisi** — ertelendi (tek-snapshot yeterli).
- **Holders serisi** — mekanizma yok; değerler Helius'a bağlı (paid'e kadar) → sona.
- Frontend dokunulmaz — seam (`Kpi.change`/`Kpi.spark`) alanları zaten mevcut (`number`/`number[]`).

## Mimari

**Global ilkeler:** SRP küçük dosyalar, DIP dar arayüzler, saf/test edilebilir türetme, best-effort
telemetri (worker Health nil-güvenli), `github.com/furkanatesc/sentinel/apps/api-go/...` modül yolu.

### Migration 0014

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS kpi_samples (
  ts        BIGINT PRIMARY KEY,   -- unix saniye (örnek anı; PK → aynı ts idempotent)
  detected  INTEGER NOT NULL,
  high_conf INTEGER NOT NULL,
  critical  INTEGER NOT NULL,
  signals   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kpi_samples_ts ON kpi_samples (ts DESC);

-- +goose Down
DROP TABLE IF EXISTS kpi_samples;
```

### Store (tokens.go + postgres + fake)

```go
// KpiSample, tek bir zaman-noktasındaki KPI agregasıdır.
type KpiSample struct {
	Ts int64
	KpiCounts
}

// TokenStore'a eklenir:
InsertKpiSample(ctx context.Context, ts int64, c KpiCounts) error   // ON CONFLICT (ts) DO UPDATE
RecentKpiSamples(ctx context.Context, limit int) ([]KpiSample, error) // ts ASC (kronolojik; spark için)
```
- postgres `InsertKpiSample`: `INSERT ... ON CONFLICT (ts) DO UPDATE SET ...` (idempotent).
- postgres `RecentKpiSamples`: `SELECT ... FROM kpi_samples ORDER BY ts DESC LIMIT $1` → ters çevir (ASC) →
  saf, kronolojik dizi döndür.
- fake: in-memory slice, ts'e göre sıralı.

### Worker (internal/trend/worker.go)

DIP: dar `Sampler` arayüzü (store bağımlılığı):
```go
type Sampler interface {
	Kpis(ctx context.Context) (store.KpiCounts, error)
	InsertKpiSample(ctx context.Context, ts int64, c store.KpiCounts) error
	PruneKpiSamples(ctx context.Context, keep int) error // retention (en yeni `keep` dışını sil)
}
type WorkerDeps struct {
	Store    Sampler
	Interval time.Duration
	Keep     int
	Now      func() int64
	Logger   *slog.Logger
	Health   health.Reporter // nil-güvenli
}
```
`Run(ctx)`: diğer ticker worker'larla aynı iskelet (immediate `cycle` + ticker). `cycle`: `Kpis()` oku →
`InsertKpiSample(now, c)` → `PruneKpiSamples(keep)` → `Health.Report(WorkerTrend, ok, err, 1)` (1 örnek/cycle).
(`PruneKpiSamples` store'a eklenir; postgres `DELETE ... WHERE ts NOT IN (SELECT ts ... ORDER BY ts DESC LIMIT keep)`.)

### Handler türetimi (overview.go)

Saf fonksiyon:
```go
// kpiTrend, kronolojik örneklerden bir metriğin spark dizisi + yüzde değişimini türetir.
// change = ilk→son yüzde ((last-first)/first*100); first==0 → 0. spark = değerler (kronolojik).
func kpiTrend(vals []int) (spark []float64, change float64)
```
`kpisHandler`: `RecentKpiSamples(ctx, N)` (N = `TREND_SPARK_WINDOW`, ör. 24) oku; 4 gerçek KPI için
her metriğin değer dizisini `kpiTrend`'e ver → `Spark`/`Change` doldur. Örnek yoksa `spark:[]`,`change:0`
(mevcut davranış korunur — geriye uyumlu). Placeholder 4 KPI dokunulmaz.

### Wiring (main.go + config + health)

- `config.go`: `TrendEnabled`/`TrendSampleIntervalSec`/`TrendSampleKeep` (+ `TREND_SPARK_WINDOW` → `TrendSparkWindow`, default 24). `Load()` env okuma.
- `health.go`: `WorkerTrend = "trend"` const.
- `main.go`: `healthReg.Register(health.WorkerTrend, cfg.TrendEnabled && bundle.Tokens != nil, interval)` +
  gates `"TREND_ENABLED"` + config-gated worker goroutine (opportunity deseni; RPC YOK, saf DB) +
  `kpisHandler`'a spark penceresi geçir.

## Veri akışı

trend worker (arka plan, 5dk) → `kpi_samples` snapshot + prune → `/api/kpis` isteğinde son 24 örnek
okunur → saf `kpiTrend` → spark/change JSON. Frontend seam değişmez (React Query poll aynı).

## Hata yönetimi

- Worker best-effort: `Kpis`/`Insert`/`Prune` hatası → WARN + `Health.Report(ok=false)`; sonraki tick dener.
- Handler: `RecentKpiSamples` hatası → spark/change boş bırak (KPI değerleri yine döner; 500 değil).
- Boş tablo (ilk örnek öncesi) → spark `[]`, change `0` (mevcut davranış).

## Test

Go `test -race`:
- **Saf `kpiTrend`:** boş → ([],0); tek örnek → ([v],0); artış → doğru spark + pozitif change; first==0 → change 0.
- **Worker cycle:** fake Sampler ile `Run` (pre-cancel ctx) → 1 InsertKpiSample + 1 Prune + `Health.Report(trend, ok, nil, 1)`.
- **Store fake:** InsertKpiSample + RecentKpiSamples kronolojik sıra + Prune retention.
- **Handler:** stub store (RecentKpiSamples örnekli) → `/api/kpis` 4 gerçek KPI spark/change dolu, 4 placeholder boş.
- Mevcut kpis testleri kırılmaz (örnek yoksa geriye uyumlu).
- postgres path'i DB-gated (yerel Postgres yok → deploy'da doğrulanır; 1a-2e deseni).

## Deploy sonrası

Railway'de trend worker System Health'te `ok` görünür; `/api/kpis` birkaç örnek sonra (≥2 tick, ~10dk)
spark/change dolu döner. Frontend Overview sparkline'ları canlanır (deploy + push kullanıcı onayına bağlı).

## Fast-follow (A bitince)

- **B) Token likidite serisi:** aynı worker'a per-token likidite snapshot'ı + `series.liquidity` (detail.go).
- Holders serisi (Helius paid sonrası), radar zaman-serisi (gerekirse).
