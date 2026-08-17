package manipulation

import "testing"

func defTh() Thresholds {
	return Thresholds{MinTxns: 20, ConfTxns: 100, WImbalance: 30, WWash: 35, WVolume: 25, WCreator: 10,
		WashMin: 3, WashMax: 15, VolMin: 3, VolMax: 20}
}

func TestScoreNeutralBelowMinTxns(t *testing.T) {
	r := Score(Inputs{Buys: 5, Sells: 5, Buyers: 5}, defTh()) // txns=10 < 20
	if r.Value != 0 || r.Confidence != 0 || len(r.Breakdown) != 0 {
		t.Fatalf("nötr beklenir, gelen %+v", r)
	}
}

func TestScoreBalancedLowManipulation(t *testing.T) {
	// dengeli akış, çok alıcı, düşük vol/likidite, creator payı yok → düşük skor
	r := Score(Inputs{Buys: 50, Sells: 50, Buyers: 90, Vol24h: 1000, Liquidity: 100000}, defTh())
	if r.Value > 5 {
		t.Fatalf("dengeli akış düşük skor beklenir, gelen %.1f", r.Value)
	}
	if r.Confidence != 1 { // txns=100 == ConfTxns
		t.Fatalf("conf 1.0 beklenir, gelen %.2f", r.Confidence)
	}
}

func TestScoreAllBuyMaxImbalance(t *testing.T) {
	r := Score(Inputs{Buys: 100, Sells: 0, Buyers: 100, Vol24h: 0, Liquidity: 100000}, defTh())
	// imbalanceNorm=1 → W_imbalance=30 katkı; diğerleri ~0
	if r.Value < 29 || r.Value > 31 {
		t.Fatalf("hep-alım ~30 beklenir, gelen %.1f", r.Value)
	}
	if len(r.Breakdown) == 0 || r.Breakdown[0].Label != "Alım/satım dengesizliği" {
		t.Fatalf("dengesizlik breakdown beklenir, gelen %+v", r.Breakdown)
	}
}

func TestScoreWashProxyMaxed(t *testing.T) {
	// 3 alıcı 60 alım → perBuyer=20 > WashMax=15 → washNorm=1 → +35
	r := Score(Inputs{Buys: 60, Sells: 0, Buyers: 3, Liquidity: 100000}, defTh())
	// imbalance=1 (+30) + wash=1 (+35) = 65
	if r.Value < 64 || r.Value > 66 {
		t.Fatalf("hep-alım+wash ~65 beklenir, gelen %.1f", r.Value)
	}
}

func TestScoreClampedTo100(t *testing.T) {
	r := Score(Inputs{Buys: 100, Sells: 0, Buyers: 1, Vol24h: 1e9, Liquidity: 1, CreatorHoldingPct: 100}, defTh())
	if r.Value != 100 {
		t.Fatalf("clamp 100 beklenir, gelen %.1f", r.Value)
	}
}

func TestScoreConfidenceRamp(t *testing.T) {
	r := Score(Inputs{Buys: 25, Sells: 25, Buyers: 40}, defTh()) // txns=50 → 0.5
	if r.Confidence < 0.49 || r.Confidence > 0.51 {
		t.Fatalf("conf 0.5 beklenir, gelen %.2f", r.Confidence)
	}
}
