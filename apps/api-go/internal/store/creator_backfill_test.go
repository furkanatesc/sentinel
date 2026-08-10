package store

import (
	"context"
	"testing"
)

func TestCreatorFillTargets(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	// pump.fun + creator boş → hedef.
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "pf", Launchpad: "Pump.fun", FirstSeenTs: 100})
	// pump.fun ama creator dolu → hedef DEĞİL.
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "done", Launchpad: "Pump.fun", FirstSeenTs: 90})
	_ = ts.SetCreatorBackfill(ctx, "done", "REALCREATOR", 5)
	// non-pump.fun → hedef DEĞİL.
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "ray", Launchpad: "Raydium", FirstSeenTs: 80})

	tgs, err := ts.CreatorFillTargets(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tgs) != 1 || tgs[0].Mint != "pf" {
		t.Fatalf("hedefler = %+v, want yalnız pf", tgs)
	}
}

func TestSetCreatorBackfillMergeAndStamp(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "pf", Launchpad: "Pump.fun", FirstSeenTs: 100})
	// bulundu → creator set + damga.
	_ = ts.SetCreatorBackfill(ctx, "pf", "AAA", 7)
	// boş creator (bulunamadı) → gerçek'i EZMEZ, damga güncellenir.
	_ = ts.SetCreatorBackfill(ctx, "pf", "", 9)
	// doğrulama: damga boş-creator çağrısında da ilerlemeli (CRITICAL merge sözleşmesinin yarısı).
	fs := ts.(*fakeTokenStore)
	if fs.byID["pf"].creatorBackfillTs != 9 {
		t.Fatalf("creatorBackfillTs = %d, want 9 (boş creator çağrısı da damgalamalı)", fs.byID["pf"].creatorBackfillTs)
	}
	if fs.byID["pf"].creator != "AAA" {
		t.Fatalf("creator = %q, want AAA (boş ikinci çağrı gerçek creator'ı ezmemeli)", fs.byID["pf"].creator)
	}
	tgs, _ := ts.CreatorFillTargets(ctx, 10)
	if len(tgs) != 0 {
		t.Fatalf("pf artık creator'lı → hedef olmamalı: %+v", tgs)
	}
	// doğrulama: CreatorDetail üzerinden creator yansımalı (creators.go agrega).
	cs := ts.(CreatorStore)
	rows, _ := cs.Creators(ctx, 10)
	if len(rows) != 1 || rows[0].Address != "AAA" {
		t.Fatalf("creator merge bozuk: %+v", rows)
	}
}
