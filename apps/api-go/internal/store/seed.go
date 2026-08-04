package store

// SeedRows, frontend mock.ts strateji verisinden türetilmiş deterministik 6 satırdır
// (aynı id'ler + aynı türetilmiş metrikler → mock|http görsel eşitliği).
func SeedRows() []StrategyRow {
	return []StrategyRow{
		{ID: "momentum-scalp", Name: "Momentum Scalp", Status: "live", Timeframe: "1-5 dk", WinRatePct: 63, ProfitFactor: 1.8, MaxDrawdownPct: 16, TotalTrades: 298, NetPnlSol: 338, LastSignal: "43 dk önce"},
		{ID: "safe-graduation", Name: "Güvenli Graduation", Status: "paper", Timeframe: "15-60 dk", WinRatePct: 55, ProfitFactor: 1.5, MaxDrawdownPct: 13, TotalTrades: 370, NetPnlSol: -90, LastSignal: "56 dk önce"},
		{ID: "creator-reputation", Name: "Creator İtibar Takibi", Status: "shadow", Timeframe: "5-30 dk", WinRatePct: 61, ProfitFactor: 3.1, MaxDrawdownPct: 29, TotalTrades: 336, NetPnlSol: 276, LastSignal: "9 dk önce"},
		{ID: "liquidity-breakout", Name: "Likidite Kırılımı", Status: "backtesting", Timeframe: "1-10 dk", WinRatePct: 61, ProfitFactor: 3.1, MaxDrawdownPct: 29, TotalTrades: 336, NetPnlSol: 276, LastSignal: "9 dk önce"},
		{ID: "anti-rug-filter", Name: "Anti-Rug Filtre", Status: "paused", Timeframe: "10-45 dk", WinRatePct: 63, ProfitFactor: 3.3, MaxDrawdownPct: 31, TotalTrades: 338, NetPnlSol: 378, LastSignal: "24 dk önce"},
		{ID: "legacy-sniper", Name: "Eski Sniper v1", Status: "archived", Timeframe: "0-2 dk", WinRatePct: 56, ProfitFactor: 1.6, MaxDrawdownPct: 14, TotalTrades: 171, NetPnlSol: 211, LastSignal: "34 dk önce"},
	}
}
