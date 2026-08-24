package opportunity

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// OpportunityStore, worker'ın bağımlı olduğu dar arayüzdür (ISP; store.TokenStore karşılar).
type OpportunityStore interface {
	OpportunityScoreTargets(ctx context.Context, limit int) ([]store.OpportunityTarget, error)
	UpdateOpportunity(ctx context.Context, u store.OpportunityUpdate) error
}

type WorkerDeps struct {
	Store    OpportunityStore
	Interval time.Duration
	Limit    int
	Logger   *slog.Logger
	Now      func() time.Time
}

type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Worker{d: d}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	for {
		if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
			w.d.Logger.Warn("opportunity cycle", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (w *Worker) scoreOnce(ctx context.Context) error {
	targets, err := w.d.Store.OpportunityScoreTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now().Unix()
	var scored int
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		res := Score(Inputs{
			Safety: tg.Safety, SafetyConf: tg.SafetyConf,
			Creator: tg.Creator, CreatorConf: tg.CreatorConf,
			Manipulation: tg.Manipulation, ManipulationConf: tg.ManipulationConf,
			Momentum: tg.Momentum, Liquidity: tg.Liquidity,
		})
		if err := w.d.Store.UpdateOpportunity(ctx, store.OpportunityUpdate{
			Mint: tg.Mint, Score: res.Value, Confidence: res.Confidence,
			Breakdown: res.Breakdown, Signal: res.Signal, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("update opportunity", "mint", tg.Mint, "err", err)
			continue
		}
		scored++
	}
	if len(targets) > 0 {
		w.d.Logger.Info("opportunity cycle", "targets", len(targets), "scored", scored)
	}
	return nil
}
