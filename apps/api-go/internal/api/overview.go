package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// Kpi, frontend Kpi (types.ts) ile birebir JSON şeklidir.
type Kpi struct {
	ID      string    `json:"id"`
	Label   string    `json:"label"`
	Value   string    `json:"value"`
	Change  float64   `json:"change"`
	Spark   []float64 `json:"spark"`
	Updated string    `json:"updated"`
	Tone    string    `json:"tone,omitempty"`
}

// kpiTrend, kronolojik değerlerden bir metriğin spark dizisi + yüzde değişimini türetir (saf).
// change = ilk→son yüzde ((last-first)/first*100); <2 örnek ya da first==0 → 0. spark = değerler.
func kpiTrend(vals []int) ([]float64, float64) {
	spark := make([]float64, len(vals))
	for i, v := range vals {
		spark[i] = float64(v)
	}
	var change float64
	if len(vals) >= 2 && vals[0] != 0 {
		change = (float64(vals[len(vals)-1]) - float64(vals[0])) / float64(vals[0]) * 100
	}
	return spark, change
}

func kpisHandler(ts store.TokenStore, sparkWindow int) http.HandlerFunc {
	if sparkWindow <= 0 {
		sparkWindow = 24 // config bağlanana kadar güvenli varsayılan
	}
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ts.Kpis(r.Context())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "kpis unavailable"})
			return
		}
		// Zaman-serisi (best-effort): örnek yoksa/okunamıyorsa spark boş, change 0 (geriye uyumlu).
		samples, err := ts.RecentKpiSamples(r.Context(), sparkWindow)
		if err != nil {
			samples = nil // best-effort: hata yut, spark/change boş kalır
		}
		detSp, detCh := kpiTrend(pickKpi(samples, func(s store.KpiSample) int { return s.Detected }))
		hcSp, hcCh := kpiTrend(pickKpi(samples, func(s store.KpiSample) int { return s.HighConf }))
		crSp, crCh := kpiTrend(pickKpi(samples, func(s store.KpiSample) int { return s.Critical }))
		sgSp, sgCh := kpiTrend(pickKpi(samples, func(s store.KpiSample) int { return s.Signals }))
		now := time.Now().UTC().Format(time.RFC3339)
		empty := []float64{}
		kpis := []Kpi{
			{ID: "detected", Label: "Tespit Edilen Token (24s)", Value: strconv.Itoa(c.Detected), Spark: detSp, Change: detCh, Updated: now},
			{ID: "highconf", Label: "Yüksek Güvenli Token", Value: strconv.Itoa(c.HighConf), Spark: hcSp, Change: hcCh, Updated: now, Tone: "positive"},
			{ID: "critical", Label: "Kritik Risk Tespiti", Value: strconv.Itoa(c.Critical), Spark: crSp, Change: crCh, Updated: now, Tone: "critical"},
			{ID: "signals", Label: "Aktif Sinyaller", Value: strconv.Itoa(c.Signals), Spark: sgSp, Change: sgCh, Updated: now},
			{ID: "positions", Label: "Açık Pozisyonlar", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "realized", Label: "Gerçekleşen K/Z (24s)", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "unrealized", Label: "Gerçekleşmemiş K/Z", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "latency", Label: "Sistem Gecikmesi", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
		}
		writeJSON(w, http.StatusOK, kpis)
	}
}

// pickKpi, örnek dizisinden tek bir metriğin değerlerini (kronolojik) çıkarır.
func pickKpi(samples []store.KpiSample, get func(store.KpiSample) int) []int {
	out := make([]int, len(samples))
	for i, s := range samples {
		out[i] = get(s)
	}
	return out
}

func radarHandler(ts store.TokenStore, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pts, err := ts.Radar(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "radar unavailable"})
			return
		}
		if pts == nil {
			pts = []store.RadarPoint{}
		}
		writeJSON(w, http.StatusOK, pts)
	}
}
