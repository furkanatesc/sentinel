package alerteval

import (
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestMatchRule(t *testing.T) {
	base := store.EventRow{Type: "new_mint", Symbol: "AAA", Launchpad: "Pump.fun", Liquidity: 20000, CreatorScore: 80, RiskLevel: "medium", Ts: 10}
	rule := store.AlertRule{Trigger: "new_mint", Scope: "Tüm tokenlar", MinLiquidity: 10000, MinCreatorScore: 70, MaxRisk: "high", Enabled: true}

	cases := []struct {
		name string
		e    store.EventRow
		r    store.AlertRule
		want bool
	}{
		{"tam eşleşme", base, rule, true},
		{"trigger farklı", base, withTrigger(rule, "liquidity_removed"), false},
		{"kapsam-dışı trigger", withType(base, "whale_activity"), withTrigger(rule, "whale_activity"), false},
		{"likidite düşük", withLiq(base, 5000), rule, false},
		{"skor düşük", withScore(base, 50), rule, false},
		{"risk tavanı aşıldı", withRisk(base, "critical"), rule, false},
		{"risk tavanı sınırında", withRisk(base, "high"), rule, true},
		{"scope launchpad eşleşir", base, withScope(rule, "Pump.fun"), true},
		{"scope launchpad eşleşmez", base, withScope(rule, "Raydium"), false},
		{"scope boş = hepsi", base, withScope(rule, ""), true},
	}
	for _, c := range cases {
		if got := MatchRule(c.e, c.r); got != c.want {
			t.Errorf("%s: MatchRule=%v want %v", c.name, got, c.want)
		}
	}
}

func withTrigger(r store.AlertRule, v string) store.AlertRule { r.Trigger = v; return r }
func withScope(r store.AlertRule, v string) store.AlertRule   { r.Scope = v; return r }
func withType(e store.EventRow, v string) store.EventRow      { e.Type = v; return e }
func withLiq(e store.EventRow, v float64) store.EventRow      { e.Liquidity = v; return e }
func withScore(e store.EventRow, v float64) store.EventRow    { e.CreatorScore = v; return e }
func withRisk(e store.EventRow, v string) store.EventRow      { e.RiskLevel = v; return e }
