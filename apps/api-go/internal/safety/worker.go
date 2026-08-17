package safety

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// SafetyStore, Worker'ın kalıcılık bağımlılığıdır (DIP; store.TokenStore karşılar).
type SafetyStore interface {
	SafetyScoreTargets(ctx context.Context, limit int) ([]store.SafetyTarget, error)
	UpdateSafety(ctx context.Context, s store.SafetyUpdate) error
}

type WorkerDeps struct {
	Store    SafetyStore
	Provider DataProvider
	Interval time.Duration
	Limit    int
	Now      func() int64
	Logger   *slog.Logger
}

// Worker, periyodik olarak skorlanacak token'ları çekip skorlayıp DB'ye yazar (Enricher deseni).
type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Interval <= 0 {
		d.Interval = 60 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 40
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
		w.d.Logger.Warn("safety score", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("safety score", "err", err)
			}
		}
	}
}

// scoreOnce, bir döngü: hedefleri çek → her birini skorla → persist (kısmi hata izole).
func (w *Worker) scoreOnce(ctx context.Context) error {
	targets, err := w.d.Store.SafetyScoreTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		data, err := w.d.Provider.FetchOnChain(ctx, tg.Mint, tg.Creator)
		if err != nil {
			w.d.Logger.Warn("fetch on-chain", "mint", tg.Mint, "err", err)
			continue
		}
		// creator holding Scorer'a GİRMEZ (double-counting olur) — sadece aşağıda persist edilir.
		res := Score(Inputs{
			MintAuthorityActive: data.MintAuthorityActive, FreezeAuthorityActive: data.FreezeAuthorityActive,
			AuthoritiesKnown: data.AuthoritiesKnown, HolderCount: data.HolderCount, Top10Pct: data.Top10Pct,
			HoldersKnown: data.HoldersKnown, HoldersCapped: data.HoldersCapped, Liquidity: tg.Liquidity, Launchpad: tg.Launchpad,
		})
		if err := w.d.Store.UpdateSafety(ctx, store.SafetyUpdate{
			Mint: tg.Mint, Score: res.Score, Confidence: res.Confidence, Top10Pct: res.Top10Pct,
			Breakdown: res.Breakdown, Risks: res.Risks, ScoredTs: now,
			CreatorHoldingPct: data.CreatorHoldingPct, CreatorHoldingKnown: data.CreatorHoldingKnown,
		}); err != nil {
			w.d.Logger.Warn("update safety", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
