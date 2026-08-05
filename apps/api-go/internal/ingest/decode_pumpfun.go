package ingest

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
	"github.com/mr-tron/base58" // base58 encode (pubkey). go get github.com/mr-tron/base58
)

const PumpFunProgramID = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"

type pumpFunDecoder struct{}

func NewPumpFunDecoder() LaunchpadDecoder { return pumpFunDecoder{} }

func (pumpFunDecoder) ProgramID() string { return PumpFunProgramID }
func (pumpFunDecoder) Launchpad() string { return "Pump.fun" }

func (d pumpFunDecoder) Decode(_ context.Context, n LogNotification, _ TxFetcher, _ MetadataFetcher) ([]Decoded, error) {
	if n.Err != nil {
		return nil, nil
	}
	if !hasMarker(n.Logs, "Instruction: Create") {
		return nil, nil
	}
	raw, ok := programDataB64(n.Logs)
	if !ok {
		return nil, nil // create var ama event data yok — atla (bozuk pipeline değil)
	}
	ev, err := parseCreateEvent(raw)
	if err != nil {
		return nil, fmt.Errorf("pumpfun createEvent parse: %w", err)
	}
	ts := int64(0) // worker gerçek zamanı damgalar
	token := store.TokenRow{
		ID: ev.mint, Mint: ev.mint, Symbol: ev.symbol, Name: ev.name,
		Spark: []float64{}, CreatorScore: 0, SafetyScore: 0, Momentum: 0,
	}
	mkEvent := func(id, typ, sev, detail string) store.EventRow {
		return store.EventRow{
			ID: n.Signature + "-" + id, Signature: n.Signature, Slot: n.Slot, Type: typ,
			Mint: ev.mint, Symbol: ev.symbol, Launchpad: "Pump.fun", DEX: "",
			RiskLevel: "medium", Severity: sev, CreatorScore: 0, Detail: detail, Ts: ts,
		}
	}
	return []Decoded{
		{Event: mkEvent("new_mint", "new_mint", "info", "Yeni token oluşturuldu (pump.fun)"), Token: token},
		{Event: mkEvent("metadata_created", "metadata_created", "info", ev.name+" ("+ev.symbol+")"), Token: token},
	}, nil
}

type createEvent struct{ name, symbol, uri, mint string }

func parseCreateEvent(raw []byte) (createEvent, error) {
	p := 8 // discriminator atla
	rd := func() (string, error) {
		if p+4 > len(raw) {
			return "", errors.New("kısa buffer (len prefix)")
		}
		ln := int(binary.LittleEndian.Uint32(raw[p : p+4]))
		p += 4
		if p+ln > len(raw) {
			return "", errors.New("kısa buffer (string body)")
		}
		s := string(raw[p : p+ln])
		p += ln
		return s, nil
	}
	name, err := rd()
	if err != nil {
		return createEvent{}, err
	}
	symbol, err := rd()
	if err != nil {
		return createEvent{}, err
	}
	uri, err := rd()
	if err != nil {
		return createEvent{}, err
	}
	if p+32 > len(raw) {
		return createEvent{}, errors.New("kısa buffer (mint)")
	}
	mint := base58.Encode(raw[p : p+32])
	return createEvent{name: name, symbol: symbol, uri: uri, mint: mint}, nil
}

func hasMarker(logs []string, marker string) bool {
	for _, l := range logs {
		if strings.Contains(l, marker) {
			return true
		}
	}
	return false
}

func programDataB64(logs []string) ([]byte, bool) {
	const pfx = "Program data: "
	for _, l := range logs {
		if strings.HasPrefix(l, pfx) {
			b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(l, pfx))
			if err == nil {
				return b, true
			}
		}
	}
	return nil, false
}
