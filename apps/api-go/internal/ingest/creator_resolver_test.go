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
