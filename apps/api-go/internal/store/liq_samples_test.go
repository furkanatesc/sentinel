package store

import (
	"context"
	"testing"
)

func TestFakeLiquiditySamples(t *testing.T) {
	f := NewFakeTokenStore().(TokenStore)
	ctx := context.Background()
	// iki token: A likidite 1000, B likidite 0 (filtrelenmeli)
	if _, err := f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "A", Symbol: "A", PoolAddr: "pA", FirstSeenTs: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "B", Symbol: "B", PoolAddr: "pB", FirstSeenTs: 1}); err != nil {
		t.Fatal(err)
	}
	if err := f.UpdateMarket(ctx, MarketUpdate{Mint: "A", Liquidity: 1000}); err != nil {
		t.Fatal(err)
	}
	// ts=100 ve ts=200'de örnek al (arada A likiditesi değişir)
	if err := f.InsertLiquiditySamples(ctx, 100, 10); err != nil {
		t.Fatal(err)
	}
	if err := f.UpdateMarket(ctx, MarketUpdate{Mint: "A", Liquidity: 1200}); err != nil {
		t.Fatal(err)
	}
	if err := f.InsertLiquiditySamples(ctx, 200, 10); err != nil {
		t.Fatal(err)
	}
	// A serisi kronolojik [1000@100, 1200@200]
	ser, err := f.LiquiditySeries(ctx, "A", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ser) != 2 || ser[0].T != 100 || ser[0].V != 1000 || ser[1].T != 200 || ser[1].V != 1200 {
		t.Fatalf("A serisi [1000@100,1200@200] beklenir: %+v", ser)
	}
	// B likidite 0 → örneklenmedi
	if s, _ := f.LiquiditySeries(ctx, "B", 10); len(s) != 0 {
		t.Fatalf("B örneklenmemeli: %+v", s)
	}
	// idempotent: aynı ts tekrar → değişmez
	if err := f.InsertLiquiditySamples(ctx, 200, 10); err != nil {
		t.Fatal(err)
	}
	if s, _ := f.LiquiditySeries(ctx, "A", 10); len(s) != 2 || s[1].V != 1200 {
		t.Fatalf("aynı ts idempotent olmalı (ilk değer korunur): %+v", s)
	}
	// prune cutoff=150 → ts=100 silinir, ts=200 kalır
	if err := f.PruneLiquiditySamples(ctx, 150); err != nil {
		t.Fatal(err)
	}
	if s, _ := f.LiquiditySeries(ctx, "A", 10); len(s) != 1 || s[0].T != 200 {
		t.Fatalf("prune sonrası A [200] beklenir: %+v", s)
	}
	// limit<=0 → boş (guard)
	if s, _ := f.LiquiditySeries(ctx, "A", 0); len(s) != 0 {
		t.Fatalf("limit<=0 → boş: %+v", s)
	}
}
