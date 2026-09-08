-- +goose Up
CREATE TABLE IF NOT EXISTS alert_rules (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    trigger           TEXT NOT NULL,
    scope             TEXT NOT NULL,
    min_liquidity     DOUBLE PRECISION NOT NULL DEFAULT 0,
    min_creator_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_risk          TEXT NOT NULL,
    channels          TEXT NOT NULL DEFAULT '[]',  -- JSON []string
    enabled           BOOLEAN NOT NULL DEFAULT true,
    created_ts        BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS alert_rules;
