# SENTINEL Backend Alt-proje 1 (Solana Ingestion) — Slice 1a Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Go servisine her-zaman-açık bir Helius ingestion worker + frontend WebSocket hub ekleyerek gerçek-zaman yeni-token olaylarını Postgres'e yaz ve bağlı frontend'lere push et; `getEvents`/`getTokens` gerçek (DB), `subscribeEvents`/`subscribeTokens` gerçek WebSocket olsun.

**Architecture:** Helius WS (`logsSubscribe`, program başına bir abonelik) → **ingestion worker** (goroutine, backoff reconnect, in-memory dedup) → **decoder registry** (OCP: `LaunchpadDecoder`) → normalize `FeedEvent`+`TokenRow` → (a) Postgres insert/upsert, (b) **WS hub** broadcast → frontend `/ws` → mevcut `useLiveEvents`/`useLiveTokens` cache patch. İlk yükte `getEvents`/`getTokens` DB'den son-N okur (RSC prefetch + client). Katmanlar SRP ile ayrık; worker somut değil interface'lere (`LaunchpadDecoder`, `TxFetcher`, `MetadataFetcher`, `EventStore`, `TokenStore`) bağımlı (DIP).

**Tech Stack:** Go 1.23, chi/v5, pgx/v5 (stdlib driver + `database/sql`), goose/v3 (embed migration), **yeni:** `github.com/gagliardetto/solana-go` (rpc + rpc/ws → `logsSubscribe` + `getTransaction`), `github.com/coder/websocket` (frontend-facing hub). Frontend: Next.js/TS, TanStack Query, mevcut hibrit `getApi()` adapter.

## Global Constraints

- **Go sürümü:** ~~`go 1.23`~~ → **`go 1.24`** (go.mod + CI `.github/workflows/api-go.yml` hizalı kalmalı). **NOT (Task 6, 2026-08-05):** `solana-go` transitif bağımlılığı Go 1.24 gerektirdiğinden go.mod + CI birlikte 1.24'e yükseltildi (kısıtın asıl amacı = go.mod/CI hizası, korundu). Deploy: Railway build'i Go 1.24 kullanır (go.mod'dan otomatik).
- **DB erişimi:** `database/sql` + pgx/v5 **stdlib** driver (`_ "github.com/jackc/pgx/v5/stdlib"`, `sql.Open("pgx", dsn)`), pgx native pool DEĞİL — Alt-proje 0 deseni.
- **Migration:** goose/v3, `//go:embed migrations/*.sql` (embed `..` üst dizini kabul etmez — yalnız `migrations/*.sql`). Yeni dosya `internal/store/migrations/0002_*.sql`, `-- +goose Up`/`-- +goose Down` bloklu.
- **JSON kontrat kilidi:** Go struct'ların JSON çıktısı frontend `apps/web/lib/api/types.ts` ile **birebir** olacak. `FeedEvent` alanları: `id,type,symbol,mint,launchpad,dex,liquidity,creatorScore,riskLevel,tokenAgeSeconds,volume5m,holderGrowthPct,severity,detail,time,ts,watchlisted`. `TokenRow` alanları: `id,name,symbol,mint,ageSeconds,price,liquidity,vol5m,holders,creatorScore,safetyScore,momentum,spark,signal,watchlisted`. Her JSON key'i bir testte kilitle (Alt-proje 0 `TestStrategyRowJSONKeys` deseni).
- **Enum değerleri (frontend union'larıyla eşleşmeli):** `EventType` ∈ new_mint|metadata_created|pool_created|first_swap|liquidity_added|liquidity_removed|creator_sell|whale_buy|suspicious_cluster|score_change|strategy_signal. `RiskLevel` ∈ critical|high|medium|good|strong. `AlertSeverity` ∈ info|positive|warning|critical. `launchpad` frontend `LAUNCHPADS`=["Pump.fun","Raydium","Moonshot","Meteora"], `dex` frontend `DEXES`=["Raydium","Meteora","Orca","Jupiter"] ile uyumlu olmalı (yoksa filtre çipi eşleşmez; event yine akar).
- **Nötr/dürüst placeholder (1a):** `creatorScore=0`, `safetyScore=0`, `riskLevel="medium"`, `liquidity/volume5m/holderGrowthPct/price/vol5m/holders/momentum=0`, `spark=[]`, `signal=null`, `watchlisted=false`. Bunlar **sahte skor değil "henüz yok"** — Alt-proje 2 (scoring) doldurur. Frontend'de nötr skor "—" gösterilir.
- **SOLID (review ölçütü):** OCP decoder registry (yeni launchpad = yeni decoder + registry kaydı; worker/hub değişmez), SRP katman ayrımı, DIP interface bağımlılığı, ISP dar interface'ler. Hiçbir frontend bileşeni `lib/api/mock`'u doğrudan import etmez.
- **Auth yok:** Alt-proje 0'daki gibi public read-only; `/ws` de public.
- **Dil:** UI/hata metinleri Türkçe (mevcut desen).

## Kapsam kararları (bilinçli — sessiz düşürme YOK)

| Karar | Gerekçe | Nereye |
|---|---|---|
| Somut decoder'lar: **pump.fun** (log-only) + **Raydium CPMM** (tx-tabanlı) | pump.fun event log'u mint/name/symbol/uri/creator'ı ağsız taşır (çekirdek "yeni mint"); Raydium CPMM yeni pool'lar için tx-tabanlı yolu + OCP'yi kanıtlar; ikisi de mevcut frontend `LAUNCHPADS`/`DEXES` sözlüğünü kullanır | Bu plan Task 4-5 |
| **PumpSwap decoder ertelendi** (framework-hazır) | PumpSwap pool'ları pump.fun mezuniyeti = zaten görülmüş mint; ayrıca `sources.ts`'e "PumpSwap" eklemeyi gerektirir. Framework hazır olduğundan sonra tek decoder + tek `sources.ts` girdisiyle eklenir | Slice 1a-follow / 1b; `docs/superpowers/followups-frontend.md` + progress.md |
| **Moonshot/Meteora decoder ertelendi** | Spec'te de framework-hazır-sonra-somut | 1b+ |
| **Event tipleri 1a:** new_mint, metadata_created, pool_created | Bunlar oluşum anından güvenilir. first_swap/liquidity_added swap/addLiquidity log parse + fiyat/likidite ister (enrichment) | first_swap/liquidity_added → 1b; liquidity_removed/creator_sell/whale_buy → 1b; suspicious_cluster/score_change → Alt-proje 2; strategy_signal → Alt-proje 4 |
| `getKpis`/`getRadar`/`getToken` MOCK kalır | Slice 1b (enrichment/agregasyon) | live-endpoints'e EKLENMEZ |
| Backfill yok | Fresh DB boş başlar, event geldikçe dolar | 1a kapsamı dışı |

---

## File Structure

**Go (`apps/api-go`) — yeni/değişen:**
```
internal/store/
├─ events.go            # (yeni) EventRow struct + EventStore interface + postgres InsertEvent/RecentEvents
├─ tokens.go            # (yeni) TokenRow struct + TokenStore interface + postgres UpsertToken/RecentTokens
├─ fake_ingest.go       # (yeni) in-memory FakeEventStore + FakeTokenStore (test/DB'siz mod)
├─ migrations/0002_create_events_tokens.sql   # (yeni) events + tokens tabloları
├─ postgres.go          # (değişir) OpenPostgres artık EventStore+TokenStore da döndürür (Bundle)
└─ store.go             # (değişmez) StrategyRow/StrategyStore
internal/ingest/
├─ types.go             # (yeni) LogNotification, Decoded, TxInfo, TokenMeta normalize tipleri
├─ decoder.go           # (yeni) LaunchpadDecoder + Registry + TxFetcher + MetadataFetcher interface'leri
├─ decode_pumpfun.go    # (yeni) pump.fun CreateEvent decoder (log-only)
├─ decode_raydium.go    # (yeni) Raydium CPMM initialize decoder (tx-tabanlı)
├─ helius.go            # (yeni) Helius WS logsSubscribe client + GetTransaction + DAS getAsset adapter
├─ worker.go            # (yeni) Worker: subscribe/route/dedup/backoff/persist/broadcast
└─ testdata/            # (yeni) kayıtlı fixture'lar (pumpfun_create.json, raydium_initialize.json)
internal/ws/
└─ hub.go               # (yeni) frontend-facing WS hub (coder/websocket), topic events/tokens
internal/api/
├─ events.go            # (yeni) GET /api/events, GET /api/tokens handler'ları
├─ ws.go                # (yeni) GET /ws handler (hub'a bağlar)
└─ router.go            # (değişir) yeni route'lar + hub/store bağlama
internal/config/config.go # (değişir) HeliusAPIKey, HeliusWSURL, HeliusRPCURL, EventsWindow
cmd/server/main.go        # (değişir) worker + hub başlat, graceful shutdown
.github/workflows/api-go.yml # (değişebilir) — mevcut go test ./... yeterli
```

**Frontend (`apps/web`) — değişen:**
```
lib/api/ws.ts           # (yeni) WS transport helper (base URL türet, topic abone, unsubscribe=close)
lib/api/http.ts         # (değişir) getEvents/getTokens gerçek fetch; subscribeEvents/subscribeTokens gerçek WS
lib/api/live-endpoints.ts # (değişir) LIVE_ENDPOINTS += getEvents,getTokens,subscribeEvents,subscribeTokens
components/**/ScoreBadge (veya eşdeğeri) # (değişir) nötr skor (0) için "—" göster (dürüst placeholder)
```

**Yeni Go bağımlılıkları:** `github.com/gagliardetto/solana-go` (rpc, rpc/ws), `github.com/coder/websocket`. `go get` + `go mod tidy` ile eklenir (Task 6/8).

---

## Interfaces (özet — tasklar arası referans)

```go
// internal/ingest/types.go
type LogNotification struct {
    Signature string
    Slot      uint64
    Err       any        // nil = başarı; non-nil ise decode edilmez
    Logs      []string
    ProgramID string     // hangi aboneliğin (program) tetiklediği
}
type Decoded struct {
    Event store.EventRow
    Token store.TokenRow  // yeni/upsert edilecek token (mint zorunlu)
}

// internal/ingest/decoder.go
type TxFetcher interface {          // DIP: decoder gerekince tx çeker (pump.fun kullanmaz)
    GetTransaction(ctx context.Context, signature string) (TxInfo, error)
}
type MetadataFetcher interface {    // DIP: mint → name/symbol (DAS getAsset); başarısızsa fallback
    Metadata(ctx context.Context, mint string) (TokenMeta, error)
}
type LaunchpadDecoder interface {
    ProgramID() string                                   // abone olunacak program
    Launchpad() string                                   // "Pump.fun" | "Raydium" (LAUNCHPADS ile uyumlu)
    Decode(ctx context.Context, n LogNotification, tx TxFetcher, md MetadataFetcher) ([]Decoded, error)
}
type Registry struct { /* map[programID]LaunchpadDecoder */ }
func (r *Registry) Register(d LaunchpadDecoder)
func (r *Registry) ProgramIDs() []string
func (r *Registry) Decoder(programID string) (LaunchpadDecoder, bool)

// internal/ingest/types.go (devam)
type TxInfo struct { AccountKeys []string }              // getTransaction'dan pozisyonel hesap listesi
type TokenMeta struct { Name, Symbol, URI string }

// internal/store/events.go
type EventRow struct { /* JSON: FeedEvent birebir */ }
type EventStore interface {
    InsertEvent(ctx context.Context, e EventRow) error
    RecentEvents(ctx context.Context, limit int) ([]EventRow, error)
}
// internal/store/tokens.go
type TokenRow struct { /* JSON: frontend TokenRow birebir */ }
type TokenStore interface {
    UpsertToken(ctx context.Context, t TokenRow) error
    RecentTokens(ctx context.Context, limit int) ([]TokenRow, error)
}

// internal/ws/hub.go
type Hub struct { /* ... */ }
func NewHub() *Hub
func (h *Hub) Run(ctx context.Context)                   // broadcast döngüsü
func (h *Hub) Broadcast(topic string, payload any)       // topic: "events" | "tokens"
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) // /ws handler
```

---

## Task 1: Postgres migration 0002 + event/token satır tipleri + interface'ler + fake store'lar

**Files:**
- Create: `apps/api-go/internal/store/migrations/0002_create_events_tokens.sql`
- Create: `apps/api-go/internal/store/events.go`
- Create: `apps/api-go/internal/store/tokens.go`
- Create: `apps/api-go/internal/store/fake_ingest.go`
- Test: `apps/api-go/internal/store/events_test.go`

**Interfaces:**
- Produces: `store.EventRow`, `store.EventStore`, `store.TokenRow`, `store.TokenStore`, `store.NewFakeEventStore()`, `store.NewFakeTokenStore()`.

- [ ] **Step 1: JSON-key kilit testini yaz (FAIL bekle)**

`events_test.go`:
```go
package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestEventRowJSONKeys(t *testing.T) {
	e := EventRow{ID: "x", Type: "new_mint", Symbol: "S", Mint: "M", Launchpad: "Pump.fun",
		DEX: "", Liquidity: 0, CreatorScore: 0, RiskLevel: "medium", TokenAgeSeconds: 0,
		Volume5m: 0, HolderGrowthPct: 0, Severity: "info", Detail: "d", Time: "t", Ts: 1, Watchlisted: false}
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"id", "type", "symbol", "mint", "launchpad", "dex", "liquidity",
		"creatorScore", "riskLevel", "tokenAgeSeconds", "volume5m", "holderGrowthPct",
		"severity", "detail", "time", "ts", "watchlisted"} {
		if _, ok := m[k]; !ok {
			t.Errorf("EventRow JSON missing key %q (contract drift)", k)
		}
	}
}

func TestTokenRowJSONKeys(t *testing.T) {
	tk := TokenRow{ID: "M", Name: "n", Symbol: "s", Mint: "M", AgeSeconds: 0}
	b, _ := json.Marshal(tk)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"id", "name", "symbol", "mint", "ageSeconds", "price", "liquidity",
		"vol5m", "holders", "creatorScore", "safetyScore", "momentum", "spark", "signal", "watchlisted"} {
		if _, ok := m[k]; !ok {
			t.Errorf("TokenRow JSON missing key %q (contract drift)", k)
		}
	}
}

func TestFakeStoresRoundTrip(t *testing.T) {
	ctx := context.Background()
	es := NewFakeEventStore()
	if err := es.InsertEvent(ctx, EventRow{ID: "e1", Type: "new_mint", Mint: "M", Ts: 1}); err != nil {
		t.Fatal(err)
	}
	got, err := es.RecentEvents(ctx, 10)
	if err != nil || len(got) != 1 || got[0].ID != "e1" {
		t.Fatalf("RecentEvents = %+v, err=%v", got, err)
	}
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "M", Mint: "M", Symbol: "S"})
	_ = ts.UpsertToken(ctx, TokenRow{ID: "M", Mint: "M", Symbol: "S2"}) // upsert = tek satır
	toks, _ := ts.RecentTokens(ctx, 10)
	if len(toks) != 1 || toks[0].Symbol != "S2" {
		t.Fatalf("RecentTokens = %+v", toks)
	}
}
```

- [ ] **Step 2: Testi çalıştır — FAIL (tipler tanımsız)**

Run: `cd apps/api-go && go test ./internal/store/ -run 'EventRow|TokenRow|FakeStores' -v`
Expected: derleme hatası (EventRow/TokenRow/NewFake* yok).

- [ ] **Step 3: `events.go` yaz**

```go
package store

import (
	"context"
	"sort"
	"sync"
)

// EventRow, frontend FeedEvent (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
type EventRow struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Symbol          string  `json:"symbol"`
	Mint            string  `json:"mint"`
	Launchpad       string  `json:"launchpad"`
	DEX             string  `json:"dex"`
	Liquidity       float64 `json:"liquidity"`
	CreatorScore    float64 `json:"creatorScore"`
	RiskLevel       string  `json:"riskLevel"`
	TokenAgeSeconds int64   `json:"tokenAgeSeconds"`
	Volume5m        float64 `json:"volume5m"`
	HolderGrowthPct float64 `json:"holderGrowthPct"`
	Severity        string  `json:"severity"`
	Detail          string  `json:"detail"`
	Time            string  `json:"time"`
	Ts              int64   `json:"ts"`
	Watchlisted     bool    `json:"watchlisted"`
}

// EventStore, append-only olay kaynağıdır (DIP).
type EventStore interface {
	InsertEvent(ctx context.Context, e EventRow) error
	RecentEvents(ctx context.Context, limit int) ([]EventRow, error)
}
```

Not: `Slot`/`Signature` DB'de sütun olarak tutulur (dedup+debug) ama JSON'a çıkmaz (frontend kontratında yok) → EventRow struct'ında JSON'a giden alan yok; slot/signature'ı postgres katmanı ayrı parametreyle alır (Task 2) ya da EventRow'a `json:"-"` alan eklenir. **Karar:** EventRow'a `Signature string \`json:"-"\`` ve `Slot uint64 \`json:"-"\`` ekle (dedup+persist için taşınır, JSON'da görünmez). Yukarıdaki struct'a bu iki alanı ekle.

- [ ] **Step 4: `tokens.go` yaz**

```go
package store

import "context"

// TokenRow, frontend TokenRow (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
type TokenRow struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Mint         string    `json:"mint"`
	AgeSeconds   int64     `json:"ageSeconds"`
	Price        float64   `json:"price"`
	Liquidity    float64   `json:"liquidity"`
	Vol5m        float64   `json:"vol5m"`
	Holders      int       `json:"holders"`
	CreatorScore float64   `json:"creatorScore"`
	SafetyScore  float64   `json:"safetyScore"`
	Momentum     float64   `json:"momentum"`
	Spark        []float64 `json:"spark"`
	Signal       *string   `json:"signal"` // "buy"|"watch"|"avoid"|null
	Watchlisted  bool      `json:"watchlisted"`
}

// TokenStore, mint-PK token kaynağıdır (upsert; DIP).
type TokenStore interface {
	UpsertToken(ctx context.Context, t TokenRow) error
	RecentTokens(ctx context.Context, limit int) ([]TokenRow, error)
}
```

Not: `Spark` marshal'da `null` yerine `[]` çıksın diye persist/oku katmanında `nil` yerine `[]float64{}` ata (frontend `spark: number[]`). Fake store da boş slice döndürsün.

- [ ] **Step 5: `fake_ingest.go` yaz** (in-memory, DB'siz mod + test)

```go
package store

import (
	"context"
	"sync"
)

type fakeEventStore struct {
	mu   sync.Mutex
	rows []EventRow // en yeni sonda
}

func NewFakeEventStore() EventStore { return &fakeEventStore{} }

func (f *fakeEventStore) InsertEvent(_ context.Context, e EventRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, e)
	return nil
}

func (f *fakeEventStore) RecentEvents(_ context.Context, limit int) ([]EventRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]EventRow, 0, limit)
	for i := len(f.rows) - 1; i >= 0 && len(out) < limit; i-- { // en yeni önce
		out = append(out, f.rows[i])
	}
	return out, nil
}

type fakeTokenStore struct {
	mu    sync.Mutex
	byID  map[string]TokenRow
	order []string // ekleme sırası
}

func NewFakeTokenStore() TokenStore { return &fakeTokenStore{byID: map[string]TokenRow{}} }

func (f *fakeTokenStore) UpsertToken(_ context.Context, t TokenRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.Spark == nil {
		t.Spark = []float64{}
	}
	if _, ok := f.byID[t.ID]; !ok {
		f.order = append(f.order, t.ID)
	}
	f.byID[t.ID] = t
	return nil
}

func (f *fakeTokenStore) RecentTokens(_ context.Context, limit int) ([]TokenRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TokenRow, 0, limit)
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, f.byID[f.order[i]])
	}
	return out, nil
}
```
(`sort` import'u events.go'da kullanılmıyorsa kaldır — yalnız gerekliyse ekle.)

- [ ] **Step 6: Migration `0002_create_events_tokens.sql` yaz**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS tokens (
    mint          TEXT PRIMARY KEY,
    symbol        TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL DEFAULT '',
    launchpad     TEXT NOT NULL DEFAULT '',
    first_seen_ts BIGINT NOT NULL,
    -- enrichment/scoring (Slice 1b / Alt-proje 2) — 1a'da nötr:
    price         DOUBLE PRECISION NOT NULL DEFAULT 0,
    liquidity     DOUBLE PRECISION NOT NULL DEFAULT 0,
    vol5m         DOUBLE PRECISION NOT NULL DEFAULT 0,
    holders       INTEGER NOT NULL DEFAULT 0,
    creator_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    safety_score  DOUBLE PRECISION NOT NULL DEFAULT 0,
    momentum      DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS events (
    id                TEXT PRIMARY KEY,
    signature         TEXT NOT NULL,
    slot              BIGINT NOT NULL,
    type              TEXT NOT NULL,
    mint              TEXT NOT NULL,
    symbol            TEXT NOT NULL DEFAULT '',
    launchpad         TEXT NOT NULL DEFAULT '',
    dex               TEXT NOT NULL DEFAULT '',
    liquidity         DOUBLE PRECISION NOT NULL DEFAULT 0,
    creator_score     DOUBLE PRECISION NOT NULL DEFAULT 0,
    risk_level        TEXT NOT NULL DEFAULT 'medium',
    token_age_seconds BIGINT NOT NULL DEFAULT 0,
    volume5m          DOUBLE PRECISION NOT NULL DEFAULT 0,
    holder_growth_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
    severity          TEXT NOT NULL DEFAULT 'info',
    detail            TEXT NOT NULL DEFAULT '',
    ts                BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events (ts DESC);

-- +goose Down
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS tokens;
```

- [ ] **Step 7: Testi çalıştır — PASS**

Run: `cd apps/api-go && go test ./internal/store/ -run 'EventRow|TokenRow|FakeStores' -v`
Expected: PASS. Sonra `go build ./... && go vet ./...`.

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/internal/store/
git commit -m "feat(api-go): events/tokens store types, interfaces, fakes, migration 0002"
```

---

## Task 2: Postgres event/token store implementasyonu + bundle açılışı

**Files:**
- Modify: `apps/api-go/internal/store/events.go` (postgres InsertEvent/RecentEvents ekle)
- Modify: `apps/api-go/internal/store/tokens.go` (postgres UpsertToken/RecentTokens ekle)
- Modify: `apps/api-go/internal/store/postgres.go` (OpenPostgres → StrategyStore + EventStore + TokenStore döndür)
- Test: `apps/api-go/internal/store/postgres_ingest_test.go` (integration; `DATABASE_URL` guard'lı — mevcut `postgres_test.go` deseni)

**Interfaces:**
- Consumes: Task 1 `EventRow`/`TokenRow`/`EventStore`/`TokenStore`.
- Produces: `store.OpenPostgres` yeni imza — bkz aşağıda. `postgresStore` artık üç interface'i de uygular.

- [ ] **Step 1: Mevcut `postgres_test.go`'yu oku**, `DATABASE_URL` guard desenini birebir izle (env yoksa `t.Skip`).

- [ ] **Step 2: Integration testi yaz (FAIL bekle)** — `postgres_ingest_test.go`:
```go
package store

import (
	"context"
	"os"
	"testing"
)

func TestPostgresEventsTokensRoundTrip(t *testing.T) {
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

	tk := TokenRow{ID: "MintX", Mint: "MintX", Symbol: "TST", Name: "Test", AgeSeconds: 0, Spark: []float64{}}
	if err := b.Tokens.UpsertToken(ctx, tk); err != nil {
		t.Fatal(err)
	}
	e := EventRow{ID: "sig1-new_mint", Signature: "sig1", Slot: 5, Type: "new_mint", Mint: "MintX",
		Symbol: "TST", Launchpad: "Pump.fun", RiskLevel: "medium", Severity: "info", Ts: 1}
	if err := b.Events.InsertEvent(ctx, e); err != nil {
		t.Fatal(err)
	}
	evs, err := b.Events.RecentEvents(ctx, 10)
	if err != nil || len(evs) == 0 || evs[0].Mint != "MintX" {
		t.Fatalf("RecentEvents=%+v err=%v", evs, err)
	}
	toks, err := b.Tokens.RecentTokens(ctx, 10)
	if err != nil || len(toks) == 0 {
		t.Fatalf("RecentTokens=%+v err=%v", toks, err)
	}
}
```

- [ ] **Step 3: `OpenPostgres` imzasını Bundle'a çevir** (`postgres.go`):
```go
// Bundle, açılan Postgres bağlantısının sunduğu store'ları toplar.
type Bundle struct {
	Strategies StrategyStore
	Events     EventStore
	Tokens     TokenStore
}

// OpenPostgres, bağlantı açar, migration'ları çalıştırır, strateji seed'ini uygular
// ve store bundle'ı ile kapatma fonksiyonu döner.
func OpenPostgres(ctx context.Context, dsn string) (Bundle, func() error, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return Bundle{}, nil, fmt.Errorf("open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return Bundle{}, nil, fmt.Errorf("ping: %w", err)
	}
	if err := runMigrations(db); err != nil {
		db.Close()
		return Bundle{}, nil, fmt.Errorf("migrate: %w", err)
	}
	if err := seedStrategies(ctx, db); err != nil {
		db.Close()
		return Bundle{}, nil, fmt.Errorf("seed: %w", err)
	}
	ps := &postgresStore{db: db}
	return Bundle{Strategies: ps, Events: ps, Tokens: ps}, db.Close, nil
}
```
**Önemli:** `main.go` ve `postgres_test.go` mevcut `OpenPostgres` imzasını (`StrategyStore, func, err`) kullanıyor → bu değişiklik onları kırar. Task 2 içinde `postgres_test.go`'daki mevcut çağrıyı `b, cleanup, err := OpenPostgres(...)` + `b.Strategies.List(...)` olacak şekilde güncelle; `main.go` Task 9'da güncellenecek (şimdilik derleme için `main.go`'yu da minimal uyarlaman gerekebilir — Task 9 tam ele alır; ara derlemede `st, cleanup = pst.Strategies, cl` yeterli).

- [ ] **Step 4: `postgresStore`'a event/token metotlarını ekle** (events.go / tokens.go içine, `postgresStore` receiver):
```go
// events.go
func (p *postgresStore) InsertEvent(ctx context.Context, e EventRow) error {
	const q = `INSERT INTO events
		(id, signature, slot, type, mint, symbol, launchpad, dex, liquidity, creator_score,
		 risk_level, token_age_seconds, volume5m, holder_growth_pct, severity, detail, ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (id) DO NOTHING`
	_, err := p.db.ExecContext(ctx, q, e.ID, e.Signature, int64(e.Slot), e.Type, e.Mint, e.Symbol,
		e.Launchpad, e.DEX, e.Liquidity, e.CreatorScore, e.RiskLevel, e.TokenAgeSeconds, e.Volume5m,
		e.HolderGrowthPct, e.Severity, e.Detail, e.Ts)
	return err
}

func (p *postgresStore) RecentEvents(ctx context.Context, limit int) ([]EventRow, error) {
	const q = `SELECT id, signature, slot, type, mint, symbol, launchpad, dex, liquidity, creator_score,
		risk_level, token_age_seconds, volume5m, holder_growth_pct, severity, detail, ts
		FROM events ORDER BY ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventRow
	for rows.Next() {
		var e EventRow
		var slot int64
		if err := rows.Scan(&e.ID, &e.Signature, &slot, &e.Type, &e.Mint, &e.Symbol, &e.Launchpad,
			&e.DEX, &e.Liquidity, &e.CreatorScore, &e.RiskLevel, &e.TokenAgeSeconds, &e.Volume5m,
			&e.HolderGrowthPct, &e.Severity, &e.Detail, &e.Ts); err != nil {
			return nil, err
		}
		e.Slot = uint64(slot)
		e.Time = "" // 1a: time frontend'de ts'ten türetilir; boş bırakılır (kontratta string)
		out = append(out, e)
	}
	return out, rows.Err()
}
```
```go
// tokens.go
func (p *postgresStore) UpsertToken(ctx context.Context, t TokenRow) error {
	const q = `INSERT INTO tokens (mint, symbol, name, launchpad, first_seen_ts)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (mint) DO UPDATE SET
			symbol = EXCLUDED.symbol, name = EXCLUDED.name, launchpad = EXCLUDED.launchpad`
	_, err := p.db.ExecContext(ctx, q, t.Mint, t.Symbol, t.Name, /*launchpad*/ "", t.firstSeenTs())
	return err
}
```
Not: `first_seen_ts` TokenRow'da yok (JSON kontratında yok). Upsert çağrısını yapan worker `first_seen_ts`'i ayrı geçmeli. **Basit çözüm:** `UpsertToken(ctx, t TokenRow, firstSeenTs int64)` imzası — interface'i buna göre güncelle (Task 1 interface'ine `firstSeenTs int64` ekle) VE fake store'u da güncelle. `RecentTokens` okurken `ageSeconds = nowUnix - first_seen_ts` hesapla:
```go
func (p *postgresStore) RecentTokens(ctx context.Context, limit int) ([]TokenRow, error) {
	const q = `SELECT mint, symbol, name, first_seen_ts, price, liquidity, vol5m, holders,
		creator_score, safety_score, momentum FROM tokens ORDER BY first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now().Unix()
	var out []TokenRow
	for rows.Next() {
		var t TokenRow
		var firstSeen int64
		if err := rows.Scan(&t.Mint, &t.Symbol, &t.Name, &firstSeen, &t.Price, &t.Liquidity,
			&t.Vol5m, &t.Holders, &t.CreatorScore, &t.SafetyScore, &t.Momentum); err != nil {
			return nil, err
		}
		t.ID = t.Mint
		t.AgeSeconds = now - firstSeen
		if t.AgeSeconds < 0 {
			t.AgeSeconds = 0
		}
		t.Spark = []float64{}
		out = append(out, t)
	}
	return out, rows.Err()
}
```
→ Task 1'in `TokenStore.UpsertToken` imzasını `UpsertToken(ctx, t TokenRow, firstSeenTs int64) error` yap; fake_ingest.go ve TestFakeStoresRoundTrip'i buna göre güncelle (fake first_seen'i saklamasa da imza uyumlu olsun; test çağrısına `, 0` ekle). `import "time"` ekle.

- [ ] **Step 5: Guard'lı integration testini çalıştır**

Run: `cd apps/api-go && go test ./internal/store/ -v` (DATABASE_URL yoksa integration skip; unit'ler PASS). Sonra `go build ./... && go vet ./...`.
Expected: birim testler PASS, integration SKIP (yerelde DB yok — deploy'da doğrulanır, Alt-proje 0 deseni).

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/store/ apps/api-go/cmd/server/main.go
git commit -m "feat(api-go): postgres event/token store + OpenPostgres Bundle"
```

---

## Task 3: ingest tipleri + decoder registry + interface'ler

**Files:**
- Create: `apps/api-go/internal/ingest/types.go`
- Create: `apps/api-go/internal/ingest/decoder.go`
- Test: `apps/api-go/internal/ingest/decoder_test.go`

**Interfaces:**
- Consumes: `store.EventRow`, `store.TokenRow`.
- Produces: `ingest.LogNotification`, `ingest.Decoded`, `ingest.TxInfo`, `ingest.TokenMeta`, `ingest.TxFetcher`, `ingest.MetadataFetcher`, `ingest.LaunchpadDecoder`, `ingest.Registry` (+ `Register`/`ProgramIDs`/`Decoder`).

- [ ] **Step 1: Registry testini yaz (FAIL bekle)** — `decoder_test.go`:
```go
package ingest

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type stubDecoder struct{ pid, lp string }

func (s stubDecoder) ProgramID() string { return s.pid }
func (s stubDecoder) Launchpad() string { return s.lp }
func (s stubDecoder) Decode(_ context.Context, _ LogNotification, _ TxFetcher, _ MetadataFetcher) ([]Decoded, error) {
	return []Decoded{{Event: store.EventRow{Type: "new_mint"}}}, nil
}

func TestRegistryLookup(t *testing.T) {
	r := NewRegistry()
	r.Register(stubDecoder{pid: "P1", lp: "Pump.fun"})
	r.Register(stubDecoder{pid: "P2", lp: "Raydium"})

	ids := r.ProgramIDs()
	if len(ids) != 2 {
		t.Fatalf("ProgramIDs len=%d, want 2", len(ids))
	}
	d, ok := r.Decoder("P1")
	if !ok || d.Launchpad() != "Pump.fun" {
		t.Fatalf("Decoder(P1) = %v, %v", d, ok)
	}
	if _, ok := r.Decoder("nope"); ok {
		t.Fatal("Decoder(nope) beklenmedik şekilde bulundu")
	}
}
```

- [ ] **Step 2: Testi çalıştır — FAIL** (`cd apps/api-go && go test ./internal/ingest/ -v`).

- [ ] **Step 3: `types.go` yaz**
```go
package ingest

import "github.com/furkanatesc/sentinel/apps/api-go/internal/store"

// LogNotification, Helius logsSubscribe bildiriminin normalize halidir.
type LogNotification struct {
	Signature string
	Slot      uint64
	Err       any // nil = başarı; non-nil ise decode edilmez
	Logs      []string
	ProgramID string // hangi aboneliğin (program) tetiklediği
}

// Decoded, bir log bildiriminden çıkarılan olay + (yeni/upsert) token.
type Decoded struct {
	Event      store.EventRow
	Token      store.TokenRow
	FirstSeen  int64 // token first_seen_ts (upsert için)
}

// TxInfo, getTransaction'dan alınan pozisyonel hesap listesidir.
type TxInfo struct {
	AccountKeys []string
}

// TokenMeta, DAS getAsset'ten alınan metadata'dır.
type TokenMeta struct {
	Name, Symbol, URI string
}
```

- [ ] **Step 4: `decoder.go` yaz**
```go
package ingest

import "context"

// TxFetcher, decoder gerektiğinde ham işlemi çeker (DIP; pump.fun kullanmaz).
type TxFetcher interface {
	GetTransaction(ctx context.Context, signature string) (TxInfo, error)
}

// MetadataFetcher, mint → name/symbol (Helius DAS getAsset); başarısızlıkta çağıran fallback yapar.
type MetadataFetcher interface {
	Metadata(ctx context.Context, mint string) (TokenMeta, error)
}

// LaunchpadDecoder, bir launchpad program'ının log'larını olaylara çevirir (OCP birimi).
type LaunchpadDecoder interface {
	ProgramID() string
	Launchpad() string
	Decode(ctx context.Context, n LogNotification, tx TxFetcher, md MetadataFetcher) ([]Decoded, error)
}

// Registry, programID → decoder eşlemesidir (OCP: yeni launchpad = Register çağrısı).
type Registry struct {
	byProgram map[string]LaunchpadDecoder
}

func NewRegistry() *Registry { return &Registry{byProgram: map[string]LaunchpadDecoder{}} }

func (r *Registry) Register(d LaunchpadDecoder) { r.byProgram[d.ProgramID()] = d }

func (r *Registry) ProgramIDs() []string {
	out := make([]string, 0, len(r.byProgram))
	for id := range r.byProgram {
		out = append(out, id)
	}
	return out
}

func (r *Registry) Decoder(programID string) (LaunchpadDecoder, bool) {
	d, ok := r.byProgram[programID]
	return d, ok
}
```

- [ ] **Step 5: Testi çalıştır — PASS** + `go build ./... && go vet ./...`.

- [ ] **Step 6: Commit**
```bash
git add apps/api-go/internal/ingest/
git commit -m "feat(api-go): ingest decoder registry + interfaces (OCP/DIP)"
```

---

## Task 4: pump.fun decoder (log-only CreateEvent) — çekirdek

**Files:**
- Create: `apps/api-go/internal/ingest/decode_pumpfun.go`
- Create: `apps/api-go/internal/ingest/testdata/pumpfun_create.json` (kayıtlı fixture)
- Test: `apps/api-go/internal/ingest/decode_pumpfun_test.go`

**Referans (doğrulanmış):** pump.fun program ID `6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P`. `create` instruction log marker'ı: `Program log: Instruction: Create`. `Program data: <base64>` satırı bir Anchor `CreateEvent` taşır; layout (Chainstack `chainstacklabs/pump-fun-bot` → `learning-examples/listen-new-tokens/listen_logsubscribe.py` ile doğrulanmış): `[8-byte discriminator][name: u32 LE len + UTF-8][symbol: len+UTF-8][uri: len+UTF-8][mint: 32B pubkey][bondingCurve: 32B][user/creator: 32B]`. 32-byte pubkey'ler base58 encode edilir.

**Interfaces:**
- Consumes: Task 3 tipleri.
- Produces: `ingest.NewPumpFunDecoder() LaunchpadDecoder`, `ingest.PumpFunProgramID` const.

- [ ] **Step 1: Fixture'ı hazırla.** `testdata/pumpfun_create.json` — GERÇEK bir pump.fun create tx'inden alınmış `logsNotification.params.result.value` snapshot'ı. Kaynak (key gerekmez): Solscan'de pump.fun program'ının `create` instruction filtresi ile bir tx aç → "Program Logs" sekmesindeki `logs[]`'i kopyala; VEYA Chainstack repo örneğinden. Format:
```json
{
  "signature": "<gerçek base58 sig>",
  "slot": 300000000,
  "err": null,
  "logs": [
    "Program 6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P invoke [1]",
    "Program log: Instruction: Create",
    "Program data: <gerçek base64 CreateEvent>",
    "Program 6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P success"
  ]
}
```
**Fixture yoksa / erişilemezse (fallback, deterministik):** decoder'ın ters işlemiyle bir `Program data:` satırı ÜRET (aynı test dosyasında bir `buildCreateEventB64(name,symbol,uri,mint,bonding,user)` yardımcı fonksiyonuyla), böylece test discriminator'dan bağımsız kalır. Discriminator'ı fixture'dan al; gerçek fixture gelene kadar 8 sıfır byte kullan ve testte "discriminator atlanır, alanlar okunur" davranışını doğrula. Gerçek fixture eklendiğinde testi ona bağla. (Bu adım, spec §9'daki "fixture implementasyonda netleşir" maddesinin somut karşılığıdır — sahte veri değil, ya gerçek snapshot ya da decoder'la tutarlı üretilmiş vektör.)

- [ ] **Step 2: Decoder testini yaz (FAIL bekle)** — `decode_pumpfun_test.go`:
```go
package ingest

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"testing"
)

// buildCreateEventB64, pump.fun CreateEvent base64 satırını decoder layout'uyla üretir (test vektörü).
func buildCreateEventB64(name, symbol, uri string, mint, bonding, user [32]byte) string {
	var b []byte
	b = append(b, make([]byte, 8)...) // discriminator (test: sıfır)
	putStr := func(s string) {
		var n [4]byte
		binary.LittleEndian.PutUint32(n[:], uint32(len(s)))
		b = append(b, n[:]...)
		b = append(b, []byte(s)...)
	}
	putStr(name)
	putStr(symbol)
	putStr(uri)
	b = append(b, mint[:]...)
	b = append(b, bonding[:]...)
	b = append(b, user[:]...)
	return base64.StdEncoding.EncodeToString(b)
}

func TestPumpFunDecode(t *testing.T) {
	var mint, bonding, user [32]byte
	mint[0], mint[31] = 1, 9 // ayırt edici baytlar
	data := buildCreateEventB64("Doge Killer", "DOGEK", "https://x/uri.json", mint, bonding, user)

	n := LogNotification{
		Signature: "sig123", Slot: 42, ProgramID: PumpFunProgramID,
		Logs: []string{
			"Program " + PumpFunProgramID + " invoke [1]",
			"Program log: Instruction: Create",
			"Program data: " + data,
			"Program " + PumpFunProgramID + " success",
		},
	}
	d := NewPumpFunDecoder()
	out, err := d.Decode(context.Background(), n, nil, nil) // pump.fun tx/md kullanmaz
	if err != nil {
		t.Fatal(err)
	}
	// new_mint + metadata_created bekleniyor
	if len(out) != 2 {
		t.Fatalf("Decoded len=%d, want 2", len(out))
	}
	e0 := out[0].Event
	if e0.Type != "new_mint" || e0.Symbol != "DOGEK" || e0.Launchpad != "Pump.fun" {
		t.Fatalf("event0 = %+v", e0)
	}
	if e0.RiskLevel != "medium" || e0.CreatorScore != 0 {
		t.Fatalf("nötr placeholder bozuk: %+v", e0)
	}
	if out[0].Token.Name != "Doge Killer" || out[0].Token.Mint == "" {
		t.Fatalf("token = %+v", out[0].Token)
	}
	if out[1].Event.Type != "metadata_created" {
		t.Fatalf("event1 type = %s", out[1].Event.Type)
	}
}

func TestPumpFunIgnoresNonCreate(t *testing.T) {
	n := LogNotification{ProgramID: PumpFunProgramID, Logs: []string{"Program log: Instruction: Buy"}}
	out, err := NewPumpFunDecoder().Decode(context.Background(), n, nil, nil)
	if err != nil || len(out) != 0 {
		t.Fatalf("create olmayan log 0 olay vermeli; got %d, err=%v", len(out), err)
	}
}
```

- [ ] **Step 3: `decode_pumpfun.go` yaz**
```go
package ingest

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/mr-tron/base58" // base58 encode (pubkey). go get github.com/mr-tron/base58
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

const PumpFunProgramID = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"

type pumpFunDecoder struct{}

func NewPumpFunDecoder() LaunchpadDecoder { return pumpFunDecoder{} }

func (pumpFunDecoder) ProgramID() string { return PumpFunProgramID }
func (pumpFunDecoder) Launchpad() string { return "Pump.fun" }

func (d pumpFunDecoder) Decode(_ context.Context, n LogNotification, _ TxFetcher, _ MetadataFetcher) ([]Decoded, error) {
	if n.Err != nil {
		return nil, nil
	}
	if !hasMarker(n.Logs, "Instruction: Create") {
		return nil, nil
	}
	raw, ok := programDataB64(n.Logs)
	if !ok {
		return nil, nil // create var ama event data yok — atla (bozuk pipeline değil)
	}
	ev, err := parseCreateEvent(raw)
	if err != nil {
		return nil, fmt.Errorf("pumpfun createEvent parse: %w", err)
	}
	ts := int64(0) // worker gerçek zamanı damgalar (aşağıda Decoded.Event.Ts worker'da set edilir)
	token := store.TokenRow{
		ID: ev.mint, Mint: ev.mint, Symbol: ev.symbol, Name: ev.name,
		Spark: []float64{}, CreatorScore: 0, SafetyScore: 0, Momentum: 0,
	}
	mkEvent := func(id, typ, sev, detail string) store.EventRow {
		return store.EventRow{
			ID: n.Signature + "-" + id, Signature: n.Signature, Slot: n.Slot, Type: typ,
			Mint: ev.mint, Symbol: ev.symbol, Launchpad: "Pump.fun", DEX: "",
			RiskLevel: "medium", Severity: sev, CreatorScore: 0, Detail: detail, Ts: ts,
		}
	}
	return []Decoded{
		{Event: mkEvent("new_mint", "new_mint", "info", "Yeni token oluşturuldu (pump.fun)"), Token: token},
		{Event: mkEvent("metadata_created", "metadata_created", "info", ev.name+" ("+ev.symbol+")"), Token: token},
	}, nil
}

type createEvent struct{ name, symbol, uri, mint string }

func parseCreateEvent(raw []byte) (createEvent, error) {
	p := 8 // discriminator atla
	rd := func() (string, error) {
		if p+4 > len(raw) {
			return "", errors.New("kısa buffer (len prefix)")
		}
		ln := int(binary.LittleEndian.Uint32(raw[p : p+4]))
		p += 4
		if p+ln > len(raw) {
			return "", errors.New("kısa buffer (string body)")
		}
		s := string(raw[p : p+ln])
		p += ln
		return s, nil
	}
	name, err := rd()
	if err != nil {
		return createEvent{}, err
	}
	symbol, err := rd()
	if err != nil {
		return createEvent{}, err
	}
	uri, err := rd()
	if err != nil {
		return createEvent{}, err
	}
	if p+32 > len(raw) {
		return createEvent{}, errors.New("kısa buffer (mint)")
	}
	mint := base58.Encode(raw[p : p+32])
	return createEvent{name: name, symbol: symbol, uri: uri, mint: mint}, nil
}

func hasMarker(logs []string, marker string) bool {
	for _, l := range logs {
		if strings.Contains(l, marker) {
			return true
		}
	}
	return false
}

func programDataB64(logs []string) ([]byte, bool) {
	const pfx = "Program data: "
	for _, l := range logs {
		if strings.HasPrefix(l, pfx) {
			b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(l, pfx))
			if err == nil {
				return b, true
			}
		}
	}
	return nil, false
}
```
Not: `go get github.com/mr-tron/base58` (yaygın, hafif Solana base58 kütüphanesi; solana-go da bunu kullanır). Alternatif: solana-go eklenince `solana.PublicKeyFromBytes(...).String()` kullanılabilir — ama Task 6'ya kadar solana-go yok, o yüzden base58 doğrudan.

- [ ] **Step 4: Testi çalıştır — PASS**

Run: `cd apps/api-go && go get github.com/mr-tron/base58 && go test ./internal/ingest/ -run PumpFun -v` → PASS. Sonra `go build ./... && go vet ./...`.

- [ ] **Step 5: Commit**
```bash
git add apps/api-go/internal/ingest/ apps/api-go/go.mod apps/api-go/go.sum
git commit -m "feat(api-go): pump.fun log-only CreateEvent decoder + fixture"
```

---

## Task 5: Raydium CPMM decoder (tx-tabanlı) — ikinci decoder + OCP kanıtı

**Files:**
- Create: `apps/api-go/internal/ingest/decode_raydium.go`
- Test: `apps/api-go/internal/ingest/decode_raydium_test.go`

**Referans (doğrulanmış program ID; log casing UNVERIFIED → tx ile doğrula):** Raydium CPMM program ID `CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C`. Pool-create instruction adı `initialize` (log marker `Instruction: Initialize` — casing case-insensitive eşleştir). Mint log'da güvenilir değil → `TxFetcher.GetTransaction(sig)` ile hesap listesinden pozisyonel çıkar. **Pozisyon varsayımı** (Raydium CPMM `initialize` account ordering; deploy'da bir gerçek tx ile doğrulanır ve gerekiyorsa index düzeltilir): `AccountKeys` içinde token0Mint/token1Mint pozisyonları. 1a'da: SOL/WSOL olmayan ilk mint'i "yeni token" kabul et (WSOL = `So11111111111111111111111111111111111111112`). Metadata `MetadataFetcher` ile; başarısızsa symbol = kısaltılmış mint (dürüst fallback).

**Interfaces:**
- Consumes: Task 3 tipleri (`TxFetcher`, `MetadataFetcher`).
- Produces: `ingest.NewRaydiumCpmmDecoder() LaunchpadDecoder`, `ingest.RaydiumCpmmProgramID` const.

- [ ] **Step 1: Testi yaz (fake TxFetcher + MetadataFetcher; FAIL bekle)** — `decode_raydium_test.go`:
```go
package ingest

import (
	"context"
	"testing"
)

type fakeTx struct{ keys []string }
func (f fakeTx) GetTransaction(_ context.Context, _ string) (TxInfo, error) {
	return TxInfo{AccountKeys: f.keys}, nil
}
type fakeMeta struct{ name, symbol string; err error }
func (f fakeMeta) Metadata(_ context.Context, _ string) (TokenMeta, error) {
	if f.err != nil { return TokenMeta{}, f.err }
	return TokenMeta{Name: f.name, Symbol: f.symbol}, nil
}

const wsol = "So11111111111111111111111111111111111111112"

func TestRaydiumCpmmDecode(t *testing.T) {
	n := LogNotification{
		Signature: "rsig", Slot: 7, ProgramID: RaydiumCpmmProgramID,
		Logs: []string{
			"Program " + RaydiumCpmmProgramID + " invoke [1]",
			"Program log: Instruction: Initialize",
			"Program " + RaydiumCpmmProgramID + " success",
		},
	}
	tx := fakeTx{keys: []string{"someProgram", wsol, "NewMintPubkey11111111111111111111111111111", "vault"}}
	md := fakeMeta{name: "New Coin", symbol: "NEW"}
	out, err := NewRaydiumCpmmDecoder().Decode(context.Background(), n, tx, md)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Event.Type != "pool_created" {
		t.Fatalf("out = %+v", out)
	}
	if out[0].Event.Mint != "NewMintPubkey11111111111111111111111111111" {
		t.Fatalf("mint = %s", out[0].Event.Mint)
	}
	if out[0].Event.Launchpad != "Raydium" || out[0].Token.Symbol != "NEW" {
		t.Fatalf("event/token = %+v / %+v", out[0].Event, out[0].Token)
	}
}

func TestRaydiumFallbackSymbolOnMetaError(t *testing.T) {
	n := LogNotification{Signature: "s", ProgramID: RaydiumCpmmProgramID,
		Logs: []string{"Program log: Instruction: Initialize"}}
	tx := fakeTx{keys: []string{"p", wsol, "MintABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890xy"}}
	md := fakeMeta{err: context.DeadlineExceeded}
	out, err := NewRaydiumCpmmDecoder().Decode(context.Background(), n, tx, md)
	if err != nil || len(out) != 1 {
		t.Fatalf("fallback: out=%+v err=%v", out, err)
	}
	if out[0].Token.Symbol == "" { // kısaltılmış mint fallback
		t.Fatal("metadata hatasında symbol fallback boş olmamalı")
	}
}

func TestRaydiumIgnoresNonInitialize(t *testing.T) {
	n := LogNotification{ProgramID: RaydiumCpmmProgramID, Logs: []string{"Program log: Instruction: Swap"}}
	out, err := NewRaydiumCpmmDecoder().Decode(context.Background(), n, fakeTx{}, fakeMeta{})
	if err != nil || len(out) != 0 {
		t.Fatalf("initialize olmayan: %d olay, err=%v", len(out), err)
	}
}
```

- [ ] **Step 2: Testi çalıştır — FAIL** (`go test ./internal/ingest/ -run Raydium -v`).

- [ ] **Step 3: `decode_raydium.go` yaz**
```go
package ingest

import (
	"context"
	"strings"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

const RaydiumCpmmProgramID = "CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C"
const wsolMint = "So11111111111111111111111111111111111111112"

type raydiumCpmmDecoder struct{}

func NewRaydiumCpmmDecoder() LaunchpadDecoder { return raydiumCpmmDecoder{} }

func (raydiumCpmmDecoder) ProgramID() string { return RaydiumCpmmProgramID }
func (raydiumCpmmDecoder) Launchpad() string { return "Raydium" }

func (d raydiumCpmmDecoder) Decode(ctx context.Context, n LogNotification, tx TxFetcher, md MetadataFetcher) ([]Decoded, error) {
	if n.Err != nil {
		return nil, nil
	}
	if !hasMarkerFold(n.Logs, "instruction: initialize") {
		return nil, nil
	}
	if tx == nil {
		return nil, nil
	}
	info, err := tx.GetTransaction(ctx, n.Signature)
	if err != nil {
		return nil, nil // tx alınamadı → drop (pipeline durmaz; worker loglar)
	}
	mint := firstNonWSOLMint(info.AccountKeys)
	if mint == "" {
		return nil, nil
	}
	name, symbol := shortMint(mint), shortMint(mint) // fallback
	if md != nil {
		if meta, err := md.Metadata(ctx, mint); err == nil {
			if meta.Name != "" {
				name = meta.Name
			}
			if meta.Symbol != "" {
				symbol = meta.Symbol
			}
		}
	}
	token := store.TokenRow{ID: mint, Mint: mint, Symbol: symbol, Name: name, Spark: []float64{}}
	ev := store.EventRow{
		ID: n.Signature + "-pool_created", Signature: n.Signature, Slot: n.Slot, Type: "pool_created",
		Mint: mint, Symbol: symbol, Launchpad: "Raydium", DEX: "Raydium",
		RiskLevel: "medium", Severity: "positive", CreatorScore: 0,
		Detail: "Yeni likidite havuzu (Raydium CPMM)",
	}
	return []Decoded{{Event: ev, Token: token}}, nil
}

func firstNonWSOLMint(keys []string) string {
	// 1a heuristiği: WSOL olmayan, base58-uzunluğunda ilk hesap. Kesin index deploy'da doğrulanır.
	for _, k := range keys {
		if k == wsolMint {
			continue
		}
		if len(k) >= 32 && len(k) <= 44 { // base58 pubkey aralığı
			return k
		}
	}
	return ""
}

func shortMint(m string) string {
	if len(m) <= 8 {
		return m
	}
	return m[:4] + "…" + m[len(m)-4:]
}

func hasMarkerFold(logs []string, lowerMarker string) bool {
	for _, l := range logs {
		if strings.Contains(strings.ToLower(l), lowerMarker) {
			return true
		}
	}
	return false
}
```
Not (dürüstlük): `firstNonWSOLMint` sezgiseldir; ilk hesap genelde program/sistem hesabıdır ama testte program adı 32-44 aralığında olabilir → deploy'da gerçek Raydium CPMM `initialize` account layout'una göre **kesin index**'e sabitlenecek (followups'a not düşülür). 1a kabul kriteri: pump.fun çekirdeği + Raydium framework/tx-yolu çalışır; Raydium index kalibrasyonu deploy doğrulaması.

- [ ] **Step 4: Testi çalıştır — PASS** + `go build ./... && go vet ./...`.

- [ ] **Step 5: Commit**
```bash
git add apps/api-go/internal/ingest/
git commit -m "feat(api-go): Raydium CPMM tx-based decoder (OCP second decoder)"
```

---

## Task 6: Helius client — WS logsSubscribe + GetTransaction + DAS getAsset

**Files:**
- Create: `apps/api-go/internal/ingest/helius.go`
- Test: `apps/api-go/internal/ingest/helius_test.go`
- Modify: `apps/api-go/go.mod` (solana-go dep)

**Referans (doğrulanmış):** WS URL `wss://mainnet.helius-rpc.com/?api-key=KEY`, RPC URL `https://mainnet.helius-rpc.com/?api-key=KEY`. `logsSubscribe` `mentions` **tek** pubkey alır → program başına bir abonelik. 10 dk inaktivite timeout → ~30-60 sn ping. DAS `getAsset` POST JSON-RPC: `{"jsonrpc":"2.0","id":"1","method":"getAsset","params":{"id":"<mint>"}}` → `result.content.metadata.name/symbol`, `result.content.json_uri`. Kütüphane: `github.com/gagliardetto/solana-go` (`rpc` + `rpc/ws`): `ws.Connect`, `Client.LogsSubscribeMentions(pubkey, commitment)`, `rpc.New(url).GetTransaction(...)`.

**Interfaces:**
- Produces: `ingest.HeliusURLs(apiKey) (wsURL, rpcURL string)`, `ingest.NewHeliusMetadata(rpcURL) MetadataFetcher`, `ingest.NewHeliusTx(rpcURL) TxFetcher`, `ingest.SubscribeLogs(ctx, wsURL, programIDs, out chan<- LogNotification) error` (worker bunu çağırır).

- [ ] **Step 1: URL + metadata mapping birim testi (ağsız; FAIL bekle)** — `helius_test.go`:
```go
package ingest

import "testing"

func TestHeliusURLs(t *testing.T) {
	ws, rpc := HeliusURLs("KEY123")
	if ws != "wss://mainnet.helius-rpc.com/?api-key=KEY123" {
		t.Fatalf("ws=%s", ws)
	}
	if rpc != "https://mainnet.helius-rpc.com/?api-key=KEY123" {
		t.Fatalf("rpc=%s", rpc)
	}
}

func TestParseGetAssetResponse(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":"1","result":{"content":{"json_uri":"https://x/j.json","metadata":{"name":"Foo","symbol":"FOO"}}}}`)
	m, err := parseGetAsset(body)
	if err != nil || m.Name != "Foo" || m.Symbol != "FOO" || m.URI != "https://x/j.json" {
		t.Fatalf("m=%+v err=%v", m, err)
	}
}
```

- [ ] **Step 2: FAIL doğrula**, sonra bağımlılığı ekle: `cd apps/api-go && go get github.com/gagliardetto/solana-go@latest && go mod tidy`.

- [ ] **Step 3: `helius.go` yaz** (URL + DAS getAsset + tx adapter + WS abone; WS/tx canlı davranış deploy'da doğrulanır, birim test yalnız saf mapping'i kilitler)
```go
package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func HeliusURLs(apiKey string) (wsURL, rpcURL string) {
	return "wss://mainnet.helius-rpc.com/?api-key=" + apiKey,
		"https://mainnet.helius-rpc.com/?api-key=" + apiKey
}

// --- DAS getAsset (MetadataFetcher) ---
type heliusMetadata struct {
	rpcURL string
	http   *http.Client
}

func NewHeliusMetadata(rpcURL string) MetadataFetcher {
	return &heliusMetadata{rpcURL: rpcURL, http: &http.Client{Timeout: 8 * time.Second}}
}

func (h *heliusMetadata) Metadata(ctx context.Context, mint string) (TokenMeta, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getAsset",
		"params": map[string]any{"id": mint},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return TokenMeta{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return TokenMeta{}, err
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return TokenMeta{}, err
	}
	return parseGetAsset(buf.Bytes())
}

func parseGetAsset(body []byte) (TokenMeta, error) {
	var r struct {
		Result struct {
			Content struct {
				JSONURI  string `json:"json_uri"`
				Metadata struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				} `json:"metadata"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return TokenMeta{}, err
	}
	return TokenMeta{Name: r.Result.Content.Metadata.Name, Symbol: r.Result.Content.Metadata.Symbol, URI: r.Result.Content.JSONURI}, nil
}

// --- getTransaction (TxFetcher) ---
type heliusTx struct{ cli *rpc.Client }

func NewHeliusTx(rpcURL string) TxFetcher { return &heliusTx{cli: rpc.New(rpcURL)} }

func (h *heliusTx) GetTransaction(ctx context.Context, signature string) (TxInfo, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return TxInfo{}, err
	}
	maxV := uint64(0)
	out, err := h.cli.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		MaxSupportedTransactionVersion: &maxV,
	})
	if err != nil {
		return TxInfo{}, err
	}
	tx, err := out.Transaction.GetTransaction()
	if err != nil {
		return TxInfo{}, err
	}
	keys := make([]string, 0, len(tx.Message.AccountKeys))
	for _, k := range tx.Message.AccountKeys {
		keys = append(keys, k.String())
	}
	return TxInfo{AccountKeys: keys}, nil
}

// --- WS logsSubscribe (worker kullanır) ---
// SubscribeLogs, her programID için ayrı abonelik açar (mentions tek pubkey alır),
// bildirimleri normalize edip out kanalına yazar. ctx iptalinde döner. Reconnect worker'da.
func SubscribeLogs(ctx context.Context, wsURL string, programIDs []string, out chan<- LogNotification) error {
	client, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return fmt.Errorf("ws connect: %w", err)
	}
	defer client.Close()

	for _, pid := range programIDs {
		pk, err := solana.PublicKeyFromBase58(pid)
		if err != nil {
			return fmt.Errorf("bad programID %s: %w", pid, err)
		}
		sub, err := client.LogsSubscribeMentions(pk, rpc.CommitmentProcessed)
		if err != nil {
			return fmt.Errorf("logsSubscribe %s: %w", pid, err)
		}
		go recvLoop(ctx, sub, pid, out)
	}
	<-ctx.Done()
	return ctx.Err()
}

func recvLoop(ctx context.Context, sub *ws.LogSubscription, programID string, out chan<- LogNotification) {
	defer sub.Unsubscribe()
	for {
		got, err := sub.Recv(ctx)
		if err != nil {
			return // bağlantı koptu → worker reconnect eder
		}
		if got == nil {
			continue
		}
		v := got.Value
		n := LogNotification{
			Signature: v.Signature.String(),
			Slot:      got.Context.Slot,
			Err:       v.Err,
			Logs:      v.Logs,
			ProgramID: programID,
		}
		select {
		case out <- n:
		case <-ctx.Done():
			return
		}
	}
}
```
Not: solana-go `ws.LogSubscription.Recv` imzası sürümle değişebilir (alpha). Task 6 uygularken `go doc github.com/gagliardetto/solana-go/rpc/ws` ile `LogsSubscribeMentions`/`Recv` imzalarını doğrula ve gerekirse `recvLoop`'u uyarl/düzelt (davranış aynı: bildirim → LogNotification). Canlı WS/tx yalnız deploy'da doğrulanır (Helius key gerekli — Alt-proje 0 Postgres deseni). `parseGetAsset` + `HeliusURLs` ağsız birim testte kilitlidir.

- [ ] **Step 4: Birim testleri çalıştır — PASS** (`go test ./internal/ingest/ -run 'HeliusURLs|GetAsset' -v`) + `go build ./... && go vet ./...`.

- [ ] **Step 5: Commit**
```bash
git add apps/api-go/internal/ingest/ apps/api-go/go.mod apps/api-go/go.sum
git commit -m "feat(api-go): Helius client (logsSubscribe WS + getTransaction + DAS getAsset)"
```

---

## Task 7: Ingestion worker — subscribe/route/dedup/backoff/persist/broadcast

**Files:**
- Create: `apps/api-go/internal/ingest/worker.go`
- Test: `apps/api-go/internal/ingest/worker_test.go`

**Interfaces:**
- Consumes: `Registry`, `TxFetcher`, `MetadataFetcher`, `store.EventStore`, `store.TokenStore`, `SubscribeLogs`; broadcast için `Broadcaster` interface (DIP → ws.Hub'a bağlanır Task 8/9).
- Produces: `ingest.Broadcaster` interface, `ingest.NewWorker(...)`, `Worker.Process(ctx, n)` (test edilebilir birim), `Worker.Run(ctx)` (canlı döngü + backoff).

- [ ] **Step 1: `Process` (saf yönlendirme+dedup) + dedup testi (FAIL bekle)** — `worker_test.go`:
```go
package ingest

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type capBroadcaster struct{ topics []string }
func (c *capBroadcaster) Broadcast(topic string, _ any) { c.topics = append(c.topics, topic) }

func newTestWorker() (*Worker, store.EventStore, store.TokenStore, *capBroadcaster) {
	reg := NewRegistry()
	reg.Register(NewPumpFunDecoder())
	es, ts := store.NewFakeEventStore(), store.NewFakeTokenStore()
	bc := &capBroadcaster{}
	w := NewWorker(WorkerDeps{Registry: reg, Events: es, Tokens: ts, Broadcast: bc, Now: func() int64 { return 111 }})
	return w, es, ts, bc
}

func TestProcessPersistsAndBroadcasts(t *testing.T) {
	w, es, ts, bc := newTestWorker()
	var mint [32]byte
	mint[0] = 3
	data := buildCreateEventB64("Cat", "CAT", "u", mint, [32]byte{}, [32]byte{})
	n := LogNotification{Signature: "sig", Slot: 1, ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data}}

	w.Process(context.Background(), n)

	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 2 { // new_mint + metadata_created
		t.Fatalf("events=%d, want 2", len(evs))
	}
	if evs[0].Ts != 111 {
		t.Fatalf("worker Ts damgalamadı: %d", evs[0].Ts)
	}
	toks, _ := ts.RecentTokens(context.Background(), 10)
	if len(toks) != 1 {
		t.Fatalf("tokens=%d, want 1", len(toks))
	}
	// events + tokens topic'lerine broadcast
	if len(bc.topics) == 0 {
		t.Fatal("broadcast yok")
	}
}

func TestProcessDedup(t *testing.T) {
	w, es, _, _ := newTestWorker()
	var mint [32]byte
	data := buildCreateEventB64("Cat", "CAT", "u", mint, [32]byte{}, [32]byte{})
	n := LogNotification{Signature: "sig", Slot: 1, ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data}}
	w.Process(context.Background(), n)
	w.Process(context.Background(), n) // aynı sig+type → dedup
	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 2 {
		t.Fatalf("dedup başarısız: events=%d, want 2 (tekrar eklenmemeli)", len(evs))
	}
}

func TestProcessUnknownProgramNoop(t *testing.T) {
	w, es, _, _ := newTestWorker()
	w.Process(context.Background(), LogNotification{ProgramID: "unknown", Logs: []string{"x"}})
	evs, _ := es.RecentEvents(context.Background(), 10)
	if len(evs) != 0 {
		t.Fatalf("bilinmeyen program olay üretmemeli: %d", len(evs))
	}
}
```

- [ ] **Step 2: FAIL doğrula** (`go test ./internal/ingest/ -run Process -v`).

- [ ] **Step 3: `worker.go` yaz**
```go
package ingest

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Broadcaster, decode edilmiş kaydı bağlı client'lara yayar (DIP → ws.Hub).
type Broadcaster interface {
	Broadcast(topic string, payload any)
}

type WorkerDeps struct {
	Registry  *Registry
	Events    store.EventStore
	Tokens    store.TokenStore
	Broadcast Broadcaster
	Tx        TxFetcher       // canlıda Helius; testte nil/fake
	Meta      MetadataFetcher // canlıda Helius; testte nil/fake
	WSURL     string          // canlı abonelik; testte boş
	Now       func() int64    // enjekte edilebilir saat (test determinizmi)
	Logger    *slog.Logger
}

type Worker struct {
	d       WorkerDeps
	mu      sync.Mutex
	seen    map[string]struct{} // dedup: signature|type
}

func NewWorker(d WorkerDeps) *Worker {
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Worker{d: d, seen: map[string]struct{}{}}
}

// Process, tek bir log bildirimini decode eder, dedup uygular, persist + broadcast yapar.
func (w *Worker) Process(ctx context.Context, n LogNotification) {
	dec, ok := w.d.Registry.Decoder(n.ProgramID)
	if !ok {
		return
	}
	decoded, err := dec.Decode(ctx, n, w.d.Tx, w.d.Meta)
	if err != nil {
		w.d.Logger.Warn("decode error", "program", n.ProgramID, "sig", n.Signature, "err", err)
		return
	}
	now := w.d.Now()
	for _, item := range decoded {
		key := n.Signature + "|" + item.Event.Type
		w.mu.Lock()
		if _, dup := w.seen[key]; dup {
			w.mu.Unlock()
			continue
		}
		w.seen[key] = struct{}{}
		w.mu.Unlock()

		e := item.Event
		e.Ts = now
		if err := w.d.Events.InsertEvent(ctx, e); err != nil {
			w.d.Logger.Warn("insert event", "err", err)
			continue
		}
		if err := w.d.Tokens.UpsertToken(ctx, item.Token, now); err != nil {
			w.d.Logger.Warn("upsert token", "err", err)
		}
		w.d.Broadcast.Broadcast("events", e)
		w.d.Broadcast.Broadcast("tokens", item.Token)
	}
}

// Run, Helius'a bağlanır, kopmada exponential backoff ile yeniden bağlanır.
// ctx iptaline kadar çalışır. WSURL boşsa (test/DB-siz) hemen döner.
func (w *Worker) Run(ctx context.Context) {
	if w.d.WSURL == "" {
		w.d.Logger.Warn("HELIUS_WS_URL yok — ingestion worker başlamadı (REST yine çalışır)")
		return
	}
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for ctx.Err() == nil {
		ch := make(chan LogNotification, 256)
		done := make(chan error, 1)
		subCtx, cancel := context.WithCancel(ctx)
		go func() { done <- SubscribeLogs(subCtx, w.d.WSURL, w.d.Registry.ProgramIDs(), ch) }()

		connected := true
		for connected {
			select {
			case <-ctx.Done():
				cancel()
				return
			case n := <-ch:
				w.Process(ctx, n)
				backoff = time.Second // sağlıklı trafik → backoff sıfırla
			case err := <-done:
				w.d.Logger.Warn("ws bağlantısı koptu, reconnect", "err", err, "backoff", backoff.String())
				connected = false
			}
		}
		cancel()
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		if backoff < maxBackoff {
			backoff *= 2
		}
	}
}
```
Not: dedup `seen` haritası sınırsız büyür — 1a'da kısa-ömürlü servis için kabul; followups'a "dedup TTL/bounded LRU" notu düş (spec §9 backpressure ile birlikte). `UpsertToken(ctx, token, now)` imzası Task 2'deki `firstSeenTs` ile uyumlu.

- [ ] **Step 4: Testleri çalıştır — PASS** + `go build ./... && go vet ./...`.

- [ ] **Step 5: Commit**
```bash
git add apps/api-go/internal/ingest/
git commit -m "feat(api-go): ingestion worker (route/dedup/persist/broadcast + backoff Run)"
```

---

## Task 8: Frontend-facing WebSocket hub

**Files:**
- Create: `apps/api-go/internal/ws/hub.go`
- Test: `apps/api-go/internal/ws/hub_test.go`
- Modify: `apps/api-go/go.mod` (coder/websocket)

**Interfaces:**
- Consumes: (yalın — payload `any`).
- Produces: `ws.Hub`, `ws.NewHub()`, `Hub.Run(ctx)`, `Hub.Broadcast(topic string, payload any)` (→ `ingest.Broadcaster` uyumlu), `Hub.ServeWS(w, r)`, `Hub.ClientCount()` (test).

- [ ] **Step 1: Broadcast/register testini yaz (gerçek ağ olmadan — iç register/unregister + fan-out; FAIL bekle)** — `hub_test.go`:
```go
package ws

import (
	"context"
	"testing"
	"time"
)

func TestHubBroadcastFanOut(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c1 := h.registerForTest()
	c2 := h.registerForTest()
	// Run döngüsünün register'ları işlemesine izin ver
	waitFor(t, func() bool { return h.ClientCount() == 2 })

	h.Broadcast("events", map[string]any{"id": "e1"})

	for _, c := range []*client{c1, c2} {
		select {
		case msg := <-c.send:
			if msg.Topic != "events" {
				t.Fatalf("topic=%s", msg.Topic)
			}
		case <-time.After(time.Second):
			t.Fatal("client mesaj almadı")
		}
	}
}

func TestHubUnregister(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)
	c := h.registerForTest()
	waitFor(t, func() bool { return h.ClientCount() == 1 })
	h.unregisterForTest(c)
	waitFor(t, func() bool { return h.ClientCount() == 0 })
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("koşul zamanında sağlanmadı")
}
```

- [ ] **Step 2: FAIL doğrula**, dep ekle: `cd apps/api-go && go get github.com/coder/websocket && go mod tidy`.

- [ ] **Step 3: `hub.go` yaz**
```go
package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	cws "github.com/coder/websocket"
)

// message, /ws üzerinden gönderilen zarf: {topic, payload}.
type message struct {
	Topic   string `json:"topic"`
	Payload any    `json:"payload"`
}

type client struct {
	send chan message // bounded; dolu ise mesaj düşürülür (yavaş client)
}

// Hub, bağlı frontend WS client'larına topic bazlı yayın yapar (SRP: yalnız fan-out).
type Hub struct {
	register   chan *client
	unregister chan *client
	broadcast  chan message
	mu         sync.RWMutex
	clients    map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan message, 256),
		clients:    map[*client]struct{}{},
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default: // yavaş client → mesajı düşür (bounded)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast, ingest.Broadcaster arayüzünü karşılar.
func (h *Hub) Broadcast(topic string, payload any) {
	select {
	case h.broadcast <- message{Topic: topic, Payload: payload}:
	default: // hub kuyruğu dolu → düşür (backpressure)
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ServeWS, /ws bağlantısını kabul eder; client'ı kaydeder ve yazma-pump çalıştırır.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := cws.Accept(w, r, &cws.AcceptOptions{InsecureSkipVerify: true}) // CORS: origin kontrolü ayrı
	if err != nil {
		return
	}
	c := &client{send: make(chan message, 64)}
	h.register <- c
	defer func() { h.unregister <- c }()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			conn.Close(cws.StatusNormalClosure, "")
			return
		case msg, ok := <-c.send:
			if !ok {
				conn.Close(cws.StatusNormalClosure, "")
				return
			}
			b, _ := json.Marshal(msg)
			if err := conn.Write(ctx, cws.MessageText, b); err != nil {
				return
			}
		}
	}
}

// --- test yardımcıları (aynı pakette; ağsız register/unregister) ---
func (h *Hub) registerForTest() *client {
	c := &client{send: make(chan message, 8)}
	h.register <- c
	return c
}
func (h *Hub) unregisterForTest(c *client) { h.unregister <- c }
```
Not: `InsecureSkipVerify:true` — WS origin doğrulaması 1a'da kapalı (public read-only, Alt-proje 0'daki gibi auth yok); followups'a "WS origin allowlist" notu. Ping/pong: coder/websocket kontrol frame'lerini otomatik yönetir; ek olarak istenirse `conn.Ping` eklenebilir (1b).

- [ ] **Step 4: Testleri çalıştır — PASS** + `go build ./... && go vet ./...`.

- [ ] **Step 5: Commit**
```bash
git add apps/api-go/internal/ws/ apps/api-go/go.mod apps/api-go/go.sum
git commit -m "feat(api-go): frontend WebSocket hub (topic broadcast, bounded send)"
```

---

## Task 9: API handler'ları + router + config + main wiring

**Files:**
- Create: `apps/api-go/internal/api/events.go` (GET /api/events, /api/tokens)
- Create: `apps/api-go/internal/api/ws.go` (GET /ws)
- Modify: `apps/api-go/internal/api/router.go` (yeni route'lar + imza)
- Modify: `apps/api-go/internal/config/config.go` (HeliusAPIKey, EventsWindow)
- Modify: `apps/api-go/cmd/server/main.go` (bundle + worker + hub başlat, graceful shutdown)
- Test: `apps/api-go/internal/api/events_test.go`, `apps/api-go/internal/config/config_test.go`

**Interfaces:**
- Consumes: `store.EventStore`/`store.TokenStore`, `ws.Hub`, `config.Config`.
- Produces: `api.NewRouter(deps RouterDeps) http.Handler` (genişletilmiş imza).

- [ ] **Step 1: Handler testini yaz (FAIL bekle)** — `events_test.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestEventsHandler(t *testing.T) {
	es := store.NewFakeEventStore()
	_ = es.InsertEvent(nil, store.EventRow{ID: "e1", Type: "new_mint", Mint: "M", Ts: 1})
	ts := store.NewFakeTokenStore()
	_ = ts.UpsertToken(nil, store.TokenRow{ID: "M", Mint: "M", Symbol: "S"}, 1)

	r := NewRouter(RouterDeps{Events: es, Tokens: ts, EventsWindow: 200})

	for _, tc := range []struct{ path, wantKey string }{
		{"/api/events", "type"}, {"/api/tokens", "symbol"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s code=%d", tc.path, w.Code)
		}
		var arr []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
			t.Fatalf("%s json: %v", tc.path, err)
		}
		if len(arr) == 0 || arr[0][tc.wantKey] == nil {
			t.Fatalf("%s body=%s", tc.path, w.Body.String())
		}
	}
}

func TestEventsHandlerEmptyIsArray(t *testing.T) {
	r := NewRouter(RouterDeps{Events: store.NewFakeEventStore(), Tokens: store.NewFakeTokenStore(), EventsWindow: 200})
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Body.String() != "[]\n" && w.Body.String() != "[]" {
		t.Fatalf("boş sonuç [] olmalı, got %q", w.Body.String())
	}
}
```
(Not: fake store'lar `nil` ctx'i kabul eder — imzada `_ context.Context`.)

- [ ] **Step 2: FAIL doğrula** (`go test ./internal/api/ -v`).

- [ ] **Step 3: `config.go` genişlet**
```go
type Config struct {
	Port         string
	DatabaseURL  string
	CORSOrigin   string
	HeliusAPIKey string
	EventsWindow int
}

func Load() Config {
	return Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CORSOrigin:   os.Getenv("CORS_ORIGIN"),
		HeliusAPIKey: os.Getenv("HELIUS_API_KEY"),
		EventsWindow: getenvInt("EVENTS_WINDOW", 200),
	}
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
```
(`import "strconv"` ekle. `config_test.go`: `TestLoadDefaults` — env temizken `EventsWindow==200`, `Port=="8080"`.)

- [ ] **Step 4: `events.go` + `ws.go` handler'ları yaz**
```go
// events.go
package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func eventsHandler(es store.EventStore, window int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := es.RecentEvents(r.Context(), window)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "events unavailable"})
			return
		}
		if rows == nil {
			rows = []store.EventRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func tokensHandler(ts store.TokenStore, window int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := ts.RecentTokens(r.Context(), window)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "tokens unavailable"})
			return
		}
		if rows == nil {
			rows = []store.TokenRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}
```
```go
// ws.go
package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/ws"
)

func wsHandler(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hub == nil {
			http.Error(w, "ws unavailable", http.StatusServiceUnavailable)
			return
		}
		hub.ServeWS(w, r)
	}
}
```

- [ ] **Step 5: `router.go`'yu genişlet** (RouterDeps ile — mevcut `NewRouter(st, corsOrigin)` çağıranları güncelle)
```go
type RouterDeps struct {
	Strategies   store.StrategyStore
	Events       store.EventStore
	Tokens       store.TokenStore
	Hub          *ws.Hub
	CORSOrigin   string
	EventsWindow int
}

func NewRouter(d RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(d.CORSOrigin))
	r.Get("/healthz", healthHandler)
	if d.Strategies != nil {
		r.Get("/api/strategies", strategiesHandler(d.Strategies))
	}
	if d.Events != nil {
		r.Get("/api/events", eventsHandler(d.Events, d.EventsWindow))
	}
	if d.Tokens != nil {
		r.Get("/api/tokens", tokensHandler(d.Tokens, d.EventsWindow))
	}
	r.Get("/ws", wsHandler(d.Hub))
	return r
}
```
→ `router_test.go`'daki mevcut `NewRouter(st, origin)` çağrılarını `NewRouter(RouterDeps{Strategies: st, CORSOrigin: origin})` olarak güncelle. `EventsWindow` 0 gelirse handler'da `if window<=0 { window=200 }` guard'ı ekle (defensive).

- [ ] **Step 6: `main.go`'yu güncelle** (bundle + hub + worker başlat)
```go
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var bundle store.Bundle
	var cleanup = func() error { return nil }
	if cfg.DatabaseURL != "" {
		dbctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		b, cl, err := store.OpenPostgres(dbctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			logger.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		bundle, cleanup = b, cl
	} else {
		logger.Warn("DATABASE_URL yok — in-memory fake store")
		bundle = store.Bundle{
			Strategies: store.NewFakeStore(store.SeedRows(), nil),
			Events:     store.NewFakeEventStore(),
			Tokens:     store.NewFakeTokenStore(),
		}
	}
	defer cleanup()

	hub := ws.NewHub()
	go hub.Run(ctx)

	// ingestion worker (Helius key varsa)
	reg := ingest.NewRegistry()
	reg.Register(ingest.NewPumpFunDecoder())
	reg.Register(ingest.NewRaydiumCpmmDecoder())
	wsURL, rpcURL := "", ""
	if cfg.HeliusAPIKey != "" {
		wsURL, rpcURL = ingest.HeliusURLs(cfg.HeliusAPIKey)
	} else {
		logger.Warn("HELIUS_API_KEY yok — ingestion worker başlamayacak (REST/mock boş DB çalışır)")
	}
	worker := ingest.NewWorker(ingest.WorkerDeps{
		Registry: reg, Events: bundle.Events, Tokens: bundle.Tokens, Broadcast: hub,
		Tx: ingest.NewHeliusTx(rpcURL), Meta: ingest.NewHeliusMetadata(rpcURL),
		WSURL: wsURL, Logger: logger,
	})
	go worker.Run(ctx)

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: api.NewRouter(api.RouterDeps{
			Strategies: bundle.Strategies, Events: bundle.Events, Tokens: bundle.Tokens,
			Hub: hub, CORSOrigin: cfg.CORSOrigin, EventsWindow: cfg.EventsWindow,
		}),
	}
	go func() {
		logger.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	logger.Info("stopped")
}
```
(imports: `ingest`, `ws` paketleri ekle. `rpcURL` boşken `NewHeliusTx("")`/`NewHeliusMetadata("")` çağrılır ama `WSURL==""` olduğundan worker.Run hemen döner → tx/meta hiç kullanılmaz; güvenli.)

- [ ] **Step 7: Tüm testleri + build çalıştır — PASS**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./...`
Expected: tüm birim testler PASS, DB integration SKIP.

- [ ] **Step 8: Commit**
```bash
git add apps/api-go/
git commit -m "feat(api-go): events/tokens/ws handlers + worker/hub wiring + config"
```

---

## Task 10: Frontend — gerçek fetch + gerçek WebSocket + hibrit ekleme + dürüst nötr skor

**Files:**
- Create: `apps/web/lib/api/ws.ts`
- Modify: `apps/web/lib/api/http.ts`
- Modify: `apps/web/lib/api/live-endpoints.ts`
- Modify: nötr-skor gösterimi (ScoreBadge/severity — mevcut bileşeni bul; bkz Step 5)
- Test: `apps/web/lib/api/ws.test.ts`, `apps/web/lib/api/http.test.ts` (mevcut test desenine göre)

**Interfaces:**
- Consumes: backend `/api/events`, `/api/tokens`, `/ws` (message `{topic,payload}`).
- Produces: `wsSubscribe(topic, cb)` helper; `httpApi.getEvents/getTokens/subscribeEvents/subscribeTokens` gerçek.

- [ ] **Step 1: WS transport testini yaz (mock WebSocket; FAIL bekle)** — `ws.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach } from "vitest";
import { wsSubscribe, wsBaseUrl } from "./ws";

class MockWS {
  static instances: MockWS[] = [];
  onmessage: ((e: { data: string }) => void) | null = null;
  onopen: (() => void) | null = null;
  closed = false;
  sent: string[] = [];
  constructor(public url: string) { MockWS.instances.push(this); }
  send(d: string) { this.sent.push(d); }
  close() { this.closed = true; }
}

beforeEach(() => {
  MockWS.instances = [];
  vi.stubGlobal("WebSocket", MockWS as unknown as typeof WebSocket);
  vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "https://api.example.com");
});

describe("wsBaseUrl", () => {
  it("https → wss", () => {
    expect(wsBaseUrl()).toBe("wss://api.example.com/ws");
  });
});

describe("wsSubscribe", () => {
  it("topic mesajında cb çağrılır, unsubscribe socket kapatır", () => {
    const cb = vi.fn();
    const unsub = wsSubscribe<{ id: string }>("events", cb);
    const ws = MockWS.instances[0];
    ws.onopen?.();
    ws.onmessage?.({ data: JSON.stringify({ topic: "events", payload: { id: "e1" } }) });
    expect(cb).toHaveBeenCalledWith({ id: "e1" });
    // farklı topic → cb çağrılmaz
    ws.onmessage?.({ data: JSON.stringify({ topic: "tokens", payload: { id: "t" } }) });
    expect(cb).toHaveBeenCalledTimes(1);
    unsub();
    expect(ws.closed).toBe(true);
  });
});
```

- [ ] **Step 2: FAIL doğrula** (`cd apps/web && npm test -- ws.test`).

- [ ] **Step 3: `ws.ts` yaz**
```ts
// WebSocket transport: NEXT_PUBLIC_API_BASE_URL'den wss türetir, topic'e abone olur.
export function wsBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!base) throw new Error("NEXT_PUBLIC_API_BASE_URL is not set");
  return base.replace(/^http/, "ws").replace(/\/$/, "") + "/ws";
}

interface Envelope<T> { topic: string; payload: T; }

// wsSubscribe, verilen topic mesajlarında cb'yi çağırır. Dönen fonksiyon aboneliği kapatır.
export function wsSubscribe<T>(topic: "events" | "tokens", cb: (payload: T) => void): () => void {
  const ws = new WebSocket(wsBaseUrl());
  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data) as Envelope<T>;
      if (msg.topic === topic) cb(msg.payload);
    } catch {
      /* bozuk mesaj yoksay */
    }
  };
  return () => ws.close();
}
```

- [ ] **Step 4: `http.ts`'i güncelle** (getEvents/getTokens gerçek fetch, subscribe* gerçek WS)
```ts
import type { FeedEvent, TokenRow, StrategyRow } from "./types";
import { wsSubscribe } from "./ws";
// ...
export const httpApi: SentinelApi = {
  // ...
  getEvents: () => getJson<FeedEvent[]>("/api/events"),
  getTokens: () => getJson<TokenRow[]>("/api/tokens"),
  // ...
  subscribeEvents: (cb) => wsSubscribe<FeedEvent>("events", cb),
  subscribeTokens: (cb) => wsSubscribe<TokenRow>("tokens", cb),
  subscribeAlerts: () => () => {}, // Alt-proje 3
  // ...
};
```
(`getKpis`/`getRadar`/`getToken` `notReady` KALIR — 1a kapsamı dışı.)

- [ ] **Step 5: `live-endpoints.ts`'e ekle**
```ts
export const LIVE_ENDPOINTS = new Set<keyof SentinelApi>([
  "getStrategies",
  "getEvents",
  "getTokens",
  "subscribeEvents",
  "subscribeTokens",
]);
```

- [ ] **Step 6: Dürüst nötr skor gösterimi.** `creatorScore`/`safetyScore === 0` (1a'da her zaman) UI'da "—"/"henüz yok" göstersin, "0/sahte skor" değil. İlgili bileşeni bul (`grep -rn "creatorScore\|safetyScore\|ScoreBadge" apps/web/components apps/web/lib`) → skor rozeti render eden yerde `value === 0 ? "—" : formatScore(value)` uygula. Bir birim testi ekle (rozet 0 → "—"). **Bu bileşen 1a öncesi mock skorlarla dolu olduğundan** regresyon riski var: değişikliği "0 ise nötr göster" ile sınırla (mock veride 0 skor yoksa görsel değişmez).

- [ ] **Step 7: Test paketini çalıştır — PASS**

Run: `cd apps/web && npm test` → tüm testler yeşil (mevcut 179 + yeni ws/http/score). `npm run build` (typecheck) yeşil.
Expected: PASS. Regresyon yok (hibrit: yalnız 5 endpoint http, kalan mock).

- [ ] **Step 8: Commit**
```bash
git add apps/web/lib/api/ apps/web/components/
git commit -m "feat(web): real getEvents/getTokens + WS subscribe (hybrid LIVE_ENDPOINTS +4)"
```

---

## Task 11: Living docs + followups + review handoff

**Files:**
- Modify: `docs/progress.md` (Backend programı → Alt-proje 1 slice 1a)
- Modify: `docs/superpowers/followups-frontend.md` (1a bölümü)
- Modify: `apps/api-go/README.md` (yeni env: HELIUS_API_KEY, EVENTS_WINDOW; /api/events,/api/tokens,/ws)
- Modify: `api_key_alinacakplatformlar.md` (Helius durumunu "implementasyon tamam, deploy'da key gir" notu)
- Verify: CI `.github/workflows/api-go.yml` `go test ./...` yeni paketleri kapsar (ingest/ws) — genelde otomatik; kontrol et.

- [ ] **Step 1: `docs/progress.md`'ye 1a slice özetini ekle** — teslim edilen bileşenler, kapsam kararları (pump.fun+Raydium somut, PumpSwap/Moonshot/Meteora ertelendi, first_swap/liquidity_added → 1b), nötr placeholder politikası, "DB+Helius canlı doğrulama deploy'da" notu.

- [ ] **Step 2: `followups-frontend.md`'ye 1a deferred'ları ekle:**
  - Raydium CPMM `initialize` account **kesin index kalibrasyonu** (gerçek tx ile; şimdilik WSOL-olmayan-ilk heuristiği).
  - PumpSwap decoder + `sources.ts`'e "PumpSwap" ekleme.
  - Worker dedup **bounded/TTL** (şu an sınırsız `seen` map).
  - WS hub **origin allowlist** (şu an InsecureSkipVerify).
  - PumpSwap/Raydium log casing UNVERIFIED → deploy'da doğrula.
  - Gerçek pump.fun fixture (Solscan snapshot) ekle (şu an üretilmiş test vektörü + discriminator doğrulaması).
  - `getKpis`/`getRadar`/`getToken` + first_swap/liquidity_added → Slice 1b.

- [ ] **Step 3: `README.md` + checklist güncelle** (env değişkenleri + endpoint listesi + Railway'e `HELIUS_API_KEY` girme adımı).

- [ ] **Step 4: Tüm test paketlerini son kez çalıştır**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test ./...` ve `cd apps/web && npm test && npm run build`
Expected: Go PASS (DB integration SKIP), frontend PASS + build yeşil.

- [ ] **Step 5: Commit + branch review handoff**
```bash
git add docs/ apps/api-go/README.md api_key_alinacakplatformlar.md
git commit -m "docs(ingestion): Alt-proje 1 slice 1a — progress, followups, README, checklist"
```
Sonra: whole-branch review (superpowers:requesting-code-review, opus) → "Ready to merge" alınınca kullanıcı onayıyla master'a merge. **Deploy DUR-noktası:** merge sonrası Railway'e `HELIUS_API_KEY` girmeden önce kullanıcıdan Helius key'i **rotate** ettirilir (sohbete sızan key iptal), taze key Railway Variables'a girilir; `/api/events` + `/ws` canlı doğrulanır (Live Feed gerçek Solana akışı).

---

## Self-Review (spec karşılaştırması)

**Spec coverage:**
- §1 ingestion worker + WS hub + getEvents/getTokens gerçek + subscribe* gerçek → Task 1-10 ✅
- §2 OCP decoder registry / SRP / DIP / reuse → Task 3 (registry), Task 4-5 (decoder'lar), interface'ler (TxFetcher/MetadataFetcher/EventStore/TokenStore), frontend hibrit ekleme ✅
- §3.1 worker/helius/decoder/registry/hub dosya yapısı → Task 3-8 ✅ (decode_pumpswap **bilinçle ertelendi**, tabloda işaretli)
- §3.2 migration 0002 events+tokens, nötr alan doldurma → Task 1-2 ✅
- §3.3 http.ts getEvents/getTokens + subscribe* WS + LIVE_ENDPOINTS + WS base URL türet + dürüst placeholder → Task 10 ✅
- §4 veri akışı → Task 7 (worker) + Task 8 (hub) + Task 10 (frontend) ✅
- §5 hata yönetimi: backoff reconnect (Task 7 Run), decode hatası drop (Task 7 Process), DAS başarısız fallback (Task 5), HELIUS_API_KEY yoksa worker başlamaz (Task 9 main), graceful shutdown (Task 9) ✅
- §6 test stratejisi: decoder unit fixture (Task 4-5), hub (Task 8), store integration guard'lı (Task 2), registry (Task 3), frontend map+WS (Task 10) ✅
- §7 kabul kriterleri 1-6 → build/test yeşil, deploy canlı doğrulama (Task 11 handoff), hibrit regresyonsuz (Task 10), OCP kanıtı (Task 3-5) ✅
- §8 Helius key kullanıcı aksiyonu → Task 11 deploy DUR-noktası ✅
- §9 açık noktalar: logsSubscribe seçildi (Task 6), program ID/log format (Task 4-5 doğrulanmış + UNVERIFIED işaretli), nötr skor UI (Task 10 Step 6), recent-N + backpressure (EventsWindow config + hub bounded) ✅

**Placeholder taraması:** Her task'ta gerçek kod + gerçek test var; "TBD/TODO" yok. Empirik belirsizlikler (Raydium account index, gerçek fixture, log casing) **açıkça işaretlendi** ve deploy doğrulama + followups'a bağlandı — bu placeholder değil, dürüst kapsam sınırı.

**Type consistency:** `EventRow`/`TokenRow` JSON key'leri frontend `FeedEvent`/`TokenRow` ile birebir (Task 1 testi kilitler). `UpsertToken(ctx, t, firstSeenTs)` imzası Task 1/2/7'de tutarlı. `Broadcaster.Broadcast(topic, payload)` Task 7/8/9'da aynı. `LaunchpadDecoder.Decode(ctx, n, tx, md)` Task 3/4/5/7'de aynı.
