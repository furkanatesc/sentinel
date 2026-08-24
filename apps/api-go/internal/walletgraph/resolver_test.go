package walletgraph

import (
	"context"
	"testing"
)

// fakeSigTx, funderSigTx'i taklit eder (ağsız).
type fakeSigTx struct {
	pages     [][]string        // newest-first sig sayfaları
	transfers map[string]string // sig → source (destination=wallet olan transfer'ın kaynağı; "" yoksa)
}

func (f *fakeSigTx) listSignatures(_ context.Context, _ string, before string, _ int) ([]string, error) {
	// before boşsa ilk sayfa; testte tek sayfa yeter.
	if before == "" && len(f.pages) > 0 {
		return f.pages[0], nil
	}
	return nil, nil
}
func (f *fakeSigTx) transferSource(_ context.Context, sig, _ string) (string, bool, error) {
	s, ok := f.transfers[sig]
	return s, ok && s != "", nil
}

func newResolverWith(rpc funderSigTx) *HeliusFunderResolver {
	return &HeliusFunderResolver{rpc: rpc, maxSigPages: 3, pageLimit: 1000}
}

func TestResolveFunder_OldestInboundTransfer(t *testing.T) {
	// sayfa newest-first: [sig_new, sig_old]; en eski = sig_old; onun kaynağı F1.
	r := newResolverWith(&fakeSigTx{
		pages:     [][]string{{"sig_new", "sig_old"}},
		transfers: map[string]string{"sig_old": "F1"},
	})
	funder, found, err := r.ResolveFunder(context.Background(), "creatorX")
	if err != nil || !found || funder != "F1" {
		t.Fatalf("funder=F1 found bekleniyordu, got %q found=%v err=%v", funder, found, err)
	}
}

func TestResolveFunder_NoTransfer_NotFound(t *testing.T) {
	r := newResolverWith(&fakeSigTx{
		pages:     [][]string{{"sig_old"}},
		transfers: map[string]string{}, // transfer yok
	})
	_, found, err := r.ResolveFunder(context.Background(), "creatorX")
	if err != nil || found {
		t.Fatalf("not-found bekleniyordu, found=%v err=%v", found, err)
	}
}
