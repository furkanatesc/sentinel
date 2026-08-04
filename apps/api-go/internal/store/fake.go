package store

import "context"

type fakeStore struct {
	rows []StrategyRow
	err  error
}

// NewFakeStore, testler için in-memory StrategyStore döndürür (err set edilirse List onu döner).
func NewFakeStore(rows []StrategyRow, err error) StrategyStore {
	return &fakeStore{rows: rows, err: err}
}

func (f *fakeStore) List(context.Context) ([]StrategyRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}
