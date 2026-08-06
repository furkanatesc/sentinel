package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GeckoTerminalClient, GeckoTerminal v2 REST'i MarketProvider'a uyarlar (keysiz).
type GeckoTerminalClient struct {
	baseURL string
	http    *http.Client
}

// NewGeckoTerminalClient, base URL (ör. https://api.geckoterminal.com/api/v2) ve http.Client alır.
func NewGeckoTerminalClient(baseURL string, hc *http.Client) *GeckoTerminalClient {
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &GeckoTerminalClient{baseURL: strings.TrimRight(baseURL, "/"), http: hc}
}

// gtResponse, GeckoTerminal JSON:API yanıt zarfıdır.
type gtResponse struct {
	Data     []gtPool  `json:"data"`
	Included []gtToken `json:"included"`
}

type gtPool struct {
	Attributes struct {
		Address     string            `json:"address"`
		Name        string            `json:"name"`
		PriceUSD    string            `json:"base_token_price_usd"`
		ReserveUSD  string            `json:"reserve_in_usd"`
		CreatedAt   string            `json:"pool_created_at"`
		VolumeUSD   map[string]string `json:"volume_usd"`
		PriceChange map[string]string `json:"price_change_percentage"`
	} `json:"attributes"`
	Relationships struct {
		BaseToken struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"base_token"`
		Dex struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"dex"`
	} `json:"relationships"`
}

type gtToken struct {
	ID         string `json:"id"`
	Attributes struct {
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		Address string `json:"address"`
	} `json:"attributes"`
}

func (c *GeckoTerminalClient) NewPools(ctx context.Context) ([]Pool, error) {
	url := c.baseURL + "/networks/solana/new_pools?include=base_token"
	var resp gtResponse
	if err := c.getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	return resp.toPools(true), nil
}

func (c *GeckoTerminalClient) PoolsByAddresses(ctx context.Context, poolAddrs []string) ([]Pool, error) {
	if len(poolAddrs) == 0 {
		return nil, nil
	}
	url := c.baseURL + "/networks/solana/pools/multi/" + strings.Join(poolAddrs, ",")
	var resp gtResponse
	if err := c.getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	return resp.toPools(false), nil
}

func (c *GeckoTerminalClient) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("geckoterminal %s: status %d", url, res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// toPools, JSON:API zarfını []Pool'a çevirir; filterDex=true ise desteklenmeyen dex elenir.
func (r *gtResponse) toPools(filterDex bool) []Pool {
	names := map[string]gtToken{}
	for _, t := range r.Included {
		names[t.ID] = t
	}
	out := make([]Pool, 0, len(r.Data))
	for _, d := range r.Data {
		dexID := d.Relationships.Dex.Data.ID
		if filterDex {
			if _, ok := DexToLaunchpad(dexID); !ok {
				continue
			}
		}
		baseID := d.Relationships.BaseToken.Data.ID
		p := Pool{
			PoolAddr:      d.Attributes.Address,
			Mint:          stripNetwork(baseID),
			Dex:           dexID,
			Price:         parseFloat(d.Attributes.PriceUSD),
			LiquidityUSD:  parseFloat(d.Attributes.ReserveUSD),
			Vol5m:         parseFloat(d.Attributes.VolumeUSD["m5"]),
			PriceChangeH1: parseFloat(d.Attributes.PriceChange["h1"]),
			CreatedAtUnix: parseTime(d.Attributes.CreatedAt),
		}
		if tok, ok := names[baseID]; ok {
			p.Name, p.Symbol = tok.Attributes.Name, tok.Attributes.Symbol
		}
		if p.Symbol == "" { // include yoksa (enrichment yolu) havuz adından türet
			p.Symbol = baseSymbolFromName(d.Attributes.Name)
			p.Name = p.Symbol
		}
		out = append(out, p)
	}
	return out
}

// stripNetwork, "solana_<addr>" → "<addr>".
func stripNetwork(id string) string {
	if i := strings.IndexByte(id, '_'); i >= 0 {
		return id[i+1:]
	}
	return id
}

// baseSymbolFromName, "TROLL / SOL" → "TROLL".
func baseSymbolFromName(name string) string {
	if i := strings.IndexByte(name, '/'); i >= 0 {
		return strings.TrimSpace(name[:i])
	}
	return strings.TrimSpace(name)
}

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseTime(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

var _ MarketProvider = (*GeckoTerminalClient)(nil)
