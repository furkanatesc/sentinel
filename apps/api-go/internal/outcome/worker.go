package outcome

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// OutcomeStore, Worker'ın kalıcılık bağımlılığıdır (DIP; store.TokenStore karşılar).
type OutcomeStore interface {
	OutcomeTargets(ctx context.Context, limit int) ([]store.OutcomeTarget, error)
	UpdateOutcome(ctx context.Context, u store.OutcomeUpdate) error
}

type WorkerDeps struct {
	Store      OutcomeStore
	Thresholds Thresholds
	Interval   time.Duration
	Limit      int
	Now        func() int64
	Logger     *slog.Logger
}

// Worker, periyodik olarak pool'lu token'ları çekip sınıflayıp DB'ye yazar (Enricher deseni; dış çağrı yok).
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
	if err := w.classifyOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("outcome classify", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.classifyOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("outcome classify", "err", err)
			}
		}
	}
}

// classifyOnce, bir döngü: hedefleri çek → her birini sınıfla → persist (kısmi hata izole).
func (w *Worker) classifyOnce(ctx context.Context) error {
	targets, err := w.d.Store.OutcomeTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		res := Classify(Input{
			CurMarketCap: tg.CurMarketCap, CurLiquidity: tg.CurLiquidity,
			PeakMarketCap: tg.PeakMarketCap, PeakLiquidity: tg.PeakLiquidity,
			Vol24h: tg.Vol24h, AgeSeconds: now - tg.FirstSeenTs,
		}, w.d.Thresholds)
		if err := w.d.Store.UpdateOutcome(ctx, store.OutcomeUpdate{
			Mint: tg.Mint, Outcome: res.Outcome, LiquidityStatus: res.LiquidityStatus,
			MaxDrawdownPct: res.MaxDrawdownPct, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("update outcome", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
