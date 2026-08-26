package store

import (
	"context"
	"testing"
)

// mustUpsertCreatorToken, fake'in UpsertToken'ıyla (creator arg dolu) token ekler.
func mustUpsertCreatorToken(t *testing.T, fs TokenStore, mint, creator string) {
	t.Helper()
	row := TokenRow{ID: mint, Mint: mint, Symbol: mint}
	if err := fs.UpsertToken(context.Background(), row, 1, creator); err != nil {
		t.Fatal(err)
	}
}

func TestFunderTargetsAndSetFunder_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	ts := fs.(TokenStore)
	// 2 creator'lı token'lar (creator dolu) — henüz funder çözülmemiş.
	mustUpsertCreatorToken(t, fs, "mintA", "creatorX")
	mustUpsertCreatorToken(t, fs, "mintB", "creatorY")

	targets, err := ts.FunderTargets(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("2 çözülmemiş creator bekleniyordu, got %d", len(targets))
	}
	// creatorX'i çöz (funder=F1) → artık target değil.
	if err := ts.SetFunder(ctx, "creatorX", "F1", 1000); err != nil {
		t.Fatal(err)
	}
	targets2, _ := ts.FunderTargets(ctx, 10)
	for _, tg := range targets2 {
		if tg.Wallet == "creatorX" {
			t.Fatal("creatorX çözüldü, target olmamalı")
		}
	}
	// not-found işareti de çözülmüş sayılır (funder="", resolved_ts>0).
	if err := ts.SetFunder(ctx, "creatorY", "", 1001); err != nil {
		t.Fatal(err)
	}
	targets3, _ := ts.FunderTargets(ctx, 10)
	if len(targets3) != 0 {
		t.Fatalf("hepsi çözüldü, 0 target bekleniyordu, got %d", len(targets3))
	}
}

func TestWalletGraphClusters_DegreeFilter_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	ts := fs.(TokenStore)
	// F1 → 2 creator (cA,cB) → küme (degree 2). F2 → 1 creator (cC) → küme değil (degree<2).
	mustUpsertCreatorToken(t, fs, "mA", "cA")
	mustUpsertCreatorToken(t, fs, "mB", "cB")
	mustUpsertCreatorToken(t, fs, "mC", "cC")
	_ = ts.SetFunder(ctx, "cA", "F1", 1000)
	_ = ts.SetFunder(ctx, "cB", "F1", 1000)
	_ = ts.SetFunder(ctx, "cC", "F2", 1000)

	rows, err := ts.WalletGraphClusters(ctx, 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	funders := map[string]bool{}
	for _, r := range rows {
		funders[r.Funder] = true
	}
	if !funders["F1"] {
		t.Fatal("F1 (degree 2) küme olmalı")
	}
	if funders["F2"] {
		t.Fatal("F2 (degree 1) küme olmamalı")
	}
}
