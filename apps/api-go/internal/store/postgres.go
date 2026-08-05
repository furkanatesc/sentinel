package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresStore struct {
	db *sql.DB
}

// Bundle, açılan Postgres bağlantısının sunduğu store'ları toplar.
type Bundle struct {
	Strategies StrategyStore
	Events     EventStore
	Tokens     TokenStore
}

// OpenPostgres, bağlantı açar, migration'ları çalıştırır, strateji seed'ini uygular
// ve store bundle'ı ile kapatma fonksiyonu döner.
func OpenPostgres(ctx context.Context, dsn string) (Bundle, func() error, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return Bundle{}, nil, fmt.Errorf("open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return Bundle{}, nil, fmt.Errorf("ping: %w", err)
	}
	if err := runMigrations(db); err != nil {
		db.Close()
		return Bundle{}, nil, fmt.Errorf("migrate: %w", err)
	}
	if err := seedStrategies(ctx, db); err != nil {
		db.Close()
		return Bundle{}, nil, fmt.Errorf("seed: %w", err)
	}
	ps := &postgresStore{db: db}
	return Bundle{Strategies: ps, Events: ps, Tokens: ps}, db.Close, nil
}

func seedStrategies(ctx context.Context, db *sql.DB) error {
	const q = `INSERT INTO strategies
		(id, name, status, timeframe, win_rate_pct, profit_factor, max_drawdown_pct, total_trades, net_pnl_sol, last_signal)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO NOTHING`
	for _, s := range SeedRows() {
		if _, err := db.ExecContext(ctx, q, s.ID, s.Name, s.Status, s.Timeframe,
			s.WinRatePct, s.ProfitFactor, s.MaxDrawdownPct, s.TotalTrades, s.NetPnlSol, s.LastSignal); err != nil {
			return err
		}
	}
	return nil
}

func (p *postgresStore) List(ctx context.Context) ([]StrategyRow, error) {
	const q = `SELECT id, name, status, timeframe, win_rate_pct, profit_factor,
		max_drawdown_pct, total_trades, net_pnl_sol, last_signal FROM strategies ORDER BY id`
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StrategyRow
	for rows.Next() {
		var s StrategyRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.Timeframe, &s.WinRatePct,
			&s.ProfitFactor, &s.MaxDrawdownPct, &s.TotalTrades, &s.NetPnlSol, &s.LastSignal); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
