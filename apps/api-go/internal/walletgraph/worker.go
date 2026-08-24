package walletgraph

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type FunderStore interface {
	FunderTargets(ctx context.Context, limit int) ([]store.FunderTarget, error)
	SetFunder(ctx context.Context, wallet, funder string, resolvedTs int64) error
}

type WorkerDeps struct {
	Store    FunderStore
	Resolver FunderResolver
	Interval time.Duration
	Limit    int
	Now      func() int64
	Logger   *slog.Logger
}

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
	if err := w.resolveOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("funder resolve", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.resolveOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("funder resolve", "err", err)
			}
		}
	}
}

func (w *Worker) resolveOnce(ctx context.Context) error {
	targets, err := w.d.Store.FunderTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		funder, _, err := w.d.Resolver.ResolveFunder(ctx, tg.Wallet)
		if err != nil {
			w.d.Logger.Warn("resolve funder", "wallet", tg.Wallet, "err", err)
			continue // RPC hatası → damgalama, sonraki tick tekrar dener.
		}
		// bulundu ya da bulunamadı: damgala (sonsuz retry yok; boş funder de "çözüldü").
		if err := w.d.Store.SetFunder(ctx, tg.Wallet, funder, now); err != nil {
			w.d.Logger.Warn("set funder", "wallet", tg.Wallet, "err", err)
		}
	}
	return nil
}
