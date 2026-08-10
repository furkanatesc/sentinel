package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func creatorsHandler(cs store.CreatorStore, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := cs.Creators(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "creators unavailable"})
			return
		}
		if rows == nil {
			rows = []store.CreatorRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func creatorDetailHandler(cs store.CreatorStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := chi.URLParam(r, "address")
		p, ok, err := cs.CreatorDetail(r.Context(), address)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "creator detail unavailable"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "creator not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}
