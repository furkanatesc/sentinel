package market

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestEnricherUpdatesMarketAndAppendsSpark(t *testing.T) {
	ts := store.NewFakeTokenStore()
	ctx := context.Background()
	// önceden keşfedilmiş token (havuzlu, spark [1])
	ts.UpsertDiscovered(ctx, store.DiscoveredToken{Mint: "M1", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 1})
	ts.UpdateMarket(ctx, store.MarketUpdate{Mint: "M1", Price: 1, Spark: []float64{1}})

	fp := &fakeProvider{byAddr: []Pool{{PoolAddr: "P1", Mint: "M1", Price: 5, LiquidityUSD: 200, Vol5m: 30, PriceChangeH1: 40}}}
	bc := &capBC{}
	e := NewEnricher(EnricherDeps{Provider: fp, Tokens: ts, Broadcast: bc, Limit: 50})

	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	toks, _ := ts.RecentTokens(ctx, 10)
	if len(toks) != 1 {
		t.Fatalf("token sayısı=%d", len(toks))
	}
	got := toks[0]
	if got.Price != 5 || got.Liquidity != 200 || got.Vol5m != 30 || got.Momentum != 70 {
		t.Fatalf("piyasa güncellenmedi: %+v", got)
	}
	if len(got.Spark) != 2 || got.Spark[1] != 5 { // mevcut [1] + yeni fiyat 5
		t.Fatalf("spark append yanlış: %+v", got.Spark)
	}
	var tokensSnap int
	for _, topic := range bc.topics {
		if topic == "tokens" {
			tokensSnap++
		}
	}
	if tokensSnap == 0 {
		t.Fatal("tokens snapshot broadcast edilmedi")
	}
}

func TestEnricherNoTargetsNoBroadcast(t *testing.T) {
	ts := store.NewFakeTokenStore()
	fp := &fakeProvider{}
	bc := &capBC{}
	e := NewEnricher(EnricherDeps{Provider: fp, Tokens: ts, Broadcast: bc, Limit: 50})
	if err := e.tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(bc.topics) != 0 {
		t.Fatalf("hedef yokken broadcast olmamalı: %v", bc.topics)
	}
}

func TestChunk(t *testing.T) {
	in := []string{"a", "b", "c", "d", "e"}
	got := chunk(in, 2)
	if len(got) != 3 || len(got[0]) != 2 || len(got[2]) != 1 {
		t.Fatalf("chunk yanlış: %v", got)
	}
}
