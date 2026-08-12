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
	SolanaRPCURL string // creatorfill resolver için alternatif genel RPC (Helius free-tier getSignaturesForAddress'i bloke ederse); boşsa Helius'a düşer
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

	SafetyEnabled     bool
	SafetyIntervalSec int
	SafetyLimit       int
	SafetyHoldersCap  int

	CreatorsListLimit int

	OutcomeEnabled        bool
	OutcomeIntervalSec    int
	OutcomeLimit          int
	OutcomeRugLiqRatio    float64
	OutcomeGraduationMcap float64
	OutcomeDumpedDrawdown float64
	OutcomeDeadVol        float64
	OutcomeMinLiqFloor    float64
	OutcomeDeadAgeSec     int

	CreatorFillEnabled     bool
	CreatorFillIntervalSec int
	CreatorFillLimit       int
	CreatorFillMaxSigPages int
	CreatorFillRatePerMin  int // Helius RPC paylaşılan istek bütçesi (istek/dk) — 429 burst'ünü önler
	CreatorFillBurst       int // token-bucket burst kapasitesi

	ReputationEnabled      bool
	ReputationIntervalSec  int
	ReputationLimit        int
	ReputationMinResolved  int
	ReputationWRug         float64
	ReputationWFail        float64
	ReputationWGrad        float64
	ReputationHighDrawdown float64
}

func Load() Config {
	return Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CORSOrigin:   os.Getenv("CORS_ORIGIN"),
		HeliusAPIKey: os.Getenv("HELIUS_API_KEY"),
		SolanaRPCURL: os.Getenv("SOLANA_RPC_URL"),
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

		SafetyEnabled:     getenvBool("SAFETY_ENABLED", true),
		SafetyIntervalSec: getenvInt("SAFETY_INTERVAL_SEC", 60),
		SafetyLimit:       getenvInt("SAFETY_LIMIT", 40),
		SafetyHoldersCap:  getenvInt("SAFETY_HOLDERS_CAP", 5000),

		CreatorsListLimit: getenvInt("CREATORS_LIST_LIMIT", 100),

		OutcomeEnabled:        getenvBool("OUTCOME_ENABLED", true),
		OutcomeIntervalSec:    getenvInt("OUTCOME_INTERVAL_SEC", 60),
		OutcomeLimit:          getenvInt("OUTCOME_LIMIT", 60),
		OutcomeRugLiqRatio:    getenvFloat("OUTCOME_RUG_LIQ_RATIO", 0.10),
		OutcomeGraduationMcap: getenvFloat("OUTCOME_GRADUATION_MCAP", 69000),
		OutcomeDumpedDrawdown: getenvFloat("OUTCOME_DUMPED_DRAWDOWN", 80),
		OutcomeDeadVol:        getenvFloat("OUTCOME_DEAD_VOL", 100),
		OutcomeMinLiqFloor:    getenvFloat("OUTCOME_MIN_LIQ_FLOOR", 500),
		OutcomeDeadAgeSec:     getenvInt("OUTCOME_DEAD_AGE_SEC", 86400),

		CreatorFillEnabled:     getenvBool("CREATORFILL_ENABLED", true),
		CreatorFillIntervalSec: getenvInt("CREATORFILL_INTERVAL_SEC", 30),
		CreatorFillLimit:       getenvInt("CREATORFILL_LIMIT", 20),
		CreatorFillMaxSigPages: getenvInt("CREATORFILL_MAX_SIG_PAGES", 3),
		CreatorFillRatePerMin:  getenvInt("CREATORFILL_RATE_PER_MIN", 120),
		CreatorFillBurst:       getenvInt("CREATORFILL_BURST", 2),

		ReputationEnabled:      getenvBool("REPUTATION_ENABLED", true),
		ReputationIntervalSec:  getenvInt("REPUTATION_INTERVAL_SEC", 60),
		ReputationLimit:        getenvInt("REPUTATION_LIMIT", 60),
		ReputationMinResolved:  getenvInt("REPUTATION_MIN_RESOLVED", 5),
		ReputationWRug:         getenvFloat("REPUTATION_W_RUG", 50),
		ReputationWFail:        getenvFloat("REPUTATION_W_FAIL", 20),
		ReputationWGrad:        getenvFloat("REPUTATION_W_GRAD", 40),
		ReputationHighDrawdown: getenvFloat("REPUTATION_HIGH_DRAWDOWN", 80),
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

func getenvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}
