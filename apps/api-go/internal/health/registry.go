// Package health, worker/altsistem canlılığını toplayan hafif telemetri katmanıdır
// (push modeli): worker'lar Reporter ile durum yazar, endpoint Snapshot okur.
package health

import (
	"regexp"
	"sync"
	"time"
)

type State string

const (
	StateOff      State = "off"
	StateStarting State = "starting"
	StateOK       State = "ok"
	StateDegraded State = "degraded"
	StateStalled  State = "stalled"
)

// Worker adı sabitleri — main.go (Register) + worker'lar (Report) AYNI adı kullansın (DRY).
const (
	WorkerIngestWS     = "ingest-ws"
	WorkerMarketDisc   = "market-disc"
	WorkerMarketEnrich = "market-enrich"
	WorkerSafety       = "safety"
	WorkerOutcome      = "outcome"
	WorkerCreatorFill  = "creatorfill"
	WorkerFunder       = "funder"
	WorkerReputation   = "reputation"
	WorkerManipulation = "manipulation"
	WorkerOpportunity  = "opportunity"
)

// WorkerStatus, tek worker'ın türetilmiş anlık durumudur (JSON: frontend SystemHealth ile birebir).
type WorkerStatus struct {
	Name           string `json:"name"`
	State          State  `json:"state"`
	LastRunAt      string `json:"lastRunAt"`
	LastErr        string `json:"lastErr"`
	CyclesRun      int    `json:"cyclesRun"`
	ItemsProcessed int    `json:"itemsProcessed"`
	IntervalSec    int    `json:"intervalSec"`
}

// Reporter, worker'lara enjekte edilen dar yazma arayüzüdür (ISP/DIP).
type Reporter interface {
	Register(name string, enabled bool, interval time.Duration)
	Report(name string, ok bool, err error, processed int)
}

type record struct {
	enabled        bool
	interval       time.Duration
	registeredAt   time.Time
	lastRunAt      time.Time
	lastOK         bool
	lastErr        string
	cyclesRun      int
	itemsProcessed int
}

// Registry, thread-safe worker durum deposudur.
type Registry struct {
	mu    sync.RWMutex
	now   func() time.Time
	works map[string]*record
	order []string
}

func NewRegistry() *Registry {
	return &Registry{now: time.Now, works: map[string]*record{}}
}

func (r *Registry) Register(name string, enabled bool, interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.works[name]
	if !ok {
		rec = &record{registeredAt: r.now()}
		r.works[name] = rec
		r.order = append(r.order, name)
	}
	rec.enabled = enabled
	rec.interval = interval
}

func (r *Registry) Report(name string, ok bool, err error, processed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, exists := r.works[name]
	if !exists {
		rec = &record{registeredAt: r.now(), enabled: true}
		r.works[name] = rec
		r.order = append(r.order, name)
	}
	rec.lastRunAt = r.now()
	rec.cyclesRun++
	rec.itemsProcessed += processed
	rec.lastOK = ok
	if err != nil {
		rec.lastErr = sanitizeErr(err)
	} else {
		rec.lastErr = ""
	}
}

func (r *Registry) Snapshot(now time.Time) []WorkerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]WorkerStatus, 0, len(r.order))
	for _, name := range r.order {
		rec := r.works[name]
		lastRun := ""
		if !rec.lastRunAt.IsZero() {
			lastRun = rec.lastRunAt.UTC().Format(time.RFC3339)
		}
		out = append(out, WorkerStatus{
			Name: name, State: deriveState(rec, now), LastRunAt: lastRun,
			LastErr: rec.lastErr, CyclesRun: rec.cyclesRun,
			ItemsProcessed: rec.itemsProcessed, IntervalSec: int(rec.interval / time.Second),
		})
	}
	return out
}

// deriveState, kaydın anlık durumunu türetir (saf; now enjekte). Sıra önemli: off → (hiç
// çalışmadı: starting/stalled) → (çalıştı ama bayat: stalled) → degraded → ok.
func deriveState(rec *record, now time.Time) State {
	if !rec.enabled {
		return StateOff
	}
	if rec.cyclesRun == 0 {
		if rec.interval == 0 || now.Sub(rec.registeredAt) <= 3*rec.interval {
			return StateStarting
		}
		return StateStalled
	}
	if rec.interval > 0 && now.Sub(rec.lastRunAt) > 3*rec.interval {
		return StateStalled
	}
	if !rec.lastOK {
		return StateDegraded
	}
	return StateOK
}

// apiKeyRe, hata mesajlarındaki `api-key=...` değerini kırpar (public endpoint — secret sızmaz).
var apiKeyRe = regexp.MustCompile(`(?i)(api-key=)[^&\s]+`)

func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	return apiKeyRe.ReplaceAllString(err.Error(), "${1}REDACTED")
}

var _ Reporter = (*Registry)(nil)
