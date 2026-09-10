package store

import (
	"context"
	"testing"
)

func TestFakeAlertEventStore(t *testing.T) {
	s := NewFakeAlertEventStore()
	ctx := context.Background()

	if wm, err := s.GetAlertWatermark(ctx); err != nil || wm != 0 {
		t.Fatalf("başlangıç watermark 0 beklenir: %d %v", wm, err)
	}
	if ins, _ := s.InsertAlertEvent(ctx, AlertEventRow{ID: "a1", RuleID: "r1", Type: "new_mint", Token: "AAA", Severity: "info", Ts: 100}); !ins {
		t.Fatal("ilk insert inserted=true beklenir")
	}
	_, _ = s.InsertAlertEvent(ctx, AlertEventRow{ID: "a2", RuleID: "r2", Type: "liquidity_removed", Token: "BBB", Severity: "critical", Ts: 200})
	// aynı ID → idempotent atlama (inserted=false)
	if ins, _ := s.InsertAlertEvent(ctx, AlertEventRow{ID: "a1", RuleID: "r1", Ts: 100}); ins {
		t.Fatal("tekrar insert inserted=false beklenir (dedup)")
	}
	got, err := s.RecentAlertEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "a2" {
		t.Fatalf("newest-first [a2 a1] beklenir: %+v", got)
	}
	if err := s.SetAlertWatermark(ctx, 200); err != nil {
		t.Fatal(err)
	}
	if wm, _ := s.GetAlertWatermark(ctx); wm != 200 {
		t.Fatalf("watermark 200 beklenir: %d", wm)
	}
	if got, _ := s.RecentAlertEvents(ctx, 1); len(got) != 1 {
		t.Fatalf("limit 1 beklenir: %d", len(got))
	}
}
