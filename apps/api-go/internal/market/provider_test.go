package market

import "testing"

func TestDexToLaunchpad(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		okExp bool
	}{
		{"pumpfun", "Pump.fun", true},
		{"pump-fun", "Pump.fun", true},
		{"raydium", "Raydium", true},
		{"raydium-clmm", "Raydium", true},
		{"raydium-cpmm", "Raydium", true},
		{"orca", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := DexToLaunchpad(c.in)
		if ok != c.okExp || got != c.want {
			t.Errorf("DexToLaunchpad(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.okExp)
		}
	}
}
