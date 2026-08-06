# Slice 1b — REST Keşif + Token Enrichment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** GeckoTerminal REST ile WS-bağımsız token keşfi + piyasa enrichment ekleyerek `getTokens`/`getEvents`'i canlı, tazelenen, gerçek piyasa verili hale getirmek.

**Architecture:** Yeni `internal/market/` paketi bir `MarketProvider` arayüzü (DIP) arkasında `GeckoTerminalClient` sunar; iki odaklı poller — `Discoverer` (yeni havuzları keşfeder, kimlik+olay yazar) ve `Enricher` (bilinen token'ların piyasa alanlarını periyodik günceller) — arayüze ve store'a bağımlıdır (SRP). Mevcut Helius ingestion worker'ı bozulmadan yanında çalışır.

**Tech Stack:** Go 1.24, `database/sql` + pgx/v5 stdlib driver, goose migrations, stdlib `net/http` + `encoding/json` (GeckoTerminal keysiz REST — yeni dependency YOK).

## Global Constraints

- **Go sürümü:** `go 1.24` (go.mod ile birebir; değiştirme).
- **Yeni dependency yok:** GeckoTerminal keysiz; yalnız stdlib kullan (`net/http`, `encoding/json`, `strconv`, `time`).
- **Seam kontratı değişmez:** `TokenRow`/`FeedEvent` JSON şekli (`apps/web/lib/api/types.ts`) sabit. `subscribeTokens` → `[]TokenRow` array (liste DEĞİŞTİRİLİR); `subscribeEvents` → tekil `EventRow`.
- **Dürüstlük:** Yalnız gerçek bilinen alanlar doldurulur. `creatorScore`/`safetyScore`/`signal` nötr (Alt-proje 2); `holders` boş (1c). Sahte veri yok.
- **Frontend'e dokunma:** 1b saf backend. `apps/web` altında değişiklik yok.
- **DB testleri** `DATABASE_URL` yoksa `t.Skip` (yerel Postgres yok — 1a deseni).
- **Clean/SOLID:** Her dosya tek sorumluluk; tüketici-tanımlı dar arayüzler; enjekte edilebilir bağımlılıklar (clock, http.Client, provider).

---

## File Structure

**Create:**
- `apps/api-go/internal/store/migrations/0003_add_token_market_columns.sql` — `pool_address` + `spark` kolonları.
- `apps/api-go/internal/market/provider.go` — `Pool`, `MarketProvider`, `Broadcaster`, dex→launchpad eşlemesi.
- `apps/api-go/internal/market/provider_test.go` — dex eşleme testleri.
- `apps/api-go/internal/market/geckoterminal.go` — `GeckoTerminalClient` (NewPools + PoolsByAddresses + JSON parse).
- `apps/api-go/internal/market/geckoterminal_test.go` — fixture-tabanlı parse/filtre testleri.
- `apps/api-go/internal/market/testdata/new_pools.json`, `pools_multi.json` — HTTP fixture'ları.
- `apps/api-go/internal/market/discoverer.go` — `Discoverer` poller.
- `apps/api-go/internal/market/discoverer_test.go`.
- `apps/api-go/internal/market/enricher.go` — `Enricher` poller + momentum/spark helper'ları.
- `apps/api-go/internal/market/enricher_test.go`.

**Modify:**
- `apps/api-go/internal/store/tokens.go` — DTO'lar (`DiscoveredToken`/`MarketUpdate`/`EnrichTarget`) + `TokenStore` arayüzüne 3 metot + postgres impl + `RecentTokens` spark parse.
- `apps/api-go/internal/store/fake_ingest.go` — `fakeTokenStore` yeni metotları + pool/spark saklama.
- `apps/api-go/internal/ingest/worker_test.go` — `failingTokenStore` + `snapshotFailingTokenStore` stub'larına 3 no-op metot (arayüz genişledi).
- `apps/api-go/internal/store/postgres_ingest_test.go` — (Task 2) yeni metotlar için round-trip iddiaları.
- `apps/api-go/internal/config/config.go` — market env alanları + helper'lar.
- `apps/api-go/cmd/server/main.go` — Discoverer + Enricher wiring.
- `docs/progress.md`, memory — yaşayan kayıt (Task 8).

---

### Task 1: Store — migration 0003 + DTO'lar + arayüz + fake + stub'lar

**Files:**
- Create: `apps/api-go/internal/store/migrations/0003_add_token_market_columns.sql`
- Modify: `apps/api-go/internal/store/tokens.go` (DTO'lar + arayüz + postgres metotları)
- Modify: `apps/api-go/internal/store/fake_ingest.go`
- Modify: `apps/api-go/internal/ingest/worker_test.go` (iki stub'a no-op metotlar)
- Test: `apps/api-go/internal/store/tokens_fake_test.go` (Create)

**Interfaces:**
- Produces:
  - `store.DiscoveredToken{Mint, Name, Symbol, Launchpad, PoolAddr string; FirstSeenTs int64}`
  - `store.MarketUpdate{Mint string; Price, Liquidity, Vol5m, Momentum float64; Spark []float64}`
  - `store.EnrichTarget{Mint, PoolAddr string; Spark []float64}`
  - `TokenStore` arayüzüne eklenenler:
    - `UpsertDiscovered(ctx context.Context, d DiscoveredToken) (inserted bool, err error)`
    - `UpdateMarket(ctx context.Context, m MarketUpdate) error`
    - `EnrichTargets(ctx context.Context, limit int) ([]EnrichTarget, error)`

- [ ] **Step 1: Migration dosyasını yaz**

`apps/api-go/internal/store/migrations/0003_add_token_market_columns.sql`:
```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS pool_address TEXT NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS spark TEXT NOT NULL DEFAULT ''; -- JSON []float64

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS spark;
ALTER TABLE tokens DROP COLUMN IF EXISTS pool_address;
```

- [ ] **Step 2: DTO'ları + arayüz genişlemesini `tokens.go`'ya ekle**

`tokens.go` içinde `TokenStore` arayüzünü genişlet ve DTO'ları ekle (mevcut `TokenRow` struct'ının altına):
```go
// DiscoveredToken, GeckoTerminal keşfinden gelen kimlik+havuz bilgisidir (1b).
type DiscoveredToken struct {
	Mint, Name, Symbol, Launchpad, PoolAddr string
	FirstSeenTs                             int64
}

// MarketUpdate, enrichment döngüsünün yazdığı piyasa alanlarıdır (1b).
type MarketUpdate struct {
	Mint                              string
	Price, Liquidity, Vol5m, Momentum float64
	Spark                             []float64
}

// EnrichTarget, enrichment için gereken minimum bilgidir: hangi havuzu sorgulayacağı + mevcut spark.
type EnrichTarget struct {
	Mint, PoolAddr string
	Spark          []float64
}

type TokenStore interface {
	UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64) error
	RecentTokens(ctx context.Context, limit int) ([]TokenRow, error)
	// 1b: keşif (kimlik+havuz) — inserted, token'ın YENİ eklendiğini bildirir (event spam'i önler).
	UpsertDiscovered(ctx context.Context, d DiscoveredToken) (inserted bool, err error)
	// 1b: enrichment (piyasa alanları).
	UpdateMarket(ctx context.Context, m MarketUpdate) error
	// 1b: enrichment hedefleri (havuz adresi olan token'lar, en yeni önce).
	EnrichTargets(ctx context.Context, limit int) ([]EnrichTarget, error)
}
```

- [ ] **Step 3: Postgres implementasyonlarını `tokens.go`'ya ekle**

`UpsertToken`'ı DEĞİŞTİRME (1a Helius yolu kullanıyor). Aşağıdakileri ekle ve `RecentTokens`'ı spark okuyacak şekilde güncelle. Dosya başına `import`'a `"encoding/json"` ekle.
```go
func (p *postgresStore) UpsertDiscovered(ctx context.Context, d DiscoveredToken) (bool, error) {
	const q = `INSERT INTO tokens (mint, symbol, name, launchpad, pool_address, first_seen_ts)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (mint) DO UPDATE SET
			symbol = EXCLUDED.symbol, name = EXCLUDED.name,
			launchpad = EXCLUDED.launchpad, pool_address = EXCLUDED.pool_address
		RETURNING (xmax = 0) AS inserted`
	var inserted bool
	err := p.db.QueryRowContext(ctx, q, d.Mint, d.Symbol, d.Name, d.Launchpad, d.PoolAddr, d.FirstSeenTs).Scan(&inserted)
	return inserted, err
}

func (p *postgresStore) UpdateMarket(ctx context.Context, m MarketUpdate) error {
	sparkJSON, err := json.Marshal(m.Spark)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET price=$2, liquidity=$3, vol5m=$4, momentum=$5, spark=$6 WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, m.Mint, m.Price, m.Liquidity, m.Vol5m, m.Momentum, string(sparkJSON))
	return err
}

func (p *postgresStore) EnrichTargets(ctx context.Context, limit int) ([]EnrichTarget, error) {
	const q = `SELECT mint, pool_address, spark FROM tokens
		WHERE pool_address <> '' ORDER BY first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EnrichTarget, 0, limit)
	for rows.Next() {
		var t EnrichTarget
		var sparkJSON string
		if err := rows.Scan(&t.Mint, &t.PoolAddr, &sparkJSON); err != nil {
			return nil, err
		}
		t.Spark = parseSparkJSON(sparkJSON)
		out = append(out, t)
	}
	return out, rows.Err()
}

// parseSparkJSON, boş/bozuk JSON'da boş dilim döner (asla nil değil).
func parseSparkJSON(s string) []float64 {
	if s == "" {
		return []float64{}
	}
	var out []float64
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return []float64{}
	}
	return out
}
```
`RecentTokens` içinde `SELECT`'e `spark` ekle ve `t.Spark = []float64{}` satırını değiştir:
```go
	const q = `SELECT mint, symbol, name, first_seen_ts, price, liquidity, vol5m, holders,
		creator_score, safety_score, momentum, spark FROM tokens ORDER BY first_seen_ts DESC LIMIT $1`
	// ... rows.Next() döngüsünde:
		var t TokenRow
		var firstSeen int64
		var sparkJSON string
		if err := rows.Scan(&t.Mint, &t.Symbol, &t.Name, &firstSeen, &t.Price, &t.Liquidity,
			&t.Vol5m, &t.Holders, &t.CreatorScore, &t.SafetyScore, &t.Momentum, &sparkJSON); err != nil {
			return nil, err
		}
		t.ID = t.Mint
		t.AgeSeconds = now - firstSeen
		if t.AgeSeconds < 0 {
			t.AgeSeconds = 0
		}
		t.Spark = parseSparkJSON(sparkJSON)
```

- [ ] **Step 4: `fakeTokenStore`'u yeni alanlar+metotlarla güncelle (`fake_ingest.go`)**

`fakeTokenStore`'u içsel kayıt tutacak şekilde yeniden yaz (TokenRow'da pool_address yok):
```go
type fakeTok struct {
	row       TokenRow
	poolAddr  string
	firstSeen int64
}

type fakeTokenStore struct {
	mu    sync.Mutex
	byID  map[string]fakeTok
	order []string
}

func NewFakeTokenStore() TokenStore { return &fakeTokenStore{byID: map[string]fakeTok{}} }

func (f *fakeTokenStore) UpsertToken(_ context.Context, t TokenRow, firstSeenTs int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.Spark == nil {
		t.Spark = []float64{}
	}
	cur, ok := f.byID[t.ID]
	if !ok {
		f.order = append(f.order, t.ID)
	}
	cur.row = t
	cur.firstSeen = firstSeenTs
	f.byID[t.ID] = cur
	return nil
}

func (f *fakeTokenStore) UpsertDiscovered(_ context.Context, d DiscoveredToken) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[d.Mint]
	inserted := !ok
	if inserted {
		f.order = append(f.order, d.Mint)
		cur.row = TokenRow{ID: d.Mint, Mint: d.Mint, Spark: []float64{}}
		cur.firstSeen = d.FirstSeenTs
	}
	cur.row.Name, cur.row.Symbol = d.Name, d.Symbol
	cur.poolAddr = d.PoolAddr
	f.byID[d.Mint] = cur
	return inserted, nil
}

func (f *fakeTokenStore) UpdateMarket(_ context.Context, m MarketUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[m.Mint]
	if !ok {
		return nil
	}
	cur.row.Price, cur.row.Liquidity = m.Price, m.Liquidity
	cur.row.Vol5m, cur.row.Momentum = m.Vol5m, m.Momentum
	if m.Spark == nil {
		m.Spark = []float64{}
	}
	cur.row.Spark = m.Spark
	f.byID[m.Mint] = cur
	return nil
}

func (f *fakeTokenStore) EnrichTargets(_ context.Context, limit int) ([]EnrichTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]EnrichTarget, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		t := f.byID[f.order[i]]
		if t.poolAddr == "" {
			continue
		}
		out = append(out, EnrichTarget{Mint: t.row.Mint, PoolAddr: t.poolAddr, Spark: t.row.Spark})
	}
	return out, nil
}

func (f *fakeTokenStore) RecentTokens(_ context.Context, limit int) ([]TokenRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TokenRow, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, f.byID[f.order[i]].row)
	}
	return out, nil
}
```

- [ ] **Step 5: `worker_test.go`'daki iki stub'a no-op metotları ekle**

`failingTokenStore` ve `snapshotFailingTokenStore` artık `TokenStore`'u karşılamıyor. Her ikisine ekle (import zaten `store`):
```go
func (f *failingTokenStore) UpsertDiscovered(context.Context, store.DiscoveredToken) (bool, error) {
	return false, nil
}
func (f *failingTokenStore) UpdateMarket(context.Context, store.MarketUpdate) error { return nil }
func (f *failingTokenStore) EnrichTargets(context.Context, int) ([]store.EnrichTarget, error) {
	return nil, nil
}
```
Aynı üç metodu `snapshotFailingTokenStore` için de ekle (aynı gövdeler).

- [ ] **Step 6: Fake store testini yaz** (`store/tokens_fake_test.go`)

```go
package store

import (
	"context"
	"testing"
)

func TestFakeUpsertDiscoveredInsertedFlag(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	d := DiscoveredToken{Mint: "M1", Name: "One", Symbol: "ONE", Launchpad: "Pump.fun", PoolAddr: "P1", FirstSeenTs: 100}
	ins, err := f.UpsertDiscovered(ctx, d)
	if err != nil || !ins {
		t.Fatalf("ilk keşif inserted=true olmalı, got inserted=%v err=%v", ins, err)
	}
	ins2, _ := f.UpsertDiscovered(ctx, d)
	if ins2 {
		t.Fatal("ikinci keşif inserted=false olmalı (dedup)")
	}
}

func TestFakeUpdateMarketAndEnrichTargets(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "M1", PoolAddr: "P1", FirstSeenTs: 1})
	if err := f.UpdateMarket(ctx, MarketUpdate{Mint: "M1", Price: 2, Liquidity: 3, Vol5m: 4, Momentum: 60, Spark: []float64{1, 2}}); err != nil {
		t.Fatal(err)
	}
	toks, _ := f.RecentTokens(ctx, 10)
	if len(toks) != 1 || toks[0].Price != 2 || toks[0].Momentum != 60 || len(toks[0].Spark) != 2 {
		t.Fatalf("UpdateMarket RecentTokens'a yansımadı: %+v", toks)
	}
	targets, _ := f.EnrichTargets(ctx, 10)
	if len(targets) != 1 || targets[0].PoolAddr != "P1" || len(targets[0].Spark) != 2 {
		t.Fatalf("EnrichTargets beklenen hedefi vermedi: %+v", targets)
	}
}

func TestFakeEnrichTargetsSkipsNoPool(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	f.UpsertToken(ctx, TokenRow{ID: "M2", Mint: "M2"}, 1) // pool_address yok
	targets, _ := f.EnrichTargets(ctx, 10)
	if len(targets) != 0 {
		t.Fatalf("havuzsuz token enrichment hedefi olmamalı: %+v", targets)
	}
}
```

- [ ] **Step 7: Testleri çalıştır**

Run: `cd apps/api-go && go test ./internal/store/... ./internal/ingest/...`
Expected: PASS (yeni fake testleri + mevcut worker/store testleri; arayüz genişlemesi derleniyor)

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/internal/store apps/api-go/internal/ingest/worker_test.go
git commit -m "feat(market): store 1b — discovery/market DTO'ları, UpsertDiscovered/UpdateMarket/EnrichTargets, migration 0003"
```

---

### Task 2: Postgres round-trip testi (gated)

**Files:**
- Modify: `apps/api-go/internal/store/postgres_ingest_test.go`

**Interfaces:**
- Consumes: Task 1'in postgres metotları (`UpsertDiscovered`, `UpdateMarket`, `EnrichTargets`, spark'lı `RecentTokens`).

- [ ] **Step 1: Round-trip testini ekle**

`postgres_ingest_test.go` sonuna ekle:
```go
func TestPostgresMarketRoundTrip(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL yok — integration testi atlanıyor")
	}
	ctx := context.Background()
	b, cleanup, err := OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	defer cleanup()

	ins, err := b.Tokens.UpsertDiscovered(ctx, DiscoveredToken{
		Mint: "MintMk", Name: "MarketTok", Symbol: "MKT", Launchpad: "Pump.fun", PoolAddr: "PoolMk", FirstSeenTs: 10})
	if err != nil || !ins {
		t.Fatalf("ilk UpsertDiscovered inserted olmalı: inserted=%v err=%v", ins, err)
	}
	ins2, _ := b.Tokens.UpsertDiscovered(ctx, DiscoveredToken{Mint: "MintMk", Symbol: "MKT", PoolAddr: "PoolMk", FirstSeenTs: 10})
	if ins2 {
		t.Fatal("ikinci UpsertDiscovered inserted=false olmalı")
	}
	if err := b.Tokens.UpdateMarket(ctx, MarketUpdate{Mint: "MintMk", Price: 0.5, Liquidity: 1000, Vol5m: 50, Momentum: 72, Spark: []float64{1, 2, 3}}); err != nil {
		t.Fatal(err)
	}
	targets, err := b.Tokens.EnrichTargets(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tg := range targets {
		if tg.Mint == "MintMk" {
			found = true
			if tg.PoolAddr != "PoolMk" || len(tg.Spark) != 3 {
				t.Fatalf("EnrichTarget yanlış: %+v", tg)
			}
		}
	}
	if !found {
		t.Fatal("EnrichTargets keşfedilen token'ı içermeli")
	}
	toks, _ := b.Tokens.RecentTokens(ctx, 50)
	for _, tk := range toks {
		if tk.Mint == "MintMk" && (tk.Price != 0.5 || tk.Momentum != 72 || len(tk.Spark) != 3) {
			t.Fatalf("RecentTokens piyasa alanlarını yansıtmadı: %+v", tk)
		}
	}
}
```

- [ ] **Step 2: Testi çalıştır (DB yoksa skip)**

Run: `cd apps/api-go && go test ./internal/store/... -run TestPostgresMarketRoundTrip -v`
Expected: SKIP ("DATABASE_URL yok") — yerelde DB yok; deploy'da/DB'li ortamda PASS. Derlemenin geçtiğini doğrular.

- [ ] **Step 3: Commit**

```bash
git add apps/api-go/internal/store/postgres_ingest_test.go
git commit -m "test(market): postgres market round-trip (gated) — UpsertDiscovered/UpdateMarket/EnrichTargets/spark"
```

---

### Task 3: `market` paketi çekirdeği — Pool, MarketProvider, dex eşlemesi

**Files:**
- Create: `apps/api-go/internal/market/provider.go`
- Test: `apps/api-go/internal/market/provider_test.go`

**Interfaces:**
- Produces:
  - `market.Pool{PoolAddr, Mint, Name, Symbol, Dex string; Price, LiquidityUSD, Vol5m, PriceChangeH1 float64; CreatedAtUnix int64}`
  - `market.MarketProvider` arayüzü: `NewPools(ctx) ([]Pool, error)`, `PoolsByAddresses(ctx, []string) ([]Pool, error)`
  - `market.Broadcaster` arayüzü: `Broadcast(topic string, payload any)`
  - `market.DexToLaunchpad(dexID string) (launchpad string, ok bool)` — desteklenen dex mi + gösterim adı

- [ ] **Step 1: dex eşleme testini yaz** (`provider_test.go`)

```go
package market

import "testing"

func TestDexToLaunchpad(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		okExp bool
	}{
		{"pumpfun", "Pump.fun", true},
		{"pump-fun", "Pump.fun", true},
		{"raydium", "Raydium", true},
		{"raydium-clmm", "Raydium", true},
		{"raydium-cpmm", "Raydium", true},
		{"orca", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := DexToLaunchpad(c.in)
		if ok != c.okExp || got != c.want {
			t.Errorf("DexToLaunchpad(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.okExp)
		}
	}
}
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd apps/api-go && go test ./internal/market/... -run TestDexToLaunchpad`
Expected: FAIL (paket/fonksiyon yok — derlenmiyor)

- [ ] **Step 3: `provider.go`'yu yaz**

```go
// Package market, agregatör (GeckoTerminal) tabanlı token keşif+enrichment sağlar (Slice 1b).
// WS ingestion'dan bağımsız REST döngüleridir.
package market

import "context"

// Pool, bir agregatör havuz kaydının kaynak-bağımsız görünümüdür.
type Pool struct {
	PoolAddr      string
	Mint          string // base token adresi
	Name, Symbol  string
	Dex           string // agregatör dex kimliği (ör. "pumpfun", "raydium")
	Price         float64
	LiquidityUSD  float64
	Vol5m         float64
	PriceChangeH1 float64 // yüzde
	CreatedAtUnix int64
}

// MarketProvider, piyasa verisi kaynağıdır (DIP). GeckoTerminal ilk somut impl; DexScreener sonra (OCP).
type MarketProvider interface {
	NewPools(ctx context.Context) ([]Pool, error)
	PoolsByAddresses(ctx context.Context, poolAddrs []string) ([]Pool, error)
}

// Broadcaster, snapshot/olayları client'lara yayar (tüketici-tanımlı arayüz; ws.Hub karşılar).
type Broadcaster interface {
	Broadcast(topic string, payload any)
}

// launchpadByDex, desteklenen dex kimliklerini SENTINEL gösterim adına eşler.
// Buradaki anahtarlar canlı GeckoTerminal örneğiyle doğrulanmalı (deploy kalibrasyonu).
var launchpadByDex = map[string]string{
	"pumpfun":      "Pump.fun",
	"pump-fun":     "Pump.fun",
	"raydium":      "Raydium",
	"raydium-clmm": "Raydium",
	"raydium-cpmm": "Raydium",
}

// DexToLaunchpad, dex kimliğini gösterim adına çevirir; desteklenmiyorsa ok=false (filtrele).
func DexToLaunchpad(dexID string) (string, bool) {
	name, ok := launchpadByDex[dexID]
	return name, ok
}
```

- [ ] **Step 4: Testin geçtiğini doğrula**

Run: `cd apps/api-go && go test ./internal/market/... -run TestDexToLaunchpad -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/market/provider.go apps/api-go/internal/market/provider_test.go
git commit -m "feat(market): Pool + MarketProvider + Broadcaster arayüzleri + dex→launchpad eşlemesi"
```

---

### Task 4: GeckoTerminalClient — NewPools + PoolsByAddresses

**Files:**
- Create: `apps/api-go/internal/market/geckoterminal.go`
- Create: `apps/api-go/internal/market/testdata/new_pools.json`
- Create: `apps/api-go/internal/market/testdata/pools_multi.json`
- Test: `apps/api-go/internal/market/geckoterminal_test.go`

**Interfaces:**
- Consumes: `Pool`, `MarketProvider`, `DexToLaunchpad` (Task 3).
- Produces: `NewGeckoTerminalClient(baseURL string, hc *http.Client) *GeckoTerminalClient` (MarketProvider'ı karşılar).

**NOT (deploy kalibrasyonu):** GeckoTerminal yanıtı JSON:API biçimidir. Aşağıdaki fixture'lar dokümante edilen şekli yansıtır; canlı örnekle alan adları farklıysa parse+fixture BİRLİKTE güncellenir (1a Raydium account-index kalibrasyonu deseni). Endpoint'ler: `GET {base}/networks/solana/new_pools?include=base_token` ve `GET {base}/networks/solana/pools/multi/{addr,addr,...}`.

- [ ] **Step 1: Fixture'ları yaz**

`testdata/new_pools.json`:
```json
{
  "data": [
    {
      "id": "solana_ABCpool",
      "type": "pool",
      "attributes": {
        "address": "ABCpool",
        "name": "TROLL / SOL",
        "base_token_price_usd": "0.00004212",
        "reserve_in_usd": "82400.50",
        "pool_created_at": "2026-08-06T10:00:00Z",
        "volume_usd": { "m5": "41200.0", "h1": "120000.0", "h24": "900000.0" },
        "price_change_percentage": { "m5": "3.1", "h1": "22.4", "h24": "-5.0" }
      },
      "relationships": {
        "base_token": { "data": { "id": "solana_TROLLmint", "type": "token" } },
        "dex": { "data": { "id": "pumpfun", "type": "dex" } }
      }
    },
    {
      "id": "solana_ORCApool",
      "type": "pool",
      "attributes": {
        "address": "ORCApool",
        "name": "SKIP / SOL",
        "base_token_price_usd": "1.0",
        "reserve_in_usd": "5000.0",
        "pool_created_at": "2026-08-06T10:01:00Z",
        "volume_usd": { "m5": "100.0" },
        "price_change_percentage": { "h1": "0.0" }
      },
      "relationships": {
        "base_token": { "data": { "id": "solana_SKIPmint", "type": "token" } },
        "dex": { "data": { "id": "orca", "type": "dex" } }
      }
    }
  ],
  "included": [
    { "id": "solana_TROLLmint", "type": "token", "attributes": { "name": "Troll Face", "symbol": "TROLL", "address": "TROLLmint" } },
    { "id": "solana_SKIPmint", "type": "token", "attributes": { "name": "Skip Me", "symbol": "SKIP", "address": "SKIPmint" } }
  ]
}
```
`testdata/pools_multi.json`:
```json
{
  "data": [
    {
      "id": "solana_ABCpool",
      "type": "pool",
      "attributes": {
        "address": "ABCpool",
        "name": "TROLL / SOL",
        "base_token_price_usd": "0.00005000",
        "reserve_in_usd": "90000.0",
        "volume_usd": { "m5": "5000.0" },
        "price_change_percentage": { "h1": "10.0" }
      },
      "relationships": {
        "base_token": { "data": { "id": "solana_TROLLmint", "type": "token" } },
        "dex": { "data": { "id": "pumpfun", "type": "dex" } }
      }
    }
  ]
}
```

- [ ] **Step 2: Parse testini yaz** (`geckoterminal_test.go`)

```go
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
```

- [ ] **Step 3: Testin başarısız olduğunu doğrula**

Run: `cd apps/api-go && go test ./internal/market/... -run TestNewPools`
Expected: FAIL (NewGeckoTerminalClient yok)

- [ ] **Step 4: `geckoterminal.go`'yu yaz**

```go
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
		Address      string            `json:"address"`
		Name         string            `json:"name"`
		PriceUSD     string            `json:"base_token_price_usd"`
		ReserveUSD   string            `json:"reserve_in_usd"`
		CreatedAt    string            `json:"pool_created_at"`
		VolumeUSD    map[string]string `json:"volume_usd"`
		PriceChange  map[string]string `json:"price_change_percentage"`
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
```

- [ ] **Step 5: Testlerin geçtiğini doğrula**

Run: `cd apps/api-go && go test ./internal/market/... -v`
Expected: PASS (parse + dex filtre + boş-adres kısa devre)

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/market/geckoterminal.go apps/api-go/internal/market/geckoterminal_test.go apps/api-go/internal/market/testdata
git commit -m "feat(market): GeckoTerminalClient — new_pools + pools/multi parse (JSON:API), dex filtresi"
```

---

### Task 5: Discoverer poller

**Files:**
- Create: `apps/api-go/internal/market/discoverer.go`
- Test: `apps/api-go/internal/market/discoverer_test.go`

**Interfaces:**
- Consumes: `MarketProvider`, `Broadcaster`, `DexToLaunchpad`, `Pool` (Task 3/4); `store.TokenStore`, `store.EventStore`, `store.DiscoveredToken`, `store.MarketUpdate`, `store.EventRow` (Task 1).
- Produces: `NewDiscoverer(DiscovererDeps) *Discoverer`; `(*Discoverer).tick(ctx) error`; `(*Discoverer).Run(ctx)`.

- [ ] **Step 1: Testi yaz** (`discoverer_test.go`)

Fake provider + capturing broadcaster:
```go
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
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd apps/api-go && go test ./internal/market/... -run TestDiscoverer`
Expected: FAIL (Discoverer yok)

- [ ] **Step 3: `discoverer.go`'yu yaz**

```go
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
		// Keşifte bedava ilk enrichment (new_pools zaten piyasa verisi taşır).
		if err := x.d.Tokens.UpdateMarket(ctx, store.MarketUpdate{
			Mint: p.Mint, Price: p.Price, Liquidity: p.LiquidityUSD, Vol5m: p.Vol5m,
			Momentum: momentumFromChange(p.PriceChangeH1), Spark: appendSpark(nil, p.Price),
		}); err != nil {
			x.d.Logger.Warn("initial market", "mint", p.Mint, "err", err)
		}
		wrote = true
		if inserted { // yalnız ilk keşifte olay (spam yok)
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
```
> `discoverer.go`, `momentumFromChange` ve `appendSpark` helper'larını kullanır. Bunlar bu task'ın Step 3b'sinde `helpers.go`'da tanımlanır (Task 6/`enricher.go` yalnız kullanır, yeniden tanımlamaz). Task 5'in iki dosyası (`discoverer.go` + `helpers.go`) birlikte derlenir; test Step 4'te çalıştırılır.

- [ ] **Step 3b: Paylaşılan helper'ları `helpers.go`'ya ekle** (Create: `apps/api-go/internal/market/helpers.go`)

```go
package market

const (
	momentumK = 0.5 // +100% → 100, -100% → 0
	sparkMax  = 16  // spark dizisinde tutulan son örnek sayısı
)

// momentumFromChange, kısa-vade fiyat değişimini 0-100 momentum'a çevirir (50=yatay). Skor DEĞİL, saf fiyat aksiyonu.
func momentumFromChange(pctH1 float64) float64 {
	m := 50 + pctH1*momentumK
	if m < 0 {
		return 0
	}
	if m > 100 {
		return 100
	}
	return m
}

// appendSpark, güncel fiyatı spark'a ekler ve son sparkMax örnekle sınırlar.
func appendSpark(cur []float64, price float64) []float64 {
	out := append(append([]float64{}, cur...), price)
	if len(out) > sparkMax {
		out = out[len(out)-sparkMax:]
	}
	return out
}
```
Helper testi (`apps/api-go/internal/market/helpers_test.go`):
```go
package market

import "testing"

func TestMomentumFromChange(t *testing.T) {
	cases := []struct{ in, want float64 }{{0, 50}, {100, 100}, {-100, 0}, {20, 60}, {300, 100}, {-300, 0}}
	for _, c := range cases {
		if got := momentumFromChange(c.in); got != c.want {
			t.Errorf("momentumFromChange(%v)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestAppendSparkCaps(t *testing.T) {
	var s []float64
	for i := 0; i < 20; i++ {
		s = appendSpark(s, float64(i))
	}
	if len(s) != sparkMax {
		t.Fatalf("spark len=%d want %d", len(s), sparkMax)
	}
	if s[len(s)-1] != 19 {
		t.Fatalf("son örnek=%v want 19", s[len(s)-1])
	}
}
```
> **NOT:** Task 6 (`enricher.go`) bu helper'ları KULLANIR ama YENİDEN TANIMLAMAZ (tek tanım `helpers.go`'da).

- [ ] **Step 4: Testleri çalıştır**

Run: `cd apps/api-go && go test ./internal/market/... -run 'TestDiscoverer|TestMomentum|TestAppendSpark' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/market/discoverer.go apps/api-go/internal/market/discoverer_test.go apps/api-go/internal/market/helpers.go apps/api-go/internal/market/helpers_test.go
git commit -m "feat(market): Discoverer poller + momentum/spark helper'ları"
```

---

### Task 6: Enricher poller

**Files:**
- Create: `apps/api-go/internal/market/enricher.go`
- Test: `apps/api-go/internal/market/enricher_test.go`

**Interfaces:**
- Consumes: `MarketProvider`, `Broadcaster`, `momentumFromChange`, `appendSpark` (Task 5 helpers.go); `store.TokenStore`, `store.MarketUpdate`, `store.EnrichTarget` (Task 1).
- Produces: `NewEnricher(EnricherDeps) *Enricher`; `(*Enricher).tick(ctx) error`; `(*Enricher).Run(ctx)`; `chunk([]string, int) [][]string`.

- [ ] **Step 1: Testi yaz** (`enricher_test.go`)

```go
package market

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestEnricherUpdatesMarketAndAppendsSpark(t *testing.T) {
	ts := store.NewFakeTokenStore()
	ctx := context.Background()
	// önceden keşfedilmiş token (havuzlu, spark [1])
	ts.UpsertDiscovered(ctx, store.DiscoveredToken{Mint: "M1", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 1})
	ts.UpdateMarket(ctx, store.MarketUpdate{Mint: "M1", Price: 1, Spark: []float64{1}})

	fp := &fakeProvider{byAddr: []Pool{{PoolAddr: "P1", Mint: "M1", Price: 5, LiquidityUSD: 200, Vol5m: 30, PriceChangeH1: 40}}}
	bc := &capBC{}
	e := NewEnricher(EnricherDeps{Provider: fp, Tokens: ts, Broadcast: bc, Limit: 50})

	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	toks, _ := ts.RecentTokens(ctx, 10)
	if len(toks) != 1 {
		t.Fatalf("token sayısı=%d", len(toks))
	}
	got := toks[0]
	if got.Price != 5 || got.Liquidity != 200 || got.Vol5m != 30 || got.Momentum != 70 {
		t.Fatalf("piyasa güncellenmedi: %+v", got)
	}
	if len(got.Spark) != 2 || got.Spark[1] != 5 { // mevcut [1] + yeni fiyat 5
		t.Fatalf("spark append yanlış: %+v", got.Spark)
	}
	var tokensSnap int
	for _, topic := range bc.topics {
		if topic == "tokens" {
			tokensSnap++
		}
	}
	if tokensSnap == 0 {
		t.Fatal("tokens snapshot broadcast edilmedi")
	}
}

func TestEnricherNoTargetsNoBroadcast(t *testing.T) {
	ts := store.NewFakeTokenStore()
	fp := &fakeProvider{}
	bc := &capBC{}
	e := NewEnricher(EnricherDeps{Provider: fp, Tokens: ts, Broadcast: bc, Limit: 50})
	if err := e.tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(bc.topics) != 0 {
		t.Fatalf("hedef yokken broadcast olmamalı: %v", bc.topics)
	}
}

func TestChunk(t *testing.T) {
	in := []string{"a", "b", "c", "d", "e"}
	got := chunk(in, 2)
	if len(got) != 3 || len(got[0]) != 2 || len(got[2]) != 1 {
		t.Fatalf("chunk yanlış: %v", got)
	}
}
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `cd apps/api-go && go test ./internal/market/... -run 'TestEnricher|TestChunk'`
Expected: FAIL (Enricher/chunk yok)

- [ ] **Step 3: `enricher.go`'yu yaz**

```go
package market

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

const maxPoolBatch = 30 // GeckoTerminal pools/multi başına en fazla adres

type EnricherDeps struct {
	Provider  MarketProvider
	Tokens    store.TokenStore
	Broadcast Broadcaster
	Interval  time.Duration
	Limit     int // enrich edilecek en yeni token sayısı + snapshot penceresi
	Logger    *slog.Logger
}

// Enricher, bilinen (havuzlu) token'ların piyasa alanlarını periyodik günceller (SRP).
type Enricher struct{ d EnricherDeps }

func NewEnricher(d EnricherDeps) *Enricher {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Limit <= 0 {
		d.Limit = 60
	}
	if d.Interval <= 0 {
		d.Interval = 30 * time.Second
	}
	return &Enricher{d: d}
}

func (x *Enricher) Run(ctx context.Context) {
	x.d.Logger.Info("market enricher başladı", "interval", x.d.Interval.String(), "limit", x.d.Limit)
	t := time.NewTicker(x.d.Interval)
	defer t.Stop()
	for {
		if err := x.tick(ctx); err != nil {
			x.d.Logger.Warn("enricher tick", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// tick, hedefleri okur, havuz verisini batch çeker, piyasa alanlarını + spark'ı günceller, snapshot yayınlar.
func (x *Enricher) tick(ctx context.Context) error {
	targets, err := x.d.Tokens.EnrichTargets(ctx, x.d.Limit)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}
	byPool := map[string]EnrichTarget{}
	addrs := make([]string, 0, len(targets))
	for _, t := range targets {
		byPool[t.PoolAddr] = t
		addrs = append(addrs, t.PoolAddr)
	}
	var updated bool
	for _, batch := range chunk(addrs, maxPoolBatch) {
		pools, err := x.d.Provider.PoolsByAddresses(ctx, batch)
		if err != nil {
			x.d.Logger.Warn("pools by addresses", "err", err)
			continue // kısmi başarı: bu batch atla, diğerleri devam
		}
		for _, p := range pools {
			t, ok := byPool[p.PoolAddr]
			if !ok {
				continue
			}
			if err := x.d.Tokens.UpdateMarket(ctx, store.MarketUpdate{
				Mint: t.Mint, Price: p.Price, Liquidity: p.LiquidityUSD, Vol5m: p.Vol5m,
				Momentum: momentumFromChange(p.PriceChangeH1), Spark: appendSpark(t.Spark, p.Price),
			}); err != nil {
				x.d.Logger.Warn("update market", "mint", t.Mint, "err", err)
				continue
			}
			updated = true
		}
	}
	if updated {
		snapshot, err := x.d.Tokens.RecentTokens(ctx, x.d.Limit)
		if err != nil {
			return err
		}
		x.d.Broadcast.Broadcast("tokens", snapshot)
	}
	return nil
}

// chunk, dilimi en fazla size'lık parçalara böler.
func chunk(s []string, size int) [][]string {
	var out [][]string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
```

- [ ] **Step 4: Tüm market paketini test et (-race)**

Run: `cd apps/api-go && go test ./internal/market/... -race -v`
Expected: PASS (client, discoverer, enricher, helpers)

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/market/enricher.go apps/api-go/internal/market/enricher_test.go
git commit -m "feat(market): Enricher poller — batch piyasa güncelleme + spark append + snapshot broadcast"
```

---

### Task 7: Config + main.go wiring

**Files:**
- Modify: `apps/api-go/internal/config/config.go`
- Modify: `apps/api-go/cmd/server/main.go`

**Interfaces:**
- Consumes: `market.NewGeckoTerminalClient`, `market.NewDiscoverer`, `market.NewEnricher`, `market.DiscovererDeps`, `market.EnricherDeps` (Task 3-6); `config.Config` yeni alanları.
- Produces: canlı iki goroutine (Discoverer + Enricher) — config-gated.

- [ ] **Step 1: Config alanlarını + helper'ları ekle** (`config.go`)

`Config` struct'ına ekle:
```go
	GeckoBaseURL     string
	MarketEnabled    bool
	DiscoverInterval int // saniye
	EnrichInterval   int // saniye
	EnrichLimit      int
```
`Load()` içine ekle:
```go
		GeckoBaseURL:     getenv("GECKOTERMINAL_BASE_URL", "https://api.geckoterminal.com/api/v2"),
		MarketEnabled:    getenvBool("MARKET_ENABLED", true),
		DiscoverInterval: getenvInt("MARKET_DISCOVER_INTERVAL_SEC", 30),
		EnrichInterval:   getenvInt("MARKET_ENRICH_INTERVAL_SEC", 30),
		EnrichLimit:      getenvInt("MARKET_ENRICH_LIMIT", 60),
```
`getenvBool` helper'ını ekle:
```go
func getenvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
```

- [ ] **Step 2: Config testini ekle/güncelle**

`apps/api-go/internal/config/config_test.go` yoksa oluştur, varsa ekle:
```go
package config

import (
	"testing"
)

func TestLoadMarketDefaults(t *testing.T) {
	t.Setenv("MARKET_ENABLED", "")
	t.Setenv("GECKOTERMINAL_BASE_URL", "")
	c := Load()
	if !c.MarketEnabled {
		t.Fatal("MARKET_ENABLED default true olmalı")
	}
	if c.GeckoBaseURL == "" || c.DiscoverInterval != 30 || c.EnrichLimit != 60 {
		t.Fatalf("market default'ları yanlış: %+v", c)
	}
}

func TestGetenvBool(t *testing.T) {
	t.Setenv("X_FLAG", "false")
	if getenvBool("X_FLAG", true) {
		t.Fatal("getenvBool 'false' okumalı")
	}
}
```

- [ ] **Step 3: main.go'ya wiring ekle**

`import` bloğuna `"github.com/furkanatesc/sentinel/apps/api-go/internal/market"` ekle. `go worker.Run(ctx)` satırından SONRA ekle:
```go
	// market keşif + enrichment (GeckoTerminal REST — WS'ten bağımsız, Slice 1b)
	if cfg.MarketEnabled {
		gt := market.NewGeckoTerminalClient(cfg.GeckoBaseURL, nil)
		disc := market.NewDiscoverer(market.DiscovererDeps{
			Provider: gt, Tokens: bundle.Tokens, Events: bundle.Events, Broadcast: hub,
			Interval: time.Duration(cfg.DiscoverInterval) * time.Second, SnapshotLimit: cfg.EventsWindow, Logger: logger,
		})
		enr := market.NewEnricher(market.EnricherDeps{
			Provider: gt, Tokens: bundle.Tokens, Broadcast: hub,
			Interval: time.Duration(cfg.EnrichInterval) * time.Second, Limit: cfg.EnrichLimit, Logger: logger,
		})
		go disc.Run(ctx)
		go enr.Run(ctx)
	} else {
		logger.Warn("MARKET_ENABLED=false — market keşif/enrichment kapalı")
	}
```
> `time` zaten import'lu; `hub` `market.Broadcaster`'ı yapısal olarak karşılar (Broadcast imzası aynı).

- [ ] **Step 4: Build + vet + tüm testler (-race)**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`
Expected: PASS (tüm paketler derlenir, market wiring dahil; DB testleri skip)

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/config apps/api-go/cmd/server/main.go
git commit -m "feat(market): config alanları + main wiring — Discoverer/Enricher goroutine'leri (config-gated)"
```

---

### Task 8: Yaşayan dokümanlar + review handoff

**Files:**
- Modify: `docs/progress.md`
- Modify: `docs/superpowers/followups-frontend.md` (varsa ilgili bölüm)
- (Memory `sentinel-backend-program` / `sentinel-deployment` — implementasyon tamamlanınca ana oturumda güncellenir)

- [ ] **Step 1: `docs/progress.md` "Backend programı" bölümünü güncelle**

Slice 1b'yi ekle: kapsam (REST keşif + enrichment), stack (GeckoTerminal keysiz), teslim edilen paketler (`internal/market/`, migration 0003, store metotları), dürüst alan durumu (price/liquidity/vol5m/momentum/spark gerçek; holders→1c, skorlar→A2, Overview→A2), kabul kriterleri (deploy'da doğrulanır), kapsam dışı (Token Detail→1c, DexScreener OCP-hazır).

- [ ] **Step 2: `followups-frontend.md`'ye 1b notunu ekle**

Overview (getKpis/getRadar) A2'ye bağlı; Token Detail (getToken)+OHLCV serisi+Helius holders 1c'de; DexScreener ikinci provider gerekince (OCP-hazır). Sessiz düşürme yok — hepsi açık işaretli.

- [ ] **Step 3: Commit**

```bash
git add docs/progress.md docs/superpowers/followups-frontend.md
git commit -m "docs(ingestion): slice 1b yaşayan kayıt — market keşif+enrichment teslim + kapsam dışı işaretleri"
```

- [ ] **Step 4: Final whole-branch review handoff**

SDD kullanılıyorsa: tüm branch için final review subagent'ı çağır (opus). Aksi halde: `go build ./... && go vet ./... && go test ./... -race` yeşil + `apps/web` testleri değişmeden yeşil olduğunu son kez doğrula, sonra merge/deploy DUR-noktası (kullanıcı onayı — global kural).

---

## Self-Review (plan yazarı)

**1. Spec coverage:**
- REST keşif (GeckoTerminal new_pools) → Task 4 (client) + Task 5 (Discoverer). ✓
- Enrichment (price/liquidity/vol5m/momentum/spark) → Task 6 (Enricher) + Task 5 (keşifte ilk enrichment). ✓
- momentum türetme → Task 5 helpers.go (`momentumFromChange`). ✓
- spark birikimli → Task 5 helpers.go (`appendSpark`) + Task 1 (spark kolonu/parse). ✓
- pool_address kolonu → Task 1 (migration 0003 + DTO + store). ✓
- MarketProvider/DIP/OCP → Task 3. ✓
- İki odaklı poller (SRP) → Task 5/6. ✓
- Config (keysiz, aralık/limit) → Task 7. ✓
- Frontend'e dokunmama → hiçbir task `apps/web`'e dokunmuyor. ✓
- Overview→A2, Token Detail→1c, holders→1c (kapsam dışı, işaretli) → Task 8. ✓
- Testler (client fixture, discoverer/enricher fake, store round-trip) → Task 2/4/5/6. ✓

**2. Placeholder scan:** Tüm kod blokları somut; "TBD/TODO/uygun hata yönetimi" yok. GeckoTerminal alan adı kalibrasyon notu açık ve 1a deseniyle gerekçeli (placeholder değil). ✓

**3. Type consistency:**
- `UpsertDiscovered(...) (bool, error)` — Task 1 tanımı, Task 5 kullanımı, worker_test.go + fake stub'ları tutarlı. ✓
- `MarketUpdate`/`EnrichTarget`/`DiscoveredToken` alanları Task 1 ↔ Task 5/6 tutarlı. ✓
- `Pool` alanları Task 3 ↔ Task 4 parse ↔ Task 5/6 kullanımı tutarlı. ✓
- `momentumFromChange`/`appendSpark` tek tanım (helpers.go, Task 5); Task 6 yalnız kullanır. ✓
- `Broadcaster.Broadcast(topic, payload)` — market tanımı ↔ ws.Hub yapısal uyumu (main.go). ✓
- `chunk`/`maxPoolBatch`/`sparkMax`/`momentumK` tek yerde. ✓

Not: Task 5'te helper konumu için düzeltme (Step 3b) planın kendisinde açık — implementer helpers.go'yu Task 5'te oluşturur, Task 6 kullanır.
