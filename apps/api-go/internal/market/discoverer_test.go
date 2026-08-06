package market

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeProvider struct {
	newPools []Pool
	byAddr   []Pool
}

func (f *fakeProvider) NewPools(context.Context) ([]Pool, error) { return f.newPools, nil }
func (f *fakeProvider) PoolsByAddresses(_ context.Context, _ []string) ([]Pool, error) {
	return f.byAddr, nil
}

type capBC struct {
	topics   []string
	payloads []any
}

func (c *capBC) Broadcast(topic string, payload any) {
	c.topics = append(c.topics, topic)
	c.payloads = append(c.payloads, payload)
}

func newDiscoverer(fp MarketProvider) (*Discoverer, store.TokenStore, store.EventStore, *capBC) {
	ts, es, bc := store.NewFakeTokenStore(), store.NewFakeEventStore(), &capBC{}
	d := NewDiscoverer(DiscovererDeps{
		Provider: fp, Tokens: ts, Events: es, Broadcast: bc,
		SnapshotLimit: 50, Now: func() int64 { return 1000 },
	})
	return d, ts, es, bc
}

func TestDiscovererWritesTokenEventAndSnapshot(t *testing.T) {
	fp := &fakeProvider{newPools: []Pool{{
		PoolAddr: "P1", Mint: "M1", Name: "One", Symbol: "ONE", Dex: "pumpfun",
		Price: 2, LiquidityUSD: 1000, Vol5m: 50, PriceChangeH1: 20, CreatedAtUnix: 900,
	}}}
	d, ts, es, bc := newDiscoverer(fp)
	if err := d.tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	toks, _ := ts.RecentTokens(context.Background(), 10)
	if len(toks) != 1 || toks[0].Symbol != "ONE" || toks[0].Price != 2 || toks[0].Momentum == 0 {
		t.Fatalf("token yazılmadı/enrich edilmedi: %+v", toks)
	}
	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 1 || evs[0].Type != "pool_created" || evs[0].Launchpad != "Pump.fun" || evs[0].Mint != "M1" {
		t.Fatalf("olay yanlış: %+v", evs)
	}
	var tokensSnap int
	for i, topic := range bc.topics {
		if topic == "tokens" {
			if _, ok := bc.payloads[i].([]store.TokenRow); !ok {
				t.Fatalf("tokens payload []store.TokenRow olmalı, got %T", bc.payloads[i])
			}
			tokensSnap++
		}
	}
	if tokensSnap == 0 {
		t.Fatal("tokens snapshot broadcast edilmedi")
	}
}

func TestDiscovererDedupSecondTickNoNewEvent(t *testing.T) {
	fp := &fakeProvider{newPools: []Pool{{PoolAddr: "P1", Mint: "M1", Symbol: "ONE", Dex: "pumpfun", CreatedAtUnix: 900}}}
	d, _, es, _ := newDiscoverer(fp)
	d.tick(context.Background())
	d.tick(context.Background()) // aynı havuz → yeni olay YOK
	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 1 {
		t.Fatalf("olaylar=%d, want 1 (dedup: yalnız ilk keşifte olay)", len(evs))
	}
}

// TestDiscovererDoesNotReenrichOnSecondTick, ilk keşif enrichment'inin yalnız ilk tick'te
// çalıştığını kanıtlar: Enricher'ın biriktirdiği spark geçmişi, discoverer'ın tekrarlayan
// tick'lerinde appendSpark(nil, ...) ile tek örneğe sıfırlanmamalı (regression: bug fix).
func TestDiscovererDoesNotReenrichOnSecondTick(t *testing.T) {
	fp := &fakeProvider{newPools: []Pool{{
		PoolAddr: "P1", Mint: "M1", Name: "One", Symbol: "ONE", Dex: "pumpfun",
		Price: 2, LiquidityUSD: 1000, Vol5m: 50, PriceChangeH1: 20, CreatedAtUnix: 900,
	}}}
	d, ts, _, _ := newDiscoverer(fp)
	ctx := context.Background()
	if err := d.tick(ctx); err != nil {
		t.Fatal(err)
	}
	// Enricher, ilk keşiften sonra bağımsız olarak spark geçmişini büyütür (simülasyon).
	simulatedSpark := []float64{2, 2.1, 2.2, 2.3, 2.4}
	if err := ts.UpdateMarket(ctx, store.MarketUpdate{
		Mint: "M1", Price: 2.4, Liquidity: 1200, Vol5m: 60, Momentum: 70, Spark: simulatedSpark,
	}); err != nil {
		t.Fatal(err)
	}
	// İkinci tick: havuz zaten biliniyor (inserted=false) → discoverer UpdateMarket'i
	// TEKRAR çalıştırmamalı; aksi halde Enricher'ın spark geçmişi tek örneğe düşer (clobber).
	if err := d.tick(ctx); err != nil {
		t.Fatal(err)
	}
	toks, _ := ts.RecentTokens(ctx, 10)
	if len(toks) != 1 {
		t.Fatalf("token sayısı=%d want 1", len(toks))
	}
	got := toks[0].Spark
	if len(got) != len(simulatedSpark) {
		t.Fatalf("spark ikinci tick'te clobber edildi: len=%d want %d (got=%v)", len(got), len(simulatedSpark), got)
	}
	for i, v := range simulatedSpark {
		if got[i] != v {
			t.Fatalf("spark ikinci tick'te değişti: got=%v want=%v", got, simulatedSpark)
		}
	}
}
