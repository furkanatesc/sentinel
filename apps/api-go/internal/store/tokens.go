package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
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

// MarketUpdate, enrichment döngüsünün yazdığı piyasa alanlarıdır (1b + detail header 1c-followup).
type MarketUpdate struct {
	Mint                              string
	Price, Liquidity, Vol5m, Momentum float64
	Spark                             []float64
	// Detail header alanları (DB'de persist → getToken canlı çağrısız header sunar).
	PriceChangeH24                               float64
	MarketCapUSD                                 float64
	Vol24h                                       float64
	TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int
}

// EnrichTarget, enrichment için gereken minimum bilgidir: hangi havuzu sorgulayacağı + mevcut spark.
type EnrichTarget struct {
	Mint, PoolAddr string
	Spark          []float64
}

// SafetyUpdate, 2a scorer'ının yazdığı token güvenliği sonucudur. CreatorHoldingPct/Known,
// Safety Scorer'ın GİRDİSİ DEĞİL — 2c manipülasyon skoru için aynı holder-fetch'ten
// (sıfır ek RPC) koşullu persist edilir (Known=false → mevcut değer EZİLMEZ).
type SafetyUpdate struct {
	Mint                string
	Score               float64
	Confidence          float64
	Top10Pct            float64
	Breakdown           []ScoreBreakdownItem
	Risks               RiskGroups
	ScoredTs            int64
	CreatorHoldingPct   float64
	CreatorHoldingKnown bool
}

// SafetyTarget, skorlanacak token için gereken minimum bilgidir.
type SafetyTarget struct {
	Mint      string
	Liquidity float64
	Launchpad string
	Creator   string
}

// CreatorFillTarget, REST creator-backfill için hedef mint'tir.
type CreatorFillTarget struct{ Mint string }

// ManipulationTarget, 2c manipülasyon skoru için gereken işlem-akışı girdileridir (h24).
type ManipulationTarget struct {
	Mint                                 string
	Buys, Sells, Buyers                  int
	CreatorHoldingPct, Vol24h, Liquidity float64
}

// ManipulationUpdate, 2c scorer'ının yazdığı manipülasyon sonucudur.
type ManipulationUpdate struct {
	Mint       string
	Score      float64
	Confidence float64
	Breakdown  []ScoreBreakdownItem
	ScoredTs   int64
}

// OpportunityTarget, kompozit opportunity için gereken alt-skorlar + confidence'lar (JOIN creators).
type OpportunityTarget struct {
	Mint                                                                     string
	Safety, SafetyConf, Creator, CreatorConf, Manipulation, ManipulationConf float64
	Momentum, Liquidity                                                      float64
}

// OpportunityUpdate, opportunity scorer'ının yazdığı kompozit sonuçtur (+ türetilmiş signal).
type OpportunityUpdate struct {
	Mint       string
	Score      float64
	Confidence float64
	Breakdown  []ScoreBreakdownItem
	Signal     string // "buy"|"watch"|"avoid"|"" ("" → last_signal boş → frontend null)
	ScoredTs   int64
}

// OutcomeTarget, sınıflandırılacak token için gereken anlık + tepe piyasa durumudur.
type OutcomeTarget struct {
	Mint                                                             string
	CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h float64
	FirstSeenTs                                                      int64
}

// OutcomeUpdate, outcome sınıflandırıcısının yazdığı sonuçtur.
type OutcomeUpdate struct {
	Mint, Outcome, LiquidityStatus string
	MaxDrawdownPct                 float64
	ScoredTs                       int64
}

// KpiCounts, Overview KPI kartları için türetilebilir sayımlardır (2d).
type KpiCounts struct{ Detected, HighConf, Critical, Signals int }

// RadarPoint, Overview radar scatter noktası (frontend RadarPoint ile birebir). level: RiskLevel.
type RadarPoint struct {
	X    float64 `json:"x"` // creatorScore
	Y    float64 `json:"y"` // momentum
	Z    float64 `json:"z"` // liquidity
	Name string  `json:"name"`
	Level string `json:"level"`
}

// TokenStore, mint-PK token kaynağıdır (upsert; DIP).
type TokenStore interface {
	// firstSeenTs, TokenRow'da olmayan (frontend kontratında yok) first_seen_ts
	// değerini ayrıca taşır; ageSeconds okumada bundan türetilir.
	UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64, creator string) error
	RecentTokens(ctx context.Context, limit int) ([]TokenRow, error)
	// 1b: keşif (kimlik+havuz) — inserted, token'ın YENİ eklendiğini bildirir (event spam'i önler).
	UpsertDiscovered(ctx context.Context, d DiscoveredToken) (inserted bool, err error)
	// 1b: enrichment (piyasa alanları).
	UpdateMarket(ctx context.Context, m MarketUpdate) error
	// 1b: enrichment hedefleri (havuz adresi olan token'lar, en yeni önce).
	EnrichTargets(ctx context.Context, limit int) ([]EnrichTarget, error)
	// 1c: getToken için tek token kimlik+havuz (bulunamazsa ok=false).
	TokenDetailBase(ctx context.Context, mint string) (TokenDetailBase, bool, error)
	// 2a: token güvenliği skorunu yazar / skorlanacak hedefleri döndürür.
	UpdateSafety(ctx context.Context, s SafetyUpdate) error
	SafetyScoreTargets(ctx context.Context, limit int) ([]SafetyTarget, error)
	// 2b-2a: token outcome (rug/graduated/dumped/dead/active) sınıflandırma sonucunu yazar / hedefleri döndürür.
	OutcomeTargets(ctx context.Context, limit int) ([]OutcomeTarget, error)
	UpdateOutcome(ctx context.Context, u OutcomeUpdate) error
	// REST creator-backfill: creator'sız pump.fun hedeflerini döndürür / bulunan creator'ı yazar (merge).
	CreatorFillTargets(ctx context.Context, limit int) ([]CreatorFillTarget, error)
	SetCreatorBackfill(ctx context.Context, mint, creator string, backfillTs int64) error
	// 2b-2b: creator itibar agregası (outcome sayımları) / hesaplanmış itibarı persist eder.
	CreatorAggregates(ctx context.Context, limit int) ([]CreatorAgg, error)
	UpsertReputation(ctx context.Context, r CreatorReputation) error
	// 2c: manipülasyon riski agrega girdilerini döndürür / hesaplanmış skoru persist eder.
	ManipulationTargets(ctx context.Context, limit int) ([]ManipulationTarget, error)
	UpdateManipulation(ctx context.Context, u ManipulationUpdate) error
	// 2d: kompozit opportunity girdilerini döndürür / hesaplanmış skoru+signal'ı persist eder.
	OpportunityScoreTargets(ctx context.Context, limit int) ([]OpportunityTarget, error)
	UpdateOpportunity(ctx context.Context, u OpportunityUpdate) error
	// 2d: Overview KPI kartları için 4 türetilebilir sayım (tek agrega).
	Kpis(ctx context.Context) (KpiCounts, error)
	// 2d: radar scatter noktaları (creatorScore/momentum/liquidity) + risk level (scoreToLevel parity).
	Radar(ctx context.Context, limit int) ([]RadarPoint, error)
}

func (p *postgresStore) UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64, creator string) error {
	const q = `INSERT INTO tokens (mint, symbol, name, launchpad, first_seen_ts, creator)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (mint) DO UPDATE SET
			symbol = EXCLUDED.symbol, name = EXCLUDED.name, launchpad = EXCLUDED.launchpad,
			creator = COALESCE(NULLIF(EXCLUDED.creator, ''), tokens.creator)`
	_, err := p.db.ExecContext(ctx, q, t.Mint, t.Symbol, t.Name,
		"" /* launchpad: TokenRow'da yok (frontend kontratı) */, firstSeenTs, creator)
	return err
}

func (p *postgresStore) RecentTokens(ctx context.Context, limit int) ([]TokenRow, error) {
	// 2b-2b: creator_score sütunu yerine creators tablosundan LEFT JOIN (creator itibarı gerçek;
	// skorlanmamış/creator'sız token → COALESCE 0, fake reputationByAddr[""] boş değer ile parite).
	// 2d: t.last_signal de seçilir (fake/postgres parite; boş → nil, frontend null).
	const q = `SELECT t.mint, t.symbol, t.name, t.first_seen_ts, t.price, t.liquidity, t.vol5m, t.holders,
		COALESCE(c.reputation_score,0), t.safety_score, t.momentum, t.spark, t.last_signal
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		ORDER BY t.first_seen_ts DESC LIMIT $1`
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
		var signal string
		if err := rows.Scan(&t.Mint, &t.Symbol, &t.Name, &firstSeen, &t.Price, &t.Liquidity,
			&t.Vol5m, &t.Holders, &t.CreatorScore, &t.SafetyScore, &t.Momentum, &sparkJSON, &signal); err != nil {
			return nil, err
		}
		t.ID = t.Mint
		t.AgeSeconds = now - firstSeen
		if t.AgeSeconds < 0 {
			t.AgeSeconds = 0
		}
		t.Spark = parseSparkJSON(sparkJSON)
		if signal != "" {
			t.Signal = &signal
		}
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
	const q = `UPDATE tokens SET price=$2, liquidity=$3, vol5m=$4, momentum=$5, spark=$6,
		price_change_h24=$7, market_cap_usd=$8, vol24h=$9,
		txns_buys=$10, txns_sells=$11, txns_buyers=$12, txns_sellers=$13,
		peak_market_cap = GREATEST(peak_market_cap, $8),
		peak_liquidity  = GREATEST(peak_liquidity, $3)
		WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, m.Mint, m.Price, m.Liquidity, m.Vol5m, m.Momentum, string(sparkJSON),
		m.PriceChangeH24, m.MarketCapUSD, m.Vol24h, m.TxnsBuys, m.TxnsSells, m.TxnsBuyers, m.TxnsSellers)
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
	// 2b-2b: creators LEFT JOIN → creator itibarı (skorlanmamış/creator'sız token → COALESCE 0/0/'').
	const q = `SELECT tokens.name, tokens.symbol, tokens.pool_address, tokens.first_seen_ts,
		tokens.price, tokens.liquidity,
		tokens.price_change_h24, tokens.market_cap_usd, tokens.vol24h,
		tokens.safety_score, tokens.safety_confidence, tokens.top10_holder_pct,
		tokens.safety_breakdown, tokens.safety_risks, tokens.safety_scored_ts,
		COALESCE(c.reputation_score,0), COALESCE(c.confidence,0), COALESCE(c.breakdown,''),
		tokens.manipulation_score, tokens.manipulation_confidence, tokens.manipulation_breakdown,
		tokens.manipulation_scored_ts, tokens.txns_buys, tokens.txns_sells, tokens.txns_buyers,
		tokens.creator_holding_pct,
		tokens.opportunity_score, tokens.opportunity_confidence, tokens.opportunity_breakdown,
		tokens.opportunity_scored_ts
		FROM tokens LEFT JOIN creators c ON c.address = tokens.creator
		WHERE tokens.mint=$1`
	var b TokenDetailBase
	var bdJSON, rkJSON, repBdJSON, manipBdJSON, oppBdJSON string
	err := p.db.QueryRowContext(ctx, q, mint).Scan(&b.Name, &b.Symbol, &b.PoolAddr, &b.FirstSeenTs,
		&b.Price, &b.Liquidity, &b.PriceChangeH24, &b.MarketCapUSD, &b.Vol24h,
		&b.SafetyScore, &b.SafetyConfidence, &b.Top10Pct, &bdJSON, &rkJSON, &b.SafetyScoredTs,
		&b.CreatorRepScore, &b.CreatorRepConfidence, &repBdJSON,
		&b.ManipulationScore, &b.ManipulationConfidence, &manipBdJSON, &b.ManipulationScoredTs,
		&b.TxnsBuys, &b.TxnsSells, &b.TxnsBuyers, &b.CreatorHoldingPct,
		&b.OpportunityScore, &b.OpportunityConfidence, &oppBdJSON, &b.OpportunityScoredTs)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenDetailBase{}, false, nil
	}
	if err != nil {
		return TokenDetailBase{}, false, err
	}
	b.SafetyBreakdown = parseBreakdownJSON(bdJSON)
	b.SafetyRisks = parseRiskGroupsJSON(rkJSON)
	b.CreatorRepBreakdown = parseBreakdownJSON(repBdJSON)
	b.ManipulationBreakdown = parseBreakdownJSON(manipBdJSON)
	b.OpportunityBreakdown = parseBreakdownJSON(oppBdJSON)
	return b, true, nil
}

func (p *postgresStore) UpdateSafety(ctx context.Context, s SafetyUpdate) error {
	bdJSON, err := json.Marshal(s.Breakdown)
	if err != nil {
		return err
	}
	rkJSON, err := json.Marshal(s.Risks)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET safety_score=$2, safety_confidence=$3, top10_holder_pct=$4,
		safety_breakdown=$5, safety_risks=$6, safety_scored_ts=$7,
		creator_holding_pct = CASE WHEN $8 THEN $9 ELSE creator_holding_pct END
		WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, s.Mint, s.Score, s.Confidence, s.Top10Pct,
		string(bdJSON), string(rkJSON), s.ScoredTs, s.CreatorHoldingKnown, s.CreatorHoldingPct)
	return err
}

func (p *postgresStore) SafetyScoreTargets(ctx context.Context, limit int) ([]SafetyTarget, error) {
	const q = `SELECT mint, liquidity, launchpad, creator FROM tokens
		WHERE pool_address <> '' ORDER BY safety_scored_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SafetyTarget, 0, limit)
	for rows.Next() {
		var t SafetyTarget
		if err := rows.Scan(&t.Mint, &t.Liquidity, &t.Launchpad, &t.Creator); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) OutcomeTargets(ctx context.Context, limit int) ([]OutcomeTarget, error) {
	const q = `SELECT mint, market_cap_usd, liquidity, peak_market_cap, peak_liquidity, vol24h, first_seen_ts
		FROM tokens WHERE pool_address <> ''
		ORDER BY outcome_scored_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OutcomeTarget, 0, limit)
	for rows.Next() {
		var t OutcomeTarget
		if err := rows.Scan(&t.Mint, &t.CurMarketCap, &t.CurLiquidity, &t.PeakMarketCap,
			&t.PeakLiquidity, &t.Vol24h, &t.FirstSeenTs); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) UpdateOutcome(ctx context.Context, u OutcomeUpdate) error {
	const q = `UPDATE tokens SET outcome=$2, liquidity_status=$3, max_drawdown_pct=$4, outcome_scored_ts=$5 WHERE mint=$1`
	_, err := p.db.ExecContext(ctx, q, u.Mint, u.Outcome, u.LiquidityStatus, u.MaxDrawdownPct, u.ScoredTs)
	return err
}

func (p *postgresStore) CreatorFillTargets(ctx context.Context, limit int) ([]CreatorFillTarget, error) {
	const q = `SELECT mint FROM tokens WHERE launchpad='Pump.fun' AND creator=''
		ORDER BY creator_backfill_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorFillTarget, 0, limit)
	for rows.Next() {
		var t CreatorFillTarget
		if err := rows.Scan(&t.Mint); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) SetCreatorBackfill(ctx context.Context, mint, creator string, backfillTs int64) error {
	const q = `UPDATE tokens SET creator=COALESCE(NULLIF($2,''), creator), creator_backfill_ts=$3 WHERE mint=$1`
	_, err := p.db.ExecContext(ctx, q, mint, creator, backfillTs)
	return err
}

func (p *postgresStore) ManipulationTargets(ctx context.Context, limit int) ([]ManipulationTarget, error) {
	const q = `SELECT mint, txns_buys, txns_sells, txns_buyers, creator_holding_pct, vol24h, liquidity
		FROM tokens WHERE pool_address <> ''
		ORDER BY manipulation_scored_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ManipulationTarget, 0, limit)
	for rows.Next() {
		var t ManipulationTarget
		if err := rows.Scan(&t.Mint, &t.Buys, &t.Sells, &t.Buyers,
			&t.CreatorHoldingPct, &t.Vol24h, &t.Liquidity); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) UpdateManipulation(ctx context.Context, u ManipulationUpdate) error {
	bdJSON, err := json.Marshal(u.Breakdown)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET manipulation_score=$2, manipulation_confidence=$3,
		manipulation_breakdown=$4, manipulation_scored_ts=$5 WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, u.Mint, u.Score, u.Confidence, string(bdJSON), u.ScoredTs)
	return err
}

func (p *postgresStore) OpportunityScoreTargets(ctx context.Context, limit int) ([]OpportunityTarget, error) {
	const q = `SELECT t.mint, t.safety_score, t.safety_confidence,
		COALESCE(c.reputation_score,0), COALESCE(c.confidence,0),
		t.manipulation_score, t.manipulation_confidence, t.momentum, t.liquidity
		FROM tokens t LEFT JOIN creators c ON c.address = t.creator
		ORDER BY t.first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OpportunityTarget, 0, limit)
	for rows.Next() {
		var t OpportunityTarget
		if err := rows.Scan(&t.Mint, &t.Safety, &t.SafetyConf, &t.Creator, &t.CreatorConf,
			&t.Manipulation, &t.ManipulationConf, &t.Momentum, &t.Liquidity); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) UpdateOpportunity(ctx context.Context, u OpportunityUpdate) error {
	bd, err := json.Marshal(u.Breakdown)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET opportunity_score=$2, opportunity_confidence=$3,
		opportunity_breakdown=$4, opportunity_scored_ts=$5, last_signal=$6 WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, u.Mint, u.Score, u.Confidence, string(bd), u.ScoredTs, u.Signal)
	return err
}

func (p *postgresStore) Kpis(ctx context.Context) (KpiCounts, error) {
	const q = `SELECT
		COUNT(*) FILTER (WHERE first_seen_ts >= $1),
		COUNT(*) FILTER (WHERE safety_score >= 70 AND safety_confidence >= 0.5),
		COUNT(*) FILTER (WHERE (manipulation_score >= 70 AND manipulation_confidence >= 0.5)
		                    OR (safety_score <= 30 AND safety_confidence >= 0.5)),
		COUNT(*) FILTER (WHERE last_signal IN ('buy','watch'))
		FROM tokens`
	var c KpiCounts
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	err := p.db.QueryRowContext(ctx, q, cutoff).Scan(&c.Detected, &c.HighConf, &c.Critical, &c.Signals)
	return c, err
}

func (p *postgresStore) Radar(ctx context.Context, limit int) ([]RadarPoint, error) {
	rows, err := p.RecentTokens(ctx, limit)
	if err != nil {
		return nil, err
	}
	return radarFrom(rows), nil
}

// scoreToLevel, frontend format.ts scoreToLevel ile birebir (parity).
func scoreToLevel(score float64) string {
	switch {
	case score <= 24:
		return "critical"
	case score <= 49:
		return "high"
	case score <= 69:
		return "medium"
	case score <= 84:
		return "good"
	default:
		return "strong"
	}
}

// radarFrom, TokenRow listesini radar noktalarına çevirir (mock radarFrom birebir).
func radarFrom(rows []TokenRow) []RadarPoint {
	out := make([]RadarPoint, 0, len(rows))
	for _, t := range rows {
		out = append(out, RadarPoint{
			X: t.CreatorScore, Y: t.Momentum, Z: t.Liquidity, Name: t.Symbol,
			Level: scoreToLevel(math.Round((t.CreatorScore + t.SafetyScore) / 2)),
		})
	}
	return out
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

func parseBreakdownJSON(s string) []ScoreBreakdownItem {
	if s == "" {
		return []ScoreBreakdownItem{}
	}
	var out []ScoreBreakdownItem
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return []ScoreBreakdownItem{}
	}
	return out
}

func parseRiskGroupsJSON(s string) RiskGroups {
	empty := RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{}, Creator: []RiskItem{}}
	if s == "" {
		return empty
	}
	var out RiskGroups
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return empty
	}
	if out.Contract == nil {
		out.Contract = []RiskItem{}
	}
	if out.Market == nil {
		out.Market = []RiskItem{}
	}
	if out.Creator == nil {
		out.Creator = []RiskItem{}
	}
	return out
}
