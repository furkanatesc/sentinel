package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestEventRowJSONKeys(t *testing.T) {
	e := EventRow{ID: "x", Type: "new_mint", Symbol: "S", Mint: "M", Launchpad: "Pump.fun",
		DEX: "", Liquidity: 0, CreatorScore: 0, RiskLevel: "medium", TokenAgeSeconds: 0,
		Volume5m: 0, HolderGrowthPct: 0, Severity: "info", Detail: "d", Time: "t", Ts: 1, Watchlisted: false}
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"id", "type", "symbol", "mint", "launchpad", "dex", "liquidity",
		"creatorScore", "riskLevel", "tokenAgeSeconds", "volume5m", "holderGrowthPct",
		"severity", "detail", "time", "ts", "watchlisted"} {
		if _, ok := m[k]; !ok {
			t.Errorf("EventRow JSON missing key %q (contract drift)", k)
		}
	}
}

func TestTokenRowJSONKeys(t *testing.T) {
	tk := TokenRow{ID: "M", Name: "n", Symbol: "s", Mint: "M", AgeSeconds: 0}
	b, _ := json.Marshal(tk)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"id", "name", "symbol", "mint", "ageSeconds", "price", "liquidity",
		"vol5m", "holders", "creatorScore", "safetyScore", "momentum", "spark", "signal", "watchlisted"} {
		if _, ok := m[k]; !ok {
			t.Errorf("TokenRow JSON missing key %q (contract drift)", k)
		}
	}
}

func TestFakeStoresRoundTrip(t *testing.T) {
	ctx := context.Background()
	es := NewFakeEventStore()
	if err := es.InsertEvent(ctx, EventRow{ID: "e1", Type: "new_mint", Mint: "M", Ts: 1}); err != nil {
		t.Fatal(err)
	}
	got, err := es.RecentEvents(ctx, 10)
	if err != nil || len(got) != 1 || got[0].ID != "e1" {
		t.Fatalf("RecentEvents = %+v, err=%v", got, err)
	}
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "M", Mint: "M", Symbol: "S"})
	_ = ts.UpsertToken(ctx, TokenRow{ID: "M", Mint: "M", Symbol: "S2"}) // upsert = tek satır
	toks, _ := ts.RecentTokens(ctx, 10)
	if len(toks) != 1 || toks[0].Symbol != "S2" {
		t.Fatalf("RecentTokens = %+v", toks)
	}
}
