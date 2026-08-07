# Slice 2a — Token Safety Scoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `tokenSafety` skorunu (0-100, açıklanabilir, kural-tabanlı) gerçeğe çevir — arka plan scorer Helius on-chain sinyallerinden (launchpad-aware authority + holder yoğunlaşması + likidite) skoru hesaplayıp DB'ye yazar; detail & liste DB'den okur.

**Architecture:** Yeni `internal/safety/` paketi: saf `Scorer` (Check registry, I/O yok) + `DataProvider` (DIP, Helius) + `Worker` (Enricher deseni, periyodik). Skor + breakdown + risks + confidence + top10% DB'ye persist edilir (migration 0005); `TokenDetailService` ve token listesi DB'den okur (Option A deseni — canlı GeckoTerminal/Helius çağrısı detail yolunda yok). Helius `getAccountInfo` (authorities) + genişletilmiş `getTokenAccounts` (holder dağılımı).

**Tech Stack:** Go 1.24, chi, pgx/v5 (database/sql), goose migration, slog. Helius RPC (getAccountInfo jsonParsed + getTokenAccounts). Yeni bağımlılık YOK.

## Global Constraints

- Go sürümü: `go 1.24.0` (go.mod'da pinli; CI ile eşleşir). Yeni Go dep EKLENMEZ.
- Frontend'e DOKUNULMAZ (saf backend; seam sabit — `TokenRow.safetyScore` + `TokenDetail.scores.tokenSafety` alanları zaten var). Frontend testleri değişmez.
- Skor alanları DB'den sunulur; detail yolu canlı on-chain çağrı YAPMAZ (throttle-dayanıklı, Option A deseni). Canlı Helius yalnız arka plan Worker'da.
- Dürüstlük: skorlanmamış token (`safety_scored_ts=0`) → `confidence:0`, `safetyScore` nötr 0 (sahte "0=güvensiz" değil). Sessiz düşürme yok — kapsam dışı alanlar spec §2.2'de.
- Clean Code & SOLID: `Scorer` saf/SRP; `DataProvider`/`SafetyStore` arayüzleri DIP; Check registry OCP; dar arayüzler ISP. TDD, sık commit.
- Her task sonunda: `go build ./...`, `go vet ./...`, `go test ./...` (ilgili paket) yeşil olmalı. `gofmt -w` uygulanmış olmalı.
- Kabul: DB round-trip + canlı Helius yalnız DEPLOY'da doğrulanır (yerel Postgres/key yok — 1a-1c deseni). Kalibrasyon riski: Helius `getAccountInfo`/`getTokenAccounts` alan şekilleri.
- Çalışma dizini: `apps/api-go/`. Modül: `github.com/furkanatesc/sentinel/apps/api-go`.

---

### Task 1: Store — schema, tipler, UpdateSafety/SafetyScoreTargets, TokenDetailBase safety alanları, fake parity

**Files:**
- Create: `apps/api-go/internal/store/migrations/0005_add_token_safety.sql`
- Modify: `apps/api-go/internal/store/token_detail.go` (TokenDetailBase struct — safety alanları)
- Modify: `apps/api-go/internal/store/tokens.go` (SafetyUpdate/SafetyTarget tipleri, TokenStore arayüzü + postgres UpdateSafety/SafetyScoreTargets/TokenDetailBase SELECT)
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake parity)
- Modify: `apps/api-go/internal/store/postgres_ingest_test.go` (round-trip testi — DATABASE_URL'liyse)
- Test: `apps/api-go/internal/store/tokens_fake_test.go` (fake UpdateSafety/SafetyScoreTargets/TokenDetailBase safety)

**Interfaces:**
- Produces:
  - `store.SafetyUpdate{Mint string; Score, Confidence, Top10Pct float64; Breakdown []ScoreBreakdownItem; Risks RiskGroups; ScoredTs int64}`
  - `store.SafetyTarget{Mint string; Liquidity float64; Launchpad string}`
  - `TokenStore.UpdateSafety(ctx, SafetyUpdate) error`
  - `TokenStore.SafetyScoreTargets(ctx, limit int) ([]SafetyTarget, error)`
  - `TokenDetailBase` yeni alanlar: `SafetyScore, SafetyConfidence, Top10Pct float64; SafetyBreakdown []ScoreBreakdownItem; SafetyRisks RiskGroups; SafetyScoredTs int64`

- [ ] **Step 1: Migration 0005 yaz**

Create `apps/api-go/internal/store/migrations/0005_add_token_safety.sql`:
```sql
-- +goose Up
-- Token Safety skoru (2a): safety_score kolonu 0002'de var; breakdown/risks/confidence/top10/scored_ts eklenir.
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS top10_holder_pct  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_breakdown  TEXT NOT NULL DEFAULT ''; -- JSON []ScoreBreakdownItem
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_risks      TEXT NOT NULL DEFAULT ''; -- JSON RiskGroups
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_scored_ts  BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_scored_ts;
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_risks;
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_breakdown;
ALTER TABLE tokens DROP COLUMN IF EXISTS top10_holder_pct;
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_confidence;
```

- [ ] **Step 2: TokenDetailBase'e safety alanları ekle**

Modify `apps/api-go/internal/store/token_detail.go` — `TokenDetailBase` struct'ına ekle (mevcut alanların altına):
```go
	// 2a Token Safety (enrichment/scorer persist eder; detail okur).
	SafetyScore, SafetyConfidence, Top10Pct float64
	SafetyBreakdown                         []ScoreBreakdownItem
	SafetyRisks                             RiskGroups
	SafetyScoredTs                          int64
```

- [ ] **Step 3: tokens.go — SafetyUpdate/SafetyTarget tipleri + arayüz metotları**

Modify `apps/api-go/internal/store/tokens.go`. `MarketUpdate` tanımının altına ekle:
```go
// SafetyUpdate, 2a scorer'ının yazdığı token güvenliği sonucudur.
type SafetyUpdate struct {
	Mint       string
	Score      float64
	Confidence float64
	Top10Pct   float64
	Breakdown  []ScoreBreakdownItem
	Risks      RiskGroups
	ScoredTs   int64
}

// SafetyTarget, skorlanacak token için gereken minimum bilgidir.
type SafetyTarget struct {
	Mint      string
	Liquidity float64
	Launchpad string
}
```
`TokenStore` arayüzüne (mevcut metotların yanına) ekle:
```go
	// 2a: token güvenliği skorunu yazar / skorlanacak hedefleri döndürür.
	UpdateSafety(ctx context.Context, s SafetyUpdate) error
	SafetyScoreTargets(ctx context.Context, limit int) ([]SafetyTarget, error)
```

- [ ] **Step 4: postgres UpdateSafety + SafetyScoreTargets + TokenDetailBase SELECT genişletme yaz**

Modify `apps/api-go/internal/store/tokens.go` — postgres impl'leri ekle:
```go
func (p *postgresStore) UpdateSafety(ctx context.Context, s SafetyUpdate) error {
	bdJSON, err := json.Marshal(s.Breakdown)
	if err != nil {
		return err
	}
	rkJSON, err := json.Marshal(s.Risks)
	if err != nil {
		return err
	}
	const q = `UPDATE tokens SET safety_score=$2, safety_confidence=$3, top10_holder_pct=$4,
		safety_breakdown=$5, safety_risks=$6, safety_scored_ts=$7 WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, s.Mint, s.Score, s.Confidence, s.Top10Pct,
		string(bdJSON), string(rkJSON), s.ScoredTs)
	return err
}

func (p *postgresStore) SafetyScoreTargets(ctx context.Context, limit int) ([]SafetyTarget, error) {
	const q = `SELECT mint, liquidity, launchpad FROM tokens
		WHERE pool_address <> '' ORDER BY safety_scored_ts ASC, first_seen_ts DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SafetyTarget, 0, limit)
	for rows.Next() {
		var t SafetyTarget
		if err := rows.Scan(&t.Mint, &t.Liquidity, &t.Launchpad); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
```
Ve `TokenDetailBase` SELECT'ini genişlet (mevcut `func (p *postgresStore) TokenDetailBase`):
```go
	const q = `SELECT name, symbol, pool_address, first_seen_ts, price, liquidity,
		price_change_h24, market_cap_usd, vol24h,
		safety_score, safety_confidence, top10_holder_pct, safety_breakdown, safety_risks, safety_scored_ts
		FROM tokens WHERE mint=$1`
	var b TokenDetailBase
	var bdJSON, rkJSON string
	err := p.db.QueryRowContext(ctx, q, mint).Scan(&b.Name, &b.Symbol, &b.PoolAddr, &b.FirstSeenTs,
		&b.Price, &b.Liquidity, &b.PriceChangeH24, &b.MarketCapUSD, &b.Vol24h,
		&b.SafetyScore, &b.SafetyConfidence, &b.Top10Pct, &bdJSON, &rkJSON, &b.SafetyScoredTs)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenDetailBase{}, false, nil
	}
	if err != nil {
		return TokenDetailBase{}, false, err
	}
	b.SafetyBreakdown = parseBreakdownJSON(bdJSON)
	b.SafetyRisks = parseRiskGroupsJSON(rkJSON)
	return b, true, nil
```
Ve dosyanın sonuna JSON parse yardımcıları (parseSparkJSON deseni — boş/bozukta güvenli):
```go
func parseBreakdownJSON(s string) []ScoreBreakdownItem {
	if s == "" {
		return []ScoreBreakdownItem{}
	}
	var out []ScoreBreakdownItem
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return []ScoreBreakdownItem{}
	}
	return out
}

func parseRiskGroupsJSON(s string) RiskGroups {
	empty := RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{}, Creator: []RiskItem{}}
	if s == "" {
		return empty
	}
	var out RiskGroups
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return empty
	}
	if out.Contract == nil {
		out.Contract = []RiskItem{}
	}
	if out.Market == nil {
		out.Market = []RiskItem{}
	}
	if out.Creator == nil {
		out.Creator = []RiskItem{}
	}
	return out
}
```

- [ ] **Step 5: Fake store parity yaz (RED)**

Modify `apps/api-go/internal/store/fake_ingest.go` — `fakeTok`'a `launchpad` + safety alanları ekle (mevcut `fakeTok` alanlarının altına):
```go
	launchpad string
	// 2a safety
	safetyScore, safetyConfidence, top10Pct float64
	safetyBreakdown                         []ScoreBreakdownItem
	safetyRisks                             RiskGroups
	safetyScoredTs                          int64
```
Mevcut `UpsertDiscovered` fake'inde `cur.poolAddr = d.PoolAddr` satırının yanına `cur.launchpad = d.Launchpad` ekle (launchpad target'ta gerekir). Ve `import` bloğuna `"sort"` ekle. Fake metotları ekle:
```go
func (f *fakeTokenStore) UpdateSafety(_ context.Context, s SafetyUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.byID[s.Mint]
	if !ok {
		return nil
	}
	cur.row.SafetyScore = s.Score
	cur.safetyScore, cur.safetyConfidence, cur.top10Pct = s.Score, s.Confidence, s.Top10Pct
	cur.safetyBreakdown, cur.safetyRisks, cur.safetyScoredTs = s.Breakdown, s.Risks, s.ScoredTs
	f.byID[s.Mint] = cur
	return nil
}

func (f *fakeTokenStore) SafetyScoreTargets(_ context.Context, limit int) ([]SafetyTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]SafetyTarget, 0, limit)
	// en eski safety_scored_ts önce; poolAddr'siz atla.
	ids := append([]string{}, f.order...)
	sort.SliceStable(ids, func(i, j int) bool {
		return f.byID[ids[i]].safetyScoredTs < f.byID[ids[j]].safetyScoredTs
	})
	for _, id := range ids {
		t := f.byID[id]
		if t.poolAddr == "" || len(out) >= limit {
			continue
		}
		out = append(out, SafetyTarget{Mint: t.row.Mint, Liquidity: t.row.Liquidity, Launchpad: t.launchpad})
	}
	return out, nil
}
```
Ve `TokenDetailBase` fake'ini safety alanlarını döndürecek şekilde genişlet:
```go
	return TokenDetailBase{
		Name: t.row.Name, Symbol: t.row.Symbol, PoolAddr: t.poolAddr, FirstSeenTs: t.firstSeen,
		Price: t.row.Price, Liquidity: t.row.Liquidity,
		PriceChangeH24: t.priceChangeH24, MarketCapUSD: t.marketCapUSD, Vol24h: t.vol24h,
		SafetyScore: t.safetyScore, SafetyConfidence: t.safetyConfidence, Top10Pct: t.top10Pct,
		SafetyBreakdown: t.safetyBreakdown, SafetyRisks: t.safetyRisks, SafetyScoredTs: t.safetyScoredTs,
	}, true, nil
```

Test yaz `apps/api-go/internal/store/tokens_fake_test.go` (mevcut dosyaya ekle):
```go
func TestFakeUpdateSafetyRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewFakeTokenStore()
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "MSAFE", Symbol: "SAF", Launchpad: "Pump.fun", PoolAddr: "P", FirstSeenTs: 1})
	err := s.UpdateSafety(ctx, SafetyUpdate{
		Mint: "MSAFE", Score: 72, Confidence: 1, Top10Pct: 44,
		Breakdown: []ScoreBreakdownItem{{Label: "Freeze authority iptal", Weight: 0, Detail: "ok"}},
		Risks:     RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{{ID: "top10-conc", Title: "x", Severity: "medium"}}, Creator: []RiskItem{}},
		ScoredTs:  100,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, ok, _ := s.TokenDetailBase(ctx, "MSAFE")
	if !ok || b.SafetyScore != 72 || b.SafetyConfidence != 1 || b.Top10Pct != 44 || b.SafetyScoredTs != 100 {
		t.Fatalf("safety round-trip yanlış: %+v", b)
	}
	if len(b.SafetyBreakdown) != 1 || len(b.SafetyRisks.Market) != 1 {
		t.Fatalf("breakdown/risks taşınmadı: %+v", b)
	}
}

func TestFakeSafetyScoreTargetsOldestFirstPoolOnly(t *testing.T) {
	ctx := context.Background()
	s := NewFakeTokenStore()
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "A", Symbol: "A", Launchpad: "Pump.fun", PoolAddr: "PA", FirstSeenTs: 1})
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "B", Symbol: "B", Launchpad: "Pump.fun", PoolAddr: "", FirstSeenTs: 2}) // pool'suz → hariç
	s.UpdateSafety(ctx, SafetyUpdate{Mint: "A", ScoredTs: 50}) // A skorlandı
	s.UpsertDiscovered(ctx, DiscoveredToken{Mint: "C", Symbol: "C", Launchpad: "Raydium", PoolAddr: "PC", FirstSeenTs: 3}) // C hiç skorlanmadı (ts 0)
	got, _ := s.SafetyScoreTargets(ctx, 10)
	if len(got) != 2 {
		t.Fatalf("pool'lu 2 hedef beklenir (B hariç): %+v", got)
	}
	if got[0].Mint != "C" {
		t.Fatalf("en eski (hiç skorlanmamış) önce gelmeli: %+v", got)
	}
	if got[0].Launchpad != "Raydium" {
		t.Fatalf("launchpad taşınmalı: %+v", got[0])
	}
}
```

- [ ] **Step 6: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/store/ -run 'Safety' -v`
Expected: FAIL / build error (UpdateSafety/SafetyScoreTargets/safety alanları tanımlı değil) → Step 2-5 tamamlanınca PASS.

- [ ] **Step 7: postgres integration testine safety round-trip ekle**

Modify `apps/api-go/internal/store/postgres_ingest_test.go` — `TestPostgresMarketRoundTrip` sonuna (cleanup'tan önce) ekle:
```go
	if err := b.Tokens.UpdateSafety(ctx, SafetyUpdate{
		Mint: "MintMk", Score: 65, Confidence: 0.5, Top10Pct: 55,
		Breakdown: []ScoreBreakdownItem{{Label: "Top-10 yoğunlaşma", Weight: -15, Detail: "orta"}},
		Risks:     RiskGroups{Contract: []RiskItem{}, Market: []RiskItem{{ID: "top10-conc", Title: "x", Severity: "medium"}}, Creator: []RiskItem{}},
		ScoredTs:  200,
	}); err != nil {
		t.Fatal(err)
	}
	sb, ok, err := b.Tokens.TokenDetailBase(ctx, "MintMk")
	if err != nil || !ok || sb.SafetyScore != 65 || sb.Top10Pct != 55 || sb.SafetyScoredTs != 200 || len(sb.SafetyBreakdown) != 1 || len(sb.SafetyRisks.Market) != 1 {
		t.Fatalf("safety DB round-trip yanlış: ok=%v err=%v base=%+v", ok, err, sb)
	}
	tgts, err := b.Tokens.SafetyScoreTargets(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	var foundTgt bool
	for _, tg := range tgts {
		if tg.Mint == "MintMk" {
			foundTgt = true
		}
	}
	if !foundTgt {
		t.Fatal("SafetyScoreTargets pool'lu token'ı içermeli")
	}
```

- [ ] **Step 8: gofmt + build + test**

Run:
```bash
cd apps/api-go && gofmt -w internal/store/ && go build ./... && go test ./internal/store/ -v -run 'Safety|Market'
```
Expected: PASS (postgres testleri DATABASE_URL yoksa SKIP).

- [ ] **Step 9: Commit**

```bash
git add apps/api-go/internal/store/
git commit -m "feat(safety): store — 0005 migration + UpdateSafety/SafetyScoreTargets + TokenDetailBase safety alanları"
```

---

### Task 2: Saf Scorer + Check registry (`internal/safety/scorer.go`)

**Files:**
- Create: `apps/api-go/internal/safety/scorer.go`
- Test: `apps/api-go/internal/safety/scorer_test.go`

**Interfaces:**
- Consumes: `store.ScoreBreakdownItem`, `store.RiskItem`, `store.RiskGroups` (Task 1'den bağımsız — mevcut tipler).
- Produces:
  - `safety.Inputs{MintAuthorityActive, FreezeAuthorityActive, AuthoritiesKnown bool; HolderCount int; Top10Pct float64; HoldersKnown bool; Liquidity float64; Launchpad string}`
  - `safety.SafetyResult{Score, Confidence, Top10Pct float64; Breakdown []store.ScoreBreakdownItem; Risks store.RiskGroups}`
  - `safety.Score(in Inputs) SafetyResult`

- [ ] **Step 1: Failing test yaz (RED)**

Create `apps/api-go/internal/safety/scorer_test.go`:
```go
package safety

import "testing"

func TestScoreNoDataNeutral(t *testing.T) {
	// Hiç on-chain veri yoksa confidence 0, skor nötr 0 (sahte "güvenli" değil).
	r := Score(Inputs{AuthoritiesKnown: false, HoldersKnown: false, Liquidity: 1000})
	if r.Confidence != 0 || r.Score != 0 {
		t.Fatalf("veri yokken nötr olmalı: %+v", r)
	}
	if r.Breakdown == nil || r.Risks.Contract == nil {
		t.Fatalf("breakdown/risks boş slice olmalı (nil değil): %+v", r)
	}
}

func TestScoreCleanTokenHigh(t *testing.T) {
	// Authority iptal + dağılım sağlıklı + likidite iyi → yüksek skor, confidence 1.
	r := Score(Inputs{
		AuthoritiesKnown: true, MintAuthorityActive: false, FreezeAuthorityActive: false,
		HoldersKnown: true, HolderCount: 500, Top10Pct: 30, Liquidity: 5000, Launchpad: "Raydium",
	})
	if r.Confidence != 1 {
		t.Fatalf("iki kaynak da bilinirse confidence 1: %v", r.Confidence)
	}
	if r.Score != 100 {
		t.Fatalf("temiz token 100 olmalı: %v (%+v)", r.Score, r.Breakdown)
	}
	if len(r.Risks.Contract) != 0 || len(r.Risks.Market) != 0 {
		t.Fatalf("temiz token'da risk olmamalı: %+v", r.Risks)
	}
}

func TestScoreFreezeActiveDeductsAndRisk(t *testing.T) {
	r := Score(Inputs{
		AuthoritiesKnown: true, FreezeAuthorityActive: true, MintAuthorityActive: false,
		HoldersKnown: true, HolderCount: 500, Top10Pct: 30, Liquidity: 5000, Launchpad: "Raydium",
	})
	if r.Score != 65 { // 100 - 35
		t.Fatalf("freeze aktif -35: %v", r.Score)
	}
	if len(r.Risks.Contract) != 1 || r.Risks.Contract[0].Severity != "high" {
		t.Fatalf("freeze contract high risk üretmeli: %+v", r.Risks.Contract)
	}
}

func TestScoreMintAuthorityLaunchpadAware(t *testing.T) {
	base := Inputs{AuthoritiesKnown: true, FreezeAuthorityActive: false, MintAuthorityActive: true,
		HoldersKnown: true, HolderCount: 500, Top10Pct: 30, Liquidity: 5000}
	pump := base
	pump.Launchpad = "Pump.fun"
	if got := Score(pump).Score; got != 100 {
		t.Fatalf("pump.fun bonding-curve mint authority cezasız olmalı: %v", got)
	}
	generic := base
	generic.Launchpad = "Raydium"
	if got := Score(generic).Score; got != 80 { // 100 - 20
		t.Fatalf("genel token'da aktif mint authority -20: %v", got)
	}
}

func TestScoreTop10Bands(t *testing.T) {
	mk := func(top10 float64) float64 {
		return Score(Inputs{AuthoritiesKnown: true, HoldersKnown: true, HolderCount: 500,
			Top10Pct: top10, Liquidity: 5000, Launchpad: "Raydium"}).Score
	}
	if mk(85) != 70 { // -30
		t.Fatalf(">80%% -30: %v", mk(85))
	}
	if mk(60) != 85 { // -15
		t.Fatalf("50-80%% -15: %v", mk(60))
	}
	if mk(40) != 100 {
		t.Fatalf("<50%% cezasız: %v", mk(40))
	}
}

func TestScoreLowHolderAndLiquidity(t *testing.T) {
	r := Score(Inputs{AuthoritiesKnown: true, HoldersKnown: true, HolderCount: 10,
		Top10Pct: 30, Liquidity: 100, Launchpad: "Raydium"})
	if r.Score != 80 { // -10 holder<20, -10 liq<500
		t.Fatalf("düşük holder+likidite -20: %v (%+v)", r.Score, r.Breakdown)
	}
}

func TestScorePartialConfidence(t *testing.T) {
	// Yalnız holder verisi var, authority yok → confidence 0.5, authority Check'leri atlanır.
	r := Score(Inputs{AuthoritiesKnown: false, HoldersKnown: true, HolderCount: 500,
		Top10Pct: 30, Liquidity: 5000, Launchpad: "Raydium"})
	if r.Confidence != 0.5 {
		t.Fatalf("tek kaynak → confidence 0.5: %v", r.Confidence)
	}
	for _, it := range r.Breakdown {
		if it.Label == "Freeze authority iptal" || it.Label == "Mint authority iptal" {
			t.Fatalf("authority bilinmiyorken authority kalemi olmamalı: %+v", r.Breakdown)
		}
	}
}

func TestScoreClampFloor(t *testing.T) {
	// Tüm bayraklar → düşüş 100'ü aşar, 0'a clamp.
	r := Score(Inputs{AuthoritiesKnown: true, FreezeAuthorityActive: true, MintAuthorityActive: true,
		HoldersKnown: true, HolderCount: 5, Top10Pct: 95, Liquidity: 10, Launchpad: "Raydium"})
	if r.Score != 0 {
		t.Fatalf("düşüşler 0'a clamp olmalı: %v", r.Score)
	}
}
```

- [ ] **Step 2: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestScore -v`
Expected: FAIL (paket/`Score` yok — build error).

- [ ] **Step 3: Scorer'ı yaz (GREEN)**

Create `apps/api-go/internal/safety/scorer.go`:
```go
// Package safety, token güvenliği skorunu (2a) üretir: saf kural-tabanlı Scorer +
// on-chain veri sağlayıcı (DIP) + periyodik Worker.
package safety

import (
	"fmt"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Ağırlıklar/eşikler (v1 sabitleri; deploy'da kalibre edilebilir).
const (
	wFreezeActive = 35.0
	wMintActive   = 20.0
	wTop10High    = 30.0 // > top10HighPct
	wTop10Mid     = 15.0 // >= top10MidPct
	wHolderLow    = 10.0 // < holderLowN
	wLiqLow       = 10.0 // < liqFloor

	top10HighPct = 80.0
	top10MidPct  = 50.0
	holderLowN   = 20
	liqFloor     = 500.0
)

// Inputs, Scorer'ın tek girdisidir (saf — I/O yok).
type Inputs struct {
	MintAuthorityActive, FreezeAuthorityActive bool
	AuthoritiesKnown                           bool // getAccountInfo başarılı mı
	HolderCount                                int
	Top10Pct                                   float64
	HoldersKnown                               bool // getTokenAccounts başarılı mı
	Liquidity                                  float64
	Launchpad                                  string
}

// SafetyResult, skorlama çıktısıdır (frontend TokenDetail seam'ine eşlenir).
type SafetyResult struct {
	Score      float64
	Confidence float64
	Top10Pct   float64
	Breakdown  []store.ScoreBreakdownItem
	Risks      store.RiskGroups
}

// checkOutcome, tek bir güvenlik kontrolünün sonucudur.
type checkOutcome struct {
	applies   bool // false → veri yok, atla (breakdown/risk üretmez)
	deduction float64
	item      store.ScoreBreakdownItem
	risk      *store.RiskItem
	riskGroup string // "contract" | "market"
}

// check, saf bir güvenlik kontrolüdür (OCP: yeni kontrol eklemek Score'u değiştirmez).
type check func(in Inputs) checkOutcome

// checks, çalıştırılan kontrol kayıt defteridir (sıra breakdown sırasını belirler).
var checks = []check{freezeCheck, mintCheck, top10Check, holderCountCheck, liquidityCheck}

// Score, girdiden 0-100 güvenlik skoru + açıklanabilir breakdown + risks + confidence üretir.
func Score(in Inputs) SafetyResult {
	conf := 0.0
	if in.AuthoritiesKnown {
		conf += 0.5
	}
	if in.HoldersKnown {
		conf += 0.5
	}
	rg := store.RiskGroups{Contract: []store.RiskItem{}, Market: []store.RiskItem{}, Creator: []store.RiskItem{}}
	bd := []store.ScoreBreakdownItem{}
	if conf == 0 {
		// Hiç on-chain veri yok → dürüst nötr (sahte "güvenli" DEĞİL).
		return SafetyResult{Score: 0, Confidence: 0, Top10Pct: 0, Breakdown: bd, Risks: rg}
	}
	score := 100.0
	for _, c := range checks {
		o := c(in)
		if !o.applies {
			continue
		}
		score -= o.deduction
		bd = append(bd, o.item)
		if o.risk != nil {
			switch o.riskGroup {
			case "contract":
				rg.Contract = append(rg.Contract, *o.risk)
			case "market":
				rg.Market = append(rg.Market, *o.risk)
			}
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return SafetyResult{Score: score, Confidence: conf, Top10Pct: in.Top10Pct, Breakdown: bd, Risks: rg}
}

func item(label string, weight float64, detail string) store.ScoreBreakdownItem {
	return store.ScoreBreakdownItem{Label: label, Weight: weight, Detail: detail}
}

func riskItem(id, title, severity, desc string) *store.RiskItem {
	return &store.RiskItem{ID: id, Title: title, Severity: severity, Description: desc, FirstSeen: "—", LastSeen: "—"}
}

func isBondingCurve(launchpad string) bool {
	return launchpad == "Pump.fun" || launchpad == "pump.fun" || launchpad == "PumpSwap"
}

func freezeCheck(in Inputs) checkOutcome {
	if !in.AuthoritiesKnown {
		return checkOutcome{}
	}
	if in.FreezeAuthorityActive {
		return checkOutcome{applies: true, deduction: wFreezeActive,
			item: item("Freeze authority aktif", -wFreezeActive, "Sahip token hesaplarını dondurabilir (honeypot riski)."),
			risk: riskItem("freeze-authority", "Freeze authority aktif", "high", "Sahip hesapları dondurup satışı engelleyebilir."), riskGroup: "contract"}
	}
	return checkOutcome{applies: true, deduction: 0, item: item("Freeze authority iptal", 0, "Dondurma riski yok.")}
}

func mintCheck(in Inputs) checkOutcome {
	if !in.AuthoritiesKnown {
		return checkOutcome{}
	}
	switch {
	case !in.MintAuthorityActive:
		return checkOutcome{applies: true, deduction: 0, item: item("Mint authority iptal", 0, "Ek arz basılamaz.")}
	case isBondingCurve(in.Launchpad):
		return checkOutcome{applies: true, deduction: 0, item: item("Mint authority bonding-curve", 0, "pump.fun eğrisi arzı sabitler — beklenen.")}
	default:
		return checkOutcome{applies: true, deduction: wMintActive,
			item: item("Mint authority aktif", -wMintActive, "Sahip ek arz basabilir (dilution riski)."),
			risk: riskItem("mint-authority", "Mint authority aktif", "medium", "Sahip token arzını artırabilir."), riskGroup: "contract"}
	}
}

func top10Check(in Inputs) checkOutcome {
	if !in.HoldersKnown {
		return checkOutcome{}
	}
	switch {
	case in.Top10Pct > top10HighPct:
		return checkOutcome{applies: true, deduction: wTop10High,
			item: item("Top-10 holder yoğunlaşması yüksek", -wTop10High, fmt.Sprintf("Top-10 %%%.0f — dump/rug riski.", in.Top10Pct)),
			risk: riskItem("top10-concentration", "Yüksek holder yoğunlaşması", "high", "Az sayıda cüzdan arzın çoğunu tutuyor."), riskGroup: "market"}
	case in.Top10Pct >= top10MidPct:
		return checkOutcome{applies: true, deduction: wTop10Mid,
			item: item("Top-10 holder yoğunlaşması orta", -wTop10Mid, fmt.Sprintf("Top-10 %%%.0f.", in.Top10Pct)),
			risk: riskItem("top10-concentration", "Orta holder yoğunlaşması", "medium", "Holder dağılımı orta düzeyde yoğun."), riskGroup: "market"}
	default:
		return checkOutcome{applies: true, deduction: 0, item: item("Holder dağılımı sağlıklı", 0, fmt.Sprintf("Top-10 %%%.0f.", in.Top10Pct))}
	}
}

func holderCountCheck(in Inputs) checkOutcome {
	if !in.HoldersKnown {
		return checkOutcome{}
	}
	if in.HolderCount < holderLowN {
		return checkOutcome{applies: true, deduction: wHolderLow,
			item: item("Holder sayısı düşük", -wHolderLow, fmt.Sprintf("%d holder — ince/rug-eğilimli.", in.HolderCount)),
			risk: riskItem("low-holders", "Az holder", "low", "Çok az cüzdan tutuyor."), riskGroup: "market"}
	}
	return checkOutcome{applies: true, deduction: 0, item: item("Holder sayısı yeterli", 0, fmt.Sprintf("%d holder.", in.HolderCount))}
}

func liquidityCheck(in Inputs) checkOutcome {
	if in.Liquidity < liqFloor {
		return checkOutcome{applies: true, deduction: wLiqLow,
			item: item("Likidite düşük", -wLiqLow, fmt.Sprintf("$%.0f — illikit/rug-eğilimli.", in.Liquidity)),
			risk: riskItem("low-liquidity", "Düşük likidite", "low", "Havuz likiditesi düşük."), riskGroup: "market"}
	}
	return checkOutcome{applies: true, deduction: 0, item: item("Likidite yeterli", 0, fmt.Sprintf("$%.0f.", in.Liquidity))}
}
```
NOT: `liquidityCheck` on-chain veri gerektirmez (DB'den) → her zaman `applies:true`. Ama `conf==0` (hiç authority+holder yok) durumunda Score erken döner, likidite kalemi de üretilmez — dürüst nötr. Bu kasıtlı.

- [ ] **Step 4: Testi çalıştır — geçmeli**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestScore -v`
Expected: PASS (8 test).

- [ ] **Step 5: gofmt + commit**

```bash
cd apps/api-go && gofmt -w internal/safety/
git add apps/api-go/internal/safety/scorer.go apps/api-go/internal/safety/scorer_test.go
git commit -m "feat(safety): saf Scorer + Check registry (launchpad-aware authority + top10 + holder + likidite)"
```

---

### Task 3: Helius authorities client (`internal/ingest/authorities.go`)

**Files:**
- Create: `apps/api-go/internal/ingest/authorities.go`
- Test: `apps/api-go/internal/ingest/authorities_test.go`

**Interfaces:**
- Produces: `ingest.HeliusAuthorities` + `NewHeliusAuthorities(rpcURL string) *HeliusAuthorities` + `(*HeliusAuthorities).MintAuthorities(ctx, mint string) (mintActive, freezeActive bool, err error)`

- [ ] **Step 1: Failing test yaz (RED)**

Create `apps/api-go/internal/ingest/authorities_test.go`:
```go
package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func authServer(t *testing.T, body string) *HeliusAuthorities {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewHeliusAuthorities(srv.URL)
}

func TestMintAuthoritiesBothRevoked(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":"1","result":{"value":{"data":{"parsed":{"info":{"mintAuthority":null,"freezeAuthority":null}}}}}}`
	h := authServer(t, body)
	mintA, freezeA, err := h.MintAuthorities(context.Background(), "MintX")
	if err != nil {
		t.Fatal(err)
	}
	if mintA || freezeA {
		t.Fatalf("null authority → aktif değil: mint=%v freeze=%v", mintA, freezeA)
	}
}

func TestMintAuthoritiesBothActive(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":"1","result":{"value":{"data":{"parsed":{"info":{"mintAuthority":"Abc111","freezeAuthority":"Def222"}}}}}}`
	h := authServer(t, body)
	mintA, freezeA, err := h.MintAuthorities(context.Background(), "MintX")
	if err != nil {
		t.Fatal(err)
	}
	if !mintA || !freezeA {
		t.Fatalf("dolu authority → aktif: mint=%v freeze=%v", mintA, freezeA)
	}
}

func TestMintAuthoritiesRPCError(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":"1","error":{"message":"boom"}}`
	h := authServer(t, body)
	if _, _, err := h.MintAuthorities(context.Background(), "MintX"); err == nil {
		t.Fatal("RPC error hata dönmeli")
	}
}
```

- [ ] **Step 2: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestMintAuthorities -v`
Expected: FAIL (HeliusAuthorities yok).

- [ ] **Step 3: authorities.go yaz (GREEN)**

Create `apps/api-go/internal/ingest/authorities.go`:
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

// HeliusAuthorities, bir SPL mint'inin mint/freeze authority durumunu getAccountInfo
// (jsonParsed) ile çeker. null authority = iptal (aktif değil).
type HeliusAuthorities struct {
	rpcURL string
	http   *http.Client
}

func NewHeliusAuthorities(rpcURL string) *HeliusAuthorities {
	return &HeliusAuthorities{rpcURL: rpcURL, http: &http.Client{Timeout: 8 * time.Second}}
}

// MintAuthorities, mint ve freeze authority'nin aktif (dolu) olup olmadığını döndürür.
func (h *HeliusAuthorities) MintAuthorities(ctx context.Context, mint string) (mintActive, freezeActive bool, err error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getAccountInfo",
		"params": []any{mint, map[string]any{"encoding": "jsonParsed"}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return false, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return false, false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false, false, fmt.Errorf("helius getAccountInfo: status %d", res.StatusCode)
	}
	var r struct {
		Result struct {
			Value struct {
				Data struct {
					Parsed struct {
						Info struct {
							MintAuthority   *string `json:"mintAuthority"`
							FreezeAuthority *string `json:"freezeAuthority"`
						} `json:"info"`
					} `json:"parsed"`
				} `json:"data"`
			} `json:"value"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return false, false, err
	}
	if r.Error != nil {
		return false, false, fmt.Errorf("helius getAccountInfo error: %s", r.Error.Message)
	}
	info := r.Result.Value.Data.Parsed.Info
	return info.MintAuthority != nil, info.FreezeAuthority != nil, nil
}
```

- [ ] **Step 4: Testi çalıştır — geçmeli**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestMintAuthorities -v`
Expected: PASS (3 test).

- [ ] **Step 5: gofmt + commit**

```bash
cd apps/api-go && gofmt -w internal/ingest/authorities.go internal/ingest/authorities_test.go
git add apps/api-go/internal/ingest/authorities.go apps/api-go/internal/ingest/authorities_test.go
git commit -m "feat(safety): Helius getAccountInfo authorities client (mint/freeze revoked tespiti)"
```

---

### Task 4: Helius holder dağılımı (`internal/ingest/holders.go` genişletme)

**Files:**
- Modify: `apps/api-go/internal/ingest/holders.go` (yeni `HolderDistribution` metodu; `pageCount`'u owner+amount döndürecek şekilde yeniden kullan)
- Test: `apps/api-go/internal/ingest/holders_test.go` (mevcut dosyaya ekle)

**Interfaces:**
- Produces: `(*HeliusHolders).HolderDistribution(ctx, mint string, cap int) (count int, top10Pct float64, capped bool, err error)`

- [ ] **Step 1: Failing test yaz (RED)**

`apps/api-go/internal/ingest/holders_test.go`'a ekle (yoksa oluştur, `package ingest`):
```go
func TestHolderDistributionTop10(t *testing.T) {
	// 12 hesap; 2 hesap aynı owner (birleşmeli). Toplam amount 120; top-10 owner toplamı hesaplanır.
	page := `{"jsonrpc":"2.0","id":"1","result":{"token_accounts":[
		{"owner":"o1","amount":"50"},{"owner":"o2","amount":"20"},{"owner":"o3","amount":"10"},
		{"owner":"o4","amount":"8"},{"owner":"o5","amount":"7"},{"owner":"o6","amount":"6"},
		{"owner":"o7","amount":"5"},{"owner":"o8","amount":"4"},{"owner":"o9","amount":"3"},
		{"owner":"o10","amount":"2"},{"owner":"o11","amount":"3"},{"owner":"o1","amount":"2"}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(page)) // tek sayfa (12 < 1000 → son sayfa)
	}))
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	count, top10, capped, err := h.HolderDistribution(context.Background(), "MintX", 5000)
	if err != nil || capped {
		t.Fatalf("err=%v capped=%v", err, capped)
	}
	// unique owner: o1(52),o2..o10,o11 = 11 owner. Toplam = 120.
	if count != 11 {
		t.Fatalf("unique owner sayısı=%d want 11", count)
	}
	// top-10 owner (en büyük 10): 52+20+10+8+7+6+5+4+3+3 = 118; %118/120 = 98.33
	if top10 < 98.0 || top10 > 98.7 {
		t.Fatalf("top10Pct=%.2f want ~98.3", top10)
	}
}
```
NOT: `holders_test.go` mevcut değilse import başlığı ekle: `import ("context"; "net/http"; "net/http/httptest"; "testing")`.

- [ ] **Step 2: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/ingest/ -run TestHolderDistribution -v`
Expected: FAIL (HolderDistribution yok).

- [ ] **Step 3: HolderDistribution'ı yaz (GREEN)**

Modify `apps/api-go/internal/ingest/holders.go` — `pageCount`'u owner+amount toplayan bir sayfa çekiciyle tamamla ve yeni metot ekle. Dosyaya ekle:
```go
// tokenAccount, getTokenAccounts sayfa öğesinin sadece gereken alanlarıdır.
type tokenAccount struct {
	Owner  string `json:"owner"`
	Amount string `json:"amount"`
}

// pageAccounts, tek sayfayı çeker (owner+amount). Kısa sayfa (< limit) → son sayfa.
func (h *HeliusHolders) pageAccounts(ctx context.Context, mint string, page int) ([]tokenAccount, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getTokenAccounts",
		"params": map[string]any{"mint": mint, "page": page, "limit": holdersPageLimit},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("helius getTokenAccounts: status %d", res.StatusCode)
	}
	var r struct {
		Result struct {
			TokenAccounts []tokenAccount `json:"token_accounts"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, fmt.Errorf("helius getTokenAccounts error: %s", r.Error.Message)
	}
	return r.Result.TokenAccounts, nil
}

// HolderDistribution, benzersiz-sahip holder sayısı ve top-10 sahip yoğunlaşması (%) döndürür.
// cap'e ulaşınca durur (capped=true — pahalı büyük token'ları sınırlar; sonuç alt-sınırdır).
func (h *HeliusHolders) HolderDistribution(ctx context.Context, mint string, cap int) (int, float64, bool, error) {
	if cap <= 0 {
		cap = 5000
	}
	byOwner := map[string]float64{}
	seen := 0
	capped := false
	for page := 1; ; page++ {
		accs, err := h.pageAccounts(ctx, mint, page)
		if err != nil {
			return 0, 0, false, err
		}
		for _, a := range accs {
			amt, _ := strconv.ParseFloat(a.Amount, 64)
			byOwner[a.Owner] += amt
		}
		seen += len(accs)
		if seen >= cap {
			capped = true
			break
		}
		if len(accs) < holdersPageLimit {
			break
		}
	}
	var total float64
	amounts := make([]float64, 0, len(byOwner))
	for _, v := range byOwner {
		amounts = append(amounts, v)
		total += v
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(amounts)))
	var top10 float64
	for i := 0; i < len(amounts) && i < 10; i++ {
		top10 += amounts[i]
	}
	pct := 0.0
	if total > 0 {
		pct = top10 / total * 100
	}
	return len(byOwner), pct, capped, nil
}
```
Import ekle: `"sort"` ve `"strconv"` (`holders.go` importlarına).

- [ ] **Step 4: Testi çalıştır — geçmeli (mevcut HoldersCount testleri de yeşil kalmalı)**

Run: `cd apps/api-go && go test ./internal/ingest/ -run 'HolderDistribution|Holders' -v`
Expected: PASS.

- [ ] **Step 5: gofmt + commit**

```bash
cd apps/api-go && gofmt -w internal/ingest/holders.go internal/ingest/holders_test.go
git add apps/api-go/internal/ingest/holders.go apps/api-go/internal/ingest/holders_test.go
git commit -m "feat(safety): Helius getTokenAccounts holder dağılımı (unique owner + top-10 yoğunlaşma)"
```

---

### Task 5: SafetyDataProvider (`internal/safety/provider.go`)

**Files:**
- Create: `apps/api-go/internal/safety/provider.go`
- Test: `apps/api-go/internal/safety/provider_test.go`

**Interfaces:**
- Consumes: `ingest.HeliusAuthorities.MintAuthorities` (Task 3), `ingest.HeliusHolders.HolderDistribution` (Task 4).
- Produces:
  - `safety.OnChainData{MintAuthorityActive, FreezeAuthorityActive, AuthoritiesKnown bool; HolderCount int; Top10Pct float64; HoldersKnown bool}`
  - `safety.DataProvider` arayüzü: `FetchOnChain(ctx, mint string) (OnChainData, error)`
  - `safety.Authorities` arayüzü: `MintAuthorities(ctx, mint string) (mintActive, freezeActive bool, err error)` (DIP — ingest.HeliusAuthorities karşılar)
  - `safety.Holders` arayüzü: `HolderDistribution(ctx, mint string, cap int) (count int, top10Pct float64, capped bool, err error)` (DIP — ingest.HeliusHolders karşılar)
  - `safety.NewHeliusProvider(auth Authorities, holders Holders, holdersCap int) *HeliusProvider`

- [ ] **Step 1: Failing test yaz (RED)**

Create `apps/api-go/internal/safety/provider_test.go`:
```go
package safety

import (
	"context"
	"errors"
	"testing"
)

type fakeAuth struct {
	mint, freeze bool
	err          error
}

func (f fakeAuth) MintAuthorities(context.Context, string) (bool, bool, error) {
	return f.mint, f.freeze, f.err
}

type fakeHolders struct {
	count  int
	top10  float64
	capped bool
	err    error
}

func (f fakeHolders) HolderDistribution(context.Context, string, int) (int, float64, bool, error) {
	return f.count, f.top10, f.capped, f.err
}

func TestFetchOnChainBothOK(t *testing.T) {
	p := NewHeliusProvider(fakeAuth{mint: true, freeze: false}, fakeHolders{count: 300, top10: 42}, 5000)
	d, err := p.FetchOnChain(context.Background(), "M")
	if err != nil {
		t.Fatal(err)
	}
	if !d.AuthoritiesKnown || !d.HoldersKnown || !d.MintAuthorityActive || d.FreezeAuthorityActive || d.HolderCount != 300 || d.Top10Pct != 42 {
		t.Fatalf("beklenmeyen: %+v", d)
	}
}

func TestFetchOnChainPartialFailureIsolated(t *testing.T) {
	// Authority hata verir, holders başarılı → AuthoritiesKnown=false, HoldersKnown=true, hard-fail YOK.
	p := NewHeliusProvider(fakeAuth{err: errors.New("boom")}, fakeHolders{count: 300, top10: 42}, 5000)
	d, err := p.FetchOnChain(context.Background(), "M")
	if err != nil {
		t.Fatalf("kısmi hata hard-fail olmamalı: %v", err)
	}
	if d.AuthoritiesKnown {
		t.Fatal("authority hatası → AuthoritiesKnown=false")
	}
	if !d.HoldersKnown || d.HolderCount != 300 {
		t.Fatalf("holders yine de bilinmeli: %+v", d)
	}
}
```

- [ ] **Step 2: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestFetchOnChain -v`
Expected: FAIL (provider yok).

- [ ] **Step 3: provider.go yaz (GREEN)**

Create `apps/api-go/internal/safety/provider.go`:
```go
package safety

import "context"

// OnChainData, Scorer'ın on-chain girdisidir (Known bayrakları kısmi-veriyi taşır).
type OnChainData struct {
	MintAuthorityActive, FreezeAuthorityActive bool
	AuthoritiesKnown                           bool
	HolderCount                                int
	Top10Pct                                   float64
	HoldersKnown                               bool
}

// DataProvider, bir mint için on-chain güvenlik verisini sağlar (DIP).
type DataProvider interface {
	FetchOnChain(ctx context.Context, mint string) (OnChainData, error)
}

// Authorities/Holders, HeliusProvider'ın bağımlı olduğu dar arayüzlerdir (DIP/ISP;
// ingest.HeliusAuthorities / ingest.HeliusHolders karşılar).
type Authorities interface {
	MintAuthorities(ctx context.Context, mint string) (mintActive, freezeActive bool, err error)
}
type Holders interface {
	HolderDistribution(ctx context.Context, mint string, cap int) (count int, top10Pct float64, capped bool, err error)
}

// HeliusProvider, authorities + holder dağılımını birleştiren somut DataProvider'dır.
// Kaynaklar bağımsız: biri hata verse diğeri yine de raporlanır (Known bayrağı ile).
type HeliusProvider struct {
	auth       Authorities
	holders    Holders
	holdersCap int
}

func NewHeliusProvider(auth Authorities, holders Holders, holdersCap int) *HeliusProvider {
	return &HeliusProvider{auth: auth, holders: holders, holdersCap: holdersCap}
}

func (p *HeliusProvider) FetchOnChain(ctx context.Context, mint string) (OnChainData, error) {
	var d OnChainData
	if mintA, freezeA, err := p.auth.MintAuthorities(ctx, mint); err == nil {
		d.MintAuthorityActive, d.FreezeAuthorityActive, d.AuthoritiesKnown = mintA, freezeA, true
	}
	if count, top10, _, err := p.holders.HolderDistribution(ctx, mint, p.holdersCap); err == nil {
		d.HolderCount, d.Top10Pct, d.HoldersKnown = count, top10, true
	}
	return d, nil
}

var _ DataProvider = (*HeliusProvider)(nil)
```

- [ ] **Step 4: Testi çalıştır — geçmeli**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestFetchOnChain -v`
Expected: PASS.

- [ ] **Step 5: gofmt + commit**

```bash
cd apps/api-go && gofmt -w internal/safety/provider.go internal/safety/provider_test.go
git add apps/api-go/internal/safety/provider.go apps/api-go/internal/safety/provider_test.go
git commit -m "feat(safety): SafetyDataProvider (authorities + holder dağılımı, kısmi-hata izole)"
```

---

### Task 6: Worker (`internal/safety/worker.go`)

**Files:**
- Create: `apps/api-go/internal/safety/worker.go`
- Test: `apps/api-go/internal/safety/worker_test.go`

**Interfaces:**
- Consumes: `safety.DataProvider` (Task 5), `safety.Score` (Task 2), `store.SafetyTarget`/`store.SafetyUpdate`/`store.ScoreBreakdownItem` (Task 1).
- Produces:
  - `safety.SafetyStore` arayüzü: `SafetyScoreTargets(ctx, limit int) ([]store.SafetyTarget, error)`, `UpdateSafety(ctx, store.SafetyUpdate) error` (store.TokenStore karşılar)
  - `safety.WorkerDeps{Store SafetyStore; Provider DataProvider; Interval time.Duration; Limit int; Now func() int64; Logger *slog.Logger}`
  - `safety.NewWorker(WorkerDeps) *Worker` + `(*Worker).Run(ctx)` + `(*Worker).scoreOnce(ctx) error`

- [ ] **Step 1: Failing test yaz (RED)**

Create `apps/api-go/internal/safety/worker_test.go`:
```go
package safety

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type fakeSafetyStore struct {
	targets []store.SafetyTarget
	updates []store.SafetyUpdate
}

func (f *fakeSafetyStore) SafetyScoreTargets(context.Context, int) ([]store.SafetyTarget, error) {
	return f.targets, nil
}
func (f *fakeSafetyStore) UpdateSafety(_ context.Context, s store.SafetyUpdate) error {
	f.updates = append(f.updates, s)
	return nil
}

type stubProvider struct{ d OnChainData }

func (s stubProvider) FetchOnChain(context.Context, string) (OnChainData, error) { return s.d, nil }

func TestScoreOncePersistsResult(t *testing.T) {
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000, Launchpad: "Raydium"}}}
	prov := stubProvider{d: OnChainData{AuthoritiesKnown: true, HoldersKnown: true, HolderCount: 500, Top10Pct: 30}}
	w := NewWorker(WorkerDeps{Store: st, Provider: prov, Limit: 10, Now: func() int64 { return 999 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(st.updates) != 1 {
		t.Fatalf("1 update beklenir: %d", len(st.updates))
	}
	u := st.updates[0]
	if u.Mint != "M1" || u.Score != 100 || u.Confidence != 1 || u.ScoredTs != 999 {
		t.Fatalf("update yanlış: %+v", u)
	}
	if len(u.Breakdown) == 0 {
		t.Fatal("breakdown persist edilmeli")
	}
}

func TestScoreOnceLiquidityLaunchpadFromTarget(t *testing.T) {
	// Aktif mint authority + pump.fun → cezasız (launchpad target'tan gelmeli).
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000, Launchpad: "Pump.fun"}}}
	prov := stubProvider{d: OnChainData{AuthoritiesKnown: true, MintAuthorityActive: true, HoldersKnown: true, HolderCount: 500, Top10Pct: 30}}
	w := NewWorker(WorkerDeps{Store: st, Provider: prov, Limit: 10, Now: func() int64 { return 1 }})
	w.scoreOnce(context.Background())
	if st.updates[0].Score != 100 {
		t.Fatalf("pump.fun mint authority cezasız olmalı: %v", st.updates[0].Score)
	}
}
```

- [ ] **Step 2: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestScoreOnce -v`
Expected: FAIL (Worker yok).

- [ ] **Step 3: worker.go yaz (GREEN)**

Create `apps/api-go/internal/safety/worker.go`:
```go
package safety

import (
	"context"
	"log/slog"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// SafetyStore, Worker'ın kalıcılık bağımlılığıdır (DIP; store.TokenStore karşılar).
type SafetyStore interface {
	SafetyScoreTargets(ctx context.Context, limit int) ([]store.SafetyTarget, error)
	UpdateSafety(ctx context.Context, s store.SafetyUpdate) error
}

type WorkerDeps struct {
	Store    SafetyStore
	Provider DataProvider
	Interval time.Duration
	Limit    int
	Now      func() int64
	Logger   *slog.Logger
}

// Worker, periyodik olarak skorlanacak token'ları çekip skorlayıp DB'ye yazar (Enricher deseni).
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
	if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
		w.d.Logger.Warn("safety score", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.scoreOnce(ctx); err != nil && ctx.Err() == nil {
				w.d.Logger.Warn("safety score", "err", err)
			}
		}
	}
}

// scoreOnce, bir döngü: hedefleri çek → her birini skorla → persist (kısmi hata izole).
func (w *Worker) scoreOnce(ctx context.Context) error {
	targets, err := w.d.Store.SafetyScoreTargets(ctx, w.d.Limit)
	if err != nil {
		return err
	}
	now := w.d.Now()
	for _, tg := range targets {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		data, err := w.d.Provider.FetchOnChain(ctx, tg.Mint)
		if err != nil {
			w.d.Logger.Warn("fetch on-chain", "mint", tg.Mint, "err", err)
			continue
		}
		res := Score(Inputs{
			MintAuthorityActive: data.MintAuthorityActive, FreezeAuthorityActive: data.FreezeAuthorityActive,
			AuthoritiesKnown: data.AuthoritiesKnown, HolderCount: data.HolderCount, Top10Pct: data.Top10Pct,
			HoldersKnown: data.HoldersKnown, Liquidity: tg.Liquidity, Launchpad: tg.Launchpad,
		})
		if err := w.d.Store.UpdateSafety(ctx, store.SafetyUpdate{
			Mint: tg.Mint, Score: res.Score, Confidence: res.Confidence, Top10Pct: res.Top10Pct,
			Breakdown: res.Breakdown, Risks: res.Risks, ScoredTs: now,
		}); err != nil {
			w.d.Logger.Warn("update safety", "mint", tg.Mint, "err", err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Testi çalıştır — geçmeli**

Run: `cd apps/api-go && go test ./internal/safety/ -v`
Expected: PASS (scorer + provider + worker testleri).

- [ ] **Step 5: gofmt + commit**

```bash
cd apps/api-go && gofmt -w internal/safety/worker.go internal/safety/worker_test.go
git add apps/api-go/internal/safety/worker.go apps/api-go/internal/safety/worker_test.go
git commit -m "feat(safety): periyodik Worker (SafetyScoreTargets → provider → Score → UpdateSafety)"
```

---

### Task 7: Detail bağlama — `TokenDetailService.Build` safety alanlarını DB'den doldurur

**Files:**
- Modify: `apps/api-go/internal/market/detail.go` (Build — tokenSafety skoru + risks + top10 base'den)
- Test: `apps/api-go/internal/market/detail_test.go` (yeni test + `fakeDetailStore`'a safety base)

**Interfaces:**
- Consumes: `store.TokenDetailBase` safety alanları (Task 1).

- [ ] **Step 1: Failing test yaz (RED)**

`apps/api-go/internal/market/detail_test.go`'a ekle:
```go
func TestBuildTokenSafetyFromStore(t *testing.T) {
	dp := &detailProvider{pools: []Pool{{PoolAddr: "P1", Mint: "M1"}}}
	fs := &fakeDetailStore{base: map[string]store.TokenDetailBase{
		"M1": {Name: "One", Symbol: "ONE", PoolAddr: "P1", FirstSeenTs: 0,
			SafetyScore: 72, SafetyConfidence: 1, Top10Pct: 44, SafetyScoredTs: 500,
			SafetyBreakdown: []store.ScoreBreakdownItem{{Label: "Freeze authority iptal", Weight: 0, Detail: "ok"}},
			SafetyRisks:     store.RiskGroups{Contract: []store.RiskItem{}, Market: []store.RiskItem{{ID: "top10-concentration", Title: "x", Severity: "medium"}}, Creator: []store.RiskItem{}}},
	}}
	svc := NewTokenDetailService(TokenDetailDeps{Store: fs, Provider: dp, Holders: &fakeHolders{n: 5}, Now: func() int64 { return 600 }})
	d, ok, err := svc.Build(context.Background(), "M1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	sd := d.Scores["tokenSafety"]
	if sd.Value != 72 || sd.Confidence != 1 || len(sd.Breakdown) != 1 {
		t.Fatalf("tokenSafety skoru DB'den gelmeli: %+v", sd)
	}
	if d.Metrics.Top10HolderPct != 44 {
		t.Fatalf("top10HolderPct DB'den: %v", d.Metrics.Top10HolderPct)
	}
	if len(d.Risks.Market) != 1 || d.Risks.Market[0].ID != "top10-concentration" {
		t.Fatalf("safety market risk detaya taşınmalı: %+v", d.Risks.Market)
	}
	// Diğer 3 skor nötr kalmalı.
	if d.Scores["opportunity"].Confidence != 0 {
		t.Fatalf("opportunity nötr kalmalı: %+v", d.Scores["opportunity"])
	}
}
```

- [ ] **Step 2: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/market/ -run TestBuildTokenSafety -v`
Expected: FAIL (tokenSafety hâlâ nötr; top10/risks boş).

- [ ] **Step 3: detail.go Build'i güncelle (GREEN)**

Modify `apps/api-go/internal/market/detail.go` — Holders bloğundan sonra (return'den önce) ekle:
```go
	// Token Safety (2a) — DB'den (arka plan scorer persist etti); diğer 3 skor nötr kalır.
	updatedAt := "—"
	if base.SafetyScoredTs > 0 {
		updatedAt = time.Unix(base.SafetyScoredTs, 0).UTC().Format(time.RFC3339)
	}
	d.Scores["tokenSafety"] = store.ScoreDetail{
		Key: "tokenSafety", Value: base.SafetyScore, Confidence: base.SafetyConfidence,
		UpdatedAt: updatedAt, Breakdown: base.SafetyBreakdown,
	}
	if d.Scores["tokenSafety"].Breakdown == nil {
		s := d.Scores["tokenSafety"]
		s.Breakdown = []store.ScoreBreakdownItem{}
		d.Scores["tokenSafety"] = s
	}
	if base.SafetyRisks.Contract != nil {
		d.Risks.Contract = base.SafetyRisks.Contract
	}
	if base.SafetyRisks.Market != nil {
		d.Risks.Market = base.SafetyRisks.Market
	}
	d.Metrics.Top10HolderPct = base.Top10Pct
```
(`time` importu `detail.go`'da zaten var.)

- [ ] **Step 4: Testi çalıştır — geçmeli (mevcut detail testleri de yeşil)**

Run: `cd apps/api-go && go test ./internal/market/ -v`
Expected: PASS (safety testi + regresyon — `TestBuildNeutralPlaceholders` hâlâ diğer 3 skoru nötr bulmalı; tokenSafety artık base'den ama testte base safety alanları 0 → Value 0/Confidence 0, yine "nötr" görünür → geçer).

NOT: `TestBuildNeutralPlaceholders` tüm 4 skorun `Value==0` olmasını bekliyorsa ve tokenSafety artık base'den geliyorsa (base safety=0) → yine 0, geçer. Eğer test spesifik olarak `Confidence==0` veya updatedAt=="—" bekliyorsa, base.SafetyScoredTs=0 → updatedAt "—", Confidence 0 → geçer. Regresyon riski yok; yine de çıktı doğrula.

- [ ] **Step 5: gofmt + commit**

```bash
cd apps/api-go && gofmt -w internal/market/detail.go internal/market/detail_test.go
git add apps/api-go/internal/market/detail.go apps/api-go/internal/market/detail_test.go
git commit -m "feat(safety): TokenDetail tokenSafety skoru + risks + top10HolderPct DB'den (Option A deseni)"
```

---

### Task 8: Config + main wiring + README + tam yeşil

**Files:**
- Modify: `apps/api-go/internal/config/config.go` (SAFETY_* config)
- Modify: `apps/api-go/internal/config/config_test.go` (default testi)
- Modify: `apps/api-go/cmd/server/main.go` (safety worker goroutine)
- Modify: `apps/api-go/README.md` (env docs)

**Interfaces:**
- Consumes: `safety.NewWorker`/`safety.NewHeliusProvider` (Task 5,6), `ingest.NewHeliusAuthorities`/`NewHeliusHolders` (Task 3,4), config alanları.

- [ ] **Step 1: Config default testi yaz (RED)**

`apps/api-go/internal/config/config_test.go`'a ekle:
```go
func TestLoadSafetyDefaults(t *testing.T) {
	t.Setenv("SAFETY_ENABLED", "")
	t.Setenv("SAFETY_INTERVAL_SEC", "")
	t.Setenv("SAFETY_LIMIT", "")
	c := Load()
	if !c.SafetyEnabled {
		t.Fatal("SAFETY_ENABLED default true olmalı")
	}
	if c.SafetyIntervalSec != 60 || c.SafetyLimit != 40 || c.SafetyHoldersCap != 5000 {
		t.Fatalf("safety default'ları yanlış: %+v", c)
	}
}
```

- [ ] **Step 2: Testi çalıştır — fail görmeli**

Run: `cd apps/api-go && go test ./internal/config/ -run TestLoadSafety -v`
Expected: FAIL (SafetyEnabled yok).

- [ ] **Step 3: config.go'ya alanları ekle (GREEN)**

Modify `apps/api-go/internal/config/config.go` — Config struct'a ekle:
```go
	SafetyEnabled     bool
	SafetyIntervalSec int
	SafetyLimit       int
	SafetyHoldersCap  int
```
Load()'a ekle:
```go
		SafetyEnabled:     getenvBool("SAFETY_ENABLED", true),
		SafetyIntervalSec: getenvInt("SAFETY_INTERVAL_SEC", 60),
		SafetyLimit:       getenvInt("SAFETY_LIMIT", 40),
		SafetyHoldersCap:  getenvInt("SAFETY_HOLDERS_CAP", 5000),
```

- [ ] **Step 4: Config testi geç**

Run: `cd apps/api-go && go test ./internal/config/ -v`
Expected: PASS.

- [ ] **Step 5: main.go'da safety worker'ı bağla**

Modify `apps/api-go/cmd/server/main.go` — detail service bloğundan sonra (srv kurulmadan önce) ekle:
```go
	// token safety scorer (2a) — arka plan; Helius key + market gerekli
	if cfg.SafetyEnabled && bundle.Tokens != nil && rpcURL != "" {
		provider := safety.NewHeliusProvider(
			ingest.NewHeliusAuthorities(rpcURL), ingest.NewHeliusHolders(rpcURL), cfg.SafetyHoldersCap)
		sw := safety.NewWorker(safety.WorkerDeps{
			Store: bundle.Tokens, Provider: provider,
			Interval: time.Duration(cfg.SafetyIntervalSec) * time.Second, Limit: cfg.SafetyLimit, Logger: logger,
		})
		go sw.Run(ctx)
	} else if cfg.SafetyEnabled {
		logger.Warn("SAFETY: Helius key veya token store yok — safety scorer başlamayacak")
	}
```
Import ekle: `"github.com/furkanatesc/sentinel/apps/api-go/internal/safety"`.

- [ ] **Step 6: README env tablosuna ekle**

Modify `apps/api-go/README.md` — env tablosuna satırlar ekle:
```markdown
| `SAFETY_ENABLED` | Hayır (default true) | Token güvenliği arka plan scorer'ı (2a). Helius key yoksa başlamaz |
| `SAFETY_INTERVAL_SEC` | Hayır (default 60) | Skorlama döngüsü aralığı (saniye) |
| `SAFETY_LIMIT` | Hayır (default 40) | Döngü başına skorlanan token |
| `SAFETY_HOLDERS_CAP` | Hayır (default 5000) | Holder dağılımı için getTokenAccounts sayfalama tavanı |
```

- [ ] **Step 7: Tam build/vet/test/race**

Run:
```bash
cd apps/api-go && gofmt -w ./... && go build ./... && go vet ./... && go test -race ./...
```
Expected: tüm paketler PASS (postgres integration SKIP; frontend'e dokunulmadı).

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/internal/config/ apps/api-go/cmd/server/main.go apps/api-go/README.md
git commit -m "feat(safety): config SAFETY_* + main worker wiring + README env"
```

---

## Deploy & Doğrulama (implementasyon sonrası, kullanıcı onayıyla)

- Merge → Railway otomatik deploy (migration 0005 goose ile). Yeni env opsiyonel (default'lar iş görür); `HELIUS_API_KEY` zaten var.
- **Canlı doğrulama:** ~1-2 skorlama döngüsü sonrası (`SAFETY_INTERVAL_SEC`) `/api/tokens`'da pool'lu token'lar gerçek `safetyScore` (0 dışı, çeşitli); `/api/token/{mint}` `scores.tokenSafety` (breakdown + confidence) + `risks.contract/market` + `metrics.top10HolderPct` gerçek. Diğer 3 skor nötr kalır.
- **Kalibrasyon riski (deploy'da):** Helius `getAccountInfo` (`data.parsed.info.mintAuthority/freezeAuthority`) + `getTokenAccounts` (`token_accounts[].owner/amount`) alan şekilleri — canlı farklıysa parse+fixture birlikte güncellenir (1a/1c deploy-hotfix deseni). getTokenAccounts bazı token'larda jsonParsed/`tokenAmount.amount` iç içe dönebilir → deploy'da doğrula.
- Whole-branch review (opus) → fix wave → merge (kullanıcı onayı, `--no-ff`).
