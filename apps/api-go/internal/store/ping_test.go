package store

import (
	"context"
	"testing"
)

func TestFakeStorePingOK(t *testing.T) {
	var p Pinger = NewFakeTokenStore().(Pinger)
	if err := p.Ping(context.Background()); err != nil {
		t.Fatalf("fake ping err = %v, want nil", err)
	}
}
