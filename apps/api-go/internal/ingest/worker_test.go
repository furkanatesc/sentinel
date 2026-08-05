package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type capBroadcaster struct {
	topics   []string
	payloads []any
}

func (c *capBroadcaster) Broadcast(topic string, payload any) {
	c.topics = append(c.topics, topic)
	c.payloads = append(c.payloads, payload)
}

func newTestWorker() (*Worker, store.EventStore, store.TokenStore, *capBroadcaster) {
	reg := NewRegistry()
	reg.Register(NewPumpFunDecoder())
	es, ts := store.NewFakeEventStore(), store.NewFakeTokenStore()
	bc := &capBroadcaster{}
	w := NewWorker(WorkerDeps{Registry: reg, Events: es, Tokens: ts, Broadcast: bc, Now: func() int64 { return 111 }})
	return w, es, ts, bc
}

func TestProcessPersistsAndBroadcasts(t *testing.T) {
	w, es, ts, bc := newTestWorker()
	var mint [32]byte
	mint[0] = 3
	data := buildCreateEventB64("Cat", "CAT", "u", mint, [32]byte{}, [32]byte{})
	n := LogNotification{Signature: "sig", Slot: 1, ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data}}

	w.Process(context.Background(), n)

	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 2 { // new_mint + metadata_created
		t.Fatalf("events=%d, want 2", len(evs))
	}
	if evs[0].Ts != 111 {
		t.Fatalf("worker Ts damgalamadı: %d", evs[0].Ts)
	}
	toks, _ := ts.RecentTokens(context.Background(), 10)
	if len(toks) != 1 {
		t.Fatalf("tokens=%d, want 1", len(toks))
	}
	// events + tokens topic'lerine broadcast
	if len(bc.topics) == 0 {
		t.Fatal("broadcast yok")
	}

	// Seam kontratı (apps/web/lib/api/contract.ts): subscribeEvents tekil FeedEvent alır
	// (prepend); subscribeTokens []TokenRow alır ve useLiveTokens tüm listeyi DEĞİŞTİRİR.
	// Bu yüzden "events" payload'u tekil EventRow, "tokens" payload'u TAM snapshot array
	// olmalı — tekil TokenRow gönderilirse frontend listeyi o tek token'a indirger.
	var eventsSeen, tokensSeen int
	for i, topic := range bc.topics {
		switch topic {
		case "events":
			eventsSeen++
			if _, ok := bc.payloads[i].(store.EventRow); !ok {
				t.Fatalf("events payload tekil store.EventRow olmalı, got %T", bc.payloads[i])
			}
		case "tokens":
			tokensSeen++
			snap, ok := bc.payloads[i].([]store.TokenRow)
			if !ok {
				t.Fatalf("tokens payload []store.TokenRow (array) olmalı — tekil TokenRow OLMAMALI (frontend seam kontratı), got %T", bc.payloads[i])
			}
			if len(snap) != 1 || snap[0].Symbol != "CAT" {
				t.Fatalf("tokens snapshot beklenen token'ı içermiyor: %+v", snap)
			}
		}
	}
	if eventsSeen != 2 {
		t.Fatalf("events broadcast=%d, want 2", eventsSeen)
	}
	if tokensSeen != 2 {
		t.Fatalf("tokens broadcast=%d, want 2 (decode edilen her item sonrası snapshot yayınlanır)", tokensSeen)
	}
}

func TestProcessDedup(t *testing.T) {
	w, es, _, _ := newTestWorker()
	var mint [32]byte
	data := buildCreateEventB64("Cat", "CAT", "u", mint, [32]byte{}, [32]byte{})
	n := LogNotification{Signature: "sig", Slot: 1, ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data}}
	w.Process(context.Background(), n)
	w.Process(context.Background(), n) // aynı sig+type → dedup
	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 2 {
		t.Fatalf("dedup başarısız: events=%d, want 2 (tekrar eklenmemeli)", len(evs))
	}
}

func TestProcessUnknownProgramNoop(t *testing.T) {
	w, es, _, _ := newTestWorker()
	w.Process(context.Background(), LogNotification{ProgramID: "unknown", Logs: []string{"x"}})
	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 0 {
		t.Fatalf("bilinmeyen program olay üretmemeli: %d", len(evs))
	}
}

// failingTokenStore, UpsertToken her zaman hata döner (Process'in "token upsert
// başarısızsa tokens broadcast'i atla" davranışını test etmek için).
type failingTokenStore struct{ err error }

func (f *failingTokenStore) UpsertToken(_ context.Context, _ store.TokenRow, _ int64) error {
	return f.err
}
func (f *failingTokenStore) RecentTokens(_ context.Context, _ int) ([]store.TokenRow, error) {
	return nil, nil
}

func TestProcessUpsertTokenFailureSkipsTokenBroadcast(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewPumpFunDecoder())
	es := store.NewFakeEventStore()
	ts := &failingTokenStore{err: errors.New("boom")}
	bc := &capBroadcaster{}
	w := NewWorker(WorkerDeps{Registry: reg, Events: es, Tokens: ts, Broadcast: bc, Now: func() int64 { return 111 }})

	var mint [32]byte
	data := buildCreateEventB64("Cat", "CAT", "u", mint, [32]byte{}, [32]byte{})
	n := LogNotification{Signature: "sig", Slot: 1, ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data}}

	w.Process(context.Background(), n)

	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 2 {
		t.Fatalf("events=%d, want 2 (event insert olayı token upsert'ten bağımsız başarılı olmalı)", len(evs))
	}
	var eventsCount, tokensCount int
	for _, topic := range bc.topics {
		switch topic {
		case "events":
			eventsCount++
		case "tokens":
			tokensCount++
		}
	}
	if eventsCount != 2 {
		t.Fatalf("events broadcast=%d, want 2", eventsCount)
	}
	if tokensCount != 0 {
		t.Fatalf("tokens broadcast=%d, want 0 (UpsertToken hata verince yayınlanmamalı)", tokensCount)
	}
}

// snapshotFailingTokenStore, UpsertToken başarılı olur ama RecentTokens (snapshot
// için kullanılan) her zaman hata döner — Process'in "snapshot alınamıyorsa tokens
// broadcast'i atla, events broadcast'i etkilenmesin" davranışını test etmek için.
type snapshotFailingTokenStore struct{ upserted int }

func (f *snapshotFailingTokenStore) UpsertToken(_ context.Context, _ store.TokenRow, _ int64) error {
	f.upserted++
	return nil
}
func (f *snapshotFailingTokenStore) RecentTokens(_ context.Context, _ int) ([]store.TokenRow, error) {
	return nil, errors.New("snapshot boom")
}

func TestProcessRecentTokensErrorSkipsTokensBroadcast(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewPumpFunDecoder())
	es := store.NewFakeEventStore()
	ts := &snapshotFailingTokenStore{}
	bc := &capBroadcaster{}
	w := NewWorker(WorkerDeps{Registry: reg, Events: es, Tokens: ts, Broadcast: bc, Now: func() int64 { return 111 }})

	var mint [32]byte
	data := buildCreateEventB64("Cat", "CAT", "u", mint, [32]byte{}, [32]byte{})
	n := LogNotification{Signature: "sig", Slot: 1, ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data}}

	w.Process(context.Background(), n)

	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 2 {
		t.Fatalf("events=%d, want 2 (event insert olayı snapshot hatasından bağımsız başarılı olmalı)", len(evs))
	}
	if ts.upserted != 2 {
		t.Fatalf("UpsertToken çağrı sayısı=%d, want 2 (upsert başarılı olmalı, sadece snapshot okuma başarısız)", ts.upserted)
	}
	var eventsCount, tokensCount int
	for _, topic := range bc.topics {
		switch topic {
		case "events":
			eventsCount++
		case "tokens":
			tokensCount++
		}
	}
	if eventsCount != 2 {
		t.Fatalf("events broadcast=%d, want 2", eventsCount)
	}
	if tokensCount != 0 {
		t.Fatalf("tokens broadcast=%d, want 0 (RecentTokens hata verince yayınlanmamalı)", tokensCount)
	}
}

func TestNextBackoffDoublesAndCapsAtSpecMax(t *testing.T) {
	const max = 30 * time.Second
	cases := []struct{ cur, want time.Duration }{
		{time.Second, 2 * time.Second},
		{2 * time.Second, 4 * time.Second},
		{4 * time.Second, 8 * time.Second},
		{8 * time.Second, 16 * time.Second},
		{16 * time.Second, max}, // 16*2=32 tavanı aşar → tam 30s'e kırpılmalı (32 DEĞİL)
		{max, max},              // steady-state: tavanda kalmalı, artmaya devam etmemeli
	}
	for _, c := range cases {
		got := nextBackoff(c.cur, max)
		if got != c.want {
			t.Errorf("nextBackoff(%s, %s) = %s, want %s", c.cur, max, got, c.want)
		}
	}
}
