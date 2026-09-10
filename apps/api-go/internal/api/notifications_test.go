package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func newNotifyRouter() http.Handler {
	return NewRouter(RouterDeps{NotifyCfg: store.NewFakeNotificationConfigStore()})
}

func TestNotificationConfigGetEndpoint(t *testing.T) {
	r := newNotifyRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/notification-config", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var cfg notificationConfigDTO
	if err := json.NewDecoder(w.Body).Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Channel != "#alerts" || cfg.MinSeverity != "warning" {
		t.Fatalf("seed ayarı beklenir: %+v", cfg)
	}
	// bağlantı-durumu gerçek Slack yokken "disconnected" döner
	if cfg.SlackState != "disconnected" {
		t.Fatalf("slackState disconnected beklenir, got %q", cfg.SlackState)
	}
}

func TestNotificationConfigSaveEndpoint(t *testing.T) {
	r := newNotifyRouter()
	body := `{"channel":"#trades","minSeverity":"critical","quietHours":{"start":"22:00","end":"06:00","enabled":true},"templates":[{"trigger":"new_mint","template":"yeni"}],"tradeApproval":false}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/notification-config", strings.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("save → 200 beklenir, got %d (%s)", w.Code, w.Body.String())
	}
	var cfg notificationConfigDTO
	_ = json.NewDecoder(w.Body).Decode(&cfg)
	if cfg.Channel != "#trades" || cfg.MinSeverity != "critical" || cfg.TradeApproval {
		t.Fatalf("kaydedilen config yanlış: %+v", cfg)
	}

	// tekrar GET → kalıcı
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/notification-config", nil))
	var cfg2 notificationConfigDTO
	_ = json.NewDecoder(w2.Body).Decode(&cfg2)
	if cfg2.MinSeverity != "critical" || !cfg2.QuietHours.Enabled {
		t.Fatalf("save sonrası GET güncel olmalı: %+v", cfg2)
	}
}

func TestNotificationConfigSaveValidation(t *testing.T) {
	r := newNotifyRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/notification-config", strings.NewReader(`{"channel":"#x"}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("minSeverity'siz → 400 beklenir, got %d", w.Code)
	}
}
