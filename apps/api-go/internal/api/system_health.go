package api

import (
	"context"
	"net/http"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// SystemHealth, /api/system-health JSON şeklidir (frontend types.ts ile birebir).
type SystemHealth struct {
	UptimeSec   int                   `json:"uptimeSec"`
	Version     string                `json:"version"`
	DBOk        bool                  `json:"dbOk"`
	DBLatencyMs int                   `json:"dbLatencyMs"`
	WSClients   int                   `json:"wsClients"`
	Workers     []health.WorkerStatus `json:"workers"`
	Gates       map[string]bool       `json:"gates"`
}

// healthSnapshotter, handler'ın registry'ye dar bağımlılığıdır (DIP; *health.Registry karşılar).
type healthSnapshotter interface {
	Snapshot(now time.Time) []health.WorkerStatus
}

func systemHealthHandler(
	snap healthSnapshotter, pinger store.Pinger, gates map[string]bool,
	version string, startedAt time.Time, wsClients func() int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		workers := []health.WorkerStatus{}
		if snap != nil {
			workers = snap.Snapshot(now)
		}
		dbOk, latencyMs := probeDB(r.Context(), pinger)
		clients := 0
		if wsClients != nil {
			clients = wsClients()
		}
		if gates == nil {
			gates = map[string]bool{}
		}
		writeJSON(w, http.StatusOK, SystemHealth{
			UptimeSec: int(now.Sub(startedAt).Seconds()), Version: version,
			DBOk: dbOk, DBLatencyMs: latencyMs, WSClients: clients,
			Workers: workers, Gates: gates,
		})
	}
}

// probeDB, kısa timeout'lu bir ping atar; başarısızlık 500 değil dbOk=false döndürür (graceful).
func probeDB(ctx context.Context, pinger store.Pinger) (bool, int) {
	if pinger == nil {
		return false, 0
	}
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := pinger.Ping(pctx); err != nil {
		return false, 0
	}
	return true, int(time.Since(start).Milliseconds())
}
