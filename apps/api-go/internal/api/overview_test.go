package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestKpiTrend(t *testing.T) {
	if sp, ch := kpiTrend(nil); len(sp) != 0 || ch != 0 {
		t.Fatalf("boş → ([],0), got %v %v", sp, ch)
	}
	if sp, ch := kpiTrend([]int{5}); len(sp) != 1 || ch != 0 {
		t.Fatalf("tek → ([5],0), got %v %v", sp, ch)
	}
	sp, ch := kpiTrend([]int{10, 15})
	if len(sp) != 2 || sp[0] != 10 || sp[1] != 15 {
		t.Fatalf("spark kronolojik: %v", sp)
	}
	if ch != 50 {
		t.Fatalf("change (15-10)/10*100=50, got %v", ch)
	}
	if _, ch := kpiTrend([]int{0, 5}); ch != 0 {
		t.Fatalf("first==0 → change 0, got %v", ch)
	}
}

func TestKpisEndpointWithSamples(t *testing.T) {
	ts := store.NewFakeTokenStore().(store.TokenStore)
	ctx := context.Background()
	// artış eğilimli 3 örnek → detected spark dolu + pozitif change
	_ = ts.InsertKpiSample(ctx, 100, store.KpiCounts{Detected: 10, HighConf: 2})
	_ = ts.InsertKpiSample(ctx, 200, store.KpiCounts{Detected: 12, HighConf: 3})
	_ = ts.InsertKpiSample(ctx, 300, store.KpiCounts{Detected: 15, HighConf: 4})

	r := NewRouter(RouterDeps{Tokens: ts, KpiSparkWindow: 24})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/kpis", nil))
	var kpis []Kpi
	json.NewDecoder(w.Body).Decode(&kpis)
	byID := map[string]Kpi{}
	for _, k := range kpis {
		byID[k.ID] = k
	}
	det := byID["detected"]
	if len(det.Spark) != 3 || det.Spark[0] != 10 || det.Spark[2] != 15 {
		t.Fatalf("detected spark [10,12,15] beklenir: %v", det.Spark)
	}
	if det.Change != 50 {
		t.Fatalf("detected change (15-10)/10*100=50 beklenir, got %v", det.Change)
	}
	// placeholder dokunulmaz (spark boş)
	if len(byID["positions"].Spark) != 0 {
		t.Fatalf("placeholder positions spark boş kalmalı: %v", byID["positions"].Spark)
	}
}

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
