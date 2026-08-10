package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/api"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/config"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/creatorfill"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/ingest"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/market"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/outcome"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/safety"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/ws"
)

// Derleme zamanı sözleşme kilidi: ws.Hub, ingest.Broadcaster'ı karşılamalı.
// Bu satır derlenmezse Broadcast imzaları sapmış demektir — assertion'ı silme, uyumsuzluğu düzelt.
var _ ingest.Broadcaster = (*ws.Hub)(nil)

// Derleme zamanı kilidi: *rate.Limiter, market.Limiter'ı karşılamalı (DIP: market
// paketi rate'i import etmez, uyum yalnız burada bağlanır). İmza saparsa bu satır kırılır.
var _ market.Limiter = (*rate.Limiter)(nil)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var bundle store.Bundle
	cleanup := func() error { return nil }
	if cfg.DatabaseURL != "" {
		dbctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		b, cl, err := store.OpenPostgres(dbctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			logger.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		bundle, cleanup = b, cl
	} else {
		logger.Warn("DATABASE_URL yok — in-memory fake store")
		fakeTokens := store.NewFakeTokenStore()
		bundle = store.Bundle{
			Strategies: store.NewFakeStore(store.SeedRows(), nil),
			Events:     store.NewFakeEventStore(),
			Tokens:     fakeTokens,
			Creators:   fakeTokens.(store.CreatorStore),
		}
	}
	defer cleanup()

	hub := ws.NewHub()
	go hub.Run(ctx)

	// ingestion worker (Helius key varsa)
	reg := ingest.NewRegistry()
	reg.Register(ingest.NewPumpFunDecoder())
	reg.Register(ingest.NewRaydiumCpmmDecoder())
	wsURL, rpcURL := "", ""
	if cfg.HeliusAPIKey != "" {
		wsURL, rpcURL = ingest.HeliusURLs(cfg.HeliusAPIKey)
	} else {
		logger.Warn("HELIUS_API_KEY yok — ingestion worker başlamayacak (REST/mock boş DB çalışır)")
	}
	worker := ingest.NewWorker(ingest.WorkerDeps{
		Registry: reg, Events: bundle.Events, Tokens: bundle.Tokens, Broadcast: hub,
		Tx: ingest.NewHeliusTx(rpcURL), Meta: ingest.NewHeliusMetadata(rpcURL),
		WSURL: wsURL, Logger: logger, TokensWindow: cfg.EventsWindow,
	})
	go worker.Run(ctx)

	// market keşif + enrichment (GeckoTerminal REST — WS'ten bağımsız, Slice 1b)
	// Paylaşılan hız sınırlayıcı: keşif + enrichment + detail TEK GeckoTerminal
	// istek bütçesini (keysiz free-tier ~30/dk) tüketir; 429 → nötr-sıfır'ı önler.
	var gtLimiter market.Limiter
	if cfg.MarketEnabled {
		gtLimiter = rate.NewLimiter(rate.Limit(float64(cfg.GeckoRatePerMin)/60.0), cfg.GeckoBurst)
		gt := market.NewGeckoTerminalClient(cfg.GeckoBaseURL, nil, market.WithLimiter(gtLimiter))
		disc := market.NewDiscoverer(market.DiscovererDeps{
			Provider: gt, Tokens: bundle.Tokens, Events: bundle.Events, Broadcast: hub,
			Interval: time.Duration(cfg.DiscoverInterval) * time.Second, SnapshotLimit: cfg.EventsWindow, Logger: logger,
		})
		enr := market.NewEnricher(market.EnricherDeps{
			Provider: gt, Tokens: bundle.Tokens, Broadcast: hub,
			Interval: time.Duration(cfg.EnrichInterval) * time.Second, Limit: cfg.EnrichLimit, Logger: logger,
		})
		go disc.Run(ctx)
		go enr.Run(ctx)
	} else {
		logger.Warn("MARKET_ENABLED=false — market keşif/enrichment kapalı")
	}

	// token detail service (GeckoTerminal header+OHLCV + Helius holders) — Slice 1c
	var detailSvc api.TokenDetailProvider
	if cfg.MarketEnabled && bundle.Tokens != nil {
		gtForDetail := market.NewGeckoTerminalClient(cfg.GeckoBaseURL, nil, market.WithLimiter(gtLimiter))
		var holders market.HoldersProvider
		if rpcURL != "" {
			holders = ingest.NewHeliusHolders(rpcURL)
		} else {
			holders = noopHolders{} // key yoksa holders 0 (dürüst)
		}
		detailSvc = market.NewTokenDetailService(market.TokenDetailDeps{
			Store: bundle.Tokens, Provider: gtForDetail, Holders: holders,
			CacheTTL:   time.Duration(cfg.TokenDetailCacheSec) * time.Second,
			OHLCVLimit: cfg.OHLCVLimit, HoldersCap: cfg.HoldersCap, Logger: logger,
		})
	}

	// token safety scorer (2a) — arka plan; Helius key + market gerekli
	if cfg.SafetyEnabled && bundle.Tokens != nil && rpcURL != "" {
		provider := safety.NewHeliusProvider(
			ingest.NewHeliusAuthorities(rpcURL), ingest.NewHeliusHolders(rpcURL), cfg.SafetyHoldersCap)
		sw := safety.NewWorker(safety.WorkerDeps{
			Store: bundle.Tokens, Provider: provider,
			Interval: time.Duration(cfg.SafetyIntervalSec) * time.Second, Limit: cfg.SafetyLimit, Logger: logger,
		})
		go sw.Run(ctx)
	} else if cfg.SafetyEnabled {
		logger.Warn("SAFETY: Helius key veya token store yok — safety scorer başlamayacak")
	}

	// token outcome sınıflandırıcı (2b-2a) — arka plan; Helius gerekmez (saf market verisi)
	if cfg.OutcomeEnabled && bundle.Tokens != nil {
		ow := outcome.NewWorker(outcome.WorkerDeps{
			Store: bundle.Tokens,
			Thresholds: outcome.Thresholds{
				RugLiqRatio: cfg.OutcomeRugLiqRatio, GraduationMcap: cfg.OutcomeGraduationMcap,
				DumpedDrawdown: cfg.OutcomeDumpedDrawdown, DeadVol: cfg.OutcomeDeadVol,
				MinLiqFloor: cfg.OutcomeMinLiqFloor, DeadAgeSec: int64(cfg.OutcomeDeadAgeSec),
			},
			Interval: time.Duration(cfg.OutcomeIntervalSec) * time.Second, Limit: cfg.OutcomeLimit, Logger: logger,
		})
		go ow.Run(ctx)
	}

	// REST creator backfill (WS blokörü baypas) — arka plan; Helius RPC gerekli
	if cfg.CreatorFillEnabled && bundle.Tokens != nil && rpcURL != "" {
		resolver := ingest.NewCreatorResolver(rpcURL, cfg.CreatorFillMaxSigPages)
		cw := creatorfill.NewWorker(creatorfill.WorkerDeps{
			Store: bundle.Tokens, Resolver: resolver,
			Interval: time.Duration(cfg.CreatorFillIntervalSec) * time.Second, Limit: cfg.CreatorFillLimit, Logger: logger,
		})
		go cw.Run(ctx)
	} else if cfg.CreatorFillEnabled {
		logger.Warn("CREATORFILL: Helius key veya token store yok — backfill başlamayacak")
	}

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: api.NewRouter(api.RouterDeps{
			Strategies: bundle.Strategies, Events: bundle.Events, Tokens: bundle.Tokens,
			Hub: hub, CORSOrigin: cfg.CORSOrigin, EventsWindow: cfg.EventsWindow,
			TokenDetail:        detailSvc,
			TokenDetailTimeout: time.Duration(cfg.TokenDetailTimeoutSec) * time.Second,
			Creators:           bundle.Creators,
			CreatorsLimit:      cfg.CreatorsListLimit,
		}),
	}
	go func() {
		logger.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	logger.Info("stopped")
}

// noopHolders, Helius key yokken holders'ı 0 döndürür (dürüst — sayı yok).
type noopHolders struct{}

func (noopHolders) HoldersCount(context.Context, string, int) (int, bool, error) {
	return 0, false, nil
}
