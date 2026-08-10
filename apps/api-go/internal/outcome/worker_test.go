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
