-- Drop triggers
DROP TRIGGER IF EXISTS update_products_updated_at ON products;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_search_products_is_new;
DROP INDEX IF EXISTS idx_search_products_search_id;
DROP INDEX IF EXISTS idx_products_price;
DROP INDEX IF EXISTS idx_products_created_at;
DROP INDEX IF EXISTS idx_products_shop_source;

-- Drop tables
DROP TABLE IF EXISTS search_products;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS searches;
