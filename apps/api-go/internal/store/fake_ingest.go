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
	// REST creator-backfill
	creatorBackfillTs int64
}

type fakeTokenStore struct {
	mu    sync.Mutex
	byID  map[string]fakeTok
	order []string // ekleme sırası
	// 2b-2b: hesaplanmış creator itibarı (adres → son itibar).
	reputationByAddr map[string]CreatorReputation
}

// NewFakeTokenStore, testler ve DB'siz mod için in-memory TokenStore döndürür.
func NewFakeTokenStore() TokenStore {
	return &fakeTokenStore{byID: map[string]fakeTok{}, reputationByAddr: map[string]CreatorReputation{}}
}

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
		row := CreatorRow{Address: addr, TotalTokens: n, RiskLevel: "medium"} // nötr → skorlanmamış creator
		if rep, ok := f.reputationByAddr[addr]; ok {
			row.ReputationScore = rep.Score
			row.RiskLevel = rep.RiskLevel
			row.ActiveTokens = rep.ActiveTokens
			row.RuggedTokens = rep.RuggedTokens
			row.SuccessRatePct = rep.SuccessRatePct
		}
		out = append(out, row)
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
		history = append(history, newHistoryItem(tk.row.Mint, tk.row.Symbol, tk.firstSeen,
			tk.marketCapUSD, tk.peakMarketCap, tk.maxDrawdownPct, tk.outcome, tk.liquidityStatus))
	}
	prof := buildCreatorProfile(address, firstSeen, len(history), history)
	// postgres LEFT JOIN parite: skorlanmışsa gerçek alanlar, yoksa nötr (reputationByAddr'de yok).
	if rep, ok := f.reputationByAddr[address]; ok {
		breakdown := rep.Breakdown
		if breakdown == nil {
			breakdown = []ScoreBreakdownItem{}
		}
		prof.Reputation = ScoreDetail{Key: "creatorReputation", Value: rep.Score, Confidence: rep.Confidence, Breakdown: breakdown}
		prof.RiskLevel = rep.RiskLevel
		prof.Metrics = CreatorMetrics{
			TotalTokens: len(history), ActiveTokens: rep.ActiveTokens, RuggedTokens: rep.RuggedTokens,
			AvgPeakMarketCap: rep.AvgPeakMarketCap, AvgLifetimeHours: rep.AvgLifetimeHours, SuccessRatePct: rep.SuccessRatePct,
		}
	}
	return prof, true, nil
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

func (f *fakeTokenStore) CreatorFillTargets(_ context.Context, limit int) ([]CreatorFillTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].creatorBackfillTs < f.byID[ids[j]].creatorBackfillTs // en eski-denenen önce
	})
	out := make([]CreatorFillTarget, 0, limit)
	for _, id := range ids {
		t := f.byID[id]
		if t.launchpad != "Pump.fun" || t.creator != "" || len(out) >= limit {
			continue
		}
		out = append(out, CreatorFillTarget{Mint: t.row.Mint})
	}
	return out, nil
}

func (f *fakeTokenStore) SetCreatorBackfill(_ context.Context, mint, creator string, backfillTs int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[mint]
	if !ok {
		return nil
	}
	if creator != "" { // boş gerçek'i ezmez (postgres COALESCE parity)
		cur.creator = creator
	}
	cur.creatorBackfillTs = backfillTs
	f.byID[mint] = cur
	return nil
}

// CreatorAggregates, creator-başına outcome agrega döndürür (postgresStore.CreatorAggregates ile
// aynı sözleşme): skorlanmamış (reputationByAddr'de yok → ScoredTs=0) creator'lar önce, sonra
// en-eski skorlanan.
func (f *fakeTokenStore) CreatorAggregates(_ context.Context, limit int) ([]CreatorAgg, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	byAddr := map[string]*CreatorAgg{}
	peakSum, lifeSum := map[string]float64{}, map[string]float64{}
	peakN, lifeN := map[string]int{}, map[string]int{}
	for _, id := range f.order {
		t := f.byID[id]
		if t.creator == "" {
			continue
		}
		a := byAddr[t.creator]
		if a == nil {
			a = &CreatorAgg{Address: t.creator}
			byAddr[t.creator] = a
		}
		a.Total++
		switch t.outcome {
		case "active":
			a.Active++
		case "rug":
			a.Rug++
		case "dumped":
			a.Dumped++
		case "dead":
			a.Dead++
		case "graduated":
			a.Graduated++
		}
		if t.peakMarketCap > 0 {
			peakSum[t.creator] += t.peakMarketCap
			peakN[t.creator]++
		}
		if t.outcome != "active" && t.outcomeScoredTs > 0 {
			lifeSum[t.creator] += float64(t.outcomeScoredTs-t.firstSeen) / 3600.0
			lifeN[t.creator]++
		}
	}
	// skorlanmamış önce, sonra en-eski scored_ts (round-robin); eşitlikte address ASC
	// (postgres `ORDER BY c.scored_ts ASC NULLS FIRST, t.creator ASC` ile parite — map
	// iterasyonu rastgele olduğundan ikincil anahtar olmadan sıra deterministik değildi).
	addrs := make([]string, 0, len(byAddr))
	for addr := range byAddr {
		addrs = append(addrs, addr)
	}
	sort.SliceStable(addrs, func(i, j int) bool {
		si, sj := f.reputationByAddr[addrs[i]].ScoredTs, f.reputationByAddr[addrs[j]].ScoredTs
		if si != sj {
			return si < sj
		}
		return addrs[i] < addrs[j]
	})
	out := make([]CreatorAgg, 0, limit)
	for _, addr := range addrs {
		if len(out) >= limit {
			break
		}
		a := *byAddr[addr]
		if peakN[addr] > 0 {
			a.AvgPeakMarketCap = peakSum[addr] / float64(peakN[addr])
		}
		if lifeN[addr] > 0 {
			a.AvgLifetimeHours = lifeSum[addr] / float64(lifeN[addr])
		}
		out = append(out, a)
	}
	return out, nil
}

func (f *fakeTokenStore) UpsertReputation(_ context.Context, r CreatorReputation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reputationByAddr[r.Address] = r
	return nil
}
