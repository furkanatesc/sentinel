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

// Limiter, dışa giden çağrıları geciktiren paylaşılan hız sınırlayıcıdır (ISP: tek
// metot). *golang.org/x/time/rate.Limiter bu arayüzü doğrudan karşılar.
type Limiter interface {
	Wait(ctx context.Context) error
}

// GeckoTerminalClient, GeckoTerminal v2 REST'i MarketProvider'a uyarlar (keysiz).
type GeckoTerminalClient struct {
	baseURL string
	http    *http.Client
	limiter Limiter                                          // nil → sınırlama yok
	sleep   func(ctx context.Context, d time.Duration) error // 429 backoff (test için enjekte edilebilir)
}

// Option, GeckoTerminalClient'ı yapılandırır (OCP: constructor'ı değiştirmeden genişlet).
type Option func(*GeckoTerminalClient)

// WithLimiter, paylaşılan hız sınırlayıcı enjekte eder. Aynı örnek birden çok
// client'a verilerek tek istek bütçesi (token-bucket) paylaşılır.
func WithLimiter(l Limiter) Option {
	return func(c *GeckoTerminalClient) { c.limiter = l }
}

// NewGeckoTerminalClient, base URL (ör. https://api.geckoterminal.com/api/v2) ve http.Client alır.
func NewGeckoTerminalClient(baseURL string, hc *http.Client, opts ...Option) *GeckoTerminalClient {
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	c := &GeckoTerminalClient{baseURL: strings.TrimRight(baseURL, "/"), http: hc, sleep: sleepCtx}
	for _, o := range opts {
		o(c)
	}
	return c
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
		MarketCap   string            `json:"market_cap_usd"`
		FDV         string            `json:"fdv_usd"`
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

// gtMaxAttempts, 429 (rate-limit) için toplam deneme sayısıdır (1 asıl + retry'lar).
const gtMaxAttempts = 3

// getJSON, her istekten önce paylaşılan limiter'dan token alır ve 429'da sınırlı
// sayıda backoff'lu retry yapar. 429 dışı statüler anında hata döner (retry yok).
func (c *GeckoTerminalClient) getJSON(ctx context.Context, url string, out any) error {
	var lastErr error
	for attempt := 0; attempt < gtMaxAttempts; attempt++ {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return err
			}
		}
		status, err := c.doOnce(ctx, url, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if status != http.StatusTooManyRequests || attempt == gtMaxAttempts-1 {
			return err
		}
		if serr := c.sleep(ctx, backoff(attempt)); serr != nil {
			return serr
		}
	}
	return lastErr
}

// doOnce, tek HTTP GET yapıp gövdeyi çözer. Retry kararı için statü kodunu döner.
func (c *GeckoTerminalClient) doOnce(ctx context.Context, url string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return res.StatusCode, fmt.Errorf("geckoterminal %s: status %d", url, res.StatusCode)
	}
	return res.StatusCode, json.NewDecoder(res.Body).Decode(out)
}

// backoff, deneme sırasına göre artan gecikme verir (0→500ms, 1→1s).
func backoff(attempt int) time.Duration {
	return time.Duration(500*(1<<attempt)) * time.Millisecond
}

// sleepCtx, ctx iptaline duyarlı uyku (varsayılan backoff uygulayıcı).
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
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

			PriceChangeH24: parseFloat(d.Attributes.PriceChange["h24"]),
			Vol24h:         parseFloat(d.Attributes.VolumeUSD["h24"]),
			MarketCapUSD:   marketCap(d.Attributes.MarketCap, d.Attributes.FDV),
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

// marketCap, market_cap_usd önceliklidir; boş/0 ise fdv_usd'ye düşer.
func marketCap(mc, fdv string) float64 {
	if v := parseFloat(mc); v > 0 {
		return v
	}
	return parseFloat(fdv)
}

type gtOHLCV struct {
	Data struct {
		Attributes struct {
			OHLCVList [][]float64 `json:"ohlcv_list"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *GeckoTerminalClient) OHLCV(ctx context.Context, poolAddr, timeframe string, limit int) ([]Candle, error) {
	url := fmt.Sprintf("%s/networks/solana/pools/%s/ohlcv/%s?aggregate=1&limit=%d", c.baseURL, poolAddr, timeframe, limit)
	var resp gtOHLCV
	if err := c.getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	list := resp.Data.Attributes.OHLCVList
	out := make([]Candle, 0, len(list))
	// ohlcv_list newest-first; artan t için ters çevir. Her satır [ts,o,h,l,c,v].
	for i := len(list) - 1; i >= 0; i-- {
		row := list[i]
		if len(row) < 6 {
			continue
		}
		out = append(out, Candle{Ts: int64(row[0]), Close: row[4], Volume: row[5]})
	}
	return out, nil
}

var _ MarketProvider = (*GeckoTerminalClient)(nil)
