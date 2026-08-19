-- eBay-specific pricing detail: shipping cost resolved for a fixed
-- delivery destination (see EbayConfig.ShipToCountry/ShipToPostalCode),
-- and live bid count for auction listings. Both nullable/zero for every
-- non-eBay shop.
ALTER TABLE products ADD COLUMN shipping_cost NUMERIC(10, 2);
ALTER TABLE products ADD COLUMN bid_count INTEGER NOT NULL DEFAULT 0;
