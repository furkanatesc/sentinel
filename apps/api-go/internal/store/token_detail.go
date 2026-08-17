package store

// Bu struct'lar frontend TokenDetail (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.

type ScoreBreakdownItem struct {
	Label  string  `json:"label"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

type ScoreDetail struct {
	Key        string               `json:"key"`
	Value      float64              `json:"value"`
	Confidence float64              `json:"confidence"`
	UpdatedAt  string               `json:"updatedAt"`
	Breakdown  []ScoreBreakdownItem `json:"breakdown"`
}

type TokenMetrics struct {
	Holders           int     `json:"holders"`
	UniqueBuyers      int     `json:"uniqueBuyers"`
	BuyRatio          float64 `json:"buyRatio"`
	SellRatio         float64 `json:"sellRatio"`
	CreatorHoldingPct float64 `json:"creatorHoldingPct"`
	Top10HolderPct    float64 `json:"top10HolderPct"`
	SniperPct         float64 `json:"sniperPct"`
	BotActivityPct    float64 `json:"botActivityPct"`
}

type SeriesPoint struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

type TokenDetailSeries struct {
	Price     []SeriesPoint `json:"price"`
	Liquidity []SeriesPoint `json:"liquidity"`
	Volume    []SeriesPoint `json:"volume"`
	Holders   []SeriesPoint `json:"holders"`
}

type RiskItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Evidence    string `json:"evidence,omitempty"`
	FirstSeen   string `json:"firstSeen"`
	LastSeen    string `json:"lastSeen"`
}

type RiskGroups struct {
	Contract []RiskItem `json:"contract"`
	Market   []RiskItem `json:"market"`
	Creator  []RiskItem `json:"creator"`
}

type TokenDetail struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Symbol         string                 `json:"symbol"`
	Mint           string                 `json:"mint"`
	AgeSeconds     int64                  `json:"ageSeconds"`
	Price          float64                `json:"price"`
	PriceChange24h float64                `json:"priceChange24h"`
	MarketCap      float64                `json:"marketCap"`
	Liquidity      float64                `json:"liquidity"`
	Volume24h      float64                `json:"volume24h"`
	Scores         map[string]ScoreDetail `json:"scores"`
	Metrics        TokenMetrics           `json:"metrics"`
	Series         TokenDetailSeries      `json:"series"`
	Risks          RiskGroups             `json:"risks"`
}

// TokenDetailBase, getToken için gereken kimlik + havuz + persist edilmiş piyasa
// header'ıdır (mint kaydından). Header alanları enrichment'ta DB'ye yazılır → detail
// canlı GeckoTerminal çağrısı YAPMADAN header'ı DB'den sunar (paylaşımlı-IP throttle'a dayanıklı).
type TokenDetailBase struct {
	Name, Symbol, PoolAddr string
	FirstSeenTs            int64
	Price, Liquidity       float64
	PriceChangeH24         float64
	MarketCapUSD           float64
	Vol24h                 float64

	// 2a Token Safety (enrichment/scorer persist eder; detail okur).
	SafetyScore, SafetyConfidence, Top10Pct float64
	SafetyBreakdown                         []ScoreBreakdownItem
	SafetyRisks                             RiskGroups
	SafetyScoredTs                          int64

	// 2b-2b creator reputation (creators tablosundan; detail scores.creatorReputation'a).
	CreatorRepScore, CreatorRepConfidence float64
	CreatorRepBreakdown                   []ScoreBreakdownItem

	// 2c manipulation risk (tokens kolonlarından; detail scores.manipulationRisk'e) + işlem-akışı metrikleri.
	ManipulationScore, ManipulationConfidence float64
	ManipulationBreakdown                     []ScoreBreakdownItem
	ManipulationScoredTs                      int64
	TxnsBuys, TxnsSells, TxnsBuyers           int
	CreatorHoldingPct                         float64
}
