package opportunity

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
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
	Health   health.Reporter
}

type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Interval <= 0 {
		d.Interval = 60 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 60
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	for {
		w.cycle(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// cycle, tek scoreOnce + health Report (best-effort).
func (w *Worker) cycle(ctx context.Context) {
	err := w.scoreOnce(ctx)
	if err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("opportunity cycle", "err", err)
	}
	if w.d.Health != nil {
		w.d.Health.Report(health.WorkerOpportunity, err == nil, err, 0)
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
