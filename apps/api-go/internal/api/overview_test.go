package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestKpisEndpoint(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Tokens: ts.(store.TokenStore)})
	req := httptest.NewRequest(http.MethodGet, "/api/kpis", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var kpis []Kpi
	json.NewDecoder(w.Body).Decode(&kpis)
	if len(kpis) != 8 {
		t.Fatalf("kpi=%d want 8 (4 gerçek + 4 placeholder)", len(kpis))
	}
	// placeholder'lar "—"
	byID := map[string]Kpi{}
	for _, k := range kpis {
		byID[k.ID] = k
	}
	if byID["positions"].Value != "—" {
		t.Fatalf("positions placeholder '—' olmalı, got %q", byID["positions"].Value)
	}
	if byID["detected"].Value == "—" {
		t.Fatalf("detected gerçek olmalı")
	}
}

func TestRadarEndpoint(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Tokens: ts.(store.TokenStore)})
	req := httptest.NewRequest(http.MethodGet, "/api/radar", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var pts []store.RadarPoint
	if err := json.NewDecoder(w.Body).Decode(&pts); err != nil {
		t.Fatal(err)
	} // boş fake → [] (nil değil)
}
