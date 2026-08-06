package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// TokenDetailProvider, tek token detayını kurar (DIP; market.TokenDetailService karşılar).
type TokenDetailProvider interface {
	Build(ctx context.Context, mint string) (store.TokenDetail, bool, error)
}

func tokenHandler(svc TokenDetailProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mint := chi.URLParam(r, "mint")
		d, ok, err := svc.Build(r.Context(), mint)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "token detail unavailable"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found"})
			return
		}
		writeJSON(w, http.StatusOK, d)
	}
}
