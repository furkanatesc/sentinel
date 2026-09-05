package safety

import (
	"context"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
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
		t.Fatalf("processed = %d, want 0 (0 hedef → persist yok)", rr.processed)
	}
}

// TestSafetyWorkerReportsProcessedCount, dolu bir cycle'da Report'a gerçek itemsProcessed
// (persist edilen token sayısı) taşındığını doğrular — sabit 0 placeholder değil.
func TestSafetyWorkerReportsProcessedCount(t *testing.T) {
	rr := &recReporter{}
	st := &fakeSafetyStore{targets: []store.SafetyTarget{
		{Mint: "M1", Liquidity: 5000, Launchpad: "Raydium"},
		{Mint: "M2", Liquidity: 3000, Launchpad: "Raydium"},
	}}
	prov := stubProvider{d: OnChainData{AuthoritiesKnown: true, HoldersKnown: true, HolderCount: 500, Top10Pct: 30}}
	w := NewWorker(WorkerDeps{Store: st, Provider: prov, Limit: 10, Health: rr, Now: func() int64 { return 1 }})
	w.cycle(context.Background())
	if rr.processed != 2 {
		t.Fatalf("processed = %d, want 2 (iki token persist edildi)", rr.processed)
	}
	if !rr.ok || rr.err != nil {
		t.Fatalf("sağlıklı cycle ok=true err=nil beklenir, got ok=%v err=%v", rr.ok, rr.err)
	}
}
