package ingest

import (
	"testing"
	"time"
)

type spyReporter struct {
	calls         int
	lastName      string
	lastProcessed int
}

func (r *spyReporter) Register(string, bool, time.Duration) {}
func (r *spyReporter) Report(name string, ok bool, err error, processed int) {
	r.calls++
	r.lastName = name
	r.lastProcessed = processed
}

func TestIngestReporterFieldWired(t *testing.T) {
	rr := &spyReporter{}
	w := NewWorker(WorkerDeps{Health: rr}) // WSURL boş → Run hemen döner; alan varlığı derlensin
	if w == nil {
		t.Fatal("nil worker")
	}
}
