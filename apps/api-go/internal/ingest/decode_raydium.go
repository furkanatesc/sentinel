package ingest

import (
	"context"
	"strings"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

const RaydiumCpmmProgramID = "CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C"
const wsolMint = "So11111111111111111111111111111111111111112"

type raydiumCpmmDecoder struct{}

func NewRaydiumCpmmDecoder() LaunchpadDecoder { return raydiumCpmmDecoder{} }

func (raydiumCpmmDecoder) ProgramID() string { return RaydiumCpmmProgramID }
func (raydiumCpmmDecoder) Launchpad() string { return "Raydium" }

func (d raydiumCpmmDecoder) Decode(ctx context.Context, n LogNotification, tx TxFetcher, md MetadataFetcher) ([]Decoded, error) {
	if n.Err != nil {
		return nil, nil
	}
	if !hasMarkerFold(n.Logs, "instruction: initialize") {
		return nil, nil
	}
	if tx == nil {
		return nil, nil
	}
	info, err := tx.GetTransaction(ctx, n.Signature)
	if err != nil {
		return nil, nil // tx alınamadı → drop (pipeline durmaz; worker loglar)
	}
	mint := firstNonWSOLMint(info.AccountKeys)
	if mint == "" {
		return nil, nil
	}
	name, symbol := shortMint(mint), shortMint(mint) // fallback
	if md != nil {
		if meta, err := md.Metadata(ctx, mint); err == nil {
			if meta.Name != "" {
				name = meta.Name
			}
			if meta.Symbol != "" {
				symbol = meta.Symbol
			}
		}
	}
	token := store.TokenRow{ID: mint, Mint: mint, Symbol: symbol, Name: name, Spark: []float64{}}
	ev := store.EventRow{
		ID: n.Signature + "-pool_created", Signature: n.Signature, Slot: n.Slot, Type: "pool_created",
		Mint: mint, Symbol: symbol, Launchpad: "Raydium", DEX: "Raydium",
		RiskLevel: "medium", Severity: "positive", CreatorScore: 0,
		Detail: "Yeni likidite havuzu (Raydium CPMM)",
	}
	return []Decoded{{Event: ev, Token: token}}, nil
}

func firstNonWSOLMint(keys []string) string {
	// 1a heuristiği: WSOL olmayan, base58-uzunluğunda ilk hesap. Kesin index deploy'da doğrulanır.
	for _, k := range keys {
		if k == wsolMint {
			continue
		}
		if len(k) >= 32 && len(k) <= 44 { // base58 pubkey aralığı
			return k
		}
	}
	return ""
}

func shortMint(m string) string {
	if len(m) <= 8 {
		return m
	}
	return m[:4] + "…" + m[len(m)-4:]
}

func hasMarkerFold(logs []string, lowerMarker string) bool {
	for _, l := range logs {
		if strings.Contains(strings.ToLower(l), lowerMarker) {
			return true
		}
	}
	return false
}
