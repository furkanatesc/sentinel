# KPI Trend (spark + change) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development veya superpowers:executing-plans. Steps checkbox (`- [ ]`).

**Goal:** `/api/kpis` `change`/`spark`'ını periyodik KPI snapshot zaman-serisinden gerçek türet.

**Architecture:** `internal/trend` worker `store.Kpis()`'i periyodik `kpi_samples` tablosuna snapshot'lar + prune; `kpisHandler` son N örnekten saf `kpiTrend` ile spark/change türetir. Entegrasyon YOK (saf DB).

**Tech Stack:** Go (chi, database/sql, goose migration), Postgres.

**Spec:** `docs/superpowers/specs/2026-09-07-sentinel-kpi-trend-design.md`

## Global Constraints
- Clean code & SOLID: dar arayüz (DIP `Sampler`), saf `kpiTrend`, SRP dosyalar.
- Best-effort telemetri: worker `Health` nil-güvenli; hata → WARN + Report(ok=false).
- Geriye uyumlu: örnek yoksa `spark:[]`,`change:0` (mevcut davranış); placeholder 4 KPI dokunulmaz.
- `go test ./... -race` + `go vet ./...` yeşil; mevcut testler kırılmaz.
- Modül yolu `github.com/furkanatesc/sentinel/apps/api-go/...`.
- Commit sonu: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## Task 1: Migration 0014 + store KpiSample seam

**Files:**
- Create: `internal/store/migrations/0014_create_kpi_samples.sql`
- Modify: `internal/store/tokens.go` (KpiSample tipi + TokenStore'a 3 metot), `internal/store/postgres.go` (impl), `internal/store/fake_ingest.go` (fake impl)
- Test: `internal/store/kpi_samples_test.go` (fake)

**Interfaces:**
- Produces: `KpiSample{Ts int64; KpiCounts}`; `InsertKpiSample(ctx, ts int64, c KpiCounts) error`; `RecentKpiSamples(ctx, limit int) ([]KpiSample, error)` (ts ASC); `PruneKpiSamples(ctx, keep int) error`.

- [ ] **Step 1: Migration dosyası** (spec §Migration 0014 SQL — birebir).

- [ ] **Step 2: Failing test (`kpi_samples_test.go`)** — fake store:
```go
package store
import ("context";"testing")
func TestFakeKpiSamplesInsertRecentPrune(t *testing.T) {
	s := NewFakeTokenStore().(TokenStore)
	ctx := context.Background()
	for i, ts := range []int64{100, 200, 300} {
		if err := s.InsertKpiSample(ctx, ts, KpiCounts{Detected: i + 1}); err != nil { t.Fatal(err) }
	}
	got, err := s.RecentKpiSamples(ctx, 10)
	if err != nil { t.Fatal(err) }
	if len(got) != 3 || got[0].Ts != 100 || got[2].Ts != 300 { t.Fatalf("kronolojik ASC beklenir: %+v", got) }
	if got[2].Detected != 3 { t.Fatalf("Detected taşınmalı: %+v", got[2]) }
	// idempotent (aynı ts güncelle)
	if err := s.InsertKpiSample(ctx, 300, KpiCounts{Detected: 9}); err != nil { t.Fatal(err) }
	got, _ = s.RecentKpiSamples(ctx, 10)
	if len(got) != 3 || got[2].Detected != 9 { t.Fatalf("aynı ts idempotent güncellemeli: %+v", got) }
	// prune: en yeni 2 kalsın
	if err := s.PruneKpiSamples(ctx, 2); err != nil { t.Fatal(err) }
	got, _ = s.RecentKpiSamples(ctx, 10)
	if len(got) != 2 || got[0].Ts != 200 { t.Fatalf("prune en yeni 2'yi tutmalı: %+v", got) }
	// limit: son 1
	got, _ = s.RecentKpiSamples(ctx, 1)
	if len(got) != 1 || got[0].Ts != 300 { t.Fatalf("limit=1 en yeni 1: %+v", got) }
}
```

- [ ] **Step 3: Run → fail** — `cd apps/api-go && go test ./internal/store/ -run TestFakeKpiSamples -v` (metotlar yok).

- [ ] **Step 4: Implement.**
  - `tokens.go`: `KpiSample` tipi + 3 metot imzasını `TokenStore` interface'ine ekle (Kpis'in altına).
  - `postgres.go`:
    - `InsertKpiSample`: `INSERT INTO kpi_samples (ts,detected,high_conf,critical,signals) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (ts) DO UPDATE SET detected=EXCLUDED.detected, high_conf=EXCLUDED.high_conf, critical=EXCLUDED.critical, signals=EXCLUDED.signals`.
    - `RecentKpiSamples`: `SELECT ts,detected,high_conf,critical,signals FROM kpi_samples ORDER BY ts DESC LIMIT $1` → satırları oku → **ters çevir** (ASC) döndür.
    - `PruneKpiSamples`: `DELETE FROM kpi_samples WHERE ts NOT IN (SELECT ts FROM kpi_samples ORDER BY ts DESC LIMIT $1)`.
  - `fake_ingest.go`: `fakeTokenStore`'a `kpiSamples map[int64]KpiCounts` alanı (yoksa) + 3 metot: Insert (map set), Recent (map→slice, ts ASC sort, son `limit`), Prune (en yeni `keep` dışını sil). Kilit (mevcut fake mutex deseni).

- [ ] **Step 5: Run → pass + build** — `go test ./internal/store/ -run TestFakeKpiSamples -race && go build ./...`.

- [ ] **Step 6: Commit** — `git add internal/store && git commit -m "feat(trend): kpi_samples migration + store seam (Insert/Recent/Prune) (Task 1)"`

---

## Task 2: Saf `kpiTrend` + kpisHandler türetimi

**Files:**
- Modify: `internal/api/overview.go`
- Test: `internal/api/overview_test.go` (Create ya da mevcut — kpiTrend + handler)

**Interfaces:**
- Consumes: `store.KpiSample`, `store.RecentKpiSamples`.
- Produces: saf `kpiTrend(vals []int) (spark []float64, change float64)`; `kpisHandler` spark penceresi parametresi alır.

- [ ] **Step 1: Failing test (`overview_test.go`)**
```go
package api
import ("testing")
func TestKpiTrend(t *testing.T) {
	if sp, ch := kpiTrend(nil); len(sp) != 0 || ch != 0 { t.Fatalf("boş → ([],0), got %v %v", sp, ch) }
	if sp, ch := kpiTrend([]int{5}); len(sp) != 1 || ch != 0 { t.Fatalf("tek → ([5],0), got %v %v", sp, ch) }
	sp, ch := kpiTrend([]int{10, 15})
	if len(sp) != 2 || sp[0] != 10 || sp[1] != 15 { t.Fatalf("spark kronolojik: %v", sp) }
	if ch != 50 { t.Fatalf("change (15-10)/10*100=50, got %v", ch) }
	if _, ch := kpiTrend([]int{0, 5}); ch != 0 { t.Fatalf("first==0 → change 0, got %v", ch) }
}
```
(Handler testi opsiyonel — stub store; en az `kpiTrend` saf testi zorunlu. Handler için mevcut kpis testi varsa spark alanının örnekli store'da dolduğunu ekle.)

- [ ] **Step 2: Run → fail** — `go test ./internal/api/ -run TestKpiTrend -v`.

- [ ] **Step 3: Implement (`overview.go`)**
```go
func kpiTrend(vals []int) ([]float64, float64) {
	spark := make([]float64, len(vals))
	for i, v := range vals { spark[i] = float64(v) }
	var change float64
	if len(vals) >= 2 && vals[0] != 0 {
		change = (float64(vals[len(vals)-1]) - float64(vals[0])) / float64(vals[0]) * 100
	}
	return spark, change
}
```
`kpisHandler` imzasına spark penceresi ekle: `func kpisHandler(ts store.TokenStore, sparkWindow int) http.HandlerFunc`. İçinde `Kpis` sonrası:
```go
samples, err := ts.RecentKpiSamples(r.Context(), sparkWindow)
if err != nil { samples = nil } // best-effort: spark boş kalır
detSp, detCh := kpiTrend(pick(samples, func(s store.KpiSample) int { return s.Detected }))
// ... high_conf/critical/signals için de
```
4 gerçek KPI'ya `Spark`/`Change` ata; `pick` küçük bir helper (samples→[]int). Placeholder 4 KPI dokunulmaz. `router.go`'da `kpisHandler(ts)` çağrısına `sparkWindow` argümanı ekle (config'ten, Task 4).

- [ ] **Step 4: Run → pass + build** — `go test ./internal/api/ -run TestKpiTrend -race && go build ./...` (router.go çağrısı Task 4'te düzelene kadar geçici sabit ör. `24` verilebilir; Task 4 config'e bağlar).

- [ ] **Step 5: Commit** — `git add internal/api && git commit -m "feat(trend): kpiTrend saf türetme + kpisHandler spark/change (Task 2)"`

---

## Task 3: trend worker + health WorkerTrend

**Files:**
- Create: `internal/trend/worker.go`, `internal/trend/worker_test.go`
- Modify: `internal/health/registry.go` (WorkerTrend const)

**Interfaces:**
- Consumes: `store.KpiCounts`, `health.Reporter`, `health.WorkerTrend`.
- Produces: `trend.Sampler` arayüzü; `trend.WorkerDeps`; `trend.NewWorker`; `(*Worker).Run(ctx)`.

- [ ] **Step 1: health const** — `registry.go` worker adı consts'a: `WorkerTrend = "trend"`.

- [ ] **Step 2: Failing test (`worker_test.go`)**
```go
package trend
import ("context";"testing";"time"; "github.com/furkanatesc/sentinel/apps/api-go/internal/store")
type fakeSampler struct{ inserts, prunes int }
func (f *fakeSampler) Kpis(context.Context) (store.KpiCounts, error) { return store.KpiCounts{Detected: 7}, nil }
func (f *fakeSampler) InsertKpiSample(_ context.Context, _ int64, _ store.KpiCounts) error { f.inserts++; return nil }
func (f *fakeSampler) PruneKpiSamples(context.Context, int) error { f.prunes++; return nil }
type recReporter struct{ name string; ok bool; processed, calls int }
func (r *recReporter) Register(string, bool, time.Duration) {}
func (r *recReporter) Report(name string, ok bool, _ error, processed int) { r.name, r.ok, r.processed, r.calls = name, ok, processed, r.calls+1 }
func TestWorkerCycleSamplesAndReports(t *testing.T) {
	fs := &fakeSampler{}; rr := &recReporter{}
	w := NewWorker(WorkerDeps{Store: fs, Interval: time.Hour, Keep: 10, Health: rr, Now: func() int64 { return 42 }})
	ctx, cancel := context.WithCancel(context.Background()); cancel()
	w.Run(ctx) // immediate cycle + ctx.Done
	if fs.inserts != 1 || fs.prunes != 1 { t.Fatalf("1 insert + 1 prune beklenir: %+v", fs) }
	if rr.calls == 0 || rr.name != "trend" || !rr.ok || rr.processed != 1 { t.Fatalf("Report(trend, ok, 1) beklenir: %+v", rr) }
}
```

- [ ] **Step 3: Run → fail** — `go test ./internal/trend/ -run TestWorkerCycle -v`.

- [ ] **Step 4: Implement (`worker.go`)** — spec §Worker: `Sampler` arayüzü (Kpis/InsertKpiSample/PruneKpiSample), `WorkerDeps` (Store/Interval/Keep/Now/Logger/Health), `NewWorker` (Now/Logger default), `Run` (immediate `cycle` + ticker; ctx.Done'da dön), `cycle`: `Kpis` → `InsertKpiSample(now, c)` → `PruneKpiSamples(keep)` → `Health.Report(health.WorkerTrend, err==nil, err, 1)` (herhangi adım hata → err set, WARN). safety/opportunity Run iskeletini birebir izle.

- [ ] **Step 5: Run → pass + build** — `go test ./internal/trend/ ./internal/health/ -race && go build ./...`.

- [ ] **Step 6: Commit** — `git add internal/trend internal/health && git commit -m "feat(trend): örnekleme worker'ı + health WorkerTrend (Task 3)"`

---

## Task 4: config + main.go wiring

**Files:**
- Modify: `internal/config/config.go`, `cmd/server/main.go`, `internal/api/router.go`
- Test: `internal/config/config_test.go` (default assertion)

**Interfaces:**
- Consumes: `trend.NewWorker`, `health.WorkerTrend`, `config.Trend*`.
- Produces: `Config.TrendEnabled/TrendSampleIntervalSec/TrendSampleKeep/TrendSparkWindow`; main.go worker goroutine + register + gate; router `kpisHandler(ts, cfg.TrendSparkWindow)`.

- [ ] **Step 1: config default test (`config_test.go`)**
```go
func TestTrendDefaults(t *testing.T) {
	t.Setenv("TREND_ENABLED", ""); t.Setenv("TREND_SAMPLE_INTERVAL_SEC", ""); t.Setenv("TREND_SPARK_WINDOW", "")
	c := Load()
	if !c.TrendEnabled { t.Fatal("TrendEnabled default true") }
	if c.TrendSampleIntervalSec != 300 { t.Fatalf("interval default 300, got %d", c.TrendSampleIntervalSec) }
	if c.TrendSampleKeep != 288 { t.Fatalf("keep default 288, got %d", c.TrendSampleKeep) }
	if c.TrendSparkWindow != 24 { t.Fatalf("spark window default 24, got %d", c.TrendSparkWindow) }
}
```

- [ ] **Step 2: Run → fail** — `go test ./internal/config/ -run TestTrendDefaults -v`.

- [ ] **Step 3: config.go** — 4 alan + `Load()`: `TrendEnabled: getenvBool("TREND_ENABLED", true)`, `TrendSampleIntervalSec: getenvInt("TREND_SAMPLE_INTERVAL_SEC", 300)`, `TrendSampleKeep: getenvInt("TREND_SAMPLE_KEEP", 288)`, `TrendSparkWindow: getenvInt("TREND_SPARK_WINDOW", 24)`.

- [ ] **Step 4: main.go** —
  - import `internal/trend`.
  - `healthReg.Register(health.WorkerTrend, cfg.TrendEnabled && bundle.Tokens != nil, time.Duration(cfg.TrendSampleIntervalSec)*time.Second)`.
  - gates map: `"TREND_ENABLED": cfg.TrendEnabled`.
  - worker goroutine (opportunity deseni, RPC yok):
```go
if cfg.TrendEnabled && bundle.Tokens != nil {
	tw := trend.NewWorker(trend.WorkerDeps{
		Store: bundle.Tokens, Interval: time.Duration(cfg.TrendSampleIntervalSec) * time.Second,
		Keep: cfg.TrendSampleKeep, Logger: logger, Health: healthReg,
	})
	go tw.Run(ctx)
}
```
  (`bundle.Tokens` `TokenStore`'u karşılar → `trend.Sampler`'ı karşılar; alt-küme.)
  - `router.go`: `kpisHandler` çağrısını `kpisHandler(d.Tokens, d.KpiSparkWindow)` yap; `RouterDeps`'e `KpiSparkWindow int` ekle; main.go RouterDeps'e `KpiSparkWindow: cfg.TrendSparkWindow`.

- [ ] **Step 5: Run → build + tüm testler** — `go build ./... && go test ./... -race && go vet ./...` yeşil.

- [ ] **Step 6: Commit** — `git add internal/config cmd/server/main.go internal/api/router.go && git commit -m "feat(trend): config + main wiring — worker goroutine + register + spark window (Task 4)"`

---

## Task 5: Whole-branch review + doküman

- [ ] **Step 1: Tüm testler + build + vet** — `go test ./... -race && go vet ./...` yeşil.
- [ ] **Step 2: Whole-branch review** — superpowers:requesting-code-review (opus). Bulguları superpowers:receiving-code-review ile ele al.
- [ ] **Step 3: Yaşayan dokümanlar** — `docs/progress.md` (KPI trend girişi) + `MEMORY.md` aktif-iş satırı + `docs/superpowers/followups-frontend.md` (B token likidite serisi + holders/radar ertelemesi).
- [ ] **Step 4: Merge/push** — kullanıcı onayıyla (DUR-noktası).

---

## Self-Review

**1. Spec coverage:** migration+store → Task 1 ✅; kpiTrend+handler → Task 2 ✅; worker+health → Task 3 ✅; config+wiring → Task 4 ✅; review+docs → Task 5 ✅. Placeholder KPI dokunulmaz (Task 2), 4 real KPI spark/change (Task 2). ✅
**2. Placeholder scan:** tüm test+impl adımları somut kod; SQL birebir spec'te. ✅
**3. Type consistency:** `KpiSample{Ts;KpiCounts}` Task 1 → Task 2/3; `Sampler`(Kpis/InsertKpiSample/PruneKpiSamples) Task 3, store 3 metodu Task 1 ile birebir; `kpiTrend(vals []int)` Task 2; `KpiSparkWindow`/`TrendSparkWindow` Task 4 tutarlı. ✅
