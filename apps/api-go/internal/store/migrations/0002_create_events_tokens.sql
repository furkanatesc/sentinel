-- +goose Up
CREATE TABLE IF NOT EXISTS tokens (
    mint          TEXT PRIMARY KEY,
    symbol        TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL DEFAULT '',
    launchpad     TEXT NOT NULL DEFAULT '',
    first_seen_ts BIGINT NOT NULL,
    -- enrichment/scoring (Slice 1b / Alt-proje 2) — 1a'da nötr:
    price         DOUBLE PRECISION NOT NULL DEFAULT 0,
    liquidity     DOUBLE PRECISION NOT NULL DEFAULT 0,
    vol5m         DOUBLE PRECISION NOT NULL DEFAULT 0,
    holders       INTEGER NOT NULL DEFAULT 0,
    creator_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    safety_score  DOUBLE PRECISION NOT NULL DEFAULT 0,
    momentum      DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS events (
    id                TEXT PRIMARY KEY,
    signature         TEXT NOT NULL,
    slot              BIGINT NOT NULL,
    type              TEXT NOT NULL,
    mint              TEXT NOT NULL,
    symbol            TEXT NOT NULL DEFAULT '',
    launchpad         TEXT NOT NULL DEFAULT '',
    dex               TEXT NOT NULL DEFAULT '',
    liquidity         DOUBLE PRECISION NOT NULL DEFAULT 0,
    creator_score     DOUBLE PRECISION NOT NULL DEFAULT 0,
    risk_level        TEXT NOT NULL DEFAULT 'medium',
    token_age_seconds BIGINT NOT NULL DEFAULT 0,
    volume5m          DOUBLE PRECISION NOT NULL DEFAULT 0,
    holder_growth_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
    severity          TEXT NOT NULL DEFAULT 'info',
    detail            TEXT NOT NULL DEFAULT '',
    ts                BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events (ts DESC);

-- +goose Down
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS tokens;
