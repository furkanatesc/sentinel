package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestEventsHandler(t *testing.T) {
	es := store.NewFakeEventStore()
	_ = es.InsertEvent(nil, store.EventRow{ID: "e1", Type: "new_mint", Mint: "M", Ts: 1})
	ts := store.NewFakeTokenStore()
	_ = ts.UpsertToken(nil, store.TokenRow{ID: "M", Mint: "M", Symbol: "S"}, 1, "")

	r := NewRouter(RouterDeps{Events: es, Tokens: ts, EventsWindow: 200})

	for _, tc := range []struct{ path, wantKey string }{
		{"/api/events", "type"}, {"/api/tokens", "symbol"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s code=%d", tc.path, w.Code)
		}
		var arr []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
			t.Fatalf("%s json: %v", tc.path, err)
		}
		if len(arr) == 0 || arr[0][tc.wantKey] == nil {
			t.Fatalf("%s body=%s", tc.path, w.Body.String())
		}
	}
}

func TestEventsHandlerEmptyIsArray(t *testing.T) {
	r := NewRouter(RouterDeps{Events: store.NewFakeEventStore(), Tokens: store.NewFakeTokenStore(), EventsWindow: 200})
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Body.String() != "[]\n" && w.Body.String() != "[]" {
		t.Fatalf("boş sonuç [] olmalı, got %q", w.Body.String())
	}
}
