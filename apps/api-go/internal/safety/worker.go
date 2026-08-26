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
// Döngü sonunda tik başına TEK özet log basar (gözlemlenebilirlik): hiçbir token
// skorlanamazsa (ör. Helius 429 → sessiz nötr-sıfır) WARN + örnek neden ile alarm verir.
func (w *Worker) scoreOnce(ctx context.Context) error {
	targets, err := w.d.Store.SafetyScoreTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	var scored, totalFail, authUnknown, holdersUnknown int
	var sampleErr error
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		data, err := w.d.Provider.FetchOnChain(ctx, tg.Mint, tg.Creator)
		if err != nil {
			// İki kaynak da başarısız → önceki gerçek skoru neutral ile EZME (skip).
			totalFail++
			sampleErr = err
			continue
		}
		if !data.AuthoritiesKnown {
			authUnknown++
		}
		if !data.HoldersKnown {
			holdersUnknown++
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
			MintAuthority: data.MintAuthorityAddr, FreezeAuthority: data.FreezeAuthorityAddr, AuthoritiesKnown: data.AuthoritiesKnown,
		}); err != nil {
			w.d.Logger.Warn("update safety", "mint", tg.Mint, "err", err)
			continue
		}
		if res.Confidence > 0 {
			scored++
		}
	}
	if len(targets) > 0 {
		w.logCycle(ctx, len(targets), scored, totalFail, authUnknown, holdersUnknown, sampleErr)
	}
	return nil
}

// logCycle, tik özetini basar: hiç skorlama olmadıysa WARN (alarm), aksi hâlde INFO.
// sampleErr (varsa) kök nedeni taşır (ör. Helius 429) — sessiz degradation'ı görünür kılar.
func (w *Worker) logCycle(ctx context.Context, targets, scored, totalFail, authUnknown, holdersUnknown int, sampleErr error) {
	attrs := []any{"targets", targets, "scored", scored, "totalFail", totalFail,
		"authUnknown", authUnknown, "holdersUnknown", holdersUnknown}
	if sampleErr != nil {
		attrs = append(attrs, "sampleErr", sampleErr.Error())
	}
	level := slog.LevelInfo
	if scored == 0 {
		level = slog.LevelWarn
	}
	w.d.Logger.Log(ctx, level, "safety cycle", attrs...)
}
