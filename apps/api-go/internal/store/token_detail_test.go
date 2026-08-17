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

// TestFakeTokenDetailBaseCreatorReputationFromCreators, 2b-2b: TokenDetailBase artık
// creator itibarını (fake: reputationByAddr) taşımalı; scoredsız creator → 0/0/boş
// (postgres COALESCE(...,0) parite).
func TestFakeTokenDetailBaseCreatorReputationFromCreators(t *testing.T) {
	ts := NewFakeTokenStore()
	f := ts.(*fakeTokenStore)
	ctx := context.Background()
	seedToken(t, f, "m1", "A", "active", 0, 100, 0)
	if err := f.UpsertReputation(ctx, CreatorReputation{
		Address: "A", Score: 72, Confidence: 1,
		Breakdown: []ScoreBreakdownItem{{Label: "Başarı oranı", Weight: 0, Detail: "ok"}},
	}); err != nil {
		t.Fatal(err)
	}
	b, ok, err := f.TokenDetailBase(ctx, "m1")
	if err != nil || !ok {
		t.Fatalf("bulunmalı: ok=%v err=%v", ok, err)
	}
	if b.CreatorRepScore != 72 || b.CreatorRepConfidence != 1 || len(b.CreatorRepBreakdown) != 1 {
		t.Fatalf("creator itibarı base'e taşınmalı: %+v", b)
	}

	// Skorlanmamış creator → nötr 0/0/boş (silme sonrası değil, hiç UpsertReputation çağrılmamış).
	seedToken(t, f, "m2", "B", "active", 0, 100, 0)
	b2, ok, err := f.TokenDetailBase(ctx, "m2")
	if err != nil || !ok {
		t.Fatalf("bulunmalı: ok=%v err=%v", ok, err)
	}
	if b2.CreatorRepScore != 0 || b2.CreatorRepConfidence != 0 || len(b2.CreatorRepBreakdown) != 0 {
		t.Fatalf("skorlanmamış creator nötr olmalı: %+v", b2)
	}
}

// TestFakeTokenDetailBaseManipulation, 2c: TokenDetailBase artık manipülasyon skoru
// (tokens.manipulation_*) + işlem-akışı metriklerini (txns_*, creator_holding_pct) taşımalı.
func TestFakeTokenDetailBaseManipulation(t *testing.T) {
	f := NewFakeTokenStore()
	f.UpsertDiscovered(context.Background(), DiscoveredToken{Mint: "m", PoolAddr: "p"})
	f.UpdateMarket(context.Background(), MarketUpdate{Mint: "m", Liquidity: 1000, Vol24h: 5000,
		TxnsBuys: 70, TxnsSells: 30, TxnsBuyers: 25})
	f.UpdateSafety(context.Background(), SafetyUpdate{Mint: "m", CreatorHoldingPct: 33, CreatorHoldingKnown: true})
	f.UpdateManipulation(context.Background(), ManipulationUpdate{Mint: "m", Score: 48, Confidence: 0.7,
		Breakdown: []ScoreBreakdownItem{{Label: "x", Weight: 48, Detail: "y"}}, ScoredTs: 99})
	b, ok, err := f.TokenDetailBase(context.Background(), "m")
	if err != nil || !ok {
		t.Fatalf("base: ok=%v err=%v", ok, err)
	}
	if b.ManipulationScore != 48 || b.ManipulationConfidence != 0.7 || b.ManipulationScoredTs != 99 {
		t.Fatalf("manipülasyon alanları yanlış: %+v", b)
	}
	if b.TxnsBuys != 70 || b.TxnsSells != 30 || b.TxnsBuyers != 25 || b.CreatorHoldingPct != 33 {
		t.Fatalf("işlem-akışı alanları yanlış: %+v", b)
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
