package walletgraph

// knownCEX, bilinen büyük Solana CEX hot-wallet'larıdır (allowlist; bunları fonlayan "küme"
// bundler DEĞİL, borsa çekimidir → dışlanır). Risk asimetriktir: EKSİK adres = false-positive
// (borsa çekimini bundler sanma, kötü); YANLIŞ adres ≈ zararsız (gerçek bir bundler'a denk gelmesi
// pratikte imkânsız). Bu yüzden güçlü-teyitli adresleri EKLEMEYE meyillidir.
//
// Doğrulama: 2026-08-24 ilk tur; 2026-08-26 re-doğrulama + genişletme (Solscan public label, arama
// başlığında görünen resmi etiket üzerinden). Etiketi çelişkili görünen bir Coinbase/Kraken adayı
// (H8sMJSCQ...) belirsizlik nedeniyle BİLİNÇLİ OLARAK DIŞLANDI (bazı kaynaklar "Kraken", bazıları
// "Coinbase" gösteriyor). Bybit adresi (AC5R...) ilk turdan; 2026-08-26 re-doğrulamada Solscan
// public-label ile TEYİT EDİLEMEDİ (Bybit'in labeled Solana hot-wallet'ı belgeli değil) → asimetri
// gereği yine de tutuluyor (yanlışsa zararsız), ama en zayıf kayıt.
var knownCEX = map[string]string{
	// 2026-08-26 re-doğrulamada Solscan public-label ile teyitli:
	"5tzFkiKscXHK5ZXCGbXZxdw7gTjjD1mBwuoFbhUvuAi9": "Binance",   // "Binance 2"
	"9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM": "Binance",   // "Binance Hot Wallet 1"
	"GJRs4FwHtemZ5ZE9x3FNvJ8TMwitKTh21yxdRPqn7npE": "Coinbase",  // "Coinbase Hot Wallet 2"
	"D89hHJT5Aqyx1trP6EnGY9jJUB3whgnq3aUvvCqedvzf": "Coinbase",  // "Coinbase Hot Wallet 3"
	"C68a6RCGLiPskbPYtAcsCjhG8tfTWYcoB4JjCrXFdqyo": "OKX",       // "OKX: Hot Wallet"
	"u6PJ8DtQuPFnfmwHbGFULQ4u4EgjDiyYKjVEsynXq2w":  "Gate.io",   // "Gate.io"
	// İlk turdan; re-doğrulamada public-label snippet'i yüzeye çıkmadı (yaygın atıflı, çelişki yok):
	"FWznbcNXWQuHTawe9RxvQ2LdCENssh12dsznf4RiouN5": "Kraken",
	// İlk turdan; re-doğrulamada TEYİT EDİLEMEDİ (en zayıf kayıt, asimetri gereği tutuluyor):
	"AC5RDfQFmDS1deWZos921JfqscXdByf8BKHs5ACWjtW2": "Bybit",
}

func IsCEX(addr string) bool { _, ok := knownCEX[addr]; return ok }
