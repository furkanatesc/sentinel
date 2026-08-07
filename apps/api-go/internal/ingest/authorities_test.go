package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func authServer(t *testing.T, body string) *HeliusAuthorities {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewHeliusAuthorities(srv.URL)
}

func TestMintAuthoritiesBothRevoked(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":"1","result":{"value":{"data":{"parsed":{"info":{"mintAuthority":null,"freezeAuthority":null}}}}}}`
	h := authServer(t, body)
	mintA, freezeA, err := h.MintAuthorities(context.Background(), "MintX")
	if err != nil {
		t.Fatal(err)
	}
	if mintA || freezeA {
		t.Fatalf("null authority → aktif değil: mint=%v freeze=%v", mintA, freezeA)
	}
}

func TestMintAuthoritiesBothActive(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":"1","result":{"value":{"data":{"parsed":{"info":{"mintAuthority":"Abc111","freezeAuthority":"Def222"}}}}}}`
	h := authServer(t, body)
	mintA, freezeA, err := h.MintAuthorities(context.Background(), "MintX")
	if err != nil {
		t.Fatal(err)
	}
	if !mintA || !freezeA {
		t.Fatalf("dolu authority → aktif: mint=%v freeze=%v", mintA, freezeA)
	}
}

func TestMintAuthoritiesRPCError(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":"1","error":{"message":"boom"}}`
	h := authServer(t, body)
	if _, _, err := h.MintAuthorities(context.Background(), "MintX"); err == nil {
		t.Fatal("RPC error hata dönmeli")
	}
}
