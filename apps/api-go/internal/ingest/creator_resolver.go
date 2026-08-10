package ingest

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// sigTxRPC, resolver'ın ihtiyaç duyduğu iki RPC çağrısını soyutlar (DIP; test için fake).
type sigTxRPC interface {
	// listSignatures, mint hesabının sinyallerini newest-first döndürür (before'dan geriye, ≤limit).
	listSignatures(ctx context.Context, acct solana.PublicKey, before solana.Signature, limit int) ([]solana.Signature, error)
	txLogs(ctx context.Context, sig solana.Signature) ([]string, error)
}

// HeliusCreatorResolver, bir mint'in create tx'inden pump.fun creator'ını çözer (REST).
type HeliusCreatorResolver struct {
	rpc         sigTxRPC
	maxSigPages int
	pageLimit   int
}

func NewCreatorResolver(rpcURL string, maxSigPages int) *HeliusCreatorResolver {
	if maxSigPages <= 0 {
		maxSigPages = 3
	}
	return &HeliusCreatorResolver{rpc: &heliusSigTx{cli: rpc.New(rpcURL)}, maxSigPages: maxSigPages, pageLimit: 1000}
}

// ResolveCreator, mint'in EN ESKİ sig'ini bulur (create tx), tx log'larından creator'ı çıkarır.
// Cap'e takılır ya da create tanınmazsa found=false (dürüst boş).
func (r *HeliusCreatorResolver) ResolveCreator(ctx context.Context, mint string) (string, bool, error) {
	acct, err := solana.PublicKeyFromBase58(mint)
	if err != nil {
		return "", false, err
	}
	var before, oldest solana.Signature
	oldestFound := false
	for page := 0; page < r.maxSigPages; page++ {
		sigs, err := r.rpc.listSignatures(ctx, acct, before, r.pageLimit)
		if err != nil {
			return "", false, err
		}
		if len(sigs) == 0 {
			break
		}
		oldest = sigs[len(sigs)-1] // newest-first → son eleman = bu sayfanın en eskisi
		oldestFound = true
		before = oldest
		if len(sigs) < r.pageLimit {
			break // en eskiye ulaşıldı (tam sayfa değil)
		}
	}
	if !oldestFound {
		return "", false, nil
	}
	logs, err := r.rpc.txLogs(ctx, oldest)
	if err != nil {
		return "", false, err
	}
	creator, ok := CreatorFromCreateLogs(logs)
	return creator, ok, nil
}

// heliusSigTx, sigTxRPC'yi solana-go rpc client ile karşılar (canlı; deploy'da doğrulanır).
type heliusSigTx struct{ cli *rpc.Client }

func (h *heliusSigTx) listSignatures(ctx context.Context, acct solana.PublicKey, before solana.Signature, limit int) ([]solana.Signature, error) {
	opts := &rpc.GetSignaturesForAddressOpts{Limit: &limit}
	if !before.IsZero() {
		opts.Before = before
	}
	res, err := h.cli.GetSignaturesForAddressWithOpts(ctx, acct, opts)
	if err != nil {
		return nil, err
	}
	out := make([]solana.Signature, 0, len(res))
	for _, s := range res {
		// s.Err bilerek yok sayılır: başarısız bir create tx zaten log'larından
		// decode edilemez (CreatorFromCreateLogs → ok=false), yani found=false'a
		// güvenle düşer; burada ayrıca filtrelemeye gerek yok.
		out = append(out, s.Signature)
	}
	return out, nil
}

func (h *heliusSigTx) txLogs(ctx context.Context, sig solana.Signature) ([]string, error) {
	maxV := uint64(0)
	res, err := h.cli.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		MaxSupportedTransactionVersion: &maxV,
	})
	if err != nil {
		return nil, err
	}
	if res == nil || res.Meta == nil {
		// Meta yok (ör. RPC node'un tx-retention penceresini aşmış) → bilerek hata DEĞİL,
		// found=false'a düşecek şekilde nil log listesi döndürülür — s.Err'in yukarıda
		// bilerek yok sayılmasıyla aynı "dürüst not-found" deseni.
		return nil, nil
	}
	return res.Meta.LogMessages, nil
}
