package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/ws"
)

// RouterDeps, router'ın bağımlılıklarıdır (DIP: nil olan store'lar için route atlanır).
type RouterDeps struct {
	Strategies            store.StrategyStore
	Events                store.EventStore
	Tokens                store.TokenStore
	TokenDetail           TokenDetailProvider
	TokenDetailTimeout    time.Duration // 0 → sınırsız (kullanıcı yolunu limiter kuyruğunda süresiz bekletmemek için)
	Hub                   *ws.Hub
	CORSOrigin            string
	EventsWindow          int
	Creators              store.CreatorStore
	CreatorsLimit         int
	WalletGraphMinCluster int
	WalletGraphMaxDegree  int
	KpiSparkWindow        int    // /api/kpis spark penceresi (son N örnek); 0 → handler varsayılanı (24)
	BacktestServiceURL    string // Python backtest servisi; boş → /api/backtest graceful 503
	AlertRules            store.AlertRuleStore
	NotifyCfg             store.NotificationConfigStore
	AlertEvents           store.AlertEventStore
	Health                healthSnapshotter
	Pinger                store.Pinger
	Gates                 map[string]bool
	Version               string
	StartedAt             time.Time
	WSClientCount         func() int
}

// NewRouter, HTTP yönlendiricisini kurar.
func NewRouter(d RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(d.CORSOrigin))
	r.Get("/healthz", healthHandler)
	r.Get("/api/system-health", systemHealthHandler(d.Health, d.Pinger, d.Gates, d.Version, d.StartedAt, d.WSClientCount))
	if d.Strategies != nil {
		r.Get("/api/strategies", strategiesHandler(d.Strategies))
	}
	if d.Events != nil {
		r.Get("/api/events", eventsHandler(d.Events, d.EventsWindow))
	}
	if d.Tokens != nil {
		r.Get("/api/tokens", tokensHandler(d.Tokens, d.EventsWindow))
		r.Get("/api/kpis", kpisHandler(d.Tokens, d.KpiSparkWindow))
		r.Get("/api/radar", radarHandler(d.Tokens, d.EventsWindow))
		mc, md := d.WalletGraphMinCluster, d.WalletGraphMaxDegree
		if mc <= 0 {
			mc = 2
		}
		if md <= 0 {
			md = 50
		}
		r.Get("/api/wallet-graph", walletGraphHandler(d.Tokens, mc, md))
		r.Get("/api/authority-graph", authorityGraphHandler(d.Tokens, mc, md))
	}
	if d.TokenDetail != nil {
		r.Get("/api/token/{mint}", tokenHandler(d.TokenDetail, d.TokenDetailTimeout))
	}
	// /api/backtest her zaman kayıtlı; BacktestServiceURL boşsa handler graceful 503 döner.
	r.Post("/api/backtest", backtestHandler(d.BacktestServiceURL, 30*time.Second))
	if d.AlertRules != nil {
		r.Get("/api/alert-rules", alertRulesListHandler(d.AlertRules))
		r.Post("/api/alert-rules", createAlertRuleHandler(d.AlertRules))
		r.Patch("/api/alert-rules/{id}", setAlertRuleEnabledHandler(d.AlertRules))
	}
	if d.NotifyCfg != nil {
		r.Get("/api/notification-config", notificationConfigHandler(d.NotifyCfg))
		r.Put("/api/notification-config", saveNotificationConfigHandler(d.NotifyCfg))
	}
	if d.AlertEvents != nil {
		limit := d.EventsWindow
		if limit <= 0 {
			limit = 100
		}
		r.Get("/api/alerts", alertsHistoryHandler(d.AlertEvents, limit))
	}
	if d.Creators != nil {
		limit := d.CreatorsLimit
		if limit <= 0 {
			limit = 100
		}
		r.Get("/api/creators", creatorsHandler(d.Creators, limit))
		r.Get("/api/creator/{address}", creatorDetailHandler(d.Creators))
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
