-- Per-search "good offer" notification thresholds (both optional/independent;
-- neither set means the search never triggers a Telegram notification).
ALTER TABLE searches ADD COLUMN max_price NUMERIC(10, 2);
ALTER TABLE searches ADD COLUMN avg_discount_pct NUMERIC(5, 2);
