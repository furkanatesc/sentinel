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

	GeckoBaseURL     string
	MarketEnabled    bool
	DiscoverInterval int // saniye
	EnrichInterval   int // saniye
	EnrichLimit      int
	GeckoRatePerMin  int // GeckoTerminal paylaşılan istek bütçesi (istek/dk)
	GeckoBurst       int // token-bucket burst kapasitesi

	TokenDetailCacheSec   int
	OHLCVLimit            int
	HoldersCap            int
	TokenDetailTimeoutSec int // /api/token isteği için üst sınır (limiter kuyruğunda süresiz bekleme olmasın)
}

func Load() Config {
	return Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CORSOrigin:   os.Getenv("CORS_ORIGIN"),
		HeliusAPIKey: os.Getenv("HELIUS_API_KEY"),
		EventsWindow: getenvInt("EVENTS_WINDOW", 200),

		GeckoBaseURL:     getenv("GECKOTERMINAL_BASE_URL", "https://api.geckoterminal.com/api/v2"),
		MarketEnabled:    getenvBool("MARKET_ENABLED", true),
		DiscoverInterval: getenvInt("MARKET_DISCOVER_INTERVAL_SEC", 30),
		EnrichInterval:   getenvInt("MARKET_ENRICH_INTERVAL_SEC", 30),
		EnrichLimit:      getenvInt("MARKET_ENRICH_LIMIT", 60),
		GeckoRatePerMin:  getenvInt("GECKOTERMINAL_RATE_PER_MIN", 25),
		GeckoBurst:       getenvInt("GECKOTERMINAL_BURST", 2),

		TokenDetailCacheSec:   getenvInt("TOKEN_DETAIL_CACHE_SEC", 20),
		OHLCVLimit:            getenvInt("OHLCV_LIMIT", 200),
		HoldersCap:            getenvInt("HOLDERS_CAP", 5000),
		TokenDetailTimeoutSec: getenvInt("TOKEN_DETAIL_TIMEOUT_SEC", 8),
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

func getenvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
