package safety

import "context"

// OnChainData, Scorer'ın on-chain girdisidir (Known bayrakları kısmi-veriyi taşır).
type OnChainData struct {
	MintAuthorityActive, FreezeAuthorityActive bool
	AuthoritiesKnown                           bool
	HolderCount                                int
	Top10Pct                                   float64
	HoldersKnown                               bool
	HoldersCapped                              bool
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
	if count, top10, capped, err := p.holders.HolderDistribution(ctx, mint, p.holdersCap); err == nil {
		d.HolderCount, d.Top10Pct, d.HoldersKnown, d.HoldersCapped = count, top10, true, capped
	}
	return d, nil
}

var _ DataProvider = (*HeliusProvider)(nil)
