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

// Degree, COUNT(DISTINCT mint) OLMALI — ham (unpivot edilmiş) satır sayısı DEĞİL. F3 tam olarak
// maxDegree (3) distinct mint'e sahip, ama m1'de F3 hem mint HEM freeze authority olduğu için ham
// satır sayısı 4 (maxDegree+1). Eğer degree yanlışlıkla ham satır sayısıyla hesaplansaydı F3
// maxDegree=3 tavanını aşıp elenirdi; distinct-mint semantiğiyle F3 küme İÇİNDE kalmalı.
func TestAuthorityGraphClusters_DegreeCountsDistinctMintNotRawRows(t *testing.T) {
	f := NewFakeTokenStore().(*fakeTokenStore)
	ctx := context.Background()
	up := func(mint, sym, mintA, freezeA string) {
		f.UpsertDiscovered(ctx, DiscoveredToken{Mint: mint, Symbol: sym, PoolAddr: "p", FirstSeenTs: 1})
		f.UpdateSafety(ctx, SafetyUpdate{Mint: mint, AuthoritiesKnown: true, MintAuthority: mintA, FreezeAuthority: freezeA})
	}
	up("m1", "M1", "F3", "F3") // F3 hem mint hem freeze → 1 distinct mint, 2 ham satır
	up("m2", "M2", "F3", "")
	up("m3", "M3", "F3", "")
	rows, err := f.AuthorityGraphClusters(ctx, 3, 3) // distinct mint=3 → tam pencerede; ham satır=4 → yanlış hesapta elenirdi
	if err != nil {
		t.Fatal(err)
	}
	distinctMints := map[string]bool{}
	rawRows := 0
	for _, r := range rows {
		if r.Authority != "F3" {
			t.Fatalf("beklenmeyen authority satırı: %+v", r)
		}
		distinctMints[r.Mint] = true
		rawRows++
	}
	if len(distinctMints) != 3 {
		t.Fatalf("F3 distinct mint=3 olmalı (degree hesabı), got %d (%+v)", len(distinctMints), distinctMints)
	}
	if rawRows != 4 {
		t.Fatalf("F3 ham satır sayısı 4 olmalı (m1 mint+freeze ayrı, m2, m3), got %d", rawRows)
	}
}
