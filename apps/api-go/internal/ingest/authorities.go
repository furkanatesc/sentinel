package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HeliusAuthorities, bir SPL mint'inin mint/freeze authority durumunu getAccountInfo
// (jsonParsed) ile çeker. null authority = iptal (aktif değil).
type HeliusAuthorities struct {
	rpcURL string
	http   *http.Client
}

func NewHeliusAuthorities(rpcURL string) *HeliusAuthorities {
	return &HeliusAuthorities{rpcURL: rpcURL, http: &http.Client{Timeout: 8 * time.Second}}
}

// MintAuthorities, mint ve freeze authority'nin aktif (dolu) olup olmadığını döndürür.
func (h *HeliusAuthorities) MintAuthorities(ctx context.Context, mint string) (mintActive, freezeActive bool, err error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getAccountInfo",
		"params": []any{mint, map[string]any{"encoding": "jsonParsed"}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return false, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return false, false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false, false, fmt.Errorf("helius getAccountInfo: status %d", res.StatusCode)
	}
	var r struct {
		Result struct {
			Value struct {
				Data struct {
					Parsed struct {
						Info struct {
							MintAuthority   *string `json:"mintAuthority"`
							FreezeAuthority *string `json:"freezeAuthority"`
						} `json:"info"`
					} `json:"parsed"`
				} `json:"data"`
			} `json:"value"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return false, false, err
	}
	if r.Error != nil {
		return false, false, fmt.Errorf("helius getAccountInfo error: %s", r.Error.Message)
	}
	info := r.Result.Value.Data.Parsed.Info
	return info.MintAuthority != nil, info.FreezeAuthority != nil, nil
}
