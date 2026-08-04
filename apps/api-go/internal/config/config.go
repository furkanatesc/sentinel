package config

import "os"

// Config, servisin env ile yapılandırmasıdır (SRP: yalnız yapılandırma).
type Config struct {
	Port        string
	DatabaseURL string
	CORSOrigin  string
}

func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		CORSOrigin:  os.Getenv("CORS_ORIGIN"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
