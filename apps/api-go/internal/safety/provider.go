package safety

import (
	"context"
	"fmt"
)

// OnChainData, Scorer'ın on-chain girdisidir (Known bayrakları kısmi-veriyi taşır).
// CreatorHoldingPct/Known, Scorer'a GİRMEZ — sadece 2c manipülasyon skoru için persist
// edilir (aynı holder-fetch'ten, sıfır ek RPC; Safety Scorer'ı bu alanlar etkilemez).
type OnChainData struct {
	MintAuthorityActive, FreezeAuthorityActive bool
	AuthoritiesKnown                           bool
	HolderCount                                int
	Top10Pct                                   float64
	HoldersKnown                               bool
	HoldersCapped                              bool
	CreatorHoldingPct                          float64
	CreatorHoldingKnown                        bool
}

// DataProvider, bir mint için on-chain güvenlik verisini sağlar (DIP).
type DataProvider interface {
	FetchOnChain(ctx context.Context, mint, creator string) (OnChainData, error)
}

// Authorities/Holders, HeliusProvider'ın bağımlı olduğu dar arayüzlerdir (DIP/ISP;
// ingest.HeliusAuthorities / ingest.HeliusHolders karşılar).
type Authorities interface {
	MintAuthorities(ctx context.Context, mint string) (mintActive, freezeActive bool, err error)
}
type Holders interface {
	HolderDistribution(ctx context.Context, mint, creator string, cap int) (count int, top10Pct, creatorPct float64, capped bool, err error)
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

func (p *HeliusProvider) FetchOnChain(ctx context.Context, mint, creator string) (OnChainData, error) {
	var d OnChainData
	var authErr, holderErr error
	if mintA, freezeA, err := p.auth.MintAuthorities(ctx, mint); err == nil {
		d.MintAuthorityActive, d.FreezeAuthorityActive, d.AuthoritiesKnown = mintA, freezeA, true
	} else {
		authErr = err
	}
	if count, top10, creatorPct, capped, err := p.holders.HolderDistribution(ctx, mint, creator, p.holdersCap); err == nil {
		d.HolderCount, d.Top10Pct, d.HoldersKnown, d.HoldersCapped = count, top10, true, capped
		d.CreatorHoldingPct, d.CreatorHoldingKnown = creatorPct, true
	} else {
		holderErr = err
	}
	// Kısmi başarı (biri çalıştı) → nil: mevcut degradation davranışı korunur, Known bayrağı
	// eksik veriyi taşır. İki kaynak da başarısızsa → hiç veri yok → hard-fail (her iki nedeni
	// de taşıyan sarmalı hata): worker skip eder (önceki gerçek skoru neutral ile ezmez) ve
	// nedeni loglar (gözlemlenebilirlik — sessiz nötr-sıfır yerine 429 görünür).
	if authErr != nil && holderErr != nil {
		return d, fmt.Errorf("safety on-chain fetch failed: authorities: %w; holders: %v", authErr, holderErr)
	}
	return d, nil
}

var _ DataProvider = (*HeliusProvider)(nil)
