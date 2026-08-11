package ingest

import (
	"context"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Limiter, dışa giden RPC çağrılarını geciktiren paylaşılan hız sınırlayıcıdır
// (ISP: tek metot). *golang.org/x/time/rate.Limiter bu arayüzü doğrudan karşılar.
type Limiter interface {
	Wait(ctx context.Context) error
}

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
	limiter     Limiter                                          // nil → sınırlama yok
	sleep       func(ctx context.Context, d time.Duration) error // 429 backoff (test için enjekte edilebilir)
}

// ResolverOption, HeliusCreatorResolver'ı yapılandırır (OCP: constructor'ı değiştirmeden genişlet).
type ResolverOption func(*HeliusCreatorResolver)

// WithLimiter, paylaşılan hız sınırlayıcı enjekte eder. Helius free-tier RPC 429'u
// PER-KEY'dir (per-IP DEĞİL) → client-side sınırlama gerçekten burst'ü engeller.
func WithLimiter(l Limiter) ResolverOption {
	return func(r *HeliusCreatorResolver) { r.limiter = l }
}

func NewCreatorResolver(rpcURL string, maxSigPages int, opts ...ResolverOption) *HeliusCreatorResolver {
	if maxSigPages <= 0 {
		maxSigPages = 3
	}
	r := &HeliusCreatorResolver{rpc: &heliusSigTx{cli: rpc.New(rpcURL)}, maxSigPages: maxSigPages, pageLimit: 1000, sleep: sleepCtx}
	for _, o := range opts {
		o(r)
	}
	return r
}

// crMaxAttempts, 429 (rate-limit) için toplam deneme sayısıdır (1 asıl + retry'lar).
const crMaxAttempts = 3

// withRetry, her RPC çağrısından önce paylaşılan limiter'dan token alır ve 429'da
// sınırlı sayıda backoff'lu retry yapar. 429 dışı hatalar anında döner (retry yok).
func (r *HeliusCreatorResolver) withRetry(ctx context.Context, fn func() error) error {
	sleep := r.sleep
	if sleep == nil {
		sleep = sleepCtx // struct-literal ile kurulan resolver'da (constructor atlanmışsa) nil-panik yerine güvenli varsayılan
	}
	var lastErr error
	for attempt := 0; attempt < crMaxAttempts; attempt++ {
		if r.limiter != nil {
			if err := r.limiter.Wait(ctx); err != nil {
				return err
			}
		}
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		if !is429(err) || attempt == crMaxAttempts-1 {
			return err
		}
		if serr := sleep(ctx, crBackoff(attempt)); serr != nil {
			return serr
		}
	}
	return lastErr
}

// is429, solana-go rpc hatasının HTTP 429 (rate-limit) olup olmadığını saptar.
// solana-go 429'u JSON'a çözemez ve mesaja tam "status code: 429" ibaresini gömer;
// bu spesifik ibareye eşleşmek, hata metninde tesadüfen "429" geçen (ör. slot/imza)
// rate-limit dışı hataların yanlışlıkla retry'lanmasını önler.
func is429(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status code: 429")
}

// crBackoff, deneme sırasına göre artan gecikme verir (0→500ms, 1→1s).
func crBackoff(attempt int) time.Duration {
	return time.Duration(500*(1<<attempt)) * time.Millisecond
}

// sleepCtx, ctx iptaline duyarlı uyku (varsayılan backoff uygulayıcı).
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
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
		var sigs []solana.Signature
		if err := r.withRetry(ctx, func() error {
			var e error
			sigs, e = r.rpc.listSignatures(ctx, acct, before, r.pageLimit)
			return e
		}); err != nil {
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
	var logs []string
	if err := r.withRetry(ctx, func() error {
		var e error
		logs, e = r.rpc.txLogs(ctx, oldest)
		return e
	}); err != nil {
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
