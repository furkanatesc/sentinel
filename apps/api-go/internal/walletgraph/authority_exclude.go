package walletgraph

// knownProgramAuthority, mint/freeze authority olarak görünse de "kontrol kümesi" OLMAYAN
// program/sistem adresleridir (paylaşılan launchpad/PDA authority → sahte god-node → dışlanır;
// 2e-1 CEX allowlist analoğu). Adresler web-doğrulandı (Solscan public label); derece tavanı
// (WalletGraphMaxDegree) bilinmeyen mega-hub'lara ek emniyet.
var knownProgramAuthority = map[string]string{
	"TSLvdd1pWpHVjahSpsvCXUbgwsL3JAcvokwaKt1eokM": "Pump.fun Token Mint Authority", // Solscan label; on-curve pump.fun tokenlerinde paylaşılan mint authority → en büyük sahte-hub kaynağı
	"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA": "SPL Token Program",
	"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb": "Token-2022 Program",
	"11111111111111111111111111111111":            "System Program",
}

// IsProgramAuthority, addr bilinen-program/sistem authority'si mi (dışlanmalı) döner.
func IsProgramAuthority(addr string) bool { _, ok := knownProgramAuthority[addr]; return ok }
