package opportunity

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.5 }

func TestScore_CompositeWeighted(t *testing.T) {
	// safety 80@1, creator 60@1, manip 10@1 (→ters 90), momentum 50, liq>0 (conf1)
	// W = 0.30+0.25+0.25+0.20 = 1.00
	// num = 80*0.30 + 60*0.25 + 90*0.25 + 50*0.20 = 24+15+22.5+10 = 71.5
	r := Score(Inputs{Safety: 80, SafetyConf: 1, Creator: 60, CreatorConf: 1,
		Manipulation: 10, ManipulationConf: 1, Momentum: 50, Liquidity: 1000})
	if r.Value != 72 {
		t.Fatalf("value=%.2f want 72 (round of 71.5)", r.Value)
	}
	if !approx(r.Confidence, 1.0) {
		t.Fatalf("conf=%.2f want 1.0", r.Confidence)
	}
	if r.Signal != "buy" {
		t.Fatalf("signal=%q want buy", r.Signal)
	}
	if len(r.Breakdown) != 4 {
		t.Fatalf("breakdown=%d want 4", len(r.Breakdown))
	}
}

func TestScore_NeutralWhenAllConfZero(t *testing.T) {
	r := Score(Inputs{Safety: 80, SafetyConf: 0, Creator: 60, CreatorConf: 0,
		Manipulation: 10, ManipulationConf: 0, Momentum: 50, Liquidity: 0})
	if r.Value != 0 || r.Confidence != 0 || len(r.Breakdown) != 0 || r.Signal != "" {
		t.Fatalf("nötr bekleniyordu: %+v", r)
	}
}

func TestScore_MomentumGatedByLiquidity(t *testing.T) {
	// momentum var ama liquidity=0 → momentum katkı vermez (conf0). Yalnız safety kalır.
	r := Score(Inputs{Safety: 80, SafetyConf: 1, Momentum: 90, Liquidity: 0})
	if !approx(r.Value, 80) { // W=0.30, num=80*0.30 → /0.30 = 80
		t.Fatalf("value=%.2f want 80 (momentum atlanmalı)", r.Value)
	}
	if !approx(r.Confidence, 0.30) {
		t.Fatalf("conf=%.2f want 0.30", r.Confidence)
	}
}

func TestScore_ManipulationInverted(t *testing.T) {
	// yüksek manip (90) tek girdi → ters 10 → düşük opportunity → avoid
	r := Score(Inputs{Manipulation: 90, ManipulationConf: 1})
	if !approx(r.Value, 10) {
		t.Fatalf("value=%.2f want 10 (100-90)", r.Value)
	}
	if r.Signal != "avoid" {
		t.Fatalf("signal=%q want avoid", r.Signal)
	}
}

func TestSignal_Thresholds(t *testing.T) {
	// watch: value 45-69 + yeterli conf
	r := Score(Inputs{Safety: 50, SafetyConf: 1})
	if r.Signal != "watch" {
		t.Fatalf("value=%.1f signal=%q want watch", r.Value, r.Signal)
	}
	// düşük conf → null (""): tek girdi conf0.2 (W=0.30*... aslında conf gelir). min-conf<0.25 kur:
	r2 := Score(Inputs{Safety: 90, SafetyConf: 0.5}) // conf = 0.30*0.5 = 0.15 < 0.25 → null
	if r2.Signal != "" {
		t.Fatalf("conf=%.2f signal=%q want '' (null)", r2.Confidence, r2.Signal)
	}
}
