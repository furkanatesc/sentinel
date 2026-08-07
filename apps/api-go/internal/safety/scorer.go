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
	HoldersCapped                              bool // holder pagination cap'e takıldı mı (confidence düşürür)
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
		if in.HoldersCapped {
			conf += 0.25
		} else {
			conf += 0.5
		}
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
			default:
				panic("safety: unknown risk group: " + o.riskGroup)
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
