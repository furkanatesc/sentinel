package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func newTestServer(st store.StrategyStore, origin string) *httptest.Server {
	return httptest.NewServer(NewRouter(RouterDeps{Strategies: st, CORSOrigin: origin}))
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(store.NewFakeStore(nil, nil), "")
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestStrategiesOK(t *testing.T) {
	srv := newTestServer(store.NewFakeStore(store.SeedRows(), nil), "https://app.example")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/strategies")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("CORS origin = %q", got)
	}
	var rows []store.StrategyRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("rows = %d, want 6", len(rows))
	}
}

func TestStrategiesStoreError(t *testing.T) {
	srv := newTestServer(store.NewFakeStore(nil, errors.New("db down")), "")
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/strategies")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}
