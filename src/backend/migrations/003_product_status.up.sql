-- Per-search product visibility: hidden = user marked it irrelevant/
-- incorrect (never touched automatically); active = still found in the
-- most recent scrape for this search (cron flips this to false when a
-- listing disappears, and back to true if it reappears). Neither ever
-- deletes the underlying product or search_products row.
ALTER TABLE search_products ADD COLUMN is_hidden BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE search_products ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;
