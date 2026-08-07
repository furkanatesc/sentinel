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
	if err := b.Tokens.UpdateMarket(ctx, MarketUpdate{Mint: "MintMk", Price: 0.5, Liquidity: 1000, Vol5m: 50, Momentum: 72,
		Spark: []float64{1, 2, 3}, PriceChangeH24: -8.5, MarketCapUSD: 123456, Vol24h: 7777}); err != nil {
		t.Fatal(err)
	}
	// Detail header alanları DB'de round-trip etmeli (migration 0004 + TokenDetailBase okuması).
	base, ok, err := b.Tokens.TokenDetailBase(ctx, "MintMk")
	if err != nil || !ok {
		t.Fatalf("TokenDetailBase ok=%v err=%v", ok, err)
	}
	if base.Price != 0.5 || base.Liquidity != 1000 || base.PriceChangeH24 != -8.5 || base.MarketCapUSD != 123456 || base.Vol24h != 7777 {
		t.Fatalf("detail header DB round-trip yanlış: %+v", base)
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
	var foundTok bool
	for _, tk := range toks {
		if tk.Mint == "MintMk" {
			foundTok = true
			if tk.Price != 0.5 || tk.Momentum != 72 || len(tk.Spark) != 3 {
				t.Fatalf("RecentTokens piyasa alanlarını yansıtmadı: %+v", tk)
			}
		}
	}
	if !foundTok {
		t.Fatal("RecentTokens keşfedilen token'ı içermeli")
	}
	if err := b.Tokens.UpdateSafety(ctx, SafetyUpdate{
		Mint: "MintMk", Score: 65, Confidence: 0.5, Top10Pct: 55,
		Breakdown: []ScoreBreakdownItem{{Label: "Top-10 yoğunlaşma", Weight: -15, Detail: "orta"}},
		Risks:     RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{{ID: "top10-conc", Title: "x", Severity: "medium"}}, Creator: []RiskItem{}},
		ScoredTs:  200,
	}); err != nil {
		t.Fatal(err)
	}
	sb, ok, err := b.Tokens.TokenDetailBase(ctx, "MintMk")
	if err != nil || !ok || sb.SafetyScore != 65 || sb.Top10Pct != 55 || sb.SafetyScoredTs != 200 || len(sb.SafetyBreakdown) != 1 || len(sb.SafetyRisks.Market) != 1 {
		t.Fatalf("safety DB round-trip yanlış: ok=%v err=%v base=%+v", ok, err, sb)
	}
	tgts, err := b.Tokens.SafetyScoreTargets(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	var foundTgt bool
	for _, tg := range tgts {
		if tg.Mint == "MintMk" {
			foundTgt = true
		}
	}
	if !foundTgt {
		t.Fatal("SafetyScoreTargets pool'lu token'ı içermeli")
	}
}
