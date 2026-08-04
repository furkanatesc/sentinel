package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSeedRows(t *testing.T) {
	rows := SeedRows()
	if len(rows) != 6 {
		t.Fatalf("SeedRows len = %d, want 6", len(rows))
	}
	if rows[0].ID != "momentum-scalp" || rows[0].Status != "live" {
		t.Fatalf("row[0] = %+v", rows[0])
	}
}

func TestStrategyRowJSONKeys(t *testing.T) {
	b, err := json.Marshal(SeedRows()[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, key := range []string{
		"id", "name", "status", "timeframe", "winRatePct", "profitFactor",
		"maxDrawdownPct", "totalTrades", "netPnlSol", "lastSignal",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q (contract drift)", key)
		}
	}
}

func TestFakeStoreList(t *testing.T) {
	st := NewFakeStore(SeedRows(), nil)
	rows, err := st.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("List len = %d, want 6", len(rows))
	}
}
