package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("CORS_ORIGIN", "")
	t.Setenv("HELIUS_API_KEY", "")
	t.Setenv("EVENTS_WINDOW", "")

	cfg := Load()
	if cfg.EventsWindow != 200 {
		t.Fatalf("EventsWindow = %d, want 200", cfg.EventsWindow)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
}

func TestLoadMarketDefaults(t *testing.T) {
	t.Setenv("MARKET_ENABLED", "")
	t.Setenv("GECKOTERMINAL_BASE_URL", "")
	c := Load()
	if !c.MarketEnabled {
		t.Fatal("MARKET_ENABLED default true olmalı")
	}
	if c.GeckoBaseURL == "" || c.DiscoverInterval != 30 || c.EnrichLimit != 60 {
		t.Fatalf("market default'ları yanlış: %+v", c)
	}
}

func TestGetenvBool(t *testing.T) {
	t.Setenv("X_FLAG", "false")
	if getenvBool("X_FLAG", true) {
		t.Fatal("getenvBool 'false' okumalı")
	}
}
