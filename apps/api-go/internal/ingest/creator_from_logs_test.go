package ingest

import (
	"encoding/base64"
	"testing"

	"github.com/mr-tron/base58"
)

func TestCreatorFromCreateLogs(t *testing.T) {
	// 2b-1 buildCreateEventB64 ile aynı layout: name/symbol/uri + mint + bonding + user(creator).
	var mint, bonding, user [32]byte
	mint[0] = 1
	user[0], user[31] = 7, 3
	data := buildCreateEventB64("Doge", "DOGE", "https://x/u.json", mint, bonding, user)
	logs := []string{
		"Program log: Instruction: Create",
		"Program data: " + data,
		"Program " + PumpFunProgramID + " success",
	}
	creator, ok := CreatorFromCreateLogs(logs)
	if !ok || creator != base58.Encode(user[:]) {
		t.Fatalf("creator = %q ok=%v, want %q", creator, ok, base58.Encode(user[:]))
	}
}

func TestCreatorFromCreateLogsNoCreate(t *testing.T) {
	// Create instruction yok → ok=false.
	if _, ok := CreatorFromCreateLogs([]string{"Program log: Instruction: Buy"}); ok {
		t.Fatal("create yokken ok=true olmamalı")
	}
	// Create var ama Program data çok kısa/bozuk → ok=false.
	bad := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
	if _, ok := CreatorFromCreateLogs([]string{"Program log: Instruction: Create", "Program data: " + bad}); ok {
		t.Fatal("bozuk data'da ok=true olmamalı")
	}
}
