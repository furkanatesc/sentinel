# SENTINEL Backend Alt-proje 0 — Platform İskeleti Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Go API servisi + Railway Postgres ayağa kaldır, tek gerçek endpoint (`GET /api/strategies`) uçtan uca çalıştır, ve frontend'i hibrit adapter'a çevirerek `getStrategies`'i gerçeğe, kalan 8 ekranı mock'ta tut.

**Architecture:** Yeni `apps/api-go/` Go servisi (chi router, katmanlı: config / store / api). `StrategyStore` interface'i (DIP) arkasında Postgres (database/sql + pgx stdlib driver + goose migration) ya da testler için in-memory fake. Frontend `getApi()` config-driven **hibrit adapter**'a döner: `LIVE_ENDPOINTS`'teki metotlar `httpApi`'ye (gerçek fetch), kalanı `mockApi`'ye bağlanır.

**Tech Stack:** Go 1.23, `github.com/go-chi/chi/v5`, `github.com/jackc/pgx/v5` (stdlib driver), `github.com/pressly/goose/v3`, Postgres (Railway). Frontend: mevcut Next.js/TypeScript/Vitest.

## Global Constraints

- **Go modül yolu:** `github.com/furkanatesc/sentinel/apps/api-go`. Tüm iç importlar bu prefix'le.
- **Go sürümü:** 1.23 (go.mod `go 1.23`).
- **Kontrat sadakati:** Go `StrategyRow` struct'ının JSON çıktısı frontend `StrategyRow` (`apps/web/lib/api/types.ts:147`) ile **birebir aynı anahtarları** taşımalı: `id, name, status, timeframe, winRatePct, profitFactor, maxDrawdownPct, totalTrades, netPnlSol, lastSignal`. Sapma = bug.
- **Hibrit kural:** Canlı endpoint runtime'da mock'a DÜŞMEZ. Yalnız `LIVE_ENDPOINTS` dışındaki metotlar mock kullanır.
- **Auth yok** (public read-only). **CORS** yalnız `CORS_ORIGIN` env'ine izin verir.
- **UTF-8:** seed verisi Türkçe karakter içerir (ör. "Güvenli Graduation", "43 dk önce") — Postgres UTF-8, Go string'leri UTF-8; sorun yok.
- **Regresyon yok:** mevcut frontend test paketi (175/175) yeşil kalmalı; `NEXT_PUBLIC_DATA_SOURCE` set değilken davranış değişmez (mock).
- **Clean code + SOLID:** SRP (config/store/api ayrı), OCP (`LIVE_ENDPOINTS` + goose migration), DIP (handler `StrategyStore` interface'ine bağımlı; frontend `getApi()` görür), küçük dosyalar.
- **Test dizini:** Go testleri kaynak yanında (`_test.go`), `apps/api-go`'dan `go test ./...`. Frontend testleri `apps/web`'den `npx vitest run`.

---

### Task 1: Go modülü + config + health endpoint (walking skeleton)

**Files:**
- Create: `apps/api-go/go.mod`, `apps/api-go/cmd/server/main.go`, `apps/api-go/internal/config/config.go`, `apps/api-go/internal/api/router.go`, `apps/api-go/internal/api/router_test.go`

**Interfaces:**
- Produces: `config.Load() config.Config` (alanlar `Port, DatabaseURL, CORSOrigin string`); `api.NewRouter(corsOrigin string) http.Handler` (Task 3'te store parametresi eklenecek); `GET /healthz` → `200 {"status":"ok"}`.

- [ ] **Step 1: Modülü başlat + chi ekle**

Run (apps/api-go dizininde):
```bash
cd apps/api-go
go mod init github.com/furkanatesc/sentinel/apps/api-go
go get github.com/go-chi/chi/v5@v5.1.0
```

- [ ] **Step 2: config paketi**

Create `apps/api-go/internal/config/config.go`:
```go
package config

import "os"

// Config, servisin env ile yapılandırmasıdır (SRP: yalnız yapılandırma).
type Config struct {
	Port        string
	DatabaseURL string
	CORSOrigin  string
}

func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		CORSOrigin:  os.Getenv("CORS_ORIGIN"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

- [ ] **Step 3: Failing test yaz (health endpoint)**

Create `apps/api-go/internal/api/router_test.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer(NewRouter(""))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}
```

- [ ] **Step 4: Testi çalıştır, FAIL doğrula**

Run: `cd apps/api-go && go test ./internal/api/`
Expected: FAIL — `NewRouter` tanımlı değil (derlenmez).

- [ ] **Step 5: router + writeJSON + health handler**

Create `apps/api-go/internal/api/router.go`:
```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter, HTTP yönlendiricisini kurar. (Task 3'te StrategyStore parametresi eklenecek.)
func NewRouter(corsOrigin string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(corsOrigin))
	r.Get("/healthz", healthHandler)
	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// corsMiddleware, yalnız verilen origin'e izin verir (boşsa header eklemez).
func corsMiddleware(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 6: main.go (graceful shutdown)**

Create `apps/api-go/cmd/server/main.go`:
```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/api"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewRouter(cfg.CORSOrigin),
	}

	go func() {
		logger.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	logger.Info("stopped")
}
```

- [ ] **Step 7: Testi çalıştır, PASS + build doğrula**

Run: `cd apps/api-go && go build ./... && go test ./...`
Expected: build OK, `TestHealthz` PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/go.mod apps/api-go/go.sum apps/api-go/cmd apps/api-go/internal
git commit -m "feat(api-go): walking skeleton — config + health endpoint"
```

---

### Task 2: Store katmanı (interface + StrategyRow struct + seed + fake)

**Files:**
- Create: `apps/api-go/internal/store/store.go`, `apps/api-go/internal/store/seed.go`, `apps/api-go/internal/store/fake.go`, `apps/api-go/internal/store/store_test.go`

**Interfaces:**
- Produces: `store.StrategyRow` struct (JSON tag'leri frontend `StrategyRow` ile birebir); `store.StrategyStore` interface (`List(ctx) ([]StrategyRow, error)`); `store.SeedRows() []StrategyRow` (6 satır); `store.NewFakeStore(rows []StrategyRow, err error) StrategyStore`.

- [ ] **Step 1: Failing test yaz**

Create `apps/api-go/internal/store/store_test.go`:
```go
package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSeedRows(t *testing.T) {
	rows := SeedRows()
	if len(rows) != 6 {
		t.Fatalf("SeedRows len = %d, want 6", len(rows))
	}
	if rows[0].ID != "momentum-scalp" || rows[0].Status != "live" {
		t.Fatalf("row[0] = %+v", rows[0])
	}
}

func TestStrategyRowJSONKeys(t *testing.T) {
	b, err := json.Marshal(SeedRows()[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, key := range []string{
		"id", "name", "status", "timeframe", "winRatePct", "profitFactor",
		"maxDrawdownPct", "totalTrades", "netPnlSol", "lastSignal",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q (contract drift)", key)
		}
	}
}

func TestFakeStoreList(t *testing.T) {
	st := NewFakeStore(SeedRows(), nil)
	rows, err := st.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("List len = %d, want 6", len(rows))
	}
}
```

- [ ] **Step 2: Testi çalıştır, FAIL doğrula**

Run: `cd apps/api-go && go test ./internal/store/`
Expected: FAIL — `SeedRows`/`StrategyRow`/`NewFakeStore` tanımlı değil.

- [ ] **Step 3: store.go (struct + interface)**

Create `apps/api-go/internal/store/store.go`:
```go
package store

import "context"

// StrategyRow, frontend StrategyRow (apps/web/lib/api/types.ts) ile birebir JSON şeklidir.
type StrategyRow struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Timeframe      string  `json:"timeframe"`
	WinRatePct     float64 `json:"winRatePct"`
	ProfitFactor   float64 `json:"profitFactor"`
	MaxDrawdownPct float64 `json:"maxDrawdownPct"`
	TotalTrades    int     `json:"totalTrades"`
	NetPnlSol      float64 `json:"netPnlSol"`
	LastSignal     string  `json:"lastSignal"`
}

// StrategyStore, strateji listesi kaynağıdır (DIP: handler bu interface'e bağımlı).
type StrategyStore interface {
	List(ctx context.Context) ([]StrategyRow, error)
}
```

- [ ] **Step 4: seed.go (6 satır — mock ile birebir)**

Create `apps/api-go/internal/store/seed.go`:
```go
package store

// SeedRows, frontend mock.ts strateji verisinden türetilmiş deterministik 6 satırdır
// (aynı id'ler + aynı türetilmiş metrikler → mock|http görsel eşitliği).
func SeedRows() []StrategyRow {
	return []StrategyRow{
		{ID: "momentum-scalp", Name: "Momentum Scalp", Status: "live", Timeframe: "1-5 dk", WinRatePct: 63, ProfitFactor: 1.8, MaxDrawdownPct: 16, TotalTrades: 298, NetPnlSol: 338, LastSignal: "43 dk önce"},
		{ID: "safe-graduation", Name: "Güvenli Graduation", Status: "paper", Timeframe: "15-60 dk", WinRatePct: 55, ProfitFactor: 1.5, MaxDrawdownPct: 13, TotalTrades: 370, NetPnlSol: -90, LastSignal: "56 dk önce"},
		{ID: "creator-reputation", Name: "Creator İtibar Takibi", Status: "shadow", Timeframe: "5-30 dk", WinRatePct: 61, ProfitFactor: 3.1, MaxDrawdownPct: 29, TotalTrades: 336, NetPnlSol: 276, LastSignal: "9 dk önce"},
		{ID: "liquidity-breakout", Name: "Likidite Kırılımı", Status: "backtesting", Timeframe: "1-10 dk", WinRatePct: 61, ProfitFactor: 3.1, MaxDrawdownPct: 29, TotalTrades: 336, NetPnlSol: 276, LastSignal: "9 dk önce"},
		{ID: "anti-rug-filter", Name: "Anti-Rug Filtre", Status: "paused", Timeframe: "10-45 dk", WinRatePct: 63, ProfitFactor: 3.3, MaxDrawdownPct: 31, TotalTrades: 338, NetPnlSol: 378, LastSignal: "24 dk önce"},
		{ID: "legacy-sniper", Name: "Eski Sniper v1", Status: "archived", Timeframe: "0-2 dk", WinRatePct: 56, ProfitFactor: 1.6, MaxDrawdownPct: 14, TotalTrades: 171, NetPnlSol: 211, LastSignal: "34 dk önce"},
	}
}
```

- [ ] **Step 5: fake.go (test store)**

Create `apps/api-go/internal/store/fake.go`:
```go
package store

import "context"

type fakeStore struct {
	rows []StrategyRow
	err  error
}

// NewFakeStore, testler için in-memory StrategyStore döndürür (err set edilirse List onu döner).
func NewFakeStore(rows []StrategyRow, err error) StrategyStore {
	return &fakeStore{rows: rows, err: err}
}

func (f *fakeStore) List(context.Context) ([]StrategyRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}
```

- [ ] **Step 6: Testi çalıştır, PASS doğrula**

Run: `cd apps/api-go && go test ./internal/store/`
Expected: 3 test PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/store
git commit -m "feat(api-go): StrategyStore interface + seed rows + fake"
```

---

### Task 3: Strategies HTTP handler (store'a bağlı) + CORS

**Files:**
- Modify: `apps/api-go/internal/api/router.go` (NewRouter store parametresi + /api/strategies), `apps/api-go/cmd/server/main.go` (fake seed store geçir)
- Create: `apps/api-go/internal/api/strategies.go`
- Modify: `apps/api-go/internal/api/router_test.go` (NewRouter çağrısı + yeni testler)

**Interfaces:**
- Consumes: `store.StrategyStore`, `store.SeedRows`, `store.NewFakeStore`.
- Produces: `api.NewRouter(st store.StrategyStore, corsOrigin string) http.Handler`; `GET /api/strategies` → `200 []StrategyRow` / store hatası → `500 {"error":...}`.

- [ ] **Step 1: Testi güncelle (failing)**

Replace `apps/api-go/internal/api/router_test.go` içeriğini:
```go
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func newTestServer(st store.StrategyStore, origin string) *httptest.Server {
	return httptest.NewServer(NewRouter(st, origin))
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(store.NewFakeStore(nil, nil), "")
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestStrategiesOK(t *testing.T) {
	srv := newTestServer(store.NewFakeStore(store.SeedRows(), nil), "https://app.example")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/strategies")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("CORS origin = %q", got)
	}
	var rows []store.StrategyRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("rows = %d, want 6", len(rows))
	}
}

func TestStrategiesStoreError(t *testing.T) {
	srv := newTestServer(store.NewFakeStore(nil, errors.New("db down")), "")
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/strategies")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Testi çalıştır, FAIL doğrula**

Run: `cd apps/api-go && go test ./internal/api/`
Expected: FAIL — `NewRouter` imzası (1 arg) uyuşmuyor; derlenmez.

- [ ] **Step 3: strategies handler**

Create `apps/api-go/internal/api/strategies.go`:
```go
package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func strategiesHandler(st store.StrategyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := st.List(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "strategies unavailable"})
			return
		}
		if rows == nil {
			rows = []store.StrategyRow{}
		}
		writeJSON(w, http.StatusOK, rows)
	}
}
```

- [ ] **Step 4: router.go — NewRouter store parametresi + mount**

Modify `apps/api-go/internal/api/router.go` — `NewRouter` imzasını ve gövdesini güncelle:
```go
func NewRouter(st store.StrategyStore, corsOrigin string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(corsOrigin))
	r.Get("/healthz", healthHandler)
	r.Get("/api/strategies", strategiesHandler(st))
	return r
}
```
Ve import bloğuna ekle:
```go
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
```

- [ ] **Step 5: main.go — store geçir**

Modify `apps/api-go/cmd/server/main.go`: import'a `store` ekle ve `NewRouter` çağrısını güncelle (Task 4'te postgres ile değişecek):
```go
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
```
```go
	st := store.NewFakeStore(store.SeedRows(), nil)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewRouter(st, cfg.CORSOrigin),
	}
```

- [ ] **Step 6: Testi çalıştır, PASS + build doğrula**

Run: `cd apps/api-go && go build ./... && go test ./...`
Expected: tüm testler PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal apps/api-go/cmd
git commit -m "feat(api-go): GET /api/strategies handler + CORS"
```

---

### Task 4: Postgres store + goose migration + seed-on-startup

**Files:**
- Create: `apps/api-go/internal/store/postgres.go`, `apps/api-go/internal/store/migrations/0001_create_strategies.sql`, `apps/api-go/internal/store/migrate.go`, `apps/api-go/internal/store/postgres_test.go`

> **Kritik:** Go `//go:embed` **üst dizine (`..`) referans veremez** — migration klasörü store paketinin İÇİNDE (`internal/store/migrations/`) olmalı, yoksa derlenmez.
- Modify: `apps/api-go/cmd/server/main.go` (DATABASE_URL varsa postgres, yoksa fake)

**Interfaces:**
- Consumes: `store.StrategyRow`, `store.SeedRows`, `store.StrategyStore`.
- Produces: `store.OpenPostgres(ctx, dsn string) (StrategyStore, func() error, error)` (migration + seed dahil).

- [ ] **Step 1: Bağımlılıkları ekle**

Run:
```bash
cd apps/api-go
go get github.com/jackc/pgx/v5/stdlib@v5.6.0
go get github.com/pressly/goose/v3@v3.21.1
```

- [ ] **Step 2: Migration SQL**

Create `apps/api-go/internal/store/migrations/0001_create_strategies.sql`:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS strategies (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    status           TEXT NOT NULL,
    timeframe        TEXT NOT NULL,
    win_rate_pct     DOUBLE PRECISION NOT NULL,
    profit_factor    DOUBLE PRECISION NOT NULL,
    max_drawdown_pct DOUBLE PRECISION NOT NULL,
    total_trades     INTEGER NOT NULL,
    net_pnl_sol      DOUBLE PRECISION NOT NULL,
    last_signal      TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS strategies;
```

- [ ] **Step 3: Failing integration test yaz (DATABASE_URL yoksa skip)**

Create `apps/api-go/internal/store/postgres_test.go`:
```go
package store

import (
	"context"
	"os"
	"testing"
)

func TestPostgresListSeeded(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL unset — postgres integration test atlandı")
	}
	ctx := context.Background()
	st, cleanup, err := OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	defer cleanup()

	rows, err := st.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 6 {
		t.Fatalf("rows = %d, want 6 (seed)", len(rows))
	}
}
```

- [ ] **Step 4: Testi çalıştır, FAIL/SKIP doğrula**

Run: `cd apps/api-go && go test ./internal/store/ -run TestPostgres`
Expected: `OpenPostgres` tanımsız → derlenmez (FAIL). (DATABASE_URL set edilirse Step 6 sonrası SKIP yerine PASS.)

- [ ] **Step 5: migrate.go (goose embed)**

Create `apps/api-go/internal/store/migrate.go` (embed yolu paket dizinine görelidir; `migrations/` store paketinin içinde olduğu için `..` yok):
```go
package store

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}
```

- [ ] **Step 6: postgres.go (open + migrate + seed + List)**

Create `apps/api-go/internal/store/postgres.go`:
```go
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresStore struct {
	db *sql.DB
}

// OpenPostgres, bağlantı açar, migration'ları çalıştırır, seed'i idempotent uygular
// ve StrategyStore ile bir kapatma fonksiyonu döner.
func OpenPostgres(ctx context.Context, dsn string) (StrategyStore, func() error, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, nil, fmt.Errorf("ping: %w", err)
	}
	if err := runMigrations(db); err != nil {
		return nil, nil, fmt.Errorf("migrate: %w", err)
	}
	if err := seedStrategies(ctx, db); err != nil {
		return nil, nil, fmt.Errorf("seed: %w", err)
	}
	return &postgresStore{db: db}, db.Close, nil
}

func seedStrategies(ctx context.Context, db *sql.DB) error {
	const q = `INSERT INTO strategies
		(id, name, status, timeframe, win_rate_pct, profit_factor, max_drawdown_pct, total_trades, net_pnl_sol, last_signal)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO NOTHING`
	for _, s := range SeedRows() {
		if _, err := db.ExecContext(ctx, q, s.ID, s.Name, s.Status, s.Timeframe,
			s.WinRatePct, s.ProfitFactor, s.MaxDrawdownPct, s.TotalTrades, s.NetPnlSol, s.LastSignal); err != nil {
			return err
		}
	}
	return nil
}

func (p *postgresStore) List(ctx context.Context) ([]StrategyRow, error) {
	const q = `SELECT id, name, status, timeframe, win_rate_pct, profit_factor,
		max_drawdown_pct, total_trades, net_pnl_sol, last_signal FROM strategies ORDER BY id`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StrategyRow
	for rows.Next() {
		var s StrategyRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.Timeframe, &s.WinRatePct,
			&s.ProfitFactor, &s.MaxDrawdownPct, &s.TotalTrades, &s.NetPnlSol, &s.LastSignal); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
```

- [ ] **Step 7: main.go — DATABASE_URL varsa postgres**

Modify `apps/api-go/cmd/server/main.go` — store seçimini güncelle (fake fallback yerel geliştirme için):
```go
	var st store.StrategyStore
	var cleanup func() error = func() error { return nil }
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		pst, cl, err := store.OpenPostgres(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			logger.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		st, cleanup = pst, cl
	} else {
		logger.Warn("DATABASE_URL yok — in-memory fake store kullanılıyor")
		st = store.NewFakeStore(store.SeedRows(), nil)
	}
	defer cleanup()
```
Ve `srv.Handler`'ı `api.NewRouter(st, cfg.CORSOrigin)` olarak bırak (değişmedi).

- [ ] **Step 8: Testler + build doğrula**

Run: `cd apps/api-go && go build ./... && go test ./...`
Expected: build OK; `TestPostgres...` DATABASE_URL yoksa SKIP, diğerleri PASS. (Yerel Postgres varsa `DATABASE_URL=... go test ./internal/store/ -run TestPostgres` → PASS.)

- [ ] **Step 9: Commit**

```bash
git add apps/api-go
git commit -m "feat(api-go): Postgres store + goose migration + seed-on-startup"
```

---

### Task 5: Frontend httpApi.getStrategies (gerçek fetch)

**Files:**
- Modify: `apps/web/lib/api/http.ts`
- Create: `apps/web/lib/api/http.test.ts`

**Interfaces:**
- Consumes: `StrategyRow` (`@/lib/api/types`), env `NEXT_PUBLIC_API_BASE_URL`.
- Produces: `httpApi.getStrategies(): Promise<StrategyRow[]>` (gerçek); diğer metotlar `notReady` kalır.

- [ ] **Step 1: Failing test yaz**

Create `apps/web/lib/api/http.test.ts`:
```ts
import { beforeEach, afterEach, it, expect, vi } from "vitest";
import { httpApi } from "./http";

const OLD = process.env.NEXT_PUBLIC_API_BASE_URL;
beforeEach(() => { process.env.NEXT_PUBLIC_API_BASE_URL = "https://api.test"; });
afterEach(() => {
  process.env.NEXT_PUBLIC_API_BASE_URL = OLD;
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

const sample = [{
  id: "momentum-scalp", name: "Momentum Scalp", status: "live", timeframe: "1-5 dk",
  winRatePct: 63, profitFactor: 1.8, maxDrawdownPct: 16, totalTrades: 298, netPnlSol: 338, lastSignal: "43 dk önce",
}];

it("getStrategies API JSON'unu StrategyRow[]'a maple", async () => {
  vi.stubGlobal("fetch", vi.fn(async () =>
    new Response(JSON.stringify(sample), { status: 200, headers: { "content-type": "application/json" } })));
  const rows = await httpApi.getStrategies();
  expect(rows).toEqual(sample);
});

it("getStrategies non-200'de reject eder", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response("boom", { status: 500 })));
  await expect(httpApi.getStrategies()).rejects.toThrow(/500/);
});

it("getStrategies API base yoksa reject eder", async () => {
  delete process.env.NEXT_PUBLIC_API_BASE_URL;
  await expect(httpApi.getStrategies()).rejects.toThrow(/NEXT_PUBLIC_API_BASE_URL/);
});
```

- [ ] **Step 2: Testi çalıştır, FAIL doğrula**

Run: `cd apps/web && npx vitest run lib/api/http.test.ts`
Expected: FAIL — `getStrategies` hâlâ `notReady` (reject "not implemented"), map testi geçmez.

- [ ] **Step 3: http.ts — getStrategies gerçek**

Modify `apps/web/lib/api/http.ts` — dosyanın başına import + yardımcılar ekle ve `getStrategies` satırını değiştir:
```ts
import type { SentinelApi } from "./contract";
import type { StrategyRow } from "./types";

const notReady = () => Promise.reject(new Error("httpApi not implemented — backend not connected yet"));

function apiBase(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!base) throw new Error("NEXT_PUBLIC_API_BASE_URL is not set");
  return base.replace(/\/$/, "");
}

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(`${apiBase()}${path}`, { headers: { accept: "application/json" } });
  if (!res.ok) throw new Error(`API ${path} failed: ${res.status}`);
  return (await res.json()) as T;
}
```
Ve `httpApi` nesnesinde:
```ts
  getStrategies: () => getJson<StrategyRow[]>("/api/strategies"),
```
(Diğer tüm metotlar `notReady` / `subscribe*` no-op olarak kalır.)

- [ ] **Step 4: Testi çalıştır, PASS doğrula**

Run: `cd apps/web && npx vitest run lib/api/http.test.ts`
Expected: 3 test PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/api/http.ts apps/web/lib/api/http.test.ts
git commit -m "feat(web): httpApi.getStrategies real fetch"
```

---

### Task 6: Frontend hibrit getApi() + LIVE_ENDPOINTS

**Files:**
- Create: `apps/web/lib/api/live-endpoints.ts`, `apps/web/lib/api/index.test.ts`
- Modify: `apps/web/lib/api/index.ts`

**Interfaces:**
- Consumes: `mockApi`, `httpApi`, `SentinelApi`, `LIVE_ENDPOINTS`.
- Produces: `getApi(): SentinelApi` (hibrit); `LIVE_ENDPOINTS: Set<keyof SentinelApi>`.

- [ ] **Step 1: Failing test yaz**

Create `apps/web/lib/api/index.test.ts`:
```ts
import { afterEach, it, expect } from "vitest";
import { getApi } from "./index";
import { mockApi } from "./mock";
import { httpApi } from "./http";

const OLD = process.env.NEXT_PUBLIC_DATA_SOURCE;
afterEach(() => { process.env.NEXT_PUBLIC_DATA_SOURCE = OLD; });

it("mock modunda tüm endpoint'ler mockApi'den", () => {
  process.env.NEXT_PUBLIC_DATA_SOURCE = "mock";
  expect(getApi().getStrategies).toBe(mockApi.getStrategies);
  expect(getApi().getTokens).toBe(mockApi.getTokens);
});

it("http modunda canlı endpoint httpApi'den, kalan mockApi'den", () => {
  process.env.NEXT_PUBLIC_DATA_SOURCE = "http";
  expect(getApi().getStrategies).toBe(httpApi.getStrategies);
  expect(getApi().getTokens).toBe(mockApi.getTokens);
});
```

- [ ] **Step 2: Testi çalıştır, FAIL doğrula**

Run: `cd apps/web && npx vitest run lib/api/index.test.ts`
Expected: FAIL — http modunda `getApi()` şu an `httpApi`'yi tümden döndürüyor; `getTokens` mock değil.

- [ ] **Step 3: live-endpoints.ts**

Create `apps/web/lib/api/live-endpoints.ts`:
```ts
import type { SentinelApi } from "./contract";

// Backend'de gerçekleşmiş endpoint'ler. Her backend alt-projesi buraya ekler (OCP).
export const LIVE_ENDPOINTS = new Set<keyof SentinelApi>(["getStrategies"]);
```

- [ ] **Step 4: index.ts — hibrit adapter**

Replace `apps/web/lib/api/index.ts` içeriğini:
```ts
import type { SentinelApi } from "./contract";
import { mockApi } from "./mock";
import { httpApi } from "./http";
import { LIVE_ENDPOINTS } from "./live-endpoints";

/**
 * Hibrit adapter: DATA_SOURCE=http iken LIVE_ENDPOINTS'teki metotlar gerçek httpApi'ye,
 * kalanı mockApi'ye bağlanır. Canlı endpoint runtime'da mock'a DÜŞMEZ. DATA_SOURCE!=http
 * iken her şey mock. Bileşenler yalnız getApi() görür (DIP).
 */
export function getApi(): SentinelApi {
  if (process.env.NEXT_PUBLIC_DATA_SOURCE !== "http") return mockApi;
  return new Proxy(mockApi, {
    get(target, prop) {
      if (typeof prop === "string" && LIVE_ENDPOINTS.has(prop as keyof SentinelApi)) {
        return httpApi[prop as keyof SentinelApi];
      }
      return target[prop as keyof SentinelApi];
    },
  }) as SentinelApi;
}

export type { SentinelApi };
```

- [ ] **Step 5: Testi çalıştır, PASS doğrula**

Run: `cd apps/web && npx vitest run lib/api/index.test.ts`
Expected: 2 test PASS.

- [ ] **Step 6: Tam frontend paketi — regresyon yok**

Run: `cd apps/web && npx vitest run`
Expected: tüm testler PASS (175 mevcut + 5 yeni = 180). Mevcut ekranlar (DATA_SOURCE set değil → mock) etkilenmez.

- [ ] **Step 7: Commit**

```bash
git add apps/web/lib/api/index.ts apps/web/lib/api/live-endpoints.ts apps/web/lib/api/index.test.ts
git commit -m "feat(web): hybrid getApi() adapter with LIVE_ENDPOINTS"
```

---

### Task 7: Env + CI + deploy dokümanı + build doğrulama

**Files:**
- Modify: `apps/web/.env.example`
- Create: `apps/api-go/README.md`, `.github/workflows/api-go.yml`, `apps/api-go/.gitignore`

**Interfaces:** yok (yapılandırma + dokümantasyon).

- [ ] **Step 1: .env.example güncelle**

Modify `apps/web/.env.example` — ekle:
```
# http modunda backend API tabanı (Railway URL). Boşsa mock kalır.
NEXT_PUBLIC_API_BASE_URL=
```
(Mevcut `NEXT_PUBLIC_DATA_SOURCE=mock` satırı kalır; canlı için Vercel'de `http` + base URL set edilir.)

- [ ] **Step 2: apps/api-go/.gitignore**

Create `apps/api-go/.gitignore`:
```
/bin/
*.exe
```

- [ ] **Step 3: CI workflow**

Create `.github/workflows/api-go.yml`:
```yaml
name: api-go
on:
  push:
    paths: ["apps/api-go/**", ".github/workflows/api-go.yml"]
  pull_request:
    paths: ["apps/api-go/**"]
jobs:
  build-test:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: apps/api-go
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - run: go build ./...
      - run: go test ./...
```

- [ ] **Step 4: README (local run + Railway deploy)**

Create `apps/api-go/README.md`:
```markdown
# SENTINEL API (Go)

Backend Alt-proje 0 — platform iskeleti. Şimdilik tek gerçek endpoint: `GET /api/strategies`.

## Yerel çalıştırma
```bash
cd apps/api-go
go run ./cmd/server           # DATABASE_URL yoksa in-memory fake store
# Postgres ile:
DATABASE_URL=postgres://user:pass@localhost:5432/sentinel PORT=8080 \
  CORS_ORIGIN=http://localhost:3000 go run ./cmd/server
```
`GET http://localhost:8080/healthz` → `{"status":"ok"}`
`GET http://localhost:8080/api/strategies` → 6 strateji.

## Railway deploy (KULLANICI ADIMI)
1. Railway'de yeni servis → GitHub repo `furkanatesc/sentinel`, **Root Directory = `apps/api-go`** (nixpacks Go'yu otomatik derler; start komutu binary'yi çalıştırır).
2. Servise **PostgreSQL** eklentisi ekle → `DATABASE_URL` otomatik enjekte olur.
3. Env: `CORS_ORIGIN=https://sentinel-brown-alpha.vercel.app`.
4. Deploy sonrası `<railway-url>/healthz` ve `/api/strategies` doğrula.

## Frontend'i bağlama (KULLANICI ADIMI)
Vercel projesine env ekle: `NEXT_PUBLIC_API_BASE_URL=<railway-url>`, `NEXT_PUBLIC_DATA_SOURCE=http` → redeploy.
Strategies ekranı gerçek API'den, diğer ekranlar mock ile çalışır (hibrit).
```

- [ ] **Step 5: Go + frontend doğrula**

Run:
```bash
cd apps/api-go && go build ./... && go test ./...
cd ../web && npx vitest run
```
Expected: Go build+test PASS; frontend 180 test PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/web/.env.example apps/api-go/README.md apps/api-go/.gitignore .github/workflows/api-go.yml
git commit -m "chore(api-go): CI + env + deploy docs"
```

---

### Task 8: Yaşayan dokümanlar + kullanıcı deploy checkpoint

**Files:**
- Modify: `docs/progress.md` (Alt-proje 0 durumunu ✅/deploy-bekliyor yap), memory `sentinel-backend-program.md`

- [ ] **Step 1: progress.md güncelle**

`docs/progress.md` → Backend programı tablosunda Alt-proje 0 satırını implementasyon tamam olarak işaretle (kod tamam, kullanıcı deploy'u bekliyor); "Sırada"yı Alt-proje 1'e çevir; karar günlüğüne kısa bir "Alt-proje 0 implementasyonu tamam" girişi ekle (commit aralığı + test sayısı). Memory `sentinel-backend-program.md` Alt-proje 0 durumunu güncelle.

- [ ] **Step 2: Commit**

```bash
git add docs/progress.md
git commit -m "docs(backend): Alt-proje 0 implementasyonu tamam"
```

- [ ] **Step 3: KULLANICI DEPLOY CHECKPOINT (DUR ve iste)**

Kod + testler tamam. Aşağıdaki adımlar kullanıcının hesabını gerektirir — **DUR, tek tek iste, "yaptın mı?" diye onay al** (global kural):
1. Railway: yeni servis (root `apps/api-go`) + PostgreSQL eklentisi + `CORS_ORIGIN` env.
2. Vercel: `NEXT_PUBLIC_API_BASE_URL=<railway-url>` + `NEXT_PUBLIC_DATA_SOURCE=http`.
3. Canlı doğrulama (birlikte): Railway `/healthz`+`/api/strategies` 200; Vercel'de Strategies ekranı gerçek API'den render, diğer 8 ekran mock ile regresyonsuz.

Bu checkpoint tamamlanınca Alt-proje 0 kapanır; whole-branch review → merge onayı (kullanıcı) → Alt-proje 1.

---

## Notlar (spec sapmaları — bilinçli, sessiz düşürme yok)

- **`/healthz` DB ping içermez** (liveness-only); DB hazırlığı `/api/strategies` üzerinden doğrulanır. Spec §3.1 "DB ping dahil" dedi — basitlik için liveness/readiness ayrımı; readiness endpoint'i gerekirse sonra eklenir.
- **goose migration'ları store paketi içinde** (`internal/store/migrations/`) — Go `//go:embed` `..` kabul etmediği için zorunlu. Migration'ın çalışması Postgres integration testinde kanıtlanır.
- **Yerel fake-store fallback** (DATABASE_URL yoksa) spec'te yoktu — geliştirme kolaylığı için eklendi; prod'da (Railway) DATABASE_URL hep set.
