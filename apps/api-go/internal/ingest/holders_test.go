package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeHeliusHolders, toplam `total` hesabı `limit`'lik sayfalar halinde döndürür.
func fakeHeliusServer(t *testing.T, total, limit int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params struct {
				Page int `json:"page"`
			} `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		start := (req.Params.Page - 1) * limit
		n := 0
		if start < total {
			n = total - start
			if n > limit {
				n = limit
			}
		}
		accs := make([]map[string]any, n)
		for i := range accs {
			accs[i] = map[string]any{"address": fmt.Sprintf("acc%d", start+i)}
		}
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "1",
			"result": map[string]any{"total": n, "limit": limit, "page": req.Params.Page, "token_accounts": accs}})
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
}

func TestHoldersCountSinglePage(t *testing.T) {
	srv := fakeHeliusServer(t, 42, 1000)
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	n, capped, err := h.HoldersCount(context.Background(), "MintX", 5000)
	if err != nil || n != 42 || capped {
		t.Fatalf("n=%d capped=%v err=%v (want 42/false)", n, capped, err)
	}
}

func TestHoldersCountMultiPage(t *testing.T) {
	srv := fakeHeliusServer(t, 2500, 1000)
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	n, capped, err := h.HoldersCount(context.Background(), "MintX", 5000)
	if err != nil || n != 2500 || capped {
		t.Fatalf("n=%d capped=%v err=%v (want 2500/false)", n, capped, err)
	}
}

func TestHoldersCountCapped(t *testing.T) {
	srv := fakeHeliusServer(t, 999999, 1000)
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	n, capped, err := h.HoldersCount(context.Background(), "MintX", 3000)
	if err != nil || !capped || n < 3000 {
		t.Fatalf("n=%d capped=%v err=%v (want >=3000/true)", n, capped, err)
	}
}
