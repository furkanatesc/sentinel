package ingest

import "testing"

func TestHeliusURLs(t *testing.T) {
	ws, rpc := HeliusURLs("KEY123")
	if ws != "wss://mainnet.helius-rpc.com/?api-key=KEY123" {
		t.Fatalf("ws=%s", ws)
	}
	if rpc != "https://mainnet.helius-rpc.com/?api-key=KEY123" {
		t.Fatalf("rpc=%s", rpc)
	}
}

func TestParseGetAssetResponse(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":"1","result":{"content":{"json_uri":"https://x/j.json","metadata":{"name":"Foo","symbol":"FOO"}}}}`)
	m, err := parseGetAsset(body)
	if err != nil || m.Name != "Foo" || m.Symbol != "FOO" || m.URI != "https://x/j.json" {
		t.Fatalf("m=%+v err=%v", m, err)
	}
}
