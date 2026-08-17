package market

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	newPools, err := os.ReadFile("testdata/new_pools.json")
	if err != nil {
		t.Fatal(err)
	}
	multi, err := os.ReadFile("testdata/pools_multi.json")
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/new_pools"):
			w.Write(newPools)
		case strings.Contains(r.URL.Path, "/pools/multi/"):
			w.Write(multi)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestNewPoolsParsesAndFiltersDex(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewGeckoTerminalClient(srv.URL, srv.Client())
	pools, err := c.NewPools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 1 { // orca elenmeli
		t.Fatalf("pools=%d, want 1 (orca filtrelenmeli)", len(pools))
	}
	p := pools[0]
	if p.PoolAddr != "ABCpool" || p.Mint != "TROLLmint" || p.Symbol != "TROLL" || p.Name != "Troll Face" {
		t.Fatalf("kimlik yanlış: %+v", p)
	}
	if p.Dex != "pumpfun" || p.Price != 0.00004212 || p.LiquidityUSD != 82400.50 || p.Vol5m != 41200.0 || p.PriceChangeH1 != 22.4 {
		t.Fatalf("piyasa alanları yanlış: %+v", p)
	}
	if p.CreatedAtUnix == 0 {
		t.Fatal("CreatedAtUnix parse edilmedi")
	}
}

func TestPoolsByAddressesParses(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewGeckoTerminalClient(srv.URL, srv.Client())
	pools, err := c.PoolsByAddresses(context.Background(), []string{"ABCpool"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 1 || pools[0].PoolAddr != "ABCpool" || pools[0].Price != 0.00005 || pools[0].PriceChangeH1 != 10.0 {
		t.Fatalf("enrichment parse yanlış: %+v", pools)
	}
}

func TestPoolsByAddressesEmptyNoCall(t *testing.T) {
	c := NewGeckoTerminalClient("http://invalid.invalid", http.DefaultClient)
	pools, err := c.PoolsByAddresses(context.Background(), nil)
	if err != nil || len(pools) != 0 {
		t.Fatalf("boş adres listesi ağ çağrısı yapmadan boş dönmeli: pools=%v err=%v", pools, err)
	}
}

func TestNewPoolsHeaderFields(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewGeckoTerminalClient(srv.URL, srv.Client())
	pools, err := c.NewPools(context.Background())
	if err != nil || len(pools) == 0 {
		t.Fatalf("pools: %v err=%v", len(pools), err)
	}
	p := pools[0]
	if p.PriceChangeH24 != -5.0 || p.Vol24h != 900000.0 {
		t.Fatalf("h24 alanları yanlış: %+v", p)
	}
	if p.MarketCapUSD != 98000.0 { // market_cap_usd öncelikli
		t.Fatalf("marketCap yanlış: %v (want 98000, fdv fallback DEĞİL)", p.MarketCapUSD)
	}
}

func TestToPoolsParsesTransactionsH24(t *testing.T) {
	body := `{"data":[{"attributes":{
		"address":"pool1","name":"AAA / SOL","base_token_price_usd":"1","reserve_in_usd":"1000",
		"transactions":{"h24":{"buys":80,"sells":20,"buyers":30,"sellers":10}}},
		"relationships":{"base_token":{"data":{"id":"solana_mintA"}},"dex":{"data":{"id":"pumpfun"}}}}]}`
	var resp gtResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pools := resp.toPools(false)
	if len(pools) != 1 {
		t.Fatalf("1 havuz bekleniyordu, gelen %d", len(pools))
	}
	p := pools[0]
	if p.TxnsBuys != 80 || p.TxnsSells != 20 || p.TxnsBuyers != 30 || p.TxnsSellers != 10 {
		t.Fatalf("h24 txns yanlış: %+v", p)
	}
}

func TestToPoolsMissingTransactionsZero(t *testing.T) {
	body := `{"data":[{"attributes":{"address":"p","name":"B / SOL","base_token_price_usd":"1"},
		"relationships":{"base_token":{"data":{"id":"solana_m"}},"dex":{"data":{"id":"pumpfun"}}}}]}`
	var resp gtResponse
	json.Unmarshal([]byte(body), &resp)
	pools := resp.toPools(false)
	if len(pools) != 1 || pools[0].TxnsBuys != 0 || pools[0].TxnsBuyers != 0 {
		t.Fatalf("eksik transactions → 0 beklenir, gelen %+v", pools)
	}
}

func TestOHLCVParsesAscending(t *testing.T) {
	newPools, _ := os.ReadFile("testdata/new_pools.json")
	ohlcv, _ := os.ReadFile("testdata/ohlcv.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/ohlcv/") {
			w.Write(ohlcv)
			return
		}
		w.Write(newPools)
	}))
	defer srv.Close()
	c := NewGeckoTerminalClient(srv.URL, srv.Client())
	candles, err := c.OHLCV(context.Background(), "ABCpool", "minute", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 3 {
		t.Fatalf("candles=%d want 3", len(candles))
	}
	// t artan sıralı olmalı (fixture newest-first → reverse)
	if !(candles[0].Ts < candles[1].Ts && candles[1].Ts < candles[2].Ts) {
		t.Fatalf("t artan sıralı değil: %+v", candles)
	}
	if candles[2].Close != 0.000051 || candles[2].Volume != 12000.0 {
		t.Fatalf("son mum yanlış (close/volume): %+v", candles[2])
	}
}
