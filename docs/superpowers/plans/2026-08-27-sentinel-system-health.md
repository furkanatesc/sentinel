# System Health (Dev/Ops Teşhis Paneli) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pipeline'ın canlı sağlığını (worker liveness, DB, ingest WS durumu) tek ekranda görünür kılan `GET /api/system-health` + dev/ops teşhis paneli inşa etmek.

**Architecture:** Merkezi thread-safe `health.Registry` (push): worker'lar dar `health.Reporter` arayüzüyle her cycle sonunda durum yazar; main.go tüm worker'ları (açık/kapalı) kaydeder; endpoint `Snapshot()` + istek-anı DB ping + gates + version'ı birleştirip döner. Frontend mevcut hibrit getApi seam + React Query poll desenini izler.

**Tech Stack:** Go (chi router, `database/sql`, `sync.RWMutex`), Next.js/TypeScript (React Query), Vitest.

**Spec:** `docs/superpowers/specs/2026-08-27-sentinel-system-health-design.md`

## Global Constraints

- **Clean code & SOLID:** dar arayüzler (ISP/DIP), SRP dosyalar, saf fonksiyonlar test edilebilir. (Kullanıcı sıkı bağlılık ister.)
- **Public endpoint — secret sızmaz:** RPC URL / API key ASLA snapshot'a girmez; `lastErr` sanitize edilir (`api-key=...` → `api-key=REDACTED`).
- **Best-effort telemetri:** `Report`/`Register` asla worker'ı bloklamaz/panik etmez; `Health` nil ise no-op (mevcut worker davranışı korunur, geriye-uyumlu).
- **Graceful degrade:** DB ping fail → `dbOk:false` + HTTP **200** (500 değil).
- **Go modül yolu:** `github.com/furkanatesc/sentinel/apps/api-go/...`
- **State enum (5):** `off | starting | ok | degraded | stalled` (spec §3.3).
- **Testler `go test ./... -race` ile yeşil kalır; mevcut worker testleri kırılmaz** (nil-guard sayesinde `Health` opsiyonel).
- Commit sonu satırı: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## File Structure

**Backend — yeni:**
- `apps/api-go/internal/health/registry.go` — Registry + Reporter + State + WorkerStatus + name consts.
- `apps/api-go/internal/health/registry_test.go` — birim testler (white-box, package health).
- `apps/api-go/internal/api/system_health.go` — handler + response struct.
- `apps/api-go/internal/api/system_health_test.go` — handler testi.

**Backend — değişecek:**
- `apps/api-go/internal/store/postgres.go` — `Pinger` arayüzü + `postgresStore.Ping` + `Bundle.Pinger`.
- `apps/api-go/internal/store/fake_ingest.go` — `fakeTokenStore.Ping`.
- `apps/api-go/internal/api/router.go` — RouterDeps yeni alanlar + route.
- `apps/api-go/internal/config/config.go` — `Version` alanı.
- `apps/api-go/cmd/server/main.go` — registry oluştur, tüm worker'ları Register, reporter'ı worker deps'lerine geçir, router deps.
- Her worker: `internal/{ingest,market,safety,outcome,creatorfill,walletgraph,reputation,manipulation,opportunity}/*.go` — `Health health.Reporter` alanı + cycle sonu `Report`.

**Frontend — değişecek:**
- `apps/web/lib/api/types.ts` — `SystemHealth` + `WorkerStatus` tipleri.
- `apps/web/lib/api/contract.ts` — `getSystemHealth()`.
- `apps/web/lib/api/mock.ts` — temsili mock.
- `apps/web/lib/api/http.ts` — `getSystemHealth` httpApi.
- `apps/web/lib/api/live-endpoints.ts` — `"getSystemHealth"`.
- `apps/web/lib/get-query-client.ts` — `qk.systemHealth`.
- `apps/web/lib/hooks/queries.ts` — `useSystemHealth` (poll).
- `apps/web/app/(app)/system-health/page.tsx` — panel (PlaceholderScreen → gerçek).
- `apps/web/lib/api/mock.test.ts` — mock şekil testi.

---

## PHASE A — Backend core (endpoint kayıt-bazlı canlı)

### Task 1: `internal/health` paketi — Registry + Reporter + State

**Files:**
- Create: `apps/api-go/internal/health/registry.go`
- Test: `apps/api-go/internal/health/registry_test.go`

**Interfaces:**
- Produces:
  - `type State string` + consts `StateOff/StateStarting/StateOK/StateDegraded/StateStalled`
  - Worker adı consts: `WorkerIngestWS="ingest-ws"`, `WorkerMarketDisc="market-disc"`, `WorkerMarketEnrich="market-enrich"`, `WorkerSafety="safety"`, `WorkerOutcome="outcome"`, `WorkerCreatorFill="creatorfill"`, `WorkerFunder="funder"`, `WorkerReputation="reputation"`, `WorkerManipulation="manipulation"`, `WorkerOpportunity="opportunity"`
  - `type WorkerStatus struct { Name string; State State; LastRunAt string; LastErr string; CyclesRun int; ItemsProcessed int; IntervalSec int }` (JSON: `name,state,lastRunAt,lastErr,cyclesRun,itemsProcessed,intervalSec`)
  - `type Reporter interface { Register(name string, enabled bool, interval time.Duration); Report(name string, ok bool, err error, processed int) }`
  - `type Registry struct{...}` + `func NewRegistry() *Registry`
  - `func (r *Registry) Register(name string, enabled bool, interval time.Duration)`
  - `func (r *Registry) Report(name string, ok bool, err error, processed int)`
  - `func (r *Registry) Snapshot(now time.Time) []WorkerStatus`
  - `var _ Reporter = (*Registry)(nil)`

- [ ] **Step 1: Write the failing tests**

```go
package health

import (
	"errors"
	"testing"
	"time"
)

func TestOffWhenDisabled(t *testing.T) {
	r := NewRegistry()
	r.Register("w", false, 30*time.Second)
	got := stateOf(t, r, "w", time.Now())
	if got != StateOff {
		t.Fatalf("state = %q, want off", got)
	}
}

func TestStartingBeforeFirstCycleWithinGrace(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	// no Report yet; 30s later still within 3×interval grace
	got := stateOf(t, r, "w", base.Add(30*time.Second))
	if got != StateStarting {
		t.Fatalf("state = %q, want starting", got)
	}
}

func TestStalledWhenNeverRanPastGrace(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	got := stateOf(t, r, "w", base.Add(200*time.Second)) // > 3×30s
	if got != StateStalled {
		t.Fatalf("state = %q, want stalled", got)
	}
}

func TestOKAfterHealthyReport(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	r.Report("w", true, nil, 5)
	got := stateOf(t, r, "w", base.Add(10*time.Second))
	if got != StateOK {
		t.Fatalf("state = %q, want ok", got)
	}
}

func TestDegradedWhenLastCycleFailed(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	r.Report("w", false, errors.New("boom"), 0)
	got := stateOf(t, r, "w", base.Add(10*time.Second))
	if got != StateDegraded {
		t.Fatalf("state = %q, want degraded", got)
	}
}

func TestStalledAfterRunButStale(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("w", true, 30*time.Second)
	r.Report("w", true, nil, 1)
	got := stateOf(t, r, "w", base.Add(200*time.Second)) // last run 200s ago > 90s
	if got != StateStalled {
		t.Fatalf("state = %q, want stalled", got)
	}
}

func TestIntervalZeroNeverTimeStalls(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	r := NewRegistry()
	r.now = func() time.Time { return base }
	r.Register("ingest-ws", true, 0)
	// even far in the future, no report → starting (event-driven, no time-stall)
	got := stateOf(t, r, "ingest-ws", base.Add(time.Hour))
	if got != StateStarting {
		t.Fatalf("state = %q, want starting", got)
	}
	r.Report("ingest-ws", true, nil, 0)
	if s := stateOf(t, r, "ingest-ws", base.Add(2*time.Hour)); s != StateOK {
		t.Fatalf("after report state = %q, want ok", s)
	}
}

func TestLastErrSanitizesAPIKey(t *testing.T) {
	r := NewRegistry()
	r.Register("w", true, 30*time.Second)
	r.Report("w", false, errors.New("get https://mainnet.helius-rpc.com/?api-key=SECRET123 failed"), 0)
	ws := findWorker(t, r.Snapshot(time.Now()), "w")
	if want := "api-key=REDACTED"; !contains(ws.LastErr, want) {
		t.Fatalf("lastErr = %q, want to contain %q", ws.LastErr, want)
	}
	if contains(ws.LastErr, "SECRET123") {
		t.Fatalf("lastErr leaked secret: %q", ws.LastErr)
	}
}

func TestSnapshotPreservesRegistrationOrderAndCounts(t *testing.T) {
	r := NewRegistry()
	r.Register("a", true, time.Second)
	r.Register("b", true, time.Second)
	r.Report("a", true, nil, 3)
	r.Report("a", true, nil, 2)
	snap := r.Snapshot(time.Now())
	if len(snap) != 2 || snap[0].Name != "a" || snap[1].Name != "b" {
		t.Fatalf("order/len wrong: %+v", snap)
	}
	if snap[0].CyclesRun != 2 || snap[0].ItemsProcessed != 5 {
		t.Fatalf("counts wrong: %+v", snap[0])
	}
}

func TestConcurrentReportsRace(t *testing.T) {
	r := NewRegistry()
	r.Register("w", true, time.Second)
	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				r.Report("w", true, nil, 1)
				_ = r.Snapshot(time.Now())
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}

// --- test helpers ---
func stateOf(t *testing.T, r *Registry, name string, now time.Time) State {
	t.Helper()
	return findWorker(t, r.Snapshot(now), name).State
}
func findWorker(t *testing.T, snap []WorkerStatus, name string) WorkerStatus {
	t.Helper()
	for _, w := range snap {
		if w.Name == name {
			return w
		}
	}
	t.Fatalf("worker %q not in snapshot", name)
	return WorkerStatus{}
}
func contains(s, sub string) bool { return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/api-go && go test ./internal/health/ -run . -v`
Expected: FAIL (package/`NewRegistry` undefined).

- [ ] **Step 3: Implement `registry.go`**

```go
// Package health, worker/altsistem canlılığını toplayan hafif telemetri katmanıdır
// (push modeli): worker'lar Reporter ile durum yazar, endpoint Snapshot okur.
package health

import (
	"regexp"
	"sync"
	"time"
)

type State string

const (
	StateOff      State = "off"
	StateStarting State = "starting"
	StateOK       State = "ok"
	StateDegraded State = "degraded"
	StateStalled  State = "stalled"
)

// Worker adı sabitleri — main.go (Register) + worker'lar (Report) AYNI adı kullansın (DRY).
const (
	WorkerIngestWS     = "ingest-ws"
	WorkerMarketDisc   = "market-disc"
	WorkerMarketEnrich = "market-enrich"
	WorkerSafety       = "safety"
	WorkerOutcome      = "outcome"
	WorkerCreatorFill  = "creatorfill"
	WorkerFunder       = "funder"
	WorkerReputation   = "reputation"
	WorkerManipulation = "manipulation"
	WorkerOpportunity  = "opportunity"
)

// WorkerStatus, tek worker'ın türetilmiş anlık durumudur (JSON: frontend SystemHealth ile birebir).
type WorkerStatus struct {
	Name           string `json:"name"`
	State          State  `json:"state"`
	LastRunAt      string `json:"lastRunAt"`
	LastErr        string `json:"lastErr"`
	CyclesRun      int    `json:"cyclesRun"`
	ItemsProcessed int    `json:"itemsProcessed"`
	IntervalSec    int    `json:"intervalSec"`
}

// Reporter, worker'lara enjekte edilen dar yazma arayüzüdür (ISP/DIP).
type Reporter interface {
	Register(name string, enabled bool, interval time.Duration)
	Report(name string, ok bool, err error, processed int)
}

type record struct {
	enabled        bool
	interval       time.Duration
	registeredAt   time.Time
	lastRunAt      time.Time
	lastOK         bool
	lastErr        string
	cyclesRun      int
	itemsProcessed int
}

// Registry, thread-safe worker durum deposudur.
type Registry struct {
	mu    sync.RWMutex
	now   func() time.Time
	works map[string]*record
	order []string
}

func NewRegistry() *Registry {
	return &Registry{now: time.Now, works: map[string]*record{}}
}

func (r *Registry) Register(name string, enabled bool, interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.works[name]
	if !ok {
		rec = &record{registeredAt: r.now()}
		r.works[name] = rec
		r.order = append(r.order, name)
	}
	rec.enabled = enabled
	rec.interval = interval
}

func (r *Registry) Report(name string, ok bool, err error, processed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, exists := r.works[name]
	if !exists {
		rec = &record{registeredAt: r.now(), enabled: true}
		r.works[name] = rec
		r.order = append(r.order, name)
	}
	rec.lastRunAt = r.now()
	rec.cyclesRun++
	rec.itemsProcessed += processed
	rec.lastOK = ok
	if err != nil {
		rec.lastErr = sanitizeErr(err)
	} else {
		rec.lastErr = ""
	}
}

func (r *Registry) Snapshot(now time.Time) []WorkerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]WorkerStatus, 0, len(r.order))
	for _, name := range r.order {
		rec := r.works[name]
		lastRun := ""
		if !rec.lastRunAt.IsZero() {
			lastRun = rec.lastRunAt.UTC().Format(time.RFC3339)
		}
		out = append(out, WorkerStatus{
			Name: name, State: deriveState(rec, now), LastRunAt: lastRun,
			LastErr: rec.lastErr, CyclesRun: rec.cyclesRun,
			ItemsProcessed: rec.itemsProcessed, IntervalSec: int(rec.interval / time.Second),
		})
	}
	return out
}

// deriveState, kaydın anlık durumunu türetir (saf; now enjekte). Sıra önemli: off → (hiç
// çalışmadı: starting/stalled) → (çalıştı ama bayat: stalled) → degraded → ok.
func deriveState(rec *record, now time.Time) State {
	if !rec.enabled {
		return StateOff
	}
	if rec.cyclesRun == 0 {
		if rec.interval == 0 || now.Sub(rec.registeredAt) <= 3*rec.interval {
			return StateStarting
		}
		return StateStalled
	}
	if rec.interval > 0 && now.Sub(rec.lastRunAt) > 3*rec.interval {
		return StateStalled
	}
	if !rec.lastOK {
		return StateDegraded
	}
	return StateOK
}

// apiKeyRe, hata mesajlarındaki `api-key=...` değerini kırpar (public endpoint — secret sızmaz).
var apiKeyRe = regexp.MustCompile(`(?i)(api-key=)[^&\s]+`)

func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	return apiKeyRe.ReplaceAllString(err.Error(), "${1}REDACTED")
}

var _ Reporter = (*Registry)(nil)
```

- [ ] **Step 4: Run tests, verify pass**

Run: `cd apps/api-go && go test ./internal/health/ -race -v`
Expected: PASS (tümü).

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/health/
git commit -m "feat(system-health): health.Registry + Reporter + state türetimi (Task 1)"
```

---

### Task 2: `store.Pinger` seam — DB reachability probe

**Files:**
- Modify: `apps/api-go/internal/store/postgres.go` (Bundle + Ping)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fakeTokenStore.Ping)
- Test: `apps/api-go/internal/store/ping_test.go` (Create)

**Interfaces:**
- Produces: `type Pinger interface { Ping(ctx context.Context) error }`; `Bundle.Pinger Pinger`; `(*postgresStore).Ping`; `(*fakeTokenStore).Ping`.

- [ ] **Step 1: Write failing test**

```go
package store

import (
	"context"
	"testing"
)

func TestFakeStorePingOK(t *testing.T) {
	var p Pinger = NewFakeTokenStore().(Pinger)
	if err := p.Ping(context.Background()); err != nil {
		t.Fatalf("fake ping err = %v, want nil", err)
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/api-go && go test ./internal/store/ -run TestFakeStorePingOK -v`
Expected: FAIL (`Pinger` undefined).

- [ ] **Step 3: Implement**

In `postgres.go` — add interface + Ping method + Bundle field. The `Bundle` struct currently (around line 19) lists `Strategies/Events/Tokens/Creators`; add `Pinger Pinger`. In `OpenPostgres`'s final return (around line 48) change to include `Pinger: ps`:

```go
// Pinger, DB erişilebilirlik probu (health endpoint için). DIP: postgres + fake karşılar.
type Pinger interface {
	Ping(ctx context.Context) error
}

func (p *postgresStore) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
```
- `Bundle` struct'ına satır ekle: `Pinger Pinger`.
- `OpenPostgres` dönüşü: `return Bundle{Strategies: ps, Events: ps, Tokens: ps, Creators: ps, Pinger: ps}, db.Close, nil`.

In `fake_ingest.go` — add to `fakeTokenStore` (concrete type, line 70):
```go
// Ping, fake store için her zaman sağlıklı (in-memory; dürüst).
func (s *fakeTokenStore) Ping(ctx context.Context) error { return nil }
```

- [ ] **Step 4: Run tests, verify pass**

Run: `cd apps/api-go && go test ./internal/store/ -run TestFakeStorePingOK -v && go build ./...`
Expected: PASS + build OK.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/store/
git commit -m "feat(system-health): store.Pinger seam (postgres PingContext + fake) (Task 2)"
```

---

### Task 3: `GET /api/system-health` handler + router wiring

**Files:**
- Create: `apps/api-go/internal/api/system_health.go`
- Create: `apps/api-go/internal/api/system_health_test.go`
- Modify: `apps/api-go/internal/api/router.go` (RouterDeps + route)

**Interfaces:**
- Consumes: `health.WorkerStatus`, `health.Registry.Snapshot`, `store.Pinger`.
- Produces:
  - `type SystemHealth struct` (JSON: `uptimeSec,version,dbOk,dbLatencyMs,wsClients,workers,gates`)
  - `type healthSnapshotter interface { Snapshot(now time.Time) []health.WorkerStatus }`
  - `func systemHealthHandler(...) http.HandlerFunc`
  - RouterDeps yeni alanlar: `Health healthSnapshotter`, `Pinger store.Pinger`, `Gates map[string]bool`, `Version string`, `StartedAt time.Time`, `WSClientCount func() int`.

- [ ] **Step 1: Write failing test**

```go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
)

type stubSnap struct{ ws []health.WorkerStatus }

func (s stubSnap) Snapshot(time.Time) []health.WorkerStatus { return s.ws }

type stubPinger struct{ err error }

func (p stubPinger) Ping(context.Context) error { return p.err }

func TestSystemHealthShape(t *testing.T) {
	d := RouterDeps{
		Health:         stubSnap{ws: []health.WorkerStatus{{Name: "safety", State: health.StateDegraded, LastErr: "api-key=REDACTED 429"}}},
		Pinger:         stubPinger{err: nil},
		Gates:          map[string]bool{"SAFETY_ENABLED": true, "WALLET_GRAPH_ENABLED": false},
		Version:        "abc123",
		StartedAt:      time.Now().Add(-time.Minute),
		WSClientCount:  func() int { return 0 },
		Tokens:         nil,
	}
	h := systemHealthHandler(d.Health, d.Pinger, d.Gates, d.Version, d.StartedAt, d.WSClientCount)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/api/system-health", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
	var got SystemHealth
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.DBOk {
		t.Fatalf("dbOk = false, want true")
	}
	if got.UptimeSec < 59 {
		t.Fatalf("uptimeSec = %d, want ~60", got.UptimeSec)
	}
	if len(got.Workers) != 1 || got.Workers[0].Name != "safety" {
		t.Fatalf("workers wrong: %+v", got.Workers)
	}
	if got.Gates["WALLET_GRAPH_ENABLED"] != false {
		t.Fatalf("gates wrong: %+v", got.Gates)
	}
}

func TestSystemHealthDBDownStill200(t *testing.T) {
	h := systemHealthHandler(
		stubSnap{}, stubPinger{err: errors.New("conn refused")},
		map[string]bool{}, "dev", time.Now(), func() int { return 0 },
	)
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodGet, "/api/system-health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 even when DB down", rr.Code)
	}
	var got SystemHealth
	_ = json.NewDecoder(rr.Body).Decode(&got)
	if got.DBOk {
		t.Fatalf("dbOk = true, want false")
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/api-go && go test ./internal/api/ -run TestSystemHealth -v`
Expected: FAIL (`systemHealthHandler`/`SystemHealth` undefined).

- [ ] **Step 3: Implement `system_health.go`**

```go
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// SystemHealth, /api/system-health JSON şeklidir (frontend types.ts ile birebir).
type SystemHealth struct {
	UptimeSec   int                   `json:"uptimeSec"`
	Version     string                `json:"version"`
	DBOk        bool                  `json:"dbOk"`
	DBLatencyMs int                   `json:"dbLatencyMs"`
	WSClients   int                   `json:"wsClients"`
	Workers     []health.WorkerStatus `json:"workers"`
	Gates       map[string]bool       `json:"gates"`
}

// healthSnapshotter, handler'ın registry'ye dar bağımlılığıdır (DIP; *health.Registry karşılar).
type healthSnapshotter interface {
	Snapshot(now time.Time) []health.WorkerStatus
}

func systemHealthHandler(
	snap healthSnapshotter, pinger store.Pinger, gates map[string]bool,
	version string, startedAt time.Time, wsClients func() int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		workers := []health.WorkerStatus{}
		if snap != nil {
			workers = snap.Snapshot(now)
		}
		dbOk, latencyMs := probeDB(r.Context(), pinger)
		clients := 0
		if wsClients != nil {
			clients = wsClients()
		}
		if gates == nil {
			gates = map[string]bool{}
		}
		writeJSON(w, http.StatusOK, SystemHealth{
			UptimeSec: int(now.Sub(startedAt).Seconds()), Version: version,
			DBOk: dbOk, DBLatencyMs: latencyMs, WSClients: clients,
			Workers: workers, Gates: gates,
		})
	}
}

// probeDB, kısa timeout'lu bir ping atar; başarısızlık 500 değil dbOk=false döndürür (graceful).
func probeDB(ctx context.Context, pinger store.Pinger) (bool, int) {
	if pinger == nil {
		return false, 0
	}
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	start := time.Now()
	if err := pinger.Ping(pctx); err != nil {
		return false, 0
	}
	return true, int(time.Since(start).Milliseconds())
}
```

- [ ] **Step 4: Wire route in `router.go`**

Add fields to `RouterDeps` (after existing fields):
```go
	Health        healthSnapshotter
	Pinger        store.Pinger
	Gates         map[string]bool
	Version       string
	StartedAt     time.Time
	WSClientCount func() int
```
Register route in `NewRouter` (unconditional — always available; import `health` not needed in router.go, only via the interface which lives in this package). After `r.Get("/healthz", healthHandler)` add:
```go
	r.Get("/api/system-health", systemHealthHandler(d.Health, d.Pinger, d.Gates, d.Version, d.StartedAt, d.WSClientCount))
```

- [ ] **Step 5: Run tests, verify pass**

Run: `cd apps/api-go && go test ./internal/api/ -run TestSystemHealth -v && go build ./...`
Expected: PASS + build OK.

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/api/
git commit -m "feat(system-health): /api/system-health handler + router deps (Task 3)"
```

---

### Task 4: `cmd/server/main.go` wiring — registry + Register all + reporter injection + config Version

**Files:**
- Modify: `apps/api-go/internal/config/config.go` (Version alanı)
- Modify: `apps/api-go/cmd/server/main.go`
- Test: `apps/api-go/internal/config/config_test.go` (Version default assertion — mevcut teste ekleme)

**Interfaces:**
- Consumes: `health.NewRegistry`, worker adı consts, `config.Version`.
- Produces: `Config.Version` (env `RAILWAY_GIT_COMMIT_SHA`, default `"dev"`); main.go'da tüm worker'lar `reg.Register(...)` ile kayıtlı; worker deps'lerine `Health: reg` geçilmiş; `RouterDeps.Health/Pinger/Gates/Version/StartedAt/WSClientCount` dolu.

> **Not:** Bu task worker `WorkerDeps`'lerine `Health` alanı EKLENMESİNE bağlı. `Health` alanı Task 5-7'de eklenecek; ama main.go bu alanı Task 4'te set etmeye çalışırsa derlenmez. **Sıra çözümü:** Task 4'te önce config.Version + registry oluşturma + `reg.Register(...)` çağrıları + RouterDeps wiring yapılır (worker deps'e `Health` set ETMEDEN). Worker deps'e `Health: reg` geçişi her worker'ın kendi task'ında (5-7) yapılır — o task worker'ın `WorkerDeps`'ine alanı ekleyip aynı anda main.go'daki construction'a `Health: reg` ekler. Böylece her task derlenir.

- [ ] **Step 1: config Version — write failing test (append to `config_test.go`)**

```go
func TestVersionDefaultsToDev(t *testing.T) {
	t.Setenv("RAILWAY_GIT_COMMIT_SHA", "")
	if got := Load().Version; got != "dev" {
		t.Fatalf("Version = %q, want dev", got)
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/api-go && go test ./internal/config/ -run TestVersionDefaultsToDev -v`
Expected: FAIL (`Version` field yok).

- [ ] **Step 3: Add `Version` to config**

In `config.go` `Config` struct add `Version string`; in `Load()` add: `Version: getenv("RAILWAY_GIT_COMMIT_SHA", "dev"),`.

- [ ] **Step 4: Wire registry in `main.go`**

After `cfg := config.Load()` and after `hub` is created, add:
```go
	reg := health.NewRegistry()
	startedAt := time.Now()
```
Register EVERY worker (enabled flag = its gate; interval = its config interval in seconds; funder/ingest as noted). Place these `reg.Register(...)` calls at the point each worker is (or would be) started, in BOTH the enabled and disabled branches. Exact calls:
```go
	reg.Register(health.WorkerIngestWS, cfg.HeliusAPIKey != "", 0) // event-driven
	reg.Register(health.WorkerMarketDisc, cfg.MarketEnabled, time.Duration(cfg.DiscoverInterval)*time.Second)
	reg.Register(health.WorkerMarketEnrich, cfg.MarketEnabled, time.Duration(cfg.EnrichInterval)*time.Second)
	reg.Register(health.WorkerSafety, cfg.SafetyEnabled && rpcURL != "", time.Duration(cfg.SafetyIntervalSec)*time.Second)
	reg.Register(health.WorkerOutcome, cfg.OutcomeEnabled, time.Duration(cfg.OutcomeIntervalSec)*time.Second)
	reg.Register(health.WorkerCreatorFill, cfg.CreatorFillEnabled && creatorFillRPC != "", time.Duration(cfg.CreatorFillIntervalSec)*time.Second)
	reg.Register(health.WorkerFunder, cfg.WalletGraphEnabled && creatorFillRPC != "", time.Duration(cfg.FunderResolveIntervalSec)*time.Second)
	reg.Register(health.WorkerReputation, cfg.ReputationEnabled, time.Duration(cfg.ReputationIntervalSec)*time.Second)
	reg.Register(health.WorkerManipulation, cfg.ManipulationEnabled, time.Duration(cfg.ManipulationIntervalSec)*time.Second)
	reg.Register(health.WorkerOpportunity, cfg.OpportunityEnabled, time.Duration(cfg.OpportunityIntervalSec)*time.Second)
```
(Place each after the corresponding variable — e.g. `rpcURL`, `creatorFillRPC` — is defined. If simpler, group them all AFTER `creatorFillRPC` is computed, since all referenced vars exist by then.)

Build the gates map:
```go
	gates := map[string]bool{
		"MARKET_ENABLED":       cfg.MarketEnabled,
		"SAFETY_ENABLED":       cfg.SafetyEnabled,
		"OUTCOME_ENABLED":      cfg.OutcomeEnabled,
		"CREATORFILL_ENABLED":  cfg.CreatorFillEnabled,
		"WALLET_GRAPH_ENABLED": cfg.WalletGraphEnabled,
		"REPUTATION_ENABLED":   cfg.ReputationEnabled,
		"MANIPULATION_ENABLED": cfg.ManipulationEnabled,
		"OPPORTUNITY_ENABLED":  cfg.OpportunityEnabled,
	}
```
Add to the `api.RouterDeps{...}` construction (find where router deps are built near end of main):
```go
		Health:        reg,
		Pinger:        bundle.Pinger,
		Gates:         gates,
		Version:       cfg.Version,
		StartedAt:     startedAt,
		WSClientCount: hub.ClientCount,
```
Add `"github.com/furkanatesc/sentinel/apps/api-go/internal/health"` and ensure `"time"` are imported.

- [ ] **Step 5: Run — build + config test**

Run: `cd apps/api-go && go build ./... && go test ./internal/config/ -run TestVersion -v`
Expected: build OK + PASS. (Endpoint now returns all workers as off/starting + gates + db + version.)

- [ ] **Step 6: Commit**

```bash
git add apps/api-go/internal/config/ apps/api-go/cmd/server/main.go
git commit -m "feat(system-health): main wiring — registry + Register tüm worker'lar + router deps (Task 4)"
```

---

## PHASE B — Worker instrumentation (canlı state)

### Task 5: `safety` worker Report (template + test)

**Files:**
- Modify: `apps/api-go/internal/safety/worker.go`
- Modify: `apps/api-go/cmd/server/main.go` (safety construction: `Health: reg`)
- Test: `apps/api-go/internal/safety/worker_report_test.go` (Create)

**Interfaces:**
- Consumes: `health.Reporter`, `health.WorkerSafety`.
- Produces: `WorkerDeps.Health health.Reporter` (opsiyonel/nil-ok).

- [ ] **Step 1: Write failing test**

```go
package safety

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type recReporter struct {
	name      string
	ok        bool
	err       error
	processed int
	calls     int
}

func (r *recReporter) Register(string, bool, time.Duration) {}
func (r *recReporter) Report(name string, ok bool, err error, processed int) {
	r.name, r.ok, r.err, r.processed, r.calls = name, ok, err, processed, r.calls+1
}

func TestSafetyWorkerReportsCycle(t *testing.T) {
	rr := &recReporter{}
	// fake store with 0 targets → cycle succeeds, ok=true
	w := NewWorker(WorkerDeps{
		Store: store.NewFakeTokenStore().(SafetyStore), Provider: nil,
		Interval: time.Hour, Health: rr,
	})
	// call the single-cycle path directly via Run with an already-cancelled ctx after first tick:
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.Run(ctx) // executes the immediate scoreOnce once, then returns on ctx.Done
	if rr.calls == 0 {
		t.Fatalf("Report never called")
	}
	if rr.name != "safety" {
		t.Fatalf("name = %q, want safety", rr.name)
	}
	_ = errors.New // keep import if unused elsewhere
}
```
> Uyarma: `NewFakeTokenStore().(SafetyStore)` fake'in `SafetyScoreTargets`/`UpdateSafety` implemente ettiğini varsayar (mevcut safety testlerine bak; etmiyorsa oradaki mevcut fake/stub'ı kullan). Amaç: 0-hedef → `scoreOnce` nil döner → `Report("safety", true, nil, 0)`.

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestSafetyWorkerReportsCycle -v`
Expected: FAIL (`WorkerDeps.Health` yok).

- [ ] **Step 3: Implement**

In `worker.go`:
1. Import `"github.com/furkanatesc/sentinel/apps/api-go/internal/health"`.
2. Add field to `WorkerDeps`: `Health health.Reporter`.
3. Refactor `Run` so each cycle reports. Replace the two `if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil { ... }` blocks with a helper call:

```go
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	w.cycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.cycle(ctx)
		}
	}
}

// cycle, tek scoreOnce + health Report (best-effort). scored/sampleErr'i Report'a taşımak için
// scoreOnce'ın döndürdüğü err yeterli: ok=(err==nil). itemsProcessed v1'de 0 (state+lastErr yeterli).
func (w *Worker) cycle(ctx context.Context) {
	err := w.scoreOnce(ctx)
	if err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("safety score", "err", err)
	}
	if w.d.Health != nil {
		w.d.Health.Report(health.WorkerSafety, err == nil, err, 0)
	}
}
```

In `main.go` safety worker construction (`safety.NewWorker(safety.WorkerDeps{...})`), add field: `Health: reg,`.

- [ ] **Step 4: Run tests, verify pass**

Run: `cd apps/api-go && go test ./internal/safety/ -race -v && go build ./...`
Expected: PASS (yeni + mevcut) + build OK.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/safety/ apps/api-go/cmd/server/main.go
git commit -m "feat(system-health): safety worker cycle Report (Task 5)"
```

---

### Task 6: `ingest-ws` worker Report (event-driven)

**Files:**
- Modify: `apps/api-go/internal/ingest/worker.go`
- Modify: `apps/api-go/cmd/server/main.go` (ingest worker construction: `Health: reg`)
- Test: `apps/api-go/internal/ingest/worker_report_test.go` (Create)

**Interfaces:**
- Consumes: `health.Reporter`, `health.WorkerIngestWS`.
- Produces: `WorkerDeps.Health health.Reporter`.

> **Bağlam:** `ingest/worker.go` `WorkerDeps`'te ZATEN `Registry *Registry` (decoder registry) var — bu FARKLI. Yeni alan adı `Health` (çakışma yok). Instrument noktası: mevcut 30s heartbeat tick (`case <-stats.C:` ~satır 125) + disconnect (`case err := <-done:` ~satır 135).

- [ ] **Step 1: Write failing test**

```go
package ingest

import (
	"testing"
	"time"
)

type recReporter struct{ calls int; lastName string; lastProcessed int }

func (r *recReporter) Register(string, bool, time.Duration) {}
func (r *recReporter) Report(name string, ok bool, err error, processed int) {
	r.calls++; r.lastName = name; r.lastProcessed = processed
}

func TestIngestReporterFieldWired(t *testing.T) {
	rr := &recReporter{}
	w := NewWorker(WorkerDeps{Health: rr}) // WSURL boş → Run hemen döner; alan varlığı derlensin
	if w == nil {
		t.Fatal("nil worker")
	}
}
```
> Not: ingest Run canlı WS ister; heartbeat/disconnect Report'unu birim testte tetiklemek zor. Bu task için asıl doğrulama **derleme + alan varlığı**; heartbeat Report'u kod incelemesiyle teyit edilir (aşağıdaki adım). İstersen `Process` yolunu değil, `stats.C` tick'ini soyutlayan küçük bir refactor eklenebilir — YAGNI, v1'de kod-inceleme yeterli.

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestIngestReporterFieldWired -v`
Expected: FAIL (`WorkerDeps.Health` yok).

- [ ] **Step 3: Implement**

In `worker.go`:
1. Import `health`.
2. Add `Health health.Reporter` to `WorkerDeps`.
3. At heartbeat tick, report healthy with processed count; at disconnect, report degraded:
```go
			case <-stats.C:
				w.d.Logger.Info("ingest heartbeat", "alınan_30s", received, "işlenen_30s", processed)
				if w.d.Health != nil {
					w.d.Health.Report(health.WorkerIngestWS, true, nil, int(processed))
				}
				received, processed = 0, 0
```
```go
			case err := <-done:
				w.d.Logger.Warn("ws bağlantısı koptu, reconnect", "err", err, "backoff", backoff.String())
				if w.d.Health != nil {
					w.d.Health.Report(health.WorkerIngestWS, false, err, 0)
				}
				connected = false
```
In `main.go` ingest worker construction, add `Health: reg,`.

- [ ] **Step 4: Run tests, verify pass**

Run: `cd apps/api-go && go test ./internal/ingest/ -race -v && go build ./...`
Expected: PASS + build OK.

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/ingest/ apps/api-go/cmd/server/main.go
git commit -m "feat(system-health): ingest-ws heartbeat/disconnect Report (Task 6)"
```

---

### Task 7: Kalan ticker worker'ları uniform Report

**Files (her biri Modify + main.go construction'a `Health: reg`):**
- `internal/market/discoverer.go` — cycle method `tick`, name `health.WorkerMarketDisc`, DiscovererDeps
- `internal/market/enricher.go` — `tick`, `health.WorkerMarketEnrich`, EnricherDeps
- `internal/outcome/worker.go` — `classifyOnce`, `health.WorkerOutcome`, WorkerDeps
- `internal/creatorfill/worker.go` — `fillOnce`, `health.WorkerCreatorFill`, WorkerDeps
- `internal/walletgraph/worker.go` — `resolveOnce`, `health.WorkerFunder`, WorkerDeps
- `internal/reputation/worker.go` — `scoreOnce`, `health.WorkerReputation`, WorkerDeps
- `internal/manipulation/worker.go` — `scoreOnce`, `health.WorkerManipulation`, WorkerDeps
- `internal/opportunity/worker.go` — `scoreOnce`, `health.WorkerOpportunity`, WorkerDeps
- Modify: `apps/api-go/cmd/server/main.go` (her worker construction'a `Health: reg`)

**Interfaces:**
- Consumes: `health.Reporter` + ilgili worker adı consts.
- Produces: her `WorkerDeps`/`DiscovererDeps`/`EnricherDeps`'e `Health health.Reporter`.

> **Uniform edit (her worker için AYNI):** Bu 8 worker Run'ı birebir aynı iskelette (`<cycleFn>(ctx)` immediate + ticker loop). Her biri için:
> 1. `health` import et.
> 2. Deps struct'ına `Health health.Reporter` ekle.
> 3. Run'daki her `<cycleFn>` çağrısını bir `cycle` helper'ına sar (safety Task 5 deseni), Report ile.

Örnek (discoverer; diğerleri method/name/receiver adı dışında AYNI):
```go
func (x *Discoverer) Run(ctx context.Context) {
	t := time.NewTicker(x.d.Interval) // mevcut interval kaynağı neyse onu koru
	defer t.Stop()
	x.cycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			x.cycle(ctx)
		}
	}
}

func (x *Discoverer) cycle(ctx context.Context) {
	err := x.tick(ctx)
	if err != nil && ctx.Err() == nil {
		x.d.Logger.Warn("market discover", "err", err) // mevcut log mesajını koru
	}
	if x.d.Health != nil {
		x.d.Health.Report(health.WorkerMarketDisc, err == nil, err, 0)
	}
}
```
> **DİKKAT:** Her worker'ın MEVCUT Run yapısı ufak farklılıklar taşıyabilir (ör. opportunity Run'ında ilk çağrı `for` içinde olabilir; discoverer/enricher `x.d`/`w.d` receiver farkı; log mesajları farklı). Her dosyada MEVCUT log mesajını ve receiver adını koru; yalnız cycle-Report sarmalını ekle. Refactor davranışı değiştirmemeli (aynı immediate+ticker semantiği).

- [ ] **Step 1: Her worker için — mevcut testleri çalıştır (baseline yeşil)**

Run: `cd apps/api-go && go test ./internal/market/ ./internal/outcome/ ./internal/creatorfill/ ./internal/walletgraph/ ./internal/reputation/ ./internal/manipulation/ ./internal/opportunity/ -v`
Expected: PASS (değişiklikten önce).

- [ ] **Step 2: Uniform edit'i 8 dosyaya uygula** (yukarıdaki desen; her dosyada doğru method/name/receiver).

- [ ] **Step 3: main.go — 8 worker construction'ına `Health: reg` ekle.**

- [ ] **Step 4: Doğrula — build + tüm paket testleri (race)**

Run: `cd apps/api-go && go build ./... && go test ./... -race`
Expected: build OK + tüm testler PASS (mevcut testler nil-Health ile kırılmaz; instrument edilmiş worker'lar davranışı korur).

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/market/ apps/api-go/internal/outcome/ apps/api-go/internal/creatorfill/ apps/api-go/internal/walletgraph/ apps/api-go/internal/reputation/ apps/api-go/internal/manipulation/ apps/api-go/internal/opportunity/ apps/api-go/cmd/server/main.go
git commit -m "feat(system-health): kalan ticker worker'lara uniform cycle Report (Task 7)"
```

---

## PHASE C — Frontend panel

### Task 8: Frontend seam — SystemHealth tipi + mock + http + query

**Files:**
- Modify: `apps/web/lib/api/types.ts`
- Modify: `apps/web/lib/api/contract.ts`
- Modify: `apps/web/lib/api/mock.ts`
- Modify: `apps/web/lib/api/http.ts`
- Modify: `apps/web/lib/api/live-endpoints.ts`
- Modify: `apps/web/lib/get-query-client.ts`
- Modify: `apps/web/lib/hooks/queries.ts`
- Modify: `apps/web/lib/api/mock.test.ts`

**Interfaces:**
- Produces: `SystemHealth`, `WorkerStatus` (TS), `SentinelApi.getSystemHealth`, `useSystemHealth`, `qk.systemHealth`.

- [ ] **Step 1: Add types (`types.ts`)**

```ts
export type WorkerState = "off" | "starting" | "ok" | "degraded" | "stalled";

export interface WorkerStatus {
  name: string;
  state: WorkerState;
  lastRunAt: string; // RFC3339 or ""
  lastErr: string;
  cyclesRun: number;
  itemsProcessed: number;
  intervalSec: number;
}

export interface SystemHealth {
  uptimeSec: number;
  version: string;
  dbOk: boolean;
  dbLatencyMs: number;
  wsClients: number;
  workers: WorkerStatus[];
  gates: Record<string, boolean>;
}
```

- [ ] **Step 2: contract + http + live-endpoints + qk + query**

`contract.ts`: import `SystemHealth` in the type import list; add to `SentinelApi`:
```ts
  getSystemHealth(): Promise<SystemHealth>;
```
`http.ts`: import `SystemHealth`; add:
```ts
  getSystemHealth: () => getJson<SystemHealth>("/api/system-health"),
```
`live-endpoints.ts`: add `"getSystemHealth",` to the Set.
`get-query-client.ts`: add to `qk`: `systemHealth: ["system-health"] as const,`.
`queries.ts`: add:
```ts
export function useSystemHealth() {
  return useQuery({
    queryKey: qk.systemHealth,
    queryFn: () => getApi().getSystemHealth(),
    refetchInterval: 10_000,
  });
}
```

- [ ] **Step 3: mock (`mock.ts`) + test (`mock.test.ts`)**

In `mock.ts`, add a representative mock to the exported `mockApi` object (near `getKpis`):
```ts
  getSystemHealth: () =>
    delay<SystemHealth>({
      uptimeSec: 3600,
      version: "dev",
      dbOk: true,
      dbLatencyMs: 4,
      wsClients: 1,
      workers: [
        { name: "ingest-ws", state: "ok", lastRunAt: new Date().toISOString(), lastErr: "", cyclesRun: 12, itemsProcessed: 340, intervalSec: 0 },
        { name: "safety", state: "degraded", lastRunAt: new Date().toISOString(), lastErr: "getTokenAccounts: status 429", cyclesRun: 42, itemsProcessed: 0, intervalSec: 60 },
        { name: "funder", state: "off", lastRunAt: "", lastErr: "", cyclesRun: 0, itemsProcessed: 0, intervalSec: 60 },
      ],
      gates: { MARKET_ENABLED: true, SAFETY_ENABLED: true, WALLET_GRAPH_ENABLED: false },
    }),
```
Import `SystemHealth` type at top of `mock.ts`. In `mock.test.ts` add:
```ts
it("getSystemHealth returns workers + gates", async () => {
  const h = await mockApi.getSystemHealth();
  expect(Array.isArray(h.workers)).toBe(true);
  expect(h.workers.length).toBeGreaterThan(0);
  expect(typeof h.dbOk).toBe("boolean");
  expect(h.gates).toHaveProperty("SAFETY_ENABLED");
});
```

- [ ] **Step 4: Run frontend tests + typecheck**

Run: `cd apps/web && npx vitest run lib/api/mock.test.ts && npx tsc --noEmit`
Expected: PASS + no type errors.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/
git commit -m "feat(system-health): frontend seam — SystemHealth tipi + mock + http + useSystemHealth (Task 8)"
```

---

### Task 9: System Health panel ekranı

**Files:**
- Modify: `apps/web/app/(app)/system-health/page.tsx`

**Interfaces:**
- Consumes: `useSystemHealth`, `SystemHealth`/`WorkerStatus`, `WorkerState`.

- [ ] **Step 1: Implement panel (replace PlaceholderScreen)**

Mevcut UI tasarım-dilini/bileşenlerini kullan (diğer ekranların kullandığı kart/tablo/badge desenine bak — ör. `components/token`/`creator` altındaki mevcut primitive'ler). İskelet:

```tsx
"use client";
import { useSystemHealth } from "@/lib/hooks/queries";
import type { WorkerState } from "@/lib/api/types";

const STATE_TONE: Record<WorkerState, string> = {
  ok: "text-emerald-500",
  starting: "text-sky-500",
  degraded: "text-amber-500",
  stalled: "text-red-500",
  off: "text-zinc-500",
};

function ago(iso: string): string {
  if (!iso) return "—";
  const s = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000));
  if (s < 60) return `${s} sn önce`;
  if (s < 3600) return `${Math.floor(s / 60)} dk önce`;
  return `${Math.floor(s / 3600)} sa önce`;
}

export default function Page() {
  const { data, isLoading, isError } = useSystemHealth();
  if (isLoading) return <div className="p-6">Yükleniyor…</div>;
  if (isError || !data) return <div className="p-6 text-red-500">Sistem sağlığı alınamadı.</div>;
  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-wrap gap-4 text-sm">
        <span>DB: <b className={data.dbOk ? "text-emerald-500" : "text-red-500"}>{data.dbOk ? `ok (${data.dbLatencyMs}ms)` : "erişilemiyor"}</b></span>
        <span>Uptime: <b>{Math.floor(data.uptimeSec / 60)} dk</b></span>
        <span>Version: <b>{data.version}</b></span>
        <span>WS istemci: <b>{data.wsClients}</b></span>
      </div>
      <table className="w-full text-sm">
        <thead><tr className="text-left text-zinc-400">
          <th className="py-2">Worker</th><th>Durum</th><th>Son çalışma</th><th>Cycles</th><th>İşlenen</th><th>Hata</th>
        </tr></thead>
        <tbody>
          {data.workers.map((w) => (
            <tr key={w.name} className="border-t border-zinc-800">
              <td className="py-2 font-mono">{w.name}</td>
              <td className={STATE_TONE[w.state]}>{w.state}</td>
              <td>{ago(w.lastRunAt)}</td>
              <td>{w.cyclesRun}</td>
              <td>{w.itemsProcessed}</td>
              <td className="text-amber-500 max-w-xs truncate" title={w.lastErr}>{w.lastErr || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="text-xs text-zinc-400">
        Gates: {Object.entries(data.gates).map(([k, v]) => `${k}=${v ? "on" : "off"}`).join(" · ")}
      </div>
    </div>
  );
}
```
> **Not:** Tailwind class'ları örnektir; projenin mevcut tema token'larına/bileşenlerine uydur (ör. varsa `Card`, `Badge`, `Table` primitive'leri kullan — ham `<table>` yerine). Amaç: mevcut tasarım-diliyle tutarlı, yeni bileşen icat etmeden.

- [ ] **Step 2: Typecheck + lint + dev smoke (mock)**

Run: `cd apps/web && npx tsc --noEmit && npm run lint`
Expected: no errors. (Mock modda `/system-health` sayfası worker tablosunu gösterir.)

- [ ] **Step 3: Commit**

```bash
git add apps/web/app/
git commit -m "feat(system-health): system-health panel ekranı (Task 9)"
```

---

## PHASE D — Doğrulama & kapanış

### Task 10: Whole-branch review + doküman güncelleme

- [ ] **Step 1: Tüm testler + build (backend + frontend)**

Run:
```bash
cd apps/api-go && go test ./... -race && go vet ./...
cd ../web && npx tsc --noEmit && npx vitest run && npm run lint
```
Expected: hepsi yeşil.

- [ ] **Step 2: Whole-branch code review** — superpowers:requesting-code-review (opus) ile `feat/backend-system-health` diff'ini incele. Bulguları superpowers:receiving-code-review ile ele al.

- [ ] **Step 3: Yaşayan dokümanlar** — `docs/progress.md` (varsa) + `MEMORY.md` aktif-iş satırını güncelle (System Health MERGED durumu). Kapsam-dışı (WS push, trend, alert) işaretli kalsın.

- [ ] **Step 4: PR aç** — kullanıcı onayıyla (push DUR-noktası: kullanıcı Railway/deploy adımlarını kendisi yönetir; push/PR öncesi kullanıcıya sor).

---

## Self-Review

**1. Spec coverage:**
- §3 registry/Reporter/Snapshot/state → Task 1 ✅
- §4.2 DB probe → Task 2 + Task 3 (`probeDB`) ✅
- §4.1 worker sinyalleri → Task 5,6,7 (Report) ✅; §4.3 global (uptime/version/wsClients/gates) → Task 3,4 ✅
- §5 endpoint → Task 3 ✅
- §6 frontend seam+panel → Task 8,9 ✅
- §7 güvenlik: sanitize → Task 1 (`sanitizeErr` + test); graceful 200 → Task 3 (`TestSystemHealthDBDownStill200`) ✅
- §8 testler → her task TDD ✅

**2. Placeholder scan:** Kod adımları somut; "mevcut bileşene uydur" notları frontend'de kaçınılmaz (proje tema primitive'leri task-anında okunmalı) ama iskelet+seam tam. ✅

**3. Type consistency:** `Reporter.Report(name, ok, err, processed)` imzası Task 1/5/6/7'de aynı; worker adı consts tek kaynak (health paketi); `SystemHealth`/`WorkerStatus` JSON etiketleri Go (Task 1/3) ↔ TS (Task 8) birebir (`lastRunAt`, `dbOk`, `itemsProcessed`, `intervalSec` …). ✅

**Bilinen kırılganlık:** Task 4 worker deps'e `Health` set etmez (o iş Task 5-7'de, her worker kendi task'ında main.go construction'ına ekler) → her task ayrı ayrı derlenir. Bu bağımlılık Task 4 notunda açıkça belirtildi.
