package walletgraph

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeFunderStore struct {
	targets []store.FunderTarget
	set     map[string]string
}

func (f *fakeFunderStore) FunderTargets(_ context.Context, _ int) ([]store.FunderTarget, error) {
	return f.targets, nil
}
func (f *fakeFunderStore) SetFunder(_ context.Context, wallet, funder string, _ int64) error {
	if f.set == nil {
		f.set = map[string]string{}
	}
	f.set[wallet] = funder
	return nil
}

type stubResolver struct{ m map[string]string; fail string }

func (s stubResolver) ResolveFunder(_ context.Context, w string) (string, bool, error) {
	if w == s.fail {
		return "", false, errors.New("rpc boom")
	}
	f, ok := s.m[w]
	return f, ok, nil
}

func TestWorker_ResolvesAndStamps_IsolatesError(t *testing.T) {
	fs := &fakeFunderStore{targets: []store.FunderTarget{{Wallet: "cA"}, {Wallet: "cB"}, {Wallet: "cErr"}}}
	res := stubResolver{m: map[string]string{"cA": "F1", "cB": ""}, fail: "cErr"}
	w := NewWorker(WorkerDeps{Store: fs, Resolver: res, Limit: 10,
		Now: func() int64 { return 1000 }, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err := w.resolveOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// cA → F1 (bulundu), cB → "" (not-found ama damgalanır), cErr → RPC hata → damgalanmaz.
	if fs.set["cA"] != "F1" {
		t.Fatalf("cA funder F1 bekleniyordu, got %q", fs.set["cA"])
	}
	if _, ok := fs.set["cB"]; !ok {
		t.Fatal("cB not-found olsa da damgalanmalı")
	}
	if _, ok := fs.set["cErr"]; ok {
		t.Fatal("cErr RPC hatası → damgalanmamalı (sonraki tick retry)")
	}
}
