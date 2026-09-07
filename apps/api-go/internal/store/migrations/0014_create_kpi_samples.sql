-- +goose Up
CREATE TABLE IF NOT EXISTS kpi_samples (
    ts        BIGINT PRIMARY KEY,   -- unix saniye (örnek anı; PK → aynı ts idempotent)
    detected  INTEGER NOT NULL,
    high_conf INTEGER NOT NULL,
    critical  INTEGER NOT NULL,
    signals   INTEGER NOT NULL
);
-- ts PRIMARY KEY btree'si ORDER BY ts DESC LIMIT için geriye taranabilir → ayrı index gereksiz.

-- +goose Down
DROP TABLE IF EXISTS kpi_samples;
