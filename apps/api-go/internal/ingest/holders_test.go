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

func TestHolderDistributionTop10(t *testing.T) {
	// 12 hesap; 2 hesap aynı owner (birleşmeli). Toplam amount 120; top-10 owner toplamı hesaplanır.
	// Helius DAS getTokenAccounts `amount`'ı JSON SAYI döndürür (string değil) — canlı şekil (deploy'da doğrulandı).
	page := `{"jsonrpc":"2.0","id":"1","result":{"token_accounts":[
		{"owner":"o1","amount":50},{"owner":"o2","amount":20},{"owner":"o3","amount":10},
		{"owner":"o4","amount":8},{"owner":"o5","amount":7},{"owner":"o6","amount":6},
		{"owner":"o7","amount":5},{"owner":"o8","amount":4},{"owner":"o9","amount":3},
		{"owner":"o10","amount":2},{"owner":"o11","amount":3},{"owner":"o1","amount":2}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(page)) // tek sayfa (12 < 1000 → son sayfa)
	}))
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	count, top10, capped, err := h.HolderDistribution(context.Background(), "MintX", 5000)
	if err != nil || capped {
		t.Fatalf("err=%v capped=%v", err, capped)
	}
	// unique owner: o1(52),o2..o10,o11 = 11 owner. Toplam = 120.
	if count != 11 {
		t.Fatalf("unique owner sayısı=%d want 11", count)
	}
	// top-10 owner (en büyük 10): 52+20+10+8+7+6+5+4+3+3 = 118; %118/120 = 98.33
	if top10 < 98.0 || top10 > 98.7 {
		t.Fatalf("top10Pct=%.2f want ~98.3", top10)
	}
}

func TestHolderDistributionStringOrNullAmountTolerant(t *testing.T) {
	// amount string ("30"), null, ve eksik → flexAmount tolere etmeli (null/bozuk → 0), sayfa düşmemeli.
	page := `{"jsonrpc":"2.0","id":"1","result":{"token_accounts":[
		{"owner":"a","amount":"30"},{"owner":"b","amount":10},{"owner":"c","amount":null}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(page))
	}))
	defer srv.Close()
	h := NewHeliusHolders(srv.URL)
	count, top10, _, err := h.HolderDistribution(context.Background(), "MintX", 5000)
	if err != nil {
		t.Fatalf("string/null amount hata vermemeli: %v", err)
	}
	if count != 3 { // a,b,c benzersiz owner (c amount 0 ama yine owner)
		t.Fatalf("count=%d want 3", count)
	}
	// toplam=40 (30+10+0); top-10 = 40 → %100
	if top10 < 99.9 || top10 > 100.1 {
		t.Fatalf("top10=%.2f want 100 (30+10 / 40)", top10)
	}
}
