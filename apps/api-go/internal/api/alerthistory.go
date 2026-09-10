package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// alertsHistoryHandler, üretilmiş alarm geçmişini döner (GET /api/alerts). Frontend AlertEvent[].
func alertsHistoryHandler(s store.AlertEventStore, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := s.RecentAlertEvents(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "alerts unavailable"})
			return
		}
		if events == nil {
			events = []store.AlertEventRow{}
		}
		writeJSON(w, http.StatusOK, events)
	}
}
