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

	"github.com/farik/avito-kitchen/internal/config"
	"github.com/farik/avito-kitchen/internal/handler"
	"github.com/farik/avito-kitchen/internal/httpclient"
	"github.com/farik/avito-kitchen/internal/httpserver"
	"github.com/farik/avito-kitchen/internal/repository"
	"github.com/farik/avito-kitchen/internal/service"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	restaurantRepo := repository.NewRestaurantRepo(pool)
	menuRepo := repository.NewMenuRepo(pool)
	orderRepo := repository.NewOrderRepo(pool)

	webhookClient := httpclient.NewRestaurantWebhookClient(time.Duration(cfg.HTTPClientTimeoutSec) * time.Second)

	catalogSvc := service.NewCatalogService(restaurantRepo, menuRepo)
	orderSvc := service.NewOrderService(orderRepo, menuRepo, restaurantRepo, webhookClient, log)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpserver.Health)
	handler.NewCatalogHandler(catalogSvc, log).Register(mux)
	handler.NewOrderHandler(orderSvc, log).Register(mux)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httpserver.WithLogging(log, mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("starting http server", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
