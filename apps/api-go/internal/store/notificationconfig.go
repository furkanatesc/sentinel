package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
)

// notificationConfigID, tek-satırlık config'in sabit anahtarıdır.
const notificationConfigID = "default"

// NotificationTemplate, bir tetikleyici için Slack mesaj şablonudur (frontend ile birebir JSON).
type NotificationTemplate struct {
	Trigger  string `json:"trigger"`
	Template string `json:"template"`
}

// QuietHours, sessiz saatler aralığıdır (teslimat bu aralıkta bastırılır).
type QuietHours struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	Enabled bool   `json:"enabled"`
}

// NotificationSettings, bildirim ayarlarının KALICI alt-kümesidir. Bağlantı-durumu
// (slackState/workspace) burada değildir — gerçek Slack OAuth'tan gelir (henüz yok).
type NotificationSettings struct {
	Channel       string                 `json:"channel"`
	MinSeverity   string                 `json:"minSeverity"`
	QuietHours    QuietHours             `json:"quietHours"`
	Templates     []NotificationTemplate `json:"templates"`
	TradeApproval bool                   `json:"tradeApproval"`
}

// NotificationConfigStore, tek-satırlık bildirim ayarları kalıcılığıdır (DIP; postgres + fake karşılar).
type NotificationConfigStore interface {
	GetNotificationSettings(ctx context.Context) (NotificationSettings, error)
	SaveNotificationSettings(ctx context.Context, s NotificationSettings) (NotificationSettings, error)
}

// SeedNotificationSettings, ilk açılış varsayılanıdır (frontend mock'unun ayar karşılığı).
func SeedNotificationSettings() NotificationSettings {
	return NotificationSettings{
		Channel:     "#alerts",
		MinSeverity: "warning",
		QuietHours:  QuietHours{Start: "23:00", End: "07:00", Enabled: false},
		Templates: []NotificationTemplate{
			{Trigger: "liquidity_removed", Template: "🚨 {{token}}: likidite çekildi — {{detail}}"},
			{Trigger: "whale_activity", Template: "🐋 {{token}}: balina hareketi — {{detail}}"},
			{Trigger: "new_mint", Template: "✨ Yeni mint: {{token}} — {{detail}}"},
		},
		TradeApproval: true,
	}
}

// --- postgres ---

func (p *postgresStore) GetNotificationSettings(ctx context.Context) (NotificationSettings, error) {
	const q = `SELECT channel, min_severity, quiet_start, quiet_end, quiet_enabled, templates, trade_approval
		FROM notification_config WHERE id=$1`
	var s NotificationSettings
	var tpl string
	err := p.db.QueryRowContext(ctx, q, notificationConfigID).Scan(
		&s.Channel, &s.MinSeverity, &s.QuietHours.Start, &s.QuietHours.End, &s.QuietHours.Enabled, &tpl, &s.TradeApproval)
	if err == sql.ErrNoRows {
		// Satır yoksa (seed atlanmış/silinmiş) güvenli varsayılan dön.
		return SeedNotificationSettings(), nil
	}
	if err != nil {
		return NotificationSettings{}, err
	}
	s.Templates = decodeTemplates(tpl)
	return s, nil
}

func (p *postgresStore) SaveNotificationSettings(ctx context.Context, s NotificationSettings) (NotificationSettings, error) {
	if s.Templates == nil {
		s.Templates = []NotificationTemplate{}
	}
	const q = `INSERT INTO notification_config
		(id, channel, min_severity, quiet_start, quiet_end, quiet_enabled, templates, trade_approval)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET
			channel=EXCLUDED.channel, min_severity=EXCLUDED.min_severity,
			quiet_start=EXCLUDED.quiet_start, quiet_end=EXCLUDED.quiet_end, quiet_enabled=EXCLUDED.quiet_enabled,
			templates=EXCLUDED.templates, trade_approval=EXCLUDED.trade_approval`
	if _, err := p.db.ExecContext(ctx, q, notificationConfigID, s.Channel, s.MinSeverity,
		s.QuietHours.Start, s.QuietHours.End, s.QuietHours.Enabled, encodeTemplates(s.Templates), s.TradeApproval); err != nil {
		return NotificationSettings{}, err
	}
	return s, nil
}

func seedNotificationConfig(ctx context.Context, db *sql.DB) error {
	// Yalnız tablo boşsa seed'le (bkz. seedAlertRules — her açılışta ezmemek için).
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_config`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	s := SeedNotificationSettings()
	const q = `INSERT INTO notification_config
		(id, channel, min_severity, quiet_start, quiet_end, quiet_enabled, templates, trade_approval)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`
	_, err := db.ExecContext(ctx, q, notificationConfigID, s.Channel, s.MinSeverity,
		s.QuietHours.Start, s.QuietHours.End, s.QuietHours.Enabled, encodeTemplates(s.Templates), s.TradeApproval)
	return err
}

func encodeTemplates(t []NotificationTemplate) string {
	if t == nil {
		t = []NotificationTemplate{}
	}
	b, _ := json.Marshal(t)
	return string(b)
}

func decodeTemplates(s string) []NotificationTemplate {
	var t []NotificationTemplate
	if err := json.Unmarshal([]byte(s), &t); err != nil || t == nil {
		return []NotificationTemplate{}
	}
	return t
}

// --- fake ---

type fakeNotificationConfigStore struct {
	mu       sync.Mutex
	settings NotificationSettings
}

// NewFakeNotificationConfigStore, DB'siz mod/testler için in-memory store (varsayılanla seed'li).
func NewFakeNotificationConfigStore() NotificationConfigStore {
	return &fakeNotificationConfigStore{settings: SeedNotificationSettings()}
}

func (f *fakeNotificationConfigStore) GetNotificationSettings(_ context.Context) (NotificationSettings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.settings, nil
}

func (f *fakeNotificationConfigStore) SaveNotificationSettings(_ context.Context, s NotificationSettings) (NotificationSettings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s.Templates == nil {
		s.Templates = []NotificationTemplate{}
	}
	f.settings = s
	return s, nil
}
