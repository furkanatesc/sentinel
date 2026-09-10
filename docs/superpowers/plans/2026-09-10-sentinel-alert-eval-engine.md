# Alarm Değerlendirme Motoru Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/alerts` geçmişini gerçek yap — aktif alarm kurallarını mevcut event akışıyla eşleştiren periyodik worker + üretilen alarmların kalıcılığı + `getAlerts`'i canlıya çevirme.

**Architecture:** Yeni `internal/alerteval` worker (trend deseni) her cycle'da watermark'tan yeni event'leri `RecentEvents` ile çeker, saf `matchRule` ile aktif kurallarla eşleştirir, eşleşmeleri `alert_events` tablosuna yazar, watermark'ı ilerletir. `GET /api/alerts` bu geçmişi döner.

**Tech Stack:** Go (chi, database/sql, pgx, goose), mevcut store/health/config desenleri; frontend TypeScript (hibrit getApi seam).

**Spec:** `docs/superpowers/specs/2026-09-10-sentinel-alert-eval-engine-design.md`

## Global Constraints

- Clean/SOLID: dar DIP arayüzleri; saf `matchRule` (SRP, tablo-testli); worker yalnız orkestrasyon.
- Entegrasyon-gerektirmez (mevcut Postgres event akışı); yeni harici bağımlılık YOK.
- Config-gated (`ALERTEVAL_ENABLED` varsayılan true, `ALERTEVAL_INTERVAL_SEC` varsayılan 30) + nil-güvenli health.
- v1 trigger kapsamı = `new_mint`, `liquidity_added`, `liquidity_removed`; diğerleri registry'de kapsam-dışı (asla eşleşmez).
- Gerçek Slack/email teslimatı HARİÇ (sona). Geriye uyumlu: mock `getAlerts` korunur.
- `go build/vet/test ./... -race` + frontend tsc/vitest/build yeşil; mevcut testler kırılmaz.
- Commit sonu:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`
  `Claude-Session: https://claude.ai/code/session_01RB25UjvJoss3v9xbG72Esx`

---

## File Structure

- Create `apps/api-go/internal/store/migrations/0018_create_alert_events.sql` — alert_events + alert_eval_meta tabloları.
- Create `apps/api-go/internal/store/alertevents.go` — AlertEventRow + AlertEventStore (postgres + fake + watermark).
- Create `apps/api-go/internal/store/alertevents_test.go` — fake insert/recent/watermark testi.
- Modify `apps/api-go/internal/store/postgres.go` — Bundle.AlertEvents + wiring (seed yok).
- Create `apps/api-go/internal/alerteval/match.go` — saf matchRule + riskRank + evaluableTriggers.
- Create `apps/api-go/internal/alerteval/match_test.go` — tablo-testi.
- Create `apps/api-go/internal/alerteval/worker.go` — Worker + WorkerDeps + Run/cycle.
- Create `apps/api-go/internal/alerteval/worker_test.go` — bir-cycle + dedup testi.
- Create `apps/api-go/internal/api/alerthistory.go` — GET /api/alerts handler.
- Create `apps/api-go/internal/api/alerthistory_test.go` — handler testi.
- Modify `apps/api-go/internal/api/router.go` — RouterDeps.AlertEvents + route.
- Modify `apps/api-go/internal/config/config.go` — AlertEvalEnabled + AlertEvalIntervalSec.
- Modify `apps/api-go/internal/health/registry.go` — WorkerAlertEval sabiti.
- Modify `apps/api-go/cmd/server/main.go` — fake bundle + worker run + health register + gate + RouterDeps.
- Modify `apps/web/lib/api/http.ts` — getAlerts canlı.
- Modify `apps/web/lib/api/live-endpoints.ts` — getAlerts ekle.
- Modify `apps/web/lib/api/mock.test.ts` (opsiyonel: getAlerts şekil zaten var).

---

## Task 1: Store — migration 0018 + AlertEventStore + watermark

**Files:**
- Create: `apps/api-go/internal/store/migrations/0018_create_alert_events.sql`
- Create: `apps/api-go/internal/store/alertevents.go`
- Test: `apps/api-go/internal/store/alertevents_test.go`
- Modify: `apps/api-go/internal/store/postgres.go`

**Interfaces:**
- Produces:
  - `type AlertEventRow struct { ID, RuleID, Type, Token, Detail, Severity, Time string; Ts int64 }` (JSON: id/type/token/detail/severity/time — frontend AlertEvent birebir; **RuleID ve Ts `json:"-"`** çünkü frontend AlertEvent kontratında yoklar. [Düzeltildi 2026-09-10 review: önceki not "ts JSON'da kalır" yanlıştı.])
  - `type AlertEventStore interface { InsertAlertEvent(ctx, AlertEventRow) error; RecentAlertEvents(ctx, limit int) ([]AlertEventRow, error); GetAlertWatermark(ctx) (int64, error); SetAlertWatermark(ctx, ts int64) error }`
  - `func NewFakeAlertEventStore() AlertEventStore`
  - `Bundle.AlertEvents AlertEventStore`

- [ ] **Step 1: Migration**

`0018_create_alert_events.sql`:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS alert_events (
    id       TEXT PRIMARY KEY,
    rule_id  TEXT NOT NULL DEFAULT '',
    type     TEXT NOT NULL DEFAULT '',
    token    TEXT NOT NULL DEFAULT '',
    detail   TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'info',
    time     TEXT NOT NULL DEFAULT '',
    ts       BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_alert_events_ts ON alert_events (ts DESC);
CREATE TABLE IF NOT EXISTS alert_eval_meta (
    id           TEXT PRIMARY KEY,   -- 'default'
    watermark_ts BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS alert_events;
DROP TABLE IF EXISTS alert_eval_meta;
```

- [ ] **Step 2: Write failing fake-store test**

`alertevents_test.go`:
```go
package store

import (
	"context"
	"testing"
)

func TestFakeAlertEventStore(t *testing.T) {
	s := NewFakeAlertEventStore()
	ctx := context.Background()

	// başlangıç watermark 0
	if wm, err := s.GetAlertWatermark(ctx); err != nil || wm != 0 {
		t.Fatalf("başlangıç watermark 0 beklenir: %d %v", wm, err)
	}
	// insert + recent (newest-first)
	_ = s.InsertAlertEvent(ctx, AlertEventRow{ID: "a1", RuleID: "r1", Type: "new_mint", Token: "AAA", Severity: "info", Ts: 100})
	_ = s.InsertAlertEvent(ctx, AlertEventRow{ID: "a2", RuleID: "r2", Type: "liquidity_removed", Token: "BBB", Severity: "critical", Ts: 200})
	got, err := s.RecentAlertEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "a2" {
		t.Fatalf("newest-first [a2 a1] beklenir: %+v", got)
	}
	// watermark set/get
	if err := s.SetAlertWatermark(ctx, 200); err != nil {
		t.Fatal(err)
	}
	if wm, _ := s.GetAlertWatermark(ctx); wm != 200 {
		t.Fatalf("watermark 200 beklenir: %d", wm)
	}
	// limit
	if got, _ := s.RecentAlertEvents(ctx, 1); len(got) != 1 {
		t.Fatalf("limit 1 beklenir: %d", len(got))
	}
}
```

- [ ] **Step 3: Run — FAIL** (`go test ./internal/store/ -run TestFakeAlertEventStore` → undefined).

- [ ] **Step 4: Implement `alertevents.go`**

```go
package store

import (
	"context"
	"database/sql"
	"sort"
	"sync"
)

const alertEvalMetaID = "default"

// AlertEventRow, üretilmiş bir alarmdır (frontend AlertEvent ile birebir JSON). RuleID iç-kullanım (JSON'da yok).
type AlertEventRow struct {
	ID       string `json:"id"`
	RuleID   string `json:"-"`
	Type     string `json:"type"`
	Token    string `json:"token"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
	Time     string `json:"time"`
	Ts       int64  `json:"-"`
}

// AlertEventStore, üretilmiş alarmların append-only kaydı + değerlendirme watermark'ıdır (DIP).
type AlertEventStore interface {
	InsertAlertEvent(ctx context.Context, e AlertEventRow) error
	RecentAlertEvents(ctx context.Context, limit int) ([]AlertEventRow, error)
	GetAlertWatermark(ctx context.Context) (int64, error)
	SetAlertWatermark(ctx context.Context, ts int64) error
}

// --- postgres ---

func (p *postgresStore) InsertAlertEvent(ctx context.Context, e AlertEventRow) error {
	const q = `INSERT INTO alert_events (id, rule_id, type, token, detail, severity, time, ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`
	_, err := p.db.ExecContext(ctx, q, e.ID, e.RuleID, e.Type, e.Token, e.Detail, e.Severity, e.Time, e.Ts)
	return err
}

func (p *postgresStore) RecentAlertEvents(ctx context.Context, limit int) ([]AlertEventRow, error) {
	const q = `SELECT id, rule_id, type, token, detail, severity, time, ts
		FROM alert_events ORDER BY ts DESC, id DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AlertEventRow{}
	for rows.Next() {
		var e AlertEventRow
		if err := rows.Scan(&e.ID, &e.RuleID, &e.Type, &e.Token, &e.Detail, &e.Severity, &e.Time, &e.Ts); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (p *postgresStore) GetAlertWatermark(ctx context.Context) (int64, error) {
	var ts int64
	err := p.db.QueryRowContext(ctx, `SELECT watermark_ts FROM alert_eval_meta WHERE id=$1`, alertEvalMetaID).Scan(&ts)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return ts, err
}

func (p *postgresStore) SetAlertWatermark(ctx context.Context, ts int64) error {
	const q = `INSERT INTO alert_eval_meta (id, watermark_ts) VALUES ($1,$2)
		ON CONFLICT (id) DO UPDATE SET watermark_ts=EXCLUDED.watermark_ts`
	_, err := p.db.ExecContext(ctx, q, alertEvalMetaID, ts)
	return err
}

// --- fake ---

type fakeAlertEventStore struct {
	mu     sync.Mutex
	events []AlertEventRow
	wm     int64
}

// NewFakeAlertEventStore, DB'siz mod/testler için in-memory store.
func NewFakeAlertEventStore() AlertEventStore { return &fakeAlertEventStore{events: []AlertEventRow{}} }

func (f *fakeAlertEventStore) InsertAlertEvent(_ context.Context, e AlertEventRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
	return nil
}

func (f *fakeAlertEventStore) RecentAlertEvents(_ context.Context, limit int) ([]AlertEventRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]AlertEventRow(nil), f.events...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ts != out[j].Ts {
			return out[i].Ts > out[j].Ts
		}
		return out[i].ID > out[j].ID
	})
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeAlertEventStore) GetAlertWatermark(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.wm, nil
}

func (f *fakeAlertEventStore) SetAlertWatermark(_ context.Context, ts int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.wm = ts
	return nil
}
```

- [ ] **Step 5: Bundle wiring** — `postgres.go`: `Bundle`'a `AlertEvents AlertEventStore` alanı ekle; `OpenPostgres` dönüşünde `AlertEvents: ps`. (Migration otomatik `runMigrations` ile çalışır; seed YOK.)

- [ ] **Step 6: Run — PASS** (`go test ./internal/store/ -race -run TestFakeAlertEventStore`).

- [ ] **Step 7: Commit** — `feat(alert-eval): migration 0018 + AlertEventStore (insert/recent/watermark) + fake (Task 1)`

---

## Task 2: alerteval — saf matchRule + riskRank

**Files:**
- Create: `apps/api-go/internal/alerteval/match.go`
- Test: `apps/api-go/internal/alerteval/match_test.go`

**Interfaces:**
- Consumes: `store.EventRow`, `store.AlertRule` (Task 1 store paketi).
- Produces:
  - `func MatchRule(e store.EventRow, r store.AlertRule) bool`
  - `var EvaluableTriggers = map[string]bool{"new_mint": true, "liquidity_added": true, "liquidity_removed": true}`

- [ ] **Step 1: Write failing table test**

`match_test.go`:
```go
package alerteval

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestMatchRule(t *testing.T) {
	base := store.EventRow{Type: "new_mint", Symbol: "AAA", Launchpad: "Pump.fun", Liquidity: 20000, CreatorScore: 80, RiskLevel: "medium", Ts: 10}
	rule := store.AlertRule{Trigger: "new_mint", Scope: "Tüm tokenlar", MinLiquidity: 10000, MinCreatorScore: 70, MaxRisk: "high", Enabled: true}

	cases := []struct {
		name string
		e    store.EventRow
		r    store.AlertRule
		want bool
	}{
		{"tam eşleşme", base, rule, true},
		{"trigger farklı", base, withTrigger(rule, "liquidity_removed"), false},
		{"kapsam-dışı trigger", withType(base, "whale_activity"), withTrigger(rule, "whale_activity"), false},
		{"likidite düşük", withLiq(base, 5000), rule, false},
		{"skor düşük", withScore(base, 50), rule, false},
		{"risk tavanı aşıldı", withRisk(base, "critical"), rule, false},
		{"risk tavanı sınırında", withRisk(base, "high"), rule, true},
		{"scope launchpad eşleşir", base, withScope(rule, "Pump.fun"), true},
		{"scope launchpad eşleşmez", base, withScope(rule, "Raydium"), false},
		{"scope boş = hepsi", base, withScope(rule, ""), true},
	}
	for _, c := range cases {
		if got := MatchRule(c.e, c.r); got != c.want {
			t.Errorf("%s: MatchRule=%v want %v", c.name, got, c.want)
		}
	}
}

func withTrigger(r store.AlertRule, v string) store.AlertRule { r.Trigger = v; return r }
func withScope(r store.AlertRule, v string) store.AlertRule   { r.Scope = v; return r }
func withType(e store.EventRow, v string) store.EventRow      { e.Type = v; return e }
func withLiq(e store.EventRow, v float64) store.EventRow      { e.Liquidity = v; return e }
func withScore(e store.EventRow, v float64) store.EventRow    { e.CreatorScore = v; return e }
func withRisk(e store.EventRow, v string) store.EventRow      { e.RiskLevel = v; return e }
```

- [ ] **Step 2: Run — FAIL** (undefined MatchRule).

- [ ] **Step 3: Implement `match.go`**

```go
// Package alerteval, aktif alarm kurallarını mevcut event akışıyla eşleştiren değerlendirme motorudur.
// Saf eşleştirme (match.go) + periyodik worker (worker.go). Yeni harici bağımlılık YOK.
package alerteval

import (
	"strings"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// EvaluableTriggers, v1'de değerlendirilen trigger'lardır (EventRow verisi güvenilir dolu).
// Kapsam-dışı trigger'lar (holder_growth/whale_activity/score_change/creator_sale/strategy_signal)
// veri gelene dek asla eşleşmez — sessiz düşürme yok, bilinçli kapsam.
var EvaluableTriggers = map[string]bool{
	"new_mint":          true,
	"liquidity_added":   true,
	"liquidity_removed": true,
}

// riskRank, risk seviyesini sıralanabilir sayıya çevirir (tavan karşılaştırması için).
func riskRank(level string) int {
	switch strings.ToLower(level) {
	case "low":
		return 0
	case "medium":
		return 1
	case "high":
		return 2
	case "critical":
		return 3
	default:
		return 1 // bilinmeyen → medium gibi davran (güvenli orta)
	}
}

// MatchRule, bir event'in bir kurala uyup uymadığını döner (saf; SRP).
func MatchRule(e store.EventRow, r store.AlertRule) bool {
	if r.Trigger != e.Type || !EvaluableTriggers[r.Trigger] {
		return false
	}
	if e.Liquidity < r.MinLiquidity {
		return false
	}
	if e.CreatorScore < r.MinCreatorScore {
		return false
	}
	if riskRank(e.RiskLevel) > riskRank(r.MaxRisk) {
		return false
	}
	scope := strings.TrimSpace(r.Scope)
	if scope == "" || strings.EqualFold(scope, "Tüm tokenlar") {
		return true
	}
	return strings.EqualFold(scope, strings.TrimSpace(e.Launchpad))
}
```

- [ ] **Step 4: Run — PASS** (`go test ./internal/alerteval/ -race -run TestMatchRule`).

- [ ] **Step 5: Commit** — `feat(alert-eval): saf MatchRule + riskRank + EvaluableTriggers (Task 2)`

---

## Task 3: alerteval — periyodik worker (cycle + dedup)

**Files:**
- Create: `apps/api-go/internal/alerteval/worker.go`
- Test: `apps/api-go/internal/alerteval/worker_test.go`

**Interfaces:**
- Consumes: `store.EventRow`, `store.AlertRule`, `store.AlertEventRow`, `health.Reporter`, `MatchRule` (Task 2).
- Produces:
  - `type Source interface { RecentEvents(ctx, limit int) ([]store.EventRow, error); ListAlertRules(ctx) ([]store.AlertRule, error) }`
  - `type Sink interface { InsertAlertEvent(ctx, store.AlertEventRow) error; GetAlertWatermark(ctx) (int64, error); SetAlertWatermark(ctx, int64) error }`
  - `type WorkerDeps struct { Source Source; Sink Sink; Interval time.Duration; Limit int; Now func() int64; Logger *slog.Logger; Health health.Reporter }`
  - `func NewWorker(WorkerDeps) *Worker`
  - `func (w *Worker) Run(ctx context.Context)`
  - `func (w *Worker) Cycle(ctx context.Context) (int, error)` (üretilen alarm sayısı — test + itemsProcessed)

- [ ] **Step 1: Write failing worker test**

`worker_test.go`:
```go
package alerteval

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type stubSource struct {
	events []store.EventRow
	rules  []store.AlertRule
}

func (s *stubSource) RecentEvents(_ context.Context, _ int) ([]store.EventRow, error) {
	return append([]store.EventRow(nil), s.events...), nil
}
func (s *stubSource) ListAlertRules(_ context.Context) ([]store.AlertRule, error) {
	return append([]store.AlertRule(nil), s.rules...), nil
}

func TestCycleProducesAlertsAndAdvancesWatermarkOnce(t *testing.T) {
	ctx := context.Background()
	src := &stubSource{
		events: []store.EventRow{
			{ID: "e1", Type: "new_mint", Symbol: "AAA", Liquidity: 20000, CreatorScore: 80, RiskLevel: "medium", Detail: "d", Time: "1m", Ts: 100},
			{ID: "e2", Type: "liquidity_removed", Symbol: "BBB", Liquidity: 0, CreatorScore: 0, RiskLevel: "critical", Detail: "rug", Time: "2m", Ts: 200},
		},
		rules: []store.AlertRule{
			{ID: "r1", Name: "Yeni mint", Trigger: "new_mint", Scope: "Tüm tokenlar", MaxRisk: "high", Enabled: true},
			{ID: "r2", Name: "Likidite", Trigger: "liquidity_removed", Scope: "Tüm tokenlar", MaxRisk: "critical", Enabled: true},
			{ID: "r3", Name: "Kapalı", Trigger: "new_mint", Scope: "Tüm tokenlar", MaxRisk: "high", Enabled: false},
		},
	}
	sink := store.NewFakeAlertEventStore()
	w := NewWorker(WorkerDeps{Source: src, Sink: sink, Limit: 100})

	n, err := w.Cycle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("2 alarm beklenir (e1→r1, e2→r2; r3 kapalı): %d", n)
	}
	got, _ := sink.RecentAlertEvents(ctx, 10)
	if len(got) != 2 || got[0].Token != "BBB" {
		t.Fatalf("newest-first 2 alarm beklenir: %+v", got)
	}
	if got[0].Detail != "Likidite — rug" {
		t.Fatalf("detail = kural adı + event detail beklenir: %q", got[0].Detail)
	}
	if wm, _ := sink.GetAlertWatermark(ctx); wm != 200 {
		t.Fatalf("watermark en yeni event ts'ine ilerlemeli: %d", wm)
	}

	// ikinci cycle: aynı event'ler watermark altında → tekrar üretmez (dedup)
	n2, _ := w.Cycle(ctx)
	if n2 != 0 {
		t.Fatalf("ikinci cycle 0 alarm beklenir (dedup): %d", n2)
	}
}
```

- [ ] **Step 2: Run — FAIL** (undefined Worker).

- [ ] **Step 3: Implement `worker.go`**

```go
package alerteval

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/health"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Source, worker'ın okuma bağımlılığıdır (DIP; store karşılar).
type Source interface {
	RecentEvents(ctx context.Context, limit int) ([]store.EventRow, error)
	ListAlertRules(ctx context.Context) ([]store.AlertRule, error)
}

// Sink, worker'ın yazma + watermark bağımlılığıdır (DIP; store.AlertEventStore karşılar).
type Sink interface {
	InsertAlertEvent(ctx context.Context, e store.AlertEventRow) error
	GetAlertWatermark(ctx context.Context) (int64, error)
	SetAlertWatermark(ctx context.Context, ts int64) error
}

type WorkerDeps struct {
	Source   Source
	Sink     Sink
	Interval time.Duration
	Limit    int // RecentEvents pencere boyutu (varsayılan 200)
	Now      func() int64
	Logger   *slog.Logger
	Health   health.Reporter // nil-güvenli
}

type Worker struct{ d WorkerDeps }

func NewWorker(d WorkerDeps) *Worker {
	if d.Now == nil {
		d.Now = func() int64 { return time.Now().Unix() }
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Limit <= 0 {
		d.Limit = 200
	}
	return &Worker{d: d}
}

// Run, immediate bir cycle + ticker döngüsü (diğer worker'larla aynı iskelet).
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.d.Interval)
	defer t.Stop()
	w.runCycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.runCycle(ctx)
		}
	}
}

func (w *Worker) runCycle(ctx context.Context) {
	n, err := w.Cycle(ctx)
	if w.d.Health != nil {
		// Health.Report imzası: Report(name, ok, err, processed) — worker adı burada geçilir (trend deseni).
		w.d.Health.Report(health.WorkerAlertEval, err == nil, err, n)
	}
	if err != nil {
		w.d.Logger.Warn("alerteval cycle hatası", "err", err)
	}
}

// Cycle, watermark'tan yeni event'leri aktif kurallarla eşleştirir, eşleşmeleri yazar,
// watermark'ı ilerletir. Üretilen alarm sayısını döner. (Saf orkestrasyon; eşleştirme MatchRule.)
func (w *Worker) Cycle(ctx context.Context) (int, error) {
	wm, err := w.d.Sink.GetAlertWatermark(ctx)
	if err != nil {
		return 0, err
	}
	events, err := w.d.Source.RecentEvents(ctx, w.d.Limit)
	if err != nil {
		return 0, err
	}
	rules, err := w.d.Source.ListAlertRules(ctx)
	if err != nil {
		return 0, err
	}
	maxTs := wm
	produced := 0
	for _, e := range events {
		if e.Ts <= wm {
			continue // watermark altında = zaten değerlendirildi (dedup)
		}
		if e.Ts > maxTs {
			maxTs = e.Ts
		}
		for _, r := range rules {
			if !r.Enabled || !MatchRule(e, r) {
				continue
			}
			token := e.Symbol
			if token == "" {
				token = e.Mint
			}
			row := store.AlertEventRow{
				ID:       fmt.Sprintf("a%d-%s", e.Ts, r.ID),
				RuleID:   r.ID,
				Type:     e.Type,
				Token:    token,
				Detail:   r.Name + " — " + e.Detail,
				Severity: e.Severity,
				Time:     e.Time,
				Ts:       e.Ts,
			}
			if err := w.d.Sink.InsertAlertEvent(ctx, row); err != nil {
				return produced, err
			}
			produced++
		}
	}
	if maxTs > wm {
		if err := w.d.Sink.SetAlertWatermark(ctx, maxTs); err != nil {
			return produced, err
		}
	}
	return produced, nil
}
```

- [ ] **Step 4: Run — PASS** (`go test ./internal/alerteval/ -race`).

- [ ] **Step 5: Commit** — `feat(alert-eval): periyodik worker (Cycle + watermark dedup + health) (Task 3)`

---

## Task 4: API — GET /api/alerts

**Files:**
- Create: `apps/api-go/internal/api/alerthistory.go`
- Test: `apps/api-go/internal/api/alerthistory_test.go`
- Modify: `apps/api-go/internal/api/router.go`

**Interfaces:**
- Consumes: `store.AlertEventStore` (Task 1).
- Produces: `RouterDeps.AlertEvents store.AlertEventStore`; route `GET /api/alerts`.

- [ ] **Step 1: Write failing handler test**

`alerthistory_test.go`:
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

func TestAlertsHistoryEndpoint(t *testing.T) {
	s := store.NewFakeAlertEventStore()
	_ = s.InsertAlertEvent(context.Background(), store.AlertEventRow{ID: "a1", Type: "new_mint", Token: "AAA", Severity: "info", Time: "1m", Ts: 100})
	r := NewRouter(RouterDeps{AlertEvents: s})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/alerts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var out []store.AlertEventRow
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Token != "AAA" {
		t.Fatalf("1 alarm beklenir: %+v", out)
	}
}
```

- [ ] **Step 2: Run — FAIL**.

- [ ] **Step 3: Implement `alerthistory.go`**

```go
package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// alertsHistoryHandler, üretilmiş alarm geçmişini döner (GET /api/alerts). Frontend AlertEvent[].
func alertsHistoryHandler(s store.AlertEventStore, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := s.RecentAlertEvents(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "alerts unavailable"})
			return
		}
		if events == nil {
			events = []store.AlertEventRow{}
		}
		writeJSON(w, http.StatusOK, events)
	}
}
```

- [ ] **Step 4: Router** — `router.go`: `RouterDeps`'e `AlertEvents store.AlertEventStore` ekle; NewRouter'da:
```go
	if d.AlertEvents != nil {
		limit := d.EventsWindow
		if limit <= 0 {
			limit = 100
		}
		r.Get("/api/alerts", alertsHistoryHandler(d.AlertEvents, limit))
	}
```

- [ ] **Step 5: Run — PASS** (`go test ./internal/api/ -race -run TestAlertsHistoryEndpoint`).

- [ ] **Step 6: Commit** — `feat(alert-eval): GET /api/alerts handler + router (Task 4)`

---

## Task 5: Entegrasyon — config + health + main (worker run + gate)

**Files:**
- Modify: `apps/api-go/internal/config/config.go`
- Modify: `apps/api-go/internal/health/registry.go`
- Modify: `apps/api-go/cmd/server/main.go`

**Interfaces:**
- Consumes: Task 1 `Bundle.AlertEvents`, Task 3 `alerteval.NewWorker`, Task 4 `RouterDeps.AlertEvents`.

- [ ] **Step 1: config** — `config.go` `Config` struct'a ekle:
```go
	AlertEvalEnabled     bool
	AlertEvalIntervalSec int
```
`Load()` içine (Trend satırlarının yanına):
```go
		AlertEvalEnabled:     getenvBool("ALERTEVAL_ENABLED", true),
		AlertEvalIntervalSec: getenvInt("ALERTEVAL_INTERVAL_SEC", 30),
```

- [ ] **Step 2: health sabiti** — `registry.go` sabit bloğuna ekle: `WorkerAlertEval = "alert-eval"`.

- [ ] **Step 3: main fake bundle** — `main.go` in-memory fake Bundle'a ekle: `AlertEvents: store.NewFakeAlertEventStore(),`.

- [ ] **Step 4: main — health register** (trend Register satırının yanına):
```go
	healthReg.Register(health.WorkerAlertEval, cfg.AlertEvalEnabled && bundle.AlertEvents != nil && bundle.Events != nil && bundle.AlertRules != nil, time.Duration(cfg.AlertEvalIntervalSec)*time.Second)
```

- [ ] **Step 5: main — gate** — `gates` map'ine: `"ALERTEVAL_ENABLED": cfg.AlertEvalEnabled,`.

- [ ] **Step 6: main — worker run** (trend `tw` bloğunun yanına). Not: `bundle` tek postgres store'dur → Source için Events+Rules aynı `bundle` üstünde; `AlertEvents` Sink:
```go
	if cfg.AlertEvalEnabled && bundle.AlertEvents != nil && bundle.Events != nil && bundle.AlertRules != nil {
		aw := alerteval.NewWorker(alerteval.WorkerDeps{
			Source:   alertEvalSource{events: bundle.Events, rules: bundle.AlertRules},
			Sink:     bundle.AlertEvents,
			Interval: time.Duration(cfg.AlertEvalIntervalSec) * time.Second,
			Logger:   logger,
			Health:   healthReg,
		})
		go aw.Run(ctx)
	}
```
`alertEvalSource` küçük adaptör (Source'u iki store'dan birleştirir) — `main.go` altına ekle:
```go
type alertEvalSource struct {
	events store.EventStore
	rules  store.AlertRuleStore
}

func (s alertEvalSource) RecentEvents(ctx context.Context, limit int) ([]store.EventRow, error) {
	return s.events.RecentEvents(ctx, limit)
}
func (s alertEvalSource) ListAlertRules(ctx context.Context) ([]store.AlertRule, error) {
	return s.rules.ListAlertRules(ctx)
}
```
`import` ekle: `".../internal/alerteval"`. (`Health: healthReg` — trend worker'ın yaptığıyla birebir aynı; registry `health.Reporter`'ı karşılar.)

- [ ] **Step 7: RouterDeps wiring** — `main.go` `api.RouterDeps{...}` içine: `AlertEvents: bundle.AlertEvents,`.

- [ ] **Step 8: Run — PASS** (`go build ./... && go vet ./... && go test ./... -race`). Beklenen: hepsi yeşil.

- [ ] **Step 9: Commit** — `feat(alert-eval): config + health + main worker wiring + gate (Task 5)`

> **NOT (executor):** `Health: healthReg` doğrudan registry'yi geçer (trend deseni, main ~290). Worker
> `runCycle` içinde `Health.Report(health.WorkerAlertEval, ok, err, n)` çağırır (isim orada geçilir).

---

## Task 6: Frontend — getAlerts canlı

**Files:**
- Modify: `apps/web/lib/api/http.ts`
- Modify: `apps/web/lib/api/live-endpoints.ts`

**Interfaces:**
- Consumes: backend `GET /api/alerts` (Task 4).

- [ ] **Step 1: http.ts** — import'a `AlertEvent` ekli mi kontrol et (değilse ekle); `getAlerts: notReady,` satırını değiştir:
```ts
  getAlerts: () => getJson<AlertEvent[]>("/api/alerts"),
```
(`AlertEvent` `./types`'tan import edilmeli — mevcut import satırına ekle.)

- [ ] **Step 2: live-endpoints.ts** — `LIVE_ENDPOINTS` set'ine `"getAlerts",` ekle.

- [ ] **Step 3: Test — mevcut suite** — `npx vitest run` (getAlerts mock testi hâlâ geçer; AlertHistoryPanel değişmez). Beklenen: tüm testler yeşil.

- [ ] **Step 4: Build** — `npm run build`. Beklenen: `/alerts` prerender başarılı.

- [ ] **Step 5: Commit** — `feat(alert-eval): frontend getAlerts canlı + LIVE_ENDPOINTS (Task 6)`

---

## Task 7: Review + docs + merge/push

- [ ] **Step 1: Whole-branch verification** — `cd apps/api-go && go build ./... && go vet ./... && go test ./... -race`; `cd apps/web && npx vitest run && npm run build`. Hepsi yeşil olmalı.
- [ ] **Step 2: Whole-branch review** — `/code-review master..HEAD` → receiving-code-review ile bulguları doğrula + gider.
- [ ] **Step 3: Docs** — `docs/progress.md`'ye dilim özeti + `docs/superpowers/followups-frontend.md`'ye "Alarm Değerlendirme Motoru — deferred" (Slack teslimatı, kapsam-dışı 5 trigger, retention/prune, WS canlı push).
- [ ] **Step 4: MEMORY** — `MEMORY.md` BURADAN AÇ + bitmiş-dilimler güncelle.
- [ ] **Step 5: Merge/push** — KULLANICI TEYİDİYLE (dış-etkili deploy). `master`'a `--no-ff` merge + push + branch sil. Deploy sonrası read-path doğrula: `/api/system-health` `alert-eval` worker + `/api/alerts`.

---

## Self-Review

**Spec coverage:** worker+watermark→T3; matchRule+v1 kapsam→T2; alert_events/watermark persist→T1; GET /api/alerts→T4; config/health/gate/main→T5; getAlerts canlı→T6; review/docs/merge→T7. Frontend değişmezliği (AlertHistoryPanel)→T6 not. ✅

**Placeholder scan:** Tüm adımlarda gerçek kod/komut var. Tek yumuşak nokta: T5 Step 6 `healthReg.Reporter(...)` — executor'a trend'i birebir kopyalaması açıkça not düşüldü (health API imzası dosyada doğrulanmalı). ✅

**Type consistency:** `MatchRule`(T2) ↔ worker(T3) çağrısı; `AlertEventRow`(T1) ↔ worker üretimi(T3) ↔ handler(T4) ↔ frontend AlertEvent; `Source`/`Sink`(T3) ↔ store metotları(T1) + main adaptör(T5); `AlertEvents` Bundle(T1)↔RouterDeps(T4)↔main(T5). `EvaluableTriggers`(T2) tek kaynak. ✅
