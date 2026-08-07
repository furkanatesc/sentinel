package market

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// DiscovererDeps, Discoverer bağımlılıklarıdır (hepsi enjekte edilebilir — test determinizmi).
type DiscovererDeps struct {
	Provider      MarketProvider
	Tokens        store.TokenStore
	Events        store.EventStore
	Broadcast     Broadcaster
	Interval      time.Duration
	SnapshotLimit int
	Now           func() int64
	Logger        *slog.Logger
}

// Discoverer, GeckoTerminal new_pools'u periyodik tarayıp yeni token'ları keşfeder (SRP).
type Discoverer struct{ d DiscovererDeps }

func NewDiscoverer(d DiscovererDeps) *Discoverer {
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.SnapshotLimit <= 0 {
		d.SnapshotLimit = 200
	}
	if d.Interval <= 0 {
		d.Interval = 30 * time.Second
	}
	return &Discoverer{d: d}
}

// Run, Interval periyoduyla tick çağırır; ctx iptaline kadar.
func (x *Discoverer) Run(ctx context.Context) {
	x.d.Logger.Info("market discoverer başladı", "interval", x.d.Interval.String())
	t := time.NewTicker(x.d.Interval)
	defer t.Stop()
	for {
		if err := x.tick(ctx); err != nil {
			x.d.Logger.Warn("discoverer tick", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// tick, tek tarama: yeni havuzları keşfet, yeni token'lar için kimlik+ilk enrichment+olay yaz, snapshot yayınla.
func (x *Discoverer) tick(ctx context.Context) error {
	pools, err := x.d.Provider.NewPools(ctx)
	if err != nil {
		return err
	}
	now := x.d.Now()
	var wrote bool
	for _, p := range pools {
		launchpad, ok := DexToLaunchpad(p.Dex)
		if !ok {
			continue
		}
		firstSeen := p.CreatedAtUnix
		if firstSeen == 0 {
			firstSeen = now
		}
		inserted, err := x.d.Tokens.UpsertDiscovered(ctx, store.DiscoveredToken{
			Mint: p.Mint, Name: p.Name, Symbol: p.Symbol, Launchpad: launchpad, PoolAddr: p.PoolAddr, FirstSeenTs: firstSeen,
		})
		if err != nil {
			x.d.Logger.Warn("upsert discovered", "mint", p.Mint, "err", err)
			continue
		}
		if inserted { // yalnız ilk keşifte enrichment+olay (Enricher sonraki güncellemelerin sahibi; spam yok)
			// Keşifte bedava ilk enrichment — yalnız ilk keşifte (sonraki güncellemeler Enricher'ın).
			if err := x.d.Tokens.UpdateMarket(ctx, store.MarketUpdate{
				Mint: p.Mint, Price: p.Price, Liquidity: p.LiquidityUSD, Vol5m: p.Vol5m,
				Momentum: momentumFromChange(p.PriceChangeH1), Spark: appendSpark(nil, p.Price),
				PriceChangeH24: p.PriceChangeH24, MarketCapUSD: p.MarketCapUSD, Vol24h: p.Vol24h,
			}); err != nil {
				x.d.Logger.Warn("initial market", "mint", p.Mint, "err", err)
			}
			wrote = true
			ev := store.EventRow{
				ID: p.PoolAddr + "|pool_created", Type: "pool_created", Symbol: p.Symbol, Mint: p.Mint,
				Launchpad: launchpad, DEX: launchpad, Liquidity: p.LiquidityUSD, RiskLevel: "medium",
				TokenAgeSeconds: now - firstSeen, Volume5m: p.Vol5m, Severity: "info",
				Detail: fmt.Sprintf("%s havuzu keşfedildi", p.Symbol), Ts: now,
			}
			if err := x.d.Events.InsertEvent(ctx, ev); err != nil {
				x.d.Logger.Warn("insert event", "err", err)
			} else {
				x.d.Broadcast.Broadcast("events", ev)
			}
		}
	}
	if wrote {
		snapshot, err := x.d.Tokens.RecentTokens(ctx, x.d.SnapshotLimit)
		if err != nil {
			return err
		}
		x.d.Broadcast.Broadcast("tokens", snapshot)
	}
	return nil
}
