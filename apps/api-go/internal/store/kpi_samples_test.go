package store

import (
	"context"
	"testing"
)

func TestFakeKpiSamplesInsertRecentPrune(t *testing.T) {
	s := NewFakeTokenStore().(TokenStore)
	ctx := context.Background()
	for i, ts := range []int64{100, 200, 300} {
		if err := s.InsertKpiSample(ctx, ts, KpiCounts{Detected: i + 1}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.RecentKpiSamples(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Ts != 100 || got[2].Ts != 300 {
		t.Fatalf("kronolojik ASC beklenir: %+v", got)
	}
	if got[2].Detected != 3 {
		t.Fatalf("Detected taşınmalı: %+v", got[2])
	}
	// idempotent (aynı ts güncelle)
	if err := s.InsertKpiSample(ctx, 300, KpiCounts{Detected: 9}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.RecentKpiSamples(ctx, 10)
	if len(got) != 3 || got[2].Detected != 9 {
		t.Fatalf("aynı ts idempotent güncellemeli: %+v", got)
	}
	// prune: en yeni 2 kalsın
	if err := s.PruneKpiSamples(ctx, 2); err != nil {
		t.Fatal(err)
	}
	got, _ = s.RecentKpiSamples(ctx, 10)
	if len(got) != 2 || got[0].Ts != 200 {
		t.Fatalf("prune en yeni 2'yi tutmalı: %+v", got)
	}
	// limit: son 1
	got, _ = s.RecentKpiSamples(ctx, 1)
	if len(got) != 1 || got[0].Ts != 300 {
		t.Fatalf("limit=1 en yeni 1: %+v", got)
	}
}
