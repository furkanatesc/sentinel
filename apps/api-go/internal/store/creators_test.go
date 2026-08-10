package store

import (
	"context"
	"testing"
)

func TestCreatorsAggregate(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	// AAA 2 token deploy etti, BBB 1.
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m2", Mint: "m2", Symbol: "S2"}, 90, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m3", Mint: "m3", Symbol: "S3"}, 80, "BBB")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m4", Mint: "m4", Symbol: "S4"}, 70, "") // creator boş → sayılmaz

	cs := ts.(CreatorStore)
	rows, err := cs.Creators(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("creator sayısı = %d, want 2 (boş creator hariç)", len(rows))
	}
	if rows[0].Address != "AAA" || rows[0].TotalTokens != 2 {
		t.Fatalf("row0 = %+v, want AAA/2 (en çok token önce)", rows[0])
	}
	if rows[0].RiskLevel != "medium" || rows[0].ReputationScore != 0 {
		t.Fatalf("nötr placeholder bozuk: %+v", rows[0])
	}
}

// TestCreatorsLimit, fake Creators'ın limit sınırlamasının postgres LIMIT $1
// semantiğiyle eşleştiğini kilitler: limit=0 → boş dilim (ALL değil), limit=1 →
// tam olarak en üstteki (en çok token'lı) creator.
func TestCreatorsLimit(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m2", Mint: "m2", Symbol: "S2"}, 90, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m3", Mint: "m3", Symbol: "S3"}, 80, "BBB")

	cs := ts.(CreatorStore)

	rows, err := cs.Creators(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("Creators(ctx, 0) = %+v, want boş dilim (postgres LIMIT $1=0 ile eşleşmeli)", rows)
	}

	rows, err = cs.Creators(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Address != "AAA" || rows[0].TotalTokens != 2 {
		t.Fatalf("Creators(ctx, 1) = %+v, want tek satır AAA/2", rows)
	}
}

func TestUpsertTokenCreatorMergeDoesNotClobber(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	// Önce gerçek creator, sonra boş creator ile (GeckoTerminal deseni) upsert → gerçek korunur.
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "REAL")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1b"}, 100, "")
	cs := ts.(CreatorStore)
	rows, _ := cs.Creators(ctx, 10)
	if len(rows) != 1 || rows[0].Address != "REAL" || rows[0].TotalTokens != 1 {
		t.Fatalf("boş creator gerçek olanı ezmemeli: %+v", rows)
	}
}
