package ingest

import "context"

// TxFetcher, decoder gerektiğinde ham işlemi çeker (DIP; pump.fun kullanmaz).
type TxFetcher interface {
	GetTransaction(ctx context.Context, signature string) (TxInfo, error)
}

// MetadataFetcher, mint → name/symbol (Helius DAS getAsset); başarısızlıkta çağıran fallback yapar.
type MetadataFetcher interface {
	Metadata(ctx context.Context, mint string) (TokenMeta, error)
}

// LaunchpadDecoder, bir launchpad program'ının log'larını olaylara çevirir (OCP birimi).
type LaunchpadDecoder interface {
	ProgramID() string
	Launchpad() string
	Decode(ctx context.Context, n LogNotification, tx TxFetcher, md MetadataFetcher) ([]Decoded, error)
}

// Registry, programID → decoder eşlemesidir (OCP: yeni launchpad = Register çağrısı).
type Registry struct {
	byProgram map[string]LaunchpadDecoder
}

func NewRegistry() *Registry { return &Registry{byProgram: map[string]LaunchpadDecoder{}} }

func (r *Registry) Register(d LaunchpadDecoder) { r.byProgram[d.ProgramID()] = d }

func (r *Registry) ProgramIDs() []string {
	out := make([]string, 0, len(r.byProgram))
	for id := range r.byProgram {
		out = append(out, id)
	}
	return out
}

func (r *Registry) Decoder(programID string) (LaunchpadDecoder, bool) {
	d, ok := r.byProgram[programID]
	return d, ok
}
