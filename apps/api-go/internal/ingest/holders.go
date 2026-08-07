package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// HeliusHolders, bir mint'in holder (token account) sayısını Helius getTokenAccounts
// ile sayfalayarak sayar. cap'e ulaşınca durur (capped=true) — pahalı büyük token'ları sınırlar.
type HeliusHolders struct {
	rpcURL string
	http   *http.Client
}

func NewHeliusHolders(rpcURL string) *HeliusHolders {
	return &HeliusHolders{rpcURL: rpcURL, http: &http.Client{Timeout: 12 * time.Second}}
}

const holdersPageLimit = 1000

func (h *HeliusHolders) HoldersCount(ctx context.Context, mint string, cap int) (int, bool, error) {
	if cap <= 0 {
		cap = 5000
	}
	total := 0
	for page := 1; ; page++ {
		n, err := h.pageCount(ctx, mint, page)
		if err != nil {
			return total, false, err
		}
		total += n
		if total >= cap {
			return total, true, nil // cap'e ulaşıldı → floor
		}
		if n < holdersPageLimit {
			return total, false, nil // son sayfa (kısa)
		}
	}
}

func (h *HeliusHolders) pageCount(ctx context.Context, mint string, page int) (int, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getTokenAccounts",
		"params": map[string]any{"mint": mint, "page": page, "limit": holdersPageLimit},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("helius getTokenAccounts: status %d", res.StatusCode)
	}
	var r struct {
		Result struct {
			TokenAccounts []json.RawMessage `json:"token_accounts"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return 0, err
	}
	if r.Error != nil {
		return 0, fmt.Errorf("helius getTokenAccounts error: %s", r.Error.Message)
	}
	return len(r.Result.TokenAccounts), nil
}

// tokenAccount, getTokenAccounts sayfa öğesinin sadece gereken alanlarıdır.
type tokenAccount struct {
	Owner  string `json:"owner"`
	Amount string `json:"amount"`
}

// pageAccounts, tek sayfayı çeker (owner+amount). Kısa sayfa (< limit) → son sayfa.
func (h *HeliusHolders) pageAccounts(ctx context.Context, mint string, page int) ([]tokenAccount, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getTokenAccounts",
		"params": map[string]any{"mint": mint, "page": page, "limit": holdersPageLimit},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("helius getTokenAccounts: status %d", res.StatusCode)
	}
	var r struct {
		Result struct {
			TokenAccounts []tokenAccount `json:"token_accounts"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, fmt.Errorf("helius getTokenAccounts error: %s", r.Error.Message)
	}
	return r.Result.TokenAccounts, nil
}

// HolderDistribution, benzersiz-sahip holder sayısı ve top-10 sahip yoğunlaşması (%) döndürür.
// cap'e ulaşınca durur (capped=true — pahalı büyük token'ları sınırlar; sonuç alt-sınırdır).
func (h *HeliusHolders) HolderDistribution(ctx context.Context, mint string, capN int) (int, float64, bool, error) {
	if capN <= 0 {
		capN = 5000
	}
	byOwner := map[string]float64{}
	seen := 0
	capped := false
	for page := 1; ; page++ {
		accs, err := h.pageAccounts(ctx, mint, page)
		if err != nil {
			return 0, 0, false, err
		}
		for _, a := range accs {
			amt, _ := strconv.ParseFloat(a.Amount, 64)
			byOwner[a.Owner] += amt
		}
		seen += len(accs)
		if seen >= capN {
			capped = true
			break
		}
		if len(accs) < holdersPageLimit {
			break
		}
	}
	var total float64
	amounts := make([]float64, 0, len(byOwner))
	for _, v := range byOwner {
		amounts = append(amounts, v)
		total += v
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(amounts)))
	var top10 float64
	for i := 0; i < len(amounts) && i < 10; i++ {
		top10 += amounts[i]
	}
	pct := 0.0
	if total > 0 {
		pct = top10 / total * 100
	}
	return len(byOwner), pct, capped, nil
}
