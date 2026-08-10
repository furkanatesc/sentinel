package outcome

import (
	"context"
	"errors"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeOutcomeStore struct {
	targets  []store.OutcomeTarget
	updates  []store.OutcomeUpdate
	failMint string // if set, UpdateOutcome returns error for this mint
}

func (f *fakeOutcomeStore) OutcomeTargets(_ context.Context, _ int) ([]store.OutcomeTarget, error) {
	return f.targets, nil
}
func (f *fakeOutcomeStore) UpdateOutcome(_ context.Context, u store.OutcomeUpdate) error {
	if u.Mint == f.failMint {
		return errors.New("simulated update error")
	}
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
	if byMint["fresh"].ScoredTs != 1000 {
		t.Fatalf("fresh ScoredTs = %d, want 1000 (Now())", byMint["fresh"].ScoredTs)
	}
	if byMint["fresh"].LiquidityStatus != LiquidityUnlocked {
		t.Fatalf("fresh LiquidityStatus = %s, want %s", byMint["fresh"].LiquidityStatus, LiquidityUnlocked)
	}
}

func TestWorkerPartialErrorIsolation(t *testing.T) {
	fs := &fakeOutcomeStore{
		targets: []store.OutcomeTarget{
			{Mint: "fails", CurMarketCap: 1000, CurLiquidity: 10, PeakMarketCap: 50000, PeakLiquidity: 20000, Vol24h: 100, FirstSeenTs: 0},
			{Mint: "succeeds", CurMarketCap: 3000, CurLiquidity: 4000, PeakMarketCap: 3200, PeakLiquidity: 4000, Vol24h: 6000, FirstSeenTs: 500},
		},
		failMint: "fails", // this target's UpdateOutcome will fail
	}
	w := NewWorker(WorkerDeps{
		Store:      fs,
		Thresholds: Thresholds{RugLiqRatio: 0.10, GraduationMcap: 69000, DumpedDrawdown: 80, DeadVol: 100, MinLiqFloor: 500, DeadAgeSec: 86400},
		Now:        func() int64 { return 1000 },
	})
	if err := w.classifyOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Only "succeeds" should be in updates; "fails" is not appended due to error
	if len(fs.updates) != 1 {
		t.Fatalf("update count = %d, want 1 (fails mint should not be persisted)", len(fs.updates))
	}
	if fs.updates[0].Mint != "succeeds" {
		t.Fatalf("update mint = %s, want 'succeeds'", fs.updates[0].Mint)
	}
	if fs.updates[0].Outcome != OutcomeActive {
		t.Fatalf("succeeds outcome = %s, want %s", fs.updates[0].Outcome, OutcomeActive)
	}
}

func TestWorkerContextCancellation(t *testing.T) {
	fs := &fakeOutcomeStore{targets: []store.OutcomeTarget{
		{Mint: "token1", CurMarketCap: 1000, CurLiquidity: 1000, PeakMarketCap: 2000, PeakLiquidity: 1000, Vol24h: 100, FirstSeenTs: 0},
		{Mint: "token2", CurMarketCap: 1000, CurLiquidity: 1000, PeakMarketCap: 2000, PeakLiquidity: 1000, Vol24h: 100, FirstSeenTs: 0},
	}}
	w := NewWorker(WorkerDeps{
		Store:      fs,
		Thresholds: Thresholds{RugLiqRatio: 0.10, GraduationMcap: 69000, DumpedDrawdown: 80, DeadVol: 100, MinLiqFloor: 500, DeadAgeSec: 86400},
		Now:        func() int64 { return 1000 },
	})
	// pre-canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := w.classifyOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	// classifyOnce should have short-circuited mid-loop, producing zero updates
	if len(fs.updates) != 0 {
		t.Fatalf("update count = %d, want 0 (should short-circuit on canceled context)", len(fs.updates))
	}
}
