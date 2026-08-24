package market

import (
	"context"
	"errors"
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
		// Header artık DB'den (base) okunuyor → provider pool header'ı KASITLI yanlış (yok sayılmalı).
		pools:   []Pool{{PoolAddr: "P1", Mint: "M1", Price: 999, LiquidityUSD: 999, PriceChangeH24: 999, MarketCapUSD: 999, Vol24h: 999}},
		candles: []Candle{{Ts: 1, Close: 4, Volume: 100}, {Ts: 2, Close: 5, Volume: 120}},
	}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0,
			Price: 5, Liquidity: 1000, PriceChangeH24: 12.5, MarketCapUSD: 90000, Vol24h: 40000},
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

// errProvider, PoolsByAddresses/OHLCV için her zaman hata döner (upstream kesinti simülasyonu).
type errProvider struct{}

func (errProvider) NewPools(context.Context) ([]Pool, error) { return nil, errUpstream }
func (errProvider) PoolsByAddresses(context.Context, []string) ([]Pool, error) {
	return nil, errUpstream
}
func (errProvider) OHLCV(context.Context, string, string, int) ([]Candle, error) {
	return nil, errUpstream
}

// errHolders, HoldersCount için her zaman hata döner.
type errHolders struct{}

func (errHolders) HoldersCount(context.Context, string, int) (int, bool, error) {
	return 0, false, errUpstream
}

var errUpstream = errors.New("upstream unavailable")

func TestBuildPartialFailureStillReturnsDetail(t *testing.T) {
	// Header DB'den (base) sunulur → GeckoTerminal TAMAMEN ölse bile header gerçek kalır;
	// yalnız OHLCV (canlı) ve holders (canlı) düşer. Option A'nın çekirdek garantisi.
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0,
			Price: 5, Liquidity: 1000, PriceChangeH24: 12.5, MarketCapUSD: 90000, Vol24h: 40000},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{
		Store: fs, Provider: errProvider{}, Holders: errHolders{},
		Now: func() int64 { return 300 },
	})
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("upstream hatası hard-fail olmamalı: ok=%v err=%v", ok, err)
	}
	if d.Symbol != "ONE" {
		t.Fatalf("kimlik store'dan gelmeli: symbol=%q want ONE", d.Symbol)
	}
	if d.Price != 5 || d.MarketCap != 90000 || d.Liquidity != 1000 || d.Volume24h != 40000 || d.PriceChange24h != 12.5 {
		t.Fatalf("header DB'den GERÇEK gelmeli (provider ölü olsa da): %+v", d)
	}
	if len(d.Series.Price) != 0 || len(d.Series.Volume) != 0 {
		t.Fatalf("OHLCV serisi canlı provider öldüğü için boş olmalı: %+v", d.Series)
	}
	if d.Metrics.Holders != 0 {
		t.Fatalf("holders nötr 0 kalmalı: %d", d.Metrics.Holders)
	}
	for _, k := range []string{"opportunity", "creatorReputation", "tokenSafety", "manipulationRisk"} {
		sd, ok := d.Scores[k]
		if !ok || sd.Value != 0 || sd.Key != k {
			t.Fatalf("nötr skor eksik/yanlış %q: %+v", k, sd)
		}
	}
	if d.Risks.Contract == nil || d.Risks.Market == nil || d.Risks.Creator == nil {
		t.Fatalf("risks boş slice olmalı (nil değil): %+v", d.Risks)
	}
	if d.Series.Price == nil || d.Series.Liquidity == nil || d.Series.Volume == nil || d.Series.Holders == nil {
		t.Fatalf("series boş slice olmalı (nil değil): %+v", d.Series)
	}
}

func TestBuildTokenSafetyFromStore(t *testing.T) {
	dp := &detailProvider{pools: []Pool{{PoolAddr: "P1", Mint: "M1"}}}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0,
			SafetyScore: 72, SafetyConfidence: 1, Top10Pct: 44, SafetyScoredTs: 500,
			SafetyBreakdown: []store.ScoreBreakdownItem{{Label: "Freeze authority iptal", Weight: 0, Detail: "ok"}},
			SafetyRisks:     store.RiskGroups{Contract: []store.RiskItem{}, Market: []store.RiskItem{{ID: "top10-concentration", Title: "x", Severity: "medium"}}, Creator: []store.RiskItem{}}},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{Store: fs, Provider: dp, Holders: &fakeHolders{n: 5}, Now: func() int64 { return 600 }})
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	sd := d.Scores["tokenSafety"]
	if sd.Value != 72 || sd.Confidence != 1 || len(sd.Breakdown) != 1 {
		t.Fatalf("tokenSafety skoru DB'den gelmeli: %+v", sd)
	}
	if d.Metrics.Top10HolderPct != 44 {
		t.Fatalf("top10HolderPct DB'den: %v", d.Metrics.Top10HolderPct)
	}
	if len(d.Risks.Market) != 1 || d.Risks.Market[0].ID != "top10-concentration" {
		t.Fatalf("safety market risk detaya taşınmalı: %+v", d.Risks.Market)
	}
	// Diğer 3 skor nötr kalmalı.
	if d.Scores["opportunity"].Confidence != 0 {
		t.Fatalf("opportunity nötr kalmalı: %+v", d.Scores["opportunity"])
	}
}

// TestBuildCreatorReputationFromStore, 2b-2b: scores.creatorReputation artık nötr placeholder
// değil, TokenDetailBase'in creator itibarı alanlarından (creators tablosu → LEFT JOIN) gelmeli;
// diğer 3 skor (tokenSafety hariç, o ayrı test) nötr kalmalı.
func TestBuildCreatorReputationFromStore(t *testing.T) {
	dp := &detailProvider{pools: []Pool{{PoolAddr: "P1", Mint: "M1"}}}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0,
			CreatorRepScore: 72, CreatorRepConfidence: 1,
			CreatorRepBreakdown: []store.ScoreBreakdownItem{{Label: "Başarı oranı", Weight: 0, Detail: "ok"}}},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{Store: fs, Provider: dp, Holders: &fakeHolders{n: 5}, Now: func() int64 { return 600 }})
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	sd := d.Scores["creatorReputation"]
	if sd.Value != 72 || sd.Confidence != 1 || len(sd.Breakdown) != 1 || sd.Key != "creatorReputation" {
		t.Fatalf("creatorReputation skoru DB'den gelmeli: %+v", sd)
	}
	// Diğer skorlar nötr kalmalı.
	if d.Scores["opportunity"].Confidence != 0 || d.Scores["tokenSafety"].Confidence != 0 || d.Scores["manipulationRisk"].Confidence != 0 {
		t.Fatalf("diğer 3 skor nötr kalmalı: %+v", d.Scores)
	}
}

// TestBuildManipulationRiskAndTxnFlowFromStore, 2c: scores.manipulationRisk artık nötr
// placeholder değil, TokenDetailBase'in manipülasyon alanlarından (tokens tablosu) gelmeli;
// metrics.uniqueBuyers/buyRatio/sellRatio/creatorHoldingPct txns_*+creator_holding_pct'ten türemeli.
func TestBuildManipulationRiskAndTxnFlowFromStore(t *testing.T) {
	dp := &detailProvider{pools: []Pool{{PoolAddr: "P1", Mint: "M1"}}}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0,
			ManipulationScore: 48, ManipulationConfidence: 0.7, ManipulationScoredTs: 99,
			ManipulationBreakdown: []store.ScoreBreakdownItem{{Label: "x", Weight: 48, Detail: "y"}},
			TxnsBuys:              70, TxnsSells: 30, TxnsBuyers: 25, CreatorHoldingPct: 33},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{Store: fs, Provider: dp, Holders: &fakeHolders{n: 5}, Now: func() int64 { return 600 }})
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	sd := d.Scores["manipulationRisk"]
	if sd.Value != 48 || sd.Confidence != 0.7 || len(sd.Breakdown) != 1 || sd.Key != "manipulationRisk" {
		t.Fatalf("manipulationRisk skoru DB'den gelmeli: %+v", sd)
	}
	if sd.UpdatedAt == "—" {
		t.Fatalf("manipulationRisk updatedAt scoredTs>0 iken gerçek zaman olmalı: %+v", sd)
	}
	if d.Metrics.UniqueBuyers != 25 {
		t.Fatalf("uniqueBuyers txns_buyers'tan gelmeli: %d", d.Metrics.UniqueBuyers)
	}
	if d.Metrics.BuyRatio != 0.7 || d.Metrics.SellRatio != 0.3 {
		t.Fatalf("buy/sellRatio txns_buys/sells'ten türemeli: buy=%v sell=%v", d.Metrics.BuyRatio, d.Metrics.SellRatio)
	}
	if d.Metrics.CreatorHoldingPct != 33 {
		t.Fatalf("creatorHoldingPct DB'den: %v", d.Metrics.CreatorHoldingPct)
	}
	// Diğer skorlar (opportunity) nötr kalmalı.
	if d.Scores["opportunity"].Confidence != 0 {
		t.Fatalf("opportunity nötr kalmalı: %+v", d.Scores["opportunity"])
	}
}

// TestBuildManipulationRiskNilBreakdown, nil breakdown + sıfır txns → dürüst-nötr guard'lar
// (boş dilim, sıfır bölme yok) doğrulanmalı.
func TestBuildManipulationRiskNilBreakdown(t *testing.T) {
	dp := &detailProvider{pools: []Pool{{PoolAddr: "P1", Mint: "M1"}}}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{Store: fs, Provider: dp, Holders: &fakeHolders{n: 5}, Now: func() int64 { return 600 }})
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	sd := d.Scores["manipulationRisk"]
	if sd.Breakdown == nil || len(sd.Breakdown) != 0 {
		t.Fatalf("nil breakdown → boş dilim olmalı (nil değil): %+v", sd.Breakdown)
	}
	if sd.UpdatedAt != "—" {
		t.Fatalf("scoredTs=0 iken updatedAt=— olmalı: %q", sd.UpdatedAt)
	}
	if d.Metrics.BuyRatio != 0 || d.Metrics.SellRatio != 0 {
		t.Fatalf("txns=0 iken buy/sellRatio 0 kalmalı (sıfıra bölme yok): buy=%v sell=%v", d.Metrics.BuyRatio, d.Metrics.SellRatio)
	}
}

// TestBuild_OpportunityFromDB, 2d: scores.opportunity artık nötr placeholder değil,
// TokenDetailBase'in opportunity alanlarından (tokens tablosu, arka plan worker persist etti) gelmeli.
func TestBuild_OpportunityFromDB(t *testing.T) {
	dp := &detailProvider{pools: []Pool{{PoolAddr: "P1", Mint: "M1"}}}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0,
			OpportunityScore: 72, OpportunityConfidence: 0.8, OpportunityScoredTs: 99,
			OpportunityBreakdown: []store.ScoreBreakdownItem{{Label: "x", Weight: 72, Detail: "y"}}},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{Store: fs, Provider: dp, Holders: &fakeHolders{n: 5}, Now: func() int64 { return 600 }})
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	sd := d.Scores["opportunity"]
	if sd.Value != 72 || sd.Confidence != 0.8 || sd.Breakdown == nil || len(sd.Breakdown) != 1 || sd.Key != "opportunity" {
		t.Fatalf("opportunity skoru DB'den gelmeli: %+v", sd)
	}
	if sd.UpdatedAt == "—" {
		t.Fatalf("opportunity updatedAt scoredTs>0 iken gerçek zaman olmalı: %+v", sd)
	}
}

func TestBuildCache(t *testing.T) {
	dp := &detailProvider{pools: []Pool{{PoolAddr: "P1", Mint: "M1"}}}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0, Price: 5, Liquidity: 1000},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{
		Store: fs, Provider: dp, Holders: &fakeHolders{n: 1}, Now: func() int64 { return 300 },
	})
	svc.Build(context.Background(), "M1")
	fs.base["M1"] = store.TokenDetailBase{Name: "One", Symbol: "ONE", PoolAddr: "P1", Price: 999} // değişti; cache eskisini vermeli
	d, _, _ := svc.Build(context.Background(), "M1")
	if d.Price != 5 {
		t.Fatalf("cache çalışmadı: price=%v want 5 (ilk çağrı TTL boyunca cache'lenmeli)", d.Price)
	}
}
