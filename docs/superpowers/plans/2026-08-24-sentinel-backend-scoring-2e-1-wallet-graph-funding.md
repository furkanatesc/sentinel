# 2e-1 Wallet Graph — Creator-Funding Kümeleme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `getWalletGraph()`'i gerçeğe döndür — aynı funding cüzdanından beslenen creator'ları kümeleyip bundler/seri-rug operatörü god node'larını (`funding_wallet`) yüzeye çıkar.

**Architecture:** Yeni `internal/walletgraph/` paketi (funder resolver [creatorfill RPC deseni] + worker + saf BuildGraph). Funder'lar arka planda yakalanıp `wallet_funders` tablosuna persist edilir (Option A); `GET /api/wallet-graph` DB'den kümeleri kurar (derece-eşiği + CEX-allowlist filtresi). Saf-DB okuma, deterministik graph — ML yok.

**Tech Stack:** Go 1.24 (chi, database/sql, goose, solana-go raw JSON-RPC), TypeScript/Next.js. Test: Go `testing`+`-race`, frontend vitest.

**Spec:** `docs/superpowers/specs/2026-08-24-sentinel-backend-scoring-2e-1-wallet-graph-funding-design.md`

## Global Constraints

- **Saf-DB okuma yolu** — `getWalletGraph` yalnız DB'den okur; funder yakalama arka plan worker'da (canlı endpoint RPC yapmaz). Option A.
- **Yeni key/ücret YOK** — funder resolver mevcut `SOLANA_RPC_URL`/Helius'u `preferRPC(SolanaRPCURL, rpcURL)` ile kullanır + paylaşılan limiter.
- **Dürüst boş** — küme yoksa `{nodes:[],edges:[]}` (sahte node ASLA). Funder bulunamazsa `funder=""` + `resolved_ts=now` (yeniden-denemez).
- **Eşikler kod/config-sabiti** — `WALLET_GRAPH_MIN_CLUSTER`(2), `WALLET_GRAPH_MAX_DEGREE`(50); CEX allowlist Go const.
- **Frontend kontratı korunur** — `GraphNode{id,type,label,address?,riskLevel,balanceSol?,firstSeen,lastSeen}`, `GraphEdge{id,source,target,type}`, `WalletGraph{nodes,edges}` `apps/web/lib/api/types.ts` ile birebir.
- **`shares_funder` üretilmez** — funding_wallet hub node + `funded` edge'leri kümeyi temsil eder (YAGNI). Edge tipleri 2e-1: `funded`, `created`.
- **RiskLevel string'leri** frontend `scoreToLevel` (format.ts) ile birebir: ≤24 critical / ≤49 high / ≤69 medium / ≤84 good / else strong. (Go `store.scoreToLevel` — 2d'de eklendi — reuse.)
- **Fake/Postgres parity** — her yeni store metodu iki tarafta aynı semantikle + parity testi.
- **Migration idempotent** — `CREATE TABLE/INDEX IF NOT EXISTS`; goose up/down.
- **Guard non-corrupting** — funder heuristiği canlı tx şekline bağlı; bulunamazsa boş, pipeline bozulmaz (1a/2a/2b-1 deseni).

---

### Task 1: Migration 0012 + wallet_funders store (FunderTargets + SetFunder)

**Files:**
- Create: `apps/api-go/internal/store/migrations/0012_create_wallet_funders.sql`
- Modify: `apps/api-go/internal/store/tokens.go` (WalletFunderStore metotları + tipler + interface)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake impl + parity)
- Test: `apps/api-go/internal/store/wallet_funders_test.go` (yeni)

**Interfaces:**
- Produces:
  - `type FunderTarget struct{ Wallet string }`
  - `FunderTargets(ctx, limit int) ([]FunderTarget, error)` — çözülmemiş creator'lar.
  - `SetFunder(ctx, wallet, funder string, resolvedTs int64) error` — wallet_funders upsert.

- [ ] **Step 1: Migration**

`0012_create_wallet_funders.sql`:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS wallet_funders (
    wallet      TEXT PRIMARY KEY,
    funder      TEXT   NOT NULL DEFAULT '',
    resolved_ts BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wallet_funders_funder ON wallet_funders (funder) WHERE funder <> '';

-- +goose Down
DROP TABLE IF EXISTS wallet_funders;
```

- [ ] **Step 2: Failing test**

`wallet_funders_test.go`:
```go
package store

import (
	"context"
	"testing"
)

func TestFunderTargetsAndSetFunder_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	ts := fs.(TokenStore)
	// 2 creator'lı token'lar (creator dolu) — henüz funder çözülmemiş.
	mustUpsertCreatorToken(t, fs, "mintA", "creatorX")
	mustUpsertCreatorToken(t, fs, "mintB", "creatorY")

	targets, err := ts.FunderTargets(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("2 çözülmemiş creator bekleniyordu, got %d", len(targets))
	}
	// creatorX'i çöz (funder=F1) → artık target değil.
	if err := ts.SetFunder(ctx, "creatorX", "F1", 1000); err != nil {
		t.Fatal(err)
	}
	targets2, _ := ts.FunderTargets(ctx, 10)
	for _, tg := range targets2 {
		if tg.Wallet == "creatorX" {
			t.Fatal("creatorX çözüldü, target olmamalı")
		}
	}
	// not-found işareti de çözülmüş sayılır (funder="", resolved_ts>0).
	if err := ts.SetFunder(ctx, "creatorY", "", 1001); err != nil {
		t.Fatal(err)
	}
	targets3, _ := ts.FunderTargets(ctx, 10)
	if len(targets3) != 0 {
		t.Fatalf("hepsi çözüldü, 0 target bekleniyordu, got %d", len(targets3))
	}
}
```
> `mustUpsertCreatorToken` yardımcısı: fake'in `UpsertToken`'ıyla (creator arg dolu) token ekler. Fake `UpsertToken` imzasını (`internal/store/fake_ingest.go`) oku — `UpsertToken(ctx, TokenRow, firstSeenTs, creator)`. Yardımcıyı test dosyasında yaz.

- [ ] **Step 3: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/store/ -run TestFunderTargetsAndSetFunder_Fake -v`
Expected: FAIL (`FunderTargets`/`SetFunder` undefined).

- [ ] **Step 4: Tipler + interface + postgres impl**

`tokens.go` — tip:
```go
// FunderTarget, funder'ı henüz çözülmemiş bir creator cüzdanıdır (2e-1).
type FunderTarget struct{ Wallet string }
```
`TokenStore` interface'e ekle (2c/2d satırlarının altına):
```go
	// 2e-1: funder'ı çözülmemiş creator hedefleri / bulunan funder'ı persist eder.
	FunderTargets(ctx context.Context, limit int) ([]FunderTarget, error)
	SetFunder(ctx context.Context, wallet, funder string, resolvedTs int64) error
```
postgres impl (tokens.go sonuna):
```go
func (p *postgresStore) FunderTargets(ctx context.Context, limit int) ([]FunderTarget, error) {
	const q = `SELECT DISTINCT t.creator FROM tokens t
		WHERE t.creator <> ''
		  AND t.creator NOT IN (SELECT wallet FROM wallet_funders WHERE resolved_ts > 0)
		ORDER BY t.creator LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]FunderTarget, 0, limit)
	for rows.Next() {
		var f FunderTarget
		if err := rows.Scan(&f.Wallet); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (p *postgresStore) SetFunder(ctx context.Context, wallet, funder string, resolvedTs int64) error {
	const q = `INSERT INTO wallet_funders (wallet, funder, resolved_ts) VALUES ($1,$2,$3)
		ON CONFLICT (wallet) DO UPDATE SET funder=EXCLUDED.funder, resolved_ts=EXCLUDED.resolved_ts`
	_, err := p.db.ExecContext(ctx, q, wallet, funder, resolvedTs)
	return err
}
```

- [ ] **Step 5: Fake impl (parity)**

`fake_ingest.go` — fake store'a `walletFunders map[string]struct{funder string; resolvedTs int64}` alanı ekle (yoksa). `FunderTargets`: fake token'lardaki distinct non-empty creator'lardan, `walletFunders[creator].resolvedTs>0` OLMAYANLAR. `SetFunder`: map'e yaz. Postgres semantiğini birebir: resolved_ts>0 → çözülmüş (funder="" dahil).

- [ ] **Step 6: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/store/ -run TestFunderTargetsAndSetFunder_Fake -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/store/migrations/0012_create_wallet_funders.sql apps/api-go/internal/store/tokens.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/wallet_funders_test.go
git commit -m "feat(2e-1): migration 0012 wallet_funders + FunderTargets/SetFunder (fake+postgres parity)"
```

---

### Task 2: Funder resolver (en eski inbound SOL → funder)

**Files:**
- Create: `apps/api-go/internal/walletgraph/resolver.go`
- Test: `apps/api-go/internal/walletgraph/resolver_test.go`

**Interfaces:**
- Produces:
  - `type FunderResolver interface { ResolveFunder(ctx, wallet string) (funder string, found bool, err error) }`
  - `func NewFunderResolver(rpcURL string, maxSigPages int, opts ...ResolverOption) *HeliusFunderResolver`
  - `type ResolverOption func(*HeliusFunderResolver)`; `WithLimiter(Limiter) ResolverOption`
  - `type Limiter interface { Wait(ctx context.Context) error }` (creatorfill ile aynı; burada yeniden tanımla — paket bağımsız)

- [ ] **Step 1: Failing test** (fake sigTx — ağsız)

`resolver_test.go`:
```go
package walletgraph

import (
	"context"
	"testing"
)

// fakeSigTx, funderSigTx'i taklit eder (ağsız).
type fakeSigTx struct {
	pages     [][]string // newest-first sig sayfaları
	transfers map[string]string // sig → source (destination=wallet olan transfer'ın kaynağı; "" yoksa)
}

func (f *fakeSigTx) listSignatures(_ context.Context, _ string, before string, _ int) ([]string, error) {
	// before boşsa ilk sayfa; testte tek sayfa yeter.
	if before == "" && len(f.pages) > 0 {
		return f.pages[0], nil
	}
	return nil, nil
}
func (f *fakeSigTx) transferSource(_ context.Context, sig, _ string) (string, bool, error) {
	s, ok := f.transfers[sig]
	return s, ok && s != "", nil
}

func TestResolveFunder_OldestInboundTransfer(t *testing.T) {
	// sayfa newest-first: [sig_new, sig_old]; en eski = sig_old; onun kaynağı F1.
	r := newResolverWith(&fakeSigTx{
		pages:     [][]string{{"sig_new", "sig_old"}},
		transfers: map[string]string{"sig_old": "F1"},
	})
	funder, found, err := r.ResolveFunder(context.Background(), "creatorX")
	if err != nil || !found || funder != "F1" {
		t.Fatalf("funder=F1 found bekleniyordu, got %q found=%v err=%v", funder, found, err)
	}
}

func TestResolveFunder_NoTransfer_NotFound(t *testing.T) {
	r := newResolverWith(&fakeSigTx{
		pages:     [][]string{{"sig_old"}},
		transfers: map[string]string{}, // transfer yok
	})
	_, found, err := r.ResolveFunder(context.Background(), "creatorX")
	if err != nil || found {
		t.Fatalf("not-found bekleniyordu, found=%v err=%v", found, err)
	}
}
```
> `newResolverWith(sigTx)` = test yardımcısı: `&HeliusFunderResolver{rpc: sigTx, maxSigPages: 3, pageLimit: 1000}` döndürür (constructor'ı atlayıp fake enjekte eder).

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/walletgraph/ -v`
Expected: FAIL (paket/tipler yok).

- [ ] **Step 3: Implement resolver**

`resolver.go` (creatorfill `HeliusCreatorResolver` + `authorities.go` raw-POST jsonParsed desenlerinin birleşimi):
```go
// Package walletgraph, creator-funding kümeleme (bundler tespiti) için funder yakalama +
// graph kurma sağlar. Funder = bir cüzdana ilk SOL gönderen (en eski inbound transfer).
package walletgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Limiter interface{ Wait(ctx context.Context) error }

type FunderResolver interface {
	ResolveFunder(ctx context.Context, wallet string) (funder string, found bool, err error)
}

// funderSigTx, resolver'ın RPC ihtiyacını soyutlar (DIP; test için fake).
type funderSigTx interface {
	listSignatures(ctx context.Context, acct, before string, limit int) ([]string, error)
	// transferSource, sig'in tx'inde destination=wallet olan İLK system transfer'ının source'unu döndürür.
	transferSource(ctx context.Context, sig, wallet string) (source string, found bool, err error)
}

type HeliusFunderResolver struct {
	rpc         funderSigTx
	maxSigPages int
	pageLimit   int
	limiter     Limiter
}

type ResolverOption func(*HeliusFunderResolver)

func WithLimiter(l Limiter) ResolverOption {
	return func(r *HeliusFunderResolver) { r.limiter = l }
}

func NewFunderResolver(rpcURL string, maxSigPages int, opts ...ResolverOption) *HeliusFunderResolver {
	if maxSigPages <= 0 {
		maxSigPages = 3
	}
	r := &HeliusFunderResolver{rpc: &httpSigTx{rpcURL: rpcURL, http: &http.Client{Timeout: 12 * time.Second}}, maxSigPages: maxSigPages, pageLimit: 1000}
	for _, o := range opts {
		o(r)
	}
	return r
}

// ResolveFunder, wallet'ın EN ESKİ imzasını bulur, o tx'te wallet'a gelen ilk SOL transfer'ının
// kaynağını (funder) döndürür. Transfer yoksa / cap'e takılırsa found=false (dürüst not-found).
func (r *HeliusFunderResolver) ResolveFunder(ctx context.Context, wallet string) (string, bool, error) {
	before := ""
	oldest := ""
	reachedEnd := false // DÜZELTME 2026-08-24 (Task 2 review Critical): cap→not-found guard.
	for page := 0; page < r.maxSigPages; page++ {
		if r.limiter != nil {
			if err := r.limiter.Wait(ctx); err != nil {
				return "", false, err
			}
		}
		sigs, err := r.rpc.listSignatures(ctx, wallet, before, r.pageLimit)
		if err != nil {
			return "", false, err
		}
		if len(sigs) == 0 {
			reachedEnd = true
			break
		}
		oldest = sigs[len(sigs)-1] // newest-first → son = en eski
		before = oldest
		if len(sigs) < r.pageLimit {
			reachedEnd = true
			break
		}
	}
	if !reachedEnd {
		// cap'e takıldık → gerçek en-eski imza doğrulanamadı → dürüst not-found (yanlış funder atfetme).
		return "", false, nil
	}
	if oldest == "" {
		return "", false, nil
	}
	if r.limiter != nil {
		if err := r.limiter.Wait(ctx); err != nil {
			return "", false, err
		}
	}
	return r.rpc.transferSource(ctx, oldest, wallet)
}

// --- httpSigTx: raw JSON-RPC (authorities.go deseni) ---
type httpSigTx struct {
	rpcURL string
	http   *http.Client
}

func (h *httpSigTx) listSignatures(ctx context.Context, acct, before string, limit int) ([]string, error) {
	params := []any{acct, map[string]any{"limit": limit}}
	if before != "" {
		params[1].(map[string]any)["before"] = before
	}
	var r struct {
		Result []struct {
			Signature string `json:"signature"`
		} `json:"result"`
		Error *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := h.call(ctx, "getSignaturesForAddress", params, &r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, fmt.Errorf("getSignaturesForAddress: %s", r.Error.Message)
	}
	out := make([]string, 0, len(r.Result))
	for _, s := range r.Result {
		out = append(out, s.Signature)
	}
	return out, nil
}

func (h *httpSigTx) transferSource(ctx context.Context, sig, wallet string) (string, bool, error) {
	params := []any{sig, map[string]any{"encoding": "jsonParsed", "maxSupportedTransactionVersion": 0}}
	var r struct {
		Result *struct {
			Transaction struct {
				Message struct {
					Instructions []parsedIx `json:"instructions"`
				} `json:"message"`
			} `json:"transaction"`
			Meta *struct {
				InnerInstructions []struct {
					Instructions []parsedIx `json:"instructions"`
				} `json:"innerInstructions"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := h.call(ctx, "getTransaction", params, &r); err != nil {
		return "", false, err
	}
	if r.Error != nil {
		return "", false, fmt.Errorf("getTransaction: %s", r.Error.Message)
	}
	if r.Result == nil {
		return "", false, nil
	}
	if src, ok := scanTransfers(r.Result.Transaction.Message.Instructions, wallet); ok {
		return src, true, nil
	}
	if r.Result.Meta != nil {
		for _, inner := range r.Result.Meta.InnerInstructions {
			if src, ok := scanTransfers(inner.Instructions, wallet); ok {
				return src, true, nil
			}
		}
	}
	return "", false, nil
}

// parsedIx, jsonParsed system transfer instruction'ının gereken alanları.
type parsedIx struct {
	Program string `json:"program"`
	Parsed  struct {
		Type string `json:"type"`
		Info struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		} `json:"info"`
	} `json:"parsed"`
}

// scanTransfers, destination==wallet olan ilk system transfer'ın source'unu döndürür.
func scanTransfers(ixs []parsedIx, wallet string) (string, bool) {
	for _, ix := range ixs {
		if ix.Program != "system" {
			continue
		}
		if ix.Parsed.Type != "transfer" && ix.Parsed.Type != "transferChecked" {
			continue
		}
		if ix.Parsed.Info.Destination == wallet && ix.Parsed.Info.Source != "" && ix.Parsed.Info.Source != wallet {
			return ix.Parsed.Info.Source, true
		}
	}
	return "", false
}

func (h *httpSigTx) call(ctx context.Context, method string, params any, out any) error {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "1", "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: status %d", method, res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

var _ FunderResolver = (*HeliusFunderResolver)(nil)
```
> Test yardımcısı `newResolverWith` `resolver_test.go` içinde: `func newResolverWith(rpc funderSigTx) *HeliusFunderResolver { return &HeliusFunderResolver{rpc: rpc, maxSigPages: 3, pageLimit: 1000} }`.

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/walletgraph/ -v`
Expected: PASS (2 test).

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/walletgraph/resolver.go apps/api-go/internal/walletgraph/resolver_test.go
git commit -m "feat(2e-1): funder resolver (en eski inbound SOL, jsonParsed transfer parse)"
```

---

### Task 3: Funder worker (creatorfill deseni)

**Files:**
- Create: `apps/api-go/internal/walletgraph/worker.go`
- Test: `apps/api-go/internal/walletgraph/worker_test.go`

**Interfaces:**
- Consumes: `FunderResolver` (Task 2); `store.FunderTarget`/`FunderTargets`/`SetFunder` (Task 1).
- Produces: `type FunderStore interface { FunderTargets(...); SetFunder(...) }`; `WorkerDeps`; `NewWorker`; `(*Worker).Run`.

- [ ] **Step 1: Failing test** (creatorfill worker_test deseni)

`worker_test.go`:
```go
package walletgraph

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeFunderStore struct {
	targets []store.FunderTarget
	set     map[string]string
}

func (f *fakeFunderStore) FunderTargets(_ context.Context, _ int) ([]store.FunderTarget, error) {
	return f.targets, nil
}
func (f *fakeFunderStore) SetFunder(_ context.Context, wallet, funder string, _ int64) error {
	if f.set == nil {
		f.set = map[string]string{}
	}
	f.set[wallet] = funder
	return nil
}

type stubResolver struct{ m map[string]string; fail string }

func (s stubResolver) ResolveFunder(_ context.Context, w string) (string, bool, error) {
	if w == s.fail {
		return "", false, errors.New("rpc boom")
	}
	f, ok := s.m[w]
	return f, ok, nil
}

func TestWorker_ResolvesAndStamps_IsolatesError(t *testing.T) {
	fs := &fakeFunderStore{targets: []store.FunderTarget{{Wallet: "cA"}, {Wallet: "cB"}, {Wallet: "cErr"}}}
	res := stubResolver{m: map[string]string{"cA": "F1", "cB": ""}, fail: "cErr"}
	w := NewWorker(WorkerDeps{Store: fs, Resolver: res, Limit: 10,
		Now: func() int64 { return 1000 }, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err := w.resolveOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// cA → F1 (bulundu), cB → "" (not-found ama damgalanır), cErr → RPC hata → damgalanmaz.
	if fs.set["cA"] != "F1" {
		t.Fatalf("cA funder F1 bekleniyordu, got %q", fs.set["cA"])
	}
	if _, ok := fs.set["cB"]; !ok {
		t.Fatal("cB not-found olsa da damgalanmalı")
	}
	if _, ok := fs.set["cErr"]; ok {
		t.Fatal("cErr RPC hatası → damgalanmamalı (sonraki tick retry)")
	}
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/walletgraph/ -run TestWorker -v`
Expected: FAIL.

- [ ] **Step 3: Implement** (creatorfill `worker.go` birebir uyarlaması)

`worker.go`:
```go
package walletgraph

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type FunderStore interface {
	FunderTargets(ctx context.Context, limit int) ([]store.FunderTarget, error)
	SetFunder(ctx context.Context, wallet, funder string, resolvedTs int64) error
}

type WorkerDeps struct {
	Store    FunderStore
	Resolver FunderResolver
	Interval time.Duration
	Limit    int
	Now      func() int64
	Logger   *slog.Logger
}

type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Interval <= 0 {
		d.Interval = 60 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 40
	}
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	if err := w.resolveOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("funder resolve", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.resolveOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("funder resolve", "err", err)
			}
		}
	}
}

func (w *Worker) resolveOnce(ctx context.Context) error {
	targets, err := w.d.Store.FunderTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		funder, _, err := w.d.Resolver.ResolveFunder(ctx, tg.Wallet)
		if err != nil {
			w.d.Logger.Warn("resolve funder", "wallet", tg.Wallet, "err", err)
			continue // RPC hatası → damgalama, sonraki tick tekrar dener.
		}
		// bulundu ya da bulunamadı: damgala (sonsuz retry yok; boş funder de "çözüldü").
		if err := w.d.Store.SetFunder(ctx, tg.Wallet, funder, now); err != nil {
			w.d.Logger.Warn("set funder", "wallet", tg.Wallet, "err", err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/walletgraph/ -race -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/walletgraph/worker.go apps/api-go/internal/walletgraph/worker_test.go
git commit -m "feat(2e-1): funder worker (creatorfill deseni, kısmi-hata izole)"
```

---

### Task 4: WalletGraphClusters store sorgusu

**Files:**
- Modify: `apps/api-go/internal/store/tokens.go` (WalletGraphClusters + ClusterRow + interface)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake parity)
- Test: `apps/api-go/internal/store/wallet_funders_test.go` (ekle)

**Interfaces:**
- Produces:
  - `type ClusterRow struct { Funder, Creator, Mint, Symbol string; SafetyScore, ReputationScore float64; FirstSeenTs int64 }`
  - `WalletGraphClusters(ctx, minCluster, maxDegree int) ([]ClusterRow, error)`

- [ ] **Step 1: Failing test**

`wallet_funders_test.go`'ye ekle:
```go
func TestWalletGraphClusters_DegreeFilter_Fake(t *testing.T) {
	ctx := context.Background()
	fs := NewFakeTokenStore()
	ts := fs.(TokenStore)
	// F1 → 2 creator (cA,cB) → küme (degree 2). F2 → 1 creator (cC) → küme değil (degree<2).
	mustUpsertCreatorToken(t, fs, "mA", "cA")
	mustUpsertCreatorToken(t, fs, "mB", "cB")
	mustUpsertCreatorToken(t, fs, "mC", "cC")
	_ = ts.SetFunder(ctx, "cA", "F1", 1000)
	_ = ts.SetFunder(ctx, "cB", "F1", 1000)
	_ = ts.SetFunder(ctx, "cC", "F2", 1000)

	rows, err := ts.WalletGraphClusters(ctx, 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	funders := map[string]bool{}
	for _, r := range rows {
		funders[r.Funder] = true
	}
	if !funders["F1"] {
		t.Fatal("F1 (degree 2) küme olmalı")
	}
	if funders["F2"] {
		t.Fatal("F2 (degree 1) küme olmamalı")
	}
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/store/ -run TestWalletGraphClusters -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

`tokens.go` — tip + interface `WalletGraphClusters(ctx, minCluster, maxDegree int) ([]ClusterRow, error)` + postgres (CTE: önce qualifying funder'lar, sonra detay):
```go
// ClusterRow, bir bundler-küme kenarıdır: funder→creator→token + skorlar.
type ClusterRow struct {
	Funder, Creator, Mint, Symbol string
	SafetyScore, ReputationScore  float64
	FirstSeenTs                   int64
}

func (p *postgresStore) WalletGraphClusters(ctx context.Context, minCluster, maxDegree int) ([]ClusterRow, error) {
	const q = `
	WITH qualifying AS (
		SELECT wf.funder
		FROM tokens t JOIN wallet_funders wf ON wf.wallet = t.creator
		WHERE wf.funder <> '' AND t.creator <> ''
		GROUP BY wf.funder
		HAVING COUNT(DISTINCT t.creator) >= $1 AND COUNT(DISTINCT t.creator) <= $2
	)
	SELECT wf.funder, t.creator, t.mint, t.symbol, t.safety_score,
		COALESCE(c.reputation_score,0), t.first_seen_ts
	FROM tokens t
	JOIN wallet_funders wf ON wf.wallet = t.creator
	JOIN qualifying q ON q.funder = wf.funder
	LEFT JOIN creators c ON c.address = t.creator
	WHERE t.creator <> ''
	ORDER BY wf.funder, t.first_seen_ts DESC`
	rows, err := p.db.QueryContext(ctx, q, minCluster, maxDegree)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClusterRow{}
	for rows.Next() {
		var r ClusterRow
		if err := rows.Scan(&r.Funder, &r.Creator, &r.Mint, &r.Symbol, &r.SafetyScore, &r.ReputationScore, &r.FirstSeenTs); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
```
Fake impl: fake token'lar + walletFunders + creators map üzerinden aynı semantik — funder başına distinct creator say, `minCluster≤degree≤maxDegree` olanların tüm (funder,creator,mint,symbol,safety,reputation,firstSeen) satırlarını döndür.

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/store/ -run TestWalletGraphClusters -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/store/tokens.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/wallet_funders_test.go
git commit -m "feat(2e-1): WalletGraphClusters store agrega (degree HAVING filtresi, fake+postgres parity)"
```

---

### Task 5: CEX allowlist + saf BuildGraph

**Files:**
- Create: `apps/api-go/internal/walletgraph/cex.go`
- Create: `apps/api-go/internal/walletgraph/graph.go`
- Modify: `apps/api-go/internal/store/tokens.go` (graph JSON tipleri: GraphNode/GraphEdge/WalletGraphResult)
- Test: `apps/api-go/internal/walletgraph/graph_test.go`

**Interfaces:**
- Consumes: `store.ClusterRow` (Task 4); `store.scoreToLevel` (2d — pakete özel, `store` içinde; graph.go kendi kopyasını KULLANMAZ → `store` export etmiyorsa graph.go kendi `scoreToLevel`'ını tanımlar, format.ts parity — bkz Not).
- Produces:
  - `store.GraphNode`/`store.GraphEdge`/`store.WalletGraphResult` (JSON şekilleri).
  - `func BuildGraph(rows []store.ClusterRow) store.WalletGraphResult` (CEX dışlama içinde).
  - `func IsCEX(addr string) bool` / `cexLabel(addr string) string`.

- [ ] **Step 1: Failing test**

`graph_test.go`:
```go
package walletgraph

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestBuildGraph_ClusterNodesAndEdges(t *testing.T) {
	rows := []store.ClusterRow{
		{Funder: "F1", Creator: "cA", Mint: "mA", Symbol: "AAA", SafetyScore: 80, ReputationScore: 40, FirstSeenTs: 1000},
		{Funder: "F1", Creator: "cB", Mint: "mB", Symbol: "BBB", SafetyScore: 20, ReputationScore: 10, FirstSeenTs: 1000},
	}
	g := BuildGraph(rows)
	// node'lar: 1 funding_wallet (F1) + 2 creator_wallet + 2 token = 5
	if len(g.Nodes) != 5 {
		t.Fatalf("5 node bekleniyordu, got %d", len(g.Nodes))
	}
	// edge'ler: 2 funded (F1→cA, F1→cB) + 2 created (cA→mA, cB→mB) = 4
	if len(g.Edges) != 4 {
		t.Fatalf("4 edge bekleniyordu, got %d", len(g.Edges))
	}
	// funding_wallet node tipi + created/funded edge tipleri doğru
	var fund int
	for _, n := range g.Nodes {
		if n.Type == "funding_wallet" {
			fund++
		}
	}
	if fund != 1 {
		t.Fatalf("1 funding_wallet bekleniyordu, got %d", fund)
	}
}

func TestBuildGraph_ExcludesCEX(t *testing.T) {
	// F1 bilinen bir CEX ise küme dışlanır → boş graph.
	cex := knownCEXSample() // test yardımcısı: cex.go'daki set'ten bir adres
	rows := []store.ClusterRow{
		{Funder: cex, Creator: "cA", Mint: "mA", Symbol: "A", FirstSeenTs: 1},
		{Funder: cex, Creator: "cB", Mint: "mB", Symbol: "B", FirstSeenTs: 1},
	}
	g := BuildGraph(rows)
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Fatalf("CEX funder dışlanmalı → boş graph, got %d node", len(g.Nodes))
	}
}

func TestBuildGraph_Empty(t *testing.T) {
	g := BuildGraph(nil)
	if g.Nodes == nil || g.Edges == nil {
		t.Fatal("boş graph nil değil, boş slice olmalı (JSON [])")
	}
}
```
> `knownCEXSample()` = `cex.go`'daki ilk adresi döndüren test yardımcısı (graph_test.go içinde tanımla).

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/walletgraph/ -run TestBuildGraph -v`
Expected: FAIL.

- [ ] **Step 3: Implement store JSON tipleri**

`tokens.go` (RadarPoint yanına):
```go
// GraphNode/GraphEdge/WalletGraphResult, frontend WalletGraph (types.ts) ile birebir JSON şekilleridir.
type GraphNode struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Label     string  `json:"label"`
	Address   string  `json:"address,omitempty"`
	RiskLevel string  `json:"riskLevel"`
	FirstSeen string  `json:"firstSeen"`
	LastSeen  string  `json:"lastSeen"`
}
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}
type WalletGraphResult struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
```
> `balanceSol` alanı 2e-1'de üretilmez → struct'a KONMAZ (frontend'de opsiyonel). `scoreToLevel` `store` paketinde zaten var (2d) ama unexported; graph.go `store` dışında → kendi `scoreToLevel`'ını tanımlar (aynı eşikler, format.ts parity). Bu kabul edilebilir küçük tekrar (paket sınırı); alternatif export etmek 2d'yi değiştirir — YAGNI.

- [ ] **Step 4: Implement cex.go + graph.go**

`cex.go`:
```go
package walletgraph

// knownCEX, bilinen büyük Solana CEX hot-wallet'larıdır (küçük allowlist; bunları fonlayan
// "küme" bundler DEĞİL, borsa çekimidir → dışlanır). Genişletme 2e-1 kapsamı dışı (followup).
var knownCEX = map[string]string{
	// örnek yer tutucular — GERÇEK adreslerle doldur (Binance/Coinbase/Kraken/OKX/Bybit hot wallets):
	// "5tzFkiKscXHK5ZXCGbXZxdw7gTjjD1mBwuoFbhUvuAi9": "Binance",
}

func IsCEX(addr string) bool { _, ok := knownCEX[addr]; return ok }
func cexLabel(addr string) string { return knownCEX[addr] }
```
> **IMPORTANT (implementer):** `knownCEX`'i en az 3-5 gerçek, doğrulanmış Solana CEX hot-wallet adresiyle doldur (Binance/Coinbase/Kraken deposit veya withdrawal cüzdanları — Solscan/Arkham'da "labeled" olanlar). Adres bulunamıyor/doğrulanamıyorsa BOŞ bırakma yerine DONE_WITH_CONCERNS ile rapor et; boş set = CEX-filtresi etkisiz (yalnız derece-eşiği çalışır, spec §2.3 kısmen). Derece-eşiği (MaxDegree, Task 6/7) zaten ana koruma; allowlist ikincil.

`graph.go`:
```go
package walletgraph

import (
	"fmt"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// BuildGraph, küme satırlarından wallet graph kurar (saf; ağsız/DB'siz). CEX funder'ları dışlar.
// funding_wallet hub + funded/created edge'leri kümeyi temsil eder (shares_funder YAGNI).
func BuildGraph(rows []store.ClusterRow) store.WalletGraphResult {
	nodes := map[string]store.GraphNode{}
	edges := map[string]store.GraphEdge{}
	// funder başına distinct creator sayısı (risk için).
	degree := map[string]map[string]struct{}{}
	for _, r := range rows {
		if IsCEX(r.Funder) {
			continue // borsa çekimi, bundler değil.
		}
		if degree[r.Funder] == nil {
			degree[r.Funder] = map[string]struct{}{}
		}
		degree[r.Funder][r.Creator] = struct{}{}
	}
	for _, r := range rows {
		if IsCEX(r.Funder) {
			continue
		}
		ts := rfc3339(r.FirstSeenTs)
		fundID, walID, tokID := "fund:"+r.Funder, "wal:"+r.Creator, "tok:"+r.Mint
		deg := len(degree[r.Funder])
		nodes[fundID] = store.GraphNode{ID: fundID, Type: "funding_wallet", Label: shortAddr(r.Funder),
			Address: r.Funder, RiskLevel: scoreToLevel(float64(100 - min(deg*20, 100))), FirstSeen: ts, LastSeen: ts}
		nodes[walID] = store.GraphNode{ID: walID, Type: "creator_wallet", Label: shortAddr(r.Creator),
			Address: r.Creator, RiskLevel: scoreToLevel(r.ReputationScore), FirstSeen: ts, LastSeen: ts}
		nodes[tokID] = store.GraphNode{ID: tokID, Type: "token", Label: r.Symbol,
			Address: r.Mint, RiskLevel: scoreToLevel(r.SafetyScore), FirstSeen: ts, LastSeen: ts}
		fE := "e:funded:" + r.Funder + ":" + r.Creator
		edges[fE] = store.GraphEdge{ID: fE, Source: fundID, Target: walID, Type: "funded"}
		cE := "e:created:" + r.Creator + ":" + r.Mint
		edges[cE] = store.GraphEdge{ID: cE, Source: walID, Target: tokID, Type: "created"}
	}
	return store.WalletGraphResult{Nodes: mapNodes(nodes), Edges: mapEdges(edges)}
}

func rfc3339(ts int64) string {
	if ts <= 0 {
		return "—"
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
func shortAddr(a string) string {
	if len(a) <= 10 {
		return a
	}
	return a[:4] + "…" + a[len(a)-4:]
}
func min(a, b int) int { if a < b { return a }; return b }

// scoreToLevel, frontend format.ts scoreToLevel ile birebir (store.scoreToLevel unexported → paket-yerel kopya).
func scoreToLevel(score float64) string {
	switch {
	case score <= 24:
		return "critical"
	case score <= 49:
		return "high"
	case score <= 69:
		return "medium"
	case score <= 84:
		return "good"
	default:
		return "strong"
	}
}

func mapNodes(m map[string]store.GraphNode) []store.GraphNode {
	out := make([]store.GraphNode, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
func mapEdges(m map[string]store.GraphEdge) []store.GraphEdge {
	out := make([]store.GraphEdge, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
```
> Go 1.21+ `min` yerleşiktir; çakışırsa yerel `min`'i kaldır. `mapNodes`/`mapEdges` deterministik sıra gerektirmez (frontend id'ye göre çizer); test sayıya bakar.

- [ ] **Step 5: Run test — PASS**

Run: `cd apps/api-go && go test ./internal/walletgraph/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/walletgraph/cex.go apps/api-go/internal/walletgraph/graph.go apps/api-go/internal/walletgraph/graph_test.go apps/api-go/internal/store/tokens.go
git commit -m "feat(2e-1): CEX allowlist + saf BuildGraph (funding_wallet hub, funded/created)"
```

---

### Task 6: API handler + config + main wiring

**Files:**
- Create: `apps/api-go/internal/api/wallet_graph.go`
- Modify: `apps/api-go/internal/api/router.go` (route + RouterDeps)
- Create: `apps/api-go/internal/api/wallet_graph_test.go`
- Modify: `apps/api-go/internal/config/config.go` + `config_test.go`
- Modify: `apps/api-go/cmd/server/main.go` (funder worker + handler wiring)
- Modify: `apps/api-go/internal/store/tokens.go` (`WalletGraphClusters` TokenStore interface'e — Task 4'te eklendiyse atla)

**Interfaces:**
- Consumes: `store.WalletGraphClusters` (Task 4); `walletgraph.BuildGraph` (Task 5); `walletgraph.NewWorker`/`NewFunderResolver` (Task 2/3).
- Produces: `walletGraphHandler`; RouterDeps `WalletGraphMinCluster`/`WalletGraphMaxDegree`; config `WalletGraph*`.

- [ ] **Step 1: Failing test** (`wallet_graph_test.go`)

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestWalletGraphEndpoint_EmptyIsArray(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Tokens: ts.(store.TokenStore), WalletGraphMinCluster: 2, WalletGraphMaxDegree: 50})
	req := httptest.NewRequest(http.MethodGet, "/api/wallet-graph", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var g store.WalletGraphResult
	if err := json.NewDecoder(w.Body).Decode(&g); err != nil {
		t.Fatal(err)
	}
	if g.Nodes == nil || g.Edges == nil {
		t.Fatal("boş graph nodes/edges [] olmalı, null değil")
	}
}
```

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/api-go && go test ./internal/api/ -run TestWalletGraph -v`
Expected: FAIL.

- [ ] **Step 3: Implement handler + route**

`wallet_graph.go`:
```go
package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/walletgraph"
)

func walletGraphHandler(ts store.TokenStore, minCluster, maxDegree int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := ts.WalletGraphClusters(r.Context(), minCluster, maxDegree)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "wallet graph unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, walletgraph.BuildGraph(rows))
	}
}
```
`router.go` — RouterDeps'e `WalletGraphMinCluster int`, `WalletGraphMaxDegree int` ekle; `if d.Tokens != nil` bloğuna:
```go
		mc, md := d.WalletGraphMinCluster, d.WalletGraphMaxDegree
		if mc <= 0 { mc = 2 }
		if md <= 0 { md = 50 }
		r.Get("/api/wallet-graph", walletGraphHandler(d.Tokens, mc, md))
```

- [ ] **Step 4: config + main wiring**

`config.go` Config + Load (2d deseni):
```go
	WalletGraphEnabled     bool
	FunderResolveIntervalSec int
	FunderResolveLimit     int
	WalletGraphMinCluster  int
	WalletGraphMaxDegree   int
```
```go
		WalletGraphEnabled:       getenvBool("WALLET_GRAPH_ENABLED", true),
		FunderResolveIntervalSec: getenvInt("FUNDER_RESOLVE_INTERVAL_SEC", 60),
		FunderResolveLimit:       getenvInt("FUNDER_RESOLVE_LIMIT", 40),
		WalletGraphMinCluster:    getenvInt("WALLET_GRAPH_MIN_CLUSTER", 2),
		WalletGraphMaxDegree:     getenvInt("WALLET_GRAPH_MAX_DEGREE", 50),
```
`config_test.go` default assert (2d deseni).
`main.go` — funder worker (creatorfill gibi `preferRPC` + paylaşılan limiter; RPC varsa çalışır). creatorfill worker bloğunun yanına:
```go
	// funder resolver worker (2e-1) — arka plan; creator cüzdanlarının funder'ını yakalar (bundler tespiti).
	if cfg.WalletGraphEnabled && bundle.Tokens != nil && creatorFillRPC != "" {
		fLimiter := rate.NewLimiter(rate.Limit(float64(cfg.CreatorFillRatePerMin)/60.0), cfg.CreatorFillBurst)
		fres := walletgraph.NewFunderResolver(creatorFillRPC, cfg.CreatorFillMaxSigPages, walletgraph.WithLimiter(fLimiter))
		fw := walletgraph.NewWorker(walletgraph.WorkerDeps{
			Store: bundle.Tokens, Resolver: fres,
			Interval: time.Duration(cfg.FunderResolveIntervalSec) * time.Second, Limit: cfg.FunderResolveLimit, Logger: logger,
		})
		go fw.Run(ctx)
	}
```
Router wiring: `NewRouter(RouterDeps{... WalletGraphMinCluster: cfg.WalletGraphMinCluster, WalletGraphMaxDegree: cfg.WalletGraphMaxDegree})`.
> `walletgraph` + `rate` import'ları main.go'da. `creatorFillRPC` zaten `preferRPC(cfg.SolanaRPCURL, rpcURL)` (2d'de eklendi). Funder worker ayrı limiter kullanır (kendi RPC bütçesi; creatorfill'le paylaşmak isteğe bağlı — ayrı tutmak defansif).

- [ ] **Step 5: Run tests — PASS**

Run: `cd apps/api-go && go test ./internal/api/ ./internal/config/ -v && go build ./...`
Expected: PASS + build.

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/api/wallet_graph.go apps/api-go/internal/api/router.go apps/api-go/internal/api/wallet_graph_test.go apps/api-go/internal/config/config.go apps/api-go/internal/config/config_test.go apps/api-go/cmd/server/main.go
git commit -m "feat(2e-1): /api/wallet-graph handler + config + funder worker wiring"
```

---

### Task 7: Frontend gerçek fetch + LIVE_ENDPOINTS + docs

**Files:**
- Modify: `apps/web/lib/api/http.ts` (getWalletGraph gerçek fetch)
- Modify: `apps/web/lib/api/live-endpoints.ts` (+1)
- Modify: `apps/api-go/README.md` (WALLET_GRAPH_* + /api/wallet-graph)
- Modify: `docs/superpowers/followups-frontend.md` (2e-1 ertelenenler)
- Test: `apps/web/lib/api/http.test.ts` + `index.test.ts`

**Interfaces:**
- Consumes: backend `/api/wallet-graph` (Task 6).

- [ ] **Step 1: Failing test** (`http.test.ts` — mevcut getTokens fetch deseni)

```ts
it("getWalletGraph gerçek API'den WalletGraph döndürür", async () => {
  const fake = { nodes: [{ id: "fund:F1", type: "funding_wallet", label: "F1", riskLevel: "high", firstSeen: "n", lastSeen: "n" }], edges: [] };
  vi.spyOn(global, "fetch").mockResolvedValue(new Response(JSON.stringify(fake)));
  const r = await httpApi.getWalletGraph();
  expect(r.nodes[0].id).toBe("fund:F1");
});
```
`index.test.ts`'e: `expect(getApi().getWalletGraph).toBe(httpApi.getWalletGraph)` (LIVE routing).

- [ ] **Step 2: Run test — FAIL**

Run: `cd apps/web && npx vitest run lib/api/http.test.ts`
Expected: FAIL (getWalletGraph hâlâ notReady).

- [ ] **Step 3: Implement**

`http.ts` — `getWalletGraph: notReady` yerine (mevcut `getJson` helper — bkz getTokens/getKpis):
```ts
  getWalletGraph: () => getJson<WalletGraph>("/api/wallet-graph"),
```
> `WalletGraph` type import'unu http.ts'e ekle (yoksa). `live-endpoints.ts`: Set'e `"getWalletGraph"` ekle.

- [ ] **Step 4: Run test — PASS**

Run: `cd apps/web && npx vitest run lib/api/`
Expected: PASS.

- [ ] **Step 5: README + followups**

`README.md` env tablosu: WALLET_GRAPH_ENABLED/FUNDER_RESOLVE_INTERVAL_SEC/FUNDER_RESOLVE_LIMIT/WALLET_GRAPH_MIN_CLUSTER/WALLET_GRAPH_MAX_DEGREE + endpoint `/api/wallet-graph`.
`followups-frontend.md` "Alt-proje 2 slice 2e-1" bölümü: 2e-2 (trade/holder graph, trader/smart_wallet, bought/sold, controls_authority, authority adresi yakalama), 2e-3 (ileri kümeleme/Python, çok-hop funding, smart-money), balanceSol (getBalance), shares_funder explicit edge, CEX allowlist genişletme, funder heuristiği rafine (transferChecked/aracı-cüzdan), creator verisi WS-dormant seyrekliği.

- [ ] **Step 6: Full suite + commit**

```bash
cd apps/api-go && go build ./... && go vet ./... && go test -race ./...
cd ../web && npx vitest run
```
Expected: tüm Go paketleri + frontend suite yeşil.
```bash
git add apps/web/lib/api/http.ts apps/web/lib/api/live-endpoints.ts apps/web/lib/api/http.test.ts apps/web/lib/api/index.test.ts apps/api-go/README.md docs/superpowers/followups-frontend.md
git commit -m "feat(2e-1): frontend getWalletGraph gerçek fetch + LIVE_ENDPOINTS + docs"
```

---

## Self-Review

**1. Spec coverage:**
- §3 funder heuristiği → Task 2 (resolver scanTransfers) ✓
- §4 migration 0012 wallet_funders → Task 1 ✓
- §5 küme sorgusu + graph kurma → Task 4 (WalletGraphClusters) + Task 5 (BuildGraph) ✓
- §5.5 CEX allowlist → Task 5 (cex.go) ✓
- §6.1 paket → Task 2/3/5 ✓ · §6.2 store → Task 1/4 ✓ · §6.3 API → Task 6 ✓ · §6.4 config+main → Task 6 ✓ · §6.5 frontend → Task 7 ✓
- §7 error handling → Task 3 (worker izole) + Task 6 (502) ✓
- §8 test → her task TDD ✓
- shares_funder üretilmez (Global Constraint) → Task 5 BuildGraph yalnız funded/created ✓

**2. Placeholder scan:** `cex.go` `knownCEX` başlangıçta boş yer-tutucu — Task 5 implementer'a AÇIK talimat: gerçek adreslerle doldur veya DONE_WITH_CONCERNS. Bu kasıtlı (adres doğrulama implementer'ın canlı-veri işi); derece-eşiği ana koruma olduğu için boş set bloke etmez. Fake parity impl'leri (Task 1/4 Step 5) prose-tarif — mevcut fake_ingest.go yapısına bağlı, postgres tam kod verili.

**3. Type consistency:** `FunderTarget`/`FunderResolver`/`FunderStore`/`ClusterRow`/`GraphNode`/`GraphEdge`/`WalletGraphResult` task'lar arası tutarlı. `ResolveFunder(ctx,wallet)(string,bool,error)` T2→T3. `BuildGraph([]store.ClusterRow) store.WalletGraphResult` T5→T6. `WalletGraphClusters(ctx,min,max)` T4→T6. scoreToLevel graph.go paket-yerel (store unexported) — bilinçli, format.ts parity.

---

## Execution Handoff

Plan tamam. İki uygulama seçeneği aşağıda.
