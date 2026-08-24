package walletgraph

// knownCEX, bilinen büyük Solana CEX hot-wallet'larıdır (küçük allowlist; bunları fonlayan
// "küme" bundler DEĞİL, borsa çekimidir → dışlanır). Genişletme 2e-1 kapsamı dışı (followup).
//
// Adresler 2026-08-24'te web araştırmasıyla doğrulandı (Solscan public label + en az bir bağımsız
// kaynak — flipside-wiki CEX listesi veya borsanın kendi Proof-of-Reserves raporu gibi). Sadece
// çakışmasız/tek-etiketli, çoklu kaynakla desteklenen adresler eklendi; etiketi çelişkili görünen
// bir Coinbase/Kraken adayı (H8sMJSCQ...) belirsizlik nedeniyle BİLİNÇLİ OLARAK DIŞLANDI.
var knownCEX = map[string]string{
	"5tzFkiKscXHK5ZXCGbXZxdw7gTjjD1mBwuoFbhUvuAi9": "Binance",
	"GJRs4FwHtemZ5ZE9x3FNvJ8TMwitKTh21yxdRPqn7npE": "Coinbase",
	"FWznbcNXWQuHTawe9RxvQ2LdCENssh12dsznf4RiouN5": "Kraken",
	"C68a6RCGLiPskbPYtAcsCjhG8tfTWYcoB4JjCrXFdqyo": "OKX",
	"AC5RDfQFmDS1deWZos921JfqscXdByf8BKHs5ACWjtW2": "Bybit",
}

func IsCEX(addr string) bool { _, ok := knownCEX[addr]; return ok }
