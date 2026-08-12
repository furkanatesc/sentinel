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

// TestCreatorsLimit, fake Creators'ın limit sınırlamasının postgres LIMIT $1
// semantiğiyle eşleştiğini kilitler: limit=0 → boş dilim (ALL değil), limit=1 →
// tam olarak en üstteki (en çok token'lı) creator.
func TestCreatorsLimit(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m2", Mint: "m2", Symbol: "S2"}, 90, "AAA")
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m3", Mint: "m3", Symbol: "S3"}, 80, "BBB")

	cs := ts.(CreatorStore)

	rows, err := cs.Creators(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("Creators(ctx, 0) = %+v, want boş dilim (postgres LIMIT $1=0 ile eşleşmeli)", rows)
	}

	rows, err = cs.Creators(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Address != "AAA" || rows[0].TotalTokens != 2 {
		t.Fatalf("Creators(ctx, 1) = %+v, want tek satır AAA/2", rows)
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

func TestCreatorDetailCarriesOutcome(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	_ = ts.UpsertToken(ctx, TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 1000, "AAA")
	_ = ts.UpdateMarket(ctx, MarketUpdate{Mint: "m1", MarketCapUSD: 10000, Liquidity: 500})
	_ = ts.UpdateOutcome(ctx, OutcomeUpdate{Mint: "m1", Outcome: "rug", LiquidityStatus: "removed", MaxDrawdownPct: 88, ScoredTs: 2000})

	p, ok, err := ts.(CreatorStore).CreatorDetail(ctx, "AAA")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	h := p.History[0]
	if h.Outcome != "rug" || h.LiquidityStatus != "removed" || h.MaxDrawdownPct != 88 {
		t.Fatalf("history outcome alanları = %+v (rug/removed/88 bekleniyor)", h)
	}
	if h.PeakMarketCap != 10000 { // UpdateMarket peak'i seed etti (GREATEST 0→10000)
		t.Fatalf("peakMarketCap = %v, want 10000", h.PeakMarketCap)
	}
	if h.CreatorSellPct != 0 { // nötr → 2c (trade-flow)
		t.Fatalf("creatorSellPct = %v, want 0 (nötr)", h.CreatorSellPct)
	}
}

func TestCreatorDetailNotFound(t *testing.T) {
	ts := NewFakeTokenStore()
	_, ok, err := ts.(CreatorStore).CreatorDetail(context.Background(), "NOPE")
	if err != nil || ok {
		t.Fatalf("bulunmayan creator: ok=%v err=%v", ok, err)
	}
}

// contains, dilimde bir string arar (riskFlags testlerinde kullanılır).
func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// TestCreatorDetailReadsRealReputation, 2b-2b: UpsertReputation ile persist edilmiş
// itibar/metrik CreatorDetail'de nötr placeholder yerine gerçek olarak dönmeli.
func TestCreatorDetailReadsRealReputation(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	f := ts.(*fakeTokenStore)
	seedToken(t, f, "m1", "A", "rug", 69000, 100, 3700)
	seedToken(t, f, "m2", "A", "graduated", 80000, 100, 3700)
	if err := f.UpsertReputation(ctx, CreatorReputation{
		Address: "A", Score: 55, Confidence: 0.4, RiskLevel: "medium",
		TotalTokens: 2, RuggedTokens: 1, GraduatedTokens: 1, SuccessRatePct: 50, AvgPeakMarketCap: 74500,
	}); err != nil {
		t.Fatal(err)
	}
	prof, ok, err := f.CreatorDetail(ctx, "A")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if prof.Reputation.Value != 55 || prof.RiskLevel != "medium" {
		t.Fatalf("reputation okunmadı: %+v", prof.Reputation)
	}
	if prof.Metrics.RuggedTokens != 1 || prof.Metrics.SuccessRatePct != 50 {
		t.Fatalf("metrics yanlış: %+v", prof.Metrics)
	}
}

// TestCreatorDetailRiskFlagsFromOutcome, per-token history item'ların outcome/liquidityStatus/
// maxDrawdown'dan riskFlags türettiğini doğrular (deriveRiskFlags).
func TestCreatorDetailRiskFlagsFromOutcome(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	f := ts.(*fakeTokenStore)
	seedTokenFull(t, f, "m1", "A", "rug", "removed", 95) // outcome=rug, liq=removed, drawdown=95
	prof, ok, err := f.CreatorDetail(ctx, "A")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	flags := prof.History[0].RiskFlags
	if !contains(flags, "Rug çekildi") || !contains(flags, "Likidite çekildi") {
		t.Fatalf("riskFlags eksik: %v", flags)
	}
	if !contains(flags, "Yüksek düşüş (%95)") {
		t.Fatalf("yüksek düşüş bayrağı eksik: %v", flags)
	}
}

// TestCreatorDetailHighDrawdownThresholdConfigurable, 2b-2b final-review fix: deriveRiskFlags'in
// "Yüksek düşüş" eşiğinin artık hardcoded const DEĞİL, cfg.ReputationHighDrawdown'dan (bkz.
// WithHighDrawdownThreshold) geldiğini kilitler. maxDrawdown=85: varsayılan eşik (80, opsiyonsuz
// NewFakeTokenStore) bayrağı tetikler; eşik 90'a çekilince (REPUTATION_HIGH_DRAWDOWN=90 paritesi)
// AYNI 85 artık tetiklemez; eşik 90 iken maxDrawdown=95 yine tetikler (eşik gerçekten kaydı,
// bayrak sadece kapanmadı kanıtı).
func TestCreatorDetailHighDrawdownThresholdConfigurable(t *testing.T) {
	ctx := context.Background()

	// Varsayılan eşik (80) — opsiyonsuz kurucu, mevcut davranışla parite.
	tsDefault := NewFakeTokenStore()
	fDefault := tsDefault.(*fakeTokenStore)
	seedTokenFull(t, fDefault, "m1", "A", "active", "unlocked", 85)
	profDefault, ok, err := fDefault.CreatorDetail(ctx, "A")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !contains(profDefault.History[0].RiskFlags, "Yüksek düşüş (%85)") {
		t.Fatalf("varsayılan eşik (80) ile drawdown=85 bayrak vermeli: %v", profDefault.History[0].RiskFlags)
	}

	// Eşik 90'a çekildi (REPUTATION_HIGH_DRAWDOWN=90 paritesi) — AYNI drawdown=85 artık tetiklememeli.
	tsHigh := NewFakeTokenStore(WithHighDrawdownThreshold(90))
	fHigh := tsHigh.(*fakeTokenStore)
	seedTokenFull(t, fHigh, "m1", "A", "active", "unlocked", 85)
	profHigh, ok, err := fHigh.CreatorDetail(ctx, "A")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if contains(profHigh.History[0].RiskFlags, "Yüksek düşüş (%85)") {
		t.Fatalf("eşik=90 iken drawdown=85 bayrak VERMEMELİ (knob inert olmamalı): %v", profHigh.History[0].RiskFlags)
	}

	// Eşik 90 iken drawdown=95 hâlâ tetiklemeli (eşik gerçekten uygulanıyor, bayrak sadece kapanmadı).
	tsHigh2 := NewFakeTokenStore(WithHighDrawdownThreshold(90))
	fHigh2 := tsHigh2.(*fakeTokenStore)
	seedTokenFull(t, fHigh2, "m1", "A", "active", "unlocked", 95)
	profHigh2, ok, err := fHigh2.CreatorDetail(ctx, "A")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !contains(profHigh2.History[0].RiskFlags, "Yüksek düşüş (%95)") {
		t.Fatalf("eşik=90 iken drawdown=95 bayrak vermeli: %v", profHigh2.History[0].RiskFlags)
	}
}

// TestCreatorsListIncludesUnscored, henüz Worker tarafından skorlanmamış creator'ların
// Creators listesinden düşmemesini (LEFT JOIN + COALESCE nötr) kilitler.
func TestCreatorsListIncludesUnscored(t *testing.T) {
	ctx := context.Background()
	ts := NewFakeTokenStore()
	f := ts.(*fakeTokenStore)
	seedToken(t, f, "m1", "A", "active", 0, 100, 0) // creator A yakalandı ama skorlanmadı
	rows, err := f.Creators(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Address != "A" || rows[0].RiskLevel != "medium" {
		t.Fatalf("skorlanmamış creator nötr olarak listelenmeli: %+v", rows)
	}
	if rows[0].ReputationScore != 0 {
		t.Fatalf("skorlanmamış creator reputationScore=0 olmalı: %+v", rows[0])
	}
}
