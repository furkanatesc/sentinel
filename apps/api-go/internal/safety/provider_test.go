package safety

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// authStub, safety.Authorities'in test-yerel sahtesi: pubkey döner (2e-2 — bool değil).
type authStub struct {
	mint, freeze *string
	err          error
}

func (a authStub) MintAuthorities(context.Context, string) (*string, *string, error) {
	return a.mint, a.freeze, a.err
}

type holdersStub struct {
	count      int
	top10      float64
	creatorPct float64
	capped     bool
	err        error
}

func (h holdersStub) HolderDistribution(context.Context, string, string, int) (int, float64, float64, bool, error) {
	return h.count, h.top10, h.creatorPct, h.capped, h.err
}

func TestFetchOnChainBothOK(t *testing.T) {
	mintPk := "Mint111"
	p := NewHeliusProvider(authStub{mint: &mintPk, freeze: nil}, holdersStub{count: 300, top10: 42}, 5000)
	d, err := p.FetchOnChain(context.Background(), "M", "")
	if err != nil {
		t.Fatal(err)
	}
	if !d.AuthoritiesKnown || !d.HoldersKnown || !d.MintAuthorityActive || d.FreezeAuthorityActive || d.HolderCount != 300 || d.Top10Pct != 42 {
		t.Fatalf("beklenmeyen: %+v", d)
	}
}

func TestFetchOnChainHoldersCappedPropagates(t *testing.T) {
	// Holders cap'e takılırsa (capped=true) OnChainData.HoldersCapped=true olmalı (confidence düşsün diye).
	mintPk := "Mint111"
	p := NewHeliusProvider(authStub{mint: &mintPk, freeze: nil}, holdersStub{count: 5000, top10: 60, capped: true}, 5000)
	d, err := p.FetchOnChain(context.Background(), "M", "")
	if err != nil {
		t.Fatal(err)
	}
	if !d.HoldersKnown {
		t.Fatal("capped olsa da HoldersKnown=true olmalı (kısmi veri var)")
	}
	if !d.HoldersCapped {
		t.Fatal("holders cap'e takılınca HoldersCapped=true olmalı")
	}
}

func TestFetchOnChainPartialFailureIsolated(t *testing.T) {
	// Authority hata verir, holders başarılı → AuthoritiesKnown=false, HoldersKnown=true, hard-fail YOK.
	p := NewHeliusProvider(authStub{err: errors.New("boom")}, holdersStub{count: 300, top10: 42}, 5000)
	d, err := p.FetchOnChain(context.Background(), "M", "")
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

func TestFetchOnChainBothFailHardErrors(t *testing.T) {
	// İki kaynak da hata verirse hiç veri yok → hard-fail (worker skip + önceki skoru koru).
	// Hata mesajı iki kaynağın nedenini de taşımalı (gözlemlenebilirlik — 429 görünsün).
	p := NewHeliusProvider(authStub{err: errors.New("auth 429")}, holdersStub{err: errors.New("holders 429")}, 5000)
	d, err := p.FetchOnChain(context.Background(), "M", "")
	if err == nil {
		t.Fatal("iki kaynak da başarısızsa hard-fail beklenir")
	}
	if d.AuthoritiesKnown || d.HoldersKnown {
		t.Fatalf("veri olmamalı: %+v", d)
	}
	if !strings.Contains(err.Error(), "auth 429") || !strings.Contains(err.Error(), "holders 429") {
		t.Fatalf("hata iki kaynağın nedenini de taşımalı: %v", err)
	}
}

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
