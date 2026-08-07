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

func TestScoreHoldersCappedLowersConfidence(t *testing.T) {
	// Holders bilinir AMA cap'e takıldı → confidence katkısı 0.5 değil 0.25.
	r := Score(Inputs{
		AuthoritiesKnown: true, MintAuthorityActive: false, FreezeAuthorityActive: false,
		HoldersKnown: true, HoldersCapped: true, HolderCount: 500, Top10Pct: 30, Liquidity: 5000, Launchpad: "Raydium",
	})
	if r.Confidence != 0.75 { // 0.5 authorities + 0.25 capped holders
		t.Fatalf("capped holders confidence 0.75 olmalı: %v", r.Confidence)
	}
}

func TestScoreHoldersNotCappedFullConfidence(t *testing.T) {
	// Holders bilinir VE cap'e takılmadı → tam 0.5 katkı, toplam confidence 1.0.
	r := Score(Inputs{
		AuthoritiesKnown: true, MintAuthorityActive: false, FreezeAuthorityActive: false,
		HoldersKnown: true, HoldersCapped: false, HolderCount: 500, Top10Pct: 30, Liquidity: 5000, Launchpad: "Raydium",
	})
	if r.Confidence != 1.0 {
		t.Fatalf("non-capped holders confidence 1.0 olmalı: %v", r.Confidence)
	}
}

func TestScoreHoldersCappedOnlyNonNeutral(t *testing.T) {
	// Authority bilinmiyor, holders bilinir+capped → confidence 0.25 (neutral DEĞİL, 0'dan büyük).
	r := Score(Inputs{
		AuthoritiesKnown: false, HoldersKnown: true, HoldersCapped: true,
		HolderCount: 500, Top10Pct: 30, Liquidity: 5000, Launchpad: "Raydium",
	})
	if r.Confidence != 0.25 {
		t.Fatalf("capped-only confidence 0.25 olmalı: %v", r.Confidence)
	}
	if r.Score == 0 && r.Confidence == 0 {
		t.Fatalf("capped-only nötr-erken-dönüş OLMAMALI: %+v", r)
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
