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
