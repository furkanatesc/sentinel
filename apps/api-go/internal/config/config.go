package config

import (
	"os"
	"strconv"
)

// Config, servisin env ile yapılandırmasıdır (SRP: yalnız yapılandırma).
type Config struct {
	Port         string
	DatabaseURL  string
	CORSOrigin   string
	HeliusAPIKey string
	EventsWindow int
}

func Load() Config {
	return Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CORSOrigin:   os.Getenv("CORS_ORIGIN"),
		HeliusAPIKey: os.Getenv("HELIUS_API_KEY"),
		EventsWindow: getenvInt("EVENTS_WINDOW", 200),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
