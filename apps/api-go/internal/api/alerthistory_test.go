package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestAlertsHistoryEndpoint(t *testing.T) {
	s := store.NewFakeAlertEventStore()
	_, _ = s.InsertAlertEvent(context.Background(), store.AlertEventRow{ID: "a1", Type: "new_mint", Token: "AAA", Severity: "info", Time: "1m", Ts: 100})
	r := NewRouter(RouterDeps{AlertEvents: s})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/alerts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var out []store.AlertEventRow
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Token != "AAA" {
		t.Fatalf("1 alarm beklenir: %+v", out)
	}
}
