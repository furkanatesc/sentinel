-- +goose Up
CREATE TABLE IF NOT EXISTS alert_events (
    id       TEXT PRIMARY KEY,
    rule_id  TEXT NOT NULL DEFAULT '',
    type     TEXT NOT NULL DEFAULT '',
    token    TEXT NOT NULL DEFAULT '',
    detail   TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'info',
    time     TEXT NOT NULL DEFAULT '',
    ts       BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_alert_events_ts ON alert_events (ts DESC);
CREATE TABLE IF NOT EXISTS alert_eval_meta (
    id           TEXT PRIMARY KEY,   -- 'default'
    watermark_ts BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS alert_events;
DROP TABLE IF EXISTS alert_eval_meta;
