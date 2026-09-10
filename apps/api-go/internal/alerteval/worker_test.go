package alerteval

import (
	"context"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

type stubSource struct {
	events []store.EventRow
	rules  []store.AlertRule
}

func (s *stubSource) RecentEvents(_ context.Context, _ int) ([]store.EventRow, error) {
	return append([]store.EventRow(nil), s.events...), nil
}
func (s *stubSource) ListAlertRules(_ context.Context) ([]store.AlertRule, error) {
	return append([]store.AlertRule(nil), s.rules...), nil
}

func TestCycleProducesAlertsAndAdvancesWatermarkOnce(t *testing.T) {
	ctx := context.Background()
	src := &stubSource{
		events: []store.EventRow{
			{ID: "e1", Type: "new_mint", Symbol: "AAA", Liquidity: 20000, CreatorScore: 80, RiskLevel: "medium", Detail: "d", Time: "1m", Ts: 100},
			{ID: "e2", Type: "liquidity_removed", Symbol: "BBB", Liquidity: 0, CreatorScore: 0, RiskLevel: "critical", Detail: "rug", Time: "2m", Ts: 200},
		},
		rules: []store.AlertRule{
			{ID: "r1", Name: "Yeni mint", Trigger: "new_mint", Scope: "Tüm tokenlar", MaxRisk: "high", Enabled: true},
			{ID: "r2", Name: "Likidite", Trigger: "liquidity_removed", Scope: "Tüm tokenlar", MaxRisk: "critical", Enabled: true},
			{ID: "r3", Name: "Kapalı", Trigger: "new_mint", Scope: "Tüm tokenlar", MaxRisk: "high", Enabled: false},
		},
	}
	sink := store.NewFakeAlertEventStore()
	w := NewWorker(WorkerDeps{Source: src, Sink: sink, Limit: 100})

	n, err := w.Cycle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("2 alarm beklenir (e1→r1, e2→r2; r3 kapalı): %d", n)
	}
	got, _ := sink.RecentAlertEvents(ctx, 10)
	if len(got) != 2 || got[0].Token != "BBB" {
		t.Fatalf("newest-first 2 alarm beklenir: %+v", got)
	}
	if got[0].Detail != "Likidite — rug" {
		t.Fatalf("detail = kural adı + event detail beklenir: %q", got[0].Detail)
	}
	if wm, _ := sink.GetAlertWatermark(ctx); wm != 200 {
		t.Fatalf("watermark en yeni event ts'ine ilerlemeli: %d", wm)
	}

	n2, _ := w.Cycle(ctx)
	if n2 != 0 {
		t.Fatalf("ikinci cycle 0 alarm beklenir (dedup): %d", n2)
	}
}
