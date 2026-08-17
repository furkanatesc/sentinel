-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_buys               INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_sells              INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_buyers             INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_sellers            INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator_holding_pct     DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_score      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_breakdown  TEXT             NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_scored_ts  BIGINT           NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_buys;
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_sells;
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_buyers;
ALTER TABLE tokens DROP COLUMN IF EXISTS txns_sellers;
ALTER TABLE tokens DROP COLUMN IF EXISTS creator_holding_pct;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_score;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_confidence;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_breakdown;
ALTER TABLE tokens DROP COLUMN IF EXISTS manipulation_scored_ts;
