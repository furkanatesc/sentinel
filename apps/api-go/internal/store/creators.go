package store

import (
	"context"
	"time"
)

// CreatorRow, frontend CreatorRow (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
// 2b-1: Address/TotalTokens gerçek; kalanlar nötr placeholder → 2b-2 (itibar skoru + outcome).
type CreatorRow struct {
	Address         string  `json:"address"`
	Label           string  `json:"label,omitempty"`
	ReputationScore float64 `json:"reputationScore"`
	RiskLevel       string  `json:"riskLevel"`
	TotalTokens     int     `json:"totalTokens"`
	ActiveTokens    int     `json:"activeTokens"`
	RuggedTokens    int     `json:"ruggedTokens"`
	SuccessRatePct  float64 `json:"successRatePct"`
	RealizedPnlSol  float64 `json:"realizedPnlSol"`
}

// CreatorTokenHistoryItem — 2b-1: ID/Symbol/Mint/CreatedAt/CurrentMarketCap gerçek; kalanlar nötr → 2b-2.
type CreatorTokenHistoryItem struct {
	ID               string   `json:"id"`
	Symbol           string   `json:"symbol"`
	Mint             string   `json:"mint"`
	CreatedAt        string   `json:"createdAt"`
	PeakMarketCap    float64  `json:"peakMarketCap"`
	CurrentMarketCap float64  `json:"currentMarketCap"`
	MaxDrawdownPct   float64  `json:"maxDrawdownPct"`
	LiquidityStatus  string   `json:"liquidityStatus"`
	CreatorSellPct   float64  `json:"creatorSellPct"`
	Outcome          string   `json:"outcome"`
	RiskFlags        []string `json:"riskFlags"`
}

type CreatorBehavior struct {
	DeployFrequency      string   `json:"deployFrequency"`
	AvgFirstSellMinutes  float64  `json:"avgFirstSellMinutes"`
	RepeatedFunders      []string `json:"repeatedFunders"`
	SimilarMetadata      bool     `json:"similarMetadata"`
	SameSocial           bool     `json:"sameSocial"`
	SameLiquidityPattern bool     `json:"sameLiquidityPattern"`
}

type CreatorMetrics struct {
	TotalTokens         int     `json:"totalTokens"`
	ActiveTokens        int     `json:"activeTokens"`
	RuggedTokens        int     `json:"ruggedTokens"`
	AvgLifetimeHours    float64 `json:"avgLifetimeHours"`
	AvgPeakMarketCap    float64 `json:"avgPeakMarketCap"`
	RealizedPnlSol      float64 `json:"realizedPnlSol"`
	SuccessRatePct      float64 `json:"successRatePct"`
	AvgFirstSellMinutes float64 `json:"avgFirstSellMinutes"`
}

type CreatorProfile struct {
	Address       string                    `json:"address"`
	Label         string                    `json:"label,omitempty"`
	WalletAgeDays int                       `json:"walletAgeDays"`
	FirstSeen     string                    `json:"firstSeen"`
	Reputation    ScoreDetail               `json:"reputation"`
	RiskLevel     string                    `json:"riskLevel"`
	Metrics       CreatorMetrics            `json:"metrics"`
	History       []CreatorTokenHistoryItem `json:"history"`
	Behavior      CreatorBehavior           `json:"behavior"`
}

// neutralReputation, 2b-2'ye ait itibar skorunun nötr placeholder'ıdır (sahte skor değil).
func neutralReputation() ScoreDetail {
	return ScoreDetail{Key: "creatorReputation", Value: 0, Confidence: 0, UpdatedAt: "", Breakdown: []ScoreBreakdownItem{}}
}

// neutralBehavior, boş/false davranış (→ 2b-2).
func neutralBehavior() CreatorBehavior {
	return CreatorBehavior{RepeatedFunders: []string{}}
}

// newHistoryItem, gerçek alanları doldurur; kalanı nötr placeholder (→ 2b-2).
func newHistoryItem(mint, symbol string, firstSeenTs int64, currentMcap float64) CreatorTokenHistoryItem {
	return CreatorTokenHistoryItem{
		ID: mint, Symbol: symbol, Mint: mint,
		CreatedAt:        time.Unix(firstSeenTs, 0).UTC().Format(time.RFC3339),
		CurrentMarketCap: currentMcap,
		LiquidityStatus:  "unlocked", // nötr (geçerli enum) → 2b-2
		Outcome:          "active",   // nötr (geçerli enum) → 2b-2
		RiskFlags:        []string{},
	}
}

// buildCreatorProfile, kimlik + gerçek totalTokens + history; kalan metrik/davranış nötr (→ 2b-2).
func buildCreatorProfile(address string, firstSeenTs int64, total int, history []CreatorTokenHistoryItem) CreatorProfile {
	if history == nil {
		history = []CreatorTokenHistoryItem{}
	}
	return CreatorProfile{
		Address:    address,
		FirstSeen:  time.Unix(firstSeenTs, 0).UTC().Format(time.RFC3339),
		Reputation: neutralReputation(),
		RiskLevel:  "medium",
		Metrics:    CreatorMetrics{TotalTokens: total},
		History:    history,
		Behavior:   neutralBehavior(),
	}
}

// CreatorStore, creator kimlik + agrega kaynağıdır (ISP: dar okuma arayüzü; DIP).
type CreatorStore interface {
	Creators(ctx context.Context, limit int) ([]CreatorRow, error)
	CreatorDetail(ctx context.Context, address string) (CreatorProfile, bool, error)
}

func (p *postgresStore) Creators(ctx context.Context, limit int) ([]CreatorRow, error) {
	const q = `SELECT creator, COUNT(*) AS total FROM tokens
		WHERE creator <> '' GROUP BY creator
		ORDER BY total DESC, MIN(first_seen_ts) ASC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorRow, 0, limit)
	for rows.Next() {
		var c CreatorRow
		if err := rows.Scan(&c.Address, &c.TotalTokens); err != nil {
			return nil, err
		}
		c.RiskLevel = "medium" // nötr placeholder → 2b-2
		out = append(out, c)
	}
	return out, rows.Err()
}

func (p *postgresStore) CreatorDetail(ctx context.Context, address string) (CreatorProfile, bool, error) {
	// Kimlik + agrega: bu creator'ın token sayısı + ilk görülme.
	var total int
	var firstSeen int64
	err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(MIN(first_seen_ts), 0) FROM tokens WHERE creator=$1`, address).
		Scan(&total, &firstSeen)
	if err != nil {
		return CreatorProfile{}, false, err
	}
	if total == 0 {
		return CreatorProfile{}, false, nil
	}
	// Token geçmişi (en yeni önce).
	rows, err := p.db.QueryContext(ctx,
		`SELECT mint, symbol, first_seen_ts, market_cap_usd FROM tokens
		 WHERE creator=$1 ORDER BY first_seen_ts DESC`, address)
	if err != nil {
		return CreatorProfile{}, false, err
	}
	defer rows.Close()
	history := make([]CreatorTokenHistoryItem, 0, total)
	for rows.Next() {
		var mint, symbol string
		var ts int64
		var mcap float64
		if err := rows.Scan(&mint, &symbol, &ts, &mcap); err != nil {
			return CreatorProfile{}, false, err
		}
		history = append(history, newHistoryItem(mint, symbol, ts, mcap))
	}
	if err := rows.Err(); err != nil {
		return CreatorProfile{}, false, err
	}
	return buildCreatorProfile(address, firstSeen, total, history), true, nil
}
