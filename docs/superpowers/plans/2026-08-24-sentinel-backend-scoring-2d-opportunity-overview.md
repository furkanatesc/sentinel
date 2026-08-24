# 2d Opportunity + Overview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Confidence-ağırlıklı kompozit `opportunity` skoru + `signal` türetme + `getKpis`/`getRadar` Overview endpoint'lerini gerçeğe döndürmek (saf-DB, sıfır RPC).

**Architecture:** Yeni `internal/opportunity/` paketi (saf Scorer + saf-DB Worker, 2c deseni). Opportunity zaten persist edilmiş skorları (safety/creator-reputation/manipulation/momentum) birleştirir — canlı çağrı yok, Option A. Overview endpoint'leri token agregasından/projeksiyonundan türetilir. Frontend yalnız 2 dosya (http.ts + live-endpoints.ts); UI dokunulmaz.

**Tech Stack:** Go 1.24 (chi, pgx/database/sql, goose migration), TypeScript/Next.js frontend. Test: Go `testing` + `-race`, frontend vitest.

**Spec:** `docs/superpowers/specs/2026-08-24-sentinel-backend-scoring-2d-opportunity-overview-design.md`

## Global Constraints

- **Saf-DB, sıfır RPC/dış-çağrı** — opportunity worker yalnız store'dan okur/yazar (Helius/GeckoTerminal YOK).
- **Yeni env opsiyonel + default'lu** — `OPPORTUNITY_ENABLED`(true)/`OPPORTUNITY_INTERVAL_SEC`(60)/`OPPORTUNITY_LIMIT`(100). Yeni key/ücret YOK.
- **Nötr = dürüst, sahte değil** — yetersiz veri (tüm girdi conf=0) → `value:0, confidence:0, breakdown:[]`; sahte skor asla.
- **Ağırlıklar/eşikler kod-sabiti** (paket düzeyi const): safety 0.30, creator 0.25, manipülasyon-ters 0.25, momentum 0.20 (Σ=1.00); signal `buy`≥70, `watch`≥45, min-conf **0.25** (DÜZELTME 2026-08-24: 0.35'ten düşürüldü — en ağır tek-girdi ağırlığı [safety 0.30] altında olmalı ki tek confident girdi sinyal verebilsin; aksi hiçbir tek-girdi token'ı buy/watch/avoid alamazdı).
- **Fake/Postgres parity** — her yeni store metodu hem `postgresStore` hem fake'te aynı semantikle; parity testi.
- **Frontend kontratı korunur** — `Kpi`/`RadarPoint`/`TokenRow.signal`/`ScoreDetail` JSON şekilleri `apps/web/lib/api/types.ts` ile birebir.
- **Migration idempotent** — `ADD COLUMN IF NOT EXISTS`; goose up/down.
- **`riskLevel`/`RiskLevel` string değerleri** frontend `scoreToLevel` (format.ts) ile birebir: `≤24 critical, ≤49 high, ≤69 medium, ≤84 good, else strong`.

---

### Task 1: Migration 0011 + opportunity store target/update (fake+postgres parity)

**Files:**
- Create: `apps/api-go/internal/store/migrations/0011_add_token_opportunity.sql`
- Modify: `apps/api-go/internal/store/tokens.go` (TokenStore interface + postgres impl + tipler)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake impl + parity)
- Test: `apps/api-go/internal/store/opportunity_test.go` (yeni)

**Interfaces:**
- Produces:
  - `type OpportunityTarget struct { Mint string; Safety, SafetyConf, Creator, CreatorConf, Manipulation, ManipulationConf, Momentum, Liquidity float64 }`
  - `type OpportunityUpdate struct { Mint string; Score, Confidence float64; Breakdown []ScoreBreakdownItem; Signal string; ScoredTs int64 }`
  - `OpportunityScoreTargets(ctx, limit int) ([]OpportunityTarget, error)` — tokens LEFT JOIN creators.
  - `UpdateOpportunity(ctx, u OpportunityUpdate) error` — 4 kolon + `last_signal`.

- [ ] **Step 1: Migration dosyası**

`0011_add_token_opportunity.sql`:
```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_score      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_breakdown  TEXT             NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_scored_ts  BIGINT           NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS last_signal            TEXT             NOT NULL DEFAULT ''; -- DÜZELTME 2026-08-24: tokens'ta yoktu (0001 strategies'te var, farklı domain)

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_score;
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_confidence;
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_breakdown;
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_scored_ts;
```

- [ ] **Step 2: Failing test (parity)**

`opportunity_test.go` — fake store üzerinden target+update round-trip (postgres DB-gated, fake her ortamda çalışır):
```go
package store

import (
	"context"
	"testing"
)

func TestOpportunityTargetsAndUpdate_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	// Bir token + skorları hazırla (fake setter'lar mevcut UpsertToken/UpdateSafety... ile).
	seedScoredToken(t, fs) // yardımcı: mint "m1", safety 80/conf1, manip 10/conf1, momentum 60, liq 1000
	ts := fs.(TokenStore)

	targets, err := ts.OpportunityScoreTargets(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) == 0 {
		t.Fatal("hedef bekleniyordu")
	}
	var got *OpportunityTarget
	for i := range targets {
		if targets[i].Mint == "m1" {
			got = &targets[i]
		}
	}
	if got == nil || got.Safety != 80 || got.Momentum != 60 || got.Liquidity != 1000 {
		t.Fatalf("target yanlış: %+v", got)
	}

	err = ts.UpdateOpportunity(ctx, OpportunityUpdate{
		Mint: "m1", Score: 72, Confidence: 0.8,
		Breakdown: []ScoreBreakdownItem{{Label: "x", Weight: 1, Detail: "y"}},
		Signal: "buy", ScoredTs: 123,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Signal RecentTokens'a yansımalı (Task 6'da select edilecek; burada fake alanı doğrula).
	rows, _ := ts.RecentTokens(ctx, 10)
	for _, r := range rows {
		if r.Mint == "m1" && (r.Signal == nil || *r.Signal != "buy") {
			t.Fatalf("signal beklenen buy, got %v", r.Signal)
		}
	}
}
```
> Not: `seedScoredToken` yardımcısını fake'in mevcut setter'larıyla yaz (bkz `fake_ingest.go` UpsertToken/UpdateSafety/UpdateManipulation/UpdateMarket + creators fake). Fake `RecentTokens` şu an `Signal`'ı set etmiyorsa Task 6'ya kadar bu assert kısmı için fake `RecentTokens` `last_signal` alanını okumalı — fake impl'i bu task'ta güncelle (fake in-memory `signal` map'i UpdateOpportunity'de doldur, RecentTokens'ta oku).

- [ ] **Step 3: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/store/ -run TestOpportunityTargetsAndUpdate_Fake -v`
Expected: FAIL (`OpportunityScoreTargets`/`UpdateOpportunity` undefined).

- [ ] **Step 4: Tipleri + interface + postgres impl ekle**

`tokens.go` — tipler (mevcut tip bloklarının yanına):
```go
// OpportunityTarget, kompozit opportunity için gereken alt-skorlar + confidence'lar (JOIN creators).
type OpportunityTarget struct {
	Mint                                                                       string
	Safety, SafetyConf, Creator, CreatorConf, Manipulation, ManipulationConf   float64
	Momentum, Liquidity                                                        float64
}

// OpportunityUpdate, opportunity scorer'ının yazdığı kompozit sonuçtur (+ türetilmiş signal).
type OpportunityUpdate struct {
	Mint       string
	Score      float64
	Confidence float64
	Breakdown  []ScoreBreakdownItem
	Signal     string // "buy"|"watch"|"avoid"|"" ("" → last_signal boş → frontend null)
	ScoredTs   int64
}
```
TokenStore interface'e ekle (mevcut 2c satırlarının altına):
```go
	// 2d: kompozit opportunity girdilerini döndürür / hesaplanmış skoru+signal'ı persist eder.
	OpportunityScoreTargets(ctx context.Context, limit int) ([]OpportunityTarget, error)
	UpdateOpportunity(ctx context.Context, u OpportunityUpdate) error
```
postgres impl (`tokens.go` sonuna):
```go
func (p *postgresStore) OpportunityScoreTargets(ctx context.Context, limit int) ([]OpportunityTarget, error) {
	const q = `SELECT t.mint, t.safety_score, t.safety_confidence,
		COALESCE(c.reputation_score,0), COALESCE(c.confidence,0),
		t.manipulation_score, t.manipulation_confidence, t.momentum, t.liquidity
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		ORDER BY t.first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OpportunityTarget, 0, limit)
	for rows.Next() {
		var t OpportunityTarget
		if err := rows.Scan(&t.Mint, &t.Safety, &t.SafetyConf, &t.Creator, &t.CreatorConf,
			&t.Manipulation, &t.ManipulationConf, &t.Momentum, &t.Liquidity); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) UpdateOpportunity(ctx context.Context, u OpportunityUpdate) error {
	bd, err := json.Marshal(u.Breakdown)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET opportunity_score=$2, opportunity_confidence=$3,
		opportunity_breakdown=$4, opportunity_scored_ts=$5, last_signal=$6 WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, u.Mint, u.Score, u.Confidence, string(bd), u.ScoredTs, u.Signal)
	return err
}
```

- [ ] **Step 5: Fake impl (parity)**

`fake_ingest.go` — fake token yapısına `signal string` + `opportunity*` alanları ekle (mevcut in-memory token struct'ına). `OpportunityScoreTargets` fake: her token için safety/manip/momentum/liquidity + creators map'inden reputation/confidence; `UpdateOpportunity` fake: token'ın `signal`/opportunity alanlarını set eder. Fake `RecentTokens`: `signal != "" → *string, else nil`. (Postgres JOIN semantiğini birebir taklit et: creator yoksa reputation/conf 0.)

- [ ] **Step 6: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/store/ -run TestOpportunityTargetsAndUpdate_Fake -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/store/migrations/0011_add_token_opportunity.sql apps/api-go/internal/store/tokens.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/opportunity_test.go
git commit -m "feat(2d): migration 0011 + opportunity store target/update (fake+postgres parity)"
```

---

### Task 2: Saf Scorer (confidence-ağırlıklı kompozit + breakdown + signal)

**Files:**
- Create: `apps/api-go/internal/opportunity/scorer.go`
- Test: `apps/api-go/internal/opportunity/scorer_test.go`

**Interfaces:**
- Consumes: `store.ScoreBreakdownItem` (Task hiçbiri — mevcut tip).
- Produces:
  - `type Inputs struct { Safety, SafetyConf, Creator, CreatorConf, Manipulation, ManipulationConf, Momentum, Liquidity float64 }`
  - `type Result struct { Value, Confidence float64; Breakdown []store.ScoreBreakdownItem; Signal string }`
  - `func Score(in Inputs) Result`

- [ ] **Step 1: Failing test**

`scorer_test.go`:
```go
package opportunity

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.5 }

func TestScore_CompositeWeighted(t *testing.T) {
	// safety 80@1, creator 60@1, manip 10@1 (→ters 90), momentum 50, liq>0 (conf1)
	// W = 0.30+0.25+0.25+0.20 = 1.00
	// num = 80*0.30 + 60*0.25 + 90*0.25 + 50*0.20 = 24+15+22.5+10 = 71.5
	r := Score(Inputs{Safety: 80, SafetyConf: 1, Creator: 60, CreatorConf: 1,
		Manipulation: 10, ManipulationConf: 1, Momentum: 50, Liquidity: 1000})
	if r.Value != 72 { // round(71.5) = 72 (math.Round half-away-from-zero)
		t.Fatalf("value=%.2f want 72 (round of 71.5)", r.Value)
	}
	if !approx(r.Confidence, 1.0) {
		t.Fatalf("conf=%.2f want 1.0", r.Confidence)
	}
	if r.Signal != "buy" {
		t.Fatalf("signal=%q want buy", r.Signal)
	}
	if len(r.Breakdown) != 4 {
		t.Fatalf("breakdown=%d want 4", len(r.Breakdown))
	}
}

func TestScore_NeutralWhenAllConfZero(t *testing.T) {
	r := Score(Inputs{Safety: 80, SafetyConf: 0, Creator: 60, CreatorConf: 0,
		Manipulation: 10, ManipulationConf: 0, Momentum: 50, Liquidity: 0})
	if r.Value != 0 || r.Confidence != 0 || len(r.Breakdown) != 0 || r.Signal != "" {
		t.Fatalf("nötr bekleniyordu: %+v", r)
	}
}

func TestScore_MomentumGatedByLiquidity(t *testing.T) {
	// momentum var ama liquidity=0 → momentum katkı vermez (conf0). Yalnız safety kalır.
	r := Score(Inputs{Safety: 80, SafetyConf: 1, Momentum: 90, Liquidity: 0})
	if !approx(r.Value, 80) { // W=0.30, num=80*0.30 → /0.30 = 80
		t.Fatalf("value=%.2f want 80 (momentum atlanmalı)", r.Value)
	}
	if !approx(r.Confidence, 0.30) {
		t.Fatalf("conf=%.2f want 0.30", r.Confidence)
	}
}

func TestScore_ManipulationInverted(t *testing.T) {
	// yüksek manip (90) tek girdi → ters 10 → düşük opportunity → avoid
	r := Score(Inputs{Manipulation: 90, ManipulationConf: 1})
	if !approx(r.Value, 10) {
		t.Fatalf("value=%.2f want 10 (100-90)", r.Value)
	}
	if r.Signal != "avoid" {
		t.Fatalf("signal=%q want avoid", r.Signal)
	}
}

func TestSignal_Thresholds(t *testing.T) {
	// watch: value 45-69 + yeterli conf
	r := Score(Inputs{Safety: 50, SafetyConf: 1})
	if r.Signal != "watch" {
		t.Fatalf("value=%.1f signal=%q want watch", r.Value, r.Signal)
	}
	// düşük conf → null (""): kısmi-confidence tek girdi → overall conf < 0.25.
	r2 := Score(Inputs{Safety: 90, SafetyConf: 0.5}) // conf = 0.30*0.5 = 0.15 < 0.25 → null
	if r2.Signal != "" {
		t.Fatalf("conf=%.2f signal=%q want '' (null)", r2.Confidence, r2.Signal)
	}
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/opportunity/ -v`
Expected: FAIL (paket/`Score` yok).

- [ ] **Step 3: Implement scorer**

`scorer.go`:
```go
// Package opportunity, diğer üç skoru (+ momentum) confidence-ağırlıklı birleştirip 0-100
// kompozit "fırsat" skoru + signal üretir. Saf, ağsız/DB'siz (SRP). Girdiler iyileştikçe
// (safety holders DAS, creator WS) opportunity otomatik keskinleşir — türev lens.
package opportunity

import (
	"fmt"
	"math"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Ağırlıklar (Σ=1.00) ve signal eşikleri — kod-sabiti (kalibrasyon; YAGNI: config değil).
const (
	wSafety       = 0.30
	wCreator      = 0.25
	wManipulation = 0.25
	wMomentum     = 0.20

	signalBuy     = 70.0
	signalWatch   = 45.0
	signalMinConf = 0.25 // en ağır tek-girdi ağırlığı (safety 0.30) altında: tek confident girdi sinyal verebilir
)

type Inputs struct {
	Safety, SafetyConf             float64
	Creator, CreatorConf           float64
	Manipulation, ManipulationConf float64
	Momentum, Liquidity            float64
}

type Result struct {
	Value, Confidence float64
	Breakdown         []store.ScoreBreakdownItem
	Signal            string // "buy"|"watch"|"avoid"|"" ("" → null)
}

type comp struct {
	label  string
	score  float64 // opportunity yönünde (yüksek=iyi), 0-100
	conf   float64
	weight float64
	detail string
}

func Score(in Inputs) Result {
	// momentum confidence proxy: enrichment yoksa (liquidity==0) momentum "bilinmiyor" → conf0.
	momentumConf := 0.0
	if in.Liquidity > 0 {
		momentumConf = 1.0
	}
	comps := []comp{
		{"Token güvenliği", in.Safety, in.SafetyConf, wSafety, fmt.Sprintf("%.0f/100 (conf %%%.0f)", in.Safety, in.SafetyConf*100)},
		{"Üretici itibarı", in.Creator, in.CreatorConf, wCreator, fmt.Sprintf("%.0f/100 (conf %%%.0f)", in.Creator, in.CreatorConf*100)},
		{"Manipülasyon (ters)", 100 - in.Manipulation, in.ManipulationConf, wManipulation, fmt.Sprintf("100-%.0f=%.0f (conf %%%.0f)", in.Manipulation, 100-in.Manipulation, in.ManipulationConf*100)},
		{"Momentum", in.Momentum, momentumConf, wMomentum, fmt.Sprintf("%.0f/100", in.Momentum)},
	}

	var num, W, wsum float64
	for _, c := range comps {
		wsum += c.weight
		ew := c.weight * c.conf
		num += c.score * ew
		W += ew
	}
	if W == 0 {
		return Result{Value: 0, Confidence: 0, Breakdown: []store.ScoreBreakdownItem{}, Signal: ""}
	}
	value := clamp(math.Round(num/W), 0, 100)
	confidence := W / wsum

	bd := []store.ScoreBreakdownItem{}
	for _, c := range comps {
		if c.conf <= 0 {
			continue // katkısız girdi breakdown'a girmez (dürüst)
		}
		contrib := c.score * c.weight * c.conf / W
		bd = append(bd, store.ScoreBreakdownItem{Label: c.label, Weight: round1(contrib), Detail: c.detail})
	}
	return Result{Value: value, Confidence: confidence, Breakdown: bd, Signal: deriveSignal(value, confidence)}
}

func deriveSignal(value, conf float64) string {
	if conf < signalMinConf {
		return ""
	}
	switch {
	case value >= signalBuy:
		return "buy"
	case value >= signalWatch:
		return "watch"
	default:
		return "avoid"
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
```

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/opportunity/ -v`
Expected: PASS (5 test).

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/opportunity/scorer.go apps/api-go/internal/opportunity/scorer_test.go
git commit -m "feat(2d): saf opportunity scorer (conf-ağırlıklı kompozit + signal)"
```

---

### Task 3: Worker (saf-DB; hedef çek → skorla → persist)

**Files:**
- Create: `apps/api-go/internal/opportunity/worker.go`
- Test: `apps/api-go/internal/opportunity/worker_test.go`

**Interfaces:**
- Consumes: `Score(Inputs) Result` (Task 2); `store.OpportunityTarget`/`OpportunityUpdate`/`OpportunityScoreTargets`/`UpdateOpportunity` (Task 1).
- Produces: `type WorkerDeps struct { Store OpportunityStore; Interval time.Duration; Limit int; Logger *slog.Logger; Now func() time.Time }`; `func NewWorker(WorkerDeps) *Worker`; `(*Worker).Run(ctx)`.
- `type OpportunityStore interface { OpportunityScoreTargets(...); UpdateOpportunity(...) }` (ISP — dar arayüz, store.TokenStore karşılar).

- [ ] **Step 1: Failing test**

`worker_test.go` (2c worker_test deseni — fake store + partial-error izole + ctx-cancel):
```go
package opportunity

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeOppStore struct {
	targets []store.OpportunityTarget
	updates []store.OpportunityUpdate
	failOn  string // bu mint'te UpdateOpportunity hata → izole edilmeli
}

func (f *fakeOppStore) OpportunityScoreTargets(_ context.Context, _ int) ([]store.OpportunityTarget, error) {
	return f.targets, nil
}
func (f *fakeOppStore) UpdateOpportunity(_ context.Context, u store.OpportunityUpdate) error {
	if u.Mint == f.failOn {
		return errors.New("boom")
	}
	f.updates = append(f.updates, u)
	return nil
}

func TestWorker_ScoresAndPersists_IsolatesError(t *testing.T) {
	fs := &fakeOppStore{
		targets: []store.OpportunityTarget{
			{Mint: "ok", Safety: 80, SafetyConf: 1, Liquidity: 1000, Momentum: 60},
			{Mint: "bad", Safety: 80, SafetyConf: 1, Liquidity: 1000, Momentum: 60},
		},
		failOn: "bad",
	}
	w := NewWorker(WorkerDeps{Store: fs, Limit: 10,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:    func() time.Time { return time.Unix(1000, 0) }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fs.updates) != 1 || fs.updates[0].Mint != "ok" {
		t.Fatalf("yalnız 'ok' persist edilmeli, got %+v", fs.updates)
	}
	if fs.updates[0].ScoredTs != 1000 {
		t.Fatalf("scoredTs=%d want 1000", fs.updates[0].ScoredTs)
	}
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/opportunity/ -run TestWorker -v`
Expected: FAIL (`NewWorker`/`WorkerDeps`/`scoreOnce` yok).

- [ ] **Step 3: Implement worker** (2c `internal/manipulation/worker.go` deseni birebir — kopyala + opportunity'ye uyarla)

`worker.go`:
```go
package opportunity

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// OpportunityStore, worker'ın bağımlı olduğu dar arayüzdür (ISP; store.TokenStore karşılar).
type OpportunityStore interface {
	OpportunityScoreTargets(ctx context.Context, limit int) ([]store.OpportunityTarget, error)
	UpdateOpportunity(ctx context.Context, u store.OpportunityUpdate) error
}

type WorkerDeps struct {
	Store    OpportunityStore
	Interval time.Duration
	Limit    int
	Logger   *slog.Logger
	Now      func() time.Time
}

type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	// DÜZELTME 2026-08-24 (Task 3 review): zero-value guard'lar (manipulation worker deseni) —
	// zero Interval time.NewTicker panic'ler, nil Logger Warn/Info'da panic'ler.
	if d.Interval <= 0 {
		d.Interval = 60 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 60
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	for {
		if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
			w.d.Logger.Warn("opportunity cycle", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (w *Worker) scoreOnce(ctx context.Context) error {
	targets, err := w.d.Store.OpportunityScoreTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now().Unix()
	var scored int
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		res := Score(Inputs{
			Safety: tg.Safety, SafetyConf: tg.SafetyConf,
			Creator: tg.Creator, CreatorConf: tg.CreatorConf,
			Manipulation: tg.Manipulation, ManipulationConf: tg.ManipulationConf,
			Momentum: tg.Momentum, Liquidity: tg.Liquidity,
		})
		if err := w.d.Store.UpdateOpportunity(ctx, store.OpportunityUpdate{
			Mint: tg.Mint, Score: res.Value, Confidence: res.Confidence,
			Breakdown: res.Breakdown, Signal: res.Signal, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("update opportunity", "mint", tg.Mint, "err", err)
			continue
		}
		scored++
	}
	if len(targets) > 0 {
		w.d.Logger.Info("opportunity cycle", "targets", len(targets), "scored", scored)
	}
	return nil
}
```

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/opportunity/ -race -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/opportunity/worker.go apps/api-go/internal/opportunity/worker_test.go
git commit -m "feat(2d): opportunity worker (saf DB, kısmi-hata izole, ctx-cancel)"
```

---

### Task 4: Kpis store agrega

**Files:**
- Modify: `apps/api-go/internal/store/tokens.go` (Kpis + KpiCounts + interface)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake parity)
- Test: `apps/api-go/internal/store/opportunity_test.go` (mevcut dosyaya ekle)

**Interfaces:**
- Produces: `type KpiCounts struct { Detected, HighConf, Critical, Signals int }`; `Kpis(ctx) (KpiCounts, error)` (TokenStore).

- [ ] **Step 1: Failing test**

`opportunity_test.go`'ye ekle:
```go
func TestKpis_Counts_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	seedKpiTokens(t, fs) // yardımcı: 1 high-safe (safety80@1), 1 kritik (manip80@1), 1 buy-signal
	ts := fs.(TokenStore)
	c, err := ts.Kpis(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c.HighConf < 1 || c.Critical < 1 || c.Signals < 1 || c.Detected < 3 {
		t.Fatalf("kpi sayımları yanlış: %+v", c)
	}
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/store/ -run TestKpis -v`
Expected: FAIL (`Kpis` undefined).

- [ ] **Step 3: Implement**

`tokens.go` — tip + interface satırı + postgres impl:
```go
// KpiCounts, Overview KPI kartları için türetilebilir sayımlardır (2d).
type KpiCounts struct{ Detected, HighConf, Critical, Signals int }
```
Interface'e: `Kpis(ctx context.Context) (KpiCounts, error)`

postgres impl (tek agrega, eşikler §Global Constraints):
```go
func (p *postgresStore) Kpis(ctx context.Context) (KpiCounts, error) {
	const q = `SELECT
		COUNT(*) FILTER (WHERE first_seen_ts >= $1),
		COUNT(*) FILTER (WHERE safety_score >= 70 AND safety_confidence >= 0.5),
		COUNT(*) FILTER (WHERE (manipulation_score >= 70 AND manipulation_confidence >= 0.5)
		                    OR (safety_score <= 30 AND safety_confidence >= 0.5)),
		COUNT(*) FILTER (WHERE last_signal IN ('buy','watch'))
		FROM tokens`
	var c KpiCounts
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	err := p.db.QueryRowContext(ctx, q, cutoff).Scan(&c.Detected, &c.HighConf, &c.Critical, &c.Signals)
	return c, err
}
```
Fake impl (`fake_ingest.go`): in-memory token'lar üzerinde aynı eşiklerle say (24s: `firstSeenTs >= now-86400`).

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/store/ -run TestKpis -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/store/tokens.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/opportunity_test.go
git commit -m "feat(2d): Kpis store agrega (4 türetilebilir sayım, fake+postgres parity)"
```

---

### Task 5: Radar store projeksiyon + Go scoreToLevel parity

**Files:**
- Modify: `apps/api-go/internal/store/tokens.go` (Radar + RadarPoint + scoreToLevel + interface)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake parity)
- Test: `apps/api-go/internal/store/opportunity_test.go` (ekle)

**Interfaces:**
- Consumes: `RecentTokens` (mevcut) — Radar onu yeniden kullanır.
- Produces: `type RadarPoint struct { X,Y,Z float64; Name, Level string }`; `Radar(ctx, limit) ([]RadarPoint, error)`; `func scoreToLevel(score float64) string`.

- [ ] **Step 1: Failing test**

```go
func TestScoreToLevel_ParityWithFrontend(t *testing.T) {
	cases := []struct {
		s    float64
		want string
	}{{10, "critical"}, {24, "critical"}, {25, "high"}, {49, "high"},
		{50, "medium"}, {69, "medium"}, {70, "good"}, {84, "good"}, {85, "strong"}, {100, "strong"}}
	for _, c := range cases {
		if got := scoreToLevel(c.s); got != c.want {
			t.Fatalf("scoreToLevel(%.0f)=%q want %q", c.s, got, c.want)
		}
	}
}

func TestRadar_Projection_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	seedScoredToken(t, fs) // Task 1 yardımcısı (m1: creator/safety/momentum/liquidity dolu)
	ts := fs.(TokenStore)
	pts, err := ts.Radar(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) == 0 {
		t.Fatal("radar noktası bekleniyordu")
	}
	// x=creatorScore, z=liquidity projeksiyonu (RecentTokens ile aynı kaynak).
	if pts[0].Name == "" || pts[0].Level == "" {
		t.Fatalf("radar noktası eksik: %+v", pts[0])
	}
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/store/ -run "TestScoreToLevel|TestRadar" -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

`tokens.go`:
```go
// RadarPoint, Overview radar scatter noktası (frontend RadarPoint ile birebir). level: RiskLevel.
type RadarPoint struct {
	X    float64 `json:"x"` // creatorScore
	Y    float64 `json:"y"` // momentum
	Z    float64 `json:"z"` // liquidity
	Name string  `json:"name"`
	Level string `json:"level"`
}

// scoreToLevel, frontend format.ts scoreToLevel ile birebir (parity).
func scoreToLevel(score float64) string {
	switch {
	case score <= 24:
		return "critical"
	case score <= 49:
		return "high"
	case score <= 69:
		return "medium"
	case score <= 84:
		return "good"
	default:
		return "strong"
	}
}

func (p *postgresStore) Radar(ctx context.Context, limit int) ([]RadarPoint, error) {
	rows, err := p.RecentTokens(ctx, limit)
	if err != nil {
		return nil, err
	}
	return radarFrom(rows), nil
}

// radarFrom, TokenRow listesini radar noktalarına çevirir (mock radarFrom birebir).
func radarFrom(rows []TokenRow) []RadarPoint {
	out := make([]RadarPoint, 0, len(rows))
	for _, t := range rows {
		out = append(out, RadarPoint{
			X: t.CreatorScore, Y: t.Momentum, Z: t.Liquidity, Name: t.Symbol,
			Level: scoreToLevel(math.Round((t.CreatorScore + t.SafetyScore) / 2)),
		})
	}
	return out
}
```
> `math` import'unu `tokens.go`'ya ekle (yoksa). Interface'e: `Radar(ctx context.Context, limit int) ([]RadarPoint, error)`.
> Fake impl: fake `RecentTokens`'ı çağırıp aynı `radarFrom`'u kullan (ortak fonksiyon → parity otomatik). Fake, `Radar` metodunu `radarFrom(fake.RecentTokens(...))` ile implement eder.

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/store/ -run "TestScoreToLevel|TestRadar" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/store/tokens.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/opportunity_test.go
git commit -m "feat(2d): Radar store projeksiyon + Go scoreToLevel parity"
```

---

### Task 6: Detail/liste bağlama (scores.opportunity + TokenRow.signal DB'den)

**Files:**
- Modify: `apps/api-go/internal/store/token_detail.go` (TokenDetailBase opportunity alanları)
- Modify: `apps/api-go/internal/store/tokens.go` (`TokenDetailBase` SELECT opportunity kolonları) — **NOT (controller ruling 2026-08-24): `RecentTokens` SELECT `last_signal` Task 1'e taşındı; Task 6 RecentTokens'a DOKUNMAZ**
- Modify: `apps/api-go/internal/market/detail.go` (`Build` scores.opportunity bağla)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake TokenDetailBase opportunity + RecentTokens signal — Task 1'de kısmen yapıldı)
- Test: `apps/api-go/internal/market/detail_test.go` (ekle) + store parity testi

**Interfaces:**
- Consumes: `store.TokenDetailBase` (opportunity alanları eklenir); `store.ScoreDetail`.

- [ ] **Step 1: Failing test** (`detail_test.go`)

```go
func TestBuild_OpportunityFromDB(t *testing.T) {
	// fake TokenDetailBase opportunity_score=72/conf0.8/breakdown/scoredTs set → scores.opportunity dolmalı
	// (mevcut detail_test kurulumunu izle; base'e opportunity alanları ekle)
	// assert: d.Scores["opportunity"].Value == 72 && Confidence == 0.8 && Breakdown != nil
}
```
> Mevcut `detail_test.go`'daki fake TokenDetailProvider/base kurulumunu izle; opportunity alanlarını doldurup `Build` çıktısında `Scores["opportunity"]`'yi doğrula. Ayrıca `RecentTokens` signal parity için `opportunity_test.go`'da `last_signal` round-trip zaten Task 1'de test edildi.

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/market/ -run TestBuild_Opportunity -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

`token_detail.go` — `TokenDetailBase`'e ekle:
```go
	// 2d opportunity (tokens kolonlarından; detail scores.opportunity'e).
	OpportunityScore, OpportunityConfidence float64
	OpportunityBreakdown                    []ScoreBreakdownItem
	OpportunityScoredTs                     int64
```
`tokens.go` `TokenDetailBase` SELECT'ine `opportunity_score, opportunity_confidence, opportunity_breakdown, opportunity_scored_ts` kolonlarını ekle + scan + breakdown JSON parse (mevcut manipulation_breakdown parse deseni birebir).
(RecentTokens `last_signal` SELECT Task 1'de yapıldı — controller ruling; burada TEKRAR EKLEME.)
`detail.go` `Build` — manipulation bloğunun altına (satır ~172 sonrası), `opportunity nötr kalır` yorumunu kaldırıp:
```go
	// Opportunity (2d) — DB'den (tokens kolonları, arka plan worker persist etti).
	oppUpdated := "—"
	if base.OpportunityScoredTs > 0 {
		oppUpdated = time.Unix(base.OpportunityScoredTs, 0).UTC().Format(time.RFC3339)
	}
	d.Scores["opportunity"] = store.ScoreDetail{
		Key: "opportunity", Value: base.OpportunityScore, Confidence: base.OpportunityConfidence,
		UpdatedAt: oppUpdated, Breakdown: base.OpportunityBreakdown,
	}
	if d.Scores["opportunity"].Breakdown == nil {
		sd := d.Scores["opportunity"]
		sd.Breakdown = []store.ScoreBreakdownItem{}
		d.Scores["opportunity"] = sd
	}
```

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/market/ ./internal/store/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/store/token_detail.go apps/api-go/internal/store/tokens.go apps/api-go/internal/market/detail.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/market/detail_test.go
git commit -m "feat(2d): detail scores.opportunity + TokenRow.signal DB'den bağla"
```

---

### Task 7: API handlers + config + main wiring

**Files:**
- Create: `apps/api-go/internal/api/overview.go` (kpisHandler + radarHandler)
- Modify: `apps/api-go/internal/api/router.go` (route + RouterDeps)
- Create: `apps/api-go/internal/api/overview_test.go`
- Modify: `apps/api-go/internal/config/config.go` (OPPORTUNITY_*)
- Modify: `apps/api-go/cmd/server/main.go` (worker wiring + router deps)
- Test: `apps/api-go/internal/config/config_test.go` (default assert)

**Interfaces:**
- Consumes: `store.TokenStore` (Kpis/Radar); `opportunity.NewWorker` (Task 3).
- Produces: `type Kpi struct{...}`; `kpisHandler`/`radarHandler`; RouterDeps `Overview store.TokenStore` (veya mevcut `Tokens` yeniden kullan).

- [ ] **Step 1: Failing test** (`overview_test.go`)

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestKpisEndpoint(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Tokens: ts.(store.TokenStore)})
	req := httptest.NewRequest(http.MethodGet, "/api/kpis", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var kpis []Kpi
	json.NewDecoder(w.Body).Decode(&kpis)
	if len(kpis) != 8 {
		t.Fatalf("kpi=%d want 8 (4 gerçek + 4 placeholder)", len(kpis))
	}
	// placeholder'lar "—"
	byID := map[string]Kpi{}
	for _, k := range kpis {
		byID[k.ID] = k
	}
	if byID["positions"].Value != "—" {
		t.Fatalf("positions placeholder '—' olmalı, got %q", byID["positions"].Value)
	}
	if byID["detected"].Value == "—" {
		t.Fatalf("detected gerçek olmalı")
	}
}

func TestRadarEndpoint(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Tokens: ts.(store.TokenStore)})
	req := httptest.NewRequest(http.MethodGet, "/api/radar", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var pts []store.RadarPoint
	if err := json.NewDecoder(w.Body).Decode(&pts); err != nil {
		t.Fatal(err)
	} // boş fake → [] (nil değil)
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/api/ -run "TestKpis|TestRadar" -v`
Expected: FAIL.

- [ ] **Step 3: Implement handlers + route**

`overview.go`:
```go
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Kpi, frontend Kpi (types.ts) ile birebir JSON şeklidir.
type Kpi struct {
	ID      string    `json:"id"`
	Label   string    `json:"label"`
	Value   string    `json:"value"`
	Change  float64   `json:"change"`
	Spark   []float64 `json:"spark"`
	Updated string    `json:"updated"`
	Tone    string    `json:"tone,omitempty"`
}

func kpisHandler(ts store.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ts.Kpis(r.Context())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "kpis unavailable"})
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		empty := []float64{}
		kpis := []Kpi{
			{ID: "detected", Label: "Tespit Edilen Token (24s)", Value: strconv.Itoa(c.Detected), Spark: empty, Updated: now},
			{ID: "highconf", Label: "Yüksek Güvenli Token", Value: strconv.Itoa(c.HighConf), Spark: empty, Updated: now, Tone: "positive"},
			{ID: "critical", Label: "Kritik Risk Tespiti", Value: strconv.Itoa(c.Critical), Spark: empty, Updated: now, Tone: "critical"},
			{ID: "signals", Label: "Aktif Sinyaller", Value: strconv.Itoa(c.Signals), Spark: empty, Updated: now},
			{ID: "positions", Label: "Açık Pozisyonlar", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "realized", Label: "Gerçekleşen K/Z (24s)", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "unrealized", Label: "Gerçekleşmemiş K/Z", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "latency", Label: "Sistem Gecikmesi", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
		}
		writeJSON(w, http.StatusOK, kpis)
	}
}

func radarHandler(ts store.TokenStore, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pts, err := ts.Radar(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "radar unavailable"})
			return
		}
		if pts == nil {
			pts = []store.RadarPoint{}
		}
		writeJSON(w, http.StatusOK, pts)
	}
}
```
`router.go` — `if d.Tokens != nil` bloğuna ekle (tokens handler'ın yanına):
```go
		r.Get("/api/kpis", kpisHandler(d.Tokens))
		r.Get("/api/radar", radarHandler(d.Tokens, d.EventsWindow))
```

- [ ] **Step 4: config + main wiring**

`config.go` — Config struct + Load (manipulation deseni):
```go
	OpportunityEnabled     bool
	OpportunityIntervalSec int
	OpportunityLimit       int
```
```go
		OpportunityEnabled:     getenvBool("OPPORTUNITY_ENABLED", true),
		OpportunityIntervalSec: getenvInt("OPPORTUNITY_INTERVAL_SEC", 60),
		OpportunityLimit:       getenvInt("OPPORTUNITY_LIMIT", 100),
```
`config_test.go` — default assert:
```go
	if cfg.OpportunityIntervalSec != 60 || cfg.OpportunityLimit != 100 || !cfg.OpportunityEnabled {
		t.Fatalf("opportunity default yanlış: %+v", cfg)
	}
```
`main.go` — reputation worker bloğunun yanına (saf DB, RPC gerekmez):
```go
	// opportunity kompozit scorer (2d) — arka plan; saf DB (RPC YOK)
	if cfg.OpportunityEnabled && bundle.Tokens != nil {
		ow := opportunity.NewWorker(opportunity.WorkerDeps{
			Store:    bundle.Tokens,
			Interval: time.Duration(cfg.OpportunityIntervalSec) * time.Second,
			Limit:    cfg.OpportunityLimit, Logger: logger,
		})
		go ow.Run(ctx)
	}
```
> `opportunity` import'unu main.go'ya ekle. `bundle.Tokens` `store.TokenStore` → `OpportunityStore`'u karşılar.

- [ ] **Step 5: Run tests — PASS**

Run: `cd apps/api-go && go test ./internal/api/ ./internal/config/ -v && go build ./...`
Expected: PASS + build temiz.

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/api/overview.go apps/api-go/internal/api/router.go apps/api-go/internal/api/overview_test.go apps/api-go/internal/config/config.go apps/api-go/internal/config/config_test.go apps/api-go/cmd/server/main.go
git commit -m "feat(2d): /api/kpis + /api/radar handlers + config + main worker wiring"
```

---

### Task 8: Frontend gerçek fetch + LIVE_ENDPOINTS + README/followups

**Files:**
- Modify: `apps/web/lib/api/http.ts` (getKpis/getRadar gerçek fetch)
- Modify: `apps/web/lib/api/live-endpoints.ts` (+2)
- Modify: `apps/api-go/README.md` (OPPORTUNITY_* + /api/kpis + /api/radar)
- Modify: `docs/superpowers/followups-frontend.md` (2d ertelenenler)
- Test: `apps/web/lib/api/http.test.ts` + `index.test.ts` (mevcut desenler)

**Interfaces:**
- Consumes: backend `/api/kpis`, `/api/radar` (Task 7).

- [ ] **Step 1: Failing test** (`http.test.ts`'e ekle — mevcut getTokens fetch testi deseni)

```ts
it("getKpis gerçek API'den Kpi[] döndürür", async () => {
  const fake = [{ id: "detected", label: "x", value: "3", change: 0, spark: [], updated: "n" }];
  vi.spyOn(global, "fetch").mockResolvedValue(new Response(JSON.stringify(fake)));
  const r = await httpApi.getKpis();
  expect(r[0].id).toBe("detected");
});
it("getRadar gerçek API'den RadarPoint[] döndürür", async () => {
  const fake = [{ x: 1, y: 2, z: 3, name: "A", level: "good" }];
  vi.spyOn(global, "fetch").mockResolvedValue(new Response(JSON.stringify(fake)));
  const r = await httpApi.getRadar();
  expect(r[0].name).toBe("A");
});
```
`index.test.ts`'e: `expect(getApi().getKpis).toBe(httpApi.getKpis)` (LIVE olduğunda http'ye bağlanır — mevcut getTokens assert deseni).

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/web && npx vitest run lib/api/http.test.ts`
Expected: FAIL (getKpis hâlâ notReady).

- [ ] **Step 3: Implement**

`http.ts` — `getKpis: notReady` / `getRadar: notReady` yerine (mevcut `getTokens` fetch deseni birebir):
```ts
  getKpis: () => fetchJSON<Kpi[]>("/api/kpis"),
  getRadar: () => fetchJSON<RadarPoint[]>("/api/radar"),
```
> `Kpi`, `RadarPoint` type import'larını http.ts'e ekle (yoksa). `fetchJSON` yardımcısı mevcut (getTokens kullanıyor) — aynı base-URL/hata deseni.
`live-endpoints.ts` — Set'e ekle: `"getKpis", "getRadar"`.

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/web && npx vitest run lib/api/`
Expected: PASS (mevcut + yeni).

- [ ] **Step 5: README + followups**

`README.md` env tablosuna 3 satır (OPPORTUNITY_ENABLED/INTERVAL_SEC/LIMIT) + endpoint listesine `/api/kpis`, `/api/radar`.
`followups-frontend.md`'ye "Alt-proje 2 slice 2d" bölümü: trading/ops KPI (A5), KPI change/spark trend serisi, radar zaman-serisi, opportunity ağırlık/eşik config'e alma, opportunity girdi-kalitesi (safety conf 0.5 holders DAS'a kadar; creator seyrek WS'e kadar).

- [ ] **Step 6: Full suite + commit**

```bash
cd apps/api-go && go build ./... && go vet ./... && go test -race ./...
cd ../web && npx vitest run
```
Expected: tüm Go paketleri + frontend suite yeşil.
```bash
git add apps/web/lib/api/http.ts apps/web/lib/api/live-endpoints.ts apps/web/lib/api/http.test.ts apps/web/lib/api/index.test.ts apps/api-go/README.md docs/superpowers/followups-frontend.md
git commit -m "feat(2d): frontend getKpis/getRadar gerçek fetch + LIVE_ENDPOINTS + docs"
```

---

## Self-Review

**1. Spec coverage:**
- §3 Opportunity formülü → Task 2 (Scorer) ✓
- §4 Signal türetme → Task 2 (deriveSignal) ✓
- §5 getKpis (4 gerçek + 4 placeholder) → Task 4 (agrega) + Task 7 (handler placeholder) ✓
- §6 getRadar (projeksiyon + level parity) → Task 5 ✓
- §7.1 paket/worker → Task 2+3 ✓ · §7.2 migration 0011 → Task 1 ✓ · §7.3 store metotları → Task 1/4/5 ✓ · §7.4 detail/liste bağlama → Task 6 ✓ · §7.5 API → Task 7 ✓ · §7.6 config+main → Task 7 ✓ · §7.7 frontend → Task 8 ✓
- §8 error handling → Task 3 (worker izole) + Task 7 (502/[]) ✓
- §9 test stratejisi → her task TDD ✓

**2. Placeholder scan:** Task 6 Step 1 testi prose-iskelet (mevcut detail_test kurulumuna bağlı) — kasıtlı, çünkü fake TokenDetailProvider kurulumu mevcut dosyaya özgü; implementer mevcut `detail_test.go` desenini izler. Diğer tüm adımlar gerçek kod. Task 1 Step 5 + Task 4/5 fake impl prose-tarif (mevcut fake_ingest.go in-memory yapısına bağlı) — gerçek kod yerine semantik+parity kuralı verildi çünkü fake struct şekli o dosyada; postgres tarafı tam kod.

**3. Type consistency:** `OpportunityTarget`/`OpportunityUpdate`/`Inputs`/`Result`/`KpiCounts`/`RadarPoint`/`Kpi` isimleri task'lar arası tutarlı. `Score(Inputs) Result` (Task 2) → Worker (Task 3) aynı imza. `scoreToLevel` (Task 5) `float64→string`. `Kpi` struct (Task 7) frontend `Kpi` ile birebir alanlar.

---

## Execution Handoff

Plan tamam. İki uygulama seçeneği aşağıda.
