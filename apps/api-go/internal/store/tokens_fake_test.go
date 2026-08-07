package store

import (
	"context"
	"testing"
)

func TestFakeUpsertDiscoveredInsertedFlag(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	d := DiscoveredToken{Mint: "M1", Name: "One", Symbol: "ONE", Launchpad: "Pump.fun", PoolAddr: "P1", FirstSeenTs: 100}
	ins, err := f.UpsertDiscovered(ctx, d)
	if err != nil || !ins {
		t.Fatalf("ilk keşif inserted=true olmalı, got inserted=%v err=%v", ins, err)
	}
	ins2, _ := f.UpsertDiscovered(ctx, d)
	if ins2 {
		t.Fatal("ikinci keşif inserted=false olmalı (dedup)")
	}
}

func TestFakeUpdateMarketAndEnrichTargets(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "M1", PoolAddr: "P1", FirstSeenTs: 1})
	if err := f.UpdateMarket(ctx, MarketUpdate{Mint: "M1", Price: 2, Liquidity: 3, Vol5m: 4, Momentum: 60, Spark: []float64{1, 2}}); err != nil {
		t.Fatal(err)
	}
	toks, _ := f.RecentTokens(ctx, 10)
	if len(toks) != 1 || toks[0].Price != 2 || toks[0].Momentum != 60 || len(toks[0].Spark) != 2 {
		t.Fatalf("UpdateMarket RecentTokens'a yansımadı: %+v", toks)
	}
	targets, _ := f.EnrichTargets(ctx, 10)
	if len(targets) != 1 || targets[0].PoolAddr != "P1" || len(targets[0].Spark) != 2 {
		t.Fatalf("EnrichTargets beklenen hedefi vermedi: %+v", targets)
	}
}

func TestFakeEnrichTargetsSkipsNoPool(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	f.UpsertToken(ctx, TokenRow{ID: "M2", Mint: "M2"}, 1) // pool_address yok
	targets, _ := f.EnrichTargets(ctx, 10)
	if len(targets) != 0 {
		t.Fatalf("havuzsuz token enrichment hedefi olmamalı: %+v", targets)
	}
}

func TestFakeUpdateSafetyRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewFakeTokenStore()
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "MSAFE", Symbol: "SAF", Launchpad: "Pump.fun", PoolAddr: "P", FirstSeenTs: 1})
	err := s.UpdateSafety(ctx, SafetyUpdate{
		Mint: "MSAFE", Score: 72, Confidence: 1, Top10Pct: 44,
		Breakdown: []ScoreBreakdownItem{{Label: "Freeze authority iptal", Weight: 0, Detail: "ok"}},
		Risks:     RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{{ID: "top10-conc", Title: "x", Severity: "medium"}}, Creator: []RiskItem{}},
		ScoredTs:  100,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, ok, _ := s.TokenDetailBase(ctx, "MSAFE")
	if !ok || b.SafetyScore != 72 || b.SafetyConfidence != 1 || b.Top10Pct != 44 || b.SafetyScoredTs != 100 {
		t.Fatalf("safety round-trip yanlış: %+v", b)
	}
	if len(b.SafetyBreakdown) != 1 || len(b.SafetyRisks.Market) != 1 {
		t.Fatalf("breakdown/risks taşınmadı: %+v", b)
	}
}

func TestFakeSafetyScoreTargetsOldestFirstPoolOnly(t *testing.T) {
	ctx := context.Background()
	s := NewFakeTokenStore()
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "A", Symbol: "A", Launchpad: "Pump.fun", PoolAddr: "PA", FirstSeenTs: 1})
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "B", Symbol: "B", Launchpad: "Pump.fun", PoolAddr: "", FirstSeenTs: 2})  // pool'suz → hariç
	s.UpdateSafety(ctx, SafetyUpdate{Mint: "A", ScoredTs: 50})                                                             // A skorlandı
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "C", Symbol: "C", Launchpad: "Raydium", PoolAddr: "PC", FirstSeenTs: 3}) // C hiç skorlanmadı (ts 0)
	got, _ := s.SafetyScoreTargets(ctx, 10)
	if len(got) != 2 {
		t.Fatalf("pool'lu 2 hedef beklenir (B hariç): %+v", got)
	}
	if got[0].Mint != "C" {
		t.Fatalf("en eski (hiç skorlanmamış) önce gelmeli: %+v", got)
	}
	if got[0].Launchpad != "Raydium" {
		t.Fatalf("launchpad taşınmalı: %+v", got[0])
	}
}
