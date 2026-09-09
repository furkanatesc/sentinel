package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func newAlertRouter() http.Handler {
	return NewRouter(RouterDeps{AlertRules: store.NewFakeAlertRuleStore()})
}

func TestAlertRulesListEndpoint(t *testing.T) {
	r := newAlertRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/alert-rules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var rules []store.AlertRule
	if err := json.NewDecoder(w.Body).Decode(&rules); err != nil {
		t.Fatal(err)
	}
	if len(rules) != 4 {
		t.Fatalf("seed 4 kural beklenir: %d", len(rules))
	}
}

func TestCreateAlertRuleEndpoint(t *testing.T) {
	r := newAlertRouter()
	body := `{"name":"Yeni Kural","trigger":"new_mint","scope":"Tüm","minLiquidity":0,"minCreatorScore":50,"maxRisk":"medium","channels":["slack"],"enabled":true}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/alert-rules", strings.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create → 201 beklenir, got %d (%s)", w.Code, w.Body.String())
	}
	var created store.AlertRule
	_ = json.NewDecoder(w.Body).Decode(&created)
	if created.ID == "" || created.Name != "Yeni Kural" {
		t.Fatalf("oluşturulan kural yanlış: %+v", created)
	}
}

func TestCreateAlertRuleValidation(t *testing.T) {
	r := newAlertRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/alert-rules", strings.NewReader(`{"trigger":"new_mint"}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("isimsiz → 400 beklenir, got %d", w.Code)
	}
}

func TestSetAlertRuleEnabledEndpoint(t *testing.T) {
	r := newAlertRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/alert-rules/r3", strings.NewReader(`{"enabled":true}`)))
	if w.Code != http.StatusNoContent {
		t.Fatalf("toggle → 204 beklenir, got %d", w.Code)
	}
}

func TestSetAlertRuleEnabledNotFound(t *testing.T) {
	r := newAlertRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/alert-rules/yok", strings.NewReader(`{"enabled":true}`)))
	if w.Code != http.StatusNotFound {
		t.Fatalf("bilinmeyen id → 404 beklenir, got %d", w.Code)
	}
}
