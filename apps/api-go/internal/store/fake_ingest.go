package store

import (
	"context"
	"sort"
	"sync"
)

type fakeEventStore struct {
	mu   sync.Mutex
	rows []EventRow // en yeni sonda
}

// NewFakeEventStore, testler ve DB'siz mod için in-memory EventStore döndürür.
func NewFakeEventStore() EventStore { return &fakeEventStore{} }

func (f *fakeEventStore) InsertEvent(_ context.Context, e EventRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, e)
	return nil
}

func (f *fakeEventStore) RecentEvents(_ context.Context, limit int) ([]EventRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]EventRow, 0, limit)
	for i := len(f.rows) - 1; i >= 0 && len(out) < limit; i-- { // en yeni önce
		out = append(out, f.rows[i])
	}
	return out, nil
}

type fakeTok struct {
	row       TokenRow
	poolAddr  string
	firstSeen int64
	creator   string
	// Detail header (TokenRow'da olmayan alanlar; enrichment yazar, TokenDetailBase okur).
	priceChangeH24, marketCapUSD, vol24h float64
	launchpad                            string
	// 2a safety
	safetyScore, safetyConfidence, top10Pct float64
	safetyBreakdown                         []ScoreBreakdownItem
	safetyRisks                             RiskGroups
	safetyScoredTs                          int64
	// 2b-2a outcome + peak
	peakMarketCap, peakLiquidity, maxDrawdownPct float64
	outcome, liquidityStatus                     string
	outcomeScoredTs                              int64
}

type fakeTokenStore struct {
	mu    sync.Mutex
	byID  map[string]fakeTok
	order []string // ekleme sırası
}

// NewFakeTokenStore, testler ve DB'siz mod için in-memory TokenStore döndürür.
func NewFakeTokenStore() TokenStore { return &fakeTokenStore{byID: map[string]fakeTok{}} }

func (f *fakeTokenStore) UpsertToken(_ context.Context, t TokenRow, firstSeenTs int64, creator string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.Spark == nil {
		t.Spark = []float64{}
	}
	cur, ok := f.byID[t.ID]
	if !ok {
		f.order = append(f.order, t.ID)
		cur.safetyBreakdown = []ScoreBreakdownItem{}
		cur.safetyRisks = RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{}, Creator: []RiskItem{}}
		cur.outcome = "active"
		cur.liquidityStatus = "unlocked"
	}
	cur.row = t
	cur.firstSeen = firstSeenTs
	if creator != "" { // boş creator mevcut gerçek olanı ezmez (postgres COALESCE parity)
		cur.creator = creator
	}
	f.byID[t.ID] = cur
	return nil
}

func (f *fakeTokenStore) Creators(_ context.Context, limit int) ([]CreatorRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := map[string]int{}
	firstOrder := map[string]int{} // ilk görülme sırası (tiebreak: erken = önce)
	for i, id := range f.order {
		c := f.byID[id].creator
		if c == "" {
			continue
		}
		if _, seen := counts[c]; !seen {
			firstOrder[c] = i
		}
		counts[c]++
	}
	out := make([]CreatorRow, 0, len(counts))
	for addr, n := range counts {
		out = append(out, CreatorRow{Address: addr, TotalTokens: n, RiskLevel: "medium"})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalTokens != out[j].TotalTokens {
			return out[i].TotalTokens > out[j].TotalTokens // en çok önce
		}
		return firstOrder[out[i].Address] < firstOrder[out[j].Address] // erken görülen önce
	})
	// Diğer fake metotlarla aynı sınırlama deseni (len(out) < limit): limit<=0 → boş
	// (postgres LIMIT $1=0 ile eşleşir; post-hoc "limit > 0 &&" bekçisi yerine).
	bounded := make([]CreatorRow, 0, len(out))
	for i := 0; i < len(out) && len(bounded) < limit; i++ {
		bounded = append(bounded, out[i])
	}
	return bounded, nil
}

func (f *fakeTokenStore) CreatorDetail(_ context.Context, address string) (CreatorProfile, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	matches := make([]fakeTok, 0, len(f.order))
	var firstSeen int64
	found := false
	for _, id := range f.order {
		tk := f.byID[id]
		if tk.creator != address {
			continue
		}
		if !found || tk.firstSeen < firstSeen {
			firstSeen = tk.firstSeen
		}
		found = true
		matches = append(matches, tk)
	}
	if !found {
		return CreatorProfile{}, false, nil
	}
	// history en yeni önce (postgres ORDER BY first_seen_ts DESC ile eşleşir; insertion sırası
	// firstSeenTs sırasıyla aynı olmayabilir — bkz. TestCreatorDetail).
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].firstSeen > matches[j].firstSeen })
	history := make([]CreatorTokenHistoryItem, 0, len(matches))
	for _, tk := range matches {
		history = append(history, newHistoryItem(tk.row.Mint, tk.row.Symbol, tk.firstSeen, tk.marketCapUSD))
	}
	return buildCreatorProfile(address, firstSeen, len(history), history), true, nil
}

func (f *fakeTokenStore) UpsertDiscovered(_ context.Context, d DiscoveredToken) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[d.Mint]
	inserted := !ok
	if inserted {
		f.order = append(f.order, d.Mint)
		cur.row = TokenRow{ID: d.Mint, Mint: d.Mint, Spark: []float64{}}
		cur.firstSeen = d.FirstSeenTs
		cur.safetyBreakdown = []ScoreBreakdownItem{}
		cur.safetyRisks = RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{}, Creator: []RiskItem{}}
		cur.outcome = "active"
		cur.liquidityStatus = "unlocked"
	}
	cur.row.Name, cur.row.Symbol = d.Name, d.Symbol
	cur.poolAddr = d.PoolAddr
	cur.launchpad = d.Launchpad
	f.byID[d.Mint] = cur
	return inserted, nil
}

func (f *fakeTokenStore) UpdateMarket(_ context.Context, m MarketUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[m.Mint]
	if !ok {
		return nil
	}
	cur.row.Price, cur.row.Liquidity = m.Price, m.Liquidity
	cur.row.Vol5m, cur.row.Momentum = m.Vol5m, m.Momentum
	cur.priceChangeH24, cur.marketCapUSD, cur.vol24h = m.PriceChangeH24, m.MarketCapUSD, m.Vol24h
	if m.Spark == nil {
		m.Spark = []float64{}
	}
	cur.row.Spark = m.Spark
	cur.peakMarketCap = max(cur.peakMarketCap, m.MarketCapUSD)
	cur.peakLiquidity = max(cur.peakLiquidity, m.Liquidity)
	f.byID[m.Mint] = cur
	return nil
}

func (f *fakeTokenStore) EnrichTargets(_ context.Context, limit int) ([]EnrichTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]EnrichTarget, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		t := f.byID[f.order[i]]
		if t.poolAddr == "" {
			continue
		}
		out = append(out, EnrichTarget{Mint: t.row.Mint, PoolAddr: t.poolAddr, Spark: t.row.Spark})
	}
	return out, nil
}

func (f *fakeTokenStore) TokenDetailBase(_ context.Context, mint string) (TokenDetailBase, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[mint]
	if !ok {
		return TokenDetailBase{}, false, nil
	}
	return TokenDetailBase{
		Name: t.row.Name, Symbol: t.row.Symbol, PoolAddr: t.poolAddr, FirstSeenTs: t.firstSeen,
		Price: t.row.Price, Liquidity: t.row.Liquidity,
		PriceChangeH24: t.priceChangeH24, MarketCapUSD: t.marketCapUSD, Vol24h: t.vol24h,
		SafetyScore: t.safetyScore, SafetyConfidence: t.safetyConfidence, Top10Pct: t.top10Pct,
		SafetyBreakdown: t.safetyBreakdown, SafetyRisks: t.safetyRisks, SafetyScoredTs: t.safetyScoredTs,
	}, true, nil
}

func (f *fakeTokenStore) UpdateSafety(_ context.Context, s SafetyUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[s.Mint]
	if !ok {
		return nil
	}
	cur.row.SafetyScore = s.Score
	cur.safetyScore, cur.safetyConfidence, cur.top10Pct = s.Score, s.Confidence, s.Top10Pct
	cur.safetyBreakdown, cur.safetyRisks, cur.safetyScoredTs = s.Breakdown, s.Risks, s.ScoredTs
	f.byID[s.Mint] = cur
	return nil
}

func (f *fakeTokenStore) SafetyScoreTargets(_ context.Context, limit int) ([]SafetyTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]SafetyTarget, 0, limit)
	// en eski safety_scored_ts önce; poolAddr'siz atla.
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].safetyScoredTs < f.byID[ids[j]].safetyScoredTs
	})
	for _, id := range ids {
		t := f.byID[id]
		if t.poolAddr == "" || len(out) >= limit {
			continue
		}
		out = append(out, SafetyTarget{Mint: t.row.Mint, Liquidity: t.row.Liquidity, Launchpad: t.launchpad})
	}
	return out, nil
}

func (f *fakeTokenStore) OutcomeTargets(_ context.Context, limit int) ([]OutcomeTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].outcomeScoredTs < f.byID[ids[j]].outcomeScoredTs // en eski önce
	})
	out := make([]OutcomeTarget, 0, limit)
	for _, id := range ids {
		t := f.byID[id]
		if t.poolAddr == "" || len(out) >= limit {
			continue
		}
		out = append(out, OutcomeTarget{
			Mint: t.row.Mint, CurMarketCap: t.marketCapUSD, CurLiquidity: t.row.Liquidity,
			PeakMarketCap: t.peakMarketCap, PeakLiquidity: t.peakLiquidity, Vol24h: t.vol24h, FirstSeenTs: t.firstSeen,
		})
	}
	return out, nil
}

func (f *fakeTokenStore) UpdateOutcome(_ context.Context, u OutcomeUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[u.Mint]
	if !ok {
		return nil
	}
	cur.outcome, cur.liquidityStatus = u.Outcome, u.LiquidityStatus
	cur.maxDrawdownPct, cur.outcomeScoredTs = u.MaxDrawdownPct, u.ScoredTs
	f.byID[u.Mint] = cur
	return nil
}

func (f *fakeTokenStore) RecentTokens(_ context.Context, limit int) ([]TokenRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TokenRow, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, f.byID[f.order[i]].row)
	}
	return out, nil
}
