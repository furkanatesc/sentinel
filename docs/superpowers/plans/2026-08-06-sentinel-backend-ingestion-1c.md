# Slice 1c — Token Detail (`getToken`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `getToken(mint)`'i gerçeğe çevirmek — Token Detay ekranını GeckoTerminal (pool header + yaşa-uyarlı OHLCV) + Helius (holders) ile canlı gerçek veriyle doldurmak; skorlar/riskler/davranış-metrikleri dürüst nötr placeholder (Alt-proje 2).

**Architecture:** `MarketProvider` `OHLCV` ile genişler + `Pool` header alanlarıyla; yeni `HoldersProvider` (DIP, Helius impl `internal/ingest`); `TokenDetailService` (`internal/market`, SRP) store + market + holders'ı birleştirip `store.TokenDetail` kurar (nötr placeholder + ~20s cache); `GET /api/token/{mint}` handler; frontend `getToken` gerçek + `LIVE_ENDPOINTS`.

**Tech Stack:** Go 1.24, `database/sql`+pgx stdlib, chi router, stdlib `net/http`+`encoding/json` (GeckoTerminal keysiz REST + Helius JSON-RPC — yeni dependency YOK). Frontend Next.js/TS.

## Global Constraints

- **Go sürümü:** `go 1.24` (değiştirme).
- **Yeni dependency yok:** GeckoTerminal keysiz; Helius mevcut (1a). Yalnız stdlib + mevcut modüller.
- **Seam kontratı birebir:** Go `store.TokenDetail` JSON = TS `TokenDetail` (`apps/web/lib/api/types.ts`). `scores` bir Record (4 ScoreKey: `opportunity`/`creatorReputation`/`tokenSafety`/`manipulationRisk`) → Go `map[string]ScoreDetail`, hepsi mevcut olmalı.
- **Dürüstlük:** Yalnız gerçek bilinen alanlar doldurulur. `scores`/`risks`/davranış-metrikleri + `series.liquidity`/`series.holders` nötr (Alt-proje 2). Sahte veri yok. `metrics.holders` capped ise cap değeri **floor** olarak döner (seam `number`; "N+" gösterimi yok — aşırı-iddia etme).
- **Bilinmeyen/pool'suz mint → 404** (dürüst; liste dışı mint UI'da ulaşılmaz).
- **Frontend değişikliği yalnız:** `http.ts` getToken + `live-endpoints.ts`. Token Detail UI/component'lerine dokunma (seam sabit).
- **DB/canlı testleri** `DATABASE_URL`/key yoksa `t.Skip` veya fake ile — yerel key/DB yok (1a/1b deseni). Canlı GeckoTerminal OHLCV + Helius holders yalnız deploy'da doğrulanır.
- **Clean/SOLID:** tüketici-tanımlı dar arayüzler, enjekte edilebilir bağımlılıklar (clock, http.Client, provider'lar).

---

## File Structure

**Create:**
- `apps/api-go/internal/store/token_detail.go` — `TokenDetail` + alt-struct'lar (seam) + `TokenDetailBase` tipi.
- `apps/api-go/internal/store/token_detail_test.go` — fake `TokenDetailBase` + JSON tag doğrulama.
- `apps/api-go/internal/market/detail.go` — `TokenDetailService` (+`DetailStore`/`HoldersProvider` tüketici arayüzleri).
- `apps/api-go/internal/market/detail_test.go`.
- `apps/api-go/internal/market/testdata/ohlcv.json` — OHLCV fixture.
- `apps/api-go/internal/ingest/holders.go` — Helius `getTokenAccounts` holders sayacı.
- `apps/api-go/internal/ingest/holders_test.go`.
- `apps/api-go/internal/api/token.go` — `tokenHandler`.
- `apps/api-go/internal/api/token_test.go`.

**Modify:**
- `apps/api-go/internal/store/tokens.go` — `TokenStore` arayüzüne `TokenDetailBase` + postgres impl.
- `apps/api-go/internal/store/fake_ingest.go` — `fakeTokenStore.TokenDetailBase`.
- `apps/api-go/internal/ingest/worker_test.go` — 2 stub'a `TokenDetailBase` no-op.
- `apps/api-go/internal/market/provider.go` — `Pool` +3 alan, `Candle`, `MarketProvider`+`OHLCV`.
- `apps/api-go/internal/market/geckoterminal.go` — Pool header parse + `OHLCV` metodu.
- `apps/api-go/internal/market/geckoterminal_test.go` — header + OHLCV testleri.
- `apps/api-go/internal/api/router.go` — `/api/token/{mint}` route + `RouterDeps.TokenDetail`.
- `apps/api-go/internal/config/config.go` — detail config alanları.
- `apps/api-go/cmd/server/main.go` — HeliusHolders + TokenDetailService + RouterDeps wiring.
- `apps/web/lib/api/http.ts` — `getToken` gerçek fetch.
- `apps/web/lib/api/live-endpoints.ts` — `LIVE_ENDPOINTS` += `getToken`.
- `apps/web/lib/api/http.test.ts` — getToken testi.
- `docs/progress.md`, `docs/superpowers/followups-frontend.md` — yaşayan kayıt (Task 8).

---

### Task 1: Store — TokenDetail seam struct'ları + TokenDetailBase

**Files:**
- Create: `apps/api-go/internal/store/token_detail.go`, `apps/api-go/internal/store/token_detail_test.go`
- Modify: `apps/api-go/internal/store/tokens.go` (arayüz + postgres), `apps/api-go/internal/store/fake_ingest.go`, `apps/api-go/internal/ingest/worker_test.go`

**Interfaces:**
- Produces:
  - `store.TokenDetail` ve alt-struct'lar (JSON tag'leri TS `TokenDetail` ile birebir).
  - `store.TokenDetailBase{Name, Symbol, PoolAddr string; FirstSeenTs int64}`
  - `TokenStore.TokenDetailBase(ctx, mint string) (TokenDetailBase, bool, error)` (bulunamazsa `ok=false`).

- [ ] **Step 1: Seam struct'larını yaz** (`token_detail.go`)

```go
package store

// Bu struct'lar frontend TokenDetail (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.

type ScoreBreakdownItem struct {
	Label  string  `json:"label"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

type ScoreDetail struct {
	Key       string               `json:"key"`
	Value     float64              `json:"value"`
	Confidence float64             `json:"confidence"`
	UpdatedAt string               `json:"updatedAt"`
	Breakdown []ScoreBreakdownItem `json:"breakdown"`
}

type TokenMetrics struct {
	Holders           int     `json:"holders"`
	UniqueBuyers      int     `json:"uniqueBuyers"`
	BuyRatio          float64 `json:"buyRatio"`
	SellRatio         float64 `json:"sellRatio"`
	CreatorHoldingPct float64 `json:"creatorHoldingPct"`
	Top10HolderPct    float64 `json:"top10HolderPct"`
	SniperPct         float64 `json:"sniperPct"`
	BotActivityPct    float64 `json:"botActivityPct"`
}

type SeriesPoint struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

type TokenDetailSeries struct {
	Price     []SeriesPoint `json:"price"`
	Liquidity []SeriesPoint `json:"liquidity"`
	Volume    []SeriesPoint `json:"volume"`
	Holders   []SeriesPoint `json:"holders"`
}

type RiskItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Evidence    string `json:"evidence,omitempty"`
	FirstSeen   string `json:"firstSeen"`
	LastSeen    string `json:"lastSeen"`
}

type RiskGroups struct {
	Contract []RiskItem `json:"contract"`
	Market   []RiskItem `json:"market"`
	Creator  []RiskItem `json:"creator"`
}

type TokenDetail struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Symbol         string                 `json:"symbol"`
	Mint           string                 `json:"mint"`
	AgeSeconds     int64                  `json:"ageSeconds"`
	Price          float64                `json:"price"`
	PriceChange24h float64                `json:"priceChange24h"`
	MarketCap      float64                `json:"marketCap"`
	Liquidity      float64                `json:"liquidity"`
	Volume24h      float64                `json:"volume24h"`
	Scores         map[string]ScoreDetail `json:"scores"`
	Metrics        TokenMetrics           `json:"metrics"`
	Series         TokenDetailSeries      `json:"series"`
	Risks          RiskGroups             `json:"risks"`
}

// TokenDetailBase, getToken için gereken kimlik + havuz bilgisidir (mint kaydından).
type TokenDetailBase struct {
	Name, Symbol, PoolAddr string
	FirstSeenTs            int64
}
```

- [ ] **Step 2: `TokenStore` arayüzüne `TokenDetailBase` ekle + postgres impl** (`tokens.go`)

Arayüze ekle (diğer metotların yanına):
```go
	// 1c: getToken için tek token kimlik+havuz (bulunamazsa ok=false).
	TokenDetailBase(ctx context.Context, mint string) (TokenDetailBase, bool, error)
```
Postgres impl (dosyaya ekle):
```go
func (p *postgresStore) TokenDetailBase(ctx context.Context, mint string) (TokenDetailBase, bool, error) {
	const q = `SELECT name, symbol, pool_address, first_seen_ts FROM tokens WHERE mint=$1`
	var b TokenDetailBase
	err := p.db.QueryRowContext(ctx, q, mint).Scan(&b.Name, &b.Symbol, &b.PoolAddr, &b.FirstSeenTs)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenDetailBase{}, false, nil
	}
	if err != nil {
		return TokenDetailBase{}, false, err
	}
	return b, true, nil
}
```
Dosya başına import gerekiyorsa `"database/sql"` ve `"errors"` ekle (yoksa).

- [ ] **Step 3: `fakeTokenStore.TokenDetailBase` ekle** (`fake_ingest.go`)

```go
func (f *fakeTokenStore) TokenDetailBase(_ context.Context, mint string) (TokenDetailBase, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[mint]
	if !ok {
		return TokenDetailBase{}, false, nil
	}
	return TokenDetailBase{Name: t.row.Name, Symbol: t.row.Symbol, PoolAddr: t.poolAddr, FirstSeenTs: t.firstSeen}, true, nil
}
```

- [ ] **Step 4: worker_test.go 2 stub'a no-op ekle** (`internal/ingest/worker_test.go`)

`failingTokenStore` ve `snapshotFailingTokenStore` her ikisine:
```go
func (f *failingTokenStore) TokenDetailBase(context.Context, string) (store.TokenDetailBase, bool, error) {
	return store.TokenDetailBase{}, false, nil
}
```
(snapshotFailingTokenStore için aynı gövde.)

- [ ] **Step 5: Testleri yaz** (`token_detail_test.go`)

```go
package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestFakeTokenDetailBase(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "M1", Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 42})
	b, ok, err := f.TokenDetailBase(ctx, "M1")
	if err != nil || !ok {
		t.Fatalf("bulunmalı: ok=%v err=%v", ok, err)
	}
	if b.Name != "One" || b.Symbol != "ONE" || b.PoolAddr != "P1" || b.FirstSeenTs != 42 {
		t.Fatalf("base yanlış: %+v", b)
	}
	if _, ok, _ := f.TokenDetailBase(ctx, "YOK"); ok {
		t.Fatal("bilinmeyen mint ok=false olmalı")
	}
}

func TestTokenDetailJSONTags(t *testing.T) {
	// Seam: camelCase alan adları frontend TokenDetail ile eşleşmeli.
	d := TokenDetail{Scores: map[string]ScoreDetail{}, Series: TokenDetailSeries{
		Price: []SeriesPoint{}, Liquidity: []SeriesPoint{}, Volume: []SeriesPoint{}, Holders: []SeriesPoint{}}}
	b, _ := json.Marshal(d)
	for _, key := range []string{`"priceChange24h"`, `"marketCap"`, `"volume24h"`, `"scores"`, `"metrics"`, `"series"`, `"risks"`, `"ageSeconds"`} {
		if !strings.Contains(string(b), key) {
			t.Fatalf("JSON'da %s yok: %s", key, b)
		}
	}
}
```

- [ ] **Step 6: Çalıştır + commit**

Run: `cd apps/api-go && go test ./internal/store/... ./internal/ingest/...`
Expected: PASS.
```bash
git add apps/api-go/internal/store apps/api-go/internal/ingest/worker_test.go
git commit -m "feat(detail): store TokenDetail seam struct'ları + TokenDetailBase (1c)"
```

---

### Task 2: market — Pool header alanları + OHLCV

**Files:**
- Modify: `apps/api-go/internal/market/provider.go`, `apps/api-go/internal/market/geckoterminal.go`
- Create: `apps/api-go/internal/market/testdata/ohlcv.json`
- Modify: `apps/api-go/internal/market/geckoterminal_test.go`, `apps/api-go/internal/market/testdata/new_pools.json` (header alanları ekle)

**Interfaces:**
- Consumes: mevcut `Pool`, `GeckoTerminalClient`, `parseFloat` (Task 1b).
- Produces:
  - `Pool` += `PriceChangeH24 float64`, `MarketCapUSD float64`, `Vol24h float64`
  - `market.Candle{Ts int64; Close, Volume float64}`
  - `MarketProvider` += `OHLCV(ctx, poolAddr, timeframe string, limit int) ([]Candle, error)` (t artan sıralı)

- [ ] **Step 1: provider.go — Pool genişlet + Candle + arayüz**

`Pool` struct'ına 3 alan ekle:
```go
	PriceChangeH24 float64 // h24 yüzde
	MarketCapUSD   float64 // market_cap_usd, yoksa fdv_usd
	Vol24h         float64
```
Ekle:
```go
// Candle, OHLCV mumunun grafik için gereken kısmıdır (close + volume).
type Candle struct {
	Ts     int64
	Close  float64
	Volume float64
}
```
`MarketProvider` arayüzüne ekle:
```go
	OHLCV(ctx context.Context, poolAddr, timeframe string, limit int) ([]Candle, error)
```

- [ ] **Step 2: OHLCV fixture** (`testdata/ohlcv.json`)

GeckoTerminal OHLCV yanıtı; `ohlcv_list` = `[[ts,open,high,low,close,volume],...]` (newest-first).
```json
{
  "data": {
    "id": "ABCpool",
    "type": "ohlcv_request_response",
    "attributes": {
      "ohlcv_list": [
        [1754476800, 0.000050, 0.000052, 0.000049, 0.000051, 12000.0],
        [1754476740, 0.000048, 0.000051, 0.000047, 0.000050, 9000.0],
        [1754476680, 0.000045, 0.000049, 0.000044, 0.000048, 7000.0]
      ]
    }
  }
}
```

- [ ] **Step 3: new_pools.json fixture'a header alanları ekle**

`testdata/new_pools.json`'daki ilk pool'un `attributes`'ına ekle (TROLL pool):
```json
        "fdv_usd": "125000.0",
        "market_cap_usd": "98000.0",
```
(`volume_usd`/`price_change_percentage` zaten `h1` içeriyor; `h24` ekle: `volume_usd`'ye `"h24": "900000.0"` zaten var; `price_change_percentage`'e `"h24": "-5.0"` zaten var — yoksa ekle.)

- [ ] **Step 4: Testleri yaz** (`geckoterminal_test.go`'ya ekle)

```go
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
```
(`os`, `net/http`, `net/http/httptest`, `strings` import'ları test dosyasında zaten var — yoksa ekle.)

- [ ] **Step 5: geckoterminal.go — Pool header parse + OHLCV**

`gtPool.Attributes`'a alan ekle:
```go
		MarketCap  string            `json:"market_cap_usd"`
		FDV        string            `json:"fdv_usd"`
```
`toPools` içinde `Pool` kurarken ekle:
```go
			PriceChangeH24: parseFloat(d.Attributes.PriceChange["h24"]),
			Vol24h:         parseFloat(d.Attributes.VolumeUSD["h24"]),
			MarketCapUSD:   marketCap(d.Attributes.MarketCap, d.Attributes.FDV),
```
Yardımcı + OHLCV metodu (dosyaya ekle):
```go
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
```

- [ ] **Step 6: Çalıştır + commit**

Run: `cd apps/api-go && go test ./internal/market/... -v`
Expected: PASS (header + OHLCV + mevcut testler).
```bash
git add apps/api-go/internal/market
git commit -m "feat(detail): market — Pool header alanları (h24/marketCap/vol24h) + OHLCV (1c)"
```

---

### Task 3: ingest — Helius holders sayacı

**Files:**
- Create: `apps/api-go/internal/ingest/holders.go`, `apps/api-go/internal/ingest/holders_test.go`

**Interfaces:**
- Produces: `NewHeliusHolders(rpcURL string) *HeliusHolders`; `(*HeliusHolders).HoldersCount(ctx, mint string, cap int) (count int, capped bool, err error)`. Enjekte edilebilir base URL + http.Client (test).

- [ ] **Step 1: Testi yaz** (`holders_test.go`)

Fake Helius: sayfa başına `limit` hesap döndürür; `getTokenAccounts` page-based.
```go
package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeHeliusHolders, toplam `total` hesabı `limit`'lik sayfalar halinde döndürür.
func fakeHeliusServer(t *testing.T, total, limit int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params struct {
				Page int `json:"page"`
			} `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		start := (req.Params.Page - 1) * limit
		n := 0
		if start < total {
			n = total - start
			if n > limit {
				n = limit
			}
		}
		accs := make([]map[string]any, n)
		for i := range accs {
			accs[i] = map[string]any{"address": fmt.Sprintf("acc%d", start+i)}
		}
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "1",
			"result": map[string]any{"total": n, "limit": limit, "page": req.Params.Page, "token_accounts": accs}})
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
}

func TestHoldersCountSinglePage(t *testing.T) {
	srv := fakeHeliusServer(t, 42, 1000)
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	n, capped, err := h.HoldersCount(context.Background(), "MintX", 5000)
	if err != nil || n != 42 || capped {
		t.Fatalf("n=%d capped=%v err=%v (want 42/false)", n, capped, err)
	}
}

func TestHoldersCountMultiPage(t *testing.T) {
	srv := fakeHeliusServer(t, 2500, 1000)
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	n, capped, err := h.HoldersCount(context.Background(), "MintX", 5000)
	if err != nil || n != 2500 || capped {
		t.Fatalf("n=%d capped=%v err=%v (want 2500/false)", n, capped, err)
	}
}

func TestHoldersCountCapped(t *testing.T) {
	srv := fakeHeliusServer(t, 999999, 1000)
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	n, capped, err := h.HoldersCount(context.Background(), "MintX", 3000)
	if err != nil || !capped || n < 3000 {
		t.Fatalf("n=%d capped=%v err=%v (want >=3000/true)", n, capped, err)
	}
}
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd apps/api-go && go test ./internal/ingest/... -run TestHoldersCount`
Expected: FAIL (NewHeliusHolders yok)

- [ ] **Step 3: holders.go'yu yaz**

```go
package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HeliusHolders, bir mint'in holder (token account) sayısını Helius getTokenAccounts
// ile sayfalayarak sayar. cap'e ulaşınca durur (capped=true) — pahalı büyük token'ları sınırlar.
type HeliusHolders struct {
	rpcURL string
	http   *http.Client
}

func NewHeliusHolders(rpcURL string) *HeliusHolders {
	return &HeliusHolders{rpcURL: rpcURL, http: &http.Client{Timeout: 12 * time.Second}}
}

const holdersPageLimit = 1000

func (h *HeliusHolders) HoldersCount(ctx context.Context, mint string, cap int) (int, bool, error) {
	if cap <= 0 {
		cap = 5000
	}
	total := 0
	for page := 1; ; page++ {
		n, err := h.pageCount(ctx, mint, page)
		if err != nil {
			return total, false, err
		}
		total += n
		if total >= cap {
			return total, true, nil // cap'e ulaşıldı → floor
		}
		if n < holdersPageLimit {
			return total, false, nil // son sayfa (kısa)
		}
	}
}

func (h *HeliusHolders) pageCount(ctx context.Context, mint string, page int) (int, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getTokenAccounts",
		"params": map[string]any{"mint": mint, "page": page, "limit": holdersPageLimit},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("helius getTokenAccounts: status %d", res.StatusCode)
	}
	var r struct {
		Result struct {
			TokenAccounts []json.RawMessage `json:"token_accounts"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return 0, err
	}
	if r.Error != nil {
		return 0, fmt.Errorf("helius getTokenAccounts error: %s", r.Error.Message)
	}
	return len(r.Result.TokenAccounts), nil
}
```
> **NOT (deploy kalibrasyonu):** Helius `getTokenAccounts` yanıt şekli (`result.token_accounts` + page-based) dokümante şemadır; canlıda alan/sayfalama farklıysa parse+fixture birlikte güncellenir (1a/1b deseni).

- [ ] **Step 4: Çalıştır + commit**

Run: `cd apps/api-go && go test ./internal/ingest/... -run TestHoldersCount -v`
Expected: PASS.
```bash
git add apps/api-go/internal/ingest/holders.go apps/api-go/internal/ingest/holders_test.go
git commit -m "feat(detail): Helius holders sayacı — getTokenAccounts sayfalama + cap (1c)"
```

---

### Task 4: market — TokenDetailService

**Files:**
- Create: `apps/api-go/internal/market/detail.go`, `apps/api-go/internal/market/detail_test.go`

**Interfaces:**
- Consumes: `MarketProvider` (Task 2), `store.TokenDetail`/`store.TokenDetailBase` (Task 1); tüketici arayüzleri `DetailStore` + `HoldersProvider` (aşağıda).
- Produces: `NewTokenDetailService(TokenDetailDeps) *TokenDetailService`; `(*TokenDetailService).Build(ctx, mint string) (store.TokenDetail, bool, error)` (mint yoksa ok=false).

- [ ] **Step 1: Testi yaz** (`detail_test.go`)

```go
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
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd apps/api-go && go test ./internal/market/... -run TestBuild`
Expected: FAIL (TokenDetailService yok)

- [ ] **Step 3: detail.go'yu yaz**

```go
package market

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// DetailStore, getToken için tek token kimlik+havuz kaynağıdır (tüketici arayüzü; store.TokenStore karşılar).
type DetailStore interface {
	TokenDetailBase(ctx context.Context, mint string) (store.TokenDetailBase, bool, error)
}

// HoldersProvider, bir mint'in holder sayısını verir (tüketici arayüzü; ingest.HeliusHolders karşılar).
type HoldersProvider interface {
	HoldersCount(ctx context.Context, mint string, cap int) (count int, capped bool, err error)
}

type TokenDetailDeps struct {
	Store       DetailStore
	Provider    MarketProvider
	Holders     HoldersProvider
	CacheTTL    time.Duration // 0 → 20s
	OHLCVLimit  int           // 0 → 200
	HoldersCap  int           // 0 → 5000
	MinuteMaxAge int64        // 0 → 21600 (6h); bundan genç → minute, değilse hour
	Now         func() int64
	Logger      *slog.Logger
}

type cacheEntry struct {
	detail store.TokenDetail
	at     int64 // unix
}

// TokenDetailService, tek token'ın detayını market+holders+store'dan birleştirir (SRP).
type TokenDetailService struct {
	d     TokenDetailDeps
	mu    sync.Mutex
	cache map[string]cacheEntry
}

func NewTokenDetailService(d TokenDetailDeps) *TokenDetailService {
	if d.CacheTTL <= 0 {
		d.CacheTTL = 20 * time.Second
	}
	if d.OHLCVLimit <= 0 {
		d.OHLCVLimit = 200
	}
	if d.HoldersCap <= 0 {
		d.HoldersCap = 5000
	}
	if d.MinuteMaxAge <= 0 {
		d.MinuteMaxAge = 21600
	}
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &TokenDetailService{d: d, cache: map[string]cacheEntry{}}
}

func (s *TokenDetailService) Build(ctx context.Context, mint string) (store.TokenDetail, bool, error) {
	now := s.d.Now()
	s.mu.Lock()
	if e, ok := s.cache[mint]; ok && now-e.at < int64(s.d.CacheTTL/time.Second) {
		s.mu.Unlock()
		return e.detail, true, nil
	}
	s.mu.Unlock()

	base, ok, err := s.d.Store.TokenDetailBase(ctx, mint)
	if err != nil {
		return store.TokenDetail{}, false, err
	}
	if !ok || base.PoolAddr == "" {
		return store.TokenDetail{}, false, nil // bilinmeyen/pool'suz → 404
	}

	age := now - base.FirstSeenTs
	if age < 0 {
		age = 0
	}

	d := store.TokenDetail{
		ID: mint, Mint: mint, Name: base.Name, Symbol: base.Symbol, AgeSeconds: age,
		Scores: neutralScores(), Metrics: store.TokenMetrics{},
		Series: store.TokenDetailSeries{
			Price: []store.SeriesPoint{}, Liquidity: []store.SeriesPoint{},
			Volume: []store.SeriesPoint{}, Holders: []store.SeriesPoint{}},
		Risks: store.RiskGroups{Contract: []store.RiskItem{}, Market: []store.RiskItem{}, Creator: []store.RiskItem{}},
	}

	// Header (mevcut PoolsByAddresses yeniden kullanılır).
	if pools, err := s.d.Provider.PoolsByAddresses(ctx, []string{base.PoolAddr}); err != nil {
		s.d.Logger.Warn("detail header", "mint", mint, "err", err)
	} else {
		for _, p := range pools {
			if p.PoolAddr == base.PoolAddr {
				d.Price, d.Liquidity = p.Price, p.LiquidityUSD
				d.PriceChange24h, d.MarketCap, d.Volume24h = p.PriceChangeH24, p.MarketCapUSD, p.Vol24h
			}
		}
	}

	// Grafik (yaşa-uyarlı OHLCV).
	tf := "hour"
	if age < s.d.MinuteMaxAge {
		tf = "minute"
	}
	if candles, err := s.d.Provider.OHLCV(ctx, base.PoolAddr, tf, s.d.OHLCVLimit); err != nil {
		s.d.Logger.Warn("detail ohlcv", "mint", mint, "err", err)
	} else {
		for _, c := range candles {
			d.Series.Price = append(d.Series.Price, store.SeriesPoint{T: c.Ts, V: c.Close})
			d.Series.Volume = append(d.Series.Volume, store.SeriesPoint{T: c.Ts, V: c.Volume})
		}
	}

	// Holders (Helius, sınırlı).
	if n, capped, err := s.d.Holders.HoldersCount(ctx, mint, s.d.HoldersCap); err != nil {
		s.d.Logger.Warn("detail holders", "mint", mint, "err", err)
	} else {
		d.Metrics.Holders = n // capped ise cap = floor (seam number; "+" gösterimi yok)
		_ = capped
	}

	s.mu.Lock()
	s.cache[mint] = cacheEntry{detail: d, at: now}
	s.mu.Unlock()
	return d, true, nil
}

// neutralScores, 4 ScoreKey için dürüst nötr placeholder üretir (Alt-proje 2 gelene kadar).
func neutralScores() map[string]store.ScoreDetail {
	keys := []string{"opportunity", "creatorReputation", "tokenSafety", "manipulationRisk"}
	m := make(map[string]store.ScoreDetail, len(keys))
	for _, k := range keys {
		m[k] = store.ScoreDetail{Key: k, Value: 0, Confidence: 0, UpdatedAt: "—", Breakdown: []store.ScoreBreakdownItem{}}
	}
	return m
}
```

- [ ] **Step 4: Çalıştır + commit**

Run: `cd apps/api-go && go test ./internal/market/... -race -v`
Expected: PASS.
```bash
git add apps/api-go/internal/market/detail.go apps/api-go/internal/market/detail_test.go
git commit -m "feat(detail): TokenDetailService — header+OHLCV+holders birleştirme, nötr placeholder, cache (1c)"
```

---

### Task 5: api — /api/token/{mint} handler

**Files:**
- Create: `apps/api-go/internal/api/token.go`, `apps/api-go/internal/api/token_test.go`
- Modify: `apps/api-go/internal/api/router.go`

**Interfaces:**
- Consumes: `store.TokenDetail` (Task 1).
- Produces: `api` tüketici arayüzü `TokenDetailProvider interface { Build(ctx, mint string) (store.TokenDetail, bool, error) }`; `RouterDeps.TokenDetail TokenDetailProvider`; route `GET /api/token/{mint}`.

- [ ] **Step 1: Testi yaz** (`token_test.go`)

```go
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeDetail struct {
	byMint map[string]store.TokenDetail
}

func (f *fakeDetail) Build(_ context.Context, mint string) (store.TokenDetail, bool, error) {
	d, ok := f.byMint[mint]
	return d, ok, nil
}

func TestTokenHandlerFound(t *testing.T) {
	fd := &fakeDetail{byMint: map[string]store.TokenDetail{
		"MintX": {ID: "MintX", Mint: "MintX", Symbol: "TST", Price: 1.5}}}
	r := NewRouter(RouterDeps{TokenDetail: fd})
	req := httptest.NewRequest(http.MethodGet, "/api/token/MintX", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), `"symbol":"TST"`) {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestTokenHandlerNotFound(t *testing.T) {
	fd := &fakeDetail{byMint: map[string]store.TokenDetail{}}
	r := NewRouter(RouterDeps{TokenDetail: fd})
	req := httptest.NewRequest(http.MethodGet, "/api/token/YOK", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d want 404", w.Code)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```
> Eğer `contains` başka bir test dosyasında zaten varsa (aynı `api` paketi), buradaki tanımı SİL (çift tanım derleme hatası). Önce `grep -rn "func contains" apps/api-go/internal/api` ile kontrol et.

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd apps/api-go && go test ./internal/api/... -run TestTokenHandler`
Expected: FAIL (TokenDetail alanı / route yok)

- [ ] **Step 3: token.go + router.go**

`token.go`:
```go
package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// TokenDetailProvider, tek token detayını kurar (DIP; market.TokenDetailService karşılar).
type TokenDetailProvider interface {
	Build(ctx context.Context, mint string) (store.TokenDetail, bool, error)
}

func tokenHandler(svc TokenDetailProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mint := chi.URLParam(r, "mint")
		d, ok, err := svc.Build(r.Context(), mint)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "token detail unavailable"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found"})
			return
		}
		writeJSON(w, http.StatusOK, d)
	}
}
```
`router.go`: `RouterDeps`'e ekle:
```go
	TokenDetail  TokenDetailProvider
```
`NewRouter` içine (Tokens route'undan sonra):
```go
	if d.TokenDetail != nil {
		r.Get("/api/token/{mint}", tokenHandler(d.TokenDetail))
	}
```

- [ ] **Step 4: Çalıştır + commit**

Run: `cd apps/api-go && go test ./internal/api/... -v`
Expected: PASS (200 + 404 + mevcut router testleri).
```bash
git add apps/api-go/internal/api
git commit -m "feat(detail): GET /api/token/{mint} handler + route (1c)"
```

---

### Task 6: config + main.go wiring

**Files:**
- Modify: `apps/api-go/internal/config/config.go`, `apps/api-go/internal/config/config_test.go`, `apps/api-go/cmd/server/main.go`

**Interfaces:**
- Consumes: `ingest.NewHeliusHolders` (Task 3), `market.NewTokenDetailService`/`market.TokenDetailDeps` (Task 4), `api.RouterDeps.TokenDetail` (Task 5).

- [ ] **Step 1: Config alanları** (`config.go`)

`Config` struct'ına:
```go
	TokenDetailCacheSec int
	OHLCVLimit          int
	HoldersCap          int
```
`Load()`'a:
```go
		TokenDetailCacheSec: getenvInt("TOKEN_DETAIL_CACHE_SEC", 20),
		OHLCVLimit:          getenvInt("OHLCV_LIMIT", 200),
		HoldersCap:          getenvInt("HOLDERS_CAP", 5000),
```

- [ ] **Step 2: Config testi** (`config_test.go`'ya ekle)

```go
func TestLoadDetailDefaults(t *testing.T) {
	t.Setenv("TOKEN_DETAIL_CACHE_SEC", "")
	c := Load()
	if c.TokenDetailCacheSec != 20 || c.OHLCVLimit != 200 || c.HoldersCap != 5000 {
		t.Fatalf("detail default'ları yanlış: %+v", c)
	}
}
```

- [ ] **Step 3: main.go wiring**

`import`'a zaten `market`, `ingest`, `config` var. Market blok'undan sonra (veya router kurulmadan önce), Helius key varsa detail service kur:
```go
	// token detail service (GeckoTerminal header+OHLCV + Helius holders) — Slice 1c
	var detailSvc api.TokenDetailProvider
	if cfg.MarketEnabled && bundle.Tokens != nil {
		gtForDetail := market.NewGeckoTerminalClient(cfg.GeckoBaseURL, nil)
		var holders market.HoldersProvider
		if rpcURL != "" {
			holders = ingest.NewHeliusHolders(rpcURL)
		} else {
			holders = noopHolders{} // key yoksa holders 0 (dürüst)
		}
		detailSvc = market.NewTokenDetailService(market.TokenDetailDeps{
			Store: bundle.Tokens, Provider: gtForDetail, Holders: holders,
			CacheTTL: time.Duration(cfg.TokenDetailCacheSec) * time.Second,
			OHLCVLimit: cfg.OHLCVLimit, HoldersCap: cfg.HoldersCap, Logger: logger,
		})
	}
```
`RouterDeps`'e `TokenDetail: detailSvc,` ekle. Dosya sonuna noop ekle:
```go
// noopHolders, Helius key yokken holders'ı 0 döndürür (dürüst — sayı yok).
type noopHolders struct{}

func (noopHolders) HoldersCount(context.Context, string, int) (int, bool, error) { return 0, false, nil }
```
`context` import'u main.go'da zaten var. `rpcURL` mevcut worker blok'unda tanımlı — kapsamda erişilebilir olduğundan emin ol (fonksiyon başında tanımlıysa OK; değilse Helius blok'unda `rpcURL` zaten hesaplanıyor).

- [ ] **Step 4: Build + vet + test**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`
Expected: PASS.
```bash
git add apps/api-go/internal/config apps/api-go/cmd/server/main.go
git commit -m "feat(detail): config + main wiring — TokenDetailService + Helius holders (1c)"
```

---

### Task 7: Frontend — getToken gerçek + LIVE_ENDPOINTS

**Files:**
- Modify: `apps/web/lib/api/http.ts`, `apps/web/lib/api/live-endpoints.ts`, `apps/web/lib/api/http.test.ts`

**Interfaces:**
- Consumes: backend `GET /api/token/{mint}` → `TokenDetail` (Task 5).

- [ ] **Step 1: http.ts getToken gerçek**

`http.ts`'de `getToken: notReady` satırını değiştir. Mevcut `getJson` helper'ını ve `getTokens` desenini izle; `getToken` kontrat imzası `(mint: string) => Promise<TokenDetail>`:
```ts
  getToken: (mint: string) => getJson<TokenDetail>(`/api/token/${encodeURIComponent(mint)}`),
```
`TokenDetail` tipini `import type` listesine ekle (http.ts başında types'tan import). `getKpis`/`getRadar` `notReady` KALIR (Alt-proje 2).

- [ ] **Step 2: live-endpoints.ts**

`LIVE_ENDPOINTS` set'ine `"getToken"` ekle.

- [ ] **Step 3: http.test.ts testi ekle**

Mevcut `http.test.ts` desenini izle (fetch mock). `getToken('mint')`'in `/api/token/mint` fetch edip dönen JSON'u `TokenDetail` olarak verdiğini doğrula. Mevcut test dosyasındaki mock/fetch yardımcılarını kullan; yeni bir mock kurma deseni icat etme — dosyadaki mevcut testlerden birini kopyalayıp uyarlayarak yaz (ör. `getTokens` testi varsa onu baz al). Örnek iskelet:
```ts
it("getToken gerçek endpoint'i çağırır", async () => {
  const detail = { id: "M", mint: "M", symbol: "TST", scores: {}, metrics: {}, series: {}, risks: {} };
  // mevcut dosyadaki fetch-mock helper'ıyla /api/token/M → detail döndür
  // ... (dosyanın kendi mock desenini kullan)
  const api = /* httpApi */;
  const got = await api.getToken("M");
  expect(got.symbol).toBe("TST");
});
```
> Implementer: `http.test.ts`'i OKU, oradaki gerçek mock desenini kullan (bu iskeleti birebir kopyalama — dosyanın kendi yardımcılarına uyarla). Amaç: getToken'ın doğru path'i fetch ettiğini + tipli döndüğünü doğrulamak.

- [ ] **Step 4: Çalıştır + commit**

Run: `cd apps/web && npm test` (veya proje test komutu)
Expected: PASS (yeni getToken testi + mevcut ~189 test yeşil; TokenDetail seam tipi derlenir).
```bash
git add apps/web/lib/api/http.ts apps/web/lib/api/live-endpoints.ts apps/web/lib/api/http.test.ts
git commit -m "feat(detail): frontend getToken gerçek fetch + LIVE_ENDPOINTS (1c)"
```

---

### Task 8: Yaşayan dokümanlar + review handoff

**Files:**
- Modify: `docs/progress.md`, `docs/superpowers/followups-frontend.md`

- [ ] **Step 1: `docs/progress.md` "Backend programı" — Slice 1c ekle**

Kapsam (getToken gerçek: header+OHLCV grafik+holders), stack (GeckoTerminal keysiz + mevcut Helius holders), teslim (store TokenDetail seam + TokenDetailService + Helius holders + /api/token/{mint} + frontend getToken canlı), dürüst durum (header/series.price&volume/holders gerçek; scores/risks/davranış-metrikleri/liq&holder-serisi nötr→A2), kabul (deploy'da), Alt-proje 1'in TAMAMLANDIĞI (1a+1b+1c).

- [ ] **Step 2: `followups-frontend.md` — 1c notu**

scores/risks/davranış-metrikleri→A2; likidite/holder serisi→ileride örnekleme; holders "N+" gerçek-total gösterimi seam `number` olduğu için ertelendi (capped→floor); bilinmeyen-mint pool keşfi→YAGNI; Helius getTokenAccounts + GeckoTerminal OHLCV alan-adı kalibrasyonu→deploy'da. Sessiz düşürme yok.

- [ ] **Step 3: Commit**

```bash
git add docs/progress.md docs/superpowers/followups-frontend.md
git commit -m "docs(ingestion): slice 1c yaşayan kayıt — Token Detail teslim + Alt-proje 1 tamam (1c)"
```

- [ ] **Step 4: Final whole-branch review handoff**

SDD: tüm branch için final review (opus). `go build ./... && go vet ./... && go test ./... -race` + `apps/web` testleri yeşil doğrula, sonra merge/deploy DUR-noktası (kullanıcı onayı).

---

## Self-Review (plan yazarı)

**1. Spec coverage:**
- getToken gerçek + endpoint → Task 5. ✓
- Header (price/priceChange24h/marketCap/liquidity/volume24h) → Task 2 (Pool alanları) + Task 4 (map). ✓
- OHLCV yaşa-uyarlı series.price+volume → Task 2 (OHLCV) + Task 4 (tf seçimi+map). ✓
- Holders (Helius sınırlı) → Task 3 + Task 4 (map). ✓
- Nötr placeholder (4 ScoreKey/risks/metrics/liq&holder serisi) → Task 1 (struct) + Task 4 (`neutralScores`+boş slice). ✓
- TokenDetailBase (mint→pool) + 404 → Task 1 + Task 4 (ok=false) + Task 5 (404). ✓
- Cache ~20s → Task 4. ✓
- Frontend getToken + LIVE_ENDPOINTS → Task 7. ✓
- Config → Task 6. ✓
- Bilinmeyen-mint 404, seam birebir, kapsam dışı (A2) → Task 4/5/8. ✓
- Testler (OHLCV/holders/service/handler/frontend) → Task 2/3/4/5/7. ✓

**2. Placeholder scan:** Kod blokları somut. Task 7 frontend testi "dosyanın mevcut mock desenini kullan" yönergesi (iskele değil) — çünkü http.test.ts'in mock helper'ını görmeden birebir kod yazmak kırılgan; implementer dosyayı okuyup uyarlar (gerekçeli, placeholder değil). Helius/GeckoTerminal alan-adı kalibrasyonu açık deferred (1a/1b deseni). ✓

**3. Type consistency:**
- `TokenDetailBase(ctx, mint) (TokenDetailBase, bool, error)` — Task 1 tanımı ↔ Task 4 `DetailStore` ↔ fake/stub'lar tutarlı. ✓
- `OHLCV(ctx, poolAddr, tf, limit) ([]Candle, error)` — Task 2 ↔ Task 4 kullanımı. ✓
- `HoldersCount(ctx, mint, cap) (int, bool, error)` — Task 3 ↔ Task 4 `HoldersProvider` ↔ main noopHolders. ✓
- `Build(ctx, mint) (store.TokenDetail, bool, error)` — Task 4 ↔ Task 5 `TokenDetailProvider` ↔ main `detailSvc`. ✓
- `Pool.{PriceChangeH24,MarketCapUSD,Vol24h}` — Task 2 ↔ Task 4 map. ✓
- `store.TokenDetail` JSON tag'leri (camelCase) ↔ TS `TokenDetail` ↔ Task 1 testi. ✓
- `scores` map anahtarları = 4 ScoreKey (`neutralScores`) ↔ TS `Record<ScoreKey,...>`. ✓
