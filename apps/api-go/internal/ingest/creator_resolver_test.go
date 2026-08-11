package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
)

type fakeSigTx struct {
	pages    [][]solana.Signature // sayfa sayfa (newest-first); son sayfanın son elemanı = en eski
	logsBy   map[solana.Signature][]string
	sigCalls int
}

func (f *fakeSigTx) listSignatures(_ context.Context, _ solana.PublicKey, before solana.Signature, _ int) ([]solana.Signature, error) {
	f.sigCalls++
	// before zero → ilk sayfa; aksi → bir sonraki sayfa (basit fake: çağrı sırasına göre).
	idx := f.sigCalls - 1
	if idx >= len(f.pages) {
		return nil, nil
	}
	return f.pages[idx], nil
}
func (f *fakeSigTx) txLogs(_ context.Context, sig solana.Signature) ([]string, error) {
	return f.logsBy[sig], nil
}

func mkCreateLogs(user [32]byte) []string {
	var mint, bonding [32]byte
	mint[0] = 9
	return []string{"Program log: Instruction: Create", "Program data: " + buildCreateEventB64("N", "S", "https://x/u.json", mint, bonding, user)}
}

func TestResolveCreatorFound(t *testing.T) {
	var user [32]byte
	user[0], user[31] = 4, 8
	oldest := solana.Signature{1}
	fx := &fakeSigTx{
		pages:  [][]solana.Signature{{solana.Signature{3}, solana.Signature{2}, oldest}}, // tek sayfa; son = en eski
		logsBy: map[solana.Signature][]string{oldest: mkCreateLogs(user)},
	}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000}
	creator, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || !found || creator != base58.Encode(user[:]) {
		t.Fatalf("creator=%q found=%v err=%v", creator, found, err)
	}
}

func TestResolveCreatorNotFoundWhenNoCreate(t *testing.T) {
	oldest := solana.Signature{1}
	fx := &fakeSigTx{
		pages:  [][]solana.Signature{{oldest}},
		logsBy: map[solana.Signature][]string{oldest: {"Program log: Instruction: Buy"}}, // create değil
	}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000}
	_, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || found {
		t.Fatalf("found=%v err=%v, want found=false", found, err)
	}
}

func TestResolveCreatorNoSignatures(t *testing.T) {
	fx := &fakeSigTx{pages: [][]solana.Signature{{}}}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000}
	_, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || found {
		t.Fatalf("found=%v err=%v, want found=false (sig yok)", found, err)
	}
}

// TestResolveCreatorMultiPageReachesOldest, küçük pageLimit ile ilk sayfanın
// TAM (== pageLimit) dönmesini zorlar → döngü `before`'ı ilerletip ikinci
// sayfayı çeker; ikinci sayfa kısa (< pageLimit) olduğundan orada durur ve
// gerçek en eski (page1'in tek elemanı) create tx olarak çözülür. Bu test,
// tek-sayfalık diğer testlerin egzersiz etmediği before-ilerletme + kısa
// sayfada durma mantığını kapsar.
func TestResolveCreatorMultiPageReachesOldest(t *testing.T) {
	var user [32]byte
	user[0], user[31] = 4, 8
	sig3 := solana.Signature{13}
	sig2 := solana.Signature{12}
	oldest := solana.Signature{11}
	fx := &fakeSigTx{
		pages: [][]solana.Signature{
			{sig3, sig2}, // sayfa 0: len==pageLimit(2) → devam, before=sig2
			{oldest},     // sayfa 1: len(1) < pageLimit(2) → dur, en eski = oldest
		},
		logsBy: map[solana.Signature][]string{oldest: mkCreateLogs(user)},
	}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 5, pageLimit: 2}
	creator, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || !found || creator != base58.Encode(user[:]) {
		t.Fatalf("creator=%q found=%v err=%v", creator, found, err)
	}
	if fx.sigCalls != 2 {
		t.Fatalf("sigCalls=%d, want 2 (birden fazla sayfa gezilmedi)", fx.sigCalls)
	}
}

// TestResolveCreatorCapHitNoFabrication, tüm sayfalar TAM (== pageLimit) dönerse
// (gerçek en eskiye asla ulaşılmadan) maxSigPages cap'inin devreye girdiğini ve
// resolver'ın cap'te elde kalan (muhtemelen create olmayan) sig'i creator diye
// UYDURMADIĞINI kanıtlar — found=false, err=nil (dürüst boş).
func TestResolveCreatorCapHitNoFabrication(t *testing.T) {
	sig5 := solana.Signature{25}
	sig4 := solana.Signature{24}
	sig3 := solana.Signature{23}
	sig2 := solana.Signature{22} // cap'te elde kalan "en eski" (gerçek en eski değil)
	fx := &fakeSigTx{
		pages: [][]solana.Signature{
			{sig5, sig4}, // sayfa 0: len==pageLimit(2) → devam
			{sig3, sig2}, // sayfa 1: len==pageLimit(2) → devam ama maxSigPages(2) doldu, döngü biter
		},
		logsBy: map[solana.Signature][]string{sig2: {"Program log: Instruction: Buy"}}, // create değil
	}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 2, pageLimit: 2}
	creator, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || found || creator != "" {
		t.Fatalf("creator=%q found=%v err=%v, want found=false (cap'te uydurma yok)", creator, found, err)
	}
	if fx.sigCalls != 2 {
		t.Fatalf("sigCalls=%d, want 2 (cap=maxSigPages'de durmalı)", fx.sigCalls)
	}
}

// countingLimiter, Wait çağrılarını sayar (rate-limit uygulandığını kanıtlar).
type countingLimiter struct{ waits int }

func (c *countingLimiter) Wait(context.Context) error { c.waits++; return nil }

// flakySigTx, ilk failsLeft çağrıda err döner, sonra sabit sigs/logs verir
// (429 backoff-retry davranışını test için).
type flakySigTx struct {
	failsLeft int
	err       error
	sigs      []solana.Signature
	logs      []string
	sigCalls  int
}

func (f *flakySigTx) listSignatures(context.Context, solana.PublicKey, solana.Signature, int) ([]solana.Signature, error) {
	f.sigCalls++
	if f.failsLeft > 0 {
		f.failsLeft--
		return nil, f.err
	}
	return f.sigs, nil
}
func (f *flakySigTx) txLogs(context.Context, solana.Signature) ([]string, error) { return f.logs, nil }

// TestResolveCreatorRateLimited, her RPC çağrısından ÖNCE limiter.Wait çağrıldığını
// doğrular (1 listSignatures + 1 txLogs = 2 Wait) — burst'ü önleyen düzeltmenin özü.
func TestResolveCreatorRateLimited(t *testing.T) {
	var user [32]byte
	user[0], user[31] = 4, 8
	oldest := solana.Signature{1}
	fx := &fakeSigTx{
		pages:  [][]solana.Signature{{oldest}},
		logsBy: map[solana.Signature][]string{oldest: mkCreateLogs(user)},
	}
	lim := &countingLimiter{}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000, limiter: lim}
	_, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if lim.waits != 2 {
		t.Fatalf("limiter.Wait=%d, want 2 (her RPC öncesi bir token)", lim.waits)
	}
}

// TestResolveCreatorRetriesOn429, 429 hatasında sınırlı backoff-retry yapıldığını
// ve sonraki denemede başarıya ulaştığını doğrular.
func TestResolveCreatorRetriesOn429(t *testing.T) {
	var user [32]byte
	user[0], user[31] = 4, 8
	oldest := solana.Signature{1}
	fx := &flakySigTx{
		failsLeft: 1,
		err:       errors.New("rpc call getSignaturesForAddress() on https://x status code: 429. could not decode body"),
		sigs:      []solana.Signature{oldest},
		logs:      mkCreateLogs(user),
	}
	slept := 0
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000,
		sleep: func(context.Context, time.Duration) error { slept++; return nil }}
	creator, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err != nil || !found || creator != base58.Encode(user[:]) {
		t.Fatalf("creator=%q found=%v err=%v", creator, found, err)
	}
	if fx.sigCalls != 2 {
		t.Fatalf("sigCalls=%d, want 2 (429 sonrası 1 retry)", fx.sigCalls)
	}
	if slept != 1 {
		t.Fatalf("slept=%d, want 1 (429 backoff)", slept)
	}
}

// TestResolveCreator429Exhausted, 429 sürekli sürerse denemenin bounded olduğunu
// (crMaxAttempts'te durup hata döndüğünü) doğrular — sonsuz retry yok.
func TestResolveCreator429Exhausted(t *testing.T) {
	fx := &flakySigTx{failsLeft: 99, err: errors.New("status code: 429")}
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000,
		sleep: func(context.Context, time.Duration) error { return nil }}
	_, found, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err == nil || found {
		t.Fatalf("want err (429 tükendi), got found=%v err=%v", found, err)
	}
	if fx.sigCalls != crMaxAttempts {
		t.Fatalf("sigCalls=%d, want %d (bounded)", fx.sigCalls, crMaxAttempts)
	}
}

// TestResolveCreatorNoRetryNon429, 429 DIŞI hatada retry YAPILMADIĞINI (anında
// hata döndüğünü) doğrular — sadece rate-limit retry'lanır.
func TestResolveCreatorNoRetryNon429(t *testing.T) {
	fx := &flakySigTx{failsLeft: 99, err: errors.New("connection refused")}
	slept := 0
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000,
		sleep: func(context.Context, time.Duration) error { slept++; return nil }}
	_, _, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err == nil {
		t.Fatal("want err")
	}
	if fx.sigCalls != 1 {
		t.Fatalf("sigCalls=%d, want 1 (429 dışı → retry yok)", fx.sigCalls)
	}
	if slept != 0 {
		t.Fatalf("slept=%d, want 0 (429 dışı → backoff yok)", slept)
	}
}

// TestResolveCreatorNoRetryOn429Substring, hata metninde tesadüfen "429" GEÇEN ama
// gerçek rate-limit ("status code: 429") OLMAYAN bir hatanın retry'lanMADIĞINI
// doğrular — is429'un dar (spesifik ibare) eşleşmesini kilitler.
func TestResolveCreatorNoRetryOn429Substring(t *testing.T) {
	fx := &flakySigTx{failsLeft: 99, err: errors.New("invalid signature at slot 84290: not found")}
	slept := 0
	r := &HeliusCreatorResolver{rpc: fx, maxSigPages: 3, pageLimit: 1000,
		sleep: func(context.Context, time.Duration) error { slept++; return nil }}
	_, _, err := r.ResolveCreator(context.Background(), base58.Encode(make([]byte, 32)))
	if err == nil {
		t.Fatal("want err")
	}
	if fx.sigCalls != 1 {
		t.Fatalf("sigCalls=%d, want 1 (metinde '429' var ama 'status code: 429' yok → retry yok)", fx.sigCalls)
	}
	if slept != 0 {
		t.Fatalf("slept=%d, want 0", slept)
	}
}
