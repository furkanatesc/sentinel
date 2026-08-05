package store

import "context"

// EventRow, frontend FeedEvent (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
// Signature ve Slot, dedup+persist için taşınır ama frontend kontratında olmadığı için JSON'a çıkmaz.
type EventRow struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Symbol          string  `json:"symbol"`
	Mint            string  `json:"mint"`
	Launchpad       string  `json:"launchpad"`
	DEX             string  `json:"dex"`
	Liquidity       float64 `json:"liquidity"`
	CreatorScore    float64 `json:"creatorScore"`
	RiskLevel       string  `json:"riskLevel"`
	TokenAgeSeconds int64   `json:"tokenAgeSeconds"`
	Volume5m        float64 `json:"volume5m"`
	HolderGrowthPct float64 `json:"holderGrowthPct"`
	Severity        string  `json:"severity"`
	Detail          string  `json:"detail"`
	Time            string  `json:"time"`
	Ts              int64   `json:"ts"`
	Watchlisted     bool    `json:"watchlisted"`
	Signature       string  `json:"-"`
	Slot            uint64  `json:"-"`
}

// EventStore, append-only olay kaynağıdır (DIP).
type EventStore interface {
	InsertEvent(ctx context.Context, e EventRow) error
	RecentEvents(ctx context.Context, limit int) ([]EventRow, error)
}
