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
