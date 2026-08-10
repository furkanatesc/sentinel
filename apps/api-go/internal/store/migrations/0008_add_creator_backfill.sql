-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator_backfill_ts BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS creator_backfill_ts;
