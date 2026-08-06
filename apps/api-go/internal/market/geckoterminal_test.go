package market

import (
	"context"
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
