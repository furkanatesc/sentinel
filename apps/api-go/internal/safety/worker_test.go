package safety

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type errProvider struct{ err error }

func (e errProvider) FetchOnChain(context.Context, string, string) (OnChainData, error) {
	return OnChainData{}, e.err
}

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

func (s stubProvider) FetchOnChain(context.Context, string, string) (OnChainData, error) {
	return s.d, nil
}

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

func TestScoreOnceSkipsPersistOnTotalFailure(t *testing.T) {
	// İki kaynak da başarısız (FetchOnChain err) → persist ATLANMALI:
	// önceki gerçek skoru neutral 0 ile ezmemek için (geçici 429 skoru silmesin).
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000}}}
	w := NewWorker(WorkerDeps{Store: st, Provider: errProvider{err: errors.New("429")}, Limit: 10, Now: func() int64 { return 1 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(st.updates) != 0 {
		t.Fatalf("total-failure'da persist atlanmalı: %d update", len(st.updates))
	}
}

func TestScoreOnceEmitsDegradedCycleSummary(t *testing.T) {
	// Gözlemlenebilirlik: hiçbir token skorlanamazsa (Helius down) worker tik başına
	// TEK özet log basmalı — WARN seviyesinde, scored=0 ve örnek hata nedeni (429) ile.
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000}, {Mint: "M2", Liquidity: 3000}}}
	w := NewWorker(WorkerDeps{Store: st, Provider: errProvider{err: errors.New("429 too many requests")},
		Limit: 10, Now: func() int64 { return 1 }, Logger: logger})
	w.scoreOnce(context.Background())
	out := buf.String()
	if !strings.Contains(out, "safety cycle") {
		t.Fatalf("özet log 'safety cycle' beklenir: %s", out)
	}
	if !strings.Contains(out, `"scored":0`) || !strings.Contains(out, `"totalFail":2`) {
		t.Fatalf("scored=0 + totalFail=2 beklenir: %s", out)
	}
	if !strings.Contains(out, `"level":"WARN"`) {
		t.Fatalf("scored==0 → WARN seviyesi beklenir: %s", out)
	}
	if !strings.Contains(out, "429 too many requests") {
		t.Fatalf("sampleErr 429 nedeni loglanmalı: %s", out)
	}
}

func TestScoreOnceHealthyCycleSummaryInfo(t *testing.T) {
	// Sağlıklı tik (skorlama var) → özet log INFO seviyesinde (alarm değil).
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000, Launchpad: "Raydium"}}}
	prov := stubProvider{d: OnChainData{AuthoritiesKnown: true, HoldersKnown: true, HolderCount: 500, Top10Pct: 30}}
	w := NewWorker(WorkerDeps{Store: st, Provider: prov, Limit: 10, Now: func() int64 { return 1 }, Logger: logger})
	w.scoreOnce(context.Background())
	out := buf.String()
	if !strings.Contains(out, "safety cycle") || !strings.Contains(out, `"scored":1`) {
		t.Fatalf("scored=1 özet beklenir: %s", out)
	}
	if !strings.Contains(out, `"level":"INFO"`) {
		t.Fatalf("sağlıklı tik → INFO seviyesi beklenir: %s", out)
	}
}

func TestWorker_PersistsAuthorityAddrs(t *testing.T) {
	// Worker, provider'ın döndürdüğü authority pubkey'ini UpdateSafety'ye taşımalı (piggyback).
	st := &fakeSafetyStore{targets: []store.SafetyTarget{{Mint: "M1", Liquidity: 5000}}}
	prov := stubProvider{d: OnChainData{
		MintAuthorityActive: true, AuthoritiesKnown: true,
		MintAuthorityAddr: "MA", FreezeAuthorityAddr: "FA",
	}}
	w := NewWorker(WorkerDeps{Store: st, Provider: prov, Limit: 10, Now: func() int64 { return 1 }})
	if err := w.scoreOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(st.updates) != 1 {
		t.Fatalf("1 update beklenir: %d", len(st.updates))
	}
	last := st.updates[0]
	if !last.AuthoritiesKnown || last.MintAuthority != "MA" || last.FreezeAuthority != "FA" {
		t.Fatalf("authority piggyback taşınmalı, got known=%v mint=%q freeze=%q", last.AuthoritiesKnown, last.MintAuthority, last.FreezeAuthority)
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
