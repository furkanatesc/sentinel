-- +goose Up
CREATE TABLE IF NOT EXISTS creators (
  address             TEXT PRIMARY KEY,
  reputation_score    DOUBLE PRECISION NOT NULL DEFAULT 0,
  confidence          DOUBLE PRECISION NOT NULL DEFAULT 0,
  risk_level          TEXT NOT NULL DEFAULT 'medium',
  breakdown           TEXT NOT NULL DEFAULT '',
  total_tokens        INTEGER NOT NULL DEFAULT 0,
  active_tokens       INTEGER NOT NULL DEFAULT 0,
  rugged_tokens       INTEGER NOT NULL DEFAULT 0,
  graduated_tokens    INTEGER NOT NULL DEFAULT 0,
  avg_peak_market_cap DOUBLE PRECISION NOT NULL DEFAULT 0,
  avg_lifetime_hours  DOUBLE PRECISION NOT NULL DEFAULT 0,
  success_rate_pct    DOUBLE PRECISION NOT NULL DEFAULT 0,
  scored_ts           BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS creators;
