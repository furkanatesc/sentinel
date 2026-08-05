package ws

import (
	"context"
	"testing"
	"time"
)

func TestHubBroadcastFanOut(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c1 := h.registerForTest()
	c2 := h.registerForTest()
	// Run döngüsünün register'ları işlemesine izin ver
	waitFor(t, func() bool { return h.ClientCount() == 2 })

	h.Broadcast("events", map[string]any{"id": "e1"})

	for _, c := range []*client{c1, c2} {
		select {
		case msg := <-c.send:
			if msg.Topic != "events" {
				t.Fatalf("topic=%s", msg.Topic)
			}
		case <-time.After(time.Second):
			t.Fatal("client mesaj almadı")
		}
	}
}

func TestHubUnregister(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)
	c := h.registerForTest()
	waitFor(t, func() bool { return h.ClientCount() == 1 })
	h.unregisterForTest(c)
	waitFor(t, func() bool { return h.ClientCount() == 0 })
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("koşul zamanında sağlanmadı")
}
