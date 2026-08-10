package creatorfill

import (
	"context"
	"errors"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeStore struct {
	targets []store.CreatorFillTarget
	sets    []struct {
		mint, creator string
		ts            int64
	}
}

func (f *fakeStore) CreatorFillTargets(_ context.Context, _ int) ([]store.CreatorFillTarget, error) {
	return f.targets, nil
}
func (f *fakeStore) SetCreatorBackfill(_ context.Context, mint, creator string, ts int64) error {
	f.sets = append(f.sets, struct {
		mint, creator string
		ts            int64
	}{mint, creator, ts})
	return nil
}

type fakeResolver struct {
	byMint map[string]string // mint → creator ("" = bulunamadı)
	fail   string
}

func (f *fakeResolver) ResolveCreator(_ context.Context, mint string) (string, bool, error) {
	if mint == f.fail {
		return "", false, errors.New("rpc down")
	}
	c := f.byMint[mint]
	return c, c != "", nil
}

func TestWorkerFillsAndStamps(t *testing.T) {
	fs := &fakeStore{targets: []store.CreatorFillTarget{{Mint: "a"}, {Mint: "b"}}}
	fr := &fakeResolver{byMint: map[string]string{"a": "CREATOR_A", "b": ""}} // b bulunamadı
	w := NewWorker(WorkerDeps{Store: fs, Resolver: fr, Now: func() int64 { return 100 }})
	if err := w.fillOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// İkisi de damgalanmalı (a creator ile, b boş ile → sonsuz retry yok).
	if len(fs.sets) != 2 {
		t.Fatalf("set sayısı = %d, want 2", len(fs.sets))
	}
	byMint := map[string]string{}
	for _, s := range fs.sets {
		byMint[s.mint] = s.creator
		if s.ts != 100 {
			t.Fatalf("%s ts = %d, want 100", s.mint, s.ts)
		}
	}
	if byMint["a"] != "CREATOR_A" || byMint["b"] != "" {
		t.Fatalf("sets = %+v", fs.sets)
	}
}

func TestWorkerResolverErrorIsolated(t *testing.T) {
	fs := &fakeStore{targets: []store.CreatorFillTarget{{Mint: "boom"}, {Mint: "ok"}}}
	fr := &fakeResolver{byMint: map[string]string{"ok": "C"}, fail: "boom"}
	w := NewWorker(WorkerDeps{Store: fs, Resolver: fr, Now: func() int64 { return 1 }})
	if err := w.fillOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// boom resolve hatası → atlanır (SetCreatorBackfill çağrılmaz); ok işlenir.
	if len(fs.sets) != 1 || fs.sets[0].mint != "ok" {
		t.Fatalf("sets = %+v, want yalnız ok", fs.sets)
	}
}
