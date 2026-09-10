package store

import (
	"context"
	"testing"
)

func TestFakeNotificationConfigStore(t *testing.T) {
	s := NewFakeNotificationConfigStore()
	ctx := context.Background()

	// seed'li varsayılan
	got, err := s.GetNotificationSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Channel != "#alerts" || got.MinSeverity != "warning" || !got.TradeApproval {
		t.Fatalf("seed varsayılanı beklenir, got %+v", got)
	}
	if len(got.Templates) != 3 {
		t.Fatalf("seed 3 şablon beklenir: %d", len(got.Templates))
	}

	// save → günceller + geri okunur
	next := NotificationSettings{
		Channel:       "#trades",
		MinSeverity:   "critical",
		QuietHours:    QuietHours{Start: "22:00", End: "06:00", Enabled: true},
		Templates:     []NotificationTemplate{{Trigger: "new_mint", Template: "yeni"}},
		TradeApproval: false,
	}
	saved, err := s.SaveNotificationSettings(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Channel != "#trades" || saved.MinSeverity != "critical" {
		t.Fatalf("save dönüşü güncel olmalı, got %+v", saved)
	}
	got, _ = s.GetNotificationSettings(ctx)
	if got.MinSeverity != "critical" || !got.QuietHours.Enabled || got.TradeApproval {
		t.Fatalf("save sonrası okuma güncel olmalı, got %+v", got)
	}
	if len(got.Templates) != 1 {
		t.Fatalf("save sonrası 1 şablon: %d", len(got.Templates))
	}

	// nil templates → boş dilim (nil değil)
	saved, _ = s.SaveNotificationSettings(ctx, NotificationSettings{Channel: "#x", Templates: nil})
	if saved.Templates == nil {
		t.Fatal("nil templates boş dilime normalize edilmeli")
	}
}
