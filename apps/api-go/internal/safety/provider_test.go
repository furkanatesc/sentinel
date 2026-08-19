package safety

import (
	"context"
	"errors"
	"strings"
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
	count      int
	top10      float64
	creatorPct float64
	capped     bool
	err        error
}

func (f fakeHolders) HolderDistribution(context.Context, string, string, int) (int, float64, float64, bool, error) {
	return f.count, f.top10, f.creatorPct, f.capped, f.err
}

func TestFetchOnChainBothOK(t *testing.T) {
	p := NewHeliusProvider(fakeAuth{mint: true, freeze: false}, fakeHolders{count: 300, top10: 42}, 5000)
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
	p := NewHeliusProvider(fakeAuth{mint: true, freeze: false}, fakeHolders{count: 5000, top10: 60, capped: true}, 5000)
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
	p := NewHeliusProvider(fakeAuth{err: errors.New("boom")}, fakeHolders{count: 300, top10: 42}, 5000)
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
	p := NewHeliusProvider(fakeAuth{err: errors.New("auth 429")}, fakeHolders{err: errors.New("holders 429")}, 5000)
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
