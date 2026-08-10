# Slice: REST Creator Backfill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** GeckoTerminal'in keşfettiği creator'sız pump.fun token'larına, WS'e güvenmeden, create transaction'ının CreateEvent'inden REST ile creator ekle — `getCreators`/`getCreator`'ı canlıda doldur.

**Architecture:** Arka plan `creatorfill.Worker` (2a Enricher deseni) creator'sız pump.fun token hedeflerini çeker → her mint için `CreatorResolver` create tx'ini bulur (getSignaturesForAddress en eski + getTransaction logMessages) → mevcut 2b-1 decoder ile creator çıkarır → `SetCreatorBackfill` (merge; boş ezmez) + denendi-damgası. WS worker tamamlayıcı kalır.

**Tech Stack:** Go (solana-go v1.22 rpc client, database/sql, goose). Yeni bağımlılık YOK; mevcut Helius free-tier RPC (bounded).

## Global Constraints

- **Go sürümü:** `go 1.24` (değiştirme). `mr-tron/base58`, `gagliardetto/solana-go v1.22.0` mevcut.
- **Decode reuse (DRY):** creator parse'ı YENİDEN YAZMA — mevcut `internal/ingest/decode_pumpfun.go` `hasCreateInstruction`/`programDataAll`/`parseCreateEvent` (unexported, aynı `ingest` paketi) reuse edilir. Creator, CreateEvent `user` alanı (mint+64 offset, 2b-1 Task 1).
- **Merge kuralı:** boş creator mevcut gerçek'i EZMEZ (`COALESCE(NULLIF($2,''), creator)` — 2b-1 deseni). fake `if creator != ""`.
- **Hedefleme:** yalnız `launchpad='Pump.fun' AND creator=''` (GeckoTerminal Discoverer `launchpad='Pump.fun'` set eder — 2a doğrulaması). Non-pump.fun backfill YOK.
- **Bounded + dürüst:** `MaxSigPages` cap; bulunamayan/decode-fail → creator boş kalır (sahte değil) + `creator_backfill_ts` damgalanır (sonsuz retry yok). Damga HER denemede güncellenir.
- **WS worker SİLİNMEZ** (tamamlayıcı; backfill zaten-creator'lı token'ları atlar — merge idempotent).
- **Yeni key/ücret YOK** — mevcut `HELIUS_API_KEY` RPC.
- **DB round-trip + canlı RPC yalnız deploy'da** doğrulanır (yerel yok — 0/1/2 deseni); postgres testleri DATABASE_URL yoksa skip.
- **Test:** `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`. NOT: `gofmt -l` repo CRLF gürültüsü — yalnız bu diff'in gerçek format kaymasını flag et.

---

### Task 1: `CreatorFromCreateLogs` — decode reuse (ingest)

**Files:**
- Create: `apps/api-go/internal/ingest/creator_from_logs.go`
- Test: `apps/api-go/internal/ingest/creator_from_logs_test.go`

**Interfaces:**
- Consumes: mevcut `hasCreateInstruction`/`programDataAll`/`parseCreateEvent` (aynı paket).
- Produces: `func CreatorFromCreateLogs(logs []string) (creator string, ok bool)` (exported). Task 2 resolver kullanır.

- [ ] **Step 1: Write the failing test**

`internal/ingest/creator_from_logs_test.go`:

```go
package ingest

import (
	"encoding/base64"
	"testing"

	"github.com/mr-tron/base58"
)

func TestCreatorFromCreateLogs(t *testing.T) {
	// 2b-1 buildCreateEventB64 ile aynı layout: name/symbol/uri + mint + bonding + user(creator).
	var mint, bonding, user [32]byte
	mint[0] = 1
	user[0], user[31] = 7, 3
	data := buildCreateEventB64("Doge", "DOGE", "https://x/u.json", mint, bonding, user)
	logs := []string{
		"Program log: Instruction: Create",
		"Program data: " + data,
		"Program " + PumpFunProgramID + " success",
	}
	creator, ok := CreatorFromCreateLogs(logs)
	if !ok || creator != base58.Encode(user[:]) {
		t.Fatalf("creator = %q ok=%v, want %q", creator, ok, base58.Encode(user[:]))
	}
}

func TestCreatorFromCreateLogsNoCreate(t *testing.T) {
	// Create instruction yok → ok=false.
	if _, ok := CreatorFromCreateLogs([]string{"Program log: Instruction: Buy"}); ok {
		t.Fatal("create yokken ok=true olmamalı")
	}
	// Create var ama Program data çok kısa/bozuk → ok=false.
	bad := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
	if _, ok := CreatorFromCreateLogs([]string{"Program log: Instruction: Create", "Program data: " + bad}); ok {
		t.Fatal("bozuk data'da ok=true olmamalı")
	}
}
```

(Not: `buildCreateEventB64` 2b-1'de `decode_pumpfun_test.go`'da tanımlı — aynı paket, test'te erişilebilir.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestCreatorFromCreateLogs -v`
Expected: FAIL (`CreatorFromCreateLogs` yok).

- [ ] **Step 3: Implement**

`internal/ingest/creator_from_logs.go`:

```go
package ingest

// CreatorFromCreateLogs, bir transaction'ın log mesajlarından pump.fun creator'ını (CreateEvent
// `user` pubkey'i) çıkarır. WS logsSubscribe ve REST getTransaction aynı "Program data:" formatını
// verdiğinden decode mantığı (hasCreateInstruction/programDataAll/parseCreateEvent) birebir reuse edilir.
// Create yoksa ya da tanınmazsa ok=false (dürüst boş).
func CreatorFromCreateLogs(logs []string) (creator string, ok bool) {
	if !hasCreateInstruction(logs) {
		return "", false
	}
	for _, raw := range programDataAll(logs) {
		if ev, ok := parseCreateEvent(raw); ok && ev.creator != "" {
			return ev.creator, true
		}
	}
	return "", false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestCreatorFromCreateLogs -race -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/ingest/creator_from_logs.go apps/api-go/internal/ingest/creator_from_logs_test.go
git commit -m "feat(ingest): CreatorFromCreateLogs — logMessages'tan creator (decode reuse)"
```

---

### Task 2: `CreatorResolver` — mint → creator via create tx (ingest)

**Files:**
- Create: `apps/api-go/internal/ingest/creator_resolver.go`
- Test: `apps/api-go/internal/ingest/creator_resolver_test.go`

**Interfaces:**
- Consumes: `CreatorFromCreateLogs` (Task 1); solana-go rpc.
- Produces:
  - `type HeliusCreatorResolver struct{...}` + `func NewCreatorResolver(rpcURL string, maxSigPages int) *HeliusCreatorResolver`.
  - Method `ResolveCreator(ctx context.Context, mint string) (creator string, found bool, err error)`. Task 4 worker (via `creatorfill.CreatorResolver` arayüzü) kullanır.
  - İç DIP arayüzleri (test için): `sigLister` (mint'in sinyalleri, sayfalı) + `txLogGetter` (sig → logMessages).

- [ ] **Step 1: Write the failing test**

`internal/ingest/creator_resolver_test.go` (orkestrasyonu fake fetcher'larla test eder — canlı RPC gerekmez):

```go
package ingest

import (
	"context"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

type fakeSigTx struct {
	pages   [][]solana.Signature // sayfa sayfa (newest-first); son sayfanın son elemanı = en eski
	logsBy  map[solana.Signature][]string
	sigCalls int
}

func (f *fakeSigTx) listSignatures(_ context.Context, _ solana.PublicKey, before solana.Signature, _ int) ([]solana.Signature, error) {
	f.sigCalls++
	// before zero → ilk sayfa; aksi → bir sonraki sayfa (basit fake: çağrı sırasına göre).
	idx := f.sigCalls - 1
	if idx >= len(f.pages) {
		return nil, nil
	}
	return f.pages[idx], nil
}
func (f *fakeSigTx) txLogs(_ context.Context, sig solana.Signature) ([]string, error) {
	return f.logsBy[sig], nil
}

func mkCreateLogs(user [32]byte) []string {
	var mint, bonding [32]byte
	mint[0] = 9
	return []string{"Program log: Instruction: Create", "Program data: " + buildCreateEventB64("N", "S", "https://x/u.json", mint, bonding, user)}
}

func TestResolveCreatorFound(t *testing.T) {
	var user [32]byte
	user[0], user[31] = 4, 8
	oldest := solana.Signature{1}
	fx := &fakeSigTx{
		pages:  [][]solana.Signature{{solana.Signature{3}, solana.Signature{2}, oldest}}, // tek sayfa; son = en eski
		logsBy: map[solana.Signature][]string{oldest: mkCreateLogs(user)},
	}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000}
	creator, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || !found || creator != base58.Encode(user[:]) {
		t.Fatalf("creator=%q found=%v err=%v", creator, found, err)
	}
}

func TestResolveCreatorNotFoundWhenNoCreate(t *testing.T) {
	oldest := solana.Signature{1}
	fx := &fakeSigTx{
		pages:  [][]solana.Signature{{oldest}},
		logsBy: map[solana.Signature][]string{oldest: {"Program log: Instruction: Buy"}}, // create değil
	}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000}
	_, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || found {
		t.Fatalf("found=%v err=%v, want found=false", found, err)
	}
}

func TestResolveCreatorNoSignatures(t *testing.T) {
	fx := &fakeSigTx{pages: [][]solana.Signature{{}}}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000}
	_, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || found {
		t.Fatalf("found=%v err=%v, want found=false (sig yok)", found, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestResolveCreator -v`
Expected: FAIL (`HeliusCreatorResolver`/`ResolveCreator` yok).

- [ ] **Step 3: Implement resolver + Helius adapter**

`internal/ingest/creator_resolver.go`:

```go
package ingest

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// sigTxRPC, resolver'ın ihtiyaç duyduğu iki RPC çağrısını soyutlar (DIP; test için fake).
type sigTxRPC interface {
	// listSignatures, mint hesabının sinyallerini newest-first döndürür (before'dan geriye, ≤limit).
	listSignatures(ctx context.Context, acct solana.PublicKey, before solana.Signature, limit int) ([]solana.Signature, error)
	txLogs(ctx context.Context, sig solana.Signature) ([]string, error)
}

// HeliusCreatorResolver, bir mint'in create tx'inden pump.fun creator'ını çözer (REST).
type HeliusCreatorResolver struct {
	rpc         sigTxRPC
	maxSigPages int
	pageLimit   int
}

func NewCreatorResolver(rpcURL string, maxSigPages int) *HeliusCreatorResolver {
	if maxSigPages <= 0 {
		maxSigPages = 3
	}
	return &HeliusCreatorResolver{rpc: &heliusSigTx{cli: rpc.New(rpcURL)}, maxSigPages: maxSigPages, pageLimit: 1000}
}

// ResolveCreator, mint'in EN ESKİ sig'ini bulur (create tx), tx log'larından creator'ı çıkarır.
// Cap'e takılır ya da create tanınmazsa found=false (dürüst boş).
func (r *HeliusCreatorResolver) ResolveCreator(ctx context.Context, mint string) (string, bool, error) {
	acct, err := solana.PublicKeyFromBase58(mint)
	if err != nil {
		return "", false, err
	}
	var before, oldest solana.Signature
	oldestFound := false
	for page := 0; page < r.maxSigPages; page++ {
		sigs, err := r.rpc.listSignatures(ctx, acct, before, r.pageLimit)
		if err != nil {
			return "", false, err
		}
		if len(sigs) == 0 {
			break
		}
		oldest = sigs[len(sigs)-1] // newest-first → son eleman = bu sayfanın en eskisi
		oldestFound = true
		before = oldest
		if len(sigs) < r.pageLimit {
			break // en eskiye ulaşıldı (tam sayfa değil)
		}
	}
	if !oldestFound {
		return "", false, nil
	}
	logs, err := r.rpc.txLogs(ctx, oldest)
	if err != nil {
		return "", false, err
	}
	creator, ok := CreatorFromCreateLogs(logs)
	return creator, ok, nil
}

// heliusSigTx, sigTxRPC'yi solana-go rpc client ile karşılar (canlı; deploy'da doğrulanır).
type heliusSigTx struct{ cli *rpc.Client }

func (h *heliusSigTx) listSignatures(ctx context.Context, acct solana.PublicKey, before solana.Signature, limit int) ([]solana.Signature, error) {
	opts := &rpc.GetSignaturesForAddressOpts{Limit: &limit}
	if !before.IsZero() {
		opts.Before = before
	}
	res, err := h.cli.GetSignaturesForAddressWithOpts(ctx, acct, opts)
	if err != nil {
		return nil, err
	}
	out := make([]solana.Signature, 0, len(res))
	for _, s := range res {
		out = append(out, s.Signature)
	}
	return out, nil
}

func (h *heliusSigTx) txLogs(ctx context.Context, sig solana.Signature) ([]string, error) {
	maxV := uint64(0)
	res, err := h.cli.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		MaxSupportedTransactionVersion: &maxV,
	})
	if err != nil {
		return nil, err
	}
	if res == nil || res.Meta == nil {
		return nil, nil
	}
	return res.Meta.LogMessages, nil
}
```

**NOT:** `listSignatures` fake'i test'te çağrı-sırasına göre sayfa döndürür (before parametresini yok sayar) — bu, cap/en-eski mantığını test etmek için yeterli. Canlı `heliusSigTx` before ile gerçek sayfalar. `pageLimit` alanı test'te 1000; canlı da 1000.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./internal/ingest/ -race`
Expected: PASS (resolver orkestrasyon testleri + mevcut ingest testleri).

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/ingest/creator_resolver.go apps/api-go/internal/ingest/creator_resolver_test.go
git commit -m "feat(ingest): HeliusCreatorResolver — mint create tx'inden creator (getSignatures+getTransaction, cap'li)"
```

---

### Task 3: Store — migration 0008 + CreatorFillTargets/SetCreatorBackfill

**Files:**
- Create: `apps/api-go/internal/store/migrations/0008_add_creator_backfill.sql`
- Modify: `apps/api-go/internal/store/tokens.go` (CreatorFillTarget struct + TokenStore interface += 2 metot + postgres impl)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fakeTok `creatorBackfillTs` + fake impl)
- Test: `apps/api-go/internal/store/creator_backfill_test.go`

**Interfaces:**
- Produces:
  - `type CreatorFillTarget struct { Mint string }`
  - `TokenStore.CreatorFillTargets(ctx, limit) ([]CreatorFillTarget, error)` + `TokenStore.SetCreatorBackfill(ctx, mint, creator string, backfillTs int64) error`. Task 4 (via `creatorfill.CreatorFillStore`) kullanır.

- [ ] **Step 1: Write the failing test**

`internal/store/creator_backfill_test.go`:

```go
package store

import (
	"context"
	"testing"
)

func TestCreatorFillTargets(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	// pump.fun + creator boş → hedef.
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "pf", Launchpad: "Pump.fun", FirstSeenTs: 100})
	// pump.fun ama creator dolu → hedef DEĞİL.
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "done", Launchpad: "Pump.fun", FirstSeenTs: 90})
	_ = ts.SetCreatorBackfill(ctx, "done", "REALCREATOR", 5)
	// non-pump.fun → hedef DEĞİL.
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "ray", Launchpad: "Raydium", FirstSeenTs: 80})

	tgs, err := ts.CreatorFillTargets(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tgs) != 1 || tgs[0].Mint != "pf" {
		t.Fatalf("hedefler = %+v, want yalnız pf", tgs)
	}
}

func TestSetCreatorBackfillMergeAndStamp(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_, _ = ts.UpsertDiscovered(ctx, DiscoveredToken{Mint: "pf", Launchpad: "Pump.fun", FirstSeenTs: 100})
	// bulundu → creator set + damga.
	_ = ts.SetCreatorBackfill(ctx, "pf", "AAA", 7)
	// boş creator (bulunamadı) → gerçek'i EZMEZ, damga güncellenir.
	_ = ts.SetCreatorBackfill(ctx, "pf", "", 9)
	tgs, _ := ts.CreatorFillTargets(ctx, 10)
	if len(tgs) != 0 {
		t.Fatalf("pf artık creator'lı → hedef olmamalı: %+v", tgs)
	}
	// doğrulama: CreatorDetail üzerinden creator yansımalı (creators.go agrega).
	cs := ts.(CreatorStore)
	rows, _ := cs.Creators(ctx, 10)
	if len(rows) != 1 || rows[0].Address != "AAA" {
		t.Fatalf("creator merge bozuk: %+v", rows)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/store/ -run "TestCreatorFillTargets|TestSetCreatorBackfill" -v`
Expected: FAIL (metotlar yok).

- [ ] **Step 3: Migration 0008**

`internal/store/migrations/0008_add_creator_backfill.sql`:

```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator_backfill_ts BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS creator_backfill_ts;
```

- [ ] **Step 4: Store struct + interface + postgres impl**

`internal/store/tokens.go` — struct (SafetyTarget yakınına):

```go
// CreatorFillTarget, REST creator-backfill için hedef mint'tir.
type CreatorFillTarget struct{ Mint string }
```

`TokenStore` interface'ine ekle:

```go
	CreatorFillTargets(ctx context.Context, limit int) ([]CreatorFillTarget, error)
	SetCreatorBackfill(ctx context.Context, mint, creator string, backfillTs int64) error
```

postgres impl:

```go
func (p *postgresStore) CreatorFillTargets(ctx context.Context, limit int) ([]CreatorFillTarget, error) {
	const q = `SELECT mint FROM tokens WHERE launchpad='Pump.fun' AND creator=''
		ORDER BY creator_backfill_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorFillTarget, 0, limit)
	for rows.Next() {
		var t CreatorFillTarget
		if err := rows.Scan(&t.Mint); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *postgresStore) SetCreatorBackfill(ctx context.Context, mint, creator string, backfillTs int64) error {
	const q = `UPDATE tokens SET creator=COALESCE(NULLIF($2,''), creator), creator_backfill_ts=$3 WHERE mint=$1`
	_, err := p.db.ExecContext(ctx, q, mint, creator, backfillTs)
	return err
}
```

- [ ] **Step 5: Fake impl (parity)**

`internal/store/fake_ingest.go` — `fakeTok`'a alan:

```go
	creatorBackfillTs int64
```

Fake metotlar:

```go
func (f *fakeTokenStore) CreatorFillTargets(_ context.Context, limit int) ([]CreatorFillTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].creatorBackfillTs < f.byID[ids[j]].creatorBackfillTs // en eski-denenen önce
	})
	out := make([]CreatorFillTarget, 0, limit)
	for _, id := range ids {
		t := f.byID[id]
		if t.launchpad != "Pump.fun" || t.creator != "" || len(out) >= limit {
			continue
		}
		out = append(out, CreatorFillTarget{Mint: t.row.Mint})
	}
	return out, nil
}

func (f *fakeTokenStore) SetCreatorBackfill(_ context.Context, mint, creator string, backfillTs int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[mint]
	if !ok {
		return nil
	}
	if creator != "" { // boş gerçek'i ezmez (postgres COALESCE parity)
		cur.creator = creator
	}
	cur.creatorBackfillTs = backfillTs
	f.byID[mint] = cur
	return nil
}
```

**NOT:** fake `CreatorFillTarget{Mint: t.row.Mint}` — `t.row.Mint` UpsertDiscovered'da set edilir (`cur.row = TokenRow{ID: d.Mint, Mint: d.Mint,...}`). Doğrula.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./internal/store/ -race`
Expected: PASS (backfill hedef/merge testleri + mevcut store testleri; `TokenStore` büyüdü → derleme için ingest/worker_test.go test-double'larına no-op stub gerekebilir — varsa ekle).

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/store/
git commit -m "feat(store): migration 0008 creator_backfill_ts + CreatorFillTargets/SetCreatorBackfill (merge, pump.fun-only)"
```

---

### Task 4: Backfill Worker (`internal/creatorfill/`)

**Files:**
- Create: `apps/api-go/internal/creatorfill/worker.go`
- Test: `apps/api-go/internal/creatorfill/worker_test.go`

**Interfaces:**
- Consumes: `store.CreatorFillTarget` (Task 3); `HeliusCreatorResolver.ResolveCreator` (Task 2).
- Produces: `creatorfill.CreatorResolver` + `creatorfill.CreatorFillStore` arayüzleri (DIP) + `WorkerDeps`/`Worker`/`NewWorker`/`Run`. Task 5 (main) wire eder.

- [ ] **Step 1: Write the failing test**

`internal/creatorfill/worker_test.go`:

```go
package creatorfill

import (
	"context"
	"errors"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeStore struct {
	targets []store.CreatorFillTarget
	sets    []struct {
		mint, creator string
		ts            int64
	}
}

func (f *fakeStore) CreatorFillTargets(_ context.Context, _ int) ([]store.CreatorFillTarget, error) {
	return f.targets, nil
}
func (f *fakeStore) SetCreatorBackfill(_ context.Context, mint, creator string, ts int64) error {
	f.sets = append(f.sets, struct {
		mint, creator string
		ts            int64
	}{mint, creator, ts})
	return nil
}

type fakeResolver struct {
	byMint map[string]string // mint → creator ("" = bulunamadı)
	fail   string
}

func (f *fakeResolver) ResolveCreator(_ context.Context, mint string) (string, bool, error) {
	if mint == f.fail {
		return "", false, errors.New("rpc down")
	}
	c := f.byMint[mint]
	return c, c != "", nil
}

func TestWorkerFillsAndStamps(t *testing.T) {
	fs := &fakeStore{targets: []store.CreatorFillTarget{{Mint: "a"}, {Mint: "b"}}}
	fr := &fakeResolver{byMint: map[string]string{"a": "CREATOR_A", "b": ""}} // b bulunamadı
	w := NewWorker(WorkerDeps{Store: fs, Resolver: fr, Now: func() int64 { return 100 }})
	if err := w.fillOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// İkisi de damgalanmalı (a creator ile, b boş ile → sonsuz retry yok).
	if len(fs.sets) != 2 {
		t.Fatalf("set sayısı = %d, want 2", len(fs.sets))
	}
	byMint := map[string]string{}
	for _, s := range fs.sets {
		byMint[s.mint] = s.creator
		if s.ts != 100 {
			t.Fatalf("%s ts = %d, want 100", s.mint, s.ts)
		}
	}
	if byMint["a"] != "CREATOR_A" || byMint["b"] != "" {
		t.Fatalf("sets = %+v", fs.sets)
	}
}

func TestWorkerResolverErrorIsolated(t *testing.T) {
	fs := &fakeStore{targets: []store.CreatorFillTarget{{Mint: "boom"}, {Mint: "ok"}}}
	fr := &fakeResolver{byMint: map[string]string{"ok": "C"}, fail: "boom"}
	w := NewWorker(WorkerDeps{Store: fs, Resolver: fr, Now: func() int64 { return 1 }})
	if err := w.fillOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	// boom resolve hatası → atlanır (SetCreatorBackfill çağrılmaz); ok işlenir.
	if len(fs.sets) != 1 || fs.sets[0].mint != "ok" {
		t.Fatalf("sets = %+v, want yalnız ok", fs.sets)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/creatorfill/ -v`
Expected: FAIL (paket/`NewWorker` yok).

- [ ] **Step 3: Implement (safety.Worker deseni)**

`internal/creatorfill/worker.go`:

```go
package creatorfill

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// CreatorResolver, mint'ten creator çözer (DIP; ingest.HeliusCreatorResolver karşılar).
type CreatorResolver interface {
	ResolveCreator(ctx context.Context, mint string) (creator string, found bool, err error)
}

// CreatorFillStore, hedef seçimi + persist (DIP; store.TokenStore karşılar).
type CreatorFillStore interface {
	CreatorFillTargets(ctx context.Context, limit int) ([]store.CreatorFillTarget, error)
	SetCreatorBackfill(ctx context.Context, mint, creator string, backfillTs int64) error
}

type WorkerDeps struct {
	Store    CreatorFillStore
	Resolver CreatorResolver
	Interval time.Duration
	Limit    int
	Now      func() int64
	Logger   *slog.Logger
}

// Worker, creator'sız pump.fun token'ları için create tx'ten creator'ı REST ile getirir (Enricher deseni).
type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Interval <= 0 {
		d.Interval = 30 * time.Second
	}
	if d.Limit <= 0 {
		d.Limit = 20
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
	if err := w.fillOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("creator backfill", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.fillOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("creator backfill", "err", err)
			}
		}
	}
}

// fillOnce, bir döngü: hedefleri çek → her mint için creator resolve → persist + damga (kısmi hata izole).
func (w *Worker) fillOnce(ctx context.Context) error {
	targets, err := w.d.Store.CreatorFillTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		creator, _, err := w.d.Resolver.ResolveCreator(ctx, tg.Mint)
		if err != nil {
			w.d.Logger.Warn("resolve creator", "mint", tg.Mint, "err", err)
			continue // RPC hatası → atla; sonraki tick tekrar dener (damgalanmadı)
		}
		// bulundu ya da bulunamadı: her iki durumda damgala (sonsuz retry yok); boş creator gerçek'i ezmez.
		if err := w.d.Store.SetCreatorBackfill(ctx, tg.Mint, creator, now); err != nil {
			w.d.Logger.Warn("set creator backfill", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
```

**NOT:** RPC hatası (resolve err) → `continue` DAMGALAMADAN (geçici hata, sonraki tick tekrar dener). "Bulunamadı" (found=false, err=nil) → DAMGALANIR (creator boş, kalıcı-ish; sonsuz retry yok). Bu ayrım önemli: geçici RPC hatası vs gerçekten-yok.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go test ./internal/creatorfill/ -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/creatorfill/
git commit -m "feat(creatorfill): backfill worker (CreatorResolver+CreatorFillStore DIP, resolve-hata izole)"
```

---

### Task 5: Config + main wiring

**Files:**
- Modify: `apps/api-go/internal/config/config.go` (CREATORFILL_* alanları)
- Modify: `apps/api-go/cmd/server/main.go` (creatorfill worker goroutine)
- Test: `apps/api-go/internal/config/config_test.go` (CREATORFILL default'ları)

**Interfaces:**
- Consumes: `ingest.NewCreatorResolver` (Task 2); `creatorfill.NewWorker`/`WorkerDeps` (Task 4).
- Produces: config CREATORFILL_* + main goroutine.

- [ ] **Step 1: Write the failing test**

`internal/config/config_test.go`'ye ekle:

```go
func TestLoadCreatorFillDefaults(t *testing.T) {
	c := Load()
	if !c.CreatorFillEnabled || c.CreatorFillIntervalSec != 30 || c.CreatorFillLimit != 20 || c.CreatorFillMaxSigPages != 3 {
		t.Fatalf("creatorfill defaults: %+v", c)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/config/ -run TestLoadCreatorFill -v`
Expected: FAIL (alanlar yok).

- [ ] **Step 3: Config alanları**

`internal/config/config.go` — `Config` struct'a ekle:

```go
	CreatorFillEnabled     bool
	CreatorFillIntervalSec int
	CreatorFillLimit       int
	CreatorFillMaxSigPages int
```

`Load()` içine:

```go
		CreatorFillEnabled:     getenvBool("CREATORFILL_ENABLED", true),
		CreatorFillIntervalSec: getenvInt("CREATORFILL_INTERVAL_SEC", 30),
		CreatorFillLimit:       getenvInt("CREATORFILL_LIMIT", 20),
		CreatorFillMaxSigPages: getenvInt("CREATORFILL_MAX_SIG_PAGES", 3),
```

- [ ] **Step 4: main.go wiring**

`cmd/server/main.go` — outcome worker bloğunun ardına (srv'den önce) ekle:

```go
	// REST creator backfill (WS blokörü baypas) — arka plan; Helius RPC gerekli
	if cfg.CreatorFillEnabled && bundle.Tokens != nil && rpcURL != "" {
		resolver := ingest.NewCreatorResolver(rpcURL, cfg.CreatorFillMaxSigPages)
		cw := creatorfill.NewWorker(creatorfill.WorkerDeps{
			Store: bundle.Tokens, Resolver: resolver,
			Interval: time.Duration(cfg.CreatorFillIntervalSec) * time.Second, Limit: cfg.CreatorFillLimit, Logger: logger,
		})
		go cw.Run(ctx)
	} else if cfg.CreatorFillEnabled {
		logger.Warn("CREATORFILL: Helius key veya token store yok — backfill başlamayacak")
	}
```

`import` bloğuna `"github.com/furkanatesc/sentinel/apps/api-go/internal/creatorfill"` ekle (`ingest` zaten import). `bundle.Tokens` (`store.TokenStore`) `CreatorFillTargets`/`SetCreatorBackfill` içerir → `creatorfill.CreatorFillStore`'u karşılar; `ingest.NewCreatorResolver(...)` (`*HeliusCreatorResolver`) `creatorfill.CreatorResolver`'ı karşılar.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`
Expected: PASS (config default + tüm paketler; main derlenir).

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/config/ apps/api-go/cmd/server/main.go
git commit -m "feat(creatorfill): config CREATORFILL_* + main worker goroutine (Helius RPC gate)"
```

---

### Task 6: Docs + yaşayan ledger + review handoff

**Files:**
- Modify: `apps/api-go/README.md` (CREATORFILL_* env)
- Modify: `docs/progress.md` (Backend programı → creator-backfill kaydı)
- Modify: `docs/superpowers/followups-frontend.md` (ilgili notlar)

**Interfaces:** yok (yalnız docs).

- [ ] **Step 1: README env**

`apps/api-go/README.md` env tablosuna: `CREATORFILL_ENABLED`(true), `CREATORFILL_INTERVAL_SEC`(30), `CREATORFILL_LIMIT`(20), `CREATORFILL_MAX_SIG_PAGES`(3) — "REST creator-backfill worker (WS blokörü baypas)".

- [ ] **Step 2: progress.md ledger**

`docs/progress.md` Backend bölümüne kayıt: WS blokörü REST-backfill ile baypas edildi — GeckoTerminal-keşif creator'sız pump.fun token'larına create tx'ten (getSignatures+getTransaction+decoder reuse) creator; migration 0008; WS worker tamamlayıcı kaldı; yeni key/ücret yok; deploy'da RPC bütçesi + create-tx-bulma kalibrasyonu.

- [ ] **Step 3: followups-frontend.md**

Ekle: creator artık GeckoTerminal token'larından backfill ediliyor (WS'e bağımlı değil) → 2b-1/2b-2a creator alanları + 2b-2b reputation canlı; non-pump.fun creator hâlâ yok (yalnız pump.fun CreateEvent); cap'e takılan/decode-fail token'lar creator='' + damgalı (nadir retry); RPC free-tier limiti gözlenirse CREATORFILL_* ile kısılır; DAS getAsset creator yolu (ucuz ama pump.fun doldurmaz) elendi.

- [ ] **Step 4: Commit**

```bash
git add apps/api-go/README.md docs/progress.md docs/superpowers/followups-frontend.md
git commit -m "docs(ingest): REST creator-backfill — README + progress ledger + followups"
```

- [ ] **Step 5: Whole-branch review handoff**

`superpowers:requesting-code-review` ile tüm branch review'ı (opus). En yüksek-riskli kontroller:
1. **En-eski-sig mantığı:** newest-first sayfalama → son eleman = en eski; `len(sigs) < pageLimit` → en eskiye ulaşıldı; cap davranışı (en eskiye ulaşılamazsa found=false, corruption yok).
2. **Decode reuse:** `CreatorFromCreateLogs` mevcut `parseCreateEvent` reuse (WS ile aynı creator); create yoksa/bozuksa ok=false.
3. **Merge + damga:** `SetCreatorBackfill` boş creator gerçek'i ezmez (COALESCE); resolve-hata (RPC) DAMGALAMAZ (retry) vs bulunamadı DAMGALAR (sonsuz retry yok) ayrımı worker'da doğru.
4. **Hedefleme:** yalnız `launchpad='Pump.fun' AND creator=''`; fake↔postgres parity.
5. **DIP/izolasyon:** resolver/store dar arayüzler; WS worker silinmedi; `TokenRow` değişmedi.

Review temizse → kullanıcıya merge onayı sor (DUR — merge/deploy kullanıcı onayı).

---

## Kabul Kriterleri (deploy sonrası doğrulanır)

1. Migration 0008 goose ile (yeni zorunlu env yok).
2. Backfill worker creator'sız pump.fun token'ları için create tx bulup creator'ı gerçek base58 yazar.
3. `/api/creators` canlıda dolar (GeckoTerminal-keşif pump.fun creator'ları); `/api/creator/{addr}` gerçek geçmiş.
4. Bulunamayan token'lar creator='' + damgalı → sonsuz retry yok, log sınırlı.
5. WS worker hâlâ çalışır (patlama olursa real-time; backfill atlar — merge idempotent).
6. RPC bütçesi makul; limit gözlenirse CREATORFILL_* ile kısılır (kod değişmez).
