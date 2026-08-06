package market

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// DetailStore, getToken için tek token kimlik+havuz kaynağıdır (tüketici arayüzü; store.TokenStore karşılar).
type DetailStore interface {
	TokenDetailBase(ctx context.Context, mint string) (store.TokenDetailBase, bool, error)
}

// HoldersProvider, bir mint'in holder sayısını verir (tüketici arayüzü; ingest.HeliusHolders karşılar).
type HoldersProvider interface {
	HoldersCount(ctx context.Context, mint string, cap int) (count int, capped bool, err error)
}

type TokenDetailDeps struct {
	Store        DetailStore
	Provider     MarketProvider
	Holders      HoldersProvider
	CacheTTL     time.Duration // 0 → 20s
	OHLCVLimit   int           // 0 → 200
	HoldersCap   int           // 0 → 5000
	MinuteMaxAge int64         // 0 → 21600 (6h); bundan genç → minute, değilse hour
	Now          func() int64
	Logger       *slog.Logger
}

type cacheEntry struct {
	detail store.TokenDetail
	at     int64 // unix
}

// TokenDetailService, tek token'ın detayını market+holders+store'dan birleştirir (SRP).
type TokenDetailService struct {
	d     TokenDetailDeps
	mu    sync.Mutex
	cache map[string]cacheEntry
}

func NewTokenDetailService(d TokenDetailDeps) *TokenDetailService {
	if d.CacheTTL <= 0 {
		d.CacheTTL = 20 * time.Second
	}
	if d.OHLCVLimit <= 0 {
		d.OHLCVLimit = 200
	}
	if d.HoldersCap <= 0 {
		d.HoldersCap = 5000
	}
	if d.MinuteMaxAge <= 0 {
		d.MinuteMaxAge = 21600
	}
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &TokenDetailService{d: d, cache: map[string]cacheEntry{}}
}

func (s *TokenDetailService) Build(ctx context.Context, mint string) (store.TokenDetail, bool, error) {
	now := s.d.Now()
	s.mu.Lock()
	if e, ok := s.cache[mint]; ok && now-e.at < int64(s.d.CacheTTL/time.Second) {
		s.mu.Unlock()
		return e.detail, true, nil
	}
	s.mu.Unlock()

	base, ok, err := s.d.Store.TokenDetailBase(ctx, mint)
	if err != nil {
		return store.TokenDetail{}, false, err
	}
	if !ok || base.PoolAddr == "" {
		return store.TokenDetail{}, false, nil // bilinmeyen/pool'suz → 404
	}

	age := now - base.FirstSeenTs
	if age < 0 {
		age = 0
	}

	d := store.TokenDetail{
		ID: mint, Mint: mint, Name: base.Name, Symbol: base.Symbol, AgeSeconds: age,
		Scores: neutralScores(), Metrics: store.TokenMetrics{},
		Series: store.TokenDetailSeries{
			Price: []store.SeriesPoint{}, Liquidity: []store.SeriesPoint{},
			Volume: []store.SeriesPoint{}, Holders: []store.SeriesPoint{}},
		Risks: store.RiskGroups{Contract: []store.RiskItem{}, Market: []store.RiskItem{}, Creator: []store.RiskItem{}},
	}

	// Header (mevcut PoolsByAddresses yeniden kullanılır).
	if pools, err := s.d.Provider.PoolsByAddresses(ctx, []string{base.PoolAddr}); err != nil {
		s.d.Logger.Warn("detail header", "mint", mint, "err", err)
	} else {
		for _, p := range pools {
			if p.PoolAddr == base.PoolAddr {
				d.Price, d.Liquidity = p.Price, p.LiquidityUSD
				d.PriceChange24h, d.MarketCap, d.Volume24h = p.PriceChangeH24, p.MarketCapUSD, p.Vol24h
			}
		}
	}

	// Grafik (yaşa-uyarlı OHLCV).
	tf := "hour"
	if age < s.d.MinuteMaxAge {
		tf = "minute"
	}
	if candles, err := s.d.Provider.OHLCV(ctx, base.PoolAddr, tf, s.d.OHLCVLimit); err != nil {
		s.d.Logger.Warn("detail ohlcv", "mint", mint, "err", err)
	} else {
		for _, c := range candles {
			d.Series.Price = append(d.Series.Price, store.SeriesPoint{T: c.Ts, V: c.Close})
			d.Series.Volume = append(d.Series.Volume, store.SeriesPoint{T: c.Ts, V: c.Volume})
		}
	}

	// Holders (Helius, sınırlı).
	if n, capped, err := s.d.Holders.HoldersCount(ctx, mint, s.d.HoldersCap); err != nil {
		s.d.Logger.Warn("detail holders", "mint", mint, "err", err)
	} else {
		d.Metrics.Holders = n // capped ise cap = floor (seam number; "+" gösterimi yok)
		_ = capped
	}

	s.mu.Lock()
	s.cache[mint] = cacheEntry{detail: d, at: now}
	s.mu.Unlock()
	return d, true, nil
}

// neutralScores, 4 ScoreKey için dürüst nötr placeholder üretir (Alt-proje 2 gelene kadar).
func neutralScores() map[string]store.ScoreDetail {
	keys := []string{"opportunity", "creatorReputation", "tokenSafety", "manipulationRisk"}
	m := make(map[string]store.ScoreDetail, len(keys))
	for _, k := range keys {
		m[k] = store.ScoreDetail{Key: k, Value: 0, Confidence: 0, UpdatedAt: "—", Breakdown: []store.ScoreBreakdownItem{}}
	}
	return m
}
