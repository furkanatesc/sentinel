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

	st := store.NewFakeStore(store.SeedRows(), nil)
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
