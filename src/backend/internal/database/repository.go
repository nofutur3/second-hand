package database

import (
	"context"
	"secondHand/src/backend/internal/domain"
)

// Repository defines the interface for data access
type Repository interface {
	// Search operations
	CreateSearch(ctx context.Context, keyword string) (*domain.Search, error)
	GetSearchByID(ctx context.Context, id int64) (*domain.Search, error)
	GetSearchByKeyword(ctx context.Context, keyword string) (*domain.Search, error)
	GetAllSearches(ctx context.Context) ([]domain.Search, error)
	UpdateSearchLastChecked(ctx context.Context, searchID int64) error

	// Product operations
	CreateProduct(ctx context.Context, product *domain.Product) error
	UpdateProduct(ctx context.Context, product *domain.Product) error
	GetProductByURL(ctx context.Context, url string) (*domain.Product, error)
	GetProductsBySearchID(ctx context.Context, searchID int64) ([]domain.Product, error)

	// Search-Product relationship
	LinkProductToSearch(ctx context.Context, searchID, productID int64) error

	// Lifecycle
	Close()
}
