package store

import (
	"context"
	"database/sql"
	"sort"
	"sync"
)

const alertEvalMetaID = "default"

// AlertEventRow, üretilmiş bir alarmdır (frontend AlertEvent ile birebir JSON). RuleID + Ts iç-kullanım
// (dedup/persist); JSON kontratında olmadıkları için çıkmazlar.
type AlertEventRow struct {
	ID       string `json:"id"`
	RuleID   string `json:"-"`
	Type     string `json:"type"`
	Token    string `json:"token"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
	Time     string `json:"time"`
	Ts       int64  `json:"-"`
}

// AlertEventStore, üretilmiş alarmların append-only kaydı + değerlendirme watermark'ıdır (DIP).
type AlertEventStore interface {
	// InsertAlertEvent, alarmı yazar. ID (event+kural kompoziti) zaten varsa idempotent atlar ve
	// inserted=false döner — worker'ın watermark sınırında tekrar-değerlendirdiği alarmları çift saymaması için.
	InsertAlertEvent(ctx context.Context, e AlertEventRow) (inserted bool, err error)
	RecentAlertEvents(ctx context.Context, limit int) ([]AlertEventRow, error)
	GetAlertWatermark(ctx context.Context) (int64, error)
	SetAlertWatermark(ctx context.Context, ts int64) error
}

// --- postgres ---

func (p *postgresStore) InsertAlertEvent(ctx context.Context, e AlertEventRow) (bool, error) {
	const q = `INSERT INTO alert_events (id, rule_id, type, token, detail, severity, time, ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`
	res, err := p.db.ExecContext(ctx, q, e.ID, e.RuleID, e.Type, e.Token, e.Detail, e.Severity, e.Time, e.Ts)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (p *postgresStore) RecentAlertEvents(ctx context.Context, limit int) ([]AlertEventRow, error) {
	const q = `SELECT id, rule_id, type, token, detail, severity, time, ts
		FROM alert_events ORDER BY ts DESC, id DESC LIMIT $1`
	rows, err := p.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AlertEventRow{}
	for rows.Next() {
		var e AlertEventRow
		if err := rows.Scan(&e.ID, &e.RuleID, &e.Type, &e.Token, &e.Detail, &e.Severity, &e.Time, &e.Ts); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (p *postgresStore) GetAlertWatermark(ctx context.Context) (int64, error) {
	var ts int64
	err := p.db.QueryRowContext(ctx, `SELECT watermark_ts FROM alert_eval_meta WHERE id=$1`, alertEvalMetaID).Scan(&ts)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return ts, err
}

func (p *postgresStore) SetAlertWatermark(ctx context.Context, ts int64) error {
	const q = `INSERT INTO alert_eval_meta (id, watermark_ts) VALUES ($1,$2)
		ON CONFLICT (id) DO UPDATE SET watermark_ts=EXCLUDED.watermark_ts`
	_, err := p.db.ExecContext(ctx, q, alertEvalMetaID, ts)
	return err
}

// --- fake ---

type fakeAlertEventStore struct {
	mu     sync.Mutex
	events []AlertEventRow
	ids    map[string]bool
	wm     int64
}

// NewFakeAlertEventStore, DB'siz mod/testler için in-memory store.
func NewFakeAlertEventStore() AlertEventStore {
	return &fakeAlertEventStore{events: []AlertEventRow{}, ids: map[string]bool{}}
}

func (f *fakeAlertEventStore) InsertAlertEvent(_ context.Context, e AlertEventRow) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ids[e.ID] { // ON CONFLICT (id) DO NOTHING parity
		return false, nil
	}
	f.ids[e.ID] = true
	f.events = append(f.events, e)
	return true, nil
}

func (f *fakeAlertEventStore) RecentAlertEvents(_ context.Context, limit int) ([]AlertEventRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]AlertEventRow(nil), f.events...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ts != out[j].Ts {
			return out[i].Ts > out[j].Ts
		}
		return out[i].ID > out[j].ID
	})
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeAlertEventStore) GetAlertWatermark(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.wm, nil
}

func (f *fakeAlertEventStore) SetAlertWatermark(_ context.Context, ts int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.wm = ts
	return nil
}
