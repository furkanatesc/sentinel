package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestAuthorityGraphEndpoint_EmptyIsArray(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Tokens: ts.(store.TokenStore), WalletGraphMinCluster: 2, WalletGraphMaxDegree: 50})
	req := httptest.NewRequest(http.MethodGet, "/api/authority-graph", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var g store.WalletGraphResult
	if err := json.NewDecoder(w.Body).Decode(&g); err != nil {
		t.Fatal(err)
	}
	if g.Nodes == nil || g.Edges == nil {
		t.Fatalf("boş graph JSON'da [] olmalı (null değil): %+v", g)
	}
}
