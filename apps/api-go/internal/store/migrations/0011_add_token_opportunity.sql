-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_score      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_breakdown  TEXT             NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_scored_ts  BIGINT           NOT NULL DEFAULT 0;
-- DÜZELTME (brief/spec yanlış varsayıyordu): last_signal 0002'de YOK — 0002'nin tokens CREATE
-- TABLE'ında bu kolon yok; "last_signal" yalnızca 0001'deki AYRI `strategies` tablosunda var
-- (farklı domain: insan-okur relatif zaman string'i, ör. "43 dk önce"). tokens.last_signal
-- burada eklenir (plan boyunca 4 task bu kolonu tokens üzerinde varsayıyor).
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS last_signal            TEXT             NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_score;
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_confidence;
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_breakdown;
ALTER TABLE tokens DROP COLUMN IF EXISTS opportunity_scored_ts;
ALTER TABLE tokens DROP COLUMN IF EXISTS last_signal;
