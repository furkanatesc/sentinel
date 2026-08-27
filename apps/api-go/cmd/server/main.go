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
	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/ingest"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/manipulation"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/market"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/opportunity"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/outcome"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/reputation"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/safety"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/walletgraph"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/ws"
)

// Derleme zamanı sözleşme kilidi: ws.Hub, ingest.Broadcaster'ı karşılamalı.
// Bu satır derlenmezse Broadcast imzaları sapmış demektir — assertion'ı silme, uyumsuzluğu düzelt.
var _ ingest.Broadcaster = (*ws.Hub)(nil)

// Derleme zamanı kilidi: *rate.Limiter, market.Limiter'ı karşılamalı (DIP: market
// paketi rate'i import etmez, uyum yalnız burada bağlanır). İmza saparsa bu satır kırılır.
var _ market.Limiter = (*rate.Limiter)(nil)

// Aynı kilit ingest.Limiter için (creator-backfill resolver'ı Helius RPC'yi throttle'lar).
var _ ingest.Limiter = (*rate.Limiter)(nil)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var bundle store.Bundle
	cleanup := func() error { return nil }
	if cfg.DatabaseURL != "" {
		dbctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		b, cl, err := store.OpenPostgres(dbctx, cfg.DatabaseURL,
			store.WithHighDrawdownThreshold(cfg.ReputationHighDrawdown))
		cancel()
		if err != nil {
			logger.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		bundle, cleanup = b, cl
	} else {
		logger.Warn("DATABASE_URL yok — in-memory fake store")
		fakeTokens := store.NewFakeTokenStore(store.WithHighDrawdownThreshold(cfg.ReputationHighDrawdown))
		bundle = store.Bundle{
			Strategies: store.NewFakeStore(store.SeedRows(), nil),
			Events:     store.NewFakeEventStore(),
			Tokens:     fakeTokens,
			Creators:   fakeTokens.(store.CreatorStore),
			Pinger:     fakeTokens.(store.Pinger),
		}
	}
	defer cleanup()

	hub := ws.NewHub()
	go hub.Run(ctx)

	// health registry (System Health, Task 4) — "reg" ismi zaten ingest.NewRegistry() için
	// kullanılıyor (aşağıda), bu yüzden ayrı isim: healthReg.
	healthReg := health.NewRegistry()
	startedAt := time.Now()

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

	// token safety scorer (2a) — arka plan; Helius key + market gerekli.
	// authorities (getAccountInfo) STANDART Solana RPC metodu → Helius free-tier 429'da
	// SOLANA_RPC_URL ile güvenilir genel sağlayıcıya yönlendirilir (creatorfill deseni);
	// boşsa Helius'a düşer. holders (getTokenAccounts) DAS-özel → Helius'ta KALIR (429'da
	// degraded ama observability fix'i ile görünür; DAS-sağlayıcı ertelendi).
	if cfg.SafetyEnabled && bundle.Tokens != nil && rpcURL != "" {
		provider := safety.NewHeliusProvider(
			ingest.NewHeliusAuthorities(preferRPC(cfg.SolanaRPCURL, rpcURL)),
			ingest.NewHeliusHolders(rpcURL), cfg.SafetyHoldersCap)
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

	// REST creator backfill (WS blokörü baypas) — arka plan; standart RPC gerekli.
	// Resolver YALNIZ standart getSignaturesForAddress+getTransaction kullanır (Helius
	// DAS'a bağımlı DEĞİL) → Helius free-tier bu metodu bloke ederse SOLANA_RPC_URL ile
	// alternatif genel sağlayıcıya yönlendirilir; boşsa Helius rpcURL'e düşer. WS + DAS
	// holders Helius'ta kalır (safety authorities de SOLANA_RPC_URL'e yönlendirildi, bkz üstte).
	creatorFillRPC := preferRPC(cfg.SolanaRPCURL, rpcURL)

	// health kayıtları (System Health, Task 4) — her worker'ın enabled/interval'ı burada
	// tek yerde toplanır; tüm referans değişkenler (rpcURL, creatorFillRPC) bu noktada mevcut.
	healthReg.Register(health.WorkerIngestWS, cfg.HeliusAPIKey != "", 0) // event-driven
	healthReg.Register(health.WorkerMarketDisc, cfg.MarketEnabled, time.Duration(cfg.DiscoverInterval)*time.Second)
	healthReg.Register(health.WorkerMarketEnrich, cfg.MarketEnabled, time.Duration(cfg.EnrichInterval)*time.Second)
	healthReg.Register(health.WorkerSafety, cfg.SafetyEnabled && rpcURL != "", time.Duration(cfg.SafetyIntervalSec)*time.Second)
	healthReg.Register(health.WorkerOutcome, cfg.OutcomeEnabled, time.Duration(cfg.OutcomeIntervalSec)*time.Second)
	healthReg.Register(health.WorkerCreatorFill, cfg.CreatorFillEnabled && creatorFillRPC != "", time.Duration(cfg.CreatorFillIntervalSec)*time.Second)
	healthReg.Register(health.WorkerFunder, cfg.WalletGraphEnabled && creatorFillRPC != "", time.Duration(cfg.FunderResolveIntervalSec)*time.Second)
	healthReg.Register(health.WorkerReputation, cfg.ReputationEnabled, time.Duration(cfg.ReputationIntervalSec)*time.Second)
	healthReg.Register(health.WorkerManipulation, cfg.ManipulationEnabled, time.Duration(cfg.ManipulationIntervalSec)*time.Second)
	healthReg.Register(health.WorkerOpportunity, cfg.OpportunityEnabled, time.Duration(cfg.OpportunityIntervalSec)*time.Second)

	gates := map[string]bool{
		"MARKET_ENABLED":       cfg.MarketEnabled,
		"SAFETY_ENABLED":       cfg.SafetyEnabled,
		"OUTCOME_ENABLED":      cfg.OutcomeEnabled,
		"CREATORFILL_ENABLED":  cfg.CreatorFillEnabled,
		"WALLET_GRAPH_ENABLED": cfg.WalletGraphEnabled,
		"REPUTATION_ENABLED":   cfg.ReputationEnabled,
		"MANIPULATION_ENABLED": cfg.ManipulationEnabled,
		"OPPORTUNITY_ENABLED":  cfg.OpportunityEnabled,
	}

	// Paylaşılan hız sınırlayıcı: creatorfill + funder worker'ları AYNI creatorFillRPC
	// (SOLANA_RPC_URL) uç noktasına vurur — iki ayrı tam-hızlı limiter toplam QPS'i
	// ikiye katlar ve zaten 429'a yatkın sağlayıcıyı daha da zorlar. Tek instance ile
	// ikisi arasında toplam SOLANA_RPC_URL baskısı sınırlanır (deploy-tunable
	// CREATORFILL_RATE_PER_MIN/BURST).
	sharedRPCLimiter := rate.NewLimiter(rate.Limit(float64(cfg.CreatorFillRatePerMin)/60.0), cfg.CreatorFillBurst)
	if cfg.CreatorFillEnabled && bundle.Tokens != nil && creatorFillRPC != "" {
		resolver := ingest.NewCreatorResolver(creatorFillRPC, cfg.CreatorFillMaxSigPages, ingest.WithLimiter(sharedRPCLimiter))
		cw := creatorfill.NewWorker(creatorfill.WorkerDeps{
			Store: bundle.Tokens, Resolver: resolver,
			Interval: time.Duration(cfg.CreatorFillIntervalSec) * time.Second, Limit: cfg.CreatorFillLimit, Logger: logger,
		})
		go cw.Run(ctx)
	} else if cfg.CreatorFillEnabled {
		logger.Warn("CREATORFILL: RPC (Helius key veya SOLANA_RPC_URL) veya token store yok — backfill başlamayacak")
	}

	// funder resolver worker (2e-1) — arka plan; creator cüzdanlarının funder'ını yakalar (bundler tespiti).
	// creatorfill ile AYNI RPC'yi (creatorFillRPC) VE aynı sharedRPCLimiter'ı kullanır
	// (toplam QPS'i sınırlamak için — bkz üstteki sharedRPCLimiter yorumu).
	if cfg.WalletGraphEnabled && bundle.Tokens != nil && creatorFillRPC != "" {
		fres := walletgraph.NewFunderResolver(creatorFillRPC, cfg.CreatorFillMaxSigPages, walletgraph.WithLimiter(sharedRPCLimiter))
		fw := walletgraph.NewWorker(walletgraph.WorkerDeps{
			Store: bundle.Tokens, Resolver: fres,
			Interval: time.Duration(cfg.FunderResolveIntervalSec) * time.Second, Limit: cfg.FunderResolveLimit, Logger: logger,
		})
		go fw.Run(ctx)
	}

	// creator reputation scorer (2b-2b) — arka plan; saf DB (RPC YOK)
	if cfg.ReputationEnabled && bundle.Tokens != nil {
		rw := reputation.NewWorker(reputation.WorkerDeps{
			Store: bundle.Tokens,
			Thresholds: reputation.Thresholds{
				MinResolved: cfg.ReputationMinResolved,
				WRug:        cfg.ReputationWRug, WFail: cfg.ReputationWFail, WGrad: cfg.ReputationWGrad,
			},
			Interval: time.Duration(cfg.ReputationIntervalSec) * time.Second, Limit: cfg.ReputationLimit, Logger: logger,
		})
		go rw.Run(ctx)
	}

	// manipulation risk scorer (2c) — arka plan; saf DB (RPC YOK)
	if cfg.ManipulationEnabled && bundle.Tokens != nil {
		mw := manipulation.NewWorker(manipulation.WorkerDeps{
			Store: bundle.Tokens,
			Thresholds: manipulation.Thresholds{
				MinTxns: cfg.ManipulationMinTxns, ConfTxns: cfg.ManipulationConfTxns,
				WImbalance: cfg.ManipulationWImbalance, WWash: cfg.ManipulationWWash,
				WVolume: cfg.ManipulationWVolume, WCreator: cfg.ManipulationWCreator,
				WashMin: cfg.ManipulationWashMin, WashMax: cfg.ManipulationWashMax,
				VolMin: cfg.ManipulationVolMin, VolMax: cfg.ManipulationVolMax,
			},
			Interval: time.Duration(cfg.ManipulationIntervalSec) * time.Second, Limit: cfg.ManipulationLimit, Logger: logger,
		})
		go mw.Run(ctx)
	}

	// opportunity kompozit scorer (2d) — arka plan; saf DB (RPC YOK)
	if cfg.OpportunityEnabled && bundle.Tokens != nil {
		ow := opportunity.NewWorker(opportunity.WorkerDeps{
			Store:    bundle.Tokens,
			Interval: time.Duration(cfg.OpportunityIntervalSec) * time.Second,
			Limit:    cfg.OpportunityLimit, Logger: logger,
		})
		go ow.Run(ctx)
	}

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: api.NewRouter(api.RouterDeps{
			Strategies: bundle.Strategies, Events: bundle.Events, Tokens: bundle.Tokens,
			Hub: hub, CORSOrigin: cfg.CORSOrigin, EventsWindow: cfg.EventsWindow,
			TokenDetail:           detailSvc,
			TokenDetailTimeout:    time.Duration(cfg.TokenDetailTimeoutSec) * time.Second,
			Creators:              bundle.Creators,
			CreatorsLimit:         cfg.CreatorsListLimit,
			WalletGraphMinCluster: cfg.WalletGraphMinCluster,
			WalletGraphMaxDegree:  cfg.WalletGraphMaxDegree,
			Health:                healthReg,
			Pinger:                bundle.Pinger,
			Gates:                 gates,
			Version:               cfg.Version,
			StartedAt:             startedAt,
			WSClientCount:         hub.ClientCount,
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

// preferRPC, override (SOLANA_RPC_URL) boş değilse onu, boşsa fallback'i (Helius rpcURL)
// döndürür. Helius free-tier 429'ında standart-RPC metodlarını (getAccountInfo /
// getSignaturesForAddress) güvenilir genel sağlayıcıya yönlendiren ortak seçim (DRY:
// creatorfill + safety authorities aynı deseni paylaşır).
func preferRPC(override, fallback string) string {
	if override != "" {
		return override
	}
	return fallback
}

// noopHolders, Helius key yokken holders'ı 0 döndürür (dürüst — sayı yok).
type noopHolders struct{}

func (noopHolders) HoldersCount(context.Context, string, int) (int, bool, error) {
	return 0, false, nil
}
