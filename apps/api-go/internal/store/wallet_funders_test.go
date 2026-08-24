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
