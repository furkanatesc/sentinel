-- +goose Up
CREATE TABLE IF NOT EXISTS kpi_samples (
    ts        BIGINT PRIMARY KEY,   -- unix saniye (örnek anı; PK → aynı ts idempotent)
    detected  INTEGER NOT NULL,
    high_conf INTEGER NOT NULL,
    critical  INTEGER NOT NULL,
    signals   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kpi_samples_ts ON kpi_samples (ts DESC);

-- +goose Down
DROP TABLE IF EXISTS kpi_samples;
