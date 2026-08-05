package store

import (
	"context"
	"sync"
)

type fakeEventStore struct {
	mu   sync.Mutex
	rows []EventRow // en yeni sonda
}

// NewFakeEventStore, testler ve DB'siz mod için in-memory EventStore döndürür.
func NewFakeEventStore() EventStore { return &fakeEventStore{} }

func (f *fakeEventStore) InsertEvent(_ context.Context, e EventRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, e)
	return nil
}

func (f *fakeEventStore) RecentEvents(_ context.Context, limit int) ([]EventRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]EventRow, 0, limit)
	for i := len(f.rows) - 1; i >= 0 && len(out) < limit; i-- { // en yeni önce
		out = append(out, f.rows[i])
	}
	return out, nil
}

type fakeTokenStore struct {
	mu    sync.Mutex
	byID  map[string]TokenRow
	order []string // ekleme sırası
}

// NewFakeTokenStore, testler ve DB'siz mod için in-memory TokenStore döndürür.
func NewFakeTokenStore() TokenStore { return &fakeTokenStore{byID: map[string]TokenRow{}} }

func (f *fakeTokenStore) UpsertToken(_ context.Context, t TokenRow, _ int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.Spark == nil {
		t.Spark = []float64{}
	}
	if _, ok := f.byID[t.ID]; !ok {
		f.order = append(f.order, t.ID)
	}
	f.byID[t.ID] = t
	return nil
}

func (f *fakeTokenStore) RecentTokens(_ context.Context, limit int) ([]TokenRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TokenRow, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, f.byID[f.order[i]])
	}
	return out, nil
}
