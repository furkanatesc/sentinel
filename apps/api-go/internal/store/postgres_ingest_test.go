package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPostgresEventsTokensRoundTrip(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL yok — integration testi atlanıyor")
	}
	ctx := context.Background()
	b, cleanup, err := OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	defer cleanup()

	tk := TokenRow{ID: "MintX", Mint: "MintX", Symbol: "TST", Name: "Test", AgeSeconds: 0, Spark: []float64{}}
	if err := b.Tokens.UpsertToken(ctx, tk, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	e := EventRow{ID: "sig1-new_mint", Signature: "sig1", Slot: 5, Type: "new_mint", Mint: "MintX",
		Symbol: "TST", Launchpad: "Pump.fun", RiskLevel: "medium", Severity: "info", Ts: 1}
	if err := b.Events.InsertEvent(ctx, e); err != nil {
		t.Fatal(err)
	}
	evs, err := b.Events.RecentEvents(ctx, 10)
	if err != nil || len(evs) == 0 || evs[0].Mint != "MintX" {
		t.Fatalf("RecentEvents=%+v err=%v", evs, err)
	}
	toks, err := b.Tokens.RecentTokens(ctx, 10)
	if err != nil || len(toks) == 0 {
		t.Fatalf("RecentTokens=%+v err=%v", toks, err)
	}
}

func TestPostgresMarketRoundTrip(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL yok — integration testi atlanıyor")
	}
	ctx := context.Background()
	b, cleanup, err := OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	defer cleanup()

	ins, err := b.Tokens.UpsertDiscovered(ctx, DiscoveredToken{
		Mint: "MintMk", Name: "MarketTok", Symbol: "MKT", Launchpad: "Pump.fun", PoolAddr: "PoolMk", FirstSeenTs: 10})
	if err != nil || !ins {
		t.Fatalf("ilk UpsertDiscovered inserted olmalı: inserted=%v err=%v", ins, err)
	}
	ins2, _ := b.Tokens.UpsertDiscovered(ctx, DiscoveredToken{Mint: "MintMk", Symbol: "MKT", PoolAddr: "PoolMk", FirstSeenTs: 10})
	if ins2 {
		t.Fatal("ikinci UpsertDiscovered inserted=false olmalı")
	}
	if err := b.Tokens.UpdateMarket(ctx, MarketUpdate{Mint: "MintMk", Price: 0.5, Liquidity: 1000, Vol5m: 50, Momentum: 72, Spark: []float64{1, 2, 3}}); err != nil {
		t.Fatal(err)
	}
	targets, err := b.Tokens.EnrichTargets(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tg := range targets {
		if tg.Mint == "MintMk" {
			found = true
			if tg.PoolAddr != "PoolMk" || len(tg.Spark) != 3 {
				t.Fatalf("EnrichTarget yanlış: %+v", tg)
			}
		}
	}
	if !found {
		t.Fatal("EnrichTargets keşfedilen token'ı içermeli")
	}
	toks, _ := b.Tokens.RecentTokens(ctx, 50)
	for _, tk := range toks {
		if tk.Mint == "MintMk" && (tk.Price != 0.5 || tk.Momentum != 72 || len(tk.Spark) != 3) {
			t.Fatalf("RecentTokens piyasa alanlarını yansıtmadı: %+v", tk)
		}
	}
}
