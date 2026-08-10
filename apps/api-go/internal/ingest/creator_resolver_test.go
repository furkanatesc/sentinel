package ingest

import (
	"context"
	"testing"

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
