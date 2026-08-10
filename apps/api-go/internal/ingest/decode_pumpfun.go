package ingest

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"

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
	if !hasCreateInstruction(n.Logs) {
		return nil, nil
	}
	// Bir create tx'inde birden çok "Program data:" satırı olabilir (create+dev-buy → CreateEvent
	// sonra TradeEvent). CreateEvent'i tanıyan ilk satırı seç; tanınmazsa dürüst hata (teşhis).
	raws := programDataAll(n.Logs)
	if len(raws) == 0 {
		return nil, nil // create var ama event data yok — atla (bozuk pipeline değil)
	}
	var ev createEvent
	found := false
	for _, raw := range raws {
		if e, ok := parseCreateEvent(raw); ok {
			ev, found = e, true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("pumpfun CreateEvent tanınamadı (%s)", diagPD(raws))
	}
	token := store.TokenRow{
		ID: ev.mint, Mint: ev.mint, Symbol: ev.symbol, Name: ev.name,
		Spark: []float64{}, CreatorScore: 0, SafetyScore: 0, Momentum: 0,
	}
	mkEvent := func(id, typ, sev, detail string) store.EventRow {
		return store.EventRow{
			ID: n.Signature + "-" + id, Signature: n.Signature, Slot: n.Slot, Type: typ,
			Mint: ev.mint, Symbol: ev.symbol, Launchpad: "Pump.fun", DEX: "",
			RiskLevel: "medium", Severity: sev, CreatorScore: 0, Detail: detail, Ts: 0, // Ts worker damgalar
		}
	}
	return []Decoded{
		{Event: mkEvent("new_mint", "new_mint", "info", "Yeni token oluşturuldu (pump.fun)"), Token: token, Creator: ev.creator},
		{Event: mkEvent("metadata_created", "metadata_created", "info", ev.name+" ("+ev.symbol+")"), Token: token, Creator: ev.creator},
	}, nil
}

type createEvent struct{ name, symbol, uri, mint, creator string }

// parseCreateEvent, bir "Program data:" borsh yükünden pump.fun CreateEvent'i (name/symbol/uri/mint)
// çıkarır. Anchor event discriminator öneki emit! (8B) vs emit_cpi! (16B) arasında değişebildiğinden
// offset'i OTOMATİK tespit eder: 3 length-prefixli geçerli UTF-8 string + uri içinde "://" +
// ardından ≥96 bayt (mint/bondingCurve/user pubkey'leri). Bu doğrulama, yanlış satırı/offset'i eler.
func parseCreateEvent(raw []byte) (createEvent, bool) {
	for _, off := range candidateOffsets(len(raw)) {
		if ev, ok := tryParseCreateAt(raw, off); ok {
			return ev, true
		}
	}
	return createEvent{}, false
}

func candidateOffsets(n int) []int {
	// Önce yaygın anchor önekleri (emit!=8, emit_cpi!=16), sonra sınırlı bir tarama (garanti).
	offs := []int{8, 16}
	for o := 0; o <= 40 && o+4 <= n; o++ {
		if o != 8 && o != 16 {
			offs = append(offs, o)
		}
	}
	return offs
}

func tryParseCreateAt(raw []byte, off int) (createEvent, bool) {
	p := off
	rd := func(maxLen int) (string, bool) {
		if p+4 > len(raw) {
			return "", false
		}
		ln := int(binary.LittleEndian.Uint32(raw[p : p+4]))
		p += 4
		if ln <= 0 || ln > maxLen || p+ln > len(raw) {
			return "", false
		}
		b := raw[p : p+ln]
		if !utf8.Valid(b) {
			return "", false
		}
		p += ln
		return string(b), true
	}
	name, ok := rd(100)
	if !ok {
		return createEvent{}, false
	}
	symbol, ok := rd(50)
	if !ok {
		return createEvent{}, false
	}
	uri, ok := rd(400)
	if !ok {
		return createEvent{}, false
	}
	if !strings.Contains(uri, "://") { // pump.fun metadata uri'si her zaman bir URL'dir
		return createEvent{}, false
	}
	if p+32 > len(raw) { // en az mint pubkey'i sığmalı
		return createEvent{}, false
	}
	mint := base58.Encode(raw[p : p+32])
	// mint(32) + bondingCurve(32) sonrası user/creator(32) — yer varsa oku, yoksa dürüst boş.
	creator := ""
	if cp := p + 64; cp+32 <= len(raw) {
		creator = base58.Encode(raw[cp : cp+32])
	}
	return createEvent{name: name, symbol: symbol, uri: uri, mint: mint, creator: creator}, true
}

func hasCreateInstruction(logs []string) bool {
	for _, l := range logs {
		// Trimmed suffix eşleşmesi: "...Instruction: Create" TRUE, "...Instruction: CreateIdempotent" FALSE.
		if strings.HasSuffix(strings.TrimSpace(l), "Instruction: Create") {
			return true
		}
	}
	return false
}

func programDataAll(logs []string) [][]byte {
	const pfx = "Program data: "
	var out [][]byte
	for _, l := range logs {
		if strings.HasPrefix(l, pfx) {
			if b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(l, pfx)); err == nil {
				out = append(out, b)
			}
		}
	}
	return out
}

// diagPD, parse edilemeyen durumda teşhis için ilk birkaç Program data satırının uzunluk+baş baytlarını özetler.
func diagPD(raws [][]byte) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d pdata satırı", len(raws))
	for i, raw := range raws {
		if i >= 3 {
			break
		}
		head := raw
		if len(head) > 24 {
			head = head[:24]
		}
		fmt.Fprintf(&sb, "; [%d] len=%d head=%x", i, len(raw), head)
	}
	return sb.String()
}
