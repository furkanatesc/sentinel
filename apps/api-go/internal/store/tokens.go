package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

// DiscoveredToken, GeckoTerminal keşfinden gelen kimlik+havuz bilgisidir (1b).
type DiscoveredToken struct {
	Mint, Name, Symbol, Launchpad, PoolAddr string
	FirstSeenTs                             int64
}

// MarketUpdate, enrichment döngüsünün yazdığı piyasa alanlarıdır (1b).
type MarketUpdate struct {
	Mint                              string
	Price, Liquidity, Vol5m, Momentum float64
	Spark                             []float64
}

// EnrichTarget, enrichment için gereken minimum bilgidir: hangi havuzu sorgulayacağı + mevcut spark.
type EnrichTarget struct {
	Mint, PoolAddr string
	Spark          []float64
}

// TokenStore, mint-PK token kaynağıdır (upsert; DIP).
type TokenStore interface {
	// firstSeenTs, TokenRow'da olmayan (frontend kontratında yok) first_seen_ts
	// değerini ayrıca taşır; ageSeconds okumada bundan türetilir.
	UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64) error
	RecentTokens(ctx context.Context, limit int) ([]TokenRow, error)
	// 1b: keşif (kimlik+havuz) — inserted, token'ın YENİ eklendiğini bildirir (event spam'i önler).
	UpsertDiscovered(ctx context.Context, d DiscoveredToken) (inserted bool, err error)
	// 1b: enrichment (piyasa alanları).
	UpdateMarket(ctx context.Context, m MarketUpdate) error
	// 1b: enrichment hedefleri (havuz adresi olan token'lar, en yeni önce).
	EnrichTargets(ctx context.Context, limit int) ([]EnrichTarget, error)
	// 1c: getToken için tek token kimlik+havuz (bulunamazsa ok=false).
	TokenDetailBase(ctx context.Context, mint string) (TokenDetailBase, bool, error)
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
		creator_score, safety_score, momentum, spark FROM tokens ORDER BY first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now().Unix()
	out := make([]TokenRow, 0, limit)
	for rows.Next() {
		var t TokenRow
		var firstSeen int64
		var sparkJSON string
		if err := rows.Scan(&t.Mint, &t.Symbol, &t.Name, &firstSeen, &t.Price, &t.Liquidity,
			&t.Vol5m, &t.Holders, &t.CreatorScore, &t.SafetyScore, &t.Momentum, &sparkJSON); err != nil {
			return nil, err
		}
		t.ID = t.Mint
		t.AgeSeconds = now - firstSeen
		if t.AgeSeconds < 0 {
			t.AgeSeconds = 0
		}
		t.Spark = parseSparkJSON(sparkJSON)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) UpsertDiscovered(ctx context.Context, d DiscoveredToken) (bool, error) {
	const q = `INSERT INTO tokens (mint, symbol, name, launchpad, pool_address, first_seen_ts)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (mint) DO UPDATE SET
			symbol = EXCLUDED.symbol, name = EXCLUDED.name,
			launchpad = EXCLUDED.launchpad, pool_address = EXCLUDED.pool_address
		RETURNING (xmax = 0) AS inserted`
	var inserted bool
	err := p.db.QueryRowContext(ctx, q, d.Mint, d.Symbol, d.Name, d.Launchpad, d.PoolAddr, d.FirstSeenTs).Scan(&inserted)
	return inserted, err
}

func (p *postgresStore) UpdateMarket(ctx context.Context, m MarketUpdate) error {
	sparkJSON, err := json.Marshal(m.Spark)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET price=$2, liquidity=$3, vol5m=$4, momentum=$5, spark=$6 WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, m.Mint, m.Price, m.Liquidity, m.Vol5m, m.Momentum, string(sparkJSON))
	return err
}

func (p *postgresStore) EnrichTargets(ctx context.Context, limit int) ([]EnrichTarget, error) {
	const q = `SELECT mint, pool_address, spark FROM tokens
		WHERE pool_address <> '' ORDER BY first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EnrichTarget, 0, limit)
	for rows.Next() {
		var t EnrichTarget
		var sparkJSON string
		if err := rows.Scan(&t.Mint, &t.PoolAddr, &sparkJSON); err != nil {
			return nil, err
		}
		t.Spark = parseSparkJSON(sparkJSON)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) TokenDetailBase(ctx context.Context, mint string) (TokenDetailBase, bool, error) {
	const q = `SELECT name, symbol, pool_address, first_seen_ts FROM tokens WHERE mint=$1`
	var b TokenDetailBase
	err := p.db.QueryRowContext(ctx, q, mint).Scan(&b.Name, &b.Symbol, &b.PoolAddr, &b.FirstSeenTs)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenDetailBase{}, false, nil
	}
	if err != nil {
		return TokenDetailBase{}, false, err
	}
	return b, true, nil
}

// parseSparkJSON, boş/bozuk JSON'da boş dilim döner (asla nil değil).
func parseSparkJSON(s string) []float64 {
	if s == "" {
		return []float64{}
	}
	var out []float64
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return []float64{}
	}
	return out
}
