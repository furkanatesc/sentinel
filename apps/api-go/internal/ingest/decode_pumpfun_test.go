package ingest

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"testing"
)

// buildCreateEventB64, pump.fun CreateEvent base64 satırını decoder layout'uyla üretir (test vektörü).
func buildCreateEventB64(name, symbol, uri string, mint, bonding, user [32]byte) string {
	var b []byte
	b = append(b, make([]byte, 8)...) // discriminator (test: sıfır)
	putStr := func(s string) {
		var n [4]byte
		binary.LittleEndian.PutUint32(n[:], uint32(len(s)))
		b = append(b, n[:]...)
		b = append(b, []byte(s)...)
	}
	putStr(name)
	putStr(symbol)
	putStr(uri)
	b = append(b, mint[:]...)
	b = append(b, bonding[:]...)
	b = append(b, user[:]...)
	return base64.StdEncoding.EncodeToString(b)
}

func TestPumpFunDecode(t *testing.T) {
	var mint, bonding, user [32]byte
	mint[0], mint[31] = 1, 9 // ayırt edici baytlar
	data := buildCreateEventB64("Doge Killer", "DOGEK", "https://x/uri.json", mint, bonding, user)

	n := LogNotification{
		Signature: "sig123", Slot: 42, ProgramID: PumpFunProgramID,
		Logs: []string{
			"Program " + PumpFunProgramID + " invoke [1]",
			"Program log: Instruction: Create",
			"Program data: " + data,
			"Program " + PumpFunProgramID + " success",
		},
	}
	d := NewPumpFunDecoder()
	out, err := d.Decode(context.Background(), n, nil, nil) // pump.fun tx/md kullanmaz
	if err != nil {
		t.Fatal(err)
	}
	// new_mint + metadata_created bekleniyor
	if len(out) != 2 {
		t.Fatalf("Decoded len=%d, want 2", len(out))
	}
	e0 := out[0].Event
	if e0.Type != "new_mint" || e0.Symbol != "DOGEK" || e0.Launchpad != "Pump.fun" {
		t.Fatalf("event0 = %+v", e0)
	}
	if e0.RiskLevel != "medium" || e0.CreatorScore != 0 {
		t.Fatalf("nötr placeholder bozuk: %+v", e0)
	}
	if out[0].Token.Name != "Doge Killer" || out[0].Token.Mint == "" {
		t.Fatalf("token = %+v", out[0].Token)
	}
	if out[1].Event.Type != "metadata_created" {
		t.Fatalf("event1 type = %s", out[1].Event.Type)
	}
}

func TestPumpFunIgnoresNonCreate(t *testing.T) {
	n := LogNotification{ProgramID: PumpFunProgramID, Logs: []string{"Program log: Instruction: Buy"}}
	out, err := NewPumpFunDecoder().Decode(context.Background(), n, nil, nil)
	if err != nil || len(out) != 0 {
		t.Fatalf("create olmayan log 0 olay vermeli; got %d, err=%v", len(out), err)
	}
}

// buildCreateEventWithPrefix, verilen boyutta bir discriminator öneğiyle CreateEvent üretir
// (emit! = 8, emit_cpi! = 16 baytlık öneki simüle etmek için).
func buildCreateEventWithPrefix(prefixLen int, name, symbol, uri string, mint [32]byte) string {
	var b []byte
	b = append(b, make([]byte, prefixLen)...)
	putStr := func(s string) {
		var n [4]byte
		binary.LittleEndian.PutUint32(n[:], uint32(len(s)))
		b = append(b, n[:]...)
		b = append(b, []byte(s)...)
	}
	putStr(name)
	putStr(symbol)
	putStr(uri)
	b = append(b, mint[:]...)         // mint
	b = append(b, make([]byte, 32)...) // bondingCurve
	b = append(b, make([]byte, 32)...) // user
	return base64.StdEncoding.EncodeToString(b)
}

// Gerçek pump.fun emit_cpi! olayları 16 baytlık önek taşır; decoder offset'i otomatik tespit etmeli.
func TestPumpFunDecodeEmitCpi16BytePrefix(t *testing.T) {
	var mint [32]byte
	mint[0], mint[31] = 7, 3
	data := buildCreateEventWithPrefix(16, "Solana Cat", "SCAT", "https://ipfs.io/ipfs/abc", mint)
	n := LogNotification{
		Signature: "sig16", ProgramID: PumpFunProgramID,
		Logs: []string{"Program log: Instruction: Create", "Program data: " + data},
	}
	out, err := NewPumpFunDecoder().Decode(context.Background(), n, nil, nil)
	if err != nil {
		t.Fatalf("16-byte önek parse edilemedi: %v", err)
	}
	if len(out) != 2 || out[0].Token.Symbol != "SCAT" || out[0].Token.Name != "Solana Cat" {
		t.Fatalf("emit_cpi decode yanlış: %+v", out)
	}
}

// Bir pump.fun BUY işlemi alıcının ATA'sını "Instruction: CreateIdempotent" ile oluşturur;
// bu, "Instruction: Create" substring'ini içerse de bir create DEĞİLDİR ve decode edilmemelidir.
func TestPumpFunIgnoresCreateIdempotent(t *testing.T) {
	// Buy'ın Program data'sı bir TradeEvent'tir (create layout'una uymaz).
	trade := base64.StdEncoding.EncodeToString(append(make([]byte, 8), make([]byte, 120)...))
	n := LogNotification{
		Signature: "buySig", ProgramID: PumpFunProgramID,
		Logs: []string{
			"Program log: Instruction: Buy",
			"Program log: Instruction: CreateIdempotent",
			"Program data: " + trade,
		},
	}
	out, err := NewPumpFunDecoder().Decode(context.Background(), n, nil, nil)
	if err != nil || len(out) != 0 {
		t.Fatalf("CreateIdempotent (buy ATA) create sanılmamalı; got %d olay, err=%v", len(out), err)
	}
}

// Create+dev-buy bundle'ında hem CreateEvent hem TradeEvent Program data satırı olur;
// decoder TradeEvent'i eleyip CreateEvent'i seçmeli.
func TestPumpFunPicksCreateAmongMultiplePDLines(t *testing.T) {
	var mint [32]byte
	mint[5] = 42
	trade := base64.StdEncoding.EncodeToString(append(make([]byte, 8), make([]byte, 120)...)) // TradeEvent-benzeri
	create := buildCreateEventWithPrefix(8, "Multi", "MLT", "https://x/m.json", mint)
	n := LogNotification{
		Signature: "multiSig", ProgramID: PumpFunProgramID,
		Logs: []string{
			"Program log: Instruction: Create",
			"Program data: " + trade,  // önce (yanlış) TradeEvent
			"Program data: " + create, // sonra gerçek CreateEvent
		},
	}
	out, err := NewPumpFunDecoder().Decode(context.Background(), n, nil, nil)
	if err != nil || len(out) != 2 || out[0].Token.Symbol != "MLT" {
		t.Fatalf("çoklu pdata'dan CreateEvent seçilmeli; got %d err=%v out=%+v", len(out), err, out)
	}
}
