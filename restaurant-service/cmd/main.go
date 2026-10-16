package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/farik/avito-kitchen/restaurant-service/internal/client"
	"github.com/farik/avito-kitchen/restaurant-service/internal/handler"
	"github.com/farik/avito-kitchen/restaurant-service/internal/store"
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := getEnv("HTTP_ADDR", ":9090")
	platformBaseURL := getEnv("PLATFORM_BASE_URL", "http://api:8080")

	s := store.New()
	platformClient := client.NewPlatformClient(platformBaseURL, 5*time.Second)
	ordersHandler := handler.NewOrdersHandler(s, platformClient, log, 3*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ordersHandler.Register(mux)

	log.Info("starting restaurant-service", "addr", addr, "platform_base_url", platformBaseURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
