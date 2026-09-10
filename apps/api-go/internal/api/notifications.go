package api

import (
	"encoding/json"
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// notificationConfigDTO, frontend NotificationConfig ile birebir JSON'dur: kalıcı ayarlar +
// bağlantı-durumu. Bağlantı-durumu (slackState/workspace) gerçek Slack OAuth'tan gelir; henüz
// yok → "disconnected"/"" döneriz (dürüst; mock modda bağlantı "connected" görünür).
type notificationConfigDTO struct {
	SlackState    string                       `json:"slackState"`
	Channel       string                       `json:"channel"`
	Workspace     string                       `json:"workspace"`
	MinSeverity   string                       `json:"minSeverity"`
	QuietHours    store.QuietHours             `json:"quietHours"`
	Templates     []store.NotificationTemplate `json:"templates"`
	TradeApproval bool                         `json:"tradeApproval"`
}

func toNotificationConfigDTO(s store.NotificationSettings) notificationConfigDTO {
	if s.Templates == nil {
		s.Templates = []store.NotificationTemplate{}
	}
	return notificationConfigDTO{
		SlackState:    "disconnected", // gerçek Slack OAuth = sona; bağlantı henüz kurulmadı
		Workspace:     "",
		Channel:       s.Channel,
		MinSeverity:   s.MinSeverity,
		QuietHours:    s.QuietHours,
		Templates:     s.Templates,
		TradeApproval: s.TradeApproval,
	}
}

func notificationConfigHandler(s store.NotificationConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := s.GetNotificationSettings(r.Context())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "notification config unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, toNotificationConfigDTO(settings))
	}
}

func saveNotificationConfigHandler(s store.NotificationConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var settings store.NotificationSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		if settings.MinSeverity == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "minSeverity required"})
			return
		}
		saved, err := s.SaveNotificationSettings(r.Context(), settings)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "save failed"})
			return
		}
		writeJSON(w, http.StatusOK, toNotificationConfigDTO(saved))
	}
}
