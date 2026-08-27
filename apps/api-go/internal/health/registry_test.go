package health

import (
	"errors"
	"testing"
	"time"
)

func TestOffWhenDisabled(t *testing.T) {
	r := NewRegistry()
	r.Register("w", false, 30*time.Second)
	got := stateOf(t, r, "w", time.Now())
	if got != StateOff {
		t.Fatalf("state = %q, want off", got)
	}
}

func TestStartingBeforeFirstCycleWithinGrace(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	// no Report yet; 30s later still within 3×interval grace
	got := stateOf(t, r, "w", base.Add(30*time.Second))
	if got != StateStarting {
		t.Fatalf("state = %q, want starting", got)
	}
}

func TestStalledWhenNeverRanPastGrace(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	got := stateOf(t, r, "w", base.Add(200*time.Second)) // > 3×30s
	if got != StateStalled {
		t.Fatalf("state = %q, want stalled", got)
	}
}

func TestOKAfterHealthyReport(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	r.Report("w", true, nil, 5)
	got := stateOf(t, r, "w", base.Add(10*time.Second))
	if got != StateOK {
		t.Fatalf("state = %q, want ok", got)
	}
}

func TestDegradedWhenLastCycleFailed(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	r.Report("w", false, errors.New("boom"), 0)
	got := stateOf(t, r, "w", base.Add(10*time.Second))
	if got != StateDegraded {
		t.Fatalf("state = %q, want degraded", got)
	}
}

func TestStalledAfterRunButStale(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	r.Report("w", true, nil, 1)
	got := stateOf(t, r, "w", base.Add(200*time.Second)) // last run 200s ago > 90s
	if got != StateStalled {
		t.Fatalf("state = %q, want stalled", got)
	}
}

func TestIntervalZeroNeverTimeStalls(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("ingest-ws", true, 0)
	// even far in the future, no report → starting (event-driven, no time-stall)
	got := stateOf(t, r, "ingest-ws", base.Add(time.Hour))
	if got != StateStarting {
		t.Fatalf("state = %q, want starting", got)
	}
	r.Report("ingest-ws", true, nil, 0)
	if s := stateOf(t, r, "ingest-ws", base.Add(2*time.Hour)); s != StateOK {
		t.Fatalf("after report state = %q, want ok", s)
	}
}

func TestLastErrSanitizesAPIKey(t *testing.T) {
	r := NewRegistry()
	r.Register("w", true, 30*time.Second)
	r.Report("w", false, errors.New("get https://mainnet.helius-rpc.com/?api-key=SECRET123 failed"), 0)
	ws := findWorker(t, r.Snapshot(time.Now()), "w")
	if want := "api-key=REDACTED"; !contains(ws.LastErr, want) {
		t.Fatalf("lastErr = %q, want to contain %q", ws.LastErr, want)
	}
	if contains(ws.LastErr, "SECRET123") {
		t.Fatalf("lastErr leaked secret: %q", ws.LastErr)
	}
}

func TestSnapshotPreservesRegistrationOrderAndCounts(t *testing.T) {
	r := NewRegistry()
	r.Register("a", true, time.Second)
	r.Register("b", true, time.Second)
	r.Report("a", true, nil, 3)
	r.Report("a", true, nil, 2)
	snap := r.Snapshot(time.Now())
	if len(snap) != 2 || snap[0].Name != "a" || snap[1].Name != "b" {
		t.Fatalf("order/len wrong: %+v", snap)
	}
	if snap[0].CyclesRun != 2 || snap[0].ItemsProcessed != 5 {
		t.Fatalf("counts wrong: %+v", snap[0])
	}
}

func TestConcurrentReportsRace(t *testing.T) {
	r := NewRegistry()
	r.Register("w", true, time.Second)
	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				r.Report("w", true, nil, 1)
				_ = r.Snapshot(time.Now())
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}

// --- test helpers ---
func stateOf(t *testing.T, r *Registry, name string, now time.Time) State {
	t.Helper()
	return findWorker(t, r.Snapshot(now), name).State
}
func findWorker(t *testing.T, snap []WorkerStatus, name string) WorkerStatus {
	t.Helper()
	for _, w := range snap {
		if w.Name == name {
			return w
		}
	}
	t.Fatalf("worker %q not in snapshot", name)
	return WorkerStatus{}
}
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
