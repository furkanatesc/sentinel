package store

import (
	"context"
	"testing"
	"time"
)

// seedScoredToken, opportunity target/update testlerinin ihtiyaç duyduğu alt-skorları fake
// store'un MEVCUT setter'larıyla ("m1" mint) hazırlar: safety 80/conf 1, manipulation 10/conf 1,
// momentum 60, liquidity 1000. Creator boş bırakılır → reputation nötr 0/0 (postgres COALESCE
// parite; skorlanmamış/creator'sız token semantiği).
func seedScoredToken(t *testing.T, ts TokenStore) {
	t.Helper()
	ctx := context.Background()
	if _, err := ts.UpsertDiscovered(ctx, DiscoveredToken{
		Mint: "m1", Symbol: "M1", Launchpad: "Pump.fun", PoolAddr: "p1", FirstSeenTs: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1"}, 1, ""); err != nil {
		t.Fatal(err)
	}
	if err := ts.UpdateSafety(ctx, SafetyUpdate{Mint: "m1", Score: 80, Confidence: 1, ScoredTs: 10}); err != nil {
		t.Fatal(err)
	}
	if err := ts.UpdateManipulation(ctx, ManipulationUpdate{Mint: "m1", Score: 10, Confidence: 1, ScoredTs: 10}); err != nil {
		t.Fatal(err)
	}
	if err := ts.UpdateMarket(ctx, MarketUpdate{Mint: "m1", Momentum: 60, Liquidity: 1000}); err != nil {
		t.Fatal(err)
	}
}

func TestOpportunityTargetsAndUpdate_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	// Bir token + skorları hazırla (fake setter'lar mevcut UpsertToken/UpdateSafety... ile).
	seedScoredToken(t, fs) // yardımcı: mint "m1", safety 80/conf1, manip 10/conf1, momentum 60, liq 1000
	ts := fs

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
		Signal:    "buy", ScoredTs: 123,
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

// seedKpiTokens, TestKpis_Counts_Fake'in ihtiyaç duyduğu 3 token'ı taze first_seen_ts (24s
// "detected" penceresi içinde) ile hazırlar: k-highconf (safety 80/conf 1 → highConf),
// k-critical (manipulation 80/conf 1 → critical), k-signal (last_signal=buy → signals).
func seedKpiTokens(t *testing.T, ts TokenStore) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Unix()

	seed := func(mint string) {
		if _, err := ts.UpsertDiscovered(ctx, DiscoveredToken{
			Mint: mint, Symbol: mint, Launchpad: "Pump.fun", PoolAddr: "p-" + mint, FirstSeenTs: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	seed("k-highconf")
	if err := ts.UpdateSafety(ctx, SafetyUpdate{Mint: "k-highconf", Score: 80, Confidence: 1, ScoredTs: now}); err != nil {
		t.Fatal(err)
	}

	seed("k-critical")
	if err := ts.UpdateManipulation(ctx, ManipulationUpdate{Mint: "k-critical", Score: 80, Confidence: 1, ScoredTs: now}); err != nil {
		t.Fatal(err)
	}

	seed("k-signal")
	if err := ts.UpdateOpportunity(ctx, OpportunityUpdate{
		Mint: "k-signal", Score: 50, Confidence: 0.5, Signal: "buy", ScoredTs: now,
	}); err != nil {
		t.Fatal(err)
	}
}

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
