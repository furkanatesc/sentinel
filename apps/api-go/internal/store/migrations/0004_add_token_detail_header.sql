-- +goose Up
-- Detail header alanları: enrichment'ta persist edilir → getToken canlı GeckoTerminal
-- çağrısı yapmadan header'ı DB'den sunar (paylaşımlı-IP throttle'a dayanıklı).
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS price_change_h24 DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS market_cap_usd   DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS vol24h           DOUBLE PRECISION NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS vol24h;
ALTER TABLE tokens DROP COLUMN IF EXISTS market_cap_usd;
ALTER TABLE tokens DROP COLUMN IF EXISTS price_change_h24;
