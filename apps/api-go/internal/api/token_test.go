package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// blockingDetail, ctx iptal edilene kadar bloke olur (yavaş/throttle'lı upstream simülasyonu).
type blockingDetail struct{}

func (blockingDetail) Build(ctx context.Context, _ string) (store.TokenDetail, bool, error) {
	<-ctx.Done()
	return store.TokenDetail{}, false, ctx.Err()
}

type fakeDetail struct {
	byMint map[string]store.TokenDetail
}

func (f *fakeDetail) Build(_ context.Context, mint string) (store.TokenDetail, bool, error) {
	d, ok := f.byMint[mint]
	return d, ok, nil
}

func TestTokenHandlerFound(t *testing.T) {
	fd := &fakeDetail{byMint: map[string]store.TokenDetail{
		"MintX": {ID: "MintX", Mint: "MintX", Symbol: "TST", Price: 1.5}}}
	r := NewRouter(RouterDeps{TokenDetail: fd})
	req := httptest.NewRequest(http.MethodGet, "/api/token/MintX", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), `"symbol":"TST"`) {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestTokenHandlerNotFound(t *testing.T) {
	fd := &fakeDetail{byMint: map[string]store.TokenDetail{}}
	r := NewRouter(RouterDeps{TokenDetail: fd})
	req := httptest.NewRequest(http.MethodGet, "/api/token/YOK", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d want 404", w.Code)
	}
}

func TestTokenHandlerTimesOut(t *testing.T) {
	r := NewRouter(RouterDeps{TokenDetail: blockingDetail{}, TokenDetailTimeout: 30 * time.Millisecond})
	req := httptest.NewRequest(http.MethodGet, "/api/token/MintX", nil)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() { r.ServeHTTP(w, req); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler süresiz bloke oldu — deadline uygulanmadı")
	}
	if w.Code != http.StatusBadGateway {
		t.Fatalf("code=%d, want 502 (deadline → degraded yanıt)", w.Code)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
