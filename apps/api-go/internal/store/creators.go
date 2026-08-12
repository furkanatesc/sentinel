package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// reputationHighDrawdown, deriveRiskFlags'in "Yüksek düşüş" bayrağını tetiklediği eşiktir (%).
const reputationHighDrawdown = 80

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

// deriveRiskFlags, per-token history item'ın outcome/liquidityStatus/maxDrawdown'undan
// insan-okunur risk bayrakları türetir (2b-2b). Non-nil döner (boş dizi, nil değil — JSON []).
func deriveRiskFlags(outcome, liquidityStatus string, maxDrawdown float64) []string {
	flags := []string{}
	switch outcome {
	case "rug":
		flags = append(flags, "Rug çekildi")
	case "dumped":
		flags = append(flags, "Dump edildi")
	case "dead":
		flags = append(flags, "Ölü (hacim yok)")
	}
	if liquidityStatus == "removed" {
		flags = append(flags, "Likidite çekildi")
	}
	if maxDrawdown >= reputationHighDrawdown {
		flags = append(flags, fmt.Sprintf("Yüksek düşüş (%%%.0f)", maxDrawdown))
	}
	return flags
}

// newHistoryItem, gerçek piyasa + outcome alanlarını doldurur; creatorSellPct nötr (→ 2c trade-flow).
func newHistoryItem(mint, symbol string, firstSeenTs int64, currentMcap, peakMcap, maxDrawdown float64, outcome, liquidityStatus string) CreatorTokenHistoryItem {
	return CreatorTokenHistoryItem{
		ID: mint, Symbol: symbol, Mint: mint,
		CreatedAt:        time.Unix(firstSeenTs, 0).UTC().Format(time.RFC3339),
		CurrentMarketCap: currentMcap,
		PeakMarketCap:    peakMcap,
		MaxDrawdownPct:   maxDrawdown,
		LiquidityStatus:  liquidityStatus,
		Outcome:          outcome,
		RiskFlags:        deriveRiskFlags(outcome, liquidityStatus, maxDrawdown),
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
	// LEFT JOIN creators: skorlanmamış creator düşmez (COALESCE nötr) — bkz. TestCreatorsListIncludesUnscored.
	const q = `SELECT t.creator, COUNT(*) AS total,
		COALESCE(c.reputation_score,0), COALESCE(c.risk_level,'medium'),
		COALESCE(c.active_tokens,0), COALESCE(c.rugged_tokens,0), COALESCE(c.success_rate_pct,0)
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		WHERE t.creator <> '' GROUP BY t.creator, c.reputation_score, c.risk_level,
			c.active_tokens, c.rugged_tokens, c.success_rate_pct
		ORDER BY total DESC, MIN(t.first_seen_ts) ASC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorRow, 0, limit)
	for rows.Next() {
		var c CreatorRow
		if err := rows.Scan(&c.Address, &c.TotalTokens, &c.ReputationScore, &c.RiskLevel,
			&c.ActiveTokens, &c.RuggedTokens, &c.SuccessRatePct); err != nil {
			return nil, err
		}
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
		`SELECT mint, symbol, first_seen_ts, market_cap_usd, peak_market_cap, max_drawdown_pct, outcome, liquidity_status
		 FROM tokens WHERE creator=$1 ORDER BY first_seen_ts DESC`, address)
	if err != nil {
		return CreatorProfile{}, false, err
	}
	defer rows.Close()
	history := make([]CreatorTokenHistoryItem, 0, total)
	for rows.Next() {
		var mint, symbol, outcome, liqStatus string
		var ts int64
		var mcap, peakMcap, drawdown float64
		if err := rows.Scan(&mint, &symbol, &ts, &mcap, &peakMcap, &drawdown, &outcome, &liqStatus); err != nil {
			return CreatorProfile{}, false, err
		}
		history = append(history, newHistoryItem(mint, symbol, ts, mcap, peakMcap, drawdown, outcome, liqStatus))
	}
	if err := rows.Err(); err != nil {
		return CreatorProfile{}, false, err
	}

	prof := buildCreatorProfile(address, firstSeen, total, history)
	// creators satırını oku (yoksa nötr — Worker henüz skorlamamış, sql.ErrNoRows beklenen durum).
	var rep CreatorReputation
	var breakdownJSON string
	repErr := p.db.QueryRowContext(ctx,
		`SELECT reputation_score, confidence, risk_level, breakdown, active_tokens, rugged_tokens,
			graduated_tokens, avg_peak_market_cap, avg_lifetime_hours, success_rate_pct
		 FROM creators WHERE address=$1`, address).
		Scan(&rep.Score, &rep.Confidence, &rep.RiskLevel, &breakdownJSON, &rep.ActiveTokens,
			&rep.RuggedTokens, &rep.GraduatedTokens, &rep.AvgPeakMarketCap, &rep.AvgLifetimeHours, &rep.SuccessRatePct)
	if repErr != nil && !errors.Is(repErr, sql.ErrNoRows) {
		return CreatorProfile{}, false, repErr
	}
	if repErr == nil {
		prof.Reputation = ScoreDetail{Key: "creatorReputation", Value: rep.Score, Confidence: rep.Confidence,
			Breakdown: parseBreakdownJSON(breakdownJSON)}
		prof.RiskLevel = rep.RiskLevel
		prof.Metrics = CreatorMetrics{
			TotalTokens: total, ActiveTokens: rep.ActiveTokens, RuggedTokens: rep.RuggedTokens,
			AvgPeakMarketCap: rep.AvgPeakMarketCap, AvgLifetimeHours: rep.AvgLifetimeHours, SuccessRatePct: rep.SuccessRatePct,
		}
	}
	return prof, true, nil
}
