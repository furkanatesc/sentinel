# Slice 2b-1 — Creator Capture + `getCreators`/`getCreator` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** pump.fun `CreateEvent` creator (dev) cüzdanını yakala, `tokens.creator` olarak persist et, ve `getCreators`/`getCreator` endpoint'lerini kimlik + `totalTokens` + token geçmişi düzeyinde gerçeğe çevir (itibar skoru/outcome/davranış nötr → 2b-2).

**Architecture:** Decoder creator'ı `Decoded` struct'ında taşır (TokenRow'a değil — frontend kontratı temiz kalır) → worker `UpsertToken`'a ayrı parametre olarak geçer → `tokens.creator` kolonu (migration 0006), boş creator gerçek olanı ezmez (`COALESCE(NULLIF(...))`). `getCreators`/`getCreator` `tokens`'tan saf SQL agrega (ayrı creators tablosu yok, YAGNI). Frontend seam üzerinden okur; UI'a dokunulmaz.

**Tech Stack:** Go (chi, database/sql, pgx/v5, goose), `mr-tron/base58`; frontend Next.js/TypeScript seam (`lib/api/`). Yeni bağımlılık YOK.

## Global Constraints

- **Go sürümü:** `go 1.24` (go.mod + CI ile eşleşir — değiştirme).
- **DIP seam:** hiçbir frontend bileşeni `lib/api/mock`'u import etmez; yalnız `getApi()` seam'inden okur. Backend'de handler somut store'a değil dar arayüze bağımlı.
- **Dürüstlük:** gerçek olmayan alanlar nötr placeholder (0 / `"medium"` / `"active"` / `"unlocked"` / boş dizi) — sahte skor DEĞİL. Nötr alanlar 2b-2'ye ait, kod yorumunda işaretli.
- **Frontend kontratı sabit:** `TokenRow`'a alan EKLENMEZ (creator ayrı taşınır). `CreatorRow`/`CreatorProfile` JSON şekli `apps/web/lib/api/types.ts` ile birebir.
- **Kalibrasyon riski deploy'da:** creator byte-offset (mint+64 = `user` alanı) canlı Helius şekliyle doğrulanır — guard sayesinde yanlış offset en kötü boş creator verir, pipeline'ı bozmaz (1a/2a hotfix deseni).
- **Nötr enum değerleri geçerli olmalı:** frontend `OUTCOME_DEFS[outcome]` / `LIQUIDITY_DEFS[liquidityStatus]` sözlük araması yapar; boş string UI'ı çökertir → `outcome="active"`, `liquidityStatus="unlocked"` (geçerli, en az yanıltıcı).
- **Test komutları:** Go `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`; frontend `cd apps/web && npm test`.

---

### Task 1: Ingest — pump.fun decoder creator yakalama

**Files:**
- Modify: `apps/api-go/internal/ingest/types.go` (Decoded struct'a `Creator` alanı)
- Modify: `apps/api-go/internal/ingest/decode_pumpfun.go` (`createEvent.creator` + parse mint+64 + `Decoded.Creator` set)
- Test: `apps/api-go/internal/ingest/decode_pumpfun_test.go` (creator assert)

**Interfaces:**
- Consumes: mevcut `LogNotification`, `buildCreateEventB64(name, symbol, uri string, mint, bonding, user [32]byte)` test helper (zaten `user` yazar).
- Produces: `Decoded.Creator string` (base58 pubkey ya da `""`); worker Task 2'de okur.

- [ ] **Step 1: Write the failing test**

`decode_pumpfun_test.go` içindeki `TestPumpFunDecode`'a, mevcut `user` baytını ayırt edici yapıp creator assert ekle. `import "github.com/mr-tron/base58"` ekle (test dosyasına).

```go
// TestPumpFunDecode başında, mint tanımının yanına:
var mint, bonding, user [32]byte
mint[0], mint[31] = 1, 9
user[0], user[31] = 7, 3 // creator (user) ayırt edici baytlar
```

Fonksiyonun sonuna (metadata_created kontrolünden sonra) ekle:

```go
	if out[0].Creator != base58.Encode(user[:]) {
		t.Fatalf("Creator = %q, want %q", out[0].Creator, base58.Encode(user[:]))
	}
```

Ayrıca creator'sız (kısa buffer) dürüst-boş davranışı için yeni test:

```go
func TestPumpFunDecodeCreatorMissingIsEmpty(t *testing.T) {
	// name/symbol/uri + yalnız mint (bonding+user YOK) → creator "" ama mint yine dolu.
	var b []byte
	b = append(b, make([]byte, 8)...) // discriminator
	putStr := func(s string) {
		var n [4]byte
		binary.LittleEndian.PutUint32(n[:], uint32(len(s)))
		b = append(b, n[:]...)
		b = append(b, []byte(s)...)
	}
	putStr("Solo")
	putStr("SOLO")
	putStr("https://x/u.json")
	var mint [32]byte
	mint[0] = 5
	b = append(b, mint[:]...) // bonding/user yok
	data := base64.StdEncoding.EncodeToString(b)
	n := LogNotification{
		Signature: "sig", Slot: 1, ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data},
	}
	out, err := NewPumpFunDecoder().Decode(context.Background(), n, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Token.Mint == "" {
		t.Fatalf("mint dolmalı: %+v", out)
	}
	if out[0].Creator != "" {
		t.Fatalf("Creator = %q, want \"\" (creator baytları yok)", out[0].Creator)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestPumpFunDecode -v`
Expected: FAIL (derleme hatası: `out[0].Creator` alanı yok).

- [ ] **Step 3: Add Creator field to Decoded**

`internal/ingest/types.go` — `Decoded` struct'ına alan ekle:

```go
// Decoded, bir log bildiriminden çıkarılan olay + (yeni/upsert) token + (varsa) creator cüzdanı.
type Decoded struct {
	Event   store.EventRow
	Token   store.TokenRow
	Creator string // pump.fun CreateEvent `user` (dev) pubkey; yoksa "" (yalnız pump.fun doldurur)
}
```

- [ ] **Step 4: Parse creator in decoder**

`internal/ingest/decode_pumpfun.go`:

`createEvent` struct'a alan ekle:

```go
type createEvent struct{ name, symbol, uri, mint, creator string }
```

`tryParseCreateAt` sonunda mint okuduktan sonra creator'ı (mint+64 = `user`) ekle. Mevcut son `return`'ü değiştir:

```go
	if p+32 > len(raw) { // en az mint pubkey'i sığmalı
		return createEvent{}, false
	}
	mint := base58.Encode(raw[p : p+32])
	// mint(32) + bondingCurve(32) sonrası user/creator(32) — yer varsa oku, yoksa dürüst boş.
	creator := ""
	if cp := p + 64; cp+32 <= len(raw) {
		creator = base58.Encode(raw[cp : cp+32])
	}
	return createEvent{name: name, symbol: symbol, uri: uri, mint: mint, creator: creator}, true
```

`Decode` içinde `Decoded` üretilen yerde `Creator` set et. Mevcut `return []Decoded{...}` bloğunu güncelle:

```go
	return []Decoded{
		{Event: mkEvent("new_mint", "new_mint", "info", "Yeni token oluşturuldu (pump.fun)"), Token: token, Creator: ev.creator},
		{Event: mkEvent("metadata_created", "metadata_created", "info", ev.name+" ("+ev.symbol+")"), Token: token, Creator: ev.creator},
	}, nil
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/api-go && go test ./internal/ingest/ -race -v`
Expected: PASS (yeni + mevcut decoder testleri).

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/ingest/types.go apps/api-go/internal/ingest/decode_pumpfun.go apps/api-go/internal/ingest/decode_pumpfun_test.go
git commit -m "feat(ingest): pump.fun CreateEvent creator (user) pubkey yakalama"
```

---

### Task 2: Store — migration 0006 + UpsertToken creator + `Creators()` liste + merge kuralı

**Files:**
- Create: `apps/api-go/internal/store/migrations/0006_add_token_creator.sql`
- Create: `apps/api-go/internal/store/creators.go` (CreatorRow struct + CreatorStore interface + postgres `Creators`)
- Modify: `apps/api-go/internal/store/tokens.go` (`UpsertToken` imzası + merge SQL)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (`fakeTok.creator` + `UpsertToken` creator + `Creators`)
- Modify: `apps/api-go/internal/ingest/worker.go:78` (creator geç)
- Modify (derleme — tümü `""`/creator arg ekler): `apps/api-go/internal/ingest/worker_test.go` (2 test-store `UpsertToken`), `apps/api-go/internal/api/events_test.go` (1 çağrı), `apps/api-go/internal/store/events_test.go` (3 çağrı), `apps/api-go/internal/store/postgres_ingest_test.go` (1 çağrı), `apps/api-go/internal/store/tokens_fake_test.go` (1 çağrı)
- Test: `apps/api-go/internal/store/creators_test.go` (agrega + merge; fake). Postgres parity varsa `postgres_ingest_test.go` deseni (DB'siz build-only).

**Interfaces:**
- Consumes: `Decoded.Creator` (Task 1).
- Produces:
  - `UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64, creator string) error` (imza değişti).
  - `type CreatorRow struct { Address string; Label string; ReputationScore float64; RiskLevel string; TotalTokens int; ActiveTokens int; RuggedTokens int; SuccessRatePct float64; RealizedPnlSol float64 }`
  - `type CreatorStore interface { Creators(ctx, limit) ([]CreatorRow, error); CreatorDetail(ctx, address) (CreatorProfile, bool, error) }` (CreatorDetail Task 3'te implemente).

- [ ] **Step 1: Write the failing test**

`internal/store/creators_test.go` (yeni):

```go
package store

import (
	"context"
	"testing"
)

func TestCreatorsAggregate(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	// AAA 2 token deploy etti, BBB 1.
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m2", Mint: "m2", Symbol: "S2"}, 90, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m3", Mint: "m3", Symbol: "S3"}, 80, "BBB")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m4", Mint: "m4", Symbol: "S4"}, 70, "") // creator boş → sayılmaz

	cs := ts.(CreatorStore)
	rows, err := cs.Creators(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("creator sayısı = %d, want 2 (boş creator hariç)", len(rows))
	}
	if rows[0].Address != "AAA" || rows[0].TotalTokens != 2 {
		t.Fatalf("row0 = %+v, want AAA/2 (en çok token önce)", rows[0])
	}
	if rows[0].RiskLevel != "medium" || rows[0].ReputationScore != 0 {
		t.Fatalf("nötr placeholder bozuk: %+v", rows[0])
	}
}

func TestUpsertTokenCreatorMergeDoesNotClobber(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	// Önce gerçek creator, sonra boş creator ile (GeckoTerminal deseni) upsert → gerçek korunur.
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "REAL")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1b"}, 100, "")
	cs := ts.(CreatorStore)
	rows, _ := cs.Creators(ctx, 10)
	if len(rows) != 1 || rows[0].Address != "REAL" || rows[0].TotalTokens != 1 {
		t.Fatalf("boş creator gerçek olanı ezmemeli: %+v", rows)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/store/ -run TestCreators -v`
Expected: FAIL (derleme: `UpsertToken` 4-arg değil, `CreatorStore`/`Creators` yok).

- [ ] **Step 3: Migration 0006**

`internal/store/migrations/0006_add_token_creator.sql`:

```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator TEXT NOT NULL DEFAULT '';
-- Agrega (GROUP BY creator) + filtre (creator<>'') için partial index.
CREATE INDEX IF NOT EXISTS idx_tokens_creator ON tokens (creator) WHERE creator <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_tokens_creator;
ALTER TABLE tokens DROP COLUMN IF EXISTS creator;
```

- [ ] **Step 4: UpsertToken imzası + merge SQL (postgres)**

`internal/store/tokens.go` — interface (satır ~75):

```go
	UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64, creator string) error
```

postgres impl (satır ~90):

```go
func (p *postgresStore) UpsertToken(ctx context.Context, t TokenRow, firstSeenTs int64, creator string) error {
	const q = `INSERT INTO tokens (mint, symbol, name, launchpad, first_seen_ts, creator)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (mint) DO UPDATE SET
			symbol = EXCLUDED.symbol, name = EXCLUDED.name, launchpad = EXCLUDED.launchpad,
			creator = COALESCE(NULLIF(EXCLUDED.creator, ''), tokens.creator)`
	_, err := p.db.ExecContext(ctx, q, t.Mint, t.Symbol, t.Name,
		"" /* launchpad: TokenRow'da yok (frontend kontratı) */, firstSeenTs, creator)
	return err
}
```

**Not:** `UpsertDiscovered` (GeckoTerminal) creator kolonuna INSERT'te DEĞMİYOR (DEFAULT '' kalır) ve ON CONFLICT SET listesinde creator YOK → mevcut gerçek creator otomatik korunur. **Değişiklik gerekmez** — merge güvenliği zaten var.

- [ ] **Step 5: Creators (postgres) + CreatorStore + CreatorRow**

`internal/store/creators.go` (yeni):

```go
package store

import "context"

// CreatorRow, frontend CreatorRow (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
// 2b-1: Address/TotalTokens gerçek; kalanlar nötr placeholder → 2b-2 (itibar skoru + outcome).
type CreatorRow struct {
	Address         string  `json:"address"`
	Label           string  `json:"label,omitempty"`
	ReputationScore float64 `json:"reputationScore"`
	RiskLevel       string  `json:"riskLevel"`
	TotalTokens     int     `json:"totalTokens"`
	ActiveTokens    int     `json:"activeTokens"`
	RuggedTokens    int     `json:"ruggedTokens"`
	SuccessRatePct  float64 `json:"successRatePct"`
	RealizedPnlSol  float64 `json:"realizedPnlSol"`
}

// CreatorStore, creator kimlik + agrega kaynağıdır (ISP: dar okuma arayüzü; DIP).
// NOT: Bu task'ta yalnız Creators var — CreatorDetail Task 3'te eklenir (OCP genişletme).
type CreatorStore interface {
	Creators(ctx context.Context, limit int) ([]CreatorRow, error)
}

func (p *postgresStore) Creators(ctx context.Context, limit int) ([]CreatorRow, error) {
	const q = `SELECT creator, COUNT(*) AS total FROM tokens
		WHERE creator <> '' GROUP BY creator
		ORDER BY total DESC, MIN(first_seen_ts) ASC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CreatorRow, 0, limit)
	for rows.Next() {
		var c CreatorRow
		if err := rows.Scan(&c.Address, &c.TotalTokens); err != nil {
			return nil, err
		}
		c.RiskLevel = "medium" // nötr placeholder → 2b-2
		out = append(out, c)
	}
	return out, rows.Err()
}
```

- [ ] **Step 6: fake store — creator alanı + UpsertToken + Creators**

`internal/store/fake_ingest.go`:

`fakeTok` struct'ına alan:

```go
	creator string
```

`UpsertToken` imza + persist:

```go
func (f *fakeTokenStore) UpsertToken(_ context.Context, t TokenRow, firstSeenTs int64, creator string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.Spark == nil {
		t.Spark = []float64{}
	}
	cur, ok := f.byID[t.ID]
	if !ok {
		f.order = append(f.order, t.ID)
		cur.safetyBreakdown = []ScoreBreakdownItem{}
		cur.safetyRisks = RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{}, Creator: []RiskItem{}}
	}
	cur.row = t
	cur.firstSeen = firstSeenTs
	if creator != "" { // boş creator mevcut gerçek olanı ezmez (postgres COALESCE parity)
		cur.creator = creator
	}
	f.byID[t.ID] = cur
	return nil
}
```

`Creators` (fake) — creators.go değil, fake_ingest.go'ya ekle (fake diğer fake metotlarla birlikte):

```go
func (f *fakeTokenStore) Creators(_ context.Context, limit int) ([]CreatorRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := map[string]int{}
	firstOrder := map[string]int{} // ilk görülme sırası (tiebreak: erken = önce)
	for i, id := range f.order {
		c := f.byID[id].creator
		if c == "" {
			continue
		}
		if _, seen := counts[c]; !seen {
			firstOrder[c] = i
		}
		counts[c]++
	}
	out := make([]CreatorRow, 0, len(counts))
	for addr, n := range counts {
		out = append(out, CreatorRow{Address: addr, TotalTokens: n, RiskLevel: "medium"})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalTokens != out[j].TotalTokens {
			return out[i].TotalTokens > out[j].TotalTokens // en çok önce
		}
		return firstOrder[out[i].Address] < firstOrder[out[j].Address] // erken görülen önce
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
```

(`sort` zaten fake_ingest.go'da import edilmiş.)

- [ ] **Step 7: worker + tüm çağrı sitelerini güncelle (derleme yeşil)**

`internal/ingest/worker.go:78`:

```go
		if err := w.d.Tokens.UpsertToken(ctx, item.Token, now, item.Creator); err != nil {
```

Test-store imzaları + çağrılar (hepsine creator arg ekle):
- `internal/ingest/worker_test.go`: iki `func (f *...) UpsertToken(_ context.Context, _ store.TokenRow, _ int64) error` → `, _ string) error`.
- `internal/api/events_test.go:16`: `ts.UpsertToken(nil, store.TokenRow{...}, 1)` → `..., 1, "")`.
- `internal/store/events_test.go:52-54,67`: `..., 0)` → `..., 0, "")` (ve satır 52-54 farklı creator'la test edilebilir ama `""` yeterli).
- `internal/store/postgres_ingest_test.go:23`: `..., time.Now().Unix())` → `..., time.Now().Unix(), "")`.
- `internal/store/tokens_fake_test.go:42`: `..., 1)` → `..., 1, "")`.

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./internal/store/ ./internal/ingest/ ./internal/api/ -race`
Expected: PASS (yeni creator testleri + tüm mevcut testler yeşil, derleme temiz).

- [ ] **Step 9: Commit**

```bash
git add apps/api-go/internal/store/ apps/api-go/internal/ingest/worker.go apps/api-go/internal/ingest/worker_test.go apps/api-go/internal/api/events_test.go
git commit -m "feat(store): tokens.creator (migration 0006) + UpsertToken creator merge + getCreators agrega"
```

---

### Task 3: Store — `CreatorDetail()` + profil struct'ları

**Files:**
- Modify: `apps/api-go/internal/store/creators.go` (CreatorProfile/Metrics/Behavior/TokenHistoryItem struct'ları + `CreatorStore`'a `CreatorDetail` + postgres impl)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake `CreatorDetail`)
- Test: `apps/api-go/internal/store/creators_test.go` (detail + not-found + history)

**Interfaces:**
- Consumes: `tokens.creator` + `market_cap_usd` (0004) + `first_seen_ts`.
- Produces:
  - `CreatorDetail(ctx, address) (CreatorProfile, bool, error)` (interface'e eklenir).
  - `CreatorProfile` (address/firstSeen/metrics.totalTokens/history gerçek; kalanı nötr).
  - Nötr yardımcı: `neutralReputation()` `ScoreDetail{Key:"creatorReputation", Breakdown:[]ScoreBreakdownItem{}}`.

- [ ] **Step 1: Write the failing test**

`creators_test.go`'ya ekle:

```go
func TestCreatorDetail(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1", Name: "Tok1"}, 1000, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m2", Mint: "m2", Symbol: "S2", Name: "Tok2"}, 900, "AAA")
	_ = ts.UpdateMarket(ctx, MarketUpdate{Mint: "m1", MarketCapUSD: 42000})

	cs := ts.(CreatorStore)
	p, ok, err := cs.CreatorDetail(ctx, "AAA")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if p.Address != "AAA" || p.Metrics.TotalTokens != 2 || len(p.History) != 2 {
		t.Fatalf("profil = %+v", p)
	}
	// History en yeni önce (first_seen 1000 > 900).
	if p.History[0].Mint != "m1" || p.History[0].CurrentMarketCap != 42000 {
		t.Fatalf("history[0] = %+v (m1/42000 bekleniyor)", p.History[0])
	}
	// Nötr placeholder'lar (2b-2) — geçerli enum değerleri.
	if p.RiskLevel != "medium" || p.Reputation.Key != "creatorReputation" || p.Reputation.Value != 0 {
		t.Fatalf("nötr reputation bozuk: %+v", p.Reputation)
	}
	if p.History[0].Outcome != "active" || p.History[0].LiquidityStatus != "unlocked" {
		t.Fatalf("nötr enum bozuk: %+v", p.History[0])
	}
	if p.History[0].RiskFlags == nil || p.Reputation.Breakdown == nil {
		t.Fatalf("diziler nil olmamalı (JSON [] için)")
	}
}

func TestCreatorDetailNotFound(t *testing.T) {
	ts := NewFakeTokenStore()
	_, ok, err := ts.(CreatorStore).CreatorDetail(context.Background(), "NOPE")
	if err != nil || ok {
		t.Fatalf("bulunmayan creator: ok=%v err=%v", ok, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/store/ -run TestCreatorDetail -v`
Expected: FAIL (derleme: `CreatorDetail` yok, `CreatorProfile` alanları yok).

- [ ] **Step 3: Profil struct'ları + nötr yardımcı + interface**

`internal/store/creators.go` — struct'ları ekle:

```go
// CreatorTokenHistoryItem — 2b-1: ID/Symbol/Mint/CreatedAt/CurrentMarketCap gerçek; kalanlar nötr → 2b-2.
type CreatorTokenHistoryItem struct {
	ID               string   `json:"id"`
	Symbol           string   `json:"symbol"`
	Mint             string   `json:"mint"`
	CreatedAt        string   `json:"createdAt"`
	PeakMarketCap    float64  `json:"peakMarketCap"`
	CurrentMarketCap float64  `json:"currentMarketCap"`
	MaxDrawdownPct   float64  `json:"maxDrawdownPct"`
	LiquidityStatus  string   `json:"liquidityStatus"`
	CreatorSellPct   float64  `json:"creatorSellPct"`
	Outcome          string   `json:"outcome"`
	RiskFlags        []string `json:"riskFlags"`
}

type CreatorBehavior struct {
	DeployFrequency      string   `json:"deployFrequency"`
	AvgFirstSellMinutes  float64  `json:"avgFirstSellMinutes"`
	RepeatedFunders      []string `json:"repeatedFunders"`
	SimilarMetadata      bool     `json:"similarMetadata"`
	SameSocial           bool     `json:"sameSocial"`
	SameLiquidityPattern bool     `json:"sameLiquidityPattern"`
}

type CreatorMetrics struct {
	TotalTokens         int     `json:"totalTokens"`
	ActiveTokens        int     `json:"activeTokens"`
	RuggedTokens        int     `json:"ruggedTokens"`
	AvgLifetimeHours    float64 `json:"avgLifetimeHours"`
	AvgPeakMarketCap    float64 `json:"avgPeakMarketCap"`
	RealizedPnlSol      float64 `json:"realizedPnlSol"`
	SuccessRatePct      float64 `json:"successRatePct"`
	AvgFirstSellMinutes float64 `json:"avgFirstSellMinutes"`
}

type CreatorProfile struct {
	Address       string                    `json:"address"`
	Label         string                    `json:"label,omitempty"`
	WalletAgeDays int                       `json:"walletAgeDays"`
	FirstSeen     string                    `json:"firstSeen"`
	Reputation    ScoreDetail               `json:"reputation"`
	RiskLevel     string                    `json:"riskLevel"`
	Metrics       CreatorMetrics            `json:"metrics"`
	History       []CreatorTokenHistoryItem `json:"history"`
	Behavior      CreatorBehavior           `json:"behavior"`
}

// neutralReputation, 2b-2'ye ait itibar skorunun nötr placeholder'ıdır (sahte skor değil).
func neutralReputation() ScoreDetail {
	return ScoreDetail{Key: "creatorReputation", Value: 0, Confidence: 0, UpdatedAt: "", Breakdown: []ScoreBreakdownItem{}}
}

// neutralBehavior, boş/false davranış (→ 2b-2).
func neutralBehavior() CreatorBehavior {
	return CreatorBehavior{RepeatedFunders: []string{}}
}
```

`CreatorStore` interface'ine `CreatorDetail` ekle (Task 2'deki interface'i genişlet — OCP):

```go
type CreatorStore interface {
	Creators(ctx context.Context, limit int) ([]CreatorRow, error)
	CreatorDetail(ctx context.Context, address string) (CreatorProfile, bool, error)
}
```

- [ ] **Step 4: postgres CreatorDetail**

`creators.go`'ya ekle (`import` bloğuna `"time"`):

```go
func (p *postgresStore) CreatorDetail(ctx context.Context, address string) (CreatorProfile, bool, error) {
	// Kimlik + agrega: bu creator'ın token sayısı + ilk görülme.
	var total int
	var firstSeen int64
	err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(MIN(first_seen_ts), 0) FROM tokens WHERE creator=$1`, address).
		Scan(&total, &firstSeen)
	if err != nil {
		return CreatorProfile{}, false, err
	}
	if total == 0 {
		return CreatorProfile{}, false, nil
	}
	// Token geçmişi (en yeni önce).
	rows, err := p.db.QueryContext(ctx,
		`SELECT mint, symbol, first_seen_ts, market_cap_usd FROM tokens
		 WHERE creator=$1 ORDER BY first_seen_ts DESC`, address)
	if err != nil {
		return CreatorProfile{}, false, err
	}
	defer rows.Close()
	history := make([]CreatorTokenHistoryItem, 0, total)
	for rows.Next() {
		var mint, symbol string
		var ts int64
		var mcap float64
		if err := rows.Scan(&mint, &symbol, &ts, &mcap); err != nil {
			return CreatorProfile{}, false, err
		}
		history = append(history, newHistoryItem(mint, symbol, ts, mcap))
	}
	if err := rows.Err(); err != nil {
		return CreatorProfile{}, false, err
	}
	return buildCreatorProfile(address, firstSeen, total, history), true, nil
}
```

`creators.go`'ya paylaşımlı yardımcılar (fake + postgres DRY):

```go
// newHistoryItem, gerçek alanları doldurur; kalanı nötr placeholder (→ 2b-2).
func newHistoryItem(mint, symbol string, firstSeenTs int64, currentMcap float64) CreatorTokenHistoryItem {
	return CreatorTokenHistoryItem{
		ID: mint, Symbol: symbol, Mint: mint,
		CreatedAt:        time.Unix(firstSeenTs, 0).UTC().Format(time.RFC3339),
		CurrentMarketCap: currentMcap,
		LiquidityStatus:  "unlocked", // nötr (geçerli enum) → 2b-2
		Outcome:          "active",   // nötr (geçerli enum) → 2b-2
		RiskFlags:        []string{},
	}
}

// buildCreatorProfile, kimlik + gerçek totalTokens + history; kalan metrik/davranış nötr (→ 2b-2).
func buildCreatorProfile(address string, firstSeenTs int64, total int, history []CreatorTokenHistoryItem) CreatorProfile {
	if history == nil {
		history = []CreatorTokenHistoryItem{}
	}
	return CreatorProfile{
		Address:    address,
		FirstSeen:  time.Unix(firstSeenTs, 0).UTC().Format(time.RFC3339),
		Reputation: neutralReputation(),
		RiskLevel:  "medium",
		Metrics:    CreatorMetrics{TotalTokens: total},
		History:    history,
		Behavior:   neutralBehavior(),
	}
}
```

- [ ] **Step 5: fake CreatorDetail**

`fake_ingest.go`'ya ekle:

```go
func (f *fakeTokenStore) CreatorDetail(_ context.Context, address string) (CreatorProfile, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	history := []CreatorTokenHistoryItem{}
	var firstSeen int64
	found := false
	// order en eski→yeni; history en yeni önce istenir → tersten gez.
	for i := len(f.order) - 1; i >= 0; i-- {
		tk := f.byID[f.order[i]]
		if tk.creator != address {
			continue
		}
		if !found || tk.firstSeen < firstSeen {
			firstSeen = tk.firstSeen
		}
		found = true
		history = append(history, newHistoryItem(tk.row.Mint, tk.row.Symbol, tk.firstSeen, tk.marketCapUSD))
	}
	if !found {
		return CreatorProfile{}, false, nil
	}
	return buildCreatorProfile(address, firstSeen, len(history), history), true, nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go test ./internal/store/ -race`
Expected: PASS (detail + not-found + Task 2 testleri).

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/store/creators.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/creators_test.go
git commit -m "feat(store): CreatorDetail — kimlik + totalTokens + token geçmişi (nötr metrik/davranış → 2b-2)"
```

---

### Task 4: API — `/api/creators` + `/api/creator/{address}` + wiring + config

**Files:**
- Create: `apps/api-go/internal/api/creators.go` (iki handler)
- Modify: `apps/api-go/internal/api/router.go` (`RouterDeps.Creators` + route'lar)
- Modify: `apps/api-go/internal/store/postgres.go` (`Bundle.Creators` = ps)
- Modify: `apps/api-go/internal/config/config.go` (`CreatorsListLimit`)
- Modify: `apps/api-go/cmd/server/main.go` (RouterDeps.Creators wiring)
- Test: `apps/api-go/internal/api/creators_test.go`

**Interfaces:**
- Consumes: `store.CreatorStore` (Task 2/3).
- Produces: `GET /api/creators` → `[]CreatorRow` (200); `GET /api/creator/{address}` → `CreatorProfile` (200) / 404 / 502.

- [ ] **Step 1: Write the failing test**

`internal/api/creators_test.go` (yeni):

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestCreatorsEndpoint(t *testing.T) {
	ts := store.NewFakeTokenStore()
	_ = ts.UpsertToken(context.Background(), store.TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	r := NewRouter(RouterDeps{Creators: ts.(store.CreatorStore), CreatorsLimit: 100})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/creators", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var out []store.CreatorRow
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Address != "AAA" || out[0].TotalTokens != 1 {
		t.Fatalf("body = %+v", out)
	}
}

func TestCreatorDetailEndpointFound(t *testing.T) {
	ts := store.NewFakeTokenStore()
	_ = ts.UpsertToken(context.Background(), store.TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	r := NewRouter(RouterDeps{Creators: ts.(store.CreatorStore), CreatorsLimit: 100})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/creator/AAA", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var p store.CreatorProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Address != "AAA" || p.Metrics.TotalTokens != 1 {
		t.Fatalf("profil = %+v", p)
	}
}

func TestCreatorDetailEndpointNotFound(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Creators: ts.(store.CreatorStore), CreatorsLimit: 100})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/creator/NOPE", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api-go && go test ./internal/api/ -run Creator -v`
Expected: FAIL (derleme: `RouterDeps.Creators`/`CreatorsLimit` yok, handler yok).

- [ ] **Step 3: Handlers**

`internal/api/creators.go` (yeni):

```go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func creatorsHandler(cs store.CreatorStore, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := cs.Creators(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "creators unavailable"})
			return
		}
		if rows == nil {
			rows = []store.CreatorRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func creatorDetailHandler(cs store.CreatorStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := chi.URLParam(r, "address")
		p, ok, err := cs.CreatorDetail(r.Context(), address)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "creator detail unavailable"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "creator not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}
```

- [ ] **Step 4: Router wiring**

`internal/api/router.go` — `RouterDeps`'e alanlar:

```go
	Creators      store.CreatorStore
	CreatorsLimit int
```

`NewRouter` içinde (Tokens bloğunun yanına):

```go
	if d.Creators != nil {
		limit := d.CreatorsLimit
		if limit <= 0 {
			limit = 100
		}
		r.Get("/api/creators", creatorsHandler(d.Creators, limit))
		r.Get("/api/creator/{address}", creatorDetailHandler(d.Creators))
	}
```

- [ ] **Step 5: Bundle + config + main wiring**

`internal/store/postgres.go` — `Bundle` struct'a `Creators CreatorStore` + `OpenPostgres` return:

```go
type Bundle struct {
	Strategies StrategyStore
	Events     EventStore
	Tokens     TokenStore
	Creators   CreatorStore
}
// ...
	return Bundle{Strategies: ps, Events: ps, Tokens: ps, Creators: ps}, db.Close, nil
```

`internal/config/config.go` — `Config` struct'a alan + `Load`'da:

```go
	CreatorsListLimit int
```
```go
		CreatorsListLimit: getenvInt("CREATORS_LIST_LIMIT", 100),
```

`cmd/server/main.go` — `RouterDeps`'e ekle:

```go
			Creators:      bundle.Creators,
			CreatorsLimit: cfg.CreatorsListLimit,
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`
Expected: PASS (API creator testleri + tüm paketler; config testleri de yeşil).

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/api/ apps/api-go/internal/store/postgres.go apps/api-go/internal/config/config.go apps/api-go/cmd/server/main.go
git commit -m "feat(api): /api/creators + /api/creator/{address} handlers + config CREATORS_LIST_LIMIT"
```

---

### Task 5: Frontend — `getCreators`/`getCreator` gerçek fetch

**Files:**
- Modify: `apps/web/lib/api/http.ts` (`getCreators`/`getCreator` fetch)
- Modify: `apps/web/lib/api/live-endpoints.ts` (`LIVE_ENDPOINTS`+2)
- Test: `apps/web/lib/api/http.test.ts` (varsa; yoksa mevcut http test dosyası deseni)

**Interfaces:**
- Consumes: backend `GET /api/creators` → `CreatorRow[]`, `GET /api/creator/{address}` → `CreatorProfile`.
- Produces: `httpApi.getCreators` / `httpApi.getCreator` gerçek; `LIVE_ENDPOINTS` üyeliği → hibrit `getApi()` bunları backend'e yönlendirir.

- [ ] **Step 1: Write the failing test**

`apps/web/lib/api/http.test.ts`'e ekle (mevcut dosya `vi.stubGlobal("fetch", ...)` + `Response` stilini + `beforeEach`'te `NEXT_PUBLIC_API_BASE_URL="https://api.test"` kullanır — birebir izle). Dosyanın en üstündeki import'a `LIVE_ENDPOINTS` ekle: `import { LIVE_ENDPOINTS } from "./live-endpoints";`.

```ts
const sampleCreators = [{
  address: "AAA", reputationScore: 0, riskLevel: "medium", totalTokens: 2,
  activeTokens: 0, ruggedTokens: 0, successRatePct: 0, realizedPnlSol: 0,
}];

it("getCreators API JSON'unu CreatorRow[]'a maple ve /api/creators çağırır", async () => {
  const fetchMock = vi.fn(async () =>
    new Response(JSON.stringify(sampleCreators), { status: 200, headers: { "content-type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
  const rows = await httpApi.getCreators();
  expect(rows).toEqual(sampleCreators);
  expect(fetchMock).toHaveBeenCalledWith(
    "https://api.test/api/creators",
    expect.objectContaining({ headers: { accept: "application/json" } }),
  );
});

const sampleProfile = {
  address: "AAA", walletAgeDays: 0, firstSeen: "2026-08-10T00:00:00Z",
  reputation: { key: "creatorReputation", value: 0, confidence: 0, updatedAt: "", breakdown: [] },
  riskLevel: "medium", metrics: { totalTokens: 2 }, history: [], behavior: { repeatedFunders: [] },
};

it("getCreator API JSON'unu CreatorProfile'e maple ve /api/creator/{address} çağırır", async () => {
  const fetchMock = vi.fn(async () =>
    new Response(JSON.stringify(sampleProfile), { status: 200, headers: { "content-type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
  const got = await httpApi.getCreator("AAA");
  expect(got).toEqual(sampleProfile);
  expect(fetchMock).toHaveBeenCalledWith(
    "https://api.test/api/creator/AAA",
    expect.objectContaining({ headers: { accept: "application/json" } }),
  );
});

it("LIVE_ENDPOINTS getCreators ve getCreator içerir", () => {
  expect(LIVE_ENDPOINTS.has("getCreators")).toBe(true);
  expect(LIVE_ENDPOINTS.has("getCreator")).toBe(true);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/web && npm test -- http`
Expected: FAIL (`getCreators` `notReady` reject eder / `LIVE_ENDPOINTS` üye değil).

- [ ] **Step 3: Implement http fetch**

`apps/web/lib/api/http.ts` — import'a tipleri ekle:

```ts
import type { StrategyRow, FeedEvent, TokenRow, TokenDetail, CreatorRow, CreatorProfile } from "./types";
```

`getCreators`/`getCreator` satırlarını değiştir:

```ts
  getCreators: () => getJson<CreatorRow[]>("/api/creators"),
  getCreator: (address: string) => getJson<CreatorProfile>(`/api/creator/${encodeURIComponent(address)}`),
```

- [ ] **Step 4: Add to LIVE_ENDPOINTS**

`apps/web/lib/api/live-endpoints.ts` — Set'e ekle:

```ts
  "getCreators",
  "getCreator",
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/web && npm test`
Expected: PASS (yeni http testleri + tüm mevcut frontend suite yeşil).

- [ ] **Step 6: Commit**

```bash
git add apps/web/lib/api/http.ts apps/web/lib/api/live-endpoints.ts apps/web/lib/api/http.test.ts
git commit -m "feat(web): getCreators/getCreator gerçek fetch + LIVE_ENDPOINTS (hibrit)"
```

---

### Task 6: Docs + yaşayan ledger + review handoff

**Files:**
- Modify: `apps/api-go/README.md` (env `CREATORS_LIST_LIMIT` + `/api/creators`,`/api/creator/{address}` endpoint'leri)
- Modify: `docs/progress.md` (Backend programı → 2b-1 kaydı)
- Modify: `docs/superpowers/followups-frontend.md` (2b-1 ertelenenler)

**Interfaces:** yok (yalnız docs).

- [ ] **Step 1: README güncelle**

`apps/api-go/README.md` — endpoint listesine `GET /api/creators` ve `GET /api/creator/{address}` ekle; env tablosuna `CREATORS_LIST_LIMIT` (default 100) ekle (mevcut env tablo desenini takip et).

- [ ] **Step 2: progress.md ledger**

`docs/progress.md` Backend programı bölümüne 2b-1 kaydı ekle: creator yakalama (migration 0006) + getCreators/getCreator canlı (kimlik+totalTokens+geçmiş); itibar skoru/outcome/davranış nötr → 2b-2; deploy'da byte-offset kalibrasyon riski.

- [ ] **Step 3: followups-frontend.md — 2b-1 ertelenenler**

Ekle (sessiz düşürme yok):
- İtibar skoru + outcome tespiti + davranış + cüzdan-yaşı + peak/drawdown + realizedPnl → 2b-2.
- `createdAt`/`firstSeen` şu an ISO 8601; frontend ham ISO gösterir (mock "1g önce" göreli formatı yerine) → **presentation-layer göreli formatlama** (WalletAddress-truncation followup'ı ile aynı sınıf).
- `activeTokens`/`ruggedTokens`/`successRatePct` nötr 0 (2b-2 outcome tespitine bağlı).
- `label` her zaman boş (creator etiketleme tanımlanmadı).
- Raydium/diğer launchpad token'ları creator'sız (yalnız pump.fun `user` yakalanır).

- [ ] **Step 4: Commit**

```bash
git add apps/api-go/README.md docs/progress.md docs/superpowers/followups-frontend.md
git commit -m "docs(scoring): 2b-1 creator capture — README + progress ledger + followups"
```

- [ ] **Step 5: Whole-branch review handoff**

`superpowers:requesting-code-review` ile tüm branch review'ı iste (opus). En yüksek-riskli kontroller:
1. **Byte-offset:** creator mint+64 (`user`) doğru mu — fixture layout `buildCreateEventB64` ile eşleşiyor; canlı kalibrasyon deploy'da.
2. **Merge kuralı:** boş creator gerçek olanı ezmiyor (postgres COALESCE + fake `if creator != ""`), UpsertDiscovered creator'a dokunmuyor.
3. **Nötr enum geçerliliği:** `outcome="active"`/`liquidityStatus="unlocked"` frontend `OUTCOME_DEFS`/`LIQUIDITY_DEFS` anahtarları (UI çökmez).
4. **JSON dizileri non-nil:** History/RiskFlags/Breakdown/RepeatedFunders `[]` serialize (nil değil).
5. **DIP/ISP:** handler `store.CreatorStore` dar arayüzüne bağımlı; frontend seam mock import etmiyor.

Review temizse → kullanıcıya merge onayı sor (DUR — push/merge kullanıcı onayı ister).

---

## Kabul Kriterleri (deploy sonrası doğrulanır)

1. Migration 0006 goose ile uygulanır (yeni env gerekmez; `CREATORS_LIST_LIMIT` opsiyonel default'lu).
2. Yeni pump.fun token'larında `tokens.creator` dolu (gerçek base58).
3. `GET /api/creators` → gerçek creator adresleri + `totalTokens` (çoklu deploy eden creator >1).
4. `GET /api/creator/{address}` → kimlik + `firstSeen` + token geçmişi (symbol/mint/createdAt/currentMarketCap gerçek); nötr alanlar dürüst placeholder.
5. Frontend `/creators` + `/creators/[address]` gerçek API'den (hibrit); UI çökmez (nötr enum'lar geçerli).
6. Byte-offset kalibrasyonu yanlışsa → hotfix (fixture+parse); guard sayesinde bozulma yok.
