package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func eventsHandler(es store.EventStore, window int) http.HandlerFunc {
	if window <= 0 {
		window = 200
	}
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := es.RecentEvents(r.Context(), window)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "events unavailable"})
			return
		}
		if rows == nil {
			rows = []store.EventRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func tokensHandler(ts store.TokenStore, window int) http.HandlerFunc {
	if window <= 0 {
		window = 200
	}
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := ts.RecentTokens(r.Context(), window)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "tokens unavailable"})
			return
		}
		if rows == nil {
			rows = []store.TokenRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}
