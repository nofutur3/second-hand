package service

import (
	"context"
	"fmt"
	domain2 "secondHand/src/backend/internal/domain"
)

// DiffService handles diff generation between searches
type DiffService struct {
	repo domain2.Repository
}

// NewDiffService creates a new diff service
func NewDiffService(repo domain2.Repository) *DiffService {
	return &DiffService{repo: repo}
}

// GenerateDiff generates a diff for a search
func (s *DiffService) GenerateDiff(ctx context.Context, searchID int64, currentProducts []domain2.Product) ([]domain2.ProductDiff, error) {
	// Get previous products
	previousProducts, err := s.repo.GetProductsBySearchID(ctx, searchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous products: %w", err)
	}

	// Create maps for efficient lookup
	previousMap := make(map[string]domain2.Product)
	for _, p := range previousProducts {
		previousMap[p.URL] = p
	}

	currentMap := make(map[string]domain2.Product)
	for _, p := range currentProducts {
		currentMap[p.URL] = p
	}

	var diffs []domain2.ProductDiff

	// Find new and updated products
	for _, current := range currentProducts {
		if previous, exists := previousMap[current.URL]; exists {
			// Product exists, check for price changes
			if current.Price != previous.Price {
				diffType := domain2.DiffTypePriceDown
				if current.Price > previous.Price {
					diffType = domain2.DiffTypePriceUp
				}

				diffs = append(diffs, domain2.ProductDiff{
					Product:      current,
					DiffType:     diffType,
					OldPrice:     &previous.Price,
					NewPrice:     &current.Price,
					PriceChanged: true,
				})
			}
		} else {
			// New product
			diffs = append(diffs, domain2.ProductDiff{
				Product:      current,
				DiffType:     domain2.DiffTypeNew,
				NewPrice:     &current.Price,
				PriceChanged: false,
			})
		}
	}

	// Find removed products
	for _, previous := range previousProducts {
		if _, exists := currentMap[previous.URL]; !exists {
			diffs = append(diffs, domain2.ProductDiff{
				Product:      previous,
				DiffType:     domain2.DiffTypeRemoved,
				OldPrice:     &previous.Price,
				PriceChanged: false,
			})
		}
	}

	return diffs, nil
}

// GetDiffForAllSearches generates diffs for all saved searches
func (s *DiffService) GetDiffForAllSearches(ctx context.Context, searchService *SearchService) (map[string][]domain2.ProductDiff, error) {
	searches, err := s.repo.GetAllSearches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get searches: %w", err)
	}

	results := make(map[string][]domain2.ProductDiff)

	for _, search := range searches {
		fmt.Printf("Checking search: %s\n", search.Keyword)

		// Perform new search
		currentProducts, err := searchService.Search(ctx, search.Keyword)
		if err != nil {
			fmt.Printf("Failed to search for '%s': %v\n", search.Keyword, err)
			continue
		}

		// Generate diff
		diffs, err := s.GenerateDiff(ctx, search.ID, currentProducts)
		if err != nil {
			fmt.Printf("Failed to generate diff for '%s': %v\n", search.Keyword, err)
			continue
		}

		if len(diffs) > 0 {
			results[search.Keyword] = diffs
		}
	}

	return results, nil
}
