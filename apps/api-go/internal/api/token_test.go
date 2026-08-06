package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

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

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
