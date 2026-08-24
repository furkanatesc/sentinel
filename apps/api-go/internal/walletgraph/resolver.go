// Package walletgraph, creator-funding kümeleme (bundler tespiti) için funder yakalama +
// graph kurma sağlar. Funder = bir cüzdana ilk SOL gönderen (en eski inbound transfer).
package walletgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Limiter interface{ Wait(ctx context.Context) error }

type FunderResolver interface {
	ResolveFunder(ctx context.Context, wallet string) (funder string, found bool, err error)
}

// funderSigTx, resolver'ın RPC ihtiyacını soyutlar (DIP; test için fake).
type funderSigTx interface {
	listSignatures(ctx context.Context, acct, before string, limit int) ([]string, error)
	// transferSource, sig'in tx'inde destination=wallet olan İLK system transfer'ının source'unu döndürür.
	transferSource(ctx context.Context, sig, wallet string) (source string, found bool, err error)
}

type HeliusFunderResolver struct {
	rpc         funderSigTx
	maxSigPages int
	pageLimit   int
	limiter     Limiter
}

type ResolverOption func(*HeliusFunderResolver)

func WithLimiter(l Limiter) ResolverOption {
	return func(r *HeliusFunderResolver) { r.limiter = l }
}

func NewFunderResolver(rpcURL string, maxSigPages int, opts ...ResolverOption) *HeliusFunderResolver {
	if maxSigPages <= 0 {
		maxSigPages = 3
	}
	r := &HeliusFunderResolver{rpc: &httpSigTx{rpcURL: rpcURL, http: &http.Client{Timeout: 12 * time.Second}}, maxSigPages: maxSigPages, pageLimit: 1000}
	for _, o := range opts {
		o(r)
	}
	return r
}

// ResolveFunder, wallet'ın EN ESKİ imzasını bulur, o tx'te wallet'a gelen ilk SOL transfer'ının
// kaynağını (funder) döndürür. Transfer yoksa / cap'e takılırsa found=false (dürüst not-found).
func (r *HeliusFunderResolver) ResolveFunder(ctx context.Context, wallet string) (string, bool, error) {
	before := ""
	oldest := ""
	for page := 0; page < r.maxSigPages; page++ {
		if r.limiter != nil {
			if err := r.limiter.Wait(ctx); err != nil {
				return "", false, err
			}
		}
		sigs, err := r.rpc.listSignatures(ctx, wallet, before, r.pageLimit)
		if err != nil {
			return "", false, err
		}
		if len(sigs) == 0 {
			break
		}
		oldest = sigs[len(sigs)-1] // newest-first → son = en eski
		before = oldest
		if len(sigs) < r.pageLimit {
			break
		}
	}
	if oldest == "" {
		return "", false, nil
	}
	if r.limiter != nil {
		if err := r.limiter.Wait(ctx); err != nil {
			return "", false, err
		}
	}
	return r.rpc.transferSource(ctx, oldest, wallet)
}

// --- httpSigTx: raw JSON-RPC (authorities.go deseni) ---
type httpSigTx struct {
	rpcURL string
	http   *http.Client
}

func (h *httpSigTx) listSignatures(ctx context.Context, acct, before string, limit int) ([]string, error) {
	params := []any{acct, map[string]any{"limit": limit}}
	if before != "" {
		params[1].(map[string]any)["before"] = before
	}
	var r struct {
		Result []struct {
			Signature string `json:"signature"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := h.call(ctx, "getSignaturesForAddress", params, &r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, fmt.Errorf("getSignaturesForAddress: %s", r.Error.Message)
	}
	out := make([]string, 0, len(r.Result))
	for _, s := range r.Result {
		out = append(out, s.Signature)
	}
	return out, nil
}

func (h *httpSigTx) transferSource(ctx context.Context, sig, wallet string) (string, bool, error) {
	params := []any{sig, map[string]any{"encoding": "jsonParsed", "maxSupportedTransactionVersion": 0}}
	var r struct {
		Result *struct {
			Transaction struct {
				Message struct {
					Instructions []parsedIx `json:"instructions"`
				} `json:"message"`
			} `json:"transaction"`
			Meta *struct {
				InnerInstructions []struct {
					Instructions []parsedIx `json:"instructions"`
				} `json:"innerInstructions"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := h.call(ctx, "getTransaction", params, &r); err != nil {
		return "", false, err
	}
	if r.Error != nil {
		return "", false, fmt.Errorf("getTransaction: %s", r.Error.Message)
	}
	if r.Result == nil {
		return "", false, nil
	}
	if src, ok := scanTransfers(r.Result.Transaction.Message.Instructions, wallet); ok {
		return src, true, nil
	}
	if r.Result.Meta != nil {
		for _, inner := range r.Result.Meta.InnerInstructions {
			if src, ok := scanTransfers(inner.Instructions, wallet); ok {
				return src, true, nil
			}
		}
	}
	return "", false, nil
}

// parsedIx, jsonParsed system transfer instruction'ının gereken alanları.
type parsedIx struct {
	Program string `json:"program"`
	Parsed  struct {
		Type string `json:"type"`
		Info struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		} `json:"info"`
	} `json:"parsed"`
}

// scanTransfers, destination==wallet olan ilk system transfer'ın source'unu döndürür.
func scanTransfers(ixs []parsedIx, wallet string) (string, bool) {
	for _, ix := range ixs {
		if ix.Program != "system" {
			continue
		}
		if ix.Parsed.Type != "transfer" && ix.Parsed.Type != "transferChecked" {
			continue
		}
		if ix.Parsed.Info.Destination == wallet && ix.Parsed.Info.Source != "" && ix.Parsed.Info.Source != wallet {
			return ix.Parsed.Info.Source, true
		}
	}
	return "", false
}

func (h *httpSigTx) call(ctx context.Context, method string, params any, out any) error {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "1", "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: status %d", method, res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

var _ FunderResolver = (*HeliusFunderResolver)(nil)
