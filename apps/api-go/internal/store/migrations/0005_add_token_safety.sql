-- +goose Up
-- Token Safety skoru (2a): safety_score kolonu 0002'de var; breakdown/risks/confidence/top10/scored_ts eklenir.
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS top10_holder_pct  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_breakdown  TEXT NOT NULL DEFAULT ''; -- JSON []ScoreBreakdownItem
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_risks      TEXT NOT NULL DEFAULT ''; -- JSON RiskGroups
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS safety_scored_ts  BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_scored_ts;
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_risks;
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_breakdown;
ALTER TABLE tokens DROP COLUMN IF EXISTS top10_holder_pct;
ALTER TABLE tokens DROP COLUMN IF EXISTS safety_confidence;
