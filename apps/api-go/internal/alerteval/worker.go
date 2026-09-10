package alerteval

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Source, worker'ın okuma bağımlılığıdır (DIP; store karşılar). EventsSince = ileri-cursor (ts ASC).
type Source interface {
	EventsSince(ctx context.Context, afterTs int64, limit int) ([]store.EventRow, error)
	ListAlertRules(ctx context.Context) ([]store.AlertRule, error)
}

// Sink, worker'ın yazma + watermark bağımlılığıdır (DIP; store.AlertEventStore karşılar).
type Sink interface {
	InsertAlertEvent(ctx context.Context, e store.AlertEventRow) (bool, error)
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

// Cycle, watermark'tan (dahil) itibaren event'leri EN ESKİDEN yeniye çeker, aktif kurallarla eşleştirir,
// eşleşmeleri yazar (idempotent — event+kural ID), watermark'ı işlenen en yeni ts'e ilerletir. YENİ üretilen
// alarm sayısını döner. İleri-cursor + inclusive alt-sınır: burst'te (>Limit) atlamadan drain eder ve
// sınır-saniyesindeki (ts == watermark) event'leri düşürmez (idempotent insert çift-üretimi önler).
func (w *Worker) Cycle(ctx context.Context) (int, error) {
	wm, err := w.d.Sink.GetAlertWatermark(ctx)
	if err != nil {
		return 0, err
	}
	events, err := w.d.Source.EventsSince(ctx, wm, w.d.Limit)
	if err != nil {
		return 0, err
	}
	rules, err := w.d.Source.ListAlertRules(ctx)
	if err != nil {
		return 0, err
	}
	now := w.d.Now()
	maxTs := wm
	produced := 0
	for _, e := range events {
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
				ID:       fmt.Sprintf("a%s-%s", e.ID, r.ID), // event+kural kompoziti → aynı-saniye çakışması yok
				RuleID:   r.ID,
				Type:     e.Type,
				Token:    token,
				Detail:   r.Name + " — " + e.Detail,
				Severity: e.Severity,
				Time:     relativeTime(now - e.Ts), // event'in Time'ı canlıda boş → ts'ten türet
				Ts:       e.Ts,
			}
			inserted, err := w.d.Sink.InsertAlertEvent(ctx, row)
			if err != nil {
				return produced, err
			}
			if inserted {
				produced++
			}
		}
	}
	if maxTs > wm {
		if err := w.d.Sink.SetAlertWatermark(ctx, maxTs); err != nil {
			return produced, err
		}
	}
	return produced, nil
}

// relativeTime, saniye farkını Türkçe göreli zaman string'ine çevirir (mock stiliyle uyumlu:
// "az önce" / "Xsn önce" / "Xdk önce" / "Xsa önce" / "Xg önce"). Insert anında dondurulur (followup: mutlak/istemci).
func relativeTime(deltaSec int64) string {
	if deltaSec < 10 {
		return "az önce"
	}
	if deltaSec < 60 {
		return fmt.Sprintf("%dsn önce", deltaSec)
	}
	if deltaSec < 3600 {
		return fmt.Sprintf("%ddk önce", deltaSec/60)
	}
	if deltaSec < 86400 {
		return fmt.Sprintf("%dsa önce", deltaSec/3600)
	}
	return fmt.Sprintf("%dg önce", deltaSec/86400)
}
