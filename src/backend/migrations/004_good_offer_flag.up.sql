-- Per-search-product "good offer" flag: set when EvaluateGoodOffer matches
-- during cron's diff check (mirrors is_hidden/is_active from 003) so the
-- frontend can show which listings triggered a Telegram alert.
ALTER TABLE search_products ADD COLUMN is_good_offer BOOLEAN NOT NULL DEFAULT FALSE;
