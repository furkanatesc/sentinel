-- +goose Up
CREATE TABLE IF NOT EXISTS notification_config (
    id             TEXT PRIMARY KEY,       -- tek satır: 'default'
    channel        TEXT NOT NULL DEFAULT '',
    min_severity   TEXT NOT NULL DEFAULT 'info',
    quiet_start    TEXT NOT NULL DEFAULT '',
    quiet_end      TEXT NOT NULL DEFAULT '',
    quiet_enabled  BOOLEAN NOT NULL DEFAULT false,
    templates      TEXT NOT NULL DEFAULT '[]',  -- JSON []NotificationTemplate
    trade_approval BOOLEAN NOT NULL DEFAULT false
);

-- +goose Down
DROP TABLE IF EXISTS notification_config;
