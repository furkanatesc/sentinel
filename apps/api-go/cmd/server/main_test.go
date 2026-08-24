package main

import "testing"

// preferRPC: override boş değilse override'ı, boşsa fallback'i döndürür.
// creatorfill ve safety-authorities aynı "Helius free-tier'ı SOLANA_RPC_URL ile
// baypas et" desenini paylaşır (DRY) — bu saf seçim mantığı tek yerde test edilir.
func TestPreferRPC(t *testing.T) {
	tests := []struct {
		name     string
		override string
		fallback string
		want     string
	}{
		{"override set → override", "https://alt-rpc.tld/?key=abc", "https://helius/?api-key=x", "https://alt-rpc.tld/?key=abc"},
		{"override boş → fallback", "", "https://helius/?api-key=x", "https://helius/?api-key=x"},
		{"ikisi de boş → boş", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := preferRPC(tt.override, tt.fallback); got != tt.want {
				t.Fatalf("preferRPC(%q, %q) = %q, want %q", tt.override, tt.fallback, got, tt.want)
			}
		})
	}
}
