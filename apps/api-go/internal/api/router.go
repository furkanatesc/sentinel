package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/ws"
)

// RouterDeps, router'ın bağımlılıklarıdır (DIP: nil olan store'lar için route atlanır).
type RouterDeps struct {
	Strategies   store.StrategyStore
	Events       store.EventStore
	Tokens       store.TokenStore
	TokenDetail  TokenDetailProvider
	Hub          *ws.Hub
	CORSOrigin   string
	EventsWindow int
}

// NewRouter, HTTP yönlendiricisini kurar.
func NewRouter(d RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(d.CORSOrigin))
	r.Get("/healthz", healthHandler)
	if d.Strategies != nil {
		r.Get("/api/strategies", strategiesHandler(d.Strategies))
	}
	if d.Events != nil {
		r.Get("/api/events", eventsHandler(d.Events, d.EventsWindow))
	}
	if d.Tokens != nil {
		r.Get("/api/tokens", tokensHandler(d.Tokens, d.EventsWindow))
	}
	if d.TokenDetail != nil {
		r.Get("/api/token/{mint}", tokenHandler(d.TokenDetail))
	}
	r.Get("/ws", wsHandler(d.Hub))
	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// corsMiddleware, yalnız verilen origin'e izin verir (boşsa header eklemez).
func corsMiddleware(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
