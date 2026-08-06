package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
