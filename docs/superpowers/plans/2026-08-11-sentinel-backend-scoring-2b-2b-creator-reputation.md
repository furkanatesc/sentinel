# Creator Reputation Score (2b-2b) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Creator itibar skorunu 2b-2a token outcome'larından saf-DB, kural-tabanlı hesaplayıp persist etmek; `reputationScore`/`riskLevel`/`CreatorMetrics`/`TokenRow.creatorScore`/`TokenDetail.scores.creatorReputation`/per-token `riskFlags` alanlarını gerçeğe döndürmek.

**Architecture:** Yeni `internal/reputation/` paketi — saf `Scorer` (SRP) + arka plan `Worker` (2a Safety deseni, **RPC YOK**). Worker periyodik olarak `tokens`'tan creator-başına outcome agrega çeker (`CreatorAggregates`), skorlar, yeni `creators` tablosuna persist eder (`UpsertReputation`). Okuma yolları (`Creators`/`CreatorDetail`/`RecentTokens`/`TokenDetail`) `creators` tablosundan LEFT JOIN ile okur (Option A: throttle-dayanıklı, canlı hesap yok).

**Tech Stack:** Go 1.24, chi, pgx/v5 (database/sql), goose migration, `log/slog`. Test: standart `testing`, fake↔postgres store parity.

## Global Constraints

- Go sürümü **1.24** (go.mod ile eşleşir; CI go 1.24).
- Frontend'e **SIFIR dokunuş** — seam sabit; `RiskLevel` = `critical|high|medium|good|strong` (frontend `lib/format.ts`; **"low" yasak**).
- `creatorReputation.higherIsBetter = true` (frontend `score-defs.ts`) → yüksek skor = iyi.
- Outcome değerleri (2b-2a): `active|graduated|dumped|rug|dead`.
- `ScoreBreakdownItem` = `{Label string, Weight float64, Detail string}` (`store/token_detail.go`).
- **Yeni RPC/key YOK** — worker yalnız DB. Config-gate `REPUTATION_ENABLED && Tokens != nil`.
- SOLID: Scorer saf/SRP; Provider+Store dar arayüzler (ISP+DIP); Worker mevcut safety/outcome deseni.
- Her task sonunda `go build ./... && go vet ./... && go test -race ./...` yeşil + commit.

---

### Task 1: Migration 0009 + store tipleri + CreatorAggregates + UpsertReputation

**Files:**
- Create: `apps/api-go/internal/store/migrations/0009_create_creators.sql`
- Create: `apps/api-go/internal/store/reputation.go`
- Modify: `apps/api-go/internal/store/tokens.go` (TokenStore arayüzüne 2 metot ekle)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake impl + reputation deposu)
- Test: `apps/api-go/internal/store/reputation_test.go`

**Interfaces:**
- Produces:
  - `store.CreatorAgg{ Address string; Total, Active, Rug, Dumped, Dead, Graduated int; AvgPeakMarketCap, AvgLifetimeHours float64 }`
  - `store.CreatorReputation{ Address string; Score, Confidence float64; RiskLevel string; Breakdown []ScoreBreakdownItem; TotalTokens, ActiveTokens, RuggedTokens, GraduatedTokens int; AvgPeakMarketCap, AvgLifetimeHours, SuccessRatePct float64; ScoredTs int64 }`
  - `TokenStore.CreatorAggregates(ctx, limit int) ([]CreatorAgg, error)`
  - `TokenStore.UpsertReputation(ctx, CreatorReputation) error`

- [ ] **Step 1: Migration dosyası**

Create `apps/api-go/internal/store/migrations/0009_create_creators.sql`:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS creators (
  address             TEXT PRIMARY KEY,
  reputation_score    DOUBLE PRECISION NOT NULL DEFAULT 0,
  confidence          DOUBLE PRECISION NOT NULL DEFAULT 0,
  risk_level          TEXT NOT NULL DEFAULT 'medium',
  breakdown           TEXT NOT NULL DEFAULT '',
  total_tokens        INTEGER NOT NULL DEFAULT 0,
  active_tokens       INTEGER NOT NULL DEFAULT 0,
  rugged_tokens       INTEGER NOT NULL DEFAULT 0,
  graduated_tokens    INTEGER NOT NULL DEFAULT 0,
  avg_peak_market_cap DOUBLE PRECISION NOT NULL DEFAULT 0,
  avg_lifetime_hours  DOUBLE PRECISION NOT NULL DEFAULT 0,
  success_rate_pct    DOUBLE PRECISION NOT NULL DEFAULT 0,
  scored_ts           BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS creators;
```

- [ ] **Step 2: store tipleri + postgres impl**

Create `apps/api-go/internal/store/reputation.go`:
```go
package store

import "context"

// CreatorAgg, bir creator'ın tokenlarının outcome agregasıdır (2b-2b scorer girdisi).
type CreatorAgg struct {
	Address                                     string
	Total, Active, Rug, Dumped, Dead, Graduated int
	AvgPeakMarketCap, AvgLifetimeHours          float64
}

// CreatorReputation, hesaplanmış itibar + metriklerdir (creators tablosuna persist).
type CreatorReputation struct {
	Address                                                 string
	Score, Confidence                                       float64
	RiskLevel                                               string
	Breakdown                                               []ScoreBreakdownItem
	TotalTokens, ActiveTokens, RuggedTokens, GraduatedTokens int
	AvgPeakMarketCap, AvgLifetimeHours, SuccessRatePct      float64
	ScoredTs                                                int64
}

// CreatorAggregates, creator-başına outcome agrega döndürür; skorlanmamış creator'lar
// önce (round-robin), sonra en-eski skorlanan. active=çözülmemiş (scorer paydaya katmaz).
func (p *postgresStore) CreatorAggregates(ctx context.Context, limit int) ([]CreatorAgg, error) {
	const q = `SELECT t.creator, COUNT(*),
		SUM(CASE WHEN t.outcome='active'    THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='rug'       THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='dumped'    THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='dead'      THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='graduated' THEN 1 ELSE 0 END),
		COALESCE(AVG(NULLIF(t.peak_market_cap,0)),0),
		COALESCE(AVG(CASE WHEN t.outcome<>'active' AND t.outcome_scored_ts>0
			THEN (t.outcome_scored_ts - t.first_seen_ts)/3600.0 END),0)
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		WHERE t.creator <> ''
		GROUP BY t.creator, c.scored_ts
		ORDER BY c.scored_ts ASC NULLS FIRST
		LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorAgg, 0, limit)
	for rows.Next() {
		var a CreatorAgg
		if err := rows.Scan(&a.Address, &a.Total, &a.Active, &a.Rug, &a.Dumped, &a.Dead,
			&a.Graduated, &a.AvgPeakMarketCap, &a.AvgLifetimeHours); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpsertReputation, hesaplanmış itibarı creators tablosuna yazar (insert veya güncelle).
func (p *postgresStore) UpsertReputation(ctx context.Context, r CreatorReputation) error {
	breakdownJSON, err := marshalBreakdown(r.Breakdown)
	if err != nil {
		return err
	}
	const q = `INSERT INTO creators (address, reputation_score, confidence, risk_level, breakdown,
		total_tokens, active_tokens, rugged_tokens, graduated_tokens,
		avg_peak_market_cap, avg_lifetime_hours, success_rate_pct, scored_ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (address) DO UPDATE SET
			reputation_score=EXCLUDED.reputation_score, confidence=EXCLUDED.confidence,
			risk_level=EXCLUDED.risk_level, breakdown=EXCLUDED.breakdown,
			total_tokens=EXCLUDED.total_tokens, active_tokens=EXCLUDED.active_tokens,
			rugged_tokens=EXCLUDED.rugged_tokens, graduated_tokens=EXCLUDED.graduated_tokens,
			avg_peak_market_cap=EXCLUDED.avg_peak_market_cap, avg_lifetime_hours=EXCLUDED.avg_lifetime_hours,
			success_rate_pct=EXCLUDED.success_rate_pct, scored_ts=EXCLUDED.scored_ts`
	_, err = p.db.ExecContext(ctx, q, r.Address, r.Score, r.Confidence, r.RiskLevel, breakdownJSON,
		r.TotalTokens, r.ActiveTokens, r.RuggedTokens, r.GraduatedTokens,
		r.AvgPeakMarketCap, r.AvgLifetimeHours, r.SuccessRatePct, r.ScoredTs)
	return err
}
```

Note: `marshalBreakdown` — token_detail persist'te breakdown JSON'a çevriliyorsa onu reuse et; yoksa `reputation.go`'ya ekle:
```go
import "encoding/json"

func marshalBreakdown(b []ScoreBreakdownItem) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	out, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```
(Eğer `UpdateSafety`'de aynı isimde bir helper zaten varsa — grep `marshalBreakdown` — onu kullan, yeniden tanımlama.)

- [ ] **Step 3: TokenStore arayüzüne ekle**

`apps/api-go/internal/store/tokens.go` — `TokenStore` interface'inde `SetCreatorBackfill` satırından sonra:
```go
	// 2b-2b: creator itibar agregası (outcome sayımları) / hesaplanmış itibarı persist eder.
	CreatorAggregates(ctx context.Context, limit int) ([]CreatorAgg, error)
	UpsertReputation(ctx context.Context, r CreatorReputation) error
```

- [ ] **Step 4: fake store impl + reputation deposu**

`apps/api-go/internal/store/fake_ingest.go` — `fakeTokenStore` struct'ına alan ekle:
```go
	reputationByAddr map[string]CreatorReputation
```
Struct'ın oluşturulduğu constructor'da (grep `fakeTokenStore{`) map'i init et: `reputationByAddr: map[string]CreatorReputation{}` (nil map'e yazma paniğini önle). Sonra metotlar:
```go
func (f *fakeTokenStore) CreatorAggregates(_ context.Context, limit int) ([]CreatorAgg, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	byAddr := map[string]*CreatorAgg{}
	var peakSum, lifeSum map[string]float64
	peakSum, lifeSum = map[string]float64{}, map[string]float64{}
	peakN, lifeN := map[string]int{}, map[string]int{}
	for _, id := range f.order {
		t := f.byID[id]
		if t.creator == "" {
			continue
		}
		a := byAddr[t.creator]
		if a == nil {
			a = &CreatorAgg{Address: t.creator}
			byAddr[t.creator] = a
		}
		a.Total++
		switch t.outcome {
		case "active":
			a.Active++
		case "rug":
			a.Rug++
		case "dumped":
			a.Dumped++
		case "dead":
			a.Dead++
		case "graduated":
			a.Graduated++
		}
		if t.row.PeakMarketCapForAgg() > 0 { // bkz not: fake token peak alanı
			peakSum[t.creator] += t.peakMarketCap
			peakN[t.creator]++
		}
		if t.outcome != "active" && t.outcomeScoredTs > 0 {
			lifeSum[t.creator] += float64(t.outcomeScoredTs-t.row.firstSeenForAgg()) / 3600.0
			lifeN[t.creator]++
		}
	}
	// skorlanmamış önce, sonra en-eski scored_ts (round-robin)
	addrs := make([]string, 0, len(byAddr))
	for addr := range byAddr {
		addrs = append(addrs, addr)
	}
	sort.SliceStable(addrs, func(i, j int) bool {
		return f.reputationByAddr[addrs[i]].ScoredTs < f.reputationByAddr[addrs[j]].ScoredTs
	})
	out := make([]CreatorAgg, 0, limit)
	for _, addr := range addrs {
		if len(out) >= limit {
			break
		}
		a := *byAddr[addr]
		if peakN[addr] > 0 {
			a.AvgPeakMarketCap = peakSum[addr] / float64(peakN[addr])
		}
		if lifeN[addr] > 0 {
			a.AvgLifetimeHours = lifeSum[addr] / float64(lifeN[addr])
		}
		out = append(out, a)
	}
	return out, nil
}

func (f *fakeTokenStore) UpsertReputation(_ context.Context, r CreatorReputation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reputationByAddr[r.Address] = r
	return nil
}
```
**NOT (fake token alanları):** Fake token struct'ının gerçek alan adlarını kullan (grep `type fakeToken` / `byID`): `peakMarketCap`, `outcomeScoredTs`, ve first_seen (muhtemelen `t.row` içinde değil, ayrı bir `firstSeenTs` alanı — grep ile teyit et). `PeakMarketCapForAgg()`/`firstSeenForAgg()` yer tutucudur — gerçek alanlarla değiştir (ör. doğrudan `t.peakMarketCap` ve `t.firstSeenTs`). first_seen fake'te yoksa Task 1'de fake token'a ekle; postgres `first_seen_ts` kolonundan okur.

- [ ] **Step 5: Failing test yaz**

Create `apps/api-go/internal/store/reputation_test.go` (fake store ile; DB-gated postgres testi yok — 1a/2a deseni):
```go
package store

import (
	"context"
	"testing"
)

func TestCreatorAggregatesGroupsByCreatorAndCountsOutcomes(t *testing.T) {
	f := newFakeTokenStore() // gerçek constructor adını kullan (grep)
	ctx := context.Background()
	// creator A: 1 rug + 1 graduated + 1 active; creator B: 1 dumped
	seedToken(t, f, "m1", "A", "rug", 69000, 100, 3700)      // peak, firstSeen, scoredTs (yardımcı seed)
	seedToken(t, f, "m2", "A", "graduated", 80000, 100, 3700)
	seedToken(t, f, "m3", "A", "active", 0, 100, 0)
	seedToken(t, f, "m4", "B", "dumped", 5000, 100, 3700)
	got, err := f.CreatorAggregates(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	byAddr := map[string]CreatorAgg{}
	for _, a := range got {
		byAddr[a.Address] = a
	}
	a := byAddr["A"]
	if a.Total != 3 || a.Rug != 1 || a.Graduated != 1 || a.Active != 1 {
		t.Fatalf("A agg yanlış: %+v", a)
	}
	if byAddr["B"].Dumped != 1 {
		t.Fatalf("B dumped=%d, want 1", byAddr["B"].Dumped)
	}
}

func TestUpsertReputationRoundTrips(t *testing.T) {
	f := newFakeTokenStore()
	ctx := context.Background()
	rep := CreatorReputation{Address: "A", Score: 60, Confidence: 1, RiskLevel: "medium", ScoredTs: 42}
	if err := f.UpsertReputation(ctx, rep); err != nil {
		t.Fatal(err)
	}
	if f.reputationByAddr["A"].Score != 60 {
		t.Fatalf("persist edilmedi: %+v", f.reputationByAddr["A"])
	}
}
```
**NOT:** `seedToken` yardımcı fonksiyonu fake store'a token ekler (mint/creator/outcome/peak/firstSeen/scoredTs). Fake store'da zaten benzer seed yardımcıları varsa (grep `func seed` / mevcut testlerde token ekleme deseni) onları kullan/uyarla; yoksa test dosyasında küçük bir helper yaz.

- [ ] **Step 6: Testleri çalıştır (kırmızı → yeşil)**

Run: `cd apps/api-go && go test ./internal/store/ -run 'TestCreatorAggregates|TestUpsertReputation' -v`
Expected: önce derleme hatası/FAIL (metotlar yok) → impl sonrası PASS.

- [ ] **Step 7: build/vet/test + commit**

```bash
cd apps/api-go && go build ./... && go vet ./... && go test -race ./internal/store/
git add apps/api-go/internal/store apps/api-go/internal/store/migrations/0009_create_creators.sql
git commit -m "feat(reputation): store — migration 0009 creators + CreatorAggregates + UpsertReputation (fake+postgres)"
```

---

### Task 2: Saf Scorer (reputation.Scorer)

**Files:**
- Create: `apps/api-go/internal/reputation/scorer.go`
- Test: `apps/api-go/internal/reputation/scorer_test.go`

**Interfaces:**
- Consumes: `store.CreatorAgg`, `store.ScoreBreakdownItem` (Task 1).
- Produces:
  - `reputation.Thresholds{ MinResolved int; WRug, WFail, WGrad float64 }`
  - `reputation.Reputation{ Score, Confidence, SuccessRatePct float64; RiskLevel string; Breakdown []store.ScoreBreakdownItem }`
  - `reputation.Score(agg store.CreatorAgg, th Thresholds) Reputation`

- [ ] **Step 1: Failing test yaz**

Create `apps/api-go/internal/reputation/scorer_test.go`:
```go
package reputation

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

var th = Thresholds{MinResolved: 5, WRug: 50, WFail: 20, WGrad: 40}

func TestScoreAllRugIsZero(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 5, Rug: 5}, th)
	if r.Score != 0 || r.RiskLevel != "critical" {
		t.Fatalf("hepsi-rug: score=%v risk=%s, want 0/critical", r.Score, r.RiskLevel)
	}
	if r.Confidence != 1 {
		t.Fatalf("confidence=%v, want 1 (5 çözülmüş)", r.Confidence)
	}
}

func TestScoreAllGraduatedIsNinety(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 5, Graduated: 5}, th)
	if r.Score != 90 || r.RiskLevel != "strong" {
		t.Fatalf("hepsi-graduated: score=%v risk=%s, want 90/strong", r.Score, r.RiskLevel)
	}
	if r.SuccessRatePct != 100 {
		t.Fatalf("successRatePct=%v, want 100", r.SuccessRatePct)
	}
}

func TestScoreDumpedDeadIsThirty(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 4, Dumped: 2, Dead: 2}, th)
	if r.Score != 30 || r.RiskLevel != "high" {
		t.Fatalf("dump/dead: score=%v risk=%s, want 30/high", r.Score, r.RiskLevel)
	}
}

func TestScoreUnresolvedIsNeutral(t *testing.T) {
	// 3 active (çözülmemiş) → resolved=0 → nötr (conf 0, risk medium)
	r := Score(store.CreatorAgg{Address: "A", Total: 3, Active: 3}, th)
	if r.Confidence != 0 || r.RiskLevel != "medium" || r.Score != 0 {
		t.Fatalf("nötr olmalı: %+v", r)
	}
	if len(r.Breakdown) != 0 {
		t.Fatalf("nötr breakdown boş olmalı: %+v", r.Breakdown)
	}
}

func TestScoreConfidenceScalesWithResolved(t *testing.T) {
	// 1 çözülmüş (K=5) → 0.2; active paydaya girmez
	r := Score(store.CreatorAgg{Address: "A", Total: 6, Rug: 1, Active: 5}, th)
	if r.Confidence != 0.2 {
		t.Fatalf("confidence=%v, want 0.2 (1/5)", r.Confidence)
	}
	// rugRate=1/1=1 → score 0
	if r.Score != 0 {
		t.Fatalf("score=%v, want 0 (resolved üzerinden rugRate=1)", r.Score)
	}
}

func TestScoreBreakdownPresentWhenResolved(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 5, Rug: 2, Graduated: 3}, th)
	if len(r.Breakdown) == 0 {
		t.Fatal("çözülmüşse breakdown dolu olmalı")
	}
}
```

- [ ] **Step 2: Test çalıştır (fail)**

Run: `cd apps/api-go && go test ./internal/reputation/ -v`
Expected: FAIL — paket/`Score` yok.

- [ ] **Step 3: Scorer implementasyonu**

Create `apps/api-go/internal/reputation/scorer.go`:
```go
package reputation

import (
	"fmt"
	"math"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Thresholds, skor ağırlıkları + güven eşiğidir (config'ten enjekte; deploy-tunable).
type Thresholds struct {
	MinResolved    int
	WRug, WFail, WGrad float64
}

// Reputation, saf skorlama sonucudur (metrikler agg'den worker'da taşınır).
type Reputation struct {
	Score, Confidence, SuccessRatePct float64
	RiskLevel                         string
	Breakdown                         []store.ScoreBreakdownItem
}

// Score, creator agrega → açıklanabilir itibar. active token'lar çözülmemiş sayılır
// (paydaya girmez). resolved==0 → nötr (conf 0, risk medium, boş breakdown).
func Score(agg store.CreatorAgg, th Thresholds) Reputation {
	resolved := agg.Rug + agg.Dumped + agg.Dead + agg.Graduated
	if resolved <= 0 {
		return Reputation{Score: 0, Confidence: 0, RiskLevel: "medium", Breakdown: []store.ScoreBreakdownItem{}}
	}
	rf := float64(resolved)
	rugRate := float64(agg.Rug) / rf
	failRate := float64(agg.Dumped+agg.Dead) / rf
	gradRate := float64(agg.Graduated) / rf

	rugPen := th.WRug * rugRate
	failPen := th.WFail * failRate
	gradRew := th.WGrad * gradRate
	score := clamp(50-rugPen-failPen+gradRew, 0, 100)

	conf := float64(resolved) / float64(th.MinResolved)
	if conf > 1 {
		conf = 1
	}
	breakdown := []store.ScoreBreakdownItem{
		{Label: "Taban", Weight: 50, Detail: "Nötr başlangıç"},
		{Label: "Rug oranı", Weight: -rugPen, Detail: fmt.Sprintf("%d/%d çözülmüş token rug", agg.Rug, resolved)},
		{Label: "Başarısız (dump/dead)", Weight: -failPen, Detail: fmt.Sprintf("%d/%d dump veya dead", agg.Dumped+agg.Dead, resolved)},
		{Label: "Graduated (başarı)", Weight: gradRew, Detail: fmt.Sprintf("%d/%d graduated", agg.Graduated, resolved)},
	}
	return Reputation{
		Score:          score,
		Confidence:     conf,
		SuccessRatePct: gradRate * 100,
		RiskLevel:      riskLevelFor(score),
		Breakdown:      breakdown,
	}
}

// riskLevelFor, frontend scoreToLevel bantlarının Go karşılığıdır (lib/format.ts).
func riskLevelFor(score float64) string {
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

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
```

- [ ] **Step 4: Test çalıştır (pass)**

Run: `cd apps/api-go && go test ./internal/reputation/ -v`
Expected: PASS (tüm senaryolar).

- [ ] **Step 5: Commit**

```bash
cd apps/api-go && go build ./... && go vet ./... && go test -race ./internal/reputation/
git add apps/api-go/internal/reputation/scorer.go apps/api-go/internal/reputation/scorer_test.go
git commit -m "feat(reputation): saf Scorer — resolved-tabanlı kural skoru + frontend riskLevel bantları"
```

---

### Task 3: Worker (reputation.Worker)

**Files:**
- Create: `apps/api-go/internal/reputation/worker.go`
- Test: `apps/api-go/internal/reputation/worker_test.go`

**Interfaces:**
- Consumes: `store.CreatorAgg`, `store.CreatorReputation` (Task 1), `Score`/`Thresholds`/`Reputation` (Task 2).
- Produces:
  - `reputation.ReputationStore` interface (`CreatorAggregates`, `UpsertReputation`) — `store.TokenStore` karşılar.
  - `reputation.WorkerDeps{ Store ReputationStore; Thresholds Thresholds; Interval time.Duration; Limit int; Now func() int64; Logger *slog.Logger }`
  - `reputation.NewWorker(WorkerDeps) *Worker`, `(*Worker).Run(ctx)`

- [ ] **Step 1: Failing test yaz**

Create `apps/api-go/internal/reputation/worker_test.go`:
```go
package reputation

import (
	"context"
	"errors"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeRepStore struct {
	aggs     []store.CreatorAgg
	aggErr   error
	upserts  []store.CreatorReputation
	failAddr string // bu adres upsert'te hata → izolasyon testi
}

func (f *fakeRepStore) CreatorAggregates(context.Context, int) ([]store.CreatorAgg, error) {
	return f.aggs, f.aggErr
}
func (f *fakeRepStore) UpsertReputation(_ context.Context, r store.CreatorReputation) error {
	if r.Address == f.failAddr {
		return errors.New("boom")
	}
	f.upserts = append(f.upserts, r)
	return nil
}

func TestWorkerScoresAndPersistsAll(t *testing.T) {
	fs := &fakeRepStore{aggs: []store.CreatorAgg{
		{Address: "A", Total: 5, Rug: 5},
		{Address: "B", Total: 5, Graduated: 5},
	}}
	w := NewWorker(WorkerDeps{Store: fs, Thresholds: th, Now: func() int64 { return 99 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fs.upserts) != 2 {
		t.Fatalf("upsert sayısı=%d, want 2", len(fs.upserts))
	}
	// metrikler agg'den taşınmalı + scoredTs=Now
	for _, u := range fs.upserts {
		if u.ScoredTs != 99 || u.TotalTokens != 5 {
			t.Fatalf("upsert alanları yanlış: %+v", u)
		}
	}
}

func TestWorkerIsolatesUpsertError(t *testing.T) {
	fs := &fakeRepStore{failAddr: "A", aggs: []store.CreatorAgg{
		{Address: "A", Total: 5, Rug: 5},
		{Address: "B", Total: 5, Graduated: 5},
	}}
	w := NewWorker(WorkerDeps{Store: fs, Thresholds: th, Now: func() int64 { return 1 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatalf("kısmi hata döngüyü kırmamalı: %v", err)
	}
	if len(fs.upserts) != 1 || fs.upserts[0].Address != "B" {
		t.Fatalf("B yine de persist edilmeli: %+v", fs.upserts)
	}
}

func TestWorkerReturnsAggError(t *testing.T) {
	fs := &fakeRepStore{aggErr: errors.New("db down")}
	w := NewWorker(WorkerDeps{Store: fs, Thresholds: th})
	if err := w.scoreOnce(context.Background()); err == nil {
		t.Fatal("agg hatası dönmeli")
	}
}
```

- [ ] **Step 2: Test çalıştır (fail)**

Run: `cd apps/api-go && go test ./internal/reputation/ -run TestWorker -v`
Expected: FAIL — Worker yok.

- [ ] **Step 3: Worker implementasyonu**

Create `apps/api-go/internal/reputation/worker.go` (safety Worker deseni birebir):
```go
package reputation

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// ReputationStore, Worker'ın kalıcılık + agrega bağımlılığıdır (DIP; store.TokenStore karşılar).
type ReputationStore interface {
	CreatorAggregates(ctx context.Context, limit int) ([]store.CreatorAgg, error)
	UpsertReputation(ctx context.Context, r store.CreatorReputation) error
}

type WorkerDeps struct {
	Store      ReputationStore
	Thresholds Thresholds
	Interval   time.Duration
	Limit      int
	Now        func() int64
	Logger     *slog.Logger
}

// Worker, periyodik olarak creator agregalarını skorlayıp persist eder (RPC YOK, saf DB).
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
		w.d.Logger.Warn("reputation", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("reputation", "err", err)
			}
		}
	}
}

// scoreOnce, bir döngü: agregaları çek → her birini skorla → persist (kısmi hata izole).
func (w *Worker) scoreOnce(ctx context.Context) error {
	aggs, err := w.d.Store.CreatorAggregates(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, agg := range aggs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r := Score(agg, w.d.Thresholds)
		if err := w.d.Store.UpsertReputation(ctx, store.CreatorReputation{
			Address: agg.Address, Score: r.Score, Confidence: r.Confidence, RiskLevel: r.RiskLevel,
			Breakdown: r.Breakdown, SuccessRatePct: r.SuccessRatePct,
			TotalTokens: agg.Total, ActiveTokens: agg.Active, RuggedTokens: agg.Rug, GraduatedTokens: agg.Graduated,
			AvgPeakMarketCap: agg.AvgPeakMarketCap, AvgLifetimeHours: agg.AvgLifetimeHours, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("upsert reputation", "address", agg.Address, "err", err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Test çalıştır (pass) + commit**

Run: `cd apps/api-go && go test -race ./internal/reputation/ -v`
Expected: PASS.
```bash
cd apps/api-go && go build ./... && go vet ./...
git add apps/api-go/internal/reputation/worker.go apps/api-go/internal/reputation/worker_test.go
git commit -m "feat(reputation): Worker — periyodik skorla+persist (safety deseni, RPC yok, kısmi-hata izole)"
```

---

### Task 4: Okuma yolları — Creators + CreatorDetail + per-token riskFlags

**Files:**
- Modify: `apps/api-go/internal/store/creators.go` (Creators/CreatorDetail LEFT JOIN + gerçek metrik; newHistoryItem riskFlags)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake Creators/CreatorDetail reputation okur)
- Test: `apps/api-go/internal/store/creators_test.go` (mevcut dosyaya ekle) veya `reputation_test.go`

**Interfaces:**
- Consumes: `creators` tablosu (Task 1), `reputationByAddr` fake deposu (Task 1).
- Produces: `Creators`/`CreatorDetail` artık gerçek reputationScore/riskLevel/metrics/riskFlags döndürür (imza değişmez).

- [ ] **Step 1: Failing test yaz**

`apps/api-go/internal/store/creators_test.go`'ya ekle (fake store):
```go
func TestCreatorDetailReadsRealReputation(t *testing.T) {
	f := newFakeTokenStore()
	ctx := context.Background()
	seedToken(t, f, "m1", "A", "rug", 69000, 100, 3700)
	seedToken(t, f, "m2", "A", "graduated", 80000, 100, 3700)
	_ = f.UpsertReputation(ctx, CreatorReputation{
		Address: "A", Score: 55, Confidence: 0.4, RiskLevel: "medium",
		TotalTokens: 2, RuggedTokens: 1, GraduatedTokens: 1, SuccessRatePct: 50, AvgPeakMarketCap: 74500,
	})
	prof, ok, err := f.CreatorDetail(ctx, "A")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if prof.Reputation.Value != 55 || prof.RiskLevel != "medium" {
		t.Fatalf("reputation okunmadı: %+v", prof.Reputation)
	}
	if prof.Metrics.RuggedTokens != 1 || prof.Metrics.SuccessRatePct != 50 {
		t.Fatalf("metrics yanlış: %+v", prof.Metrics)
	}
}

func TestCreatorDetailRiskFlagsFromOutcome(t *testing.T) {
	f := newFakeTokenStore()
	ctx := context.Background()
	seedTokenFull(t, f, "m1", "A", "rug", "removed", 95) // outcome=rug, liq=removed, drawdown=95
	prof, _, _ := f.CreatorDetail(ctx, "A")
	flags := prof.History[0].RiskFlags
	if !contains(flags, "Rug çekildi") || !contains(flags, "Likidite çekildi") {
		t.Fatalf("riskFlags eksik: %v", flags)
	}
}

func TestCreatorsListIncludesUnscored(t *testing.T) {
	f := newFakeTokenStore()
	ctx := context.Background()
	seedToken(t, f, "m1", "A", "active", 0, 100, 0) // creator A yakalandı ama skorlanmadı
	rows, err := f.Creators(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Address != "A" || rows[0].RiskLevel != "medium" {
		t.Fatalf("skorlanmamış creator nötr olarak listelenmeli: %+v", rows)
	}
}
```
(`contains`/`seedTokenFull` yardımcıları test dosyasında; mevcut helper varsa reuse.)

- [ ] **Step 2: Test çalıştır (fail)**

Run: `cd apps/api-go && go test ./internal/store/ -run TestCreatorDetailReadsRealReputation -v`
Expected: FAIL — CreatorDetail hâlâ nötr döndürüyor.

- [ ] **Step 3: postgres Creators/CreatorDetail LEFT JOIN**

`apps/api-go/internal/store/creators.go` — `Creators` sorgusunu değiştir (LEFT JOIN creators, gerçek alanlar):
```go
func (p *postgresStore) Creators(ctx context.Context, limit int) ([]CreatorRow, error) {
	const q = `SELECT t.creator, COUNT(*) AS total,
		COALESCE(c.reputation_score,0), COALESCE(c.risk_level,'medium'),
		COALESCE(c.active_tokens,0), COALESCE(c.rugged_tokens,0), COALESCE(c.success_rate_pct,0)
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		WHERE t.creator <> '' GROUP BY t.creator, c.reputation_score, c.risk_level,
			c.active_tokens, c.rugged_tokens, c.success_rate_pct
		ORDER BY total DESC, MIN(t.first_seen_ts) ASC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorRow, 0, limit)
	for rows.Next() {
		var c CreatorRow
		if err := rows.Scan(&c.Address, &c.TotalTokens, &c.ReputationScore, &c.RiskLevel,
			&c.ActiveTokens, &c.RuggedTokens, &c.SuccessRatePct); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
```
`CreatorDetail` — kimlik/history sorgusundan sonra creators satırını oku ve profile'a bas:
```go
	// creators satırını oku (yoksa nötr — skorlanmamış creator)
	var rep CreatorReputation
	var breakdownJSON string
	err = p.db.QueryRowContext(ctx,
		`SELECT reputation_score, confidence, risk_level, breakdown, active_tokens, rugged_tokens,
			graduated_tokens, avg_peak_market_cap, avg_lifetime_hours, success_rate_pct
		 FROM creators WHERE address=$1`, address).
		Scan(&rep.Score, &rep.Confidence, &rep.RiskLevel, &breakdownJSON, &rep.ActiveTokens,
			&rep.RuggedTokens, &rep.GraduatedTokens, &rep.AvgPeakMarketCap, &rep.AvgLifetimeHours, &rep.SuccessRatePct)
	scored := err == nil
	if err != nil && err != sql.ErrNoRows {
		return CreatorProfile{}, false, err
	}
	prof := buildCreatorProfile(address, firstSeen, total, history)
	if scored {
		prof.Reputation = ScoreDetail{Key: "creatorReputation", Value: rep.Score, Confidence: rep.Confidence,
			Breakdown: parseBreakdownJSON(breakdownJSON)}
		prof.RiskLevel = rep.RiskLevel
		prof.Metrics = CreatorMetrics{
			TotalTokens: total, ActiveTokens: rep.ActiveTokens, RuggedTokens: rep.RuggedTokens,
			AvgPeakMarketCap: rep.AvgPeakMarketCap, AvgLifetimeHours: rep.AvgLifetimeHours, SuccessRatePct: rep.SuccessRatePct,
		}
	}
	return prof, true, nil
```
(`sql` import'u varsa kullan; `parseBreakdownJSON` `tokens.go`'da mevcut — reuse.)

- [ ] **Step 4: newHistoryItem riskFlags türetme**

`apps/api-go/internal/store/creators.go` — `newHistoryItem` imzasına drawdown eşiği eklemeden, mevcut alanlardan türet (eşik sabit 80 ya da paket düzeyi `const reputationHighDrawdown = 80`; config'ten geçirmek istersen imzaya `highDrawdown float64` ekle — basitlik için sabit):
```go
func deriveRiskFlags(outcome, liquidityStatus string, maxDrawdown float64) []string {
	flags := []string{}
	switch outcome {
	case "rug":
		flags = append(flags, "Rug çekildi")
	case "dumped":
		flags = append(flags, "Dump edildi")
	case "dead":
		flags = append(flags, "Ölü (hacim yok)")
	}
	if liquidityStatus == "removed" {
		flags = append(flags, "Likidite çekildi")
	}
	if maxDrawdown >= 80 {
		flags = append(flags, fmt.Sprintf("Yüksek düşüş (%%%.0f)", maxDrawdown))
	}
	return flags
}
```
`newHistoryItem` içinde `RiskFlags: []string{}` yerine `RiskFlags: deriveRiskFlags(outcome, liquidityStatus, maxDrawdown)`. (`fmt` import ekle.)

- [ ] **Step 5: fake Creators/CreatorDetail reputation okur**

`fake_ingest.go` — fake `Creators`/`CreatorDetail` `reputationByAddr`'dan okusun (postgres davranışını yansıt): skorlanmışsa gerçek alanlar, yoksa nötr; `CreatorDetail` history'de `deriveRiskFlags` kullan. (Mevcut fake `Creators`/`CreatorDetail` implementasyonunu grep'le bul ve aynı şekilde güncelle.)

- [ ] **Step 6: Test çalıştır (pass) + commit**

Run: `cd apps/api-go && go test -race ./internal/store/`
Expected: PASS + mevcut creators testleri regresyonsuz.
```bash
cd apps/api-go && go build ./... && go vet ./...
git add apps/api-go/internal/store
git commit -m "feat(reputation): Creators/CreatorDetail gerçek itibar+metrik okuma + per-token riskFlags"
```

---

### Task 5: TokenRow.creatorScore + TokenDetail.creatorReputation

**Files:**
- Modify: `apps/api-go/internal/store/tokens.go` (RecentTokens LEFT JOIN; TokenDetailBase alanları)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake RecentTokens creatorScore; TokenDetailBase)
- Modify: `apps/api-go/internal/market/detail.go` (scores.creatorReputation base'den)
- Test: `apps/api-go/internal/store/tokens_test.go` + `apps/api-go/internal/market/detail_test.go`

**Interfaces:**
- Consumes: `creators` tablosu (Task 1), `TokenDetailBase` (mevcut).
- Produces: `TokenRow.CreatorScore` = creator itibarı; `TokenDetail.Scores["creatorReputation"]` gerçek.

- [ ] **Step 1: Failing test yaz**

`tokens_test.go`'ya ekle:
```go
func TestRecentTokensCreatorScoreFromCreators(t *testing.T) {
	f := newFakeTokenStore()
	ctx := context.Background()
	seedTokenWithCreator(t, f, "m1", "A") // token'ın creator'ı A
	_ = f.UpsertReputation(ctx, CreatorReputation{Address: "A", Score: 72})
	rows, _ := f.RecentTokens(ctx, 10)
	if rows[0].CreatorScore != 72 {
		t.Fatalf("creatorScore=%v, want 72 (creator itibarı)", rows[0].CreatorScore)
	}
}
```

- [ ] **Step 2: postgres RecentTokens LEFT JOIN**

`RecentTokens` sorgusunu değiştir — `creator_score` yerine LEFT JOIN creators:
```go
	const q = `SELECT t.mint, t.symbol, t.name, t.first_seen_ts, t.price, t.liquidity, t.vol5m, t.holders,
		COALESCE(c.reputation_score,0), t.safety_score, t.momentum, t.spark
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		ORDER BY t.first_seen_ts DESC LIMIT $1`
```
(Scan sırası aynı; `t.CreatorScore` artık creators'tan.)

- [ ] **Step 3: TokenDetailBase creator reputation alanları + detail**

`token_detail.go` — `TokenDetailBase`'e ekle:
```go
	// 2b-2b creator reputation (creators tablosundan; detail scores.creatorReputation'a).
	CreatorRepScore, CreatorRepConfidence float64
	CreatorRepBreakdown                   []ScoreBreakdownItem
```
`postgresStore.TokenDetailBase` sorgusunda `LEFT JOIN creators c ON c.address = tokens.creator` ekleyip `COALESCE(c.reputation_score,0)`, `COALESCE(c.confidence,0)`, `COALESCE(c.breakdown,'')` seç ve struct'a doldur (mevcut TokenDetailBase sorgusunu grep'le bul; safety alanları deseni birebir).
`market/detail.go` — `scores["creatorReputation"]` nötr yerine base'den:
```go
	Scores: map[string]store.ScoreDetail{
		...
		"creatorReputation": {Key: "creatorReputation", Value: base.CreatorRepScore,
			Confidence: base.CreatorRepConfidence, Breakdown: base.CreatorRepBreakdown},
		...
	}
```
(Mevcut nötr creatorReputation satırını grep'le bul — `detail.go`'da; yalnız o satırı değiştir.)

- [ ] **Step 4: fake güncelle + test (pass)**

`fake_ingest.go` fake `RecentTokens` + `TokenDetailBase` reputation okusun.
Run: `cd apps/api-go && go test -race ./internal/store/ ./internal/market/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd apps/api-go && go build ./... && go vet ./...
git add apps/api-go/internal/store apps/api-go/internal/market
git commit -m "feat(reputation): TokenRow.creatorScore + TokenDetail.creatorReputation creators'tan (LEFT JOIN)"
```

---

### Task 6: Config + main wiring + README

**Files:**
- Modify: `apps/api-go/internal/config/config.go` (REPUTATION_* alanları + Load)
- Modify: `apps/api-go/internal/config/config_test.go` (default testi)
- Modify: `apps/api-go/cmd/server/main.go` (worker goroutine)
- Modify: `apps/api-go/README.md` (env tablosu)

**Interfaces:**
- Consumes: `reputation.NewWorker`/`WorkerDeps`/`Thresholds` (Task 3), config değerleri.

- [ ] **Step 1: config alanları + Load**

`config.go` `Config` struct'ına ekle:
```go
	ReputationEnabled     bool
	ReputationIntervalSec int
	ReputationLimit       int
	ReputationMinResolved int
	ReputationWRug        float64
	ReputationWFail       float64
	ReputationWGrad       float64
	ReputationHighDrawdown float64
```
`Load()`'a ekle:
```go
		ReputationEnabled:     getenvBool("REPUTATION_ENABLED", true),
		ReputationIntervalSec: getenvInt("REPUTATION_INTERVAL_SEC", 60),
		ReputationLimit:       getenvInt("REPUTATION_LIMIT", 60),
		ReputationMinResolved: getenvInt("REPUTATION_MIN_RESOLVED", 5),
		ReputationWRug:        getenvFloat("REPUTATION_W_RUG", 50),
		ReputationWFail:       getenvFloat("REPUTATION_W_FAIL", 20),
		ReputationWGrad:       getenvFloat("REPUTATION_W_GRAD", 40),
		ReputationHighDrawdown: getenvFloat("REPUTATION_HIGH_DRAWDOWN", 80),
```

- [ ] **Step 2: config default testi**

`config_test.go`'ya:
```go
func TestLoadReputationDefaults(t *testing.T) {
	c := Load()
	if !c.ReputationEnabled || c.ReputationIntervalSec != 60 || c.ReputationLimit != 60 ||
		c.ReputationMinResolved != 5 || c.ReputationWRug != 50 || c.ReputationWFail != 20 ||
		c.ReputationWGrad != 40 || c.ReputationHighDrawdown != 80 {
		t.Fatalf("reputation defaults: %+v", c)
	}
}
```
Run: `cd apps/api-go && go test ./internal/config/ -run TestLoadReputationDefaults -v` → PASS.

- [ ] **Step 3: main.go worker wiring**

`cmd/server/main.go` — creatorfill worker bloğundan sonra (RPC gerekmez):
```go
	// creator reputation scorer (2b-2b) — arka plan; saf DB (RPC YOK)
	if cfg.ReputationEnabled && bundle.Tokens != nil {
		rw := reputation.NewWorker(reputation.WorkerDeps{
			Store: bundle.Tokens,
			Thresholds: reputation.Thresholds{
				MinResolved: cfg.ReputationMinResolved,
				WRug:        cfg.ReputationWRug, WFail: cfg.ReputationWFail, WGrad: cfg.ReputationWGrad,
			},
			Interval: time.Duration(cfg.ReputationIntervalSec) * time.Second, Limit: cfg.ReputationLimit, Logger: logger,
		})
		go rw.Run(ctx)
	}
```
`import` bloğuna `"github.com/furkanatesc/sentinel/apps/api-go/internal/reputation"` ekle.
(`REPUTATION_HIGH_DRAWDOWN` şimdilik `deriveRiskFlags`'te sabit 80; config alanı ileride bağlanabilir — Task 4 notu. İstersen `deriveRiskFlags`'i config'ten geçir; basitlik için sabit bırakıldı, alan yine de eklendi tutarlılık için.)

- [ ] **Step 4: README env tablosu**

`apps/api-go/README.md` — `CREATORFILL_BURST` satırından sonra:
```markdown
| `REPUTATION_ENABLED` | Hayır (default true) | Creator itibar scorer'ı (2b-2b). Saf DB, RPC gerektirmez |
| `REPUTATION_INTERVAL_SEC` | Hayır (default 60) | Skorlama döngüsü aralığı (saniye) |
| `REPUTATION_LIMIT` | Hayır (default 60) | Döngü başına skorlanan creator |
| `REPUTATION_MIN_RESOLVED` | Hayır (default 5) | Tam güven için gereken çözülmüş token (confidence K) |
| `REPUTATION_W_RUG` | Hayır (default 50) | Rug oranı ceza ağırlığı |
| `REPUTATION_W_FAIL` | Hayır (default 20) | Dump/dead oranı ceza ağırlığı |
| `REPUTATION_W_GRAD` | Hayır (default 40) | Graduated oranı ödül ağırlığı |
| `REPUTATION_HIGH_DRAWDOWN` | Hayır (default 80) | per-token "yüksek düşüş" bayrağı eşiği (%) |
```

- [ ] **Step 5: Tam doğrulama + commit**

```bash
cd apps/api-go && go build ./... && go vet ./... && go test -race ./...
git add apps/api-go/internal/config apps/api-go/cmd/server/main.go apps/api-go/README.md
git commit -m "feat(reputation): config REPUTATION_* + main worker wiring + README (2b-2b tamam)"
```

---

## Self-Review (plan yazarı tarafından)

**Spec coverage:** §3 formül → Task 2; §4.1 paket → Task 2+3; §4.2 migration → Task 1; §4.3 store metotları → Task 1 (agg/upsert) + Task 4 (Creators/CreatorDetail) + Task 5 (RecentTokens/TokenDetail); §4.4 riskFlags → Task 4; §4.5 config/wiring → Task 6; §5 error handling → Task 3 (worker) + mevcut handler deseni; §6 test → her task TDD. Tümü kapsandı.

**Placeholder tarama:** `PeakMarketCapForAgg()`/`firstSeenForAgg()`/`seedToken`/`seedTokenFull`/`seedTokenWithCreator`/`newFakeTokenStore`/`contains` — bunlar **fake store'un gerçek alan/yardımcı adlarına bağlı** yer tutuculardır; her biri "grep ile gerçek adı bul/uyarla" notuyla işaretli (fake store yapısı okunmadan kesin ad verilemez). Uygulayıcı Task 1'de fake store'u açıp gerçek alan adlarını (peakMarketCap/outcomeScoredTs/firstSeenTs/creator/outcome/launchpad) kullanır. Kod adımları gerçek Go içerir.

**Tip tutarlılığı:** `CreatorAgg`/`CreatorReputation` (Task 1) → `Score` girdisi (Task 2) → Worker upsert (Task 3) → okuma (Task 4/5) zincirinde alan adları tutarlı. `Reputation` struct (scorer) metrik taşımaz; metrikler agg'den worker'da `store.CreatorReputation`'a taşınır. `riskLevelFor` bantları frontend `scoreToLevel` ile birebir.

**Not:** Fake store yardımcıları (seed*) mevcut testlerde farklı biçimde olabilir; Task 1 uygulayıcısı fake store dosyasını okuyup mevcut seed desenini reuse etmeli (yeni helper yazmak yerine).
