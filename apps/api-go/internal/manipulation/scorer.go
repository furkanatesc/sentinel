// Package manipulation, işlem-akışı agrega proxy'lerinden manipülasyon riski (0-100, inverted:
// yüksek=daha çok manipülasyon) hesaplar. Saf, ağsız/DB'siz (SRP). Skor bir PROXY'dir; gerçek
// per-trade wash/sniper tespiti 2e trade-flow ile yükseltilir.
package manipulation

import (
	"fmt"
	"math"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Thresholds, ağırlıklar + normalleştirme bantlarıdır (config'ten enjekte; deploy-tunable).
type Thresholds struct {
	MinTxns, ConfTxns                    int
	WImbalance, WWash, WVolume, WCreator float64
	WashMin, WashMax, VolMin, VolMax     float64
}

// Inputs, bir token'ın h24 işlem-akışı girdileridir.
type Inputs struct {
	Buys, Sells, Buyers                  int
	CreatorHoldingPct, Vol24h, Liquidity float64
}

// Result, hesaplanmış manipülasyon skorudur (breakdown yalnız katkısı>0 bileşenleri içerir).
type Result struct {
	Value, Confidence float64
	Breakdown         []store.ScoreBreakdownItem
}

func Score(in Inputs, th Thresholds) Result {
	txns := in.Buys + in.Sells
	if txns == 0 || txns < th.MinTxns {
		return Result{Value: 0, Confidence: 0, Breakdown: []store.ScoreBreakdownItem{}}
	}

	buyRatio := float64(in.Buys) / float64(txns)
	imbalanceNorm := clamp01(math.Abs(buyRatio-0.5) * 2)

	buyers := in.Buyers
	if buyers < 1 {
		buyers = 1
	}
	perBuyer := float64(in.Buys) / float64(buyers)
	washNorm := normBand(perBuyer, th.WashMin, th.WashMax)

	liq := in.Liquidity
	if liq < 1 {
		liq = 1
	}
	volLiq := in.Vol24h / liq
	volNorm := normBand(volLiq, th.VolMin, th.VolMax)

	creatorNorm := clamp01(in.CreatorHoldingPct / 100)

	cImb := th.WImbalance * imbalanceNorm
	cWash := th.WWash * washNorm
	cVol := th.WVolume * volNorm
	cCreator := th.WCreator * creatorNorm

	value := cImb + cWash + cVol + cCreator
	value = clamp(0, 100, value)

	bd := []store.ScoreBreakdownItem{}
	if cImb > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Alım/satım dengesizliği", Weight: round1(cImb), Detail: fmt.Sprintf("buyRatio=%.2f", buyRatio)})
	}
	if cWash > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Wash-proxy (işlem/alıcı)", Weight: round1(cWash), Detail: fmt.Sprintf("%.1f işlem/alıcı", perBuyer)})
	}
	if cVol > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Hacim/likidite anomalisi", Weight: round1(cVol), Detail: fmt.Sprintf("%.1fx", volLiq)})
	}
	if cCreator > 0 {
		bd = append(bd, store.ScoreBreakdownItem{Label: "Creator payı", Weight: round1(cCreator), Detail: fmt.Sprintf("%%%.0f", in.CreatorHoldingPct)})
	}

	conf := float64(txns) / float64(th.ConfTxns)
	if conf > 1 {
		conf = 1
	}
	return Result{Value: value, Confidence: conf, Breakdown: bd}
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// clamp, x'i [lo,hi] aralığına sıkıştırır (spec §3: value = clamp[0,100](...)).
func clamp(lo, hi, x float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// normBand, x'i [min,max] bandına göre [0,1]'e taşır (min altı 0, max üstü 1).
func normBand(x, min, max float64) float64 {
	if max <= min {
		return 0
	}
	return clamp01((x - min) / (max - min))
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }
