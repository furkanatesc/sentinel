package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
)

type stubSnap struct{ ws []health.WorkerStatus }

func (s stubSnap) Snapshot(time.Time) []health.WorkerStatus { return s.ws }

type stubPinger struct{ err error }

func (p stubPinger) Ping(context.Context) error { return p.err }

func TestSystemHealthShape(t *testing.T) {
	d := RouterDeps{
		Health:        stubSnap{ws: []health.WorkerStatus{{Name: "safety", State: health.StateDegraded, LastErr: "api-key=REDACTED 429"}}},
		Pinger:        stubPinger{err: nil},
		Gates:         map[string]bool{"SAFETY_ENABLED": true, "WALLET_GRAPH_ENABLED": false},
		Version:       "abc123",
		StartedAt:     time.Now().Add(-time.Minute),
		WSClientCount: func() int { return 0 },
		Tokens:        nil,
	}
	h := systemHealthHandler(d.Health, d.Pinger, d.Gates, d.Version, d.StartedAt, d.WSClientCount)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/api/system-health", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
	var got SystemHealth
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.DBOk {
		t.Fatalf("dbOk = false, want true")
	}
	if got.UptimeSec < 59 {
		t.Fatalf("uptimeSec = %d, want ~60", got.UptimeSec)
	}
	if len(got.Workers) != 1 || got.Workers[0].Name != "safety" {
		t.Fatalf("workers wrong: %+v", got.Workers)
	}
	if got.Gates["WALLET_GRAPH_ENABLED"] != false {
		t.Fatalf("gates wrong: %+v", got.Gates)
	}
}

func TestSystemHealthDBDownStill200(t *testing.T) {
	h := systemHealthHandler(
		stubSnap{}, stubPinger{err: errors.New("conn refused")},
		map[string]bool{}, "dev", time.Now(), func() int { return 0 },
	)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/api/system-health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 even when DB down", rr.Code)
	}
	var got SystemHealth
	_ = json.NewDecoder(rr.Body).Decode(&got)
	if got.DBOk {
		t.Fatalf("dbOk = true, want false")
	}
}
