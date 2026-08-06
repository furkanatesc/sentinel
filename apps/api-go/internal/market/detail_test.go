package market

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeDetailStore struct {
	base map[string]store.TokenDetailBase
}

func (f *fakeDetailStore) TokenDetailBase(_ context.Context, mint string) (store.TokenDetailBase, bool, error) {
	b, ok := f.base[mint]
	return b, ok, nil
}

type fakeHolders struct {
	n      int
	capped bool
}

func (f *fakeHolders) HoldersCount(_ context.Context, _ string, _ int) (int, bool, error) {
	return f.n, f.capped, nil
}

// fakeProvider (discoverer_test.go) NewPools/PoolsByAddresses sağlar; OHLCV'yi burada genişletiyoruz.
type detailProvider struct {
	pools   []Pool
	candles []Candle
}

func (d *detailProvider) NewPools(context.Context) ([]Pool, error) { return d.pools, nil }
func (d *detailProvider) PoolsByAddresses(context.Context, []string) ([]Pool, error) {
	return d.pools, nil
}
func (d *detailProvider) OHLCV(context.Context, string, string, int) ([]Candle, error) {
	return d.candles, nil
}

func newDetailSvc(ageSeconds int64) (*TokenDetailService, *detailProvider) {
	dp := &detailProvider{
		pools:   []Pool{{PoolAddr: "P1", Mint: "M1", Price: 5, LiquidityUSD: 1000, Vol5m: 10, PriceChangeH24: 12.5, MarketCapUSD: 90000, Vol24h: 40000}},
		candles: []Candle{{Ts: 1, Close: 4, Volume: 100}, {Ts: 2, Close: 5, Volume: 120}},
	}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{
		Store: fs, Provider: dp, Holders: &fakeHolders{n: 312},
		Now: func() int64 { return ageSeconds }, // first_seen=0 → ageSeconds
	})
	return svc, dp
}

func TestBuildMapsRealFields(t *testing.T) {
	svc, _ := newDetailSvc(300)
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if d.Symbol != "ONE" || d.Price != 5 || d.PriceChange24h != 12.5 || d.MarketCap != 90000 || d.Liquidity != 1000 || d.Volume24h != 40000 {
		t.Fatalf("header yanlış: %+v", d)
	}
	if d.Metrics.Holders != 312 {
		t.Fatalf("holders=%d want 312", d.Metrics.Holders)
	}
	if len(d.Series.Price) != 2 || d.Series.Price[1].V != 5 || len(d.Series.Volume) != 2 {
		t.Fatalf("series yanlış: %+v", d.Series)
	}
}

func TestBuildNeutralPlaceholders(t *testing.T) {
	svc, _ := newDetailSvc(300)
	d, _, _ := svc.Build(context.Background(), "M1")
	for _, k := range []string{"opportunity", "creatorReputation", "tokenSafety", "manipulationRisk"} {
		sd, ok := d.Scores[k]
		if !ok || sd.Value != 0 || sd.Key != k {
			t.Fatalf("nötr skor eksik/yanlış %q: %+v", k, sd)
		}
	}
	if d.Risks.Contract == nil || d.Risks.Market == nil || d.Risks.Creator == nil {
		t.Fatalf("risks boş slice olmalı (nil değil): %+v", d.Risks)
	}
	if d.Series.Liquidity == nil || d.Series.Holders == nil {
		t.Fatal("liquidity/holders serisi boş slice olmalı (nil değil)")
	}
	if d.Metrics.BuyRatio != 0 || d.Metrics.Top10HolderPct != 0 {
		t.Fatal("davranış metrikleri nötr 0 olmalı")
	}
}

func TestBuildAgeAdaptiveTimeframe(t *testing.T) {
	// Genç (<6h) → minute; yaşlı → hour. detailProvider tf'i kaydetmiyor; ayrı casus provider.
	spy := &tfSpyProvider{}
	svc := NewTokenDetailService(TokenDetailDeps{
		Store:    &fakeDetailStore{base: map[string]store.TokenDetailBase{"M1": {PoolAddr: "P1"}}},
		Provider: spy, Holders: &fakeHolders{}, Now: func() int64 { return 100 }, // age=100s (genç)
	})
	svc.Build(context.Background(), "M1")
	if spy.tf != "minute" {
		t.Fatalf("genç token tf=%q want minute", spy.tf)
	}
	spy2 := &tfSpyProvider{}
	svc2 := NewTokenDetailService(TokenDetailDeps{
		Store:    &fakeDetailStore{base: map[string]store.TokenDetailBase{"M1": {PoolAddr: "P1"}}},
		Provider: spy2, Holders: &fakeHolders{}, Now: func() int64 { return 100000 }, // age=100000s (>6h)
	})
	svc2.Build(context.Background(), "M1")
	if spy2.tf != "hour" {
		t.Fatalf("yaşlı token tf=%q want hour", spy2.tf)
	}
}

type tfSpyProvider struct{ tf string }

func (p *tfSpyProvider) NewPools(context.Context) ([]Pool, error) { return nil, nil }
func (p *tfSpyProvider) PoolsByAddresses(_ context.Context, _ []string) ([]Pool, error) {
	return []Pool{{PoolAddr: "P1"}}, nil
}
func (p *tfSpyProvider) OHLCV(_ context.Context, _, tf string, _ int) ([]Candle, error) {
	p.tf = tf
	return nil, nil
}

func TestBuildUnknownMint(t *testing.T) {
	svc, _ := newDetailSvc(300)
	_, ok, err := svc.Build(context.Background(), "YOK")
	if err != nil || ok {
		t.Fatalf("bilinmeyen mint ok=false olmalı: ok=%v err=%v", ok, err)
	}
}

func TestBuildCache(t *testing.T) {
	svc, dp := newDetailSvc(300)
	svc.Build(context.Background(), "M1")
	dp.pools = []Pool{{PoolAddr: "P1", Mint: "M1", Price: 999}} // değişti; cache eskisini vermeli
	d, _, _ := svc.Build(context.Background(), "M1")
	if d.Price != 5 {
		t.Fatalf("cache çalışmadı: price=%v want 5 (ilk çağrı cache'lenmeli)", d.Price)
	}
}
