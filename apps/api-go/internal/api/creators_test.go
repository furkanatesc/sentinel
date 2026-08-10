package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

func TestCreatorsEndpoint(t *testing.T) {
	ts := store.NewFakeTokenStore()
	_ = ts.UpsertToken(context.Background(), store.TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	r := NewRouter(RouterDeps{Creators: ts.(store.CreatorStore), CreatorsLimit: 100})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/creators", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var out []store.CreatorRow
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Address != "AAA" || out[0].TotalTokens != 1 {
		t.Fatalf("body = %+v", out)
	}
}

func TestCreatorDetailEndpointFound(t *testing.T) {
	ts := store.NewFakeTokenStore()
	_ = ts.UpsertToken(context.Background(), store.TokenRow{ID: "m1", Mint: "m1", Symbol: "S1"}, 100, "AAA")
	r := NewRouter(RouterDeps{Creators: ts.(store.CreatorStore), CreatorsLimit: 100})

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/creator/AAA", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var p store.CreatorProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Address != "AAA" || p.Metrics.TotalTokens != 1 {
		t.Fatalf("profil = %+v", p)
	}
}

func TestCreatorDetailEndpointNotFound(t *testing.T) {
	ts := store.NewFakeTokenStore()
	r := NewRouter(RouterDeps{Creators: ts.(store.CreatorStore), CreatorsLimit: 100})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/creator/NOPE", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
