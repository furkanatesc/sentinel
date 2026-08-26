package store

import (
	"context"
	"sort"
	"sync"
	"time"
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
	// 2c manipülasyon riski: txns_* Task 4'te, creatorHoldingPct Task 5'te yazılır — bu task
	// sadece alanları + okuma/yazma yüzeyini açar (bkz. ManipulationTargets/UpdateManipulation).
	txnsBuys, txnsSells, txnsBuyers, txnsSellers int
	creatorHoldingPct                            float64
	manipScore, manipConf                        float64
	manipBreakdown                               []ScoreBreakdownItem
	manipScoredTs                                int64
	// 2d: kompozit opportunity skoru + türetilmiş signal (last_signal parity; boş → nil/null).
	signal                            string
	opportunityScore, opportunityConf float64
	opportunityBreakdown              []ScoreBreakdownItem
	opportunityScoredTs               int64
	// 2e-2 authority pubkey (piggyback)
	mintAuthority, freezeAuthority string
}

type fakeTokenStore struct {
	mu    sync.Mutex
	byID  map[string]fakeTok
	order []string // ekleme sırası
	// 2b-2b: hesaplanmış creator itibarı (adres → son itibar).
	reputationByAddr map[string]CreatorReputation
	// highDrawdownThreshold, deriveRiskFlags'in "Yüksek düşüş" eşiğidir (2b-2b:
	// cfg.ReputationHighDrawdown, bkz. WithHighDrawdownThreshold); <=0 → paket varsayılanı (80).
	highDrawdownThreshold float64
	// 2e-1: wallet_funders parity (wallet → funder + resolved_ts; resolved_ts>0 → çözülmüş,
	// funder="" not-found işareti dahil).
	walletFunders map[string]struct {
		funder     string
		resolvedTs int64
	}
}

// NewFakeTokenStore, testler ve DB'siz mod için in-memory TokenStore döndürür. opts (ör.
// WithHighDrawdownThreshold) postgresStore ile aynı creatorStoreConfig'i paylaşır (parite);
// verilmezse paket varsayılanları geçerli olur — mevcut sıfır-argümanlı çağrılar kırılmaz.
func NewFakeTokenStore(opts ...CreatorStoreOption) TokenStore {
	cfg := applyCreatorStoreOptions(opts)
	return &fakeTokenStore{
		byID: map[string]fakeTok{}, reputationByAddr: map[string]CreatorReputation{},
		highDrawdownThreshold: cfg.highDrawdownThreshold,
		walletFunders: map[string]struct {
			funder     string
			resolvedTs int64
		}{},
	}
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
			tk.marketCapUSD, tk.peakMarketCap, tk.maxDrawdownPct, tk.outcome, tk.liquidityStatus, f.highDrawdownThreshold))
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
	cur.txnsBuys, cur.txnsSells = m.TxnsBuys, m.TxnsSells
	cur.txnsBuyers, cur.txnsSellers = m.TxnsBuyers, m.TxnsSellers
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
	// 2b-2b: creator itibarı (postgres LEFT JOIN creators parite); skorlanmamış/creator'sız → nötr 0/0/boş.
	rep := f.reputationByAddr[t.creator]
	repBreakdown := rep.Breakdown
	if repBreakdown == nil {
		repBreakdown = []ScoreBreakdownItem{}
	}
	return TokenDetailBase{
		Name: t.row.Name, Symbol: t.row.Symbol, PoolAddr: t.poolAddr, FirstSeenTs: t.firstSeen,
		Price: t.row.Price, Liquidity: t.row.Liquidity,
		PriceChangeH24: t.priceChangeH24, MarketCapUSD: t.marketCapUSD, Vol24h: t.vol24h,
		SafetyScore: t.safetyScore, SafetyConfidence: t.safetyConfidence, Top10Pct: t.top10Pct,
		SafetyBreakdown: t.safetyBreakdown, SafetyRisks: t.safetyRisks, SafetyScoredTs: t.safetyScoredTs,
		CreatorRepScore: rep.Score, CreatorRepConfidence: rep.Confidence, CreatorRepBreakdown: repBreakdown,
		ManipulationScore: t.manipScore, ManipulationConfidence: t.manipConf,
		ManipulationBreakdown: breakdownOrEmpty(t.manipBreakdown), ManipulationScoredTs: t.manipScoredTs,
		TxnsBuys: t.txnsBuys, TxnsSells: t.txnsSells, TxnsBuyers: t.txnsBuyers,
		CreatorHoldingPct: t.creatorHoldingPct,
		OpportunityScore:  t.opportunityScore, OpportunityConfidence: t.opportunityConf,
		OpportunityBreakdown: breakdownOrEmpty(t.opportunityBreakdown), OpportunityScoredTs: t.opportunityScoredTs,
	}, true, nil
}

// breakdownOrEmpty, nil breakdown'ı boş dilime çevirir (postgres COALESCE parite; dürüst-nötr; 2c/2d ortak).
func breakdownOrEmpty(b []ScoreBreakdownItem) []ScoreBreakdownItem {
	if b == nil {
		return []ScoreBreakdownItem{}
	}
	return b
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
	if s.CreatorHoldingKnown {
		cur.creatorHoldingPct = s.CreatorHoldingPct
	}
	if s.AuthoritiesKnown {
		cur.mintAuthority, cur.freezeAuthority = s.MintAuthority, s.FreezeAuthority
	}
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
		out = append(out, SafetyTarget{Mint: t.row.Mint, Liquidity: t.row.Liquidity, Launchpad: t.launchpad, Creator: t.creator})
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
		tk := f.byID[f.order[i]]
		row := tk.row
		// 2b-2b: creatorScore artık nötr değil — creator itibarından (postgres LEFT JOIN creators parite;
		// skorlanmamış/creator'sız → 0, COALESCE(c.reputation_score,0) ile eşleşir).
		row.CreatorScore = f.reputationByAddr[tk.creator].Score
		// 2d: last_signal parity — boş → nil (frontend null), doluysa *string.
		if tk.signal != "" {
			signal := tk.signal
			row.Signal = &signal
		} else {
			row.Signal = nil
		}
		out = append(out, row)
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

func (f *fakeTokenStore) UpdateManipulation(_ context.Context, u ManipulationUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[u.Mint]
	if !ok {
		return nil
	}
	cur.manipScore, cur.manipConf = u.Score, u.Confidence
	cur.manipBreakdown, cur.manipScoredTs = u.Breakdown, u.ScoredTs
	f.byID[u.Mint] = cur
	return nil
}

func (f *fakeTokenStore) ManipulationTargets(_ context.Context, limit int) ([]ManipulationTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].manipScoredTs < f.byID[ids[j]].manipScoredTs
	})
	out := make([]ManipulationTarget, 0, limit)
	for _, id := range ids {
		t := f.byID[id]
		if t.poolAddr == "" || len(out) >= limit {
			continue
		}
		out = append(out, ManipulationTarget{
			Mint: t.row.Mint, Buys: t.txnsBuys, Sells: t.txnsSells, Buyers: t.txnsBuyers,
			CreatorHoldingPct: t.creatorHoldingPct, Vol24h: t.vol24h, Liquidity: t.row.Liquidity,
		})
	}
	return out, nil
}

// OpportunityScoreTargets, postgres `t LEFT JOIN creators c` semantiğini birebir taklit eder:
// tüm token'lar (pool_address filtresi YOK — postgres sorgusu da filtrelemiyor), creator'sız/
// skorlanmamış creator → reputation/confidence 0 (COALESCE parite).
func (f *fakeTokenStore) OpportunityScoreTargets(_ context.Context, limit int) ([]OpportunityTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]OpportunityTarget, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- { // en yeni önce (first_seen_ts DESC)
		t := f.byID[f.order[i]]
		rep := f.reputationByAddr[t.creator]
		out = append(out, OpportunityTarget{
			Mint: t.row.Mint, Safety: t.safetyScore, SafetyConf: t.safetyConfidence,
			Creator: rep.Score, CreatorConf: rep.Confidence,
			Manipulation: t.manipScore, ManipulationConf: t.manipConf,
			Momentum: t.row.Momentum, Liquidity: t.row.Liquidity,
		})
	}
	return out, nil
}

func (f *fakeTokenStore) UpdateOpportunity(_ context.Context, u OpportunityUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[u.Mint]
	if !ok {
		return nil
	}
	cur.opportunityScore, cur.opportunityConf = u.Score, u.Confidence
	cur.opportunityBreakdown, cur.opportunityScoredTs = u.Breakdown, u.ScoredTs
	cur.signal = u.Signal
	f.byID[u.Mint] = cur
	return nil
}

// Kpis, postgresStore.Kpis ile AYNI eşikler+boolean mantığıyla (24s cutoff = now-86400,
// highConf/critical/signals eşikleri) in-memory token'ları sayar (parite).
func (f *fakeTokenStore) Kpis(_ context.Context) (KpiCounts, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var c KpiCounts
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	for _, id := range f.order {
		t := f.byID[id]
		if t.firstSeen >= cutoff {
			c.Detected++
		}
		if t.safetyScore >= 70 && t.safetyConfidence >= 0.5 {
			c.HighConf++
		}
		if (t.manipScore >= 70 && t.manipConf >= 0.5) || (t.safetyScore <= 30 && t.safetyConfidence >= 0.5) {
			c.Critical++
		}
		if t.signal == "buy" || t.signal == "watch" {
			c.Signals++
		}
	}
	return c, nil
}

func (f *fakeTokenStore) Radar(ctx context.Context, limit int) ([]RadarPoint, error) {
	rows, err := f.RecentTokens(ctx, limit)
	if err != nil {
		return nil, err
	}
	return radarFrom(rows), nil
}

// FunderTargets, postgresStore.FunderTargets ile parite: fake token'lardaki distinct
// non-empty creator'lardan resolved_ts>0 OLMAYANLAR (funder="" not-found işareti de çözülmüş
// sayılır — postgres semantiğiyle birebir).
func (f *fakeTokenStore) FunderTargets(_ context.Context, limit int) ([]FunderTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	seen := map[string]bool{}
	creators := make([]string, 0, len(f.order))
	for _, id := range f.order {
		c := f.byID[id].creator
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		creators = append(creators, c)
	}
	sort.Strings(creators) // postgres ORDER BY t.creator parity
	out := make([]FunderTarget, 0, limit)
	for _, c := range creators {
		if len(out) >= limit {
			break
		}
		if wf, ok := f.walletFunders[c]; ok && wf.resolvedTs > 0 {
			continue
		}
		out = append(out, FunderTarget{Wallet: c})
	}
	return out, nil
}

func (f *fakeTokenStore) SetFunder(_ context.Context, wallet, funder string, resolvedTs int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.walletFunders[wallet] = struct {
		funder     string
		resolvedTs int64
	}{funder: funder, resolvedTs: resolvedTs}
	return nil
}

// WalletGraphClusters, postgresStore.WalletGraphClusters ile AYNI semantik: funder→creator
// (wallet_funders) + creator→token (tokens.creator) üzerinden funder başına DISTINCT creator
// sayısı (degree). minCluster<=degree<=maxDegree olan funder'ların TÜM (funder,creator,mint,
// symbol,safety,reputation,firstSeen) kenarları döner (postgres CTE parity).
func (f *fakeTokenStore) WalletGraphClusters(_ context.Context, minCluster, maxDegree int) ([]ClusterRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// funder → distinct creator set (degree hesabı).
	creatorsByFunder := map[string]map[string]bool{}
	for _, id := range f.order {
		t := f.byID[id]
		if t.creator == "" {
			continue
		}
		wf, ok := f.walletFunders[t.creator]
		if !ok || wf.funder == "" {
			continue
		}
		set, ok := creatorsByFunder[wf.funder]
		if !ok {
			set = map[string]bool{}
			creatorsByFunder[wf.funder] = set
		}
		set[t.creator] = true
	}
	qualifying := map[string]bool{}
	for funder, set := range creatorsByFunder {
		degree := len(set)
		if degree >= minCluster && degree <= maxDegree {
			qualifying[funder] = true
		}
	}
	out := []ClusterRow{}
	for _, id := range f.order {
		t := f.byID[id]
		if t.creator == "" {
			continue
		}
		wf, ok := f.walletFunders[t.creator]
		if !ok || wf.funder == "" || !qualifying[wf.funder] {
			continue
		}
		out = append(out, ClusterRow{
			Funder: wf.funder, Creator: t.creator, Mint: t.row.Mint, Symbol: t.row.Symbol,
			SafetyScore: t.safetyScore, ReputationScore: f.reputationByAddr[t.creator].Score,
			FirstSeenTs: t.firstSeen,
		})
	}
	return out, nil
}

// AuthorityGraphClusters, postgresStore.AuthorityGraphClusters ile AYNI semantik: mint+freeze
// authority'lerini (authority,mint,role) satırlarına unpivot eder, authority başına DISTINCT mint
// sayar (degree). minCluster<=degree<=maxDegree olan authority'lerin TÜM ham (mint/freeze ayrı,
// "both" birleştirmesi YOK — Task 5'te) kenarları döner (postgres CTE parity).
func (f *fakeTokenStore) AuthorityGraphClusters(_ context.Context, minCluster, maxDegree int) ([]AuthorityRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// (authority, role) → token satırları topla + authority başına distinct mint say.
	type ar struct {
		authority, mint, symbol, role string
		safety                        float64
		firstSeen                     int64
	}
	var all []ar
	mints := map[string]map[string]bool{}
	add := func(authority, role string, t fakeTok) {
		if authority == "" {
			return
		}
		all = append(all, ar{authority, t.row.Mint, t.row.Symbol, role, t.safetyScore, t.firstSeen})
		if mints[authority] == nil {
			mints[authority] = map[string]bool{}
		}
		mints[authority][t.row.Mint] = true
	}
	for _, id := range f.order {
		t := f.byID[id]
		add(t.mintAuthority, "mint", t)
		add(t.freezeAuthority, "freeze", t)
	}
	out := []AuthorityRow{}
	for _, a := range all {
		deg := len(mints[a.authority])
		if deg >= minCluster && deg <= maxDegree {
			out = append(out, AuthorityRow{Authority: a.authority, Mint: a.mint, Symbol: a.symbol, Role: a.role, SafetyScore: a.safety, FirstSeenTs: a.firstSeen})
		}
	}
	return out, nil
}
