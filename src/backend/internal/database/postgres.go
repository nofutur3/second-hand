package database

import (
	"context"
	"fmt"
	"secondHand/src/backend/internal/config"
	"secondHand/src/backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements the Repository interface using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(cfg *config.DatabaseConfig) (*PostgresRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

// Close closes the database connection pool
func (r *PostgresRepository) Close() {
	r.pool.Close()
}

// CreateSearch creates a new search or returns existing one
func (r *PostgresRepository) CreateSearch(ctx context.Context, keyword string) (*domain.Search, error) {
	// Try to get existing search first
	existing, err := r.GetSearchByKeyword(ctx, keyword)
	if err == nil && existing != nil {
		return existing, nil
	}

	query := `
		INSERT INTO searches (keyword, created_at)
		VALUES ($1, $2)
		ON CONFLICT (keyword) DO UPDATE SET keyword = EXCLUDED.keyword
		RETURNING id, keyword, created_at, last_checked_at, max_price, avg_discount_pct
	`

	search := &domain.Search{}
	now := time.Now()
	err = r.pool.QueryRow(ctx, query, keyword, now).Scan(
		&search.ID,
		&search.Keyword,
		&search.CreatedAt,
		&search.LastCheckedAt,
		&search.MaxPrice,
		&search.AvgDiscountPct,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create search: %w", err)
	}

	return search, nil
}

// GetSearchByID retrieves a search by ID
func (r *PostgresRepository) GetSearchByID(ctx context.Context, id int64) (*domain.Search, error) {
	query := `SELECT id, keyword, created_at, last_checked_at, max_price, avg_discount_pct FROM searches WHERE id = $1`

	search := &domain.Search{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&search.ID,
		&search.Keyword,
		&search.CreatedAt,
		&search.LastCheckedAt,
		&search.MaxPrice,
		&search.AvgDiscountPct,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get search: %w", err)
	}

	return search, nil
}

// GetSearchByKeyword retrieves a search by keyword
func (r *PostgresRepository) GetSearchByKeyword(ctx context.Context, keyword string) (*domain.Search, error) {
	query := `SELECT id, keyword, created_at, last_checked_at, max_price, avg_discount_pct FROM searches WHERE keyword = $1`

	search := &domain.Search{}
	err := r.pool.QueryRow(ctx, query, keyword).Scan(
		&search.ID,
		&search.Keyword,
		&search.CreatedAt,
		&search.LastCheckedAt,
		&search.MaxPrice,
		&search.AvgDiscountPct,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get search: %w", err)
	}

	return search, nil
}

// GetAllSearches retrieves all searches
func (r *PostgresRepository) GetAllSearches(ctx context.Context) ([]domain.Search, error) {
	query := `SELECT id, keyword, created_at, last_checked_at, max_price, avg_discount_pct FROM searches ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get searches: %w", err)
	}
	defer rows.Close()

	var searches []domain.Search
	for rows.Next() {
		var search domain.Search
		if err := rows.Scan(&search.ID, &search.Keyword, &search.CreatedAt, &search.LastCheckedAt, &search.MaxPrice, &search.AvgDiscountPct); err != nil {
			return nil, fmt.Errorf("failed to scan search: %w", err)
		}
		searches = append(searches, search)
	}

	return searches, nil
}

// UpdateSearchLastChecked updates the last_checked_at timestamp
func (r *PostgresRepository) UpdateSearchLastChecked(ctx context.Context, searchID int64) error {
	query := `UPDATE searches SET last_checked_at = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, time.Now(), searchID)
	if err != nil {
		return fmt.Errorf("failed to update search: %w", err)
	}
	return nil
}

// SetGoodOfferConfig sets the per-search "good offer" notification thresholds
func (r *PostgresRepository) SetGoodOfferConfig(ctx context.Context, searchID int64, maxPrice *float64, avgDiscountPct *float64) error {
	query := `UPDATE searches SET max_price = $1, avg_discount_pct = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, maxPrice, avgDiscountPct, searchID)
	if err != nil {
		return fmt.Errorf("failed to set good offer config: %w", err)
	}
	return nil
}

// CreateProduct creates a new product
func (r *PostgresRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	query := `
		INSERT INTO products (shop_source, title, description, price, currency, auction_type, 
			ending_time, condition, url, image_url, location, seller_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	err := r.pool.QueryRow(ctx, query,
		product.ShopSource,
		product.Title,
		product.Description,
		product.Price,
		product.Currency,
		product.AuctionType,
		product.EndingTime,
		product.Condition,
		product.URL,
		product.ImageURL,
		product.Location,
		product.SellerName,
		now,
		now,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

// UpdateProduct updates an existing product
func (r *PostgresRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	query := `
		UPDATE products SET
			title = $1,
			description = $2,
			price = $3,
			currency = $4,
			auction_type = $5,
			ending_time = $6,
			condition = $7,
			image_url = $8,
			location = $9,
			seller_name = $10,
			updated_at = $11
		WHERE id = $12
	`

	_, err := r.pool.Exec(ctx, query,
		product.Title,
		product.Description,
		product.Price,
		product.Currency,
		product.AuctionType,
		product.EndingTime,
		product.Condition,
		product.ImageURL,
		product.Location,
		product.SellerName,
		time.Now(),
		product.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	return nil
}

// GetProductByURL retrieves a product by URL
func (r *PostgresRepository) GetProductByURL(ctx context.Context, url string) (*domain.Product, error) {
	query := `
		SELECT id, shop_source, title, description, price, currency, auction_type,
			ending_time, condition, url, image_url, location, seller_name, created_at, updated_at
		FROM products WHERE url = $1
	`

	product := &domain.Product{}
	err := r.pool.QueryRow(ctx, query, url).Scan(
		&product.ID,
		&product.ShopSource,
		&product.Title,
		&product.Description,
		&product.Price,
		&product.Currency,
		&product.AuctionType,
		&product.EndingTime,
		&product.Condition,
		&product.URL,
		&product.ImageURL,
		&product.Location,
		&product.SellerName,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return product, nil
}

// GetProductsBySearchID retrieves products for a specific search
func (r *PostgresRepository) GetProductsBySearchID(ctx context.Context, searchID int64) ([]domain.Product, error) {
	query := `
		SELECT p.id, p.shop_source, p.title, p.description, p.price, p.currency, p.auction_type,
			p.ending_time, p.condition, p.url, p.image_url, p.location, p.seller_name, 
			p.created_at, p.updated_at
		FROM products p
		INNER JOIN search_products sp ON p.id = sp.product_id
		WHERE sp.search_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, searchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var product domain.Product
		if err := rows.Scan(
			&product.ID,
			&product.ShopSource,
			&product.Title,
			&product.Description,
			&product.Price,
			&product.Currency,
			&product.AuctionType,
			&product.EndingTime,
			&product.Condition,
			&product.URL,
			&product.ImageURL,
			&product.Location,
			&product.SellerName,
			&product.CreatedAt,
			&product.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	return products, nil
}

// LinkProductToSearch links a product to a search
func (r *PostgresRepository) LinkProductToSearch(ctx context.Context, searchID, productID int64) error {
	query := `
		INSERT INTO search_products (search_id, product_id, found_at, is_new)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (search_id, product_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, searchID, productID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to link product to search: %w", err)
	}

	return nil
}

// GetNewProductsSinceLastCheck retrieves new products since last check
func (r *PostgresRepository) GetNewProductsSinceLastCheck(ctx context.Context, searchID int64) ([]domain.Product, error) {
	query := `
		SELECT p.id, p.shop_source, p.title, p.description, p.price, p.currency, p.auction_type,
			p.ending_time, p.condition, p.url, p.image_url, p.location, p.seller_name, 
			p.created_at, p.updated_at
		FROM products p
		INNER JOIN search_products sp ON p.id = sp.product_id
		WHERE sp.search_id = $1 AND sp.is_new = TRUE
		ORDER BY p.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, searchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var product domain.Product
		if err := rows.Scan(
			&product.ID,
			&product.ShopSource,
			&product.Title,
			&product.Description,
			&product.Price,
			&product.Currency,
			&product.AuctionType,
			&product.EndingTime,
			&product.Condition,
			&product.URL,
			&product.ImageURL,
			&product.Location,
			&product.SellerName,
			&product.CreatedAt,
			&product.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	return products, nil
}

// MarkProductsAsChecked marks products as checked (not new)
func (r *PostgresRepository) MarkProductsAsChecked(ctx context.Context, searchID int64, productIDs []int64) error {
	if len(productIDs) == 0 {
		return nil
	}

	query := `
		UPDATE search_products
		SET is_new = FALSE
		WHERE search_id = $1 AND product_id = ANY($2)
	`

	_, err := r.pool.Exec(ctx, query, searchID, productIDs)
	if err != nil {
		return fmt.Errorf("failed to mark products as checked: %w", err)
	}

	return nil
}
