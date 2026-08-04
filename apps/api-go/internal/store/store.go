package store

import "context"

// StrategyRow, frontend StrategyRow (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
type StrategyRow struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Timeframe      string  `json:"timeframe"`
	WinRatePct     float64 `json:"winRatePct"`
	ProfitFactor   float64 `json:"profitFactor"`
	MaxDrawdownPct float64 `json:"maxDrawdownPct"`
	TotalTrades    int     `json:"totalTrades"`
	NetPnlSol      float64 `json:"netPnlSol"`
	LastSignal     string  `json:"lastSignal"`
}

// StrategyStore, strateji listesi kaynağıdır (DIP: handler bu interface'e bağımlı).
type StrategyStore interface {
	List(ctx context.Context) ([]StrategyRow, error)
}
