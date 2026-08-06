package market

import "testing"

func TestMomentumFromChange(t *testing.T) {
	cases := []struct{ in, want float64 }{{0, 50}, {100, 100}, {-100, 0}, {20, 60}, {300, 100}, {-300, 0}}
	for _, c := range cases {
		if got := momentumFromChange(c.in); got != c.want {
			t.Errorf("momentumFromChange(%v)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestAppendSparkCaps(t *testing.T) {
	var s []float64
	for i := 0; i < 20; i++ {
		s = appendSpark(s, float64(i))
	}
	if len(s) != sparkMax {
		t.Fatalf("spark len=%d want %d", len(s), sparkMax)
	}
	if s[len(s)-1] != 19 {
		t.Fatalf("son örnek=%v want 19", s[len(s)-1])
	}
}
