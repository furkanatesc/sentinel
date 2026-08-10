-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator TEXT NOT NULL DEFAULT '';
-- Agrega (GROUP BY creator) + filtre (creator<>'') için partial index.
CREATE INDEX IF NOT EXISTS idx_tokens_creator ON tokens (creator) WHERE creator <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_tokens_creator;
ALTER TABLE tokens DROP COLUMN IF EXISTS creator;
