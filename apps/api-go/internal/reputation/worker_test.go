package reputation

import (
	"context"
	"errors"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeRepStore struct {
	aggs     []store.CreatorAgg
	aggErr   error
	upserts  []store.CreatorReputation
	failAddr string // bu adres upsert'te hata → izolasyon testi
}

func (f *fakeRepStore) CreatorAggregates(context.Context, int) ([]store.CreatorAgg, error) {
	return f.aggs, f.aggErr
}
func (f *fakeRepStore) UpsertReputation(_ context.Context, r store.CreatorReputation) error {
	if r.Address == f.failAddr {
		return errors.New("boom")
	}
	f.upserts = append(f.upserts, r)
	return nil
}

func TestWorkerScoresAndPersistsAll(t *testing.T) {
	fs := &fakeRepStore{aggs: []store.CreatorAgg{
		{Address: "A", Total: 5, Rug: 5},
		{Address: "B", Total: 5, Graduated: 5},
	}}
	w := NewWorker(WorkerDeps{Store: fs, Thresholds: th, Now: func() int64 { return 99 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fs.upserts) != 2 {
		t.Fatalf("upsert sayısı=%d, want 2", len(fs.upserts))
	}
	// metrikler agg'den taşınmalı + scoredTs=Now
	for _, u := range fs.upserts {
		if u.ScoredTs != 99 || u.TotalTokens != 5 {
			t.Fatalf("upsert alanları yanlış: %+v", u)
		}
	}
}

func TestWorkerIsolatesUpsertError(t *testing.T) {
	fs := &fakeRepStore{failAddr: "A", aggs: []store.CreatorAgg{
		{Address: "A", Total: 5, Rug: 5},
		{Address: "B", Total: 5, Graduated: 5},
	}}
	w := NewWorker(WorkerDeps{Store: fs, Thresholds: th, Now: func() int64 { return 1 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatalf("kısmi hata döngüyü kırmamalı: %v", err)
	}
	if len(fs.upserts) != 1 || fs.upserts[0].Address != "B" {
		t.Fatalf("B yine de persist edilmeli: %+v", fs.upserts)
	}
}

func TestWorkerReturnsAggError(t *testing.T) {
	fs := &fakeRepStore{aggErr: errors.New("db down")}
	w := NewWorker(WorkerDeps{Store: fs, Thresholds: th})
	if err := w.scoreOnce(context.Background()); err == nil {
		t.Fatal("agg hatası dönmeli")
	}
}
