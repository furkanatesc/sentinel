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

	"github.com/furkanatesc/sentinel/apps/api-go/internal/api"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/config"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	var st store.StrategyStore
	var cleanup func() error = func() error { return nil }
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		pst, cl, err := store.OpenPostgres(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			logger.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		st, cleanup = pst, cl
	} else {
		logger.Warn("DATABASE_URL yok — in-memory fake store kullanılıyor")
		st = store.NewFakeStore(store.SeedRows(), nil)
	}
	defer cleanup()

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewRouter(st, cfg.CORSOrigin),
	}

	go func() {
		logger.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	logger.Info("stopped")
}
