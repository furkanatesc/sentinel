-- +goose Up
-- 2e-2 controls_authority: mint/freeze authority pubkey'i token'a persist edilir (safety piggyback).
-- '' = iptal edilmiş VEYA henüz skorlanmamış (ikisi de küme adayı değil).
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS mint_authority   TEXT NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS freeze_authority TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_tokens_mint_authority   ON tokens (mint_authority)   WHERE mint_authority   <> '';
CREATE INDEX IF NOT EXISTS idx_tokens_freeze_authority ON tokens (freeze_authority) WHERE freeze_authority <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_tokens_freeze_authority;
DROP INDEX IF EXISTS idx_tokens_mint_authority;
ALTER TABLE tokens DROP COLUMN IF EXISTS freeze_authority;
ALTER TABLE tokens DROP COLUMN IF EXISTS mint_authority;
