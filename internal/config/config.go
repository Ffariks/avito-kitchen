package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	// HTTPClientTimeoutSec — таймаут исходящих запросов к сервисам заведений.
	HTTPClientTimeoutSec int
}

func Load() Config {
	return Config{
		HTTPAddr:             getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://avito:avito@localhost:5432/avito_kitchen?sslmode=disable"),
		HTTPClientTimeoutSec: 5,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
