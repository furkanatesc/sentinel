package reputation

import (
	"fmt"
	"math"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Thresholds, skor ağırlıkları + güven eşiğidir (config'ten enjekte; deploy-tunable).
type Thresholds struct {
	MinResolved        int
	WRug, WFail, WGrad float64
}

// Reputation, saf skorlama sonucudur (metrikler agg'den worker'da taşınır).
type Reputation struct {
	Score, Confidence, SuccessRatePct float64
	RiskLevel                         string
	Breakdown                         []store.ScoreBreakdownItem
}

// Score, creator agrega → açıklanabilir itibar. active token'lar çözülmemiş sayılır
// (paydaya girmez). resolved==0 → nötr (conf 0, risk medium, boş breakdown).
func Score(agg store.CreatorAgg, th Thresholds) Reputation {
	resolved := agg.Rug + agg.Dumped + agg.Dead + agg.Graduated
	if resolved <= 0 {
		return Reputation{Score: 0, Confidence: 0, RiskLevel: "medium", Breakdown: []store.ScoreBreakdownItem{}}
	}
	rf := float64(resolved)
	rugRate := float64(agg.Rug) / rf
	failRate := float64(agg.Dumped+agg.Dead) / rf
	gradRate := float64(agg.Graduated) / rf

	rugPen := th.WRug * rugRate
	failPen := th.WFail * failRate
	gradRew := th.WGrad * gradRate
	score := clamp(50-rugPen-failPen+gradRew, 0, 100)

	conf := float64(resolved) / float64(th.MinResolved)
	if conf > 1 {
		conf = 1
	}
	breakdown := []store.ScoreBreakdownItem{
		{Label: "Taban", Weight: 50, Detail: "Nötr başlangıç"},
		{Label: "Rug oranı", Weight: -rugPen, Detail: fmt.Sprintf("%d/%d çözülmüş token rug", agg.Rug, resolved)},
		{Label: "Başarısız (dump/dead)", Weight: -failPen, Detail: fmt.Sprintf("%d/%d dump veya dead", agg.Dumped+agg.Dead, resolved)},
		{Label: "Graduated (başarı)", Weight: gradRew, Detail: fmt.Sprintf("%d/%d graduated", agg.Graduated, resolved)},
	}
	return Reputation{
		Score:          score,
		Confidence:     conf,
		SuccessRatePct: gradRate * 100,
		RiskLevel:      riskLevelFor(score),
		Breakdown:      breakdown,
	}
}

// riskLevelFor, frontend scoreToLevel bantlarının Go karşılığıdır (lib/format.ts).
func riskLevelFor(score float64) string {
	switch {
	case score <= 24:
		return "critical"
	case score <= 49:
		return "high"
	case score <= 69:
		return "medium"
	case score <= 84:
		return "good"
	default:
		return "strong"
	}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
