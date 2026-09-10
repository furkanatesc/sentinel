package alerteval

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Source, worker'ın okuma bağımlılığıdır (DIP; store karşılar).
type Source interface {
	RecentEvents(ctx context.Context, limit int) ([]store.EventRow, error)
	ListAlertRules(ctx context.Context) ([]store.AlertRule, error)
}

// Sink, worker'ın yazma + watermark bağımlılığıdır (DIP; store.AlertEventStore karşılar).
type Sink interface {
	InsertAlertEvent(ctx context.Context, e store.AlertEventRow) error
	GetAlertWatermark(ctx context.Context) (int64, error)
	SetAlertWatermark(ctx context.Context, ts int64) error
}

type WorkerDeps struct {
	Source   Source
	Sink     Sink
	Interval time.Duration
	Limit    int // RecentEvents pencere boyutu (varsayılan 200)
	Now      func() int64
	Logger   *slog.Logger
	Health   health.Reporter // nil-güvenli
}

type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Limit <= 0 {
		d.Limit = 200
	}
	return &Worker{d: d}
}

// Run, immediate bir cycle + ticker döngüsü (diğer worker'larla aynı iskelet).
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	w.runCycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.runCycle(ctx)
		}
	}
}

func (w *Worker) runCycle(ctx context.Context) {
	n, err := w.Cycle(ctx)
	if w.d.Health != nil {
		// Health.Report imzası: Report(name, ok, err, processed) — worker adı burada geçilir (trend deseni).
		w.d.Health.Report(health.WorkerAlertEval, err == nil, err, n)
	}
	if err != nil {
		w.d.Logger.Warn("alerteval cycle hatası", "err", err)
	}
}

// Cycle, watermark'tan yeni event'leri aktif kurallarla eşleştirir, eşleşmeleri yazar,
// watermark'ı ilerletir. Üretilen alarm sayısını döner. (Saf orkestrasyon; eşleştirme MatchRule.)
func (w *Worker) Cycle(ctx context.Context) (int, error) {
	wm, err := w.d.Sink.GetAlertWatermark(ctx)
	if err != nil {
		return 0, err
	}
	events, err := w.d.Source.RecentEvents(ctx, w.d.Limit)
	if err != nil {
		return 0, err
	}
	rules, err := w.d.Source.ListAlertRules(ctx)
	if err != nil {
		return 0, err
	}
	maxTs := wm
	produced := 0
	for _, e := range events {
		if e.Ts <= wm {
			continue // watermark altında = zaten değerlendirildi (dedup)
		}
		if e.Ts > maxTs {
			maxTs = e.Ts
		}
		for _, r := range rules {
			if !r.Enabled || !MatchRule(e, r) {
				continue
			}
			token := e.Symbol
			if token == "" {
				token = e.Mint
			}
			row := store.AlertEventRow{
				ID:       fmt.Sprintf("a%d-%s", e.Ts, r.ID),
				RuleID:   r.ID,
				Type:     e.Type,
				Token:    token,
				Detail:   r.Name + " — " + e.Detail,
				Severity: e.Severity,
				Time:     e.Time,
				Ts:       e.Ts,
			}
			if err := w.d.Sink.InsertAlertEvent(ctx, row); err != nil {
				return produced, err
			}
			produced++
		}
	}
	if maxTs > wm {
		if err := w.d.Sink.SetAlertWatermark(ctx, maxTs); err != nil {
			return produced, err
		}
	}
	return produced, nil
}
