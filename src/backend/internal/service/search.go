package service

import (
	"context"
	"fmt"
	"secondHand/src/backend/internal/adapter"
	domain2 "secondHand/src/backend/internal/domain"
	"sync"
)

// SearchService handles search operations
type SearchService struct {
	repo     domain2.Repository
	registry *adapter.Registry
}

// NewSearchService creates a new search service
func NewSearchService(repo domain2.Repository, registry *adapter.Registry) *SearchService {
	return &SearchService{
		repo:     repo,
		registry: registry,
	}
}

// SearchResult represents results from a single adapter
type SearchResult struct {
	AdapterName string
	Products    []domain2.Product
	Error       error
}

// Search performs a search across all adapters
func (s *SearchService) Search(ctx context.Context, keyword string) ([]domain2.Product, error) {
	return s.SearchWithFilter(ctx, keyword, "")
}

// SearchWithFilter performs a search, optionally filtering by adapter name
func (s *SearchService) SearchWithFilter(ctx context.Context, keyword string, adapterFilter string) ([]domain2.Product, error) {
	// Create or get search record
	search, err := s.repo.CreateSearch(ctx, keyword)
	if err != nil {
		return nil, fmt.Errorf("failed to create search: %w", err)
	}

	// Get adapters (filter if requested)
	adapters := s.registry.GetAllAdapters()
	if adapterFilter != "" {
		// Filter to only the requested adapter
		var filtered []domain2.ShopAdapter
		for _, adapter := range adapters {
			if adapter.Name() == adapterFilter {
				filtered = append(filtered, adapter)
				break
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("adapter not found: %s", adapterFilter)
		}
		adapters = filtered
	}

	// Search across selected adapters in parallel
	results := make(chan SearchResult, len(adapters))
	var wg sync.WaitGroup

	for _, adapter := range adapters {
		wg.Add(1)
		go func(a domain2.ShopAdapter) {
			defer wg.Done()

			products, err := a.Search(ctx, keyword)
			results <- SearchResult{
				AdapterName: a.Name(),
				Products:    products,
				Error:       err,
			}
		}(adapter)
	}

	// Wait for all adapters to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allProducts []domain2.Product
	var errors []error

	for result := range results {
		if result.Error != nil {
			fmt.Printf("Error from %s: %v\n", result.AdapterName, result.Error)
			errors = append(errors, result.Error)
			continue
		}

		fmt.Printf("Found %d products from %s\n", len(result.Products), result.AdapterName)

		// Save products to database
		for i := range result.Products {
			product := &result.Products[i]

			// Check if product already exists
			existing, err := s.repo.GetProductByURL(ctx, product.URL)
			if err == nil && existing != nil {
				// Product exists, check if price changed
				if existing.Price != product.Price {
					product.ID = existing.ID
					if err := s.repo.UpdateProduct(ctx, product); err != nil {
						fmt.Printf("Failed to update product %s: %v\n", product.URL, err)
					}
				}
				product.ID = existing.ID
			} else {
				// New product
				if err := s.repo.CreateProduct(ctx, product); err != nil {
					fmt.Printf("Failed to create product %s: %v\n", product.URL, err)
					continue
				}
			}

			// Link product to search
			if err := s.repo.LinkProductToSearch(ctx, search.ID, product.ID); err != nil {
				fmt.Printf("Failed to link product to search: %v\n", err)
			}

			allProducts = append(allProducts, *product)
		}
	}

	// Update search last checked timestamp
	if err := s.repo.UpdateSearchLastChecked(ctx, search.ID); err != nil {
		fmt.Printf("Failed to update search last checked: %v\n", err)
	}

	if len(allProducts) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("all adapters failed: %v", errors)
	}

	return allProducts, nil
}

// GetSearchProducts retrieves products for a saved search
func (s *SearchService) GetSearchProducts(ctx context.Context, keyword string) ([]domain2.Product, error) {
	search, err := s.repo.GetSearchByKeyword(ctx, keyword)
	if err != nil {
		return nil, fmt.Errorf("search not found: %w", err)
	}

	return s.repo.GetProductsBySearchID(ctx, search.ID)
}
