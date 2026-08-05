package store

import "context"

// TokenRow, frontend TokenRow (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
type TokenRow struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Mint         string    `json:"mint"`
	AgeSeconds   int64     `json:"ageSeconds"`
	Price        float64   `json:"price"`
	Liquidity    float64   `json:"liquidity"`
	Vol5m        float64   `json:"vol5m"`
	Holders      int       `json:"holders"`
	CreatorScore float64   `json:"creatorScore"`
	SafetyScore  float64   `json:"safetyScore"`
	Momentum     float64   `json:"momentum"`
	Spark        []float64 `json:"spark"`
	Signal       *string   `json:"signal"` // "buy"|"watch"|"avoid"|null
	Watchlisted  bool      `json:"watchlisted"`
}

// TokenStore, mint-PK token kaynağıdır (upsert; DIP).
type TokenStore interface {
	UpsertToken(ctx context.Context, t TokenRow) error
	RecentTokens(ctx context.Context, limit int) ([]TokenRow, error)
}
