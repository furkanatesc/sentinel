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

// seedTokenFull, seedToken'a ek olarak liquidityStatus/maxDrawdown taşır (2b-2b riskFlags
// testleri için — deriveRiskFlags outcome+liquidityStatus+maxDrawdown'dan türetir).
func seedTokenFull(t *testing.T, ts TokenStore, mint, creator, outcome, liquidityStatus string, maxDrawdown float64) {
	t.Helper()
	ctx := context.Background()
	if _, err := ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: mint, FirstSeenTs: 100}); err != nil {
		t.Fatal(err)
	}
	if err := ts.UpsertToken(ctx, TokenRow{ID: mint, Mint: mint}, 100, creator); err != nil {
		t.Fatal(err)
	}
	if err := ts.UpdateOutcome(ctx, OutcomeUpdate{
		Mint: mint, Outcome: outcome, LiquidityStatus: liquidityStatus, MaxDrawdownPct: maxDrawdown, ScoredTs: 200,
	}); err != nil {
		t.Fatal(err)
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

// TestCreatorAggregatesOrdersUnscoredFirstThenOldestScoredThenAddress, fake↔postgres
// arasında sürüklenme riski taşıyan sözleşmeyi kapsar: skorlanmamış (reputationByAddr'de
// yok → ScoredTs=0) creator'lar önce, sonra en-eski scored_ts, eşitlikte address ASC
// (round 1 review: tiebreak olmadan hem postgres `ORDER BY c.scored_ts ASC NULLS FIRST`
// hem fake'in map-iterasyon sıralı sort.SliceStable'ı deterministik değildi). Ayrıca
// AvgPeakMarketCap/AvgLifetimeHours aritmetiğini seedToken fixture'ıyla doğrular.
func TestCreatorAggregatesOrdersUnscoredFirstThenOldestScoredThenAddress(t *testing.T) {
	ts := NewFakeTokenStore()
	f := ts.(*fakeTokenStore)
	ctx := context.Background()

	// creator A (skorlanmamış): 2 rug — averages'i doğrulamak için kullanılan fixture.
	seedToken(t, f, "m1", "A", "rug", 1000, 0, 3600) // lifetime = 3600/3600 = 1h
	seedToken(t, f, "m2", "A", "rug", 3000, 0, 7200) // lifetime = 7200/3600 = 2h
	// creator Z (skorlanmamış, A ile eşit ScoredTs=0 — address tiebreak'ini test eder).
	seedToken(t, f, "m3", "Z", "dumped", 500, 0, 1800)
	// creator B (scored_ts=10 — skorlanmamışlardan sonra, M'den önce).
	seedToken(t, f, "m4", "B", "graduated", 200, 0, 900)
	if err := f.UpsertReputation(ctx, CreatorReputation{Address: "B", ScoredTs: 10}); err != nil {
		t.Fatal(err)
	}
	// creator M (scored_ts=50 — en son).
	seedToken(t, f, "m5", "M", "dead", 100, 0, 100)
	if err := f.UpsertReputation(ctx, CreatorReputation{Address: "M", ScoredTs: 50}); err != nil {
		t.Fatal(err)
	}

	got, err := f.CreatorAggregates(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("agrega sayısı = %d, want 4: %+v", len(got), got)
	}
	wantOrder := []string{"A", "Z", "B", "M"}
	for i, want := range wantOrder {
		if got[i].Address != want {
			t.Fatalf("sıra[%d] = %q, want %q (tam sıra: %+v)", i, got[i].Address, want, got)
		}
	}

	a := got[0]
	if a.AvgPeakMarketCap != 2000 {
		t.Fatalf("A.AvgPeakMarketCap = %v, want 2000", a.AvgPeakMarketCap)
	}
	if a.AvgLifetimeHours != 1.5 {
		t.Fatalf("A.AvgLifetimeHours = %v, want 1.5", a.AvgLifetimeHours)
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
