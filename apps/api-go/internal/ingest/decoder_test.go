package ingest

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type stubDecoder struct{ pid, lp string }

func (s stubDecoder) ProgramID() string { return s.pid }
func (s stubDecoder) Launchpad() string { return s.lp }
func (s stubDecoder) Decode(_ context.Context, _ LogNotification, _ TxFetcher, _ MetadataFetcher) ([]Decoded, error) {
	return []Decoded{{Event: store.EventRow{Type: "new_mint"}}}, nil
}

func TestRegistryLookup(t *testing.T) {
	r := NewRegistry()
	r.Register(stubDecoder{pid: "P1", lp: "Pump.fun"})
	r.Register(stubDecoder{pid: "P2", lp: "Raydium"})

	ids := r.ProgramIDs()
	if len(ids) != 2 {
		t.Fatalf("ProgramIDs len=%d, want 2", len(ids))
	}
	d, ok := r.Decoder("P1")
	if !ok || d.Launchpad() != "Pump.fun" {
		t.Fatalf("Decoder(P1) = %v, %v", d, ok)
	}
	if _, ok := r.Decoder("nope"); ok {
		t.Fatal("Decoder(nope) beklenmedik şekilde bulundu")
	}
}
