package store

import "context"

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

// CreatorStore, creator kimlik + agrega kaynağıdır (ISP: dar okuma arayüzü; DIP).
// NOT: Bu task'ta yalnız Creators var — CreatorDetail Task 3'te eklenir (OCP genişletme).
type CreatorStore interface {
	Creators(ctx context.Context, limit int) ([]CreatorRow, error)
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
