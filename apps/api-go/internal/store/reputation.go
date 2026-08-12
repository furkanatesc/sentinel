package store

import (
	"context"
	"encoding/json"
)

// CreatorAgg, bir creator'ın tokenlarının outcome agregasıdır (2b-2b scorer girdisi).
type CreatorAgg struct {
	Address                                     string
	Total, Active, Rug, Dumped, Dead, Graduated int
	AvgPeakMarketCap, AvgLifetimeHours          float64
}

// CreatorReputation, hesaplanmış itibar + metriklerdir (creators tablosuna persist).
type CreatorReputation struct {
	Address                                                  string
	Score, Confidence                                        float64
	RiskLevel                                                string
	Breakdown                                                []ScoreBreakdownItem
	TotalTokens, ActiveTokens, RuggedTokens, GraduatedTokens int
	AvgPeakMarketCap, AvgLifetimeHours, SuccessRatePct       float64
	ScoredTs                                                 int64
}

// marshalBreakdown, ScoreBreakdownItem dilimini creators.breakdown kolonu için JSON'a çevirir.
func marshalBreakdown(b []ScoreBreakdownItem) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	out, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// CreatorAggregates, creator-başına outcome agrega döndürür; skorlanmamış creator'lar
// önce (round-robin), sonra en-eski skorlanan. active=çözülmemiş (scorer paydaya katmaz).
func (p *postgresStore) CreatorAggregates(ctx context.Context, limit int) ([]CreatorAgg, error) {
	const q = `SELECT t.creator, COUNT(*),
		SUM(CASE WHEN t.outcome='active'    THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='rug'       THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='dumped'    THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='dead'      THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.outcome='graduated' THEN 1 ELSE 0 END),
		COALESCE(AVG(NULLIF(t.peak_market_cap,0)),0),
		COALESCE(AVG(CASE WHEN t.outcome<>'active' AND t.outcome_scored_ts>0
			THEN (t.outcome_scored_ts - t.first_seen_ts)/3600.0 END),0)
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		WHERE t.creator <> ''
		GROUP BY t.creator, c.scored_ts
		ORDER BY c.scored_ts ASC NULLS FIRST
		LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorAgg, 0, limit)
	for rows.Next() {
		var a CreatorAgg
		if err := rows.Scan(&a.Address, &a.Total, &a.Active, &a.Rug, &a.Dumped, &a.Dead,
			&a.Graduated, &a.AvgPeakMarketCap, &a.AvgLifetimeHours); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpsertReputation, hesaplanmış itibarı creators tablosuna yazar (insert veya güncelle).
func (p *postgresStore) UpsertReputation(ctx context.Context, r CreatorReputation) error {
	breakdownJSON, err := marshalBreakdown(r.Breakdown)
	if err != nil {
		return err
	}
	const q = `INSERT INTO creators (address, reputation_score, confidence, risk_level, breakdown,
		total_tokens, active_tokens, rugged_tokens, graduated_tokens,
		avg_peak_market_cap, avg_lifetime_hours, success_rate_pct, scored_ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (address) DO UPDATE SET
			reputation_score=EXCLUDED.reputation_score, confidence=EXCLUDED.confidence,
			risk_level=EXCLUDED.risk_level, breakdown=EXCLUDED.breakdown,
			total_tokens=EXCLUDED.total_tokens, active_tokens=EXCLUDED.active_tokens,
			rugged_tokens=EXCLUDED.rugged_tokens, graduated_tokens=EXCLUDED.graduated_tokens,
			avg_peak_market_cap=EXCLUDED.avg_peak_market_cap, avg_lifetime_hours=EXCLUDED.avg_lifetime_hours,
			success_rate_pct=EXCLUDED.success_rate_pct, scored_ts=EXCLUDED.scored_ts`
	_, err = p.db.ExecContext(ctx, q, r.Address, r.Score, r.Confidence, r.RiskLevel, breakdownJSON,
		r.TotalTokens, r.ActiveTokens, r.RuggedTokens, r.GraduatedTokens,
		r.AvgPeakMarketCap, r.AvgLifetimeHours, r.SuccessRatePct, r.ScoredTs)
	return err
}
