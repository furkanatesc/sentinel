package outcome

import "testing"

func defThresholds() Thresholds {
	return Thresholds{RugLiqRatio: 0.10, GraduationMcap: 69000, DumpedDrawdown: 80, DeadVol: 100, MinLiqFloor: 500, DeadAgeSec: 86400}
}

func TestClassify(t *testing.T) {
	th := defThresholds()
	cases := []struct {
		name string
		in   Input
		want string
	}{
		{"rug: peak likidite yüksek, cur ~0", Input{CurMarketCap: 1000, CurLiquidity: 10, PeakMarketCap: 50000, PeakLiquidity: 20000, Vol24h: 5000, AgeSeconds: 3600}, OutcomeRug},
		{"graduated: peak mcap eşik üstü + likit", Input{CurMarketCap: 80000, CurLiquidity: 30000, PeakMarketCap: 90000, PeakLiquidity: 30000, Vol24h: 40000, AgeSeconds: 7200}, OutcomeGraduated},
		{"dumped: drawdown yüksek, likidite duruyor", Input{CurMarketCap: 5000, CurLiquidity: 8000, PeakMarketCap: 50000, PeakLiquidity: 9000, Vol24h: 3000, AgeSeconds: 7200}, OutcomeDumped},
		{"dead: yaşlı + vol~0", Input{CurMarketCap: 200, CurLiquidity: 600, PeakMarketCap: 800, PeakLiquidity: 700, Vol24h: 10, AgeSeconds: 200000}, OutcomeDead},
		{"active: taze + likit", Input{CurMarketCap: 3000, CurLiquidity: 4000, PeakMarketCap: 3200, PeakLiquidity: 4000, Vol24h: 6000, AgeSeconds: 600}, OutcomeActive},
		{"rug graduated'ı ezer (mezun sonrası rug)", Input{CurMarketCap: 2000, CurLiquidity: 50, PeakMarketCap: 90000, PeakLiquidity: 30000, Vol24h: 1000, AgeSeconds: 9000}, OutcomeRug},
	}
	for _, c := range cases {
		if got := Classify(c.in, th).Outcome; got != c.want {
			t.Errorf("%s: outcome = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestClassifyDrawdownAndLiquidityStatus(t *testing.T) {
	th := defThresholds()
	// peak=0 → drawdown 0, liquidityStatus unlocked, active.
	r := Classify(Input{CurMarketCap: 0, CurLiquidity: 0, PeakMarketCap: 0, PeakLiquidity: 0, Vol24h: 0, AgeSeconds: 10}, th)
	if r.MaxDrawdownPct != 0 || r.LiquidityStatus != LiquidityUnlocked || r.Outcome != OutcomeActive {
		t.Fatalf("peak=0 durumu: %+v", r)
	}
	// rug → liquidityStatus removed + drawdown hesaplanır.
	r = Classify(Input{CurMarketCap: 5000, CurLiquidity: 10, PeakMarketCap: 20000, PeakLiquidity: 15000, Vol24h: 100, AgeSeconds: 3600}, th)
	if r.LiquidityStatus != LiquidityRemoved {
		t.Fatalf("rug liquidityStatus = %q, want removed", r.LiquidityStatus)
	}
	if r.MaxDrawdownPct != 75 { // (20000-5000)/20000*100
		t.Fatalf("drawdown = %v, want 75", r.MaxDrawdownPct)
	}
}
