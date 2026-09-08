package store

import (
	"context"
	"errors"
	"testing"
)

func TestFakeAlertRuleStore(t *testing.T) {
	s := NewFakeAlertRuleStore()
	ctx := context.Background()

	// seed'li (4 varsayılan)
	rules, err := s.ListAlertRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 4 {
		t.Fatalf("seed 4 kural beklenir: %d", len(rules))
	}

	// create → id üretir + listeye ekler
	created, err := s.CreateAlertRule(ctx, AlertRule{Name: "Yeni", Trigger: "new_mint", Scope: "Tüm", MaxRisk: "medium", Channels: []string{"slack"}, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("create id üretmeli")
	}
	rules, _ = s.ListAlertRules(ctx)
	if len(rules) != 5 {
		t.Fatalf("create sonrası 5: %d", len(rules))
	}

	// setEnabled → çevirir
	if err := s.SetAlertRuleEnabled(ctx, "r3", true); err != nil {
		t.Fatal(err)
	}
	rules, _ = s.ListAlertRules(ctx)
	for _, r := range rules {
		if r.ID == "r3" && !r.Enabled {
			t.Fatal("r3 enabled=true olmalı")
		}
	}

	// bilinmeyen id → ErrNotFound
	if err := s.SetAlertRuleEnabled(ctx, "yok", false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bilinmeyen id → ErrNotFound beklenir, got %v", err)
	}
}
