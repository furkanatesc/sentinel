// Package trend, mevcut KPI agregalarını periyodik snapshot'layan hafif zaman-serisi worker'ıdır
// (spark/change için). Saf DB — yeni harici bağımlılık YOK.
package trend

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Sampler, worker'ın store'a dar bağımlılığıdır (DIP; store.TokenStore karşılar).
type Sampler interface {
	Kpis(ctx context.Context) (store.KpiCounts, error)
	InsertKpiSample(ctx context.Context, ts int64, c store.KpiCounts) error
	PruneKpiSamples(ctx context.Context, keep int) error
	// trend B: en yeni N token likiditesi snapshot + yaş-tabanlı prune.
	InsertLiquiditySamples(ctx context.Context, ts int64, limit int) error
	PruneLiquiditySamples(ctx context.Context, cutoff int64) error
}

type WorkerDeps struct {
	Store    Sampler
	Interval time.Duration
	Keep     int          // retention: en yeni `keep` KPI örneği tutulur
	Now      func() int64 // enjekte edilebilir saat (test determinizmi)
	Logger   *slog.Logger
	Health   health.Reporter // nil-güvenli
	// trend B: token likidite örnekleme (aynı interval).
	LiqEnabled     bool
	LiqSampleLimit int   // en yeni N token
	LiqKeepSeconds int64 // yaş-tabanlı retention (ts < now-LiqKeepSeconds silinir)
}

type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d}
}

// Run, immediate bir cycle + ardından ticker döngüsü (diğer worker'larla aynı iskelet).
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	w.cycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.cycle(ctx)
		}
	}
}

// cycle, tek snapshot: Kpis oku → InsertKpiSample(now) → prune → health Report.
// itemsProcessed = o cycle'da persist edilen örnek sayısı (0 erken-hata, 1 insert başarılı) —
// diğer worker'ların "başarıyla persist edilen" konvansiyonuyla tutarlı (System Health).
func (w *Worker) cycle(ctx context.Context) {
	n, err := w.sampleOnce(ctx)
	// Likidite örnekleme KPI'dan bağımsız: KPI örneği yine yazılır; likidite hatası cycle'ı
	// degraded raporlar (ilk hata korunur).
	if w.d.LiqEnabled {
		if lerr := w.liqSampleOnce(ctx); lerr != nil && err == nil {
			err = lerr
		}
	}
	if err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("trend sample", "err", err)
	}
	if w.d.Health != nil {
		w.d.Health.Report(health.WorkerTrend, err == nil, err, n)
	}
}

// liqSampleOnce, en yeni N token likiditesini snapshot'lar + yaş-tabanlı prune.
func (w *Worker) liqSampleOnce(ctx context.Context) error {
	if err := w.d.Store.InsertLiquiditySamples(ctx, w.d.Now(), w.d.LiqSampleLimit); err != nil {
		return err
	}
	if w.d.LiqKeepSeconds > 0 {
		if err := w.d.Store.PruneLiquiditySamples(ctx, w.d.Now()-w.d.LiqKeepSeconds); err != nil {
			return err
		}
	}
	return nil
}

// sampleOnce, persist edilen örnek sayısını döndürür: Kpis/Insert hatası → (0,err);
// insert başarılı → 1 (prune hatası bunu değiştirmez — örnek zaten yazıldı).
func (w *Worker) sampleOnce(ctx context.Context) (int, error) {
	c, err := w.d.Store.Kpis(ctx)
	if err != nil {
		return 0, err
	}
	if err := w.d.Store.InsertKpiSample(ctx, w.d.Now(), c); err != nil {
		return 0, err
	}
	if w.d.Keep > 0 {
		if err := w.d.Store.PruneKpiSamples(ctx, w.d.Keep); err != nil {
			return 1, err
		}
	}
	return 1, nil
}
