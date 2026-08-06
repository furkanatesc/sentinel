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
