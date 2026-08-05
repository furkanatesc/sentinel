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

func (p *postgresStore) InsertEvent(ctx context.Context, e EventRow) error {
	const q = `INSERT INTO events
		(id, signature, slot, type, mint, symbol, launchpad, dex, liquidity, creator_score,
		 risk_level, token_age_seconds, volume5m, holder_growth_pct, severity, detail, ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (id) DO NOTHING`
	_, err := p.db.ExecContext(ctx, q, e.ID, e.Signature, int64(e.Slot), e.Type, e.Mint, e.Symbol,
		e.Launchpad, e.DEX, e.Liquidity, e.CreatorScore, e.RiskLevel, e.TokenAgeSeconds, e.Volume5m,
		e.HolderGrowthPct, e.Severity, e.Detail, e.Ts)
	return err
}

func (p *postgresStore) RecentEvents(ctx context.Context, limit int) ([]EventRow, error) {
	const q = `SELECT id, signature, slot, type, mint, symbol, launchpad, dex, liquidity, creator_score,
		risk_level, token_age_seconds, volume5m, holder_growth_pct, severity, detail, ts
		FROM events ORDER BY ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventRow
	for rows.Next() {
		var e EventRow
		var slot int64
		if err := rows.Scan(&e.ID, &e.Signature, &slot, &e.Type, &e.Mint, &e.Symbol, &e.Launchpad,
			&e.DEX, &e.Liquidity, &e.CreatorScore, &e.RiskLevel, &e.TokenAgeSeconds, &e.Volume5m,
			&e.HolderGrowthPct, &e.Severity, &e.Detail, &e.Ts); err != nil {
			return nil, err
		}
		e.Slot = uint64(slot)
		e.Time = "" // 1a: time frontend'de ts'ten türetilir; boş bırakılır (kontratta string)
		out = append(out, e)
	}
	return out, rows.Err()
}
