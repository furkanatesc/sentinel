package creatorfill

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// CreatorResolver, mint'ten creator çözer (DIP; ingest.HeliusCreatorResolver karşılar).
type CreatorResolver interface {
	ResolveCreator(ctx context.Context, mint string) (creator string, found bool, err error)
}

// CreatorFillStore, hedef seçimi + persist (DIP; store.TokenStore karşılar).
type CreatorFillStore interface {
	CreatorFillTargets(ctx context.Context, limit int) ([]store.CreatorFillTarget, error)
	SetCreatorBackfill(ctx context.Context, mint, creator string, backfillTs int64) error
}

type WorkerDeps struct {
	Store    CreatorFillStore
	Resolver CreatorResolver
	Interval time.Duration
	Limit    int
	Now      func() int64
	Logger   *slog.Logger
}

// Worker, creator'sız pump.fun token'ları için create tx'ten creator'ı REST ile getirir (Enricher deseni).
type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Interval <= 0 {
		d.Interval = 30 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 20
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
	if err := w.fillOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("creator backfill", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.fillOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("creator backfill", "err", err)
			}
		}
	}
}

// fillOnce, bir döngü: hedefleri çek → her mint için creator resolve → persist + damga (kısmi hata izole).
func (w *Worker) fillOnce(ctx context.Context) error {
	targets, err := w.d.Store.CreatorFillTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		creator, _, err := w.d.Resolver.ResolveCreator(ctx, tg.Mint)
		if err != nil {
			w.d.Logger.Warn("resolve creator", "mint", tg.Mint, "err", err)
			continue // RPC hatası → atla; sonraki tick tekrar dener (damgalanmadı)
		}
		// bulundu ya da bulunamadı: her iki durumda damgala (sonsuz retry yok); boş creator gerçek'i ezmez.
		if err := w.d.Store.SetCreatorBackfill(ctx, tg.Mint, creator, now); err != nil {
			w.d.Logger.Warn("set creator backfill", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
