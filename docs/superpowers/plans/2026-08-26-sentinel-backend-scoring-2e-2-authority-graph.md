# 2e-2 Authority Graph (controls_authority slice) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ≥K token'ın mint/freeze authority'sini kontrol eden cüzdanı `authority_wallet` god-node kümesi olarak yüzeye çıkaran `/api/authority-graph` endpoint'i (bilinen-program/derece filtreli), authority pubkey'ini safety worker'a piggyback yakalayarak (sıfır ek RPC).

**Architecture:** Saf-DB Option A (2e-1 deseni). Authority pubkey'i safety'nin zaten çağırdığı `getAccountInfo`'dan yakalanır (şu an atılıyor), `CreatorHoldingPct` piggyback deseniyle `tokens` kolonlarına persist edilir; `getAuthorityGraph` DB'den unpivot+HAVING agregasyonla küme kurar (canlı RPC yok). Yeni worker/flag YOK.

**Tech Stack:** Go 1.21+ (builtin `min`), chi router, goose migration, database/sql + Postgres; frontend Next.js/TS (getApi hibrit seam).

**Spec:** `docs/superpowers/specs/2026-08-26-sentinel-backend-scoring-2e-2-authority-graph-design.md`

## Global Constraints

- **Saf-DB (Option A):** okuma yolu canlı RPC yapmaz; veri arka planda yakalanıp persist edilir. Graph kurma saf/deterministik Go (ağsız/DB'siz `BuildAuthorityGraph`).
- **Sıfır ek RPC:** authority pubkey'i safety'nin mevcut `getAccountInfo` çağrısından gelir; yeni RPC/worker/key YOK.
- **Fake + Postgres parity:** her yeni store metodu hem `fakeTokenStore` hem `postgresStore`'da aynı semantikle (2a/2b/2e-1 deseni).
- **Dürüst-nötr:** authority bilinmiyorsa (RPC fail) mevcut değer EZİLMEZ (`AuthoritiesKnown=false` guard, `CreatorHoldingKnown` deseni); iptal edilmiş authority = `''`; boş küme = `{nodes:[],edges:[]}`.
- **Bilinen-program allowlist ZORUNLU:** paylaşılan launchpad/program authority'leri dışlanır (2e-1 CEX analoğu) + derece tavanı (`WalletGraphMaxDegree`) bilinmeyen mega-hub'a emniyet.
- **UI dokunulmaz:** frontend yalnız seam (contract/mock/http/live-endpoints) + tip eklemesi; graph render bileşeni değişmez (1c/2d/2e-1 deseni).
- **Migration:** goose formatı (`-- +goose Up` / `-- +goose Down`), `IF NOT EXISTS`/`IF EXISTS` idempotent.
- Her task sonu: `go build ./... && go vet ./... && go test -race ./...` yeşil + commit.

---

### Task 1: Authority pubkey'ini ingest + safety provider'da yakala

**Files:**
- Modify: `apps/api-go/internal/ingest/authorities.go:24-67`
- Modify: `apps/api-go/internal/ingest/authorities_test.go`
- Modify: `apps/api-go/internal/safety/provider.go:11-20` (OnChainData), `:29-31` (Authorities interface), `:48-70` (FetchOnChain)
- Modify: `apps/api-go/internal/safety/provider_test.go` (fake Authorities)

**Interfaces:**
- Produces: `Authorities.MintAuthorities(ctx, mint) (mintAuthority, freezeAuthority *string, err error)`; `OnChainData.MintAuthorityAddr, FreezeAuthorityAddr string` (nil pubkey → `""`).
- Consumes: nothing (stack tabanı).

- [ ] **Step 1: `provider_test.go` fake Authorities'i pubkey döndürecek şekilde güncelle + yeni assertion yaz**

`provider_test.go` içindeki mevcut fake authorities tipini bul (ör. `stubAuth`/`fakeAuth`) ve imzasını değiştir; yeni bir test ekle:

```go
// FetchOnChain, authority pubkey'lerini OnChainData'ya taşımalı + bool'u türetmeli.
func TestFetchOnChain_CapturesAuthorityAddrs(t *testing.T) {
	mintPk, freezePk := "MintAuth111", "FreezeAuth222"
	p := NewHeliusProvider(
		authStub{mint: &mintPk, freeze: &freezePk},
		holdersStub{count: 100, top10: 30},
		5000,
	)
	d, err := p.FetchOnChain(context.Background(), "mintX", "creatorX")
	if err != nil {
		t.Fatal(err)
	}
	if d.MintAuthorityAddr != "MintAuth111" || d.FreezeAuthorityAddr != "FreezeAuth222" {
		t.Fatalf("authority addr taşınmalı, got mint=%q freeze=%q", d.MintAuthorityAddr, d.FreezeAuthorityAddr)
	}
	if !d.MintAuthorityActive || !d.FreezeAuthorityActive || !d.AuthoritiesKnown {
		t.Fatalf("pubkey!=nil → active+known türetilmeli")
	}
	// null authority → boş addr + active=false.
	p2 := NewHeliusProvider(authStub{mint: nil, freeze: nil}, holdersStub{count: 1}, 5000)
	d2, _ := p2.FetchOnChain(context.Background(), "m2", "c2")
	if d2.MintAuthorityAddr != "" || d2.MintAuthorityActive {
		t.Fatalf("null authority → boş addr + active=false, got addr=%q active=%v", d2.MintAuthorityAddr, d2.MintAuthorityActive)
	}
}
```

`authStub`/`holdersStub` yoksa test-yerel tanımla (imzalar §provider.go):
```go
type authStub struct{ mint, freeze *string; err error }
func (a authStub) MintAuthorities(context.Context, string) (*string, *string, error) { return a.mint, a.freeze, a.err }
type holdersStub struct{ count int; top10, creatorPct float64; capped bool; err error }
func (h holdersStub) HolderDistribution(context.Context, string, string, int) (int, float64, float64, bool, error) {
	return h.count, h.top10, h.creatorPct, h.capped, h.err
}
```

- [ ] **Step 2: Testi çalıştır, FAIL gör**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestFetchOnChain_CapturesAuthorityAddrs -v`
Expected: FAIL — derleme hatası (`MintAuthorityAddr` yok / `MintAuthorities` imza uyuşmaz).

- [ ] **Step 3: `OnChainData`'ya iki alan ekle** (`provider.go:11-20`)

```go
type OnChainData struct {
	MintAuthorityActive, FreezeAuthorityActive bool
	MintAuthorityAddr, FreezeAuthorityAddr     string // 2e-2: authority pubkey (piggyback; "" = iptal/bilinmiyor). Scorer'a GİRMEZ.
	AuthoritiesKnown                           bool
	HolderCount                                int
	Top10Pct                                   float64
	HoldersKnown                               bool
	HoldersCapped                              bool
	CreatorHoldingPct                          float64
	CreatorHoldingKnown                        bool
}
```

- [ ] **Step 4: `Authorities` arayüzünü pubkey döndürecek şekilde değiştir** (`provider.go:29-31`)

```go
// Authorities, mint/freeze authority pubkey'ini döndürür (nil = iptal edilmiş/aktif değil). 2e-2:
// pubkey (bool değil) döner → safety active'i türetir + authority-graph pubkey'i persist eder.
type Authorities interface {
	MintAuthorities(ctx context.Context, mint string) (mintAuthority, freezeAuthority *string, err error)
}
```

- [ ] **Step 5: `FetchOnChain`'de pubkey'i yakala + bool türet** (`provider.go:48-61`)

```go
	if mintPk, freezePk, err := p.auth.MintAuthorities(ctx, mint); err == nil {
		d.MintAuthorityActive, d.FreezeAuthorityActive, d.AuthoritiesKnown = mintPk != nil, freezePk != nil, true
		if mintPk != nil {
			d.MintAuthorityAddr = *mintPk
		}
		if freezePk != nil {
			d.FreezeAuthorityAddr = *freezePk
		}
	} else {
		authErr = err
	}
```

- [ ] **Step 6: `ingest/authorities.go`'yu pubkey döndürecek şekilde güncelle** (`:24`, `:66`)

İmza (`:24`):
```go
func (h *HeliusAuthorities) MintAuthorities(ctx context.Context, mint string) (mintAuthority, freezeAuthority *string, err error) {
```
Tüm erken `return false, false, err` → `return nil, nil, err` (satır 30-31, 39-40, 59-60, 62-63 civarı — hepsi 3 dönüşe uyar). Son satır (`:66`):
```go
	info := r.Result.Value.Data.Parsed.Info
	return info.MintAuthority, info.FreezeAuthority, nil
```
(`info.MintAuthority`/`FreezeAuthority` zaten `*string` — satır 48-49.)

- [ ] **Step 7: `authorities_test.go`'yu yeni imzaya uyarla**

Mevcut testlerdeki `mintActive, freezeActive, err := h.MintAuthorities(...)` çağrılarını `mintPk, freezePk, err := ...` yap ve assertion'ları pubkey/nil üzerinden güncelle (ör. dolu authority → `mintPk != nil && *mintPk == "<beklenen>"`; null → `mintPk == nil`).

- [ ] **Step 8: Tüm safety+ingest testleri yeşil (safety skoru regresyonsuz)**

Run: `cd apps/api-go && go build ./... && go test -race ./internal/safety/ ./internal/ingest/ -v`
Expected: PASS — yeni test + mevcut safety skoru testleri (bool türetme korunduğu için).

- [ ] **Step 9: Commit**

```bash
git add apps/api-go/internal/ingest/authorities.go apps/api-go/internal/ingest/authorities_test.go apps/api-go/internal/safety/provider.go apps/api-go/internal/safety/provider_test.go
git commit -m "feat(2e-2): authority pubkey yakala (Authorities *string döndür + OnChainData addr)"
```

---

### Task 2: migration 0013 + authority pubkey persist (SafetyUpdate + store) + GraphEdge.Role

**Files:**
- Create: `apps/api-go/internal/store/migrations/0013_add_token_authorities.sql`
- Modify: `apps/api-go/internal/store/tokens.go:58-68` (SafetyUpdate), `:161-166` (GraphEdge), `:352-368` (postgres UpdateSafety)
- Modify: `apps/api-go/internal/store/fake_ingest.go:35-66` (fakeTok), `:304-320` (fake UpdateSafety)
- Modify: `apps/api-go/internal/store/tokens_fake_test.go` (yeni parity test)

**Interfaces:**
- Consumes: (Task 1) `OnChainData.MintAuthorityAddr/FreezeAuthorityAddr/AuthoritiesKnown`.
- Produces: `SafetyUpdate.MintAuthority, FreezeAuthority string; AuthoritiesKnown bool`; `tokens.mint_authority/freeze_authority` kolonları; `GraphEdge.Role string`.

- [ ] **Step 1: migration 0013 yaz**

`apps/api-go/internal/store/migrations/0013_add_token_authorities.sql`:
```sql
-- +goose Up
-- 2e-2 controls_authority: mint/freeze authority pubkey'i token'a persist edilir (safety piggyback).
-- '' = iptal edilmiş VEYA henüz skorlanmamış (ikisi de küme adayı değil).
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS mint_authority   TEXT NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS freeze_authority TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_tokens_mint_authority   ON tokens (mint_authority)   WHERE mint_authority   <> '';
CREATE INDEX IF NOT EXISTS idx_tokens_freeze_authority ON tokens (freeze_authority) WHERE freeze_authority <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_tokens_freeze_authority;
DROP INDEX IF EXISTS idx_tokens_mint_authority;
ALTER TABLE tokens DROP COLUMN IF EXISTS freeze_authority;
ALTER TABLE tokens DROP COLUMN IF EXISTS mint_authority;
```

- [ ] **Step 2: Fake parity testi yaz** (`tokens_fake_test.go`)

`TestFakeUpdateSafetyCreatorHoldingConditional` desenini taklit et:
```go
func TestFakeUpdateSafetyAuthorityConditional(t *testing.T) {
	f := NewFakeTokenStore()
	ctx := context.Background()
	f.UpsertDiscovered(ctx, DiscoveredToken{Mint: "m", Symbol: "S", PoolAddr: "p", FirstSeenTs: 1})
	// known=true → yazılır.
	f.UpdateSafety(ctx, SafetyUpdate{Mint: "m", AuthoritiesKnown: true, MintAuthority: "MA", FreezeAuthority: "FA"})
	rows, _ := f.AuthorityGraphClusters(ctx, 1, 50) // Task 4'te tanımlanır; bu test Task 4 sonrası da geçer
	_ = rows                                        // burada yalnız yazımı doğrula (aşağıda doğrudan)
	// known=false → EZİLMEZ (mevcut "MA" korunur).
	f.UpdateSafety(ctx, SafetyUpdate{Mint: "m", AuthoritiesKnown: false, MintAuthority: "", FreezeAuthority: ""})
	got := f.byID["m"]
	if got.mintAuthority != "MA" || got.freezeAuthority != "FA" {
		t.Fatalf("known=false authority'yi ezmemeli, got mint=%q freeze=%q", got.mintAuthority, got.freezeAuthority)
	}
}
```
> Not: Bu test `AuthorityGraphClusters` çağrısını içermesin (henüz Task 4). Sadeleştir: `AuthorityGraphClusters` satırını SİL, yalnız `f.byID["m"]` alanlarını doğrula (fake iç erişim aynı pakette mümkün).

- [ ] **Step 3: Testi çalıştır, FAIL gör**

Run: `cd apps/api-go && go test ./internal/store/ -run TestFakeUpdateSafetyAuthorityConditional -v`
Expected: FAIL — `SafetyUpdate.AuthoritiesKnown`/`MintAuthority` yok, `fakeTok.mintAuthority` yok.

- [ ] **Step 4: `SafetyUpdate`'e alanları ekle** (`tokens.go:58-68`)

```go
type SafetyUpdate struct {
	Mint                string
	Score               float64
	Confidence          float64
	Top10Pct            float64
	Breakdown           []ScoreBreakdownItem
	Risks               RiskGroups
	ScoredTs            int64
	CreatorHoldingPct   float64
	CreatorHoldingKnown bool
	// 2e-2: authority pubkey (piggyback; AuthoritiesKnown=false → mevcut değer EZİLMEZ).
	MintAuthority    string
	FreezeAuthority  string
	AuthoritiesKnown bool
}
```

- [ ] **Step 5: `GraphEdge`'e Role ekle** (`tokens.go:161-166`)

```go
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Role   string `json:"role,omitempty"` // 2e-2: controls_authority için "mint"/"freeze"/"both" (diğer edge'lerde boş → omit).
}
```
> (Alanların tam JSON etiketleri mevcut struct'la aynı olmalı — yukarıdaki dört alan `tokens.go:161-166`'daki mevcut tanımı birebir yansıtır; yalnız `Role` eklenir.)

- [ ] **Step 6: `fakeTok`'a iki alan ekle + fake UpdateSafety** (`fake_ingest.go:35-66`, `:304-320`)

`fakeTok` struct'ına (`:35`):
```go
	mintAuthority, freezeAuthority string // 2e-2 authority pubkey (piggyback)
```
`fake UpdateSafety` içine (`:311-314` civarı, CreatorHolding guard'ından sonra):
```go
	if s.AuthoritiesKnown {
		cur.mintAuthority, cur.freezeAuthority = s.MintAuthority, s.FreezeAuthority
	}
```
(`cur`'u map'e geri yazan mevcut satır korunur — `f.byID[s.Mint] = cur`.)

- [ ] **Step 7: postgres `UpdateSafety` query'sini genişlet** (`tokens.go:361-366`)

```go
	const q = `UPDATE tokens SET safety_score=$2, safety_confidence=$3, top10_holder_pct=$4,
		safety_breakdown=$5, safety_risks=$6, safety_scored_ts=$7,
		creator_holding_pct = CASE WHEN $8 THEN $9 ELSE creator_holding_pct END,
		mint_authority       = CASE WHEN $10 THEN $11 ELSE mint_authority   END,
		freeze_authority     = CASE WHEN $10 THEN $12 ELSE freeze_authority END
		WHERE mint=$1`
	_, err = p.db.ExecContext(ctx, q, s.Mint, s.Score, s.Confidence, s.Top10Pct,
		string(bdJSON), string(rkJSON), s.ScoredTs, s.CreatorHoldingKnown, s.CreatorHoldingPct,
		s.AuthoritiesKnown, s.MintAuthority, s.FreezeAuthority)
	return err
```

- [ ] **Step 8: Test yeşil + build**

Run: `cd apps/api-go && go build ./... && go test ./internal/store/ -run TestFakeUpdateSafetyAuthorityConditional -v && go test -race ./internal/store/`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add apps/api-go/internal/store/migrations/0013_add_token_authorities.sql apps/api-go/internal/store/tokens.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/tokens_fake_test.go
git commit -m "feat(2e-2): migration 0013 authority kolonları + SafetyUpdate piggyback persist + GraphEdge.Role"
```

---

### Task 3: Safety worker'ı authority pubkey persist edecek şekilde bağla (piggyback)

**Files:**
- Modify: `apps/api-go/internal/safety/worker.go:97-101` (UpdateSafety çağrısı)
- Modify: `apps/api-go/internal/safety/worker_test.go` (piggyback assertion)

**Interfaces:**
- Consumes: (Task 1) `OnChainData.MintAuthorityAddr/FreezeAuthorityAddr/AuthoritiesKnown`; (Task 2) `SafetyUpdate.MintAuthority/FreezeAuthority/AuthoritiesKnown`.
- Produces: worker her skorlamada authority pubkey'ini persist eder.

- [ ] **Step 1: `worker_test.go`'ya piggyback testi ekle**

Mevcut worker test altyapısını (fake SafetyStore — `UpdateSafety` çağrılarını yakalayan) bul. Yakalanan `SafetyUpdate`'te authority alanlarını doğrula:
```go
// Worker, provider'ın döndürdüğü authority pubkey'ini UpdateSafety'ye taşımalı (piggyback).
func TestWorker_PersistsAuthorityAddrs(t *testing.T) {
	store := &captureStore{ /* SafetyScoreTargets: 1 hedef döndürür; UpdateSafety'yi kaydeder */ }
	prov := stubProvider{data: store.OnChainData{ // safety.OnChainData
		MintAuthorityActive: true, AuthoritiesKnown: true,
		MintAuthorityAddr: "MA", FreezeAuthorityAddr: "FA",
	}}
	w := NewWorker(WorkerDeps{Store: store, Provider: prov, Now: func() int64 { return 1 }})
	w.scoreOnce(context.Background())
	last := store.lastUpdate // yakalanan SafetyUpdate
	if !last.AuthoritiesKnown || last.MintAuthority != "MA" || last.FreezeAuthority != "FA" {
		t.Fatalf("authority piggyback taşınmalı, got known=%v mint=%q freeze=%q", last.AuthoritiesKnown, last.MintAuthority, last.FreezeAuthority)
	}
}
```
> Mevcut worker_test.go'da benzer bir capture-store/stub-provider varsa ONU reuse et (yeni tip tanımlama); yoksa test-yerel minimal `captureStore` (SafetyStore arayüzü: `SafetyScoreTargets`+`UpdateSafety`) ve `stubProvider` (DataProvider) tanımla.

- [ ] **Step 2: Testi çalıştır, FAIL gör**

Run: `cd apps/api-go && go test ./internal/safety/ -run TestWorker_PersistsAuthorityAddrs -v`
Expected: FAIL — `UpdateSafety`'ye authority alanları geçilmiyor (boş).

- [ ] **Step 3: Worker'da UpdateSafety çağrısına authority alanlarını ekle** (`worker.go:97-101`)

```go
		if err := w.d.Store.UpdateSafety(ctx, store.SafetyUpdate{
			Mint: tg.Mint, Score: res.Score, Confidence: res.Confidence, Top10Pct: res.Top10Pct,
			Breakdown: res.Breakdown, Risks: res.Risks, ScoredTs: now,
			CreatorHoldingPct: data.CreatorHoldingPct, CreatorHoldingKnown: data.CreatorHoldingKnown,
			MintAuthority: data.MintAuthorityAddr, FreezeAuthority: data.FreezeAuthorityAddr, AuthoritiesKnown: data.AuthoritiesKnown,
		}); err != nil {
```

- [ ] **Step 4: Test + tüm safety yeşil**

Run: `cd apps/api-go && go test -race ./internal/safety/ -v`
Expected: PASS (yeni test + mevcut worker/cycle testleri).

- [ ] **Step 5: Commit**

```bash
git add apps/api-go/internal/safety/worker.go apps/api-go/internal/safety/worker_test.go
git commit -m "feat(2e-2): safety worker authority pubkey piggyback persist"
```

---

### Task 4: `AuthorityGraphClusters` store agregasyonu (fake + postgres parity)

**Files:**
- Modify: `apps/api-go/internal/store/tokens.go:131-136` (AuthorityRow tipi ekle), `:205-214` (TokenStore interface), yeni postgres metodu
- Modify: `apps/api-go/internal/store/fake_ingest.go` (fake metod)
- Create: `apps/api-go/internal/store/authority_graph_test.go`

**Interfaces:**
- Produces: `AuthorityRow{Authority, Mint, Symbol, Role string; SafetyScore float64; FirstSeenTs int64}`; `TokenStore.AuthorityGraphClusters(ctx, minCluster, maxDegree int) ([]AuthorityRow, error)`.
- Consumes: (Task 2) `tokens.mint_authority/freeze_authority` kolonları.

- [ ] **Step 1: `AuthorityRow` tipini ekle** (`tokens.go`, ClusterRow'un yanına `:136` sonrası)

```go
// AuthorityRow, bir controls_authority kenarıdır: authority→token + rol + skorlar (2e-2).
type AuthorityRow struct {
	Authority, Mint, Symbol, Role string
	SafetyScore                   float64
	FirstSeenTs                   int64
}
```

- [ ] **Step 2: Parity testi yaz** (`authority_graph_test.go`)

```go
package store

import (
	"context"
	"testing"
)

// authority F1, ≥2 token'ın mint authority'si → küme; F2 tek token → dışarıda.
func TestAuthorityGraphClusters_DegreeThreshold(t *testing.T) {
	f := NewFakeTokenStore().(*fakeTokenStore)
	ctx := context.Background()
	up := func(mint, sym, mintA, freezeA string) {
		f.UpsertDiscovered(ctx, DiscoveredToken{Mint: mint, Symbol: sym, PoolAddr: "p", FirstSeenTs: 1})
		f.UpdateSafety(ctx, SafetyUpdate{Mint: mint, AuthoritiesKnown: true, MintAuthority: mintA, FreezeAuthority: freezeA})
	}
	up("mA", "A", "F1", "")   // F1 mint
	up("mB", "B", "F1", "F1") // F1 mint+freeze → mB'de İKİ ham satır (mint + freeze); "both" birleştirme Task 5'te
	up("mC", "C", "F2", "")   // F2 tek token → küme değil
	rows, err := f.AuthorityGraphClusters(ctx, 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	byAuth := map[string]int{}
	mbMint, mbFreeze := false, false
	for _, r := range rows {
		byAuth[r.Authority]++
		if r.Authority == "F1" && r.Mint == "mB" && r.Role == "mint" {
			mbMint = true
		}
		if r.Authority == "F1" && r.Mint == "mB" && r.Role == "freeze" {
			mbFreeze = true
		}
	}
	if byAuth["F1"] == 0 || byAuth["F2"] != 0 {
		t.Fatalf("F1 küme olmalı, F2 dışarıda, got %+v", byAuth)
	}
	// Store HAM satır döndürür: mB için F1 hem mint hem freeze satırı (birleştirme YOK — Task 5 mergeRole).
	if !mbMint || !mbFreeze {
		t.Fatalf("mB'de F1 için ayrı mint+freeze ham satırları beklenir, got mint=%v freeze=%v", mbMint, mbFreeze)
	}
}

// maxDegree tavanı: çok-yüksek-dereceli authority (program-benzeri) elenmeli.
func TestAuthorityGraphClusters_MaxDegreeCeiling(t *testing.T) {
	f := NewFakeTokenStore().(*fakeTokenStore)
	ctx := context.Background()
	for i, m := range []string{"m1", "m2", "m3"} {
		f.UpsertDiscovered(ctx, DiscoveredToken{Mint: m, Symbol: m, PoolAddr: "p", FirstSeenTs: int64(i + 1)})
		f.UpdateSafety(ctx, SafetyUpdate{Mint: m, AuthoritiesKnown: true, MintAuthority: "PROG"})
	}
	rows, _ := f.AuthorityGraphClusters(ctx, 2, 2) // degree 3 > maxDegree 2 → elenir
	if len(rows) != 0 {
		t.Fatalf("maxDegree tavanı aşan authority elenmeli, got %d row", len(rows))
	}
}
```

- [ ] **Step 3: Testi çalıştır, FAIL gör**

Run: `cd apps/api-go && go test ./internal/store/ -run TestAuthorityGraphClusters -v`
Expected: FAIL — `AuthorityGraphClusters` yok.

- [ ] **Step 4: `TokenStore` arayüzüne ekle** (`tokens.go:213` sonrası)

```go
	// 2e-2: controls_authority agregası — mint+freeze authority unpivot; degree [minCluster,maxDegree]
	// aralığındaki authority'lerin tüm (authority,token,role) kenarlarını döndürür.
	AuthorityGraphClusters(ctx context.Context, minCluster, maxDegree int) ([]AuthorityRow, error)
```

- [ ] **Step 5: postgres implementasyonu** (`tokens.go`, WalletGraphClusters'ın yanına)

```go
func (p *postgresStore) AuthorityGraphClusters(ctx context.Context, minCluster, maxDegree int) ([]AuthorityRow, error) {
	const q = `
	WITH auth AS (
		SELECT mint_authority   AS authority, mint, symbol, safety_score, first_seen_ts, 'mint'   AS role
			FROM tokens WHERE mint_authority   <> ''
		UNION ALL
		SELECT freeze_authority AS authority, mint, symbol, safety_score, first_seen_ts, 'freeze' AS role
			FROM tokens WHERE freeze_authority <> ''
	), qualifying AS (
		SELECT authority FROM auth
		GROUP BY authority
		HAVING COUNT(DISTINCT mint) >= $1 AND COUNT(DISTINCT mint) <= $2
	)
	SELECT a.authority, a.mint, a.symbol, a.role, a.safety_score, a.first_seen_ts
	FROM auth a JOIN qualifying q ON q.authority = a.authority
	ORDER BY a.authority, a.first_seen_ts DESC, a.mint`
	rows, err := p.db.QueryContext(ctx, q, minCluster, maxDegree)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuthorityRow{}
	for rows.Next() {
		var r AuthorityRow
		if err := rows.Scan(&r.Authority, &r.Mint, &r.Symbol, &r.Role, &r.SafetyScore, &r.FirstSeenTs); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
```
> Rol birleştirme (`mint`+`freeze` aynı authority-token → `both`) SQL'de değil, `BuildAuthorityGraph`'ta yapılır (Task 5) — postgres ayrı `mint`/`freeze` satırları döndürür, fake de öyle (parity). `both` mantığı tek yerde (saf Go).

- [ ] **Step 6: fake implementasyonu** (`fake_ingest.go`, WalletGraphClusters deseni)

```go
func (f *fakeTokenStore) AuthorityGraphClusters(_ context.Context, minCluster, maxDegree int) ([]AuthorityRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// (authority, role) → token satırları topla + authority başına distinct mint say.
	type ar struct{ authority, mint, symbol, role string; safety float64; firstSeen int64 }
	var all []ar
	mints := map[string]map[string]bool{}
	add := func(authority, role string, t fakeTok) {
		if authority == "" {
			return
		}
		all = append(all, ar{authority, t.row.Mint, t.row.Symbol, role, t.safetyScore, t.firstSeen})
		if mints[authority] == nil {
			mints[authority] = map[string]bool{}
		}
		mints[authority][t.row.Mint] = true
	}
	for _, id := range f.order {
		t := f.byID[id]
		add(t.mintAuthority, "mint", t)
		add(t.freezeAuthority, "freeze", t)
	}
	out := []AuthorityRow{}
	for _, a := range all {
		deg := len(mints[a.authority])
		if deg >= minCluster && deg <= maxDegree {
			out = append(out, AuthorityRow{Authority: a.authority, Mint: a.mint, Symbol: a.symbol, Role: a.role, SafetyScore: a.safety, FirstSeenTs: a.firstSeen})
		}
	}
	return out, nil
}
```

- [ ] **Step 7: Test yeşil + build**

Run: `cd apps/api-go && go build ./... && go test -race ./internal/store/ -run TestAuthorityGraphClusters -v`
Expected: PASS (her iki test).

- [ ] **Step 8: Commit**

```bash
git add apps/api-go/internal/store/tokens.go apps/api-go/internal/store/fake_ingest.go apps/api-go/internal/store/authority_graph_test.go
git commit -m "feat(2e-2): AuthorityGraphClusters store agrega (unpivot+HAVING, fake+pg parity)"
```

---

### Task 5: `BuildAuthorityGraph` + bilinen-program allowlist

**Files:**
- Create: `apps/api-go/internal/walletgraph/authority_exclude.go`
- Create: `apps/api-go/internal/walletgraph/authority_graph.go`
- Create: `apps/api-go/internal/walletgraph/authority_graph_test.go`

**Interfaces:**
- Consumes: (Task 4) `store.AuthorityRow`; (Task 2) `store.GraphEdge.Role`; mevcut `store.WalletGraphResult/GraphNode/GraphEdge`, `store.ScoreToLevel`, `shortAddr`/`rfc3339`/`mapNodes`/`mapEdges` (graph.go).
- Produces: `BuildAuthorityGraph(rows []store.AuthorityRow) store.WalletGraphResult`; `IsProgramAuthority(addr string) bool`.

- [ ] **Step 1: Bilinen-program adreslerini web'de doğrula (CEX deseni)**

Solscan public-label ile şu adayları doğrula/ekle (pump.fun paylaşılan authority = en kritik sahte-hub kaynağı):
- SPL Token Program: `TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA`
- Token-2022 Program: `TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb`
- System Program: `11111111111111111111111111111111`
- **pump.fun mint-authority** — Solscan'de pump.fun bir token'ının `mintAuthority`/`freezeAuthority`'sine bakıp paylaşılan PDA/authority adresini tespit et (WebSearch: `pump.fun mint authority address solscan`). Bulunanı ekle; bulunamazsa yorum düş ("pump.fun authority doğrulanamadı; derece tavanı emniyet") ve SADECE doğrulananları koy.
> Prensip (2e-1 CEX asimetrisi): **eksik program adresi** = sahte hub (kötü ama derece tavanı yakalar); **yanlış program adresi** ≈ zararsız. Yalnız çok-kaynaklı/etiketli adres ekle.

- [ ] **Step 2: `authority_exclude.go` yaz**

```go
package walletgraph

// knownProgramAuthority, mint/freeze authority olarak görünse de "kontrol kümesi" OLMAYAN
// program/sistem adresleridir (paylaşılan launchpad/PDA authority → sahte god-node → dışlanır;
// 2e-1 CEX allowlist analoğu). Adresler web-doğrulandı (Solscan public label); derece tavanı
// (WalletGraphMaxDegree) bilinmeyen mega-hub'lara ek emniyet.
var knownProgramAuthority = map[string]string{
	"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA": "SPL Token Program",
	"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb": "Token-2022 Program",
	"11111111111111111111111111111111":            "System Program",
	// + Step 1'de doğrulanan pump.fun authority adres(ler)i buraya.
}

func IsProgramAuthority(addr string) bool { _, ok := knownProgramAuthority[addr]; return ok }
```

- [ ] **Step 3: `BuildAuthorityGraph` testi yaz** (`authority_graph_test.go`)

```go
package walletgraph

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestBuildAuthorityGraph_HubRolesAndExclusion(t *testing.T) {
	rows := []store.AuthorityRow{
		{Authority: "F1", Mint: "mA", Symbol: "A", Role: "mint", SafetyScore: 40, FirstSeenTs: 1},
		{Authority: "F1", Mint: "mB", Symbol: "B", Role: "mint", SafetyScore: 40, FirstSeenTs: 2},
		{Authority: "F1", Mint: "mB", Symbol: "B", Role: "freeze", SafetyScore: 40, FirstSeenTs: 2}, // mB: mint+freeze → both
		{Authority: "11111111111111111111111111111111", Mint: "mC", Symbol: "C", Role: "mint", FirstSeenTs: 3}, // program → dışlanır
	}
	g := BuildAuthorityGraph(rows)
	// authority_wallet hub + token node'ları (program hariç).
	var hub, both bool
	for _, n := range g.Nodes {
		if n.ID == "auth:F1" && n.Type == "authority_wallet" {
			hub = true
		}
		if n.Address == "11111111111111111111111111111111" {
			t.Fatalf("program authority node üretilmemeli")
		}
	}
	for _, e := range g.Edges {
		if e.Type == "controls_authority" && e.Source == "auth:F1" && e.Target == "tok:mB" && e.Role == "both" {
			both = true
		}
	}
	if !hub {
		t.Fatalf("authority_wallet hub node beklenir")
	}
	if !both {
		t.Fatalf("mB edge rolü 'both' beklenir (mint+freeze)")
	}
}

func TestBuildAuthorityGraph_EmptyIsEmpty(t *testing.T) {
	g := BuildAuthorityGraph(nil)
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Fatalf("boş girdi → boş graph")
	}
}
```

- [ ] **Step 4: Testi çalıştır, FAIL gör**

Run: `cd apps/api-go && go test ./internal/walletgraph/ -run TestBuildAuthorityGraph -v`
Expected: FAIL — `BuildAuthorityGraph` yok.

- [ ] **Step 5: `authority_graph.go` yaz** (graph.go helper'larını reuse)

```go
package walletgraph

import "github.com/furkanatesc/sentinel/apps/api-go/internal/store"

// BuildAuthorityGraph, controls_authority kümelerini kurar (saf; ağsız/DB'siz). Bilinen-program
// authority'lerini dışlar. authority_wallet hub + controls_authority edge (rol: mint/freeze/both).
func BuildAuthorityGraph(rows []store.AuthorityRow) store.WalletGraphResult {
	nodes := map[string]store.GraphNode{}
	edges := map[string]store.GraphEdge{}
	degree := map[string]map[string]struct{}{}         // authority → distinct mint (risk)
	roleByEdge := map[string]string{}                  // "auth|mint" → birleşik rol
	for _, r := range rows {
		if IsProgramAuthority(r.Authority) {
			continue
		}
		if degree[r.Authority] == nil {
			degree[r.Authority] = map[string]struct{}{}
		}
		degree[r.Authority][r.Mint] = struct{}{}
		key := r.Authority + "|" + r.Mint
		roleByEdge[key] = mergeRole(roleByEdge[key], r.Role)
	}
	for _, r := range rows {
		if IsProgramAuthority(r.Authority) {
			continue
		}
		ts := rfc3339(r.FirstSeenTs)
		authID, tokID := "auth:"+r.Authority, "tok:"+r.Mint
		deg := len(degree[r.Authority])
		nodes[authID] = store.GraphNode{ID: authID, Type: "authority_wallet", Label: shortAddr(r.Authority),
			Address: r.Authority, RiskLevel: store.ScoreToLevel(float64(100 - min(deg*20, 100))), FirstSeen: ts, LastSeen: ts}
		nodes[tokID] = store.GraphNode{ID: tokID, Type: "token", Label: r.Symbol,
			Address: r.Mint, RiskLevel: store.ScoreToLevel(r.SafetyScore), FirstSeen: ts, LastSeen: ts}
		eID := "e:ctrl:" + r.Authority + ":" + r.Mint
		edges[eID] = store.GraphEdge{ID: eID, Source: authID, Target: tokID, Type: "controls_authority",
			Role: roleByEdge[r.Authority+"|"+r.Mint]}
	}
	return store.WalletGraphResult{Nodes: mapNodes(nodes), Edges: mapEdges(edges)}
}

// mergeRole, aynı (authority,token) için mint+freeze rollerini "both"a birleştirir.
func mergeRole(existing, incoming string) string {
	if existing == "" || existing == incoming {
		return incoming
	}
	return "both"
}
```

- [ ] **Step 6: Test yeşil + build**

Run: `cd apps/api-go && go build ./... && go test -race ./internal/walletgraph/ -v`
Expected: PASS (yeni testler + mevcut 2e-1 BuildGraph/CEX/resolver/worker testleri).

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/walletgraph/authority_exclude.go apps/api-go/internal/walletgraph/authority_graph.go apps/api-go/internal/walletgraph/authority_graph_test.go
git commit -m "feat(2e-2): BuildAuthorityGraph (authority_wallet hub, controls_authority rol) + program allowlist"
```

---

### Task 6: `/api/authority-graph` endpoint

**Files:**
- Create: `apps/api-go/internal/api/authority_graph.go`
- Modify: `apps/api-go/internal/api/router.go:47-54`
- Create: `apps/api-go/internal/api/authority_graph_test.go`

**Interfaces:**
- Consumes: (Task 4) `store.TokenStore.AuthorityGraphClusters`; (Task 5) `walletgraph.BuildAuthorityGraph`.
- Produces: `GET /api/authority-graph` → `WalletGraphResult` JSON.

- [ ] **Step 1: Endpoint testi yaz** (`authority_graph_test.go`)

`wallet_graph_test.go` desenini taklit et:
```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestAuthorityGraphEndpoint_EmptyIsArray(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Tokens: ts.(store.TokenStore), WalletGraphMinCluster: 2, WalletGraphMaxDegree: 50})
	req := httptest.NewRequest(http.MethodGet, "/api/authority-graph", nil)
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
		t.Fatalf("boş graph JSON'da [] olmalı (null değil): %+v", g)
	}
}
```

- [ ] **Step 2: Testi çalıştır, FAIL gör**

Run: `cd apps/api-go && go test ./internal/api/ -run TestAuthorityGraphEndpoint -v`
Expected: FAIL — 404 (route yok).

- [ ] **Step 3: Handler yaz** (`authority_graph.go`)

```go
package api

import (
	"net/http"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/furkanatesc/sentinel/apps/api-go/internal/walletgraph"
)

func authorityGraphHandler(ts store.TokenStore, minCluster, maxDegree int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := ts.AuthorityGraphClusters(r.Context(), minCluster, maxDegree)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "authority graph unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, walletgraph.BuildAuthorityGraph(rows))
	}
}
```

- [ ] **Step 4: Route'u kaydet** (`router.go:54` civarı, wallet-graph'ın yanına)

```go
		r.Get("/api/wallet-graph", walletGraphHandler(d.Tokens, mc, md))
		r.Get("/api/authority-graph", authorityGraphHandler(d.Tokens, mc, md))
```

- [ ] **Step 5: Test yeşil + build**

Run: `cd apps/api-go && go build ./... && go test -race ./internal/api/ -run TestAuthorityGraphEndpoint -v`
Expected: PASS.

- [ ] **Step 6: Tüm backend suite yeşil**

Run: `cd apps/api-go && go build ./... && go vet ./... && go test -race ./...`
Expected: tüm paketler PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api-go/internal/api/authority_graph.go apps/api-go/internal/api/router.go apps/api-go/internal/api/authority_graph_test.go
git commit -m "feat(2e-2): /api/authority-graph handler + route (wallet-graph deseni)"
```

---

### Task 7: Frontend `getAuthorityGraph` seam (UI dokunulmaz)

**Files:**
- Modify: `apps/web/lib/api/contract.ts:10` (SentinelApi)
- Modify: `apps/web/lib/api/types.ts:125-136` (GraphNodeType += authority_wallet, GraphEdge += role?)
- Modify: `apps/web/lib/api/mock.ts:550` (mock getAuthorityGraph)
- Modify: `apps/web/lib/api/http.ts:28` (http getAuthorityGraph)
- Modify: `apps/web/lib/api/live-endpoints.ts` (LIVE_ENDPOINTS += getAuthorityGraph)

**Interfaces:**
- Consumes: backend `GET /api/authority-graph` → `WalletGraph` JSON şekli (Task 6).
- Produces: `SentinelApi.getAuthorityGraph(): Promise<WalletGraph>` (hibrit: live → httpApi, değilse mock).

- [ ] **Step 1: `types.ts` — authority_wallet node tipi + GraphEdge role** (`:125-136`)

```ts
export type GraphNodeType =
  | "creator_wallet" | "funding_wallet" | "authority_wallet" | "token" | "liquidity_pool"
  | "trader_wallet" | "smart_wallet" | "suspicious_wallet" | "exchange_wallet";
export type GraphEdgeType =
  | "funded" | "created" | "bought" | "sold" | "transferred"
  | "provided_liquidity" | "removed_liquidity" | "shares_funder" | "controls_authority";

export interface GraphNode {
  id: string; type: GraphNodeType; label: string;
  address?: string; riskLevel: RiskLevel; balanceSol?: number; firstSeen: string; lastSeen: string;
}
export interface GraphEdge { id: string; source: string; target: string; type: GraphEdgeType; role?: "mint" | "freeze" | "both"; }
```
> Graph render bileşeninin node-tipini stil/renk map'iyle çizip çizmediğini kontrol et (`rg "authority_wallet|funding_wallet" apps/web/components apps/web/app`). Map varsa ve fallback yoksa, `funding_wallet` ile aynı stili `authority_wallet`'a EKLE (görünmez node bırakma). Yoksa dokunma.

- [ ] **Step 2: `contract.ts` — SentinelApi'ye ekle** (`:10`, getWalletGraph'ın yanına)

```ts
  getWalletGraph(): Promise<WalletGraph>;
  getAuthorityGraph(): Promise<WalletGraph>;
```

- [ ] **Step 3: `mock.ts` — mock impl** (`:550`, getWalletGraph'ın yanına)

```ts
  getWalletGraph: () => delay(walletGraph),
  getAuthorityGraph: () => delay({ nodes: [], edges: [] }),
```
> Mock boş graph döndürür (authority-graph için mock fixture YAGNI; gerçek veri backend'den gelir). `WalletGraph` şekli.

- [ ] **Step 4: `http.ts` — http impl** (`:28`, getWalletGraph'ın yanına)

```ts
  getWalletGraph: () => getJson<WalletGraph>("/api/wallet-graph"),
  getAuthorityGraph: () => getJson<WalletGraph>("/api/authority-graph"),
```

- [ ] **Step 5: `live-endpoints.ts` — LIVE_ENDPOINTS'e ekle**

```ts
  "getWalletGraph",
  "getAuthorityGraph",
```

- [ ] **Step 6: Frontend testleri + tip kontrolü yeşil**

Run: `cd apps/web && npm run typecheck 2>/dev/null || npx tsc --noEmit` sonra `npm test`
Expected: PASS (mevcut suite + tip uyumu; UI değişmediği için snapshot/render testleri yeşil kalır).

- [ ] **Step 7: Commit**

```bash
git add apps/web/lib/api/contract.ts apps/web/lib/api/types.ts apps/web/lib/api/mock.ts apps/web/lib/api/http.ts apps/web/lib/api/live-endpoints.ts
git commit -m "feat(2e-2): frontend getAuthorityGraph seam + authority_wallet tip (UI dokunulmaz)"
```

---

## Self-Review

**1. Spec coverage:**
- §2.1 controls_authority slice → tüm task'lar. §2.2 safety piggyback → Task 1+3. §2.3 birleşik edge+rol → Task 2 (GraphEdge.Role) + Task 5 (mergeRole/both). §2.4 program allowlist + derece tavanı → Task 5 (IsProgramAuthority) + Task 4 (maxDegree HAVING). §2.5 küme odağı (minCluster) → Task 4. §2.6 DEFER → kapsam dışı, dokunulmadı. §2.7 dürüstlük (boş=honest, Known guard) → Task 2 (AuthoritiesKnown guard) + Task 6 (empty-is-array). §3 pubkey capture → Task 1. §4 migration 0013 → Task 2. §5 sorgu+build → Task 4+5. §6.4 endpoint → Task 6. §6.6 frontend → Task 7. **Gap yok.**
- **Config (§6.5):** yeni env yok — mevcut `WalletGraphMinCluster/MaxDegree` reuse (router.go zaten mc/md hesaplıyor); ek task gerekmez. ✓

**2. Placeholder scan:** Her adımda gerçek kod var. Task 5 Step 1 web-doğrulama gerçek bir aksiyon (CEX deseni, 2e-1'de yapıldı) — adres listesi doğrulama sonrası netleşir, ama kod iskeleti + doğrulanmış 3 program adresi mevcut; pump.fun adresi "doğrula, bulunursa ekle, bulunmazsa derece tavanı emniyet" olarak açık (sessiz düşürme yok). Kabul edilir.

**3. Type consistency:** `MintAuthorities` → `(*string,*string,error)` (Task 1) tüm çağrı sitelerinde tutarlı (provider.go, authorities.go, testler). `SafetyUpdate.MintAuthority/FreezeAuthority/AuthoritiesKnown` (Task 2) → worker (Task 3) → yok, worker aynı alan adlarını kullanır. `OnChainData.MintAuthorityAddr/FreezeAuthorityAddr` (Task 1) → worker (Task 3) tutarlı. `AuthorityRow` (Task 4) → `BuildAuthorityGraph` (Task 5) → handler (Task 6) tutarlı. `GraphEdge.Role` (Task 2) → BuildAuthorityGraph (Task 5) → frontend `role?` (Task 7) tutarlı. `authorityGraphHandler` (Task 6) ↔ `AuthorityGraphClusters`/`BuildAuthorityGraph` tutarlı. **Tutarlı.**

---

## Deploy notu (plan sonrası, ayrı onayla)

- Migration 0013 prod'da goose ile uygulanır (Railway deploy). Authority kolonları mevcut token'larda safety worker onları yeniden skorlayana kadar `''` → graph zamanla dolar (dürüst).
- **Yeni env YOK.** `WALLET_GRAPH_ENABLED` bu slice'ı gate ETMEZ (yakalama safety'ye piggyback; safety canlı) → `/api/authority-graph` prod'da **gerçek veri** döndürebilir (wallet-graph'ın aksine).
- 2d/2e-1 deseni: merge → Railway deploy → canlı doğrulama (`/api/authority-graph` 200 + şekil).
