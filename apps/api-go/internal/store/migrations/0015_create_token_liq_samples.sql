-- +goose Up
CREATE TABLE IF NOT EXISTS token_liq_samples (
    mint      TEXT NOT NULL,
    ts        BIGINT NOT NULL,
    liquidity DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (mint, ts)
);
-- (mint, ts) PK btree → WHERE mint=$1 ORDER BY ts hem yazma-conflict hem okuma için yeter; ek index yok.

-- +goose Down
DROP TABLE IF EXISTS token_liq_samples;
