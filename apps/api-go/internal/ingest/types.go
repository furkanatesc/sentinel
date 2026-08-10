package ingest

import "github.com/furkanatesc/sentinel/apps/api-go/internal/store"

// LogNotification, Helius logsSubscribe bildiriminin normalize halidir.
type LogNotification struct {
	Signature string
	Slot      uint64
	Err       any // nil = başarı; non-nil ise decode edilmez
	Logs      []string
	ProgramID string // hangi aboneliğin (program) tetiklediği
}

// Decoded, bir log bildiriminden çıkarılan olay + (yeni/upsert) token + (varsa) creator cüzdanı.
type Decoded struct {
	Event   store.EventRow
	Token   store.TokenRow
	Creator string // pump.fun CreateEvent `user` (dev) pubkey; yoksa "" (yalnız pump.fun doldurur)
}

// TxInfo, getTransaction'dan alınan pozisyonel hesap listesidir.
type TxInfo struct {
	AccountKeys []string
}

// TokenMeta, DAS getAsset'ten alınan metadata'dır.
type TokenMeta struct {
	Name, Symbol, URI string
}
