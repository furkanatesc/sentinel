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
	Registry     *Registry
	Events       store.EventStore
	Tokens       store.TokenStore
	Broadcast    Broadcaster
	Tx           TxFetcher       // canlıda Helius; testte nil/fake
	Meta         MetadataFetcher // canlıda Helius; testte nil/fake
	WSURL        string          // canlı abonelik; testte boş
	Now          func() int64    // enjekte edilebilir saat (test determinizmi)
	Logger       *slog.Logger
	TokensWindow int // "tokens" broadcast'i için snapshot penceresi (RecentTokens limit)
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
	if d.TokensWindow <= 0 {
		d.TokensWindow = 200
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
		// Seam kontratı (apps/web/lib/api/contract.ts): subscribeTokens []TokenRow alır,
		// useLiveTokens setQueryData ile TÜM listeyi bununla DEĞİŞTİRİR (prepend değil).
		// Tekil token göndermek frontend listesini o tek token'a indirger — bu yüzden
		// upsert sonrası tam snapshot okunup array olarak yayınlanır.
		snapshot, err := w.d.Tokens.RecentTokens(ctx, w.d.TokensWindow)
		if err != nil {
			w.d.Logger.Warn("tokens snapshot", "err", err)
			continue // kısmi/hatalı snapshot yayınlama
		}
		w.d.Broadcast.Broadcast("tokens", snapshot)
	}
}

// Run, Helius'a bağlanır, kopmada exponential backoff ile yeniden bağlanır.
// ctx iptaline kadar çalışır. WSURL boşsa (test/DB-siz) hemen döner.
func (w *Worker) Run(ctx context.Context) {
	if w.d.WSURL == "" {
		w.d.Logger.Warn("HELIUS_WS_URL yok — ingestion worker başlamadı (REST yine çalışır)")
		return
	}
	w.d.Logger.Info("ingestion worker başladı", "programlar", len(w.d.Registry.ProgramIDs()))
	// Teşhis heartbeat'i: Helius'un teslim ettiği bildirim/işlenen sayısını periyodik logla.
	// (Başarılı decode sessizdir; bu, "bağlı ama veri gelmiyor" durumunu görünür kılar.)
	stats := time.NewTicker(30 * time.Second)
	defer stats.Stop()
	var received, processed int64

	// Free-tier workaround: Helius standart logsSubscribe kısa bir teslimat penceresinden
	// sonra (bağlantıyı kapatmadan) susuyor. Aboneliği periyodik olarak proaktif yenileyerek
	// taze teslimat penceresi açıyoruz. Bu "zorunlu yenileme" gerçek kopmadan farklıdır:
	// backoff uygulanmaz, hemen (kısa bir bağlantı-kapanma boşluğuyla) yeniden abone olunur.
	const refreshInterval = 25 * time.Second
	refresh := time.NewTicker(refreshInterval)
	defer refresh.Stop()

	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for ctx.Err() == nil {
		ch := make(chan LogNotification, 256)
		done := make(chan error, 1)
		subCtx, cancel := context.WithCancel(ctx)
		go func() { done <- SubscribeLogs(subCtx, w.d.WSURL, w.d.Registry.ProgramIDs(), ch) }()
		refresh.Reset(refreshInterval) // yenileme süresini yeni bağlantıdan itibaren ölç

		connected, forced := true, false
		for connected {
			select {
			case <-ctx.Done():
				cancel()
				return
			case <-stats.C:
				w.d.Logger.Info("ingest heartbeat", "alınan_30s", received, "işlenen_30s", processed)
				received, processed = 0, 0
			case <-refresh.C:
				forced = true
				connected = false
			case n := <-ch:
				received++
				if _, ok := w.d.Registry.Decoder(n.ProgramID); ok {
					processed++
				}
				w.Process(ctx, n)
				backoff = time.Second // sağlıklı trafik → backoff sıfırla
			case err := <-done:
				w.d.Logger.Warn("ws bağlantısı koptu, reconnect", "err", err, "backoff", backoff.String())
				connected = false
			}
		}
		cancel()
		if forced {
			// Proaktif yenileme: eski bağlantının kapanması için kısa boşluk, sonra hemen yeniden abone ol.
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return
			}
			backoff = time.Second
			continue
		}
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
