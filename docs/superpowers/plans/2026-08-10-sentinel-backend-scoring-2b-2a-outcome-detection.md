# Slice 2b-2a — Token Outcome Detection + Peak Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Her tokenin akıbetini (active/graduated/dumped/rug/dead) piyasa trajektörisinden sınıflandır (peak takibi + saf sınıflandırıcı + arka plan worker), ve Creator token geçmişinde gerçek outcome/peakMarketCap/maxDrawdownPct/liquidityStatus göster.

**Architecture:** Enricher marketCap/likidite tepesini `GREATEST` ile running-max saklar (migration 0007). Saf `Classify` (ağsız, config eşikli) tekil tokenin cur+peak durumundan outcome üretir. Arka plan `outcome.Worker` (2a safety Worker deseni, **Helius YOK**) periyodik olarak pool'lu token'ları sınıflar → DB persist. `CreatorDetail` bu alanları DB'den okur (2b-1'de nötrdü). Frontend'e sıfır dokunuş.

**Tech Stack:** Go (chi, database/sql, pgx/v5, goose). Yeni bağımlılık YOK; Helius YOK; harici çağrı YOK (saf market verisi mevcut GeckoTerminal enrichment'ından).

## Global Constraints

- **Go sürümü:** `go 1.24` (go.mod + CI ile eşleşir — değiştirme). `max()` built-in kullanılabilir (go 1.21+).
- **Honesty invariant:** gerçek olmayan alan yok; nötr/deferred alanlar açık işaretli. `outcome`/`liquidityStatus` değerleri **geçerli frontend enum key'leri** olmalı — frontend `OUTCOME_DEFS[outcome]`/`LIQUIDITY_DEFS[liquidityStatus]` sözlük araması yapar; geçersiz/boş değer UI'ı çökertir. Geçerli değerler: outcome ∈ {active, graduated, dumped, rug, dead}; liquidityStatus ∈ {unlocked, removed} ("locked" bu dilimde KULLANILMAZ — on-chain LP-lock doğrulaması yok).
- **Frontend kontratı sabit:** `TokenRow` değişmez; `CreatorTokenHistoryItem` JSON şekli `apps/web/lib/api/types.ts` ile birebir. **Frontend'e sıfır dokunuş** (seam alanları taşıyor).
- **Fake ↔ postgres parity:** her yeni store metodu (peak, OutcomeTargets, UpdateOutcome, CreatorDetail okuma) fake ve postgres'te aynı davranır; testler bunu doğrular.
- **Eşikler config'ten** (deploy'da kalibrasyon env ile, kod değişmeden — OCP). Saf sınıflandırıcı yan-etkisiz.
- **DB round-trip yalnız deploy'da** doğrulanır (yerel Postgres yok — 0/1/2a/2b-1 deseni); postgres testleri DATABASE_URL yoksa skip.
- **Test komutları:** Go `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`.
- NOT: `gofmt -l` repo'da CRLF artefaktı yüzünden çoğu dosyayı işaretler (pre-existing Windows) — bu gürültüyü flag etme; yalnız bu diff'in gerçekten bozduğu formatı.

---

### Task 1: Saf Classifier (`internal/outcome/classifier.go`)

**Files:**
- Create: `apps/api-go/internal/outcome/classifier.go`
- Test: `apps/api-go/internal/outcome/classifier_test.go`

**Interfaces:**
- Consumes: hiçbir şey (saf, ağsız).
- Produces: `outcome.Input`, `outcome.Result`, `outcome.Thresholds` struct'ları + `outcome.Classify(Input, Thresholds) Result` + outcome/liquidity sabitleri. Task 3 (worker) ve Task 2 (store test dolaylı) kullanır.

- [ ] **Step 1: Write the failing test**

`internal/outcome/classifier_test.go`:

```go
package outcome

import "testing"

func defThresholds() Thresholds {
	return Thresholds{RugLiqRatio: 0.10, GraduationMcap: 69000, DumpedDrawdown: 80, DeadVol: 100, MinLiqFloor: 500, DeadAgeSec: 86400}
}

func TestClassify(t *testing.T) {
	th := defThresholds()
	cases := []struct {
		name string
		in   Input
		want string
	}{
		{"rug: peak likidite yüksek, cur ~0", Input{CurMarketCap: 1000, CurLiquidity: 10, PeakMarketCap: 50000, PeakLiquidity: 20000, Vol24h: 5000, AgeSeconds: 3600}, OutcomeRug},
		{"graduated: peak mcap eşik üstü + likit", Input{CurMarketCap: 80000, CurLiquidity: 30000, PeakMarketCap: 90000, PeakLiquidity: 30000, Vol24h: 40000, AgeSeconds: 7200}, OutcomeGraduated},
		{"dumped: drawdown yüksek, likidite duruyor", Input{CurMarketCap: 5000, CurLiquidity: 8000, PeakMarketCap: 50000, PeakLiquidity: 9000, Vol24h: 3000, AgeSeconds: 7200}, OutcomeDumped},
		{"dead: yaşlı + vol~0", Input{CurMarketCap: 200, CurLiquidity: 600, PeakMarketCap: 800, PeakLiquidity: 700, Vol24h: 10, AgeSeconds: 200000}, OutcomeDead},
		{"active: taze + likit", Input{CurMarketCap: 3000, CurLiquidity: 4000, PeakMarketCap: 3200, PeakLiquidity: 4000, Vol24h: 6000, AgeSeconds: 600}, OutcomeActive},
		{"rug graduated'ı ezer (mezun sonrası rug)", Input{CurMarketCap: 2000, CurLiquidity: 50, PeakMarketCap: 90000, PeakLiquidity: 30000, Vol24h: 1000, AgeSeconds: 9000}, OutcomeRug},
	}
	for _, c := range cases {
		if got := Classify(c.in, th).Outcome; got != c.want {
			t.Errorf("%s: outcome = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestClassifyDrawdownAndLiquidityStatus(t *testing.T) {
	th := defThresholds()
	// peak=0 → drawdown 0, liquidityStatus unlocked, active.
	r := Classify(Input{CurMarketCap: 0, CurLiquidity: 0, PeakMarketCap: 0, PeakLiquidity: 0, Vol24h: 0, AgeSeconds: 10}, th)
	if r.MaxDrawdownPct != 0 || r.LiquidityStatus != LiquidityUnlocked || r.Outcome != OutcomeActive {
		t.Fatalf("peak=0 durumu: %+v", r)
	}
	// rug → liquidityStatus removed + drawdown hesaplanır.
	r = Classify(Input{CurMarketCap: 5000, CurLiquidity: 10, PeakMarketCap: 20000, PeakLiquidity: 15000, Vol24h: 100, AgeSeconds: 3600}, th)
	if r.LiquidityStatus != LiquidityRemoved {
		t.Fatalf("rug liquidityStatus = %q, want removed", r.LiquidityStatus)
	}
	if r.MaxDrawdownPct != 75 { // (20000-5000)/20000*100
		t.Fatalf("drawdown = %v, want 75", r.MaxDrawdownPct)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/outcome/ -run TestClassify -v`
Expected: FAIL (paket/`Classify` yok).

- [ ] **Step 3: Implement the classifier**

`internal/outcome/classifier.go`:

```go
// Package outcome, tokenin piyasa trajektörisinden akıbetini (outcome) sınıflandırır.
package outcome

// Outcome ve liquidityStatus için geçerli frontend enum key'leri (apps/web/lib/creator/outcome-defs.ts).
const (
	OutcomeActive    = "active"
	OutcomeGraduated = "graduated"
	OutcomeDumped    = "dumped"
	OutcomeRug       = "rug"
	OutcomeDead      = "dead"

	LiquidityUnlocked = "unlocked"
	LiquidityRemoved  = "removed"
)

// Input, tekil tokenin anlık + tepe piyasa durumudur.
type Input struct {
	CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h float64
	AgeSeconds int64
}

// Result, sınıflandırma çıktısıdır.
type Result struct {
	Outcome         string
	MaxDrawdownPct  float64
	LiquidityStatus string
}

// Thresholds, sınıflandırma eşikleridir (config'ten; deploy'da kalibre edilir).
type Thresholds struct {
	RugLiqRatio, GraduationMcap, DumpedDrawdown, DeadVol, MinLiqFloor float64
	DeadAgeSec                                                        int64
}

// Classify, öncelikli-eşleşme (ilk eşleşen kazanır) ile outcome üretir. Sıra: rug → graduated
// → dumped → dead → active (terminal/kötü sinyaller önce).
func Classify(in Input, t Thresholds) Result {
	drawdown := 0.0
	if in.PeakMarketCap > 0 {
		drawdown = (in.PeakMarketCap - in.CurMarketCap) / in.PeakMarketCap * 100
		if drawdown < 0 {
			drawdown = 0
		}
		if drawdown > 100 {
			drawdown = 100
		}
	}
	switch {
	case in.PeakLiquidity >= t.MinLiqFloor && in.CurLiquidity <= in.PeakLiquidity*t.RugLiqRatio:
		return Result{OutcomeRug, drawdown, LiquidityRemoved} // LP çekildi
	case in.PeakMarketCap >= t.GraduationMcap && in.CurLiquidity >= t.MinLiqFloor:
		return Result{OutcomeGraduated, drawdown, LiquidityUnlocked} // yüksek-cap + likit
	case drawdown >= t.DumpedDrawdown && in.CurLiquidity >= t.MinLiqFloor:
		return Result{OutcomeDumped, drawdown, LiquidityUnlocked} // fiyat çöktü, likidite duruyor
	case in.AgeSeconds >= t.DeadAgeSec && in.Vol24h <= t.DeadVol:
		return Result{OutcomeDead, drawdown, LiquidityUnlocked} // yaşlı + işlemsiz
	default:
		return Result{OutcomeActive, drawdown, LiquidityUnlocked}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api-go && go test ./internal/outcome/ -race -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/outcome/classifier.go apps/api-go/internal/outcome/classifier_test.go
git commit -m "feat(outcome): saf 5'li outcome sınıflandırıcı (rug/graduated/dumped/dead/active)"
```

---

### Task 2: Store — migration 0007 + peak takibi + OutcomeTargets/UpdateOutcome

**Files:**
- Create: `apps/api-go/internal/store/migrations/0007_add_token_outcome.sql`
- Modify: `apps/api-go/internal/store/tokens.go` (OutcomeTarget/OutcomeUpdate struct'ları + TokenStore interface += 2 metot + postgres UpdateMarket peak + postgres OutcomeTargets/UpdateOutcome)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fakeTok yeni alanlar + insert default'ları + UpdateMarket peak + OutcomeTargets/UpdateOutcome)
- Test: `apps/api-go/internal/store/outcome_test.go`

**Interfaces:**
- Consumes: mevcut `MarketUpdate` (peak için struct değişmez — `MarketCapUSD`+`Liquidity` taşır).
- Produces:
  - `type OutcomeTarget struct { Mint string; CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h float64; FirstSeenTs int64 }`
  - `type OutcomeUpdate struct { Mint, Outcome, LiquidityStatus string; MaxDrawdownPct float64; ScoredTs int64 }`
  - `TokenStore.OutcomeTargets(ctx, limit) ([]OutcomeTarget, error)` + `TokenStore.UpdateOutcome(ctx, OutcomeUpdate) error`. Task 3 worker (via `outcome.OutcomeStore`) kullanır.

- [ ] **Step 1: Write the failing test**

`internal/store/outcome_test.go`:

```go
package store

import (
	"context"
	"testing"
)

func TestPeakTrackingMonotonic(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore() // TokenStore — OutcomeTargets/UpdateOutcome içerir
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "m1", PoolAddr: "p1", FirstSeenTs: 100})
	_ = ts.UpdateMarket(ctx, MarketUpdate{Mint: "m1", MarketCapUSD: 100, Liquidity: 200})
	_ = ts.UpdateMarket(ctx, MarketUpdate{Mint: "m1", MarketCapUSD: 50, Liquidity: 80}) // düşük — peak korunur
	tgs, err := ts.OutcomeTargets(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tgs) != 1 {
		t.Fatalf("hedef sayısı = %d, want 1", len(tgs))
	}
	tg := tgs[0]
	if tg.PeakMarketCap != 100 || tg.PeakLiquidity != 200 {
		t.Fatalf("peak = mcap %v / liq %v, want 100/200 (düşmemeli)", tg.PeakMarketCap, tg.PeakLiquidity)
	}
	if tg.CurMarketCap != 50 || tg.CurLiquidity != 80 {
		t.Fatalf("cur = mcap %v / liq %v, want 50/80", tg.CurMarketCap, tg.CurLiquidity)
	}
}

func TestUpdateOutcomeAndTargetsOrdering(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "a", PoolAddr: "pa", FirstSeenTs: 100})
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "b", PoolAddr: "pb", FirstSeenTs: 90})
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "c", FirstSeenTs: 80}) // pool yok → hedef değil

	if err := ts.UpdateOutcome(ctx, OutcomeUpdate{Mint: "a", Outcome: "rug", LiquidityStatus: "removed", MaxDrawdownPct: 90, ScoredTs: 500}); err != nil {
		t.Fatal(err)
	}
	tgs, err := ts.OutcomeTargets(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tgs) != 2 {
		t.Fatalf("hedef sayısı = %d, want 2 (pool'suz hariç)", len(tgs))
	}
	// en eski outcome_scored_ts önce: b (0) a'dan (500) önce gelmeli.
	if tgs[0].Mint != "b" {
		t.Fatalf("ilk hedef = %q, want b (henüz skorlanmamış, ts=0 en eski)", tgs[0].Mint)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/store/ -run "TestPeakTracking|TestUpdateOutcome" -v`
Expected: FAIL (derleme: `OutcomeTargets`/`UpdateOutcome`/`OutcomeTarget` yok).

- [ ] **Step 3: Migration 0007**

`internal/store/migrations/0007_add_token_outcome.sql`:

```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS peak_market_cap   DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS peak_liquidity    DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS outcome           TEXT NOT NULL DEFAULT 'active';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS max_drawdown_pct  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS liquidity_status  TEXT NOT NULL DEFAULT 'unlocked';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS outcome_scored_ts BIGINT NOT NULL DEFAULT 0;
-- Peak seed: mevcut token'lar için tepeyi güncelden başlat (gerçek tarihsel tepe kayıp;
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

- [ ] **Step 4: Store structs + interface + postgres impl**

`internal/store/tokens.go` — `SafetyTarget` yakınına struct'lar ekle:

```go
// OutcomeTarget, sınıflandırılacak token için gereken anlık + tepe piyasa durumudur.
type OutcomeTarget struct {
	Mint                                                            string
	CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h float64
	FirstSeenTs                                                     int64
}

// OutcomeUpdate, outcome sınıflandırıcısının yazdığı sonuçtur.
type OutcomeUpdate struct {
	Mint, Outcome, LiquidityStatus string
	MaxDrawdownPct                 float64
	ScoredTs                       int64
}
```

`TokenStore` interface'ine ekle (safety metotlarının yanına):

```go
	OutcomeTargets(ctx context.Context, limit int) ([]OutcomeTarget, error)
	UpdateOutcome(ctx context.Context, u OutcomeUpdate) error
```

postgres `UpdateMarket` SQL'ine peak running-max ekle (mevcut sorguyu değiştir):

```go
	const q = `UPDATE tokens SET price=$2, liquidity=$3, vol5m=$4, momentum=$5, spark=$6,
		price_change_h24=$7, market_cap_usd=$8, vol24h=$9,
		peak_market_cap = GREATEST(peak_market_cap, $8),
		peak_liquidity  = GREATEST(peak_liquidity, $3)
		WHERE mint=$1`
```

postgres yeni metotlar (`tokens.go`):

```go
func (p *postgresStore) OutcomeTargets(ctx context.Context, limit int) ([]OutcomeTarget, error) {
	const q = `SELECT mint, market_cap_usd, liquidity, peak_market_cap, peak_liquidity, vol24h, first_seen_ts
		FROM tokens WHERE pool_address <> ''
		ORDER BY outcome_scored_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OutcomeTarget, 0, limit)
	for rows.Next() {
		var t OutcomeTarget
		if err := rows.Scan(&t.Mint, &t.CurMarketCap, &t.CurLiquidity, &t.PeakMarketCap,
			&t.PeakLiquidity, &t.Vol24h, &t.FirstSeenTs); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) UpdateOutcome(ctx context.Context, u OutcomeUpdate) error {
	const q = `UPDATE tokens SET outcome=$2, liquidity_status=$3, max_drawdown_pct=$4, outcome_scored_ts=$5 WHERE mint=$1`
	_, err := p.db.ExecContext(ctx, q, u.Mint, u.Outcome, u.LiquidityStatus, u.MaxDrawdownPct, u.ScoredTs)
	return err
}
```

- [ ] **Step 5: Fake impl (parity)**

`internal/store/fake_ingest.go` — `fakeTok` struct'a alanlar ekle:

```go
	// 2b-2a outcome + peak
	peakMarketCap, peakLiquidity, maxDrawdownPct float64
	outcome, liquidityStatus                     string
	outcomeScoredTs                              int64
```

`UpsertToken` insert dalında (`if !ok {`) ve `UpsertDiscovered` insert dalında (`if inserted {`) default'ları set et (DB DEFAULT parity):

```go
		cur.outcome = "active"
		cur.liquidityStatus = "unlocked"
```

Fake `UpdateMarket`'a peak running-max ekle (mevcut alan atamalarının yanına, `f.byID[m.Mint] = cur`'dan önce):

```go
	cur.peakMarketCap = max(cur.peakMarketCap, m.MarketCapUSD)
	cur.peakLiquidity = max(cur.peakLiquidity, m.Liquidity)
```

Fake yeni metotlar:

```go
func (f *fakeTokenStore) OutcomeTargets(_ context.Context, limit int) ([]OutcomeTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].outcomeScoredTs < f.byID[ids[j]].outcomeScoredTs // en eski önce
	})
	out := make([]OutcomeTarget, 0, limit)
	for _, id := range ids {
		t := f.byID[id]
		if t.poolAddr == "" || len(out) >= limit {
			continue
		}
		out = append(out, OutcomeTarget{
			Mint: t.row.Mint, CurMarketCap: t.marketCapUSD, CurLiquidity: t.row.Liquidity,
			PeakMarketCap: t.peakMarketCap, PeakLiquidity: t.peakLiquidity, Vol24h: t.vol24h, FirstSeenTs: t.firstSeen,
		})
	}
	return out, nil
}

func (f *fakeTokenStore) UpdateOutcome(_ context.Context, u OutcomeUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[u.Mint]
	if !ok {
		return nil
	}
	cur.outcome, cur.liquidityStatus = u.Outcome, u.LiquidityStatus
	cur.maxDrawdownPct, cur.outcomeScoredTs = u.MaxDrawdownPct, u.ScoredTs
	f.byID[u.Mint] = cur
	return nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./internal/store/ -race`
Expected: PASS (peak monotonic + outcome update/targets ordering + mevcut store testleri).

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/store/
git commit -m "feat(store): migration 0007 outcome/peak kolonları + peak GREATEST + OutcomeTargets/UpdateOutcome"
```

---

### Task 3: Outcome Worker (`internal/outcome/worker.go`)

**Files:**
- Create: `apps/api-go/internal/outcome/worker.go`
- Test: `apps/api-go/internal/outcome/worker_test.go`

**Interfaces:**
- Consumes: `outcome.Classify`/`Input`/`Thresholds` (Task 1); `store.OutcomeTarget`/`store.OutcomeUpdate` (Task 2).
- Produces: `outcome.OutcomeStore` arayüzü (DIP; `store.TokenStore` karşılar) + `outcome.WorkerDeps`/`Worker`/`NewWorker`/`Run`. Task 4 (main) wire eder.

- [ ] **Step 1: Write the failing test**

`internal/outcome/worker_test.go`:

```go
package outcome

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeOutcomeStore struct {
	targets []store.OutcomeTarget
	updates []store.OutcomeUpdate
}

func (f *fakeOutcomeStore) OutcomeTargets(_ context.Context, _ int) ([]store.OutcomeTarget, error) {
	return f.targets, nil
}
func (f *fakeOutcomeStore) UpdateOutcome(_ context.Context, u store.OutcomeUpdate) error {
	f.updates = append(f.updates, u)
	return nil
}

func TestWorkerClassifiesAndPersists(t *testing.T) {
	fs := &fakeOutcomeStore{targets: []store.OutcomeTarget{
		{Mint: "rugged", CurMarketCap: 1000, CurLiquidity: 10, PeakMarketCap: 50000, PeakLiquidity: 20000, Vol24h: 100, FirstSeenTs: 0},
		{Mint: "fresh", CurMarketCap: 3000, CurLiquidity: 4000, PeakMarketCap: 3200, PeakLiquidity: 4000, Vol24h: 6000, FirstSeenTs: 500},
	}}
	w := NewWorker(WorkerDeps{
		Store:      fs,
		Thresholds: Thresholds{RugLiqRatio: 0.10, GraduationMcap: 69000, DumpedDrawdown: 80, DeadVol: 100, MinLiqFloor: 500, DeadAgeSec: 86400},
		Now:        func() int64 { return 1000 },
	})
	if err := w.classifyOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fs.updates) != 2 {
		t.Fatalf("update sayısı = %d, want 2", len(fs.updates))
	}
	byMint := map[string]store.OutcomeUpdate{}
	for _, u := range fs.updates {
		byMint[u.Mint] = u
	}
	if byMint["rugged"].Outcome != OutcomeRug || byMint["rugged"].LiquidityStatus != LiquidityRemoved {
		t.Fatalf("rugged = %+v", byMint["rugged"])
	}
	if byMint["fresh"].Outcome != OutcomeActive {
		t.Fatalf("fresh = %+v", byMint["fresh"])
	}
	if byMint["rugged"].ScoredTs != 1000 {
		t.Fatalf("ScoredTs = %d, want 1000 (Now())", byMint["rugged"].ScoredTs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/outcome/ -run TestWorker -v`
Expected: FAIL (`NewWorker`/`WorkerDeps`/`classifyOnce` yok).

- [ ] **Step 3: Implement the worker (safety.Worker deseni, Helius yok)**

`internal/outcome/worker.go`:

```go
package outcome

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// OutcomeStore, Worker'ın kalıcılık bağımlılığıdır (DIP; store.TokenStore karşılar).
type OutcomeStore interface {
	OutcomeTargets(ctx context.Context, limit int) ([]store.OutcomeTarget, error)
	UpdateOutcome(ctx context.Context, u store.OutcomeUpdate) error
}

type WorkerDeps struct {
	Store      OutcomeStore
	Thresholds Thresholds
	Interval   time.Duration
	Limit      int
	Now        func() int64
	Logger     *slog.Logger
}

// Worker, periyodik olarak pool'lu token'ları çekip sınıflayıp DB'ye yazar (Enricher deseni; dış çağrı yok).
type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Interval <= 0 {
		d.Interval = 60 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 60
	}
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	if err := w.classifyOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("outcome classify", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.classifyOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("outcome classify", "err", err)
			}
		}
	}
}

// classifyOnce, bir döngü: hedefleri çek → her birini sınıfla → persist (kısmi hata izole).
func (w *Worker) classifyOnce(ctx context.Context) error {
	targets, err := w.d.Store.OutcomeTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		res := Classify(Input{
			CurMarketCap: tg.CurMarketCap, CurLiquidity: tg.CurLiquidity,
			PeakMarketCap: tg.PeakMarketCap, PeakLiquidity: tg.PeakLiquidity,
			Vol24h: tg.Vol24h, AgeSeconds: now - tg.FirstSeenTs,
		}, w.d.Thresholds)
		if err := w.d.Store.UpdateOutcome(ctx, store.OutcomeUpdate{
			Mint: tg.Mint, Outcome: res.Outcome, LiquidityStatus: res.LiquidityStatus,
			MaxDrawdownPct: res.MaxDrawdownPct, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("update outcome", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go test ./internal/outcome/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/outcome/worker.go apps/api-go/internal/outcome/worker_test.go
git commit -m "feat(outcome): arka plan worker (OutcomeStore DIP, safety Worker deseni, Helius yok)"
```

---

### Task 4: Config + main wiring

**Files:**
- Modify: `apps/api-go/internal/config/config.go` (OUTCOME_* alanları + `getenvFloat` helper)
- Modify: `apps/api-go/cmd/server/main.go` (outcome worker goroutine)
- Test: `apps/api-go/internal/config/config_test.go` (OUTCOME default'ları)

**Interfaces:**
- Consumes: `outcome.NewWorker`/`WorkerDeps`/`Thresholds` (Task 3); `cfg` alanları.
- Produces: config OUTCOME_* alanları; main.go'da config-gated `outcome.Worker` goroutine.

- [ ] **Step 1: Write the failing test**

`internal/config/config_test.go`'ye ekle (mevcut default-test desenini izle):

```go
func TestLoadOutcomeDefaults(t *testing.T) {
	c := Load()
	if !c.OutcomeEnabled || c.OutcomeIntervalSec != 60 || c.OutcomeLimit != 60 {
		t.Fatalf("outcome worker defaults: %+v", c)
	}
	if c.OutcomeRugLiqRatio != 0.10 || c.OutcomeGraduationMcap != 69000 || c.OutcomeDumpedDrawdown != 80 {
		t.Fatalf("outcome eşik defaults: %+v", c)
	}
	if c.OutcomeDeadVol != 100 || c.OutcomeMinLiqFloor != 500 || c.OutcomeDeadAgeSec != 86400 {
		t.Fatalf("outcome dead/floor defaults: %+v", c)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/config/ -run TestLoadOutcome -v`
Expected: FAIL (alanlar yok).

- [ ] **Step 3: Config alanları + getenvFloat**

`internal/config/config.go` — `Config` struct'a (CreatorsListLimit'ten sonra) ekle:

```go
	OutcomeEnabled        bool
	OutcomeIntervalSec    int
	OutcomeLimit          int
	OutcomeRugLiqRatio    float64
	OutcomeGraduationMcap float64
	OutcomeDumpedDrawdown float64
	OutcomeDeadVol        float64
	OutcomeMinLiqFloor    float64
	OutcomeDeadAgeSec     int
```

`Load()` içine ekle:

```go
		OutcomeEnabled:        getenvBool("OUTCOME_ENABLED", true),
		OutcomeIntervalSec:    getenvInt("OUTCOME_INTERVAL_SEC", 60),
		OutcomeLimit:          getenvInt("OUTCOME_LIMIT", 60),
		OutcomeRugLiqRatio:    getenvFloat("OUTCOME_RUG_LIQ_RATIO", 0.10),
		OutcomeGraduationMcap: getenvFloat("OUTCOME_GRADUATION_MCAP", 69000),
		OutcomeDumpedDrawdown: getenvFloat("OUTCOME_DUMPED_DRAWDOWN", 80),
		OutcomeDeadVol:        getenvFloat("OUTCOME_DEAD_VOL", 100),
		OutcomeMinLiqFloor:    getenvFloat("OUTCOME_MIN_LIQ_FLOOR", 500),
		OutcomeDeadAgeSec:     getenvInt("OUTCOME_DEAD_AGE_SEC", 86400),
```

`getenvFloat` helper ekle (getenvBool'un yanına):

```go
func getenvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}
```

- [ ] **Step 4: main.go wiring**

`cmd/server/main.go` — safety worker bloğunun (satır ~120-130) hemen ardına, `srv := &http.Server{...}`'den önce ekle:

```go
	// token outcome sınıflandırıcı (2b-2a) — arka plan; Helius gerekmez (saf market verisi)
	if cfg.OutcomeEnabled && bundle.Tokens != nil {
		ow := outcome.NewWorker(outcome.WorkerDeps{
			Store: bundle.Tokens,
			Thresholds: outcome.Thresholds{
				RugLiqRatio: cfg.OutcomeRugLiqRatio, GraduationMcap: cfg.OutcomeGraduationMcap,
				DumpedDrawdown: cfg.OutcomeDumpedDrawdown, DeadVol: cfg.OutcomeDeadVol,
				MinLiqFloor: cfg.OutcomeMinLiqFloor, DeadAgeSec: int64(cfg.OutcomeDeadAgeSec),
			},
			Interval: time.Duration(cfg.OutcomeIntervalSec) * time.Second, Limit: cfg.OutcomeLimit, Logger: logger,
		})
		go ow.Run(ctx)
	}
```

`import` bloğuna `"github.com/furkanatesc/sentinel/apps/api-go/internal/outcome"` ekle. `bundle.Tokens` (`store.TokenStore`) `OutcomeTargets`/`UpdateOutcome` içerdiği için `outcome.OutcomeStore`'u yapısal karşılar.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./internal/config/ ./internal/outcome/ -race`
Expected: PASS (config default'ları + mevcut testler; main derlenir).

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/config/ apps/api-go/cmd/server/main.go
git commit -m "feat(outcome): config OUTCOME_* + getenvFloat + main worker goroutine wiring"
```

---

### Task 5: Read wiring — CreatorDetail gerçek outcome/peak/drawdown

**Files:**
- Modify: `apps/api-go/internal/store/creators.go` (`newHistoryItem` imzası + postgres `CreatorDetail` sorgusu)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake `CreatorDetail` yeni alanları taşır)
- Test: `apps/api-go/internal/store/creators_test.go` (mevcut `TestCreatorDetail` genişlet ya da yeni test)

**Interfaces:**
- Consumes: `tokens` outcome/peak kolonları (Task 2); `fakeTok` outcome/peak alanları (Task 2).
- Produces: `CreatorTokenHistoryItem` artık gerçek `Outcome`/`PeakMarketCap`/`MaxDrawdownPct`/`LiquidityStatus` taşır (`CreatorSellPct` nötr kalır).

- [ ] **Step 1: Write the failing test**

`internal/store/creators_test.go`'ye ekle:

```go
func TestCreatorDetailCarriesOutcome(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 1000, "AAA")
	_ = ts.UpdateMarket(ctx, MarketUpdate{Mint: "m1", MarketCapUSD: 10000, Liquidity: 500})
	_ = ts.UpdateOutcome(ctx, OutcomeUpdate{Mint: "m1", Outcome: "rug", LiquidityStatus: "removed", MaxDrawdownPct: 88, ScoredTs: 2000})

	p, ok, err := ts.(CreatorStore).CreatorDetail(ctx, "AAA")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	h := p.History[0]
	if h.Outcome != "rug" || h.LiquidityStatus != "removed" || h.MaxDrawdownPct != 88 {
		t.Fatalf("history outcome alanları = %+v (rug/removed/88 bekleniyor)", h)
	}
	if h.PeakMarketCap != 10000 { // UpdateMarket peak'i seed etti (GREATEST 0→10000)
		t.Fatalf("peakMarketCap = %v, want 10000", h.PeakMarketCap)
	}
	if h.CreatorSellPct != 0 { // nötr → 2c (trade-flow)
		t.Fatalf("creatorSellPct = %v, want 0 (nötr)", h.CreatorSellPct)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/store/ -run TestCreatorDetailCarriesOutcome -v`
Expected: FAIL (history hâlâ nötr "active"/"unlocked"/0 döner).

- [ ] **Step 3: newHistoryItem imzası (creators.go)**

`newHistoryItem`'ı gerçek alanları taşıyacak şekilde değiştir:

```go
// newHistoryItem, gerçek piyasa + outcome alanlarını doldurur; creatorSellPct nötr (→ 2c trade-flow).
func newHistoryItem(mint, symbol string, firstSeenTs int64, currentMcap, peakMcap, maxDrawdown float64, outcome, liquidityStatus string) CreatorTokenHistoryItem {
	return CreatorTokenHistoryItem{
		ID: mint, Symbol: symbol, Mint: mint,
		CreatedAt:        time.Unix(firstSeenTs, 0).UTC().Format(time.RFC3339),
		CurrentMarketCap: currentMcap,
		PeakMarketCap:    peakMcap,
		MaxDrawdownPct:   maxDrawdown,
		LiquidityStatus:  liquidityStatus,
		Outcome:          outcome,
		RiskFlags:        []string{}, // nötr → 2b-2b (risk bayrakları itibar skoruyla)
	}
}
```

- [ ] **Step 4: postgres CreatorDetail sorgusu (creators.go)**

`CreatorDetail` history sorgusunu + scan'i güncelle:

```go
	rows, err := p.db.QueryContext(ctx,
		`SELECT mint, symbol, first_seen_ts, market_cap_usd, peak_market_cap, max_drawdown_pct, outcome, liquidity_status
		 FROM tokens WHERE creator=$1 ORDER BY first_seen_ts DESC`, address)
	if err != nil {
		return CreatorProfile{}, false, err
	}
	defer rows.Close()
	history := make([]CreatorTokenHistoryItem, 0, total)
	for rows.Next() {
		var mint, symbol, outcome, liqStatus string
		var ts int64
		var mcap, peakMcap, drawdown float64
		if err := rows.Scan(&mint, &symbol, &ts, &mcap, &peakMcap, &drawdown, &outcome, &liqStatus); err != nil {
			return CreatorProfile{}, false, err
		}
		history = append(history, newHistoryItem(mint, symbol, ts, mcap, peakMcap, drawdown, outcome, liqStatus))
	}
```

- [ ] **Step 5: fake CreatorDetail (fake_ingest.go)**

Fake `CreatorDetail` içindeki `newHistoryItem` çağrısını yeni alanlarla güncelle:

```go
	for _, tk := range matches {
		history = append(history, newHistoryItem(tk.row.Mint, tk.row.Symbol, tk.firstSeen,
			tk.marketCapUSD, tk.peakMarketCap, tk.maxDrawdownPct, tk.outcome, tk.liquidityStatus))
	}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go test ./internal/store/ -race`
Expected: PASS (yeni outcome-history testi + mevcut CreatorDetail/creators testleri).

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/store/creators.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/creators_test.go
git commit -m "feat(store): CreatorDetail geçmişi gerçek outcome/peak/drawdown/liquidityStatus okur"
```

---

### Task 6: Docs + yaşayan ledger + review handoff

**Files:**
- Modify: `apps/api-go/README.md` (OUTCOME_* env)
- Modify: `docs/progress.md` (Backend programı → 2b-2a kaydı)
- Modify: `docs/superpowers/followups-frontend.md` (2b-2a ertelenenler)

**Interfaces:** yok (yalnız docs).

- [ ] **Step 1: README env**

`apps/api-go/README.md` env tablosuna ekle (mevcut desen): `OUTCOME_ENABLED`(true), `OUTCOME_INTERVAL_SEC`(60), `OUTCOME_LIMIT`(60), `OUTCOME_RUG_LIQ_RATIO`(0.10), `OUTCOME_GRADUATION_MCAP`(69000), `OUTCOME_DUMPED_DRAWDOWN`(80), `OUTCOME_DEAD_VOL`(100), `OUTCOME_MIN_LIQ_FLOOR`(500), `OUTCOME_DEAD_AGE_SEC`(86400) — "outcome sınıflandırma eşikleri, deploy'da kalibre edilebilir".

- [ ] **Step 2: progress.md ledger**

`docs/progress.md` Backend bölümüne 2b-2a kaydı: peak takibi (migration 0007) + 5'li outcome sınıflandırıcı + arka plan worker (Helius yok) → CreatorDetail geçmişi gerçek outcome/drawdown/liquidityStatus; itibar skoru/walletAge/creatorHoldingPct → 2b-2b; eşik-kalibrasyon deploy'da.

- [ ] **Step 3: followups-frontend.md — 2b-2a ertelenenler**

Ekle (sessiz düşürme yok):
- İtibar skoru + metrics + walletAgeDays + creatorHoldingPct → 2b-2b.
- `creatorSellPct` (per-token creator satış %) → trade-flow (2c/2e).
- `behavior.*` → trade-flow + funding-graph (2c/2e).
- `liquidityStatus="locked"` kullanılmıyor → on-chain LP-lock doğrulaması (gelecek).
- Gerçek pump.fun mezuniyet (Raydium migration) tespiti → marketCap-eşiği proxy kullanıldı.
- Peak seed conservative (migration'dan itibaren); gerçek tarihsel tepe kayıp.
- Outcome eşikleri deploy'da gerçek dağılıma göre kalibre edilecek (`OUTCOME_*`).

- [ ] **Step 4: Commit**

```bash
git add apps/api-go/README.md docs/progress.md docs/superpowers/followups-frontend.md
git commit -m "docs(scoring): 2b-2a outcome detection — README + progress ledger + followups"
```

- [ ] **Step 5: Whole-branch review handoff**

`superpowers:requesting-code-review` ile tüm branch review'ı (opus). En yüksek-riskli kontroller:
1. **Classifier kural sırası** — rug graduated'ı ezer; drawdown clamp + peak=0 guard; eşik sınırları.
2. **Peak GREATEST** — `UpdateMarket` peak yalnız yükselir (postgres `GREATEST($8,$3)` param eşleşmesi doğru: $8=market_cap, $3=liquidity), fake `max()` parity.
3. **Enum geçerliliği** — outcome/liquidityStatus yalnız geçerli frontend key'leri (`OUTCOME_DEFS`/`LIQUIDITY_DEFS`); fake insert default'ları "active"/"unlocked".
4. **Fake↔postgres parity** — OutcomeTargets ordering (en eski outcome_scored_ts önce, pool'suz hariç), UpdateOutcome, CreatorDetail okuma.
5. **Interface tutarlılığı** — `store.TokenStore` += 2 metot her iki impl'de; `outcome.OutcomeStore` dar arayüz `bundle.Tokens`'ı karşılar; `TokenRow` değişmedi.

Review temizse → kullanıcıya merge onayı sor (DUR — merge/deploy kullanıcı onayı ister).

---

## Kabul Kriterleri (deploy sonrası doğrulanır)

1. Migration 0007 goose ile uygulanır (yeni zorunlu env yok; peak seed mevcut token'lara).
2. Enricher birkaç tick sonra `peak_market_cap`/`peak_liquidity` ≥ güncel.
3. Outcome worker token'ları sınıflar; yaşlı+çökmüş → rug/dead, taze/likit → active; max_drawdown_pct gerçek.
4. `/api/creator/{address}` geçmişinde outcome/peakMarketCap/maxDrawdownPct/liquidityStatus **gerçek**; creatorSellPct hâlâ 0.
5. Frontend Creator geçmiş tablosu gerçek outcome rozetleri (UI regresyonsuz — enum'lar geçerli).
6. Eşikler gerçek dağılıma göre `OUTCOME_*` ile kalibre edilebilir (kod değişmez).
