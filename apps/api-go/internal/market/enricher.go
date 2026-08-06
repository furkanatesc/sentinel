package market

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

const maxPoolBatch = 30 // GeckoTerminal pools/multi başına en fazla adres

type EnricherDeps struct {
	Provider  MarketProvider
	Tokens    store.TokenStore
	Broadcast Broadcaster
	Interval  time.Duration
	Limit     int // enrich edilecek en yeni token sayısı + snapshot penceresi
	Logger    *slog.Logger
}

// Enricher, bilinen (havuzlu) token'ların piyasa alanlarını periyodik günceller (SRP).
type Enricher struct{ d EnricherDeps }

func NewEnricher(d EnricherDeps) *Enricher {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Limit <= 0 {
		d.Limit = 60
	}
	if d.Interval <= 0 {
		d.Interval = 30 * time.Second
	}
	return &Enricher{d: d}
}

func (x *Enricher) Run(ctx context.Context) {
	x.d.Logger.Info("market enricher başladı", "interval", x.d.Interval.String(), "limit", x.d.Limit)
	t := time.NewTicker(x.d.Interval)
	defer t.Stop()
	for {
		if err := x.tick(ctx); err != nil {
			x.d.Logger.Warn("enricher tick", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// tick, hedefleri okur, havuz verisini batch çeker, piyasa alanlarını + spark'ı günceller, snapshot yayınlar.
func (x *Enricher) tick(ctx context.Context) error {
	targets, err := x.d.Tokens.EnrichTargets(ctx, x.d.Limit)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}
	byPool := map[string]store.EnrichTarget{}
	addrs := make([]string, 0, len(targets))
	for _, t := range targets {
		byPool[t.PoolAddr] = t
		addrs = append(addrs, t.PoolAddr)
	}
	var updated bool
	for _, batch := range chunk(addrs, maxPoolBatch) {
		pools, err := x.d.Provider.PoolsByAddresses(ctx, batch)
		if err != nil {
			x.d.Logger.Warn("pools by addresses", "err", err)
			continue // kısmi başarı: bu batch atla, diğerleri devam
		}
		for _, p := range pools {
			t, ok := byPool[p.PoolAddr]
			if !ok {
				continue
			}
			if err := x.d.Tokens.UpdateMarket(ctx, store.MarketUpdate{
				Mint: t.Mint, Price: p.Price, Liquidity: p.LiquidityUSD, Vol5m: p.Vol5m,
				Momentum: momentumFromChange(p.PriceChangeH1), Spark: appendSpark(t.Spark, p.Price),
			}); err != nil {
				x.d.Logger.Warn("update market", "mint", t.Mint, "err", err)
				continue
			}
			updated = true
		}
	}
	if updated {
		snapshot, err := x.d.Tokens.RecentTokens(ctx, x.d.Limit)
		if err != nil {
			return err
		}
		x.d.Broadcast.Broadcast("tokens", snapshot)
	}
	return nil
}

// chunk, dilimi en fazla size'lık parçalara böler.
func chunk(s []string, size int) [][]string {
	var out [][]string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
