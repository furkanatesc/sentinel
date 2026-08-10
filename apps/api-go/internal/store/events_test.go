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
	if err := es.InsertEvent(ctx, EventRow{ID: "e2", Type: "new_mint", Mint: "M2", Ts: 2}); err != nil {
		t.Fatal(err)
	}
	got, err := es.RecentEvents(ctx, 10)
	if err != nil || len(got) != 2 || got[0].ID != "e2" || got[1].ID != "e1" {
		t.Fatalf("RecentEvents = %+v, err=%v, want newest-first [e2 e1]", got, err)
	}
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "M", Mint: "M", Symbol: "S"}, 0, "")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "M", Mint: "M", Symbol: "S2"}, 0, "") // upsert = tek satır
	_ = ts.UpsertToken(ctx, TokenRow{ID: "M3", Mint: "M3", Symbol: "S3"}, 0, "")
	toks, _ := ts.RecentTokens(ctx, 10)
	if len(toks) != 2 || toks[0].Symbol != "S3" || toks[1].Symbol != "S2" {
		t.Fatalf("RecentTokens = %+v, want newest-first [S3 S2]", toks)
	}
}

// TestFakeTokenStoreSparkNormalization, TokenStore'a nil Spark ile eklenen bir
// satırın JSON'da "spark":null değil "spark":[] olarak çıktığını kilitler
// (frontend kontratı: spark: number[], null değil).
func TestFakeTokenStoreSparkNormalization(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	if err := ts.UpsertToken(ctx, TokenRow{ID: "M", Mint: "M", Symbol: "S"}, 0, ""); err != nil { // Spark nil bırakıldı
		t.Fatal(err)
	}
	toks, err := ts.RecentTokens(ctx, 10)
	if err != nil || len(toks) != 1 {
		t.Fatalf("RecentTokens = %+v, err=%v", toks, err)
	}
	b, err := json.Marshal(toks[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	spark, ok := m["spark"].([]any)
	if !ok {
		t.Fatalf(`"spark" = %#v (%T), want a JSON array ([]), not null`, m["spark"], m["spark"])
	}
	if len(spark) != 0 {
		t.Fatalf("spark = %v, want empty array", spark)
	}
}

// TestEmptyResultsSerializeAsArrayNotNull, RecentEvents/RecentTokens sıfır satır
// döndürdüğünde JSON'da "null" değil "[]" çıktığını kilitler (plan Global
// Constraints: boş sonuçlar [] olarak serileşir). postgresStore aynı
// make([]T, 0, limit) deseniyle inşa edildiği için bu, o davranışın
// kontratını da güvenceye alır (DB'siz ortamda doğrudan postgres yolu
// egzersiz edilemez; postgres_ingest_test.go DATABASE_URL ile bunu kapsar).
func TestEmptyResultsSerializeAsArrayNotNull(t *testing.T) {
	ctx := context.Background()

	evs, err := NewFakeEventStore().RecentEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if evs == nil {
		t.Fatal("RecentEvents on empty store returned nil slice, want non-nil empty slice")
	}
	eb, err := json.Marshal(evs)
	if err != nil {
		t.Fatal(err)
	}
	if string(eb) != "[]" {
		t.Fatalf("RecentEvents empty JSON = %s, want []", eb)
	}

	toks, err := NewFakeTokenStore().RecentTokens(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if toks == nil {
		t.Fatal("RecentTokens on empty store returned nil slice, want non-nil empty slice")
	}
	tb, err := json.Marshal(toks)
	if err != nil {
		t.Fatal(err)
	}
	if string(tb) != "[]" {
		t.Fatalf("RecentTokens empty JSON = %s, want []", tb)
	}
}
