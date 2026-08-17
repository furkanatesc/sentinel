package manipulation

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// ManipulationStore, Worker'ın kalıcılık + agrega bağımlılığıdır (DIP; store.TokenStore karşılar).
type ManipulationStore interface {
	ManipulationTargets(ctx context.Context, limit int) ([]store.ManipulationTarget, error)
	UpdateManipulation(ctx context.Context, u store.ManipulationUpdate) error
}

type WorkerDeps struct {
	Store      ManipulationStore
	Thresholds Thresholds
	Interval   time.Duration
	Limit      int
	Now        func() int64
	Logger     *slog.Logger
}

// Worker, periyodik olarak token'ları çekip manipülasyon skorunu hesaplayıp yazar (RPC YOK, saf DB).
type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Interval <= 0 {
		d.Interval = 60 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 60
	}
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("manipulation", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("manipulation", "err", err)
			}
		}
	}
}

// scoreOnce, bir döngü: hedefleri çek → her birini skorla → persist (kısmi hata izole).
func (w *Worker) scoreOnce(ctx context.Context) error {
	targets, err := w.d.Store.ManipulationTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r := Score(Inputs{
			Buys: tg.Buys, Sells: tg.Sells, Buyers: tg.Buyers,
			CreatorHoldingPct: tg.CreatorHoldingPct, Vol24h: tg.Vol24h, Liquidity: tg.Liquidity,
		}, w.d.Thresholds)
		if err := w.d.Store.UpdateManipulation(ctx, store.ManipulationUpdate{
			Mint: tg.Mint, Score: r.Value, Confidence: r.Confidence, Breakdown: r.Breakdown, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("update manipulation", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
