# Manipulation Risk (Slice 2c) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `manipulationRisk` skorunu (agrega-proxy, saf-DB, kural-tabanlı Go) hesaplayıp Token Detay ekranına + işlem-akışı metriklerine (`uniqueBuyers`/`buyRatio`/`sellRatio`/`creatorHoldingPct`) gerçek değer olarak bağlamak.

**Architecture:** 2a/2b'deki **Option A** deseni: arka plan worker DB'den okur → skorlar → DB'ye yazar (canlı RPC/HTTP yok, throttle-dayanıklı). Veri kaynağı mevcut çağrıların yan-ürünü: GeckoTerminal `transactions.h24` (Enricher'ın aynı `pools/multi` yanıtı) + creator payı (Safety worker'ın holder-fetch'i). Frontend seam'i (`store.TokenMetrics`+`ScoreDetail`) zaten tüm alanları taşıyor → **frontend sıfır dokunuş**.

**Tech Stack:** Go 1.24, chi, pgx/v5 (database/sql), goose migration, slog. Test: standart `testing` + fake store parity.

**Spec:** `docs/superpowers/specs/2026-08-14-sentinel-backend-scoring-2c-manipulation-design.md`

## Global Constraints

- **go.mod `go 1.24`** — CI (`api-go.yml`) ile eşleşir; yükseltme YOK.
- **SOLID + clean code** — saf Scorer (SRP), dar DIP arayüzleri (ISP), config-gate; ölçüt spec §4.
- **Dürüst nötr** — sahte skor yok: `txns < MIN_TXNS` → `value=0/conf=0/breakdown=[]`; skorlanmamış → nötr. JSON dizileri asla `nil` değil (`[]ScoreBreakdownItem{}`).
- **Frontend'e SIFIR dokunuş** — `apps/web/` değişmez; mevcut testler (`token.test.ts` vb.) regresyonsuz.
- **Migration append-only** — `0010_add_token_manipulation.sql`; mevcut migration'lar değişmez.
- **Ağırlıklar deploy-tunable** — `MANIPULATION_*` env; kod sabit değer gömme yok.
- **Her task sonunda** `go build ./... && go vet ./... && go test -race ./...` yeşil + commit.
- **Çalışma dizini:** `apps/api-go/` (komutlar buradan çalışır).

---

### Task 1: Store temeli — migration 0010 + manipulation tipleri + ManipulationTargets/UpdateManipulation

Manipülasyon skorunun okuma/yazma DB yüzeyi. Migration TÜM 9 kolonu ekler (txns×4 + creator_holding_pct + manipulation×4); bu task yalnız manipulation-özel okuma/yazmayı bağlar (txns yazımı Task 4, creator_holding yazımı Task 5).

**Files:**
- Create: `apps/api-go/internal/store/migrations/0010_add_token_manipulation.sql`
- Modify: `apps/api-go/internal/store/tokens.go` (tipler + `TokenStore` arayüzü + postgres metotları)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake tokenEntry alanları + fake metotlar)
- Test: `apps/api-go/internal/store/tokens_fake_test.go`

**Interfaces:**
- Consumes: mevcut `postgresStore`, `fakeTokenStore`, `ScoreBreakdownItem`, `parseBreakdownJSON`.
- Produces:
  - `store.ManipulationTarget{Mint string; Buys, Sells, Buyers int; CreatorHoldingPct, Vol24h, Liquidity float64}`
  - `store.ManipulationUpdate{Mint string; Score, Confidence float64; Breakdown []ScoreBreakdownItem; ScoredTs int64}`
  - `TokenStore.ManipulationTargets(ctx, limit int) ([]ManipulationTarget, error)`
  - `TokenStore.UpdateManipulation(ctx, u ManipulationUpdate) error`
  - fake tokenEntry yeni alanları: `txnsBuys, txnsSells, txnsBuyers, txnsSellers int; creatorHoldingPct, manipScore, manipConf float64; manipBreakdown []ScoreBreakdownItem; manipScoredTs int64`

- [ ] **Step 1: Migration dosyasını yaz**

`apps/api-go/internal/store/migrations/0010_add_token_manipulation.sql`:
```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_buys               INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_sells              INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_buyers             INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_sellers            INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator_holding_pct     DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_score      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_breakdown  TEXT             NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_scored_ts  BIGINT           NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_buys;
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_sells;
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_buyers;
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_sellers;
ALTER TABLE tokens DROP COLUMN IF EXISTS creator_holding_pct;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_score;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_confidence;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_breakdown;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_scored_ts;
```

- [ ] **Step 2: Store tiplerini + arayüz metotlarını ekle**

`tokens.go`'da (diğer target/update tiplerinin yanına, ör. `SafetyTarget` sonrası):
```go
// ManipulationTarget, 2c manipülasyon skoru için gereken işlem-akışı girdileridir (h24).
type ManipulationTarget struct {
	Mint                                 string
	Buys, Sells, Buyers                  int
	CreatorHoldingPct, Vol24h, Liquidity float64
}

// ManipulationUpdate, 2c scorer'ının yazdığı manipülasyon sonucudur.
type ManipulationUpdate struct {
	Mint       string
	Score      float64
	Confidence float64
	Breakdown  []ScoreBreakdownItem
	ScoredTs   int64
}
```
`TokenStore` arayüzüne (satır 113 `UpsertReputation` sonrası, `}` öncesi):
```go
	// 2c: manipülasyon riski agrega girdilerini döndürür / hesaplanmış skoru persist eder.
	ManipulationTargets(ctx context.Context, limit int) ([]ManipulationTarget, error)
	UpdateManipulation(ctx context.Context, u ManipulationUpdate) error
```

- [ ] **Step 3: Failing test yaz (fake round-trip + round-robin)**

`tokens_fake_test.go`'ya ekle:
```go
func TestFakeUpdateManipulationRoundTrip(t *testing.T) {
	f := NewFakeTokenStore()
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "m1", PoolAddr: "p1"})
	bd := []ScoreBreakdownItem{{Label: "Alım/satım dengesizliği", Weight: 30, Detail: "buyRatio=0.90"}}
	if err := f.UpdateManipulation(context.Background(), ManipulationUpdate{
		Mint: "m1", Score: 42, Confidence: 0.5, Breakdown: bd, ScoredTs: 1234,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	targets, err := f.ManipulationTargets(context.Background(), 10)
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 1 || targets[0].Mint != "m1" {
		t.Fatalf("beklenen 1 hedef m1, gelen %+v", targets)
	}
}

func TestFakeManipulationTargetsPoolOnlyOldestFirst(t *testing.T) {
	f := NewFakeTokenStore()
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "np", PoolAddr: ""}) // pool'suz → elenir
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "a", PoolAddr: "pa"})
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "b", PoolAddr: "pb"})
	f.UpdateManipulation(context.Background(), ManipulationUpdate{Mint: "a", ScoredTs: 500}) // a skorlandı
	targets, _ := f.ManipulationTargets(context.Background(), 10)
	if len(targets) != 2 {
		t.Fatalf("pool'lu 2 token bekleniyordu, gelen %d", len(targets))
	}
	if targets[0].Mint != "b" { // skorlanmamış (ts=0) önce
		t.Fatalf("skorlanmamış b önce beklenir, gelen %s", targets[0].Mint)
	}
}
```

- [ ] **Step 4: Testin fail ettiğini doğrula**

Run: `go test ./internal/store/ -run TestFakeUpdateManipulation -v`
Expected: FAIL (derleme hatası: `UpdateManipulation`/`ManipulationTargets` tanımsız).

- [ ] **Step 5: Fake store'u implemente et**

`fake_ingest.go`'da tokenEntry struct'ına alanları ekle (mevcut safety alanlarının yanına):
```go
	txnsBuys, txnsSells, txnsBuyers, txnsSellers int
	creatorHoldingPct                            float64
	manipScore, manipConf                        float64
	manipBreakdown                               []ScoreBreakdownItem
	manipScoredTs                                int64
```
Fake metotlar (SafetyScoreTargets desenini izle — `sort` zaten import'lu):
```go
func (f *fakeTokenStore) UpdateManipulation(_ context.Context, u ManipulationUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[u.Mint]
	if !ok {
		return nil
	}
	cur.manipScore, cur.manipConf = u.Score, u.Confidence
	cur.manipBreakdown, cur.manipScoredTs = u.Breakdown, u.ScoredTs
	f.byID[u.Mint] = cur
	return nil
}

func (f *fakeTokenStore) ManipulationTargets(_ context.Context, limit int) ([]ManipulationTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].manipScoredTs < f.byID[ids[j]].manipScoredTs
	})
	out := make([]ManipulationTarget, 0, limit)
	for _, id := range ids {
		t := f.byID[id]
		if t.poolAddr == "" || len(out) >= limit {
			continue
		}
		out = append(out, ManipulationTarget{
			Mint: t.row.Mint, Buys: t.txnsBuys, Sells: t.txnsSells, Buyers: t.txnsBuyers,
			CreatorHoldingPct: t.creatorHoldingPct, Vol24h: t.vol24h, Liquidity: t.row.Liquidity,
		})
	}
	return out, nil
}
```

- [ ] **Step 6: Postgres metotlarını implemente et**

`tokens.go`'da (`SafetyScoreTargets` deseninde):
```go
func (p *postgresStore) ManipulationTargets(ctx context.Context, limit int) ([]ManipulationTarget, error) {
	const q = `SELECT mint, txns_buys, txns_sells, txns_buyers, creator_holding_pct, vol24h, liquidity
		FROM tokens WHERE pool_address <> ''
		ORDER BY manipulation_scored_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ManipulationTarget, 0, limit)
	for rows.Next() {
		var t ManipulationTarget
		if err := rows.Scan(&t.Mint, &t.Buys, &t.Sells, &t.Buyers,
			&t.CreatorHoldingPct, &t.Vol24h, &t.Liquidity); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) UpdateManipulation(ctx context.Context, u ManipulationUpdate) error {
	bdJSON, err := json.Marshal(u.Breakdown)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET manipulation_score=$2, manipulation_confidence=$3,
		manipulation_breakdown=$4, manipulation_scored_ts=$5 WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, u.Mint, u.Score, u.Confidence, string(bdJSON), u.ScoredTs)
	return err
}
```
> Not: `vol24h` kolon adı 0003'te `vol24h` (bkz `OutcomeTargets`), `vol5m` ayrı. Doğrula.

- [ ] **Step 7: Test + vet + build yeşil**

Run: `go test ./internal/store/ -run TestFakeManipulation -v && go build ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/internal/store/
git commit -m "feat(2c): migration 0010 + manipulation store tipleri + Targets/Update (fake+postgres)"
```

---

### Task 2: Saf Scorer (`internal/manipulation/scorer.go`)

Ağsız/DB'siz saf hesaplama (SRP). Config eşikleri enjekte.

**Files:**
- Create: `apps/api-go/internal/manipulation/scorer.go`
- Test: `apps/api-go/internal/manipulation/scorer_test.go`

**Interfaces:**
- Consumes: `store.ScoreBreakdownItem`.
- Produces:
  - `manipulation.Thresholds{MinTxns, ConfTxns int; WImbalance, WWash, WVolume, WCreator, WashMin, WashMax, VolMin, VolMax float64}`
  - `manipulation.Inputs{Buys, Sells, Buyers int; CreatorHoldingPct, Vol24h, Liquidity float64}`
  - `manipulation.Result{Value, Confidence float64; Breakdown []store.ScoreBreakdownItem}`
  - `func Score(in Inputs, th Thresholds) Result`

- [ ] **Step 1: Failing test yaz**

`scorer_test.go`:
```go
package manipulation

import "testing"

func defTh() Thresholds {
	return Thresholds{MinTxns: 20, ConfTxns: 100, WImbalance: 30, WWash: 35, WVolume: 25, WCreator: 10,
		WashMin: 3, WashMax: 15, VolMin: 3, VolMax: 20}
}

func TestScoreNeutralBelowMinTxns(t *testing.T) {
	r := Score(Inputs{Buys: 5, Sells: 5, Buyers: 5}, defTh()) // txns=10 < 20
	if r.Value != 0 || r.Confidence != 0 || len(r.Breakdown) != 0 {
		t.Fatalf("nötr beklenir, gelen %+v", r)
	}
}

func TestScoreBalancedLowManipulation(t *testing.T) {
	// dengeli akış, çok alıcı, düşük vol/likidite, creator payı yok → düşük skor
	r := Score(Inputs{Buys: 50, Sells: 50, Buyers: 90, Vol24h: 1000, Liquidity: 100000}, defTh())
	if r.Value > 5 {
		t.Fatalf("dengeli akış düşük skor beklenir, gelen %.1f", r.Value)
	}
	if r.Confidence != 1 { // txns=100 == ConfTxns
		t.Fatalf("conf 1.0 beklenir, gelen %.2f", r.Confidence)
	}
}

func TestScoreAllBuyMaxImbalance(t *testing.T) {
	r := Score(Inputs{Buys: 100, Sells: 0, Buyers: 100, Vol24h: 0, Liquidity: 100000}, defTh())
	// imbalanceNorm=1 → W_imbalance=30 katkı; diğerleri ~0
	if r.Value < 29 || r.Value > 31 {
		t.Fatalf("hep-alım ~30 beklenir, gelen %.1f", r.Value)
	}
	if len(r.Breakdown) == 0 || r.Breakdown[0].Label != "Alım/satım dengesizliği" {
		t.Fatalf("dengesizlik breakdown beklenir, gelen %+v", r.Breakdown)
	}
}

func TestScoreWashProxyMaxed(t *testing.T) {
	// 3 alıcı 60 alım → perBuyer=20 > WashMax=15 → washNorm=1 → +35
	r := Score(Inputs{Buys: 60, Sells: 0, Buyers: 3, Liquidity: 100000}, defTh())
	// imbalance=1 (+30) + wash=1 (+35) = 65
	if r.Value < 64 || r.Value > 66 {
		t.Fatalf("hep-alım+wash ~65 beklenir, gelen %.1f", r.Value)
	}
}

func TestScoreClampedTo100(t *testing.T) {
	r := Score(Inputs{Buys: 100, Sells: 0, Buyers: 1, Vol24h: 1e9, Liquidity: 1, CreatorHoldingPct: 100}, defTh())
	if r.Value != 100 {
		t.Fatalf("clamp 100 beklenir, gelen %.1f", r.Value)
	}
}

func TestScoreConfidenceRamp(t *testing.T) {
	r := Score(Inputs{Buys: 25, Sells: 25, Buyers: 40}, defTh()) // txns=50 → 0.5
	if r.Confidence < 0.49 || r.Confidence > 0.51 {
		t.Fatalf("conf 0.5 beklenir, gelen %.2f", r.Confidence)
	}
}
```

- [ ] **Step 2: Testin fail ettiğini doğrula**

Run: `go test ./internal/manipulation/ -v`
Expected: FAIL (paket/`Score` yok).

- [ ] **Step 3: Scorer'ı implemente et**

`scorer.go`:
```go
// Package manipulation, işlem-akışı agrega proxy'lerinden manipülasyon riski (0-100, inverted:
// yüksek=daha çok manipülasyon) hesaplar. Saf, ağsız/DB'siz (SRP). Skor bir PROXY'dir; gerçek
// per-trade wash/sniper tespiti 2e trade-flow ile yükseltilir.
package manipulation

import (
	"fmt"
	"math"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Thresholds, ağırlıklar + normalleştirme bantlarıdır (config'ten enjekte; deploy-tunable).
type Thresholds struct {
	MinTxns, ConfTxns                    int
	WImbalance, WWash, WVolume, WCreator float64
	WashMin, WashMax, VolMin, VolMax     float64
}

// Inputs, bir token'ın h24 işlem-akışı girdileridir.
type Inputs struct {
	Buys, Sells, Buyers                  int
	CreatorHoldingPct, Vol24h, Liquidity float64
}

// Result, hesaplanmış manipülasyon skorudur (breakdown yalnız katkısı>0 bileşenleri içerir).
type Result struct {
	Value, Confidence float64
	Breakdown         []store.ScoreBreakdownItem
}

func Score(in Inputs, th Thresholds) Result {
	txns := in.Buys + in.Sells
	if txns < th.MinTxns {
		return Result{Value: 0, Confidence: 0, Breakdown: []store.ScoreBreakdownItem{}}
	}

	buyRatio := float64(in.Buys) / float64(txns)
	imbalanceNorm := clamp01(math.Abs(buyRatio-0.5) * 2)

	buyers := in.Buyers
	if buyers < 1 {
		buyers = 1
	}
	perBuyer := float64(in.Buys) / float64(buyers)
	washNorm := normBand(perBuyer, th.WashMin, th.WashMax)

	liq := in.Liquidity
	if liq < 1 {
		liq = 1
	}
	volLiq := in.Vol24h / liq
	volNorm := normBand(volLiq, th.VolMin, th.VolMax)

	creatorNorm := clamp01(in.CreatorHoldingPct / 100)

	cImb := th.WImbalance * imbalanceNorm
	cWash := th.WWash * washNorm
	cVol := th.WVolume * volNorm
	cCreator := th.WCreator * creatorNorm

	value := cImb + cWash + cVol + cCreator
	if value > 100 {
		value = 100
	}

	bd := []store.ScoreBreakdownItem{}
	if cImb > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Alım/satım dengesizliği", Weight: round1(cImb), Detail: fmt.Sprintf("buyRatio=%.2f", buyRatio)})
	}
	if cWash > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Wash-proxy (işlem/alıcı)", Weight: round1(cWash), Detail: fmt.Sprintf("%.1f işlem/alıcı", perBuyer)})
	}
	if cVol > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Hacim/likidite anomalisi", Weight: round1(cVol), Detail: fmt.Sprintf("%.1fx", volLiq)})
	}
	if cCreator > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Creator payı", Weight: round1(cCreator), Detail: fmt.Sprintf("%%%.0f", in.CreatorHoldingPct)})
	}

	conf := float64(txns) / float64(th.ConfTxns)
	if conf > 1 {
		conf = 1
	}
	return Result{Value: value, Confidence: conf, Breakdown: bd}
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// normBand, x'i [min,max] bandına göre [0,1]'e taşır (min altı 0, max üstü 1).
func normBand(x, min, max float64) float64 {
	if max <= min {
		return 0
	}
	return clamp01((x - min) / (max - min))
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }
```

- [ ] **Step 4: Test yeşil**

Run: `go test ./internal/manipulation/ -v`
Expected: PASS (tüm senaryolar).

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/manipulation/scorer.go apps/api-go/internal/manipulation/scorer_test.go
git commit -m "feat(2c): saf manipulation Scorer (inverted proxy, config eşikleri)"
```

---

### Task 3: Worker (`internal/manipulation/worker.go`)

Arka plan döngü — saf DB, RPC yok (reputation Worker deseni birebir).

**Files:**
- Create: `apps/api-go/internal/manipulation/worker.go`
- Test: `apps/api-go/internal/manipulation/worker_test.go`

**Interfaces:**
- Consumes: `store.ManipulationTarget`, `store.ManipulationUpdate`, `Score`, `Thresholds`.
- Produces:
  - `manipulation.ManipulationStore` arayüzü: `ManipulationTargets(ctx, limit int) ([]store.ManipulationTarget, error)` + `UpdateManipulation(ctx, store.ManipulationUpdate) error` (`store.TokenStore` karşılar).
  - `manipulation.WorkerDeps{Store ManipulationStore; Thresholds Thresholds; Interval time.Duration; Limit int; Now func() int64; Logger *slog.Logger}`
  - `manipulation.NewWorker(WorkerDeps) *Worker` + `(*Worker).Run(ctx)`

- [ ] **Step 1: Failing test yaz (kısmi-hata izolasyon + ctx-cancel)**

`worker_test.go`:
```go
package manipulation

import (
	"context"
	"errors"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type stubStore struct {
	targets []store.ManipulationTarget
	updated []store.ManipulationUpdate
	failOn  string // bu mint'te UpdateManipulation hata döner
}

func (s *stubStore) ManipulationTargets(_ context.Context, _ int) ([]store.ManipulationTarget, error) {
	return s.targets, nil
}
func (s *stubStore) UpdateManipulation(_ context.Context, u store.ManipulationUpdate) error {
	if u.Mint == s.failOn {
		return errors.New("boom")
	}
	s.updated = append(s.updated, u)
	return nil
}

func TestWorkerScoreOncePartialErrorIsolated(t *testing.T) {
	st := &stubStore{failOn: "bad", targets: []store.ManipulationTarget{
		{Mint: "bad", Buys: 100, Sells: 0, Buyers: 1, Liquidity: 1000},
		{Mint: "good", Buys: 100, Sells: 0, Buyers: 1, Liquidity: 1000},
	}}
	w := NewWorker(WorkerDeps{Store: st, Thresholds: defTh(), Now: func() int64 { return 7 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatalf("scoreOnce err: %v", err)
	}
	if len(st.updated) != 1 || st.updated[0].Mint != "good" {
		t.Fatalf("yalnız good yazılmalı (bad izole), gelen %+v", st.updated)
	}
	if st.updated[0].ScoredTs != 7 {
		t.Fatalf("ScoredTs Now()=7 beklenir, gelen %d", st.updated[0].ScoredTs)
	}
}

func TestWorkerScoreOnceCtxCancel(t *testing.T) {
	st := &stubStore{targets: []store.ManipulationTarget{{Mint: "a", Buys: 50, Sells: 50, Buyers: 5}}}
	w := NewWorker(WorkerDeps{Store: st, Thresholds: defTh()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.scoreOnce(ctx); err == nil {
		t.Fatalf("iptal edilmiş ctx'te hata beklenir")
	}
}
```

- [ ] **Step 2: Testin fail ettiğini doğrula**

Run: `go test ./internal/manipulation/ -run TestWorker -v`
Expected: FAIL (`NewWorker`/`WorkerDeps` yok).

- [ ] **Step 3: Worker'ı implemente et (reputation/worker.go deseni)**

`worker.go`:
```go
package manipulation

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// ManipulationStore, Worker'ın kalıcılık + agrega bağımlılığıdır (DIP; store.TokenStore karşılar).
type ManipulationStore interface {
	ManipulationTargets(ctx context.Context, limit int) ([]store.ManipulationTarget, error)
	UpdateManipulation(ctx context.Context, u store.ManipulationUpdate) error
}

type WorkerDeps struct {
	Store      ManipulationStore
	Thresholds Thresholds
	Interval   time.Duration
	Limit      int
	Now        func() int64
	Logger     *slog.Logger
}

// Worker, periyodik olarak token'ları çekip manipülasyon skorunu hesaplayıp yazar (RPC YOK, saf DB).
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
	if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("manipulation", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("manipulation", "err", err)
			}
		}
	}
}

// scoreOnce, bir döngü: hedefleri çek → her birini skorla → persist (kısmi hata izole).
func (w *Worker) scoreOnce(ctx context.Context) error {
	targets, err := w.d.Store.ManipulationTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r := Score(Inputs{
			Buys: tg.Buys, Sells: tg.Sells, Buyers: tg.Buyers,
			CreatorHoldingPct: tg.CreatorHoldingPct, Vol24h: tg.Vol24h, Liquidity: tg.Liquidity,
		}, w.d.Thresholds)
		if err := w.d.Store.UpdateManipulation(ctx, store.ManipulationUpdate{
			Mint: tg.Mint, Score: r.Value, Confidence: r.Confidence, Breakdown: r.Breakdown, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("update manipulation", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Test yeşil**

Run: `go test ./internal/manipulation/ -race -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/manipulation/worker.go apps/api-go/internal/manipulation/worker_test.go
git commit -m "feat(2c): manipulation Worker (saf DB, kısmi-hata izole, ctx-cancel)"
```

---

### Task 4: Enricher — GeckoTerminal `transactions.h24` parse + persist

`transactions` alanı zaten API yanıtında; parse edilmiyor. Aynı `pools/multi` çağrısı → yeni yük yok.

**Files:**
- Modify: `apps/api-go/internal/market/provider.go` (`Pool` alanları)
- Modify: `apps/api-go/internal/market/geckoterminal.go` (`gtPool` parse + `toPools`)
- Modify: `apps/api-go/internal/market/enricher.go` (`UpdateMarket` argümanları)
- Modify: `apps/api-go/internal/store/tokens.go` (`MarketUpdate` alanları + `UpdateMarket` SQL)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (`UpdateMarket` fake — txns alanlarını set eder)
- Test: `apps/api-go/internal/market/geckoterminal_test.go`, `apps/api-go/internal/store/tokens_fake_test.go`

**Interfaces:**
- Consumes: mevcut `Pool`, `MarketUpdate`, `toPools`, `UpdateMarket`.
- Produces:
  - `market.Pool` +`TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int`
  - `store.MarketUpdate` +`TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int`
  - fake tokenEntry `txnsBuys/txnsSells/txnsBuyers/txnsSellers` alanları (Task 1'de eklendi) `UpdateMarket`'ta doldurulur.

- [ ] **Step 1: Failing test yaz (GT transactions parse)**

`geckoterminal_test.go`'ya ekle (mevcut fixture-parse desenini izle):
```go
func TestToPoolsParsesTransactionsH24(t *testing.T) {
	body := `{"data":[{"attributes":{
		"address":"pool1","name":"AAA / SOL","base_token_price_usd":"1","reserve_in_usd":"1000",
		"transactions":{"h24":{"buys":80,"sells":20,"buyers":30,"sellers":10}}},
		"relationships":{"base_token":{"data":{"id":"solana_mintA"}},"dex":{"data":{"id":"pumpfun"}}}}]}`
	var resp gtResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pools := resp.toPools(false)
	if len(pools) != 1 {
		t.Fatalf("1 havuz bekleniyordu, gelen %d", len(pools))
	}
	p := pools[0]
	if p.TxnsBuys != 80 || p.TxnsSells != 20 || p.TxnsBuyers != 30 || p.TxnsSellers != 10 {
		t.Fatalf("h24 txns yanlış: %+v", p)
	}
}

func TestToPoolsMissingTransactionsZero(t *testing.T) {
	body := `{"data":[{"attributes":{"address":"p","name":"B / SOL","base_token_price_usd":"1"},
		"relationships":{"base_token":{"data":{"id":"solana_m"}},"dex":{"data":{"id":"pumpfun"}}}}]}`
	var resp gtResponse
	json.Unmarshal([]byte(body), &resp)
	pools := resp.toPools(false)
	if len(pools) != 1 || pools[0].TxnsBuys != 0 || pools[0].TxnsBuyers != 0 {
		t.Fatalf("eksik transactions → 0 beklenir, gelen %+v", pools)
	}
}
```
> `encoding/json` test dosyasında import edilmeli (yoksa ekle).

- [ ] **Step 2: Testin fail ettiğini doğrula**

Run: `go test ./internal/market/ -run TestToPools -v`
Expected: FAIL (`TxnsBuys` alanı yok).

- [ ] **Step 3: Pool + gtPool + toPools'u genişlet**

`provider.go` `Pool` struct'ına ekle (Vol24h sonrası):
```go
	TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int // h24 işlem sayıları (2c manipülasyon)
```
`geckoterminal.go` `gtPool.Attributes`'a ekle (`FDV` sonrası):
```go
		Transactions map[string]struct {
			Buys    int `json:"buys"`
			Sells   int `json:"sells"`
			Buyers  int `json:"buyers"`
			Sellers int `json:"sellers"`
		} `json:"transactions"`
```
`toPools`'ta `Pool` kurarken (`Vol24h` satırından sonra, aynı literal içinde), h24'ü doldur — struct literal sonrası ekleme daha temiz:
```go
		if tx, ok := d.Attributes.Transactions["h24"]; ok {
			p.TxnsBuys, p.TxnsSells = tx.Buys, tx.Sells
			p.TxnsBuyers, p.TxnsSellers = tx.Buyers, tx.Sellers
		}
```
(bu blok `out = append(out, p)` ÖNCESİNE, `p` değişkeni set edildikten sonra yerleştirilir.)

- [ ] **Step 4: MarketUpdate + UpdateMarket (fake+postgres) genişlet**

`store/tokens.go` `MarketUpdate`'e ekle (`Vol24h` sonrası):
```go
	TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int
```
postgres `UpdateMarket` SQL'ine kolonları ekle (`vol24h=$9` sonrası, peak satırlarından önce; parametre numaralarını kaydır):
```go
	const q = `UPDATE tokens SET price=$2, liquidity=$3, vol5m=$4, momentum=$5, spark=$6,
		price_change_h24=$7, market_cap_usd=$8, vol24h=$9,
		txns_buys=$10, txns_sells=$11, txns_buyers=$12, txns_sellers=$13,
		peak_market_cap = GREATEST(peak_market_cap, $8),
		peak_liquidity  = GREATEST(peak_liquidity, $3)
		WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, m.Mint, m.Price, m.Liquidity, m.Vol5m, m.Momentum, string(sparkJSON),
		m.PriceChangeH24, m.MarketCapUSD, m.Vol24h, m.TxnsBuys, m.TxnsSells, m.TxnsBuyers, m.TxnsSellers)
```
fake `UpdateMarket`'a ekle (`cur.vol24h = ...` satırından sonra):
```go
	cur.txnsBuys, cur.txnsSells = m.TxnsBuys, m.TxnsSells
	cur.txnsBuyers, cur.txnsSellers = m.TxnsBuyers, m.TxnsSellers
```

- [ ] **Step 5: Enricher tick'te txns geçir**

`enricher.go` `UpdateMarket` çağrısına ekle (`Vol24h: p.Vol24h` sonrası, aynı literal):
```go
					TxnsBuys: p.TxnsBuys, TxnsSells: p.TxnsSells, TxnsBuyers: p.TxnsBuyers, TxnsSellers: p.TxnsSellers,
```

- [ ] **Step 6: Fake round-trip testi (UpdateMarket → ManipulationTargets txns görür)**

`tokens_fake_test.go`'ya ekle:
```go
func TestFakeUpdateMarketPersistsTxns(t *testing.T) {
	f := NewFakeTokenStore()
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "m", PoolAddr: "p"})
	f.UpdateMarket(context.Background(), MarketUpdate{Mint: "m", Liquidity: 5000, Vol24h: 20000,
		TxnsBuys: 80, TxnsSells: 20, TxnsBuyers: 30, TxnsSellers: 10})
	tg, _ := f.ManipulationTargets(context.Background(), 10)
	if len(tg) != 1 || tg[0].Buys != 80 || tg[0].Sells != 20 || tg[0].Buyers != 30 ||
		tg[0].Vol24h != 20000 || tg[0].Liquidity != 5000 {
		t.Fatalf("txns/market ManipulationTargets'a taşınmalı, gelen %+v", tg)
	}
}
```

- [ ] **Step 7: Test + build + vet yeşil**

Run: `go test ./internal/market/ ./internal/store/ -race && go build ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/internal/market/ apps/api-go/internal/store/
git commit -m "feat(2c): Enricher GeckoTerminal transactions.h24 parse+persist (sıfır ek çağrı)"
```

---

### Task 5: Safety worker — `creator_holding_pct` (sıfır ek RPC)

Safety worker'ın zaten yaptığı holder-fetch'inden creator payını türet. Safety Scorer DEĞİŞMEZ (creator payı safety girdisi değil).

**Files:**
- Modify: `apps/api-go/internal/store/tokens.go` (`SafetyTarget.Creator`, `SafetyScoreTargets` SELECT, `SafetyUpdate` alanları, `UpdateSafety` koşullu SQL)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake `SafetyScoreTargets` creator, fake `UpdateSafety` koşullu creator_holding)
- Modify: `apps/api-go/internal/ingest/holders.go` (`HolderDistribution` imza + creator payı)
- Modify: `apps/api-go/internal/safety/provider.go` (`Holders` arayüzü, `OnChainData`, `FetchOnChain` imza)
- Modify: `apps/api-go/internal/safety/worker.go` (creator geçir + persist)
- Test: `apps/api-go/internal/ingest/holders_test.go`, `apps/api-go/internal/store/tokens_fake_test.go`, `apps/api-go/internal/safety/*_test.go` (imza güncellemeleri)

**Interfaces:**
- Consumes: mevcut `HolderDistribution`, `FetchOnChain`, `OnChainData`, `SafetyUpdate`, `SafetyTarget`.
- Produces (imza değişiklikleri — TÜM çağıran/testler güncellenir):
  - `store.SafetyTarget` +`Creator string`
  - `store.SafetyUpdate` +`CreatorHoldingPct float64; CreatorHoldingKnown bool`
  - `safety.Holders.HolderDistribution(ctx, mint, creator string, capN int) (count int, top10Pct, creatorPct float64, capped bool, err error)`
  - `safety.OnChainData` +`CreatorHoldingPct float64; CreatorHoldingKnown bool`
  - `safety.DataProvider.FetchOnChain(ctx, mint, creator string) (OnChainData, error)`

- [ ] **Step 1: Failing test yaz (HolderDistribution creator payı)**

`ingest/holders.go`'nun mevcut testine (`holders_test.go`) creator-pay senaryosu ekle. Mevcut testlerdeki mock RPC desenini izle; yeni imza `HolderDistribution(ctx, mint, creator, cap)`:
```go
func TestHolderDistributionCreatorPct(t *testing.T) {
	// creator "cA" toplam arzın %40'ını tutuyor (owner map: cA=400, o2=300, o3=300)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":{"token_accounts":[
			{"owner":"cA","amount":400},{"owner":"o2","amount":300},{"owner":"o3","amount":300}]}}`))
	}))
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	_, _, creatorPct, _, err := h.HolderDistribution(context.Background(), "mint", "cA", 5000)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if creatorPct < 39.9 || creatorPct > 40.1 {
		t.Fatalf("creator payı %%40 beklenir, gelen %.2f", creatorPct)
	}
}

func TestHolderDistributionCreatorAbsentZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":{"token_accounts":[{"owner":"o2","amount":1000}]}}`))
	}))
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	_, _, creatorPct, _, _ := h.HolderDistribution(context.Background(), "mint", "cA", 5000)
	if creatorPct != 0 {
		t.Fatalf("creator listede yok → %%0 beklenir, gelen %.2f", creatorPct)
	}
}
```
> Mevcut `holders_test.go`'da `HolderDistribution` çağıran testler varsa yeni imzaya (creator="" ekleyerek) güncelle.

- [ ] **Step 2: Testin fail ettiğini doğrula**

Run: `go test ./internal/ingest/ -run TestHolderDistribution -v`
Expected: FAIL (imza uyuşmazlığı — `creatorPct` dönmüyor).

- [ ] **Step 3: HolderDistribution imzasını + creator payını implemente et**

`holders.go` `HolderDistribution` imzasını değiştir ve creator payını hesapla:
```go
func (h *HeliusHolders) HolderDistribution(ctx context.Context, mint, creator string, capN int) (int, float64, float64, bool, error) {
	if capN <= 0 {
		capN = 5000
	}
	byOwner := map[string]float64{}
	// ... (mevcut sayfalama döngüsü aynı; return'ler 5 değere çıkar) ...
	// döngü hatalarında: return 0, 0, 0, false, err
	// ... top10 hesabı aynı ...
	pct := 0.0
	if total > 0 {
		pct = top10 / total * 100
	}
	creatorPct := 0.0
	if creator != "" && total > 0 {
		creatorPct = byOwner[creator] / total * 100
	}
	return len(byOwner), pct, creatorPct, capped, nil
}
```

- [ ] **Step 4: safety.Holders + OnChainData + FetchOnChain'i güncelle**

`safety/provider.go`:
```go
type Holders interface {
	HolderDistribution(ctx context.Context, mint, creator string, cap int) (count int, top10Pct, creatorPct float64, capped bool, err error)
}
```
`OnChainData`'ya ekle:
```go
	CreatorHoldingPct   float64
	CreatorHoldingKnown bool
```
`DataProvider` + `FetchOnChain` imzası:
```go
type DataProvider interface {
	FetchOnChain(ctx context.Context, mint, creator string) (OnChainData, error)
}

func (p *HeliusProvider) FetchOnChain(ctx context.Context, mint, creator string) (OnChainData, error) {
	var d OnChainData
	if mintA, freezeA, err := p.auth.MintAuthorities(ctx, mint); err == nil {
		d.MintAuthorityActive, d.FreezeAuthorityActive, d.AuthoritiesKnown = mintA, freezeA, true
	}
	if count, top10, creatorPct, capped, err := p.holders.HolderDistribution(ctx, mint, creator, p.holdersCap); err == nil {
		d.HolderCount, d.Top10Pct, d.HoldersKnown, d.HoldersCapped = count, top10, true, capped
		d.CreatorHoldingPct, d.CreatorHoldingKnown = creatorPct, true
	}
	return d, nil
}
```

- [ ] **Step 5: SafetyTarget.Creator + SafetyScoreTargets SELECT güncelle**

`store/tokens.go` `SafetyTarget`'a `Creator string` ekle; `SafetyScoreTargets` SQL'i:
```go
	const q = `SELECT mint, liquidity, launchpad, creator FROM tokens
		WHERE pool_address <> '' ORDER BY safety_scored_ts ASC, first_seen_ts DESC LIMIT $1`
	// Scan: &t.Mint, &t.Liquidity, &t.Launchpad, &t.Creator
```
fake `SafetyScoreTargets` `SafetyTarget{...}` literaline `Creator: t.creator` ekle.

- [ ] **Step 6: SafetyUpdate + UpdateSafety koşullu creator_holding**

`store/tokens.go` `SafetyUpdate`'e ekle:
```go
	CreatorHoldingPct   float64
	CreatorHoldingKnown bool
```
postgres `UpdateSafety` SQL'i (koşullu — known=false ise EZMEZ):
```go
	const q = `UPDATE tokens SET safety_score=$2, safety_confidence=$3, top10_holder_pct=$4,
		safety_breakdown=$5, safety_risks=$6, safety_scored_ts=$7,
		creator_holding_pct = CASE WHEN $8 THEN $9 ELSE creator_holding_pct END
		WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, s.Mint, s.Score, s.Confidence, s.Top10Pct,
		string(bdJSON), string(rkJSON), s.ScoredTs, s.CreatorHoldingKnown, s.CreatorHoldingPct)
```
fake `UpdateSafety`'ye ekle (mevcut set'lerden sonra):
```go
	if s.CreatorHoldingKnown {
		cur.creatorHoldingPct = s.CreatorHoldingPct
	}
```

- [ ] **Step 7: Safety worker'da creator geçir + persist**

`safety/worker.go` `scoreOnce`: `FetchOnChain` çağrısına `tg.Creator` ekle; `UpdateSafety` literaline creator_holding ekle:
```go
		data, err := w.d.Provider.FetchOnChain(ctx, tg.Mint, tg.Creator)
		// ... Score(...) aynı — creator holding Scorer'a GİRMEZ ...
		if err := w.d.Store.UpdateSafety(ctx, store.SafetyUpdate{
			Mint: tg.Mint, Score: res.Score, Confidence: res.Confidence, Top10Pct: res.Top10Pct,
			Breakdown: res.Breakdown, Risks: res.Risks, ScoredTs: now,
			CreatorHoldingPct: data.CreatorHoldingPct, CreatorHoldingKnown: data.CreatorHoldingKnown,
		}); err != nil { ... }
```

- [ ] **Step 8: Fake koşullu-yazma testi + mevcut safety testleri güncelle**

`tokens_fake_test.go`:
```go
func TestFakeUpdateSafetyCreatorHoldingConditional(t *testing.T) {
	f := NewFakeTokenStore()
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "m", PoolAddr: "p"})
	f.UpdateSafety(context.Background(), SafetyUpdate{Mint: "m", CreatorHoldingPct: 55, CreatorHoldingKnown: true})
	tg, _ := f.ManipulationTargets(context.Background(), 10)
	if tg[0].CreatorHoldingPct != 55 {
		t.Fatalf("known=true → 55 yazılmalı, gelen %.1f", tg[0].CreatorHoldingPct)
	}
	f.UpdateSafety(context.Background(), SafetyUpdate{Mint: "m", CreatorHoldingKnown: false, CreatorHoldingPct: 0})
	tg, _ = f.ManipulationTargets(context.Background(), 10)
	if tg[0].CreatorHoldingPct != 55 {
		t.Fatalf("known=false → mevcut 55 EZİLMEMELİ, gelen %.1f", tg[0].CreatorHoldingPct)
	}
}
```
`safety/provider_test.go` + `worker_test.go`'daki mock'ları yeni imzalara güncelle (`FetchOnChain(ctx, mint, creator)`, `HolderDistribution(ctx, mint, creator, cap)` — 5 dönüş). Mevcut assert'ler korunur; creator="" geçilirse creatorPct=0 (regresyonsuz).

- [ ] **Step 9: Test + build + vet yeşil**

Run: `go test ./internal/ingest/ ./internal/safety/ ./internal/store/ -race && go build ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add apps/api-go/internal/
git commit -m "feat(2c): safety worker creator_holding_pct türetme+koşullu persist (sıfır ek RPC; Scorer değişmez)"
```

---

### Task 6: Detail bağlama — `manipulationRisk` + işlem-akışı metrikleri

Skorları/metrikleri DB'den okuyup `TokenDetail`'e bağla. Option A: canlı çağrı yok.

**Files:**
- Modify: `apps/api-go/internal/store/token_detail.go` (`TokenDetailBase` alanları)
- Modify: `apps/api-go/internal/store/tokens.go` (`TokenDetailBase` postgres SELECT+Scan)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (`TokenDetailBase` fake alanları)
- Modify: `apps/api-go/internal/market/detail.go` (`Build` — manipulationRisk + metrics)
- Test: `apps/api-go/internal/store/token_detail_test.go`, `apps/api-go/internal/market/detail_test.go`

**Interfaces:**
- Consumes: mevcut `TokenDetailBase`, `market/detail.go Build`, `neutralScores`.
- Produces:
  - `store.TokenDetailBase` +`ManipulationScore, ManipulationConfidence float64; ManipulationBreakdown []ScoreBreakdownItem; ManipulationScoredTs int64; TxnsBuys, TxnsSells, TxnsBuyers int; CreatorHoldingPct float64`

- [ ] **Step 1: Failing test yaz (fake TokenDetailBase manipülasyon alanları)**

`token_detail_test.go`'ya ekle:
```go
func TestFakeTokenDetailBaseManipulation(t *testing.T) {
	f := NewFakeTokenStore()
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "m", PoolAddr: "p"})
	f.UpdateMarket(context.Background(), MarketUpdate{Mint: "m", Liquidity: 1000, Vol24h: 5000,
		TxnsBuys: 70, TxnsSells: 30, TxnsBuyers: 25})
	f.UpdateSafety(context.Background(), SafetyUpdate{Mint: "m", CreatorHoldingPct: 33, CreatorHoldingKnown: true})
	f.UpdateManipulation(context.Background(), ManipulationUpdate{Mint: "m", Score: 48, Confidence: 0.7,
		Breakdown: []ScoreBreakdownItem{{Label: "x", Weight: 48, Detail: "y"}}, ScoredTs: 99})
	b, ok, err := f.TokenDetailBase(context.Background(), "m")
	if err != nil || !ok {
		t.Fatalf("base: ok=%v err=%v", ok, err)
	}
	if b.ManipulationScore != 48 || b.ManipulationConfidence != 0.7 || b.ManipulationScoredTs != 99 {
		t.Fatalf("manipülasyon alanları yanlış: %+v", b)
	}
	if b.TxnsBuys != 70 || b.TxnsSells != 30 || b.TxnsBuyers != 25 || b.CreatorHoldingPct != 33 {
		t.Fatalf("işlem-akışı alanları yanlış: %+v", b)
	}
}
```

- [ ] **Step 2: Testin fail ettiğini doğrula**

Run: `go test ./internal/store/ -run TestFakeTokenDetailBaseManipulation -v`
Expected: FAIL (alanlar yok).

- [ ] **Step 3: TokenDetailBase struct + fake + postgres okuma**

`token_detail.go` `TokenDetailBase`'e ekle (creator reputation bloğundan sonra):
```go
	// 2c manipulation risk (tokens kolonlarından; detail scores.manipulationRisk'e) + işlem-akışı metrikleri.
	ManipulationScore, ManipulationConfidence float64
	ManipulationBreakdown                     []ScoreBreakdownItem
	ManipulationScoredTs                      int64
	TxnsBuys, TxnsSells, TxnsBuyers           int
	CreatorHoldingPct                         float64
```
fake `TokenDetailBase` return literaline ekle:
```go
		ManipulationScore: t.manipScore, ManipulationConfidence: t.manipConf,
		ManipulationBreakdown: manipBreakdownOrEmpty(t.manipBreakdown), ManipulationScoredTs: t.manipScoredTs,
		TxnsBuys: t.txnsBuys, TxnsSells: t.txnsSells, TxnsBuyers: t.txnsBuyers,
		CreatorHoldingPct: t.creatorHoldingPct,
```
fake dosyasına yardımcı ekle (nil→boş dilim, parity):
```go
func manipBreakdownOrEmpty(b []ScoreBreakdownItem) []ScoreBreakdownItem {
	if b == nil {
		return []ScoreBreakdownItem{}
	}
	return b
}
```
postgres `TokenDetailBase` SQL'ine kolonları ekle (SELECT sonuna, `COALESCE(c.breakdown,'')` sonrası) ve Scan'i genişlet:
```go
	// SELECT'e ekle:
	//   , tokens.manipulation_score, tokens.manipulation_confidence, tokens.manipulation_breakdown,
	//     tokens.manipulation_scored_ts, tokens.txns_buys, tokens.txns_sells, tokens.txns_buyers,
	//     tokens.creator_holding_pct
	var manipBdJSON string
	// Scan sonuna ekle:
	//   &b.ManipulationScore, &b.ManipulationConfidence, &manipBdJSON, &b.ManipulationScoredTs,
	//   &b.TxnsBuys, &b.TxnsSells, &b.TxnsBuyers, &b.CreatorHoldingPct
	// Scan sonrası:
	b.ManipulationBreakdown = parseBreakdownJSON(manipBdJSON)
```

- [ ] **Step 4: market/detail.go Build — manipulationRisk + metrics**

`detail.go`'da creatorReputation bloğundan sonra (ve `neutralScores()` manipulationRisk'i override edilir):
```go
	// Manipulation Risk (2c) — DB'den (tokens kolonları, arka plan scorer persist etti); opportunity nötr kalır.
	manipUpdated := "—"
	if base.ManipulationScoredTs > 0 {
		manipUpdated = time.Unix(base.ManipulationScoredTs, 0).UTC().Format(time.RFC3339)
	}
	d.Scores["manipulationRisk"] = store.ScoreDetail{
		Key: "manipulationRisk", Value: base.ManipulationScore, Confidence: base.ManipulationConfidence,
		UpdatedAt: manipUpdated, Breakdown: base.ManipulationBreakdown,
	}
	if d.Scores["manipulationRisk"].Breakdown == nil {
		sd := d.Scores["manipulationRisk"]
		sd.Breakdown = []store.ScoreBreakdownItem{}
		d.Scores["manipulationRisk"] = sd
	}

	// İşlem-akışı metrikleri (2c) — top10/holders zaten set; sniperPct/botActivityPct 2e'ye kadar 0.
	txns := base.TxnsBuys + base.TxnsSells
	d.Metrics.UniqueBuyers = base.TxnsBuyers
	if txns > 0 {
		d.Metrics.BuyRatio = float64(base.TxnsBuys) / float64(txns)
		d.Metrics.SellRatio = float64(base.TxnsSells) / float64(txns)
	}
	d.Metrics.CreatorHoldingPct = base.CreatorHoldingPct
```

- [ ] **Step 5: detail_test.go — manipülasyon skoru + metrik testi**

`market/detail_test.go`'daki mevcut fake DetailStore desenini izleyerek `TokenDetailBase`'i manipülasyon alanlarıyla döndüren bir senaryo ekle; assert:
```go
	// d.Scores["manipulationRisk"].Value == base.ManipulationScore
	// d.Metrics.UniqueBuyers == base.TxnsBuyers
	// d.Metrics.BuyRatio ~ buys/(buys+sells)
	// d.Metrics.CreatorHoldingPct == base.CreatorHoldingPct
```
(Mevcut test dosyasındaki stub `DetailStore` struct alan-eklemesiyle güncellenir; nil-breakdown → `[]` guard doğrulanır.)

- [ ] **Step 6: Test + build + vet yeşil**

Run: `go test ./internal/store/ ./internal/market/ -race && go build ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/
git commit -m "feat(2c): TokenDetail manipulationRisk + işlem-akışı metrikleri DB'den (Option A)"
```

---

### Task 7: Config + main wiring + README

Worker'ı ayağa kaldır + eşikleri deploy-tunable yap.

**Files:**
- Modify: `apps/api-go/internal/config/config.go` (`MANIPULATION_*`)
- Modify: `apps/api-go/cmd/server/main.go` (worker goroutine)
- Modify: `apps/api-go/README.md` (env tablosu + 2c notu)
- Test: `apps/api-go/internal/config/config_test.go`

**Interfaces:**
- Consumes: `manipulation.NewWorker`, `manipulation.WorkerDeps`, `manipulation.Thresholds`, `config.Config`.
- Produces: `config.Config` +12 `Manipulation*` alanı.

- [ ] **Step 1: Failing test yaz (config defaults)**

`config_test.go`'ya ekle (mevcut default-testi desenini izle):
```go
func TestManipulationDefaults(t *testing.T) {
	c := Load()
	if !c.ManipulationEnabled {
		t.Fatal("MANIPULATION_ENABLED default true beklenir")
	}
	if c.ManipulationMinTxns != 20 || c.ManipulationConfTxns != 100 {
		t.Fatalf("eşik defaultları yanlış: minTxns=%d confTxns=%d", c.ManipulationMinTxns, c.ManipulationConfTxns)
	}
	if c.ManipulationWImbalance != 30 || c.ManipulationWWash != 35 ||
		c.ManipulationWVolume != 25 || c.ManipulationWCreator != 10 {
		t.Fatalf("ağırlık defaultları yanlış: %+v", c)
	}
}
```

- [ ] **Step 2: Testin fail ettiğini doğrula**

Run: `go test ./internal/config/ -run TestManipulation -v`
Expected: FAIL (alanlar yok).

- [ ] **Step 3: Config alanlarını + Load()'u ekle**

`config.go` `Config` struct'ına (Reputation bloğu sonrası):
```go
	ManipulationEnabled     bool
	ManipulationIntervalSec int
	ManipulationLimit       int
	ManipulationMinTxns     int
	ManipulationConfTxns    int
	ManipulationWImbalance  float64
	ManipulationWWash       float64
	ManipulationWVolume     float64
	ManipulationWCreator    float64
	ManipulationWashMin     float64
	ManipulationWashMax     float64
	ManipulationVolMin      float64
	ManipulationVolMax      float64
```
`Load()`'a (Reputation bloğu sonrası):
```go
		ManipulationEnabled:     getenvBool("MANIPULATION_ENABLED", true),
		ManipulationIntervalSec: getenvInt("MANIPULATION_INTERVAL_SEC", 60),
		ManipulationLimit:       getenvInt("MANIPULATION_LIMIT", 60),
		ManipulationMinTxns:     getenvInt("MANIPULATION_MIN_TXNS", 20),
		ManipulationConfTxns:    getenvInt("MANIPULATION_CONF_TXNS", 100),
		ManipulationWImbalance:  getenvFloat("MANIPULATION_W_IMBALANCE", 30),
		ManipulationWWash:       getenvFloat("MANIPULATION_W_WASH", 35),
		ManipulationWVolume:     getenvFloat("MANIPULATION_W_VOLUME", 25),
		ManipulationWCreator:    getenvFloat("MANIPULATION_W_CREATOR", 10),
		ManipulationWashMin:     getenvFloat("MANIPULATION_WASH_MIN", 3),
		ManipulationWashMax:     getenvFloat("MANIPULATION_WASH_MAX", 15),
		ManipulationVolMin:      getenvFloat("MANIPULATION_VOL_MIN", 3),
		ManipulationVolMax:      getenvFloat("MANIPULATION_VOL_MAX", 20),
```
> Not: `getenvFloat`/`getenvInt` `>0` şartıyla default'a düşer → `MIN` alanları için 0 girilemez (kabul; band-min 0 istenirse çok küçük değer). `WASH_MIN`/`VOL_MIN` için bu YAGNI sınırı (spec'te park).

- [ ] **Step 4: main.go worker wiring**

`main.go`'da reputation worker bloğundan sonra:
```go
	// manipulation risk scorer (2c) — arka plan; saf DB (RPC YOK)
	if cfg.ManipulationEnabled && bundle.Tokens != nil {
		mw := manipulation.NewWorker(manipulation.WorkerDeps{
			Store: bundle.Tokens,
			Thresholds: manipulation.Thresholds{
				MinTxns: cfg.ManipulationMinTxns, ConfTxns: cfg.ManipulationConfTxns,
				WImbalance: cfg.ManipulationWImbalance, WWash: cfg.ManipulationWWash,
				WVolume: cfg.ManipulationWVolume, WCreator: cfg.ManipulationWCreator,
				WashMin: cfg.ManipulationWashMin, WashMax: cfg.ManipulationWashMax,
				VolMin: cfg.ManipulationVolMin, VolMax: cfg.ManipulationVolMax,
			},
			Interval: time.Duration(cfg.ManipulationIntervalSec) * time.Second, Limit: cfg.ManipulationLimit, Logger: logger,
		})
		go mw.Run(ctx)
	}
```
Import ekle: `"github.com/furkanatesc/sentinel/apps/api-go/internal/manipulation"`.

- [ ] **Step 5: README env tablosu + 2c notu**

`apps/api-go/README.md`'ye `MANIPULATION_*` satırlarını env tablosuna ekle + kısa 2c bölümü (agrega-proxy, saf-DB, inverted; ertelenen sniper/bot/creatorSell → 2e).

- [ ] **Step 6: Test + build + vet + full race yeşil**

Run: `go test -race ./... && go build ./... && go vet ./...`
Expected: PASS (tüm paketler).

- [ ] **Step 7: gofmt**

Run: `gofmt -l . ` (boş çıktı beklenir; değilse `gofmt -w .`)
> CI'da gofmt gate YOK (`api-go.yml` yalnız build+test) → elle temizle.

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/
git commit -m "feat(2c): config MANIPULATION_* + main worker wiring + README (2c tamam)"
```

---

## Frontend doğrulaması (sıfır kod değişikliği)

- [ ] `apps/web/` DEĞİŞMEDİ (`git status` temiz). Mevcut frontend testleri regresyonsuz — çalıştır:

Run: `cd apps/web && npm test`
Expected: mevcut sayı (≥193) PASS.

---

## Deploy sonrası doğrulama (kod değil — deploy adımı, kullanıcı onaylı)

Merge + Railway otomatik deploy sonrası (migration 0010 goose ile):
- `/healthz` 200.
- `/api/token/{mint}` (likit, enrichment görmüş bir mint) → `scores.manipulationRisk.value` gerçek (0 değil, `breakdown` dolu); `metrics.uniqueBuyers`/`buyRatio`/`sellRatio`/`creatorHoldingPct` dolu.
- **Kalibrasyon riski:** GeckoTerminal `transactions.h24` alan-adı/şekli (1a/2a deseni — canlıda teyit); eşikler `MANIPULATION_*` ile tunable (kod değişmez).
- Düşük-aktivite token'larda `manipulationRisk=0/conf=0` = dürüst (txns<20).

---

## Followups (bilinçle ertelenen — sessiz düşürme yok)

`followups-frontend.md` "2c" bölümüne yaz:
- `sniperPct`/`botActivityPct`/`creatorSellPct`/`CreatorBehavior.*` → 2e (per-wallet/trade-flow).
- `txns_sellers` persist ama Scorer tüketmez (satış-tarafı wash → 2e simetrik).
- `getenvFloat` `>0` → `WASH_MIN`/`VOL_MIN` 0 girilemez (band-min footgun sınırı).
- creator_holding yalnız safety-scored token'da dolar (kısmi-veri; skorlanmamışta creatorNorm=0).
- Postgres `ManipulationTargets`/`UpdateManipulation`/`UpdateSafety` koşullu / TokenDetailBase yeni kolonlar için canlı-DB parity testi (DB-gated env; deploy-doğrulama deseni).
