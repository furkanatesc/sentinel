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
