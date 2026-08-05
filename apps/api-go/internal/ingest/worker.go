package ingest

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Broadcaster, decode edilmiş kaydı bağlı client'lara yayar (DIP → ws.Hub).
type Broadcaster interface {
	Broadcast(topic string, payload any)
}

type WorkerDeps struct {
	Registry  *Registry
	Events    store.EventStore
	Tokens    store.TokenStore
	Broadcast Broadcaster
	Tx        TxFetcher       // canlıda Helius; testte nil/fake
	Meta      MetadataFetcher // canlıda Helius; testte nil/fake
	WSURL     string          // canlı abonelik; testte boş
	Now       func() int64    // enjekte edilebilir saat (test determinizmi)
	Logger    *slog.Logger
}

type Worker struct {
	d    WorkerDeps
	mu   sync.Mutex
	seen map[string]struct{} // dedup: signature|type
}

func NewWorker(d WorkerDeps) *Worker {
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d, seen: map[string]struct{}{}}
}

// Process, tek bir log bildirimini decode eder, dedup uygular, persist + broadcast yapar.
func (w *Worker) Process(ctx context.Context, n LogNotification) {
	dec, ok := w.d.Registry.Decoder(n.ProgramID)
	if !ok {
		return
	}
	decoded, err := dec.Decode(ctx, n, w.d.Tx, w.d.Meta)
	if err != nil {
		w.d.Logger.Warn("decode error", "program", n.ProgramID, "sig", n.Signature, "err", err)
		return
	}
	now := w.d.Now()
	for _, item := range decoded {
		key := n.Signature + "|" + item.Event.Type
		w.mu.Lock()
		if _, dup := w.seen[key]; dup {
			w.mu.Unlock()
			continue
		}
		w.seen[key] = struct{}{}
		w.mu.Unlock()

		e := item.Event
		e.Ts = now
		if err := w.d.Events.InsertEvent(ctx, e); err != nil {
			w.d.Logger.Warn("insert event", "err", err)
			continue
		}
		w.d.Broadcast.Broadcast("events", e)
		if err := w.d.Tokens.UpsertToken(ctx, item.Token, now); err != nil {
			w.d.Logger.Warn("upsert token", "err", err)
			continue // persist edilmemiş token'ı yayınlama; olay yine de yayınlandı (gerçek)
		}
		w.d.Broadcast.Broadcast("tokens", item.Token)
	}
}

// Run, Helius'a bağlanır, kopmada exponential backoff ile yeniden bağlanır.
// ctx iptaline kadar çalışır. WSURL boşsa (test/DB-siz) hemen döner.
func (w *Worker) Run(ctx context.Context) {
	if w.d.WSURL == "" {
		w.d.Logger.Warn("HELIUS_WS_URL yok — ingestion worker başlamadı (REST yine çalışır)")
		return
	}
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for ctx.Err() == nil {
		ch := make(chan LogNotification, 256)
		done := make(chan error, 1)
		subCtx, cancel := context.WithCancel(ctx)
		go func() { done <- SubscribeLogs(subCtx, w.d.WSURL, w.d.Registry.ProgramIDs(), ch) }()

		connected := true
		for connected {
			select {
			case <-ctx.Done():
				cancel()
				return
			case n := <-ch:
				w.Process(ctx, n)
				backoff = time.Second // sağlıklı trafik → backoff sıfırla
			case err := <-done:
				w.d.Logger.Warn("ws bağlantısı koptu, reconnect", "err", err, "backoff", backoff.String())
				connected = false
			}
		}
		cancel()
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		backoff = nextBackoff(backoff, maxBackoff)
	}
}

// nextBackoff, mevcut backoff'u ikiye katlar ve max'ta kırpar (asla max'ı aşmaz).
// Saf fonksiyon: Run'ın select/kanal mantığından ayrık, doğrudan test edilebilir.
func nextBackoff(cur, max time.Duration) time.Duration {
	next := cur * 2
	if next > max {
		next = max
	}
	return next
}
