package store

import (
	"context"
	"testing"
)

// authority F1, ≥2 token'ın mint authority'si → küme; F2 tek token → dışarıda.
func TestAuthorityGraphClusters_DegreeThreshold(t *testing.T) {
	f := NewFakeTokenStore().(*fakeTokenStore)
	ctx := context.Background()
	up := func(mint, sym, mintA, freezeA string) {
		f.UpsertDiscovered(ctx, DiscoveredToken{Mint: mint, Symbol: sym, PoolAddr: "p", FirstSeenTs: 1})
		f.UpdateSafety(ctx, SafetyUpdate{Mint: mint, AuthoritiesKnown: true, MintAuthority: mintA, FreezeAuthority: freezeA})
	}
	up("mA", "A", "F1", "")   // F1 mint
	up("mB", "B", "F1", "F1") // F1 mint+freeze → mB'de rol "both"
	up("mC", "C", "F2", "")   // F2 tek token → küme değil
	rows, err := f.AuthorityGraphClusters(ctx, 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	byAuth := map[string]int{}
	mBMint, mBFreeze := false, false
	for _, r := range rows {
		byAuth[r.Authority]++
		if r.Authority == "F1" && r.Mint == "mB" && r.Role == "mint" {
			mBMint = true
		}
		if r.Authority == "F1" && r.Mint == "mB" && r.Role == "freeze" {
			mBFreeze = true
		}
	}
	if byAuth["F1"] == 0 || byAuth["F2"] != 0 {
		t.Fatalf("F1 küme olmalı, F2 dışarıda, got %+v", byAuth)
	}
	// Rol birleştirme ("both") bu katmanda YAPILMAZ (Task 5'te, BuildAuthorityGraph'ta) — store
	// katmanı mB için ham (F1,mB,mint) VE (F1,mB,freeze) satırlarını AYRI döndürür (postgres/fake parity).
	if !mBMint || !mBFreeze {
		t.Fatalf("mB'de F1 hem mint hem freeze rolüyle AYRI satır beklenir, got byAuth=%+v mBMint=%v mBFreeze=%v", byAuth, mBMint, mBFreeze)
	}
}

// maxDegree tavanı: çok-yüksek-dereceli authority (program-benzeri) elenmeli.
func TestAuthorityGraphClusters_MaxDegreeCeiling(t *testing.T) {
	f := NewFakeTokenStore().(*fakeTokenStore)
	ctx := context.Background()
	for i, m := range []string{"m1", "m2", "m3"} {
		f.UpsertDiscovered(ctx, DiscoveredToken{Mint: m, Symbol: m, PoolAddr: "p", FirstSeenTs: int64(i + 1)})
		f.UpdateSafety(ctx, SafetyUpdate{Mint: m, AuthoritiesKnown: true, MintAuthority: "PROG"})
	}
	rows, _ := f.AuthorityGraphClusters(ctx, 2, 2) // degree 3 > maxDegree 2 → elenir
	if len(rows) != 0 {
		t.Fatalf("maxDegree tavanı aşan authority elenmeli, got %d row", len(rows))
	}
}
