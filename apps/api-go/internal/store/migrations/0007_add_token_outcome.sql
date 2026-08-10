-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS peak_market_cap   DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS peak_liquidity    DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS outcome           TEXT NOT NULL DEFAULT 'active';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS max_drawdown_pct  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS liquidity_status  TEXT NOT NULL DEFAULT 'unlocked';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS outcome_scored_ts BIGINT NOT NULL DEFAULT 0;
-- Peak seed: mevcut token'lar için tepeyi güncelden başlat (gerçek tarihsel tepe kayıp;
-- peak ≥ güncel garantisi honest-conservative — drawdown migration'dan itibaren ölçülür).
UPDATE tokens SET peak_market_cap = market_cap_usd WHERE peak_market_cap = 0 AND market_cap_usd > 0;
UPDATE tokens SET peak_liquidity  = liquidity      WHERE peak_liquidity  = 0 AND liquidity > 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS outcome_scored_ts;
ALTER TABLE tokens DROP COLUMN IF EXISTS liquidity_status;
ALTER TABLE tokens DROP COLUMN IF EXISTS max_drawdown_pct;
ALTER TABLE tokens DROP COLUMN IF EXISTS outcome;
ALTER TABLE tokens DROP COLUMN IF EXISTS peak_liquidity;
ALTER TABLE tokens DROP COLUMN IF EXISTS peak_market_cap;
