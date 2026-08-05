package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func HeliusURLs(apiKey string) (wsURL, rpcURL string) {
	return "wss://mainnet.helius-rpc.com/?api-key=" + apiKey,
		"https://mainnet.helius-rpc.com/?api-key=" + apiKey
}

// --- DAS getAsset (MetadataFetcher) ---
type heliusMetadata struct {
	rpcURL string
	http   *http.Client
}

func NewHeliusMetadata(rpcURL string) MetadataFetcher {
	return &heliusMetadata{rpcURL: rpcURL, http: &http.Client{Timeout: 8 * time.Second}}
}

func (h *heliusMetadata) Metadata(ctx context.Context, mint string) (TokenMeta, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "getAsset",
		"params": map[string]any{"id": mint},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return TokenMeta{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http.Do(req)
	if err != nil {
		return TokenMeta{}, err
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return TokenMeta{}, err
	}
	return parseGetAsset(buf.Bytes())
}

func parseGetAsset(body []byte) (TokenMeta, error) {
	var r struct {
		Result struct {
			Content struct {
				JSONURI  string `json:"json_uri"`
				Metadata struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				} `json:"metadata"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return TokenMeta{}, err
	}
	if r.Error != nil {
		return TokenMeta{}, fmt.Errorf("helius getAsset error %d: %s", r.Error.Code, r.Error.Message)
	}
	return TokenMeta{Name: r.Result.Content.Metadata.Name, Symbol: r.Result.Content.Metadata.Symbol, URI: r.Result.Content.JSONURI}, nil
}

// --- getTransaction (TxFetcher) ---
type heliusTx struct{ cli *rpc.Client }

func NewHeliusTx(rpcURL string) TxFetcher { return &heliusTx{cli: rpc.New(rpcURL)} }

func (h *heliusTx) GetTransaction(ctx context.Context, signature string) (TxInfo, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return TxInfo{}, err
	}
	maxV := uint64(0)
	out, err := h.cli.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		MaxSupportedTransactionVersion: &maxV,
	})
	if err != nil {
		return TxInfo{}, err
	}
	tx, err := out.Transaction.GetTransaction()
	if err != nil {
		return TxInfo{}, err
	}
	keys := make([]string, 0, len(tx.Message.AccountKeys))
	for _, k := range tx.Message.AccountKeys {
		keys = append(keys, k.String())
	}
	return TxInfo{AccountKeys: keys}, nil
}

// --- WS logsSubscribe (worker kullanır) ---
// SubscribeLogs, her programID için ayrı abonelik açar (mentions tek pubkey alır),
// bildirimleri normalize edip out kanalına yazar. ctx iptalinde döner. Reconnect worker'da.
func SubscribeLogs(ctx context.Context, wsURL string, programIDs []string, out chan<- LogNotification) error {
	client, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return fmt.Errorf("ws connect: %w", err)
	}
	defer client.Close()

	for _, pid := range programIDs {
		pk, err := solana.PublicKeyFromBase58(pid)
		if err != nil {
			return fmt.Errorf("bad programID %s: %w", pid, err)
		}
		sub, err := client.LogsSubscribeMentions(pk, rpc.CommitmentProcessed)
		if err != nil {
			return fmt.Errorf("logsSubscribe %s: %w", pid, err)
		}
		go recvLoop(ctx, sub, pid, out)
	}
	<-ctx.Done()
	return ctx.Err()
}

func recvLoop(ctx context.Context, sub *ws.LogSubscription, programID string, out chan<- LogNotification) {
	defer sub.Unsubscribe()
	for {
		got, err := sub.Recv(ctx)
		if err != nil {
			return // bağlantı koptu → worker reconnect eder
		}
		if got == nil {
			continue
		}
		v := got.Value
		n := LogNotification{
			Signature: v.Signature.String(),
			Slot:      got.Context.Slot,
			Err:       v.Err,
			Logs:      v.Logs,
			ProgramID: programID,
		}
		select {
		case out <- n:
		case <-ctx.Done():
			return
		}
	}
}
