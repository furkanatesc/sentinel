package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AlertRule, alarm kuralıdır (frontend AlertRule ile birebir JSON). Uygulamanın ilk
// yazılabilir (mutation) kaynağı — kural CRUD; trade-mutasyonu DEĞİL (yapısal güvenlik korunur).
type AlertRule struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Trigger         string   `json:"trigger"`
	Scope           string   `json:"scope"`
	MinLiquidity    float64  `json:"minLiquidity"`
	MinCreatorScore float64  `json:"minCreatorScore"`
	MaxRisk         string   `json:"maxRisk"`
	Channels        []string `json:"channels"`
	Enabled         bool     `json:"enabled"`
	CreatedTs       int64    `json:"-"`
}

// AlertRuleStore, alarm kuralı kalıcılığıdır (DIP; postgres + fake karşılar).
type AlertRuleStore interface {
	ListAlertRules(ctx context.Context) ([]AlertRule, error)
	CreateAlertRule(ctx context.Context, r AlertRule) (AlertRule, error)
	SetAlertRuleEnabled(ctx context.Context, id string, enabled bool) error
}

// SeedAlertRules, ilk açılışta eklenecek varsayılan kurallardır (frontend mock'unun karşılığı).
func SeedAlertRules() []AlertRule {
	return []AlertRule{
		{ID: "r1", Name: "Balina alımı — tüm tokenlar", Trigger: "whale_activity", Scope: "Tüm tokenlar", MinLiquidity: 50000, MaxRisk: "high", Channels: []string{"web", "slack"}, Enabled: true},
		{ID: "r2", Name: "Likidite çekilişi — kritik", Trigger: "liquidity_removed", Scope: "Tüm tokenlar", MaxRisk: "critical", Channels: []string{"web", "slack", "email"}, Enabled: true},
		{ID: "r3", Name: "Yüksek skorlu yeni mint", Trigger: "new_mint", Scope: "Pump.fun", MinLiquidity: 10000, MinCreatorScore: 70, MaxRisk: "medium", Channels: []string{"slack"}, Enabled: false},
		{ID: "r4", Name: "Üretici satışı uyarısı", Trigger: "creator_sale", Scope: "Tüm tokenlar", MaxRisk: "high", Channels: []string{"web"}, Enabled: true},
	}
}

// --- postgres ---

func (p *postgresStore) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	const q = `SELECT id, name, trigger, scope, min_liquidity, min_creator_score, max_risk, channels, enabled, created_ts
		FROM alert_rules ORDER BY created_ts ASC, id ASC`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AlertRule{}
	for rows.Next() {
		var r AlertRule
		var ch string
		if err := rows.Scan(&r.ID, &r.Name, &r.Trigger, &r.Scope, &r.MinLiquidity, &r.MinCreatorScore, &r.MaxRisk, &ch, &r.Enabled, &r.CreatedTs); err != nil {
			return nil, err
		}
		r.Channels = decodeChannels(ch)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *postgresStore) CreateAlertRule(ctx context.Context, r AlertRule) (AlertRule, error) {
	if r.ID == "" {
		r.ID = newRuleID()
	}
	if r.CreatedTs == 0 {
		r.CreatedTs = time.Now().Unix()
	}
	if r.Channels == nil {
		r.Channels = []string{}
	}
	const q = `INSERT INTO alert_rules
		(id, name, trigger, scope, min_liquidity, min_creator_score, max_risk, channels, enabled, created_ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	if _, err := p.db.ExecContext(ctx, q, r.ID, r.Name, r.Trigger, r.Scope, r.MinLiquidity, r.MinCreatorScore, r.MaxRisk, encodeChannels(r.Channels), r.Enabled, r.CreatedTs); err != nil {
		return AlertRule{}, err
	}
	return r, nil
}

func (p *postgresStore) SetAlertRuleEnabled(ctx context.Context, id string, enabled bool) error {
	res, err := p.db.ExecContext(ctx, `UPDATE alert_rules SET enabled=$1 WHERE id=$2`, enabled, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ErrNotFound, bilinmeyen id (PATCH 404 için).
var ErrNotFound = fmt.Errorf("alert rule not found")

func seedAlertRules(ctx context.Context, db *sql.DB) error {
	const q = `INSERT INTO alert_rules
		(id, name, trigger, scope, min_liquidity, min_creator_score, max_risk, channels, enabled, created_ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (id) DO NOTHING`
	now := time.Now().Unix()
	for i, r := range SeedAlertRules() {
		if _, err := db.ExecContext(ctx, q, r.ID, r.Name, r.Trigger, r.Scope, r.MinLiquidity, r.MinCreatorScore, r.MaxRisk, encodeChannels(r.Channels), r.Enabled, now+int64(i)); err != nil {
			return err
		}
	}
	return nil
}

func encodeChannels(ch []string) string {
	if ch == nil {
		ch = []string{}
	}
	b, _ := json.Marshal(ch)
	return string(b)
}

func decodeChannels(s string) []string {
	var ch []string
	if err := json.Unmarshal([]byte(s), &ch); err != nil || ch == nil {
		return []string{}
	}
	return ch
}

func newRuleID() string {
	return fmt.Sprintf("r%d", time.Now().UnixNano())
}

// --- fake ---

type fakeAlertRuleStore struct {
	mu    sync.Mutex
	rules []AlertRule
}

// NewFakeAlertRuleStore, DB'siz mod/testler için in-memory store (varsayılan kurallarla seed'li).
func NewFakeAlertRuleStore() AlertRuleStore {
	return &fakeAlertRuleStore{rules: SeedAlertRules()}
}

func (f *fakeAlertRuleStore) ListAlertRules(_ context.Context) ([]AlertRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]AlertRule(nil), f.rules...), nil
}

func (f *fakeAlertRuleStore) CreateAlertRule(_ context.Context, r AlertRule) (AlertRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.ID == "" {
		r.ID = newRuleID()
	}
	if r.CreatedTs == 0 {
		r.CreatedTs = time.Now().Unix()
	}
	if r.Channels == nil {
		r.Channels = []string{}
	}
	f.rules = append(f.rules, r)
	return r, nil
}

func (f *fakeAlertRuleStore) SetAlertRuleEnabled(_ context.Context, id string, enabled bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.rules {
		if f.rules[i].ID == id {
			f.rules[i].Enabled = enabled
			return nil
		}
	}
	return ErrNotFound
}
