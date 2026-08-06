package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestFakeTokenDetailBase(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "M1", Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 42})
	b, ok, err := f.TokenDetailBase(ctx, "M1")
	if err != nil || !ok {
		t.Fatalf("bulunmalı: ok=%v err=%v", ok, err)
	}
	if b.Name != "One" || b.Symbol != "ONE" || b.PoolAddr != "P1" || b.FirstSeenTs != 42 {
		t.Fatalf("base yanlış: %+v", b)
	}
	if _, ok, _ := f.TokenDetailBase(ctx, "YOK"); ok {
		t.Fatal("bilinmeyen mint ok=false olmalı")
	}
}

func TestTokenDetailJSONTags(t *testing.T) {
	// Seam: camelCase alan adları frontend TokenDetail ile eşleşmeli.
	d := TokenDetail{Scores: map[string]ScoreDetail{}, Series: TokenDetailSeries{
		Price: []SeriesPoint{}, Liquidity: []SeriesPoint{}, Volume: []SeriesPoint{}, Holders: []SeriesPoint{}}}
	b, _ := json.Marshal(d)
	for _, key := range []string{`"priceChange24h"`, `"marketCap"`, `"volume24h"`, `"scores"`, `"metrics"`, `"series"`, `"risks"`, `"ageSeconds"`} {
		if !strings.Contains(string(b), key) {
			t.Fatalf("JSON'da %s yok: %s", key, b)
		}
	}
}
