package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func alertRulesListHandler(s store.AlertRuleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rules, err := s.ListAlertRules(r.Context())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "alert rules unavailable"})
			return
		}
		if rules == nil {
			rules = []store.AlertRule{}
		}
		writeJSON(w, http.StatusOK, rules)
	}
}

func createAlertRuleHandler(s store.AlertRuleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rule store.AlertRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		if rule.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
			return
		}
		created, err := s.CreateAlertRule(r.Context(), rule)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "create failed"})
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func setAlertRuleEnabledHandler(s store.AlertRuleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		if err := s.SetAlertRuleEnabled(r.Context(), id, body.Enabled); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "update failed"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
