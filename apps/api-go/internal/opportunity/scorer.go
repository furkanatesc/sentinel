// Package opportunity, diğer üç skoru (+ momentum) confidence-ağırlıklı birleştirip 0-100
// kompozit "fırsat" skoru + signal üretir. Saf, ağsız/DB'siz (SRP). Girdiler iyileştikçe
// (safety holders DAS, creator WS) opportunity otomatik keskinleşir — türev lens.
package opportunity

import (
	"fmt"
	"math"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Ağırlıklar (Σ=1.00) ve signal eşikleri — kod-sabiti (kalibrasyon; YAGNI: config değil).
const (
	wSafety       = 0.30
	wCreator      = 0.25
	wManipulation = 0.25
	wMomentum     = 0.20

	signalBuy     = 70.0
	signalWatch   = 45.0
	signalMinConf = 0.25 // en ağır tek-girdi ağırlığı (safety 0.30) altında: tek confident girdi sinyal verebilir
)

type Inputs struct {
	Safety, SafetyConf             float64
	Creator, CreatorConf           float64
	Manipulation, ManipulationConf float64
	Momentum, Liquidity            float64
}

type Result struct {
	Value, Confidence float64
	Breakdown         []store.ScoreBreakdownItem
	Signal            string // "buy"|"watch"|"avoid"|"" ("" → null)
}

type comp struct {
	label  string
	score  float64 // opportunity yönünde (yüksek=iyi), 0-100
	conf   float64
	weight float64
	detail string
}

func Score(in Inputs) Result {
	// momentum confidence proxy: enrichment yoksa (liquidity==0) momentum "bilinmiyor" → conf0.
	momentumConf := 0.0
	if in.Liquidity > 0 {
		momentumConf = 1.0
	}
	comps := []comp{
		{"Token güvenliği", in.Safety, in.SafetyConf, wSafety, fmt.Sprintf("%.0f/100 (conf %%%.0f)", in.Safety, in.SafetyConf*100)},
		{"Üretici itibarı", in.Creator, in.CreatorConf, wCreator, fmt.Sprintf("%.0f/100 (conf %%%.0f)", in.Creator, in.CreatorConf*100)},
		{"Manipülasyon (ters)", 100 - in.Manipulation, in.ManipulationConf, wManipulation, fmt.Sprintf("100-%.0f=%.0f (conf %%%.0f)", in.Manipulation, 100-in.Manipulation, in.ManipulationConf*100)},
		{"Momentum", in.Momentum, momentumConf, wMomentum, fmt.Sprintf("%.0f/100", in.Momentum)},
	}

	var num, W, wsum float64
	for _, c := range comps {
		wsum += c.weight
		ew := c.weight * c.conf
		num += c.score * ew
		W += ew
	}
	if W == 0 {
		return Result{Value: 0, Confidence: 0, Breakdown: []store.ScoreBreakdownItem{}, Signal: ""}
	}
	value := clamp(math.Round(num/W), 0, 100)
	confidence := W / wsum

	bd := []store.ScoreBreakdownItem{}
	for _, c := range comps {
		if c.conf <= 0 {
			continue // katkısız girdi breakdown'a girmez (dürüst)
		}
		contrib := c.score * c.weight * c.conf / W
		bd = append(bd, store.ScoreBreakdownItem{Label: c.label, Weight: round1(contrib), Detail: c.detail})
	}
	return Result{Value: value, Confidence: confidence, Breakdown: bd, Signal: deriveSignal(value, confidence)}
}

func deriveSignal(value, conf float64) string {
	if conf < signalMinConf {
		return ""
	}
	switch {
	case value >= signalBuy:
		return "buy"
	case value >= signalWatch:
		return "watch"
	default:
		return "avoid"
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
