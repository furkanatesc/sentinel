package safety

import (
	"context"
	"testing"
	"time"
)

// recReporter, health.Reporter'ın test spy'ıdır (brief'ten).
type recReporter struct {
	name      string
	ok        bool
	err       error
	processed int
	calls     int
}

func (r *recReporter) Register(string, bool, time.Duration) {}
func (r *recReporter) Report(name string, ok bool, err error, processed int) {
	r.name, r.ok, r.err, r.processed, r.calls = name, ok, err, processed, r.calls+1
}

// TestSafetyWorkerReportsCycle, Run'ın her cycle'da Health.Report çağırdığını doğrular
// (Ruling-2: store.NewFakeTokenStore() SafetyStore'u karşılamıyor — mevcut safety
// testlerinin kullandığı fakeSafetyStore kullanılır; 0 hedef → scoreOnce nil döner →
// Report("safety", true, nil, 0)).
func TestSafetyWorkerReportsCycle(t *testing.T) {
	rr := &recReporter{}
	st := &fakeSafetyStore{}
	w := NewWorker(WorkerDeps{
		Store: st, Provider: nil,
		Interval: time.Hour, Health: rr,
	})
	// ctx'i hemen iptal et: Run immediate cycle'ı çalıştırır, sonra ctx.Done ile döner.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.Run(ctx)
	if rr.calls == 0 {
		t.Fatalf("Report never called")
	}
	if rr.name != "safety" {
		t.Fatalf("name = %q, want safety", rr.name)
	}
	if !rr.ok || rr.err != nil {
		t.Fatalf("0-target cycle should report ok=true err=nil, got ok=%v err=%v", rr.ok, rr.err)
	}
	if rr.processed != 0 {
		t.Fatalf("processed = %d, want 0 (v1 itemsProcessed placeholder)", rr.processed)
	}
}
