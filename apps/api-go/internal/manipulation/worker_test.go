package manipulation

import (
	"context"
	"errors"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type stubStore struct {
	targets []store.ManipulationTarget
	updated []store.ManipulationUpdate
	failOn  string // bu mint'te UpdateManipulation hata döner
}

func (s *stubStore) ManipulationTargets(_ context.Context, _ int) ([]store.ManipulationTarget, error) {
	return s.targets, nil
}
func (s *stubStore) UpdateManipulation(_ context.Context, u store.ManipulationUpdate) error {
	if u.Mint == s.failOn {
		return errors.New("boom")
	}
	s.updated = append(s.updated, u)
	return nil
}

func TestWorkerScoreOncePartialErrorIsolated(t *testing.T) {
	st := &stubStore{failOn: "bad", targets: []store.ManipulationTarget{
		{Mint: "bad", Buys: 100, Sells: 0, Buyers: 1, Liquidity: 1000},
		{Mint: "good", Buys: 100, Sells: 0, Buyers: 1, Liquidity: 1000},
	}}
	w := NewWorker(WorkerDeps{Store: st, Thresholds: defTh(), Now: func() int64 { return 7 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatalf("scoreOnce err: %v", err)
	}
	if len(st.updated) != 1 || st.updated[0].Mint != "good" {
		t.Fatalf("yalnız good yazılmalı (bad izole), gelen %+v", st.updated)
	}
	if st.updated[0].ScoredTs != 7 {
		t.Fatalf("ScoredTs Now()=7 beklenir, gelen %d", st.updated[0].ScoredTs)
	}
}

func TestWorkerScoreOnceCtxCancel(t *testing.T) {
	st := &stubStore{targets: []store.ManipulationTarget{{Mint: "a", Buys: 50, Sells: 50, Buyers: 5}}}
	w := NewWorker(WorkerDeps{Store: st, Thresholds: defTh()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.scoreOnce(ctx); err == nil {
		t.Fatalf("iptal edilmiş ctx'te hata beklenir")
	}
}
