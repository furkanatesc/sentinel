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

type fakeTok struct {
	row       TokenRow
	poolAddr  string
	firstSeen int64
}

type fakeTokenStore struct {
	mu    sync.Mutex
	byID  map[string]fakeTok
	order []string // ekleme sırası
}

// NewFakeTokenStore, testler ve DB'siz mod için in-memory TokenStore döndürür.
func NewFakeTokenStore() TokenStore { return &fakeTokenStore{byID: map[string]fakeTok{}} }

func (f *fakeTokenStore) UpsertToken(_ context.Context, t TokenRow, firstSeenTs int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.Spark == nil {
		t.Spark = []float64{}
	}
	cur, ok := f.byID[t.ID]
	if !ok {
		f.order = append(f.order, t.ID)
	}
	cur.row = t
	cur.firstSeen = firstSeenTs
	f.byID[t.ID] = cur
	return nil
}

func (f *fakeTokenStore) UpsertDiscovered(_ context.Context, d DiscoveredToken) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[d.Mint]
	inserted := !ok
	if inserted {
		f.order = append(f.order, d.Mint)
		cur.row = TokenRow{ID: d.Mint, Mint: d.Mint, Spark: []float64{}}
		cur.firstSeen = d.FirstSeenTs
	}
	cur.row.Name, cur.row.Symbol = d.Name, d.Symbol
	cur.poolAddr = d.PoolAddr
	f.byID[d.Mint] = cur
	return inserted, nil
}

func (f *fakeTokenStore) UpdateMarket(_ context.Context, m MarketUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[m.Mint]
	if !ok {
		return nil
	}
	cur.row.Price, cur.row.Liquidity = m.Price, m.Liquidity
	cur.row.Vol5m, cur.row.Momentum = m.Vol5m, m.Momentum
	if m.Spark == nil {
		m.Spark = []float64{}
	}
	cur.row.Spark = m.Spark
	f.byID[m.Mint] = cur
	return nil
}

func (f *fakeTokenStore) EnrichTargets(_ context.Context, limit int) ([]EnrichTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]EnrichTarget, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		t := f.byID[f.order[i]]
		if t.poolAddr == "" {
			continue
		}
		out = append(out, EnrichTarget{Mint: t.row.Mint, PoolAddr: t.poolAddr, Spark: t.row.Spark})
	}
	return out, nil
}

func (f *fakeTokenStore) TokenDetailBase(_ context.Context, mint string) (TokenDetailBase, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[mint]
	if !ok {
		return TokenDetailBase{}, false, nil
	}
	return TokenDetailBase{Name: t.row.Name, Symbol: t.row.Symbol, PoolAddr: t.poolAddr, FirstSeenTs: t.firstSeen}, true, nil
}

func (f *fakeTokenStore) RecentTokens(_ context.Context, limit int) ([]TokenRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TokenRow, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, f.byID[f.order[i]].row)
	}
	return out, nil
}
