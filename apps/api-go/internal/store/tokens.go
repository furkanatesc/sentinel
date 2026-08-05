package store

import (
	"context"
	"time"
)

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
	// firstSeenTs, TokenRow'da olmayan (frontend kontratında yok) first_seen_ts
	// değerini ayrıca taşır; ageSeconds okumada bundan türetilir.
	UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64) error
	RecentTokens(ctx context.Context, limit int) ([]TokenRow, error)
}

func (p *postgresStore) UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64) error {
	const q = `INSERT INTO tokens (mint, symbol, name, launchpad, first_seen_ts)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (mint) DO UPDATE SET
			symbol = EXCLUDED.symbol, name = EXCLUDED.name, launchpad = EXCLUDED.launchpad`
	_, err := p.db.ExecContext(ctx, q, t.Mint, t.Symbol, t.Name, "" /* launchpad: TokenRow'da yok (frontend kontratı) */, firstSeenTs)
	return err
}

func (p *postgresStore) RecentTokens(ctx context.Context, limit int) ([]TokenRow, error) {
	const q = `SELECT mint, symbol, name, first_seen_ts, price, liquidity, vol5m, holders,
		creator_score, safety_score, momentum FROM tokens ORDER BY first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now().Unix()
	var out []TokenRow
	for rows.Next() {
		var t TokenRow
		var firstSeen int64
		if err := rows.Scan(&t.Mint, &t.Symbol, &t.Name, &firstSeen, &t.Price, &t.Liquidity,
			&t.Vol5m, &t.Holders, &t.CreatorScore, &t.SafetyScore, &t.Momentum); err != nil {
			return nil, err
		}
		t.ID = t.Mint
		t.AgeSeconds = now - firstSeen
		if t.AgeSeconds < 0 {
			t.AgeSeconds = 0
		}
		t.Spark = []float64{}
		out = append(out, t)
	}
	return out, rows.Err()
}
