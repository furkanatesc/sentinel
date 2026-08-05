package ingest

import (
	"context"
	"testing"
)

type fakeTx struct{ keys []string }

func (f fakeTx) GetTransaction(_ context.Context, _ string) (TxInfo, error) {
	return TxInfo{AccountKeys: f.keys}, nil
}

type fakeMeta struct {
	name, symbol string
	err          error
}

func (f fakeMeta) Metadata(_ context.Context, _ string) (TokenMeta, error) {
	if f.err != nil {
		return TokenMeta{}, f.err
	}
	return TokenMeta{Name: f.name, Symbol: f.symbol}, nil
}

const wsol = "So11111111111111111111111111111111111111112"

func TestRaydiumCpmmDecode(t *testing.T) {
	n := LogNotification{
		Signature: "rsig", Slot: 7, ProgramID: RaydiumCpmmProgramID,
		Logs: []string{
			"Program " + RaydiumCpmmProgramID + " invoke [1]",
			"Program log: Instruction: Initialize",
			"Program " + RaydiumCpmmProgramID + " success",
		},
	}
	tx := fakeTx{keys: []string{"someProgram", wsol, "NewMintPubkey11111111111111111111111111111", "vault"}}
	md := fakeMeta{name: "New Coin", symbol: "NEW"}
	out, err := NewRaydiumCpmmDecoder().Decode(context.Background(), n, tx, md)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Event.Type != "pool_created" {
		t.Fatalf("out = %+v", out)
	}
	if out[0].Event.Mint != "NewMintPubkey11111111111111111111111111111" {
		t.Fatalf("mint = %s", out[0].Event.Mint)
	}
	if out[0].Event.Launchpad != "Raydium" || out[0].Token.Symbol != "NEW" {
		t.Fatalf("event/token = %+v / %+v", out[0].Event, out[0].Token)
	}
}

func TestRaydiumFallbackSymbolOnMetaError(t *testing.T) {
	n := LogNotification{Signature: "s", ProgramID: RaydiumCpmmProgramID,
		Logs: []string{"Program log: Instruction: Initialize"}}
	tx := fakeTx{keys: []string{"p", wsol, "MintABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890xy"}}
	md := fakeMeta{err: context.DeadlineExceeded}
	out, err := NewRaydiumCpmmDecoder().Decode(context.Background(), n, tx, md)
	if err != nil || len(out) != 1 {
		t.Fatalf("fallback: out=%+v err=%v", out, err)
	}
	if out[0].Token.Symbol == "" { // kısaltılmış mint fallback
		t.Fatal("metadata hatasında symbol fallback boş olmamalı")
	}
}

func TestRaydiumIgnoresNonInitialize(t *testing.T) {
	n := LogNotification{ProgramID: RaydiumCpmmProgramID, Logs: []string{"Program log: Instruction: Swap"}}
	out, err := NewRaydiumCpmmDecoder().Decode(context.Background(), n, fakeTx{}, fakeMeta{})
	if err != nil || len(out) != 0 {
		t.Fatalf("initialize olmayan: %d olay, err=%v", len(out), err)
	}
}
