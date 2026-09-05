package ingest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
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

// call, tek bir health.Report çağrısının kaydıdır (thread-safe spy için).
type call struct {
	name      string
	ok        bool
	processed int
}

// syncReporter, Run goroutine'inden gelen Report'ları thread-safe biriktirir (heartbeat/
// disconnect yolunu canlı WS olmadan doğrulamak için — race-safe okuma sağlar).
type syncReporter struct {
	mu    sync.Mutex
	calls []call
}

func (r *syncReporter) Register(string, bool, time.Duration) {}
func (r *syncReporter) Report(name string, ok bool, err error, processed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call{name: name, ok: ok, processed: processed})
}
func (r *syncReporter) snapshot() []call {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]call(nil), r.calls...)
}

// TestRunReportsHeartbeatWithProcessedCount, fake Subscribe ile beslenen N decode-edilebilir
// bildirimden sonra heartbeat tick'inin Report(ingest-ws, ok=true, processed=N) çağırdığını
// doğrular — System Health planında ertelenen "ingest-ws canlı-WS testi" D-artığını kapatır.
func TestRunReportsHeartbeatWithProcessedCount(t *testing.T) {
	rr := &syncReporter{}
	reg := NewRegistry()
	reg.Register(NewPumpFunDecoder())
	es, ts := store.NewFakeEventStore(), store.NewFakeTokenStore()

	var mint [32]byte
	data := buildCreateEventB64("Cat", "CAT", "https://x/c.json", mint, [32]byte{}, [32]byte{})
	// Aynı Signature dedup'lanır (Process seen map'i); farklı sig'lerle 3 decode-edilebilir bildirim.
	notifs := make([]LogNotification, 3)
	for i := range notifs {
		notifs[i] = LogNotification{
			Signature: "sig" + string(rune('A'+i)), Slot: uint64(i + 1), ProgramID: PumpFunProgramID,
			Logs: []string{"Program log: Instruction: Create", "Program data: " + data},
		}
	}

	// fake Subscribe: bildirimleri kanala basar, sonra ctx iptaline kadar bloklar (canlı WS taklidi).
	fakeSub := func(ctx context.Context, _ string, _ []string, out chan<- LogNotification) error {
		for _, n := range notifs {
			out <- n
		}
		<-ctx.Done()
		return ctx.Err()
	}

	w := NewWorker(WorkerDeps{
		Registry: reg, Events: es, Tokens: ts, Broadcast: &capBroadcaster{},
		WSURL: "ws://fake", Subscribe: fakeSub, StatsInterval: 20 * time.Millisecond,
		Health: rr, Now: func() int64 { return 111 },
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	// İlk heartbeat'i (processed==3) bekle. Bildirimler ~µs'de tüketilir, ilk tick 20ms'de.
	deadline := time.After(2 * time.Second)
	for {
		hb, ok := findHeartbeat(rr.snapshot(), 3)
		if ok {
			if !hb.ok {
				t.Fatalf("heartbeat ok=false, want true")
			}
			break
		}
		select {
		case <-deadline:
			t.Fatalf("processed=3 heartbeat gelmedi: %+v", rr.snapshot())
		case <-time.After(2 * time.Millisecond):
		}
	}
}

// findHeartbeat, ok=true + verilen processed değerine sahip bir ingest-ws Report'u arar.
func findHeartbeat(calls []call, processed int) (call, bool) {
	for _, c := range calls {
		if c.name == health.WorkerIngestWS && c.ok && c.processed == processed {
			return c, true
		}
	}
	return call{}, false
}

// TestRunReportsDisconnect, Subscribe hata dönünce Run'ın Report(ingest-ws, ok=false, ..., 0)
// çağırdığını doğrular (reconnect/backoff yolu).
func TestRunReportsDisconnect(t *testing.T) {
	rr := &syncReporter{}
	reg := NewRegistry() // boş registry yeter; disconnect yolu decode gerektirmez

	// fake Subscribe: hemen hata döner (WS koptu). ctx iptaline kadar tekrar denenebilir.
	fakeSub := func(ctx context.Context, _ string, _ []string, _ chan<- LogNotification) error {
		return errors.New("ws connect: 429 max usage reached")
	}

	w := NewWorker(WorkerDeps{
		Registry: reg, Events: store.NewFakeEventStore(), Tokens: store.NewFakeTokenStore(),
		Broadcast: &capBroadcaster{}, WSURL: "ws://fake", Subscribe: fakeSub,
		StatsInterval: time.Hour, Health: rr, Now: func() int64 { return 1 },
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	deadline := time.After(2 * time.Second)
	for {
		if hasDisconnect(rr.snapshot()) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("disconnect Report gelmedi: %+v", rr.snapshot())
		case <-time.After(2 * time.Millisecond):
		}
	}
}

// hasDisconnect, ok=false bir ingest-ws Report'u olup olmadığını söyler.
func hasDisconnect(calls []call) bool {
	for _, c := range calls {
		if c.name == health.WorkerIngestWS && !c.ok {
			return true
		}
	}
	return false
}
