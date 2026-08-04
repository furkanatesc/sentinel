-- +goose Up
CREATE TABLE IF NOT EXISTS strategies (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    status           TEXT NOT NULL,
    timeframe        TEXT NOT NULL,
    win_rate_pct     DOUBLE PRECISION NOT NULL,
    profit_factor    DOUBLE PRECISION NOT NULL,
    max_drawdown_pct DOUBLE PRECISION NOT NULL,
    total_trades     INTEGER NOT NULL,
    net_pnl_sol      DOUBLE PRECISION NOT NULL,
    last_signal      TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS strategies;
