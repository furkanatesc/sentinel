package reputation

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// ReputationStore, Worker'ın kalıcılık + agrega bağımlılığıdır (DIP; store.TokenStore karşılar).
type ReputationStore interface {
	CreatorAggregates(ctx context.Context, limit int) ([]store.CreatorAgg, error)
	UpsertReputation(ctx context.Context, r store.CreatorReputation) error
}

type WorkerDeps struct {
	Store      ReputationStore
	Thresholds Thresholds
	Interval   time.Duration
	Limit      int
	Now        func() int64
	Logger     *slog.Logger
}

// Worker, periyodik olarak creator agregalarını skorlayıp persist eder (RPC YOK, saf DB).
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
		w.d.Logger.Warn("reputation", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("reputation", "err", err)
			}
		}
	}
}

// scoreOnce, bir döngü: agregaları çek → her birini skorla → persist (kısmi hata izole).
func (w *Worker) scoreOnce(ctx context.Context) error {
	aggs, err := w.d.Store.CreatorAggregates(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, agg := range aggs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r := Score(agg, w.d.Thresholds)
		if err := w.d.Store.UpsertReputation(ctx, store.CreatorReputation{
			Address: agg.Address, Score: r.Score, Confidence: r.Confidence, RiskLevel: r.RiskLevel,
			Breakdown: r.Breakdown, SuccessRatePct: r.SuccessRatePct,
			TotalTokens: agg.Total, ActiveTokens: agg.Active, RuggedTokens: agg.Rug, GraduatedTokens: agg.Graduated,
			AvgPeakMarketCap: agg.AvgPeakMarketCap, AvgLifetimeHours: agg.AvgLifetimeHours, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("upsert reputation", "address", agg.Address, "err", err)
		}
	}
	return nil
}
