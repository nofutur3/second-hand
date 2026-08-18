package service

import (
	"context"
	"fmt"
	domain2 "secondHand/internal/domain"
)

// DiffService handles diff generation between searches
type DiffService struct {
	repo domain2.Repository
}

// NewDiffService creates a new diff service
func NewDiffService(repo domain2.Repository) *DiffService {
	return &DiffService{repo: repo}
}

// GenerateDiff compares previousProducts (the search's state as of before
// this run's scrape) against currentProducts (what this run just found).
// previousProducts must be fetched by the caller *before* running the
// scrape that produced currentProducts - GetDiffForAllSearches's
// SearchService.Search call links every found product into the same
// search_products rows this would otherwise read, which would make every
// current product look "previously seen" and erase DiffTypeNew entirely.
func (s *DiffService) GenerateDiff(ctx context.Context, searchID int64, previousProducts, currentProducts []domain2.Product) ([]domain2.ProductDiff, error) {
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
	var removedIDs []int64
	for _, previous := range previousProducts {
		if _, exists := currentMap[previous.URL]; !exists {
			diffs = append(diffs, domain2.ProductDiff{
				Product:      previous,
				DiffType:     domain2.DiffTypeRemoved,
				OldPrice:     &previous.Price,
				PriceChanged: false,
			})
			removedIDs = append(removedIDs, previous.ID)
		}
	}

	// Flag them as no longer listed (not deleted - just hidden from the
	// default view) so the frontend can stop showing delisted offers
	// without a human having to notice and remove them by hand.
	if len(removedIDs) > 0 {
		if err := s.repo.MarkProductsInactive(ctx, searchID, removedIDs); err != nil {
			return nil, fmt.Errorf("failed to mark removed products inactive: %w", err)
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

		// Snapshot state before the scrape below links newly-found
		// products into these same rows (see GenerateDiff's comment).
		previousProducts, err := s.repo.GetProductsBySearchID(ctx, search.ID)
		if err != nil {
			fmt.Printf("Failed to get previous products for '%s': %v\n", search.Keyword, err)
			continue
		}

		// Perform new search
		currentProducts, err := searchService.Search(ctx, search.Keyword)
		if err != nil {
			fmt.Printf("Failed to search for '%s': %v\n", search.Keyword, err)
			continue
		}

		// Generate diff
		diffs, err := s.GenerateDiff(ctx, search.ID, previousProducts, currentProducts)
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
