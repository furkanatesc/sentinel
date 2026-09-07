package trend

import (
	"context"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeSampler struct{ inserts, prunes int }

func (f *fakeSampler) Kpis(context.Context) (store.KpiCounts, error) {
	return store.KpiCounts{Detected: 7}, nil
}
func (f *fakeSampler) InsertKpiSample(context.Context, int64, store.KpiCounts) error {
	f.inserts++
	return nil
}
func (f *fakeSampler) PruneKpiSamples(context.Context, int) error {
	f.prunes++
	return nil
}

type recReporter struct {
	name      string
	ok        bool
	processed int
	calls     int
}

func (r *recReporter) Register(string, bool, time.Duration) {}
func (r *recReporter) Report(name string, ok bool, _ error, processed int) {
	r.name, r.ok, r.processed, r.calls = name, ok, processed, r.calls+1
}

func TestWorkerCycleSamplesAndReports(t *testing.T) {
	fs := &fakeSampler{}
	rr := &recReporter{}
	w := NewWorker(WorkerDeps{Store: fs, Interval: time.Hour, Keep: 10, Health: rr, Now: func() int64 { return 42 }})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.Run(ctx) // immediate cycle + ctx.Done ile dön
	if fs.inserts != 1 || fs.prunes != 1 {
		t.Fatalf("1 insert + 1 prune beklenir: %+v", fs)
	}
	if rr.calls == 0 || rr.name != "trend" || !rr.ok || rr.processed != 1 {
		t.Fatalf("Report(trend, ok, 1) beklenir: %+v", rr)
	}
}
