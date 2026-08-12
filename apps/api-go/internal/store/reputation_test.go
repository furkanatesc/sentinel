package store

import (
	"context"
	"testing"
)

// seedToken, fake store'a outcome-agrega testleri için minimal bir token ekler:
// mint/creator/outcome/peak-market-cap/first-seen/outcome-scored-ts.
func seedToken(t *testing.T, ts TokenStore, mint, creator, outcome string, peak float64, firstSeen, scoredTs int64) {
	t.Helper()
	ctx := context.Background()
	if _, err := ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: mint, FirstSeenTs: firstSeen}); err != nil {
		t.Fatal(err)
	}
	if err := ts.UpsertToken(ctx, TokenRow{ID: mint, Mint: mint}, firstSeen, creator); err != nil {
		t.Fatal(err)
	}
	if peak > 0 {
		if err := ts.UpdateMarket(ctx, MarketUpdate{Mint: mint, MarketCapUSD: peak}); err != nil {
			t.Fatal(err)
		}
	}
	if outcome != "active" {
		if err := ts.UpdateOutcome(ctx, OutcomeUpdate{Mint: mint, Outcome: outcome, ScoredTs: scoredTs}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCreatorAggregatesGroupsByCreatorAndCountsOutcomes(t *testing.T) {
	ts := NewFakeTokenStore()
	f := ts.(*fakeTokenStore)
	ctx := context.Background()
	// creator A: 1 rug + 1 graduated + 1 active; creator B: 1 dumped
	seedToken(t, f, "m1", "A", "rug", 69000, 100, 3700)
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
	ts := NewFakeTokenStore()
	f := ts.(*fakeTokenStore)
	ctx := context.Background()
	rep := CreatorReputation{Address: "A", Score: 60, Confidence: 1, RiskLevel: "medium", ScoredTs: 42}
	if err := f.UpsertReputation(ctx, rep); err != nil {
		t.Fatal(err)
	}
	if f.reputationByAddr["A"].Score != 60 {
		t.Fatalf("persist edilmedi: %+v", f.reputationByAddr["A"])
	}
}
