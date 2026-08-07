package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// TokenDetailProvider, tek token detayını kurar (DIP; market.TokenDetailService karşılar).
type TokenDetailProvider interface {
	Build(ctx context.Context, mint string) (store.TokenDetail, bool, error)
}

// tokenHandler, /api/token/{mint} isteğini işler. timeout > 0 ise Build çağrısı
// deadline'lı bir ctx alır: paylaşılan GeckoTerminal limiter'ı kuyruğa girdiğinde
// kullanıcı isteği süresiz bloke olmaz; deadline aşılırsa rate.Limiter.Wait hızlı
// hata döner → 502 degraded yanıt (partial-success tasarımıyla tutarlı).
func tokenHandler(svc TokenDetailProvider, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mint := chi.URLParam(r, "mint")
		ctx := r.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		d, ok, err := svc.Build(ctx, mint)
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
