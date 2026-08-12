package reputation

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

var th = Thresholds{MinResolved: 5, WRug: 50, WFail: 20, WGrad: 40}

func TestScoreAllRugIsZero(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 5, Rug: 5}, th)
	if r.Score != 0 || r.RiskLevel != "critical" {
		t.Fatalf("hepsi-rug: score=%v risk=%s, want 0/critical", r.Score, r.RiskLevel)
	}
	if r.Confidence != 1 {
		t.Fatalf("confidence=%v, want 1 (5 çözülmüş)", r.Confidence)
	}
}

func TestScoreAllGraduatedIsNinety(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 5, Graduated: 5}, th)
	if r.Score != 90 || r.RiskLevel != "strong" {
		t.Fatalf("hepsi-graduated: score=%v risk=%s, want 90/strong", r.Score, r.RiskLevel)
	}
	if r.SuccessRatePct != 100 {
		t.Fatalf("successRatePct=%v, want 100", r.SuccessRatePct)
	}
}

func TestScoreDumpedDeadIsThirty(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 4, Dumped: 2, Dead: 2}, th)
	if r.Score != 30 || r.RiskLevel != "high" {
		t.Fatalf("dump/dead: score=%v risk=%s, want 30/high", r.Score, r.RiskLevel)
	}
}

func TestScoreUnresolvedIsNeutral(t *testing.T) {
	// 3 active (çözülmemiş) → resolved=0 → nötr (conf 0, risk medium)
	r := Score(store.CreatorAgg{Address: "A", Total: 3, Active: 3}, th)
	if r.Confidence != 0 || r.RiskLevel != "medium" || r.Score != 0 {
		t.Fatalf("nötr olmalı: %+v", r)
	}
	if len(r.Breakdown) != 0 {
		t.Fatalf("nötr breakdown boş olmalı: %+v", r.Breakdown)
	}
}

func TestScoreConfidenceScalesWithResolved(t *testing.T) {
	// 1 çözülmüş (K=5) → 0.2; active paydaya girmez
	r := Score(store.CreatorAgg{Address: "A", Total: 6, Rug: 1, Active: 5}, th)
	if r.Confidence != 0.2 {
		t.Fatalf("confidence=%v, want 0.2 (1/5)", r.Confidence)
	}
	// rugRate=1/1=1 → score 0
	if r.Score != 0 {
		t.Fatalf("score=%v, want 0 (resolved üzerinden rugRate=1)", r.Score)
	}
}

func TestScoreBreakdownPresentWhenResolved(t *testing.T) {
	r := Score(store.CreatorAgg{Address: "A", Total: 5, Rug: 2, Graduated: 3}, th)
	if len(r.Breakdown) == 0 {
		t.Fatal("çözülmüşse breakdown dolu olmalı")
	}
}
