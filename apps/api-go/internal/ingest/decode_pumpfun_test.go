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
