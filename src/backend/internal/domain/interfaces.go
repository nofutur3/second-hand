package domain

import (
	"context"
)

// ShopAdapter defines the interface that each shop-specific adapter must implement
type ShopAdapter interface {
	// Name returns the name of the shop
	Name() string

	// Search performs a search on the shop and returns found products
	Search(ctx context.Context, keyword string) ([]Product, error)

	// SupportsSearch returns true if the shop supports searching
	SupportsSearch() bool
}

// Repository defines the interface for database operations
type Repository interface {
	// Search operations
	CreateSearch(ctx context.Context, keyword string) (*Search, error)
	GetSearchByID(ctx context.Context, id int64) (*Search, error)
	GetSearchByKeyword(ctx context.Context, keyword string) (*Search, error)
	GetAllSearches(ctx context.Context) ([]Search, error)
	UpdateSearchLastChecked(ctx context.Context, searchID int64) error
	SetGoodOfferConfig(ctx context.Context, searchID int64, maxPrice *float64, avgDiscountPct *float64) error
	DeleteSearch(ctx context.Context, searchID int64) error

	// Product operations
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error
	GetProductByURL(ctx context.Context, url string) (*Product, error)
	GetProductsBySearchID(ctx context.Context, searchID int64) ([]Product, error)

	// SearchProduct operations
	LinkProductToSearch(ctx context.Context, searchID, productID int64) error
	GetNewProductsSinceLastCheck(ctx context.Context, searchID int64) ([]Product, error)
	MarkProductsAsChecked(ctx context.Context, searchID int64, productIDs []int64) error
	MarkProductsInactive(ctx context.Context, searchID int64, productIDs []int64) error
}

// OutputFormatter defines the interface for different output formats
type OutputFormatter interface {
	FormatProducts(products []Product, verbose bool) (string, error)
	FormatDiff(diffs []ProductDiff, verbose bool) (string, error)
}
