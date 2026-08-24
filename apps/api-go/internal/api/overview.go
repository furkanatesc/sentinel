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

func kpisHandler(ts store.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ts.Kpis(r.Context())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "kpis unavailable"})
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		empty := []float64{}
		kpis := []Kpi{
			{ID: "detected", Label: "Tespit Edilen Token (24s)", Value: strconv.Itoa(c.Detected), Spark: empty, Updated: now},
			{ID: "highconf", Label: "Yüksek Güvenli Token", Value: strconv.Itoa(c.HighConf), Spark: empty, Updated: now, Tone: "positive"},
			{ID: "critical", Label: "Kritik Risk Tespiti", Value: strconv.Itoa(c.Critical), Spark: empty, Updated: now, Tone: "critical"},
			{ID: "signals", Label: "Aktif Sinyaller", Value: strconv.Itoa(c.Signals), Spark: empty, Updated: now},
			{ID: "positions", Label: "Açık Pozisyonlar", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "realized", Label: "Gerçekleşen K/Z (24s)", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "unrealized", Label: "Gerçekleşmemiş K/Z", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
			{ID: "latency", Label: "Sistem Gecikmesi", Value: "—", Spark: empty, Updated: now, Tone: "neutral"},
		}
		writeJSON(w, http.StatusOK, kpis)
	}
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
