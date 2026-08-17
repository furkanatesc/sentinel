package safety

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeSafetyStore struct {
	targets []store.SafetyTarget
	updates []store.SafetyUpdate
}

func (f *fakeSafetyStore) SafetyScoreTargets(context.Context, int) ([]store.SafetyTarget, error) {
	return f.targets, nil
}
func (f *fakeSafetyStore) UpdateSafety(_ context.Context, s store.SafetyUpdate) error {
	f.updates = append(f.updates, s)
	return nil
}

type stubProvider struct{ d OnChainData }

func (s stubProvider) FetchOnChain(context.Context, string, string) (OnChainData, error) {
	return s.d, nil
}

func TestScoreOncePersistsResult(t *testing.T) {
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000, Launchpad: "Raydium"}}}
	prov := stubProvider{d: OnChainData{AuthoritiesKnown: true, HoldersKnown: true, HolderCount: 500, Top10Pct: 30}}
	w := NewWorker(WorkerDeps{Store: st, Provider: prov, Limit: 10, Now: func() int64 { return 999 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(st.updates) != 1 {
		t.Fatalf("1 update beklenir: %d", len(st.updates))
	}
	u := st.updates[0]
	if u.Mint != "M1" || u.Score != 100 || u.Confidence != 1 || u.ScoredTs != 999 {
		t.Fatalf("update yanlış: %+v", u)
	}
	if len(u.Breakdown) == 0 {
		t.Fatal("breakdown persist edilmeli")
	}
}

func TestScoreOnceLiquidityLaunchpadFromTarget(t *testing.T) {
	// Aktif mint authority + pump.fun → cezasız (launchpad target'tan gelmeli).
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000, Launchpad: "Pump.fun"}}}
	prov := stubProvider{d: OnChainData{AuthoritiesKnown: true, MintAuthorityActive: true, HoldersKnown: true, HolderCount: 500, Top10Pct: 30}}
	w := NewWorker(WorkerDeps{Store: st, Provider: prov, Limit: 10, Now: func() int64 { return 1 }})
	w.scoreOnce(context.Background())
	if st.updates[0].Score != 100 {
		t.Fatalf("pump.fun mint authority cezasız olmalı: %v", st.updates[0].Score)
	}
}
