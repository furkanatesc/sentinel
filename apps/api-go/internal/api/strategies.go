package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func strategiesHandler(st store.StrategyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := st.List(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "strategies unavailable"})
			return
		}
		if rows == nil {
			rows = []store.StrategyRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}
