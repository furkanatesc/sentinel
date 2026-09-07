# Token Likidite Serisi Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development veya superpowers:executing-plans. Steps checkbox (`- [ ]`).

**Goal:** Token Detail'in boş `series.liquidity`'sini gerçeğe döndür — trend worker en yeni N token'ın likiditesini snapshot'lar, detail.go per-mint okur.

**Architecture:** Migration 0015 `token_liq_samples` + store seam (Insert set-based / Prune yaş / Series read) + mevcut `internal/trend` worker cycle genişlemesi + detail.go wiring. Entegrasyon YOK (likidite zaten DB'de).

**Tech Stack:** Go (database/sql, goose), Postgres.

**Spec:** `docs/superpowers/specs/2026-09-07-sentinel-liquidity-series-design.md`

## Global Constraints
- Clean code & SOLID: `Sampler` DIP genişlemesi dar; detail Store'a tek okuma metodu; SRP.
- Best-effort: worker likidite hatası KPI sample'ı geçersiz kılmaz; detail okuma hatası → boş seri (500 değil).
- Geriye uyumlu: örneksiz mint / `LiqSeriesLimit=0` → boş seri (mevcut davranış).
- `go build`/`go vet`/`go test ./... -race` yeşil; mevcut testler kırılmaz.
- Modül yolu `github.com/furkanatesc/sentinel/apps/api-go/...`. Commit sonu: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## Task 1: Migration 0015 + store likidite seam

**Files:**
- Create: `internal/store/migrations/0015_create_token_liq_samples.sql`
- Modify: `internal/store/tokens.go` (interface + postgres), `internal/store/fake_ingest.go` (fake)
- Test: `internal/store/liq_samples_test.go`

**Interfaces:**
- Produces: `InsertLiquiditySamples(ctx, ts int64, limit int) error`; `PruneLiquiditySamples(ctx, cutoff int64) error`; `LiquiditySeries(ctx, mint string, limit int) ([]SeriesPoint, error)`.

- [ ] **Step 1: Migration** (spec §Migration 0015 SQL — birebir).

- [ ] **Step 2: Failing test (`liq_samples_test.go`)**
```go
package store
import ("context";"testing")
func TestFakeLiquiditySamples(t *testing.T) {
	f := NewFakeTokenStore().(TokenStore)
	ctx := context.Background()
	// iki token: A likidite 1000, B likidite 0 (filtrelenmeli)
	_ = f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "A", Symbol: "A", PoolAddr: "pA", FirstSeenTs: 2})
	_ = f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "B", Symbol: "B", PoolAddr: "pB", FirstSeenTs: 1})
	_ = f.UpdateMarket(ctx, MarketUpdate{Mint: "A", Liquidity: 1000})
	// ts=100 ve ts=200'de örnek al
	if err := f.InsertLiquiditySamples(ctx, 100, 10); err != nil { t.Fatal(err) }
	_ = f.UpdateMarket(ctx, MarketUpdate{Mint: "A", Liquidity: 1200})
	if err := f.InsertLiquiditySamples(ctx, 200, 10); err != nil { t.Fatal(err) }
	// A serisi kronolojik [1000@100, 1200@200]
	ser, err := f.LiquiditySeries(ctx, "A", 10)
	if err != nil { t.Fatal(err) }
	if len(ser) != 2 || ser[0].T != 100 || ser[0].V != 1000 || ser[1].V != 1200 { t.Fatalf("A serisi: %+v", ser) }
	// B likidite 0 → örneklenmedi
	if s, _ := f.LiquiditySeries(ctx, "B", 10); len(s) != 0 { t.Fatalf("B örneklenmemeli: %+v", s) }
	// prune cutoff=150 → ts=100 silinir, ts=200 kalır
	if err := f.PruneLiquiditySamples(ctx, 150); err != nil { t.Fatal(err) }
	if s, _ := f.LiquiditySeries(ctx, "A", 10); len(s) != 1 || s[0].T != 200 { t.Fatalf("prune sonrası A: %+v", s) }
	// limit<=0 → boş (guard)
	if s, _ := f.LiquiditySeries(ctx, "A", 0); len(s) != 0 { t.Fatalf("limit<=0 → boş: %+v", s) }
}
```

- [ ] **Step 3: Run → fail** — `cd apps/api-go && go test ./internal/store/ -run TestFakeLiquiditySamples -v`.

- [ ] **Step 4: Implement.**
  - `tokens.go`: interface'e 3 metot (Kpis bloğunun altına). postgres:
    - Insert: `INSERT INTO token_liq_samples (mint, ts, liquidity) SELECT mint, $1, liquidity FROM tokens WHERE liquidity > 0 ORDER BY first_seen_ts DESC LIMIT $2 ON CONFLICT (mint, ts) DO NOTHING`.
    - Prune: `DELETE FROM token_liq_samples WHERE ts < $1`.
    - Series: `if limit <= 0 { return nil, nil }`; `SELECT ts, liquidity FROM token_liq_samples WHERE mint=$1 ORDER BY ts ASC LIMIT $2` → `[]SeriesPoint{T,V}`.
  - `fake_ingest.go`: `liqSamples map[string]map[int64]float64` alanı. Insert: `RecentTokens(limit)` çağır, her satır `Liquidity>0` ise `liqSamples[mint][ts]=liq`. Prune: her mint için ts<cutoff sil. Series: `if limit<=0 return nil`; mint'in ts'lerini ASC sırala, son `limit`, `SeriesPoint` döndür. (mutex.)

- [ ] **Step 5: Run → pass + build** — `go test ./internal/store/ -run TestFakeLiquiditySamples -race && go build ./...`.

- [ ] **Step 6: Commit** — `git add internal/store && git commit -m "feat(liq): token_liq_samples migration + store seam (Insert/Prune/Series) (Task 1)"`

---

## Task 2: trend worker likidite örnekleme

**Files:**
- Modify: `internal/trend/worker.go`
- Test: `internal/trend/worker_test.go` (fake Sampler'a 2 metot + yeni assertion)

**Interfaces:**
- Consumes: store 3 metodu (Insert/Prune Liquidity).
- Produces: `Sampler` arayüzüne `InsertLiquiditySamples`+`PruneLiquiditySamples`; `WorkerDeps`'e `LiqEnabled bool`, `LiqSampleLimit int`, `LiqKeepSeconds int64`.

- [ ] **Step 1: Failing test** — mevcut `fakeSampler`'a 2 metot ekle (`liqInserts`, `liqPrunes` sayaç) + yeni test:
```go
func (f *fakeSampler) InsertLiquiditySamples(context.Context, int64, int) error { f.liqInserts++; return nil }
func (f *fakeSampler) PruneLiquiditySamples(context.Context, int64) error { f.liqPrunes++; return nil }
// TestWorkerCycleSamplesLiquidity
func TestWorkerCycleSamplesLiquidity(t *testing.T) {
	fs := &fakeSampler{}
	w := NewWorker(WorkerDeps{Store: fs, Interval: time.Hour, Keep: 10, LiqEnabled: true, LiqSampleLimit: 5, LiqKeepSeconds: 100, Now: func() int64 { return 1000 }})
	ctx, cancel := context.WithCancel(context.Background()); cancel()
	w.Run(ctx)
	if fs.inserts != 1 || fs.liqInserts != 1 || fs.liqPrunes != 1 {
		t.Fatalf("KPI + likidite örnek + prune beklenir: %+v", fs)
	}
}
```
(Mevcut `TestWorkerCycleSamplesAndReports` LiqEnabled=false default → liq çağrılmaz; `errSampler`/`fakeSampler` yeni 2 metodu implemente etmeli — derleme için ekle.)

- [ ] **Step 2: Run → fail** — `go test ./internal/trend/ -run TestWorkerCycleSamplesLiquidity -v`.

- [ ] **Step 3: Implement (`worker.go`)** — `Sampler` arayüzüne 2 metot; `WorkerDeps`'e 3 alan. `cycle`: mevcut `sampleOnce` (KPI) çağrısından sonra, `LiqEnabled` ise `liqSampleOnce` çağır; err'leri birleştir (ilk hata Report'a). 
```go
func (w *Worker) cycle(ctx context.Context) {
	n, err := w.sampleOnce(ctx)
	if w.d.LiqEnabled {
		if lerr := w.liqSampleOnce(ctx); lerr != nil && err == nil { err = lerr }
	}
	if err != nil && ctx.Err() == nil { w.d.Logger.Warn("trend sample", "err", err) }
	if w.d.Health != nil { w.d.Health.Report(health.WorkerTrend, err == nil, err, n) }
}
func (w *Worker) liqSampleOnce(ctx context.Context) error {
	if err := w.d.Store.InsertLiquiditySamples(ctx, w.d.Now(), w.d.LiqSampleLimit); err != nil { return err }
	if w.d.LiqKeepSeconds > 0 {
		if err := w.d.Store.PruneLiquiditySamples(ctx, w.d.Now()-w.d.LiqKeepSeconds); err != nil { return err }
	}
	return nil
}
```

- [ ] **Step 4: Run → pass + build** — `go test ./internal/trend/ -race && go build ./...`.

- [ ] **Step 5: Commit** — `git add internal/trend && git commit -m "feat(liq): trend worker likidite örnekleme + prune (Task 2)"`

---

## Task 3: detail.go Series.Liquidity + config + main wiring

**Files:**
- Modify: `internal/market/detail.go`, `internal/config/config.go`, `cmd/server/main.go`
- Test: `internal/market/detail_test.go` (LiquiditySeries dolu/boş), `internal/config/config_test.go`

**Interfaces:**
- Consumes: store `LiquiditySeries`.
- Produces: detail Store arayüzüne `LiquiditySeries`; `TokenDetailDeps.LiqSeriesLimit int`; config `TrendLiq*`.

- [ ] **Step 1: config default test (`config_test.go`)**
```go
func TestTrendLiqDefaults(t *testing.T) {
	t.Setenv("TREND_LIQ_ENABLED",""); t.Setenv("TREND_LIQ_SAMPLE_LIMIT",""); t.Setenv("TREND_LIQ_KEEP_HOURS",""); t.Setenv("TREND_LIQ_SERIES_LIMIT","")
	c := Load()
	if !c.TrendLiqEnabled || c.TrendLiqSampleLimit != 200 || c.TrendLiqKeepHours != 48 || c.TrendLiqSeriesLimit != 500 {
		t.Fatalf("liq defaults: %+v", c)
	}
}
```

- [ ] **Step 2: detail test** — `detail_test.go`'ya mevcut fake/stub Store'a `LiquiditySeries` ekle; bir test: örnekli store → `Series.Liquidity` dolu; `LiqSeriesLimit=0` → boş. (Mevcut detail testinin fake Store'unu bul; metodu ekle. Örnek assertion:)
```go
// stub'a: func (s ...) LiquiditySeries(_ context.Context, mint string, _ int) ([]store.SeriesPoint, error) { return []store.SeriesPoint{{T:1,V:100},{T:2,V:120}}, nil }
// test: LiqSeriesLimit:10 ile servis → d.Series.Liquidity len 2; LiqSeriesLimit:0 → len 0.
```

- [ ] **Step 3: Run → fail** (config + detail).

- [ ] **Step 4: Implement.**
  - `config.go`: `TrendLiqEnabled`/`TrendLiqSampleLimit`/`TrendLiqKeepHours`/`TrendLiqSeriesLimit` + `Load()`: `getenvBool("TREND_LIQ_ENABLED",true)`, `getenvInt("TREND_LIQ_SAMPLE_LIMIT",200)`, `getenvInt("TREND_LIQ_KEEP_HOURS",48)`, `getenvInt("TREND_LIQ_SERIES_LIMIT",500)`.
  - `detail.go`: detail Store arayüzüne `LiquiditySeries(ctx, mint, limit) ([]store.SeriesPoint, error)`; `TokenDetailDeps`'e `LiqSeriesLimit int`; OHLCV sonrası spec §detail.go bloğu (LiqSeriesLimit>0 → doldur, hata → WARN+boş).
  - `main.go`: trend worker construction'a `LiqEnabled: cfg.TrendLiqEnabled, LiqSampleLimit: cfg.TrendLiqSampleLimit, LiqKeepSeconds: int64(cfg.TrendLiqKeepHours)*3600`; detail service construction'a `LiqSeriesLimit: cfg.TrendLiqSeriesLimit`; gates map'e `"TREND_LIQ_ENABLED": cfg.TrendLiqEnabled`.

- [ ] **Step 5: Run → build + tüm testler** — `go build ./... && go test ./... -race && go vet ./...` yeşil.

- [ ] **Step 6: Commit** — `git add internal/market internal/config cmd/server/main.go && git commit -m "feat(liq): detail.go Series.Liquidity + config + main wiring (Task 3)"`

---

## Task 4: Whole-branch review + doküman

- [ ] **Step 1: Tüm testler + build + vet** yeşil.
- [ ] **Step 2: Whole-branch review** — superpowers:requesting-code-review (opus); bulguları superpowers:receiving-code-review ile ele al.
- [ ] **Step 3: Yaşayan dokümanlar** — `docs/progress.md` + `MEMORY.md` + `followups-frontend.md` (holders serisi Helius'a, C radar ertelemesi güncelle).
- [ ] **Step 4: Merge/push** — kullanıcı onayıyla (DUR-noktası).

---

## Self-Review

**1. Spec coverage:** migration+store → Task 1 ✅; worker örnekleme → Task 2 ✅; detail+config+wiring → Task 3 ✅; review+docs → Task 4 ✅.
**2. Placeholder scan:** SQL + test kodları somut; detail fake Store metodu "mevcut testin fake'ine ekle" — task-anında dosyaya bakılır (iskelet+imza tam).
**3. Type consistency:** `InsertLiquiditySamples(ts,limit)`/`PruneLiquiditySamples(cutoff)`/`LiquiditySeries(mint,limit)` Task 1 → Task 2 (Sampler) + Task 3 (detail Store); `SeriesPoint{T,V}` mevcut; `LiqSampleLimit`/`LiqKeepSeconds`/`LiqSeriesLimit`/`Trend Liq*` tutarlı.
