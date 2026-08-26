-- +goose Up
CREATE TABLE IF NOT EXISTS wallet_funders (
    wallet      TEXT PRIMARY KEY,
    funder      TEXT   NOT NULL DEFAULT '',
    resolved_ts BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wallet_funders_funder ON wallet_funders (funder) WHERE funder <> '';

-- +goose Down
DROP TABLE IF EXISTS wallet_funders;
