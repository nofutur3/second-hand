package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	domain2 "secondHand/src/backend/internal/domain"
)

// fakeRepo is an in-memory domain2.Repository for unit-testing
// SearchService/DiffService without a real database.
type fakeRepo struct {
	mu sync.Mutex

	searches          map[int64]*domain2.Search
	searchIDByKeyword map[string]int64
	nextSearchID      int64

	products       map[int64]*domain2.Product
	productIDByURL map[string]int64
	nextProductID  int64

	links       map[int64]map[int64]bool // searchID -> productID -> isNew
	inactiveIDs map[int64]map[int64]bool // searchID -> productID -> true

	createProductErr         error
	getProductsBySearchIDErr error
	markProductsInactiveErr  error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		searches:          make(map[int64]*domain2.Search),
		searchIDByKeyword: make(map[string]int64),
		products:          make(map[int64]*domain2.Product),
		productIDByURL:    make(map[string]int64),
		links:             make(map[int64]map[int64]bool),
		inactiveIDs:       make(map[int64]map[int64]bool),
	}
}

func (f *fakeRepo) CreateSearch(ctx context.Context, keyword string) (*domain2.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if id, ok := f.searchIDByKeyword[keyword]; ok {
		return f.searches[id], nil
	}

	f.nextSearchID++
	search := &domain2.Search{ID: f.nextSearchID, Keyword: keyword}
	f.searches[search.ID] = search
	f.searchIDByKeyword[keyword] = search.ID
	return search, nil
}

func (f *fakeRepo) GetSearchByID(ctx context.Context, id int64) (*domain2.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	search, ok := f.searches[id]
	if !ok {
		return nil, fmt.Errorf("search not found: %d", id)
	}
	return search, nil
}

func (f *fakeRepo) GetSearchByKeyword(ctx context.Context, keyword string) (*domain2.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id, ok := f.searchIDByKeyword[keyword]
	if !ok {
		return nil, fmt.Errorf("search not found: %s", keyword)
	}
	return f.searches[id], nil
}

func (f *fakeRepo) GetAllSearches(ctx context.Context) ([]domain2.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	searches := make([]domain2.Search, 0, len(f.searches))
	for _, s := range f.searches {
		searches = append(searches, *s)
	}
	return searches, nil
}

func (f *fakeRepo) UpdateSearchLastChecked(ctx context.Context, searchID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	search, ok := f.searches[searchID]
	if !ok {
		return fmt.Errorf("search not found: %d", searchID)
	}
	now := time.Now()
	search.LastCheckedAt = &now
	return nil
}

func (f *fakeRepo) SetGoodOfferConfig(ctx context.Context, searchID int64, maxPrice *float64, avgDiscountPct *float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	search, ok := f.searches[searchID]
	if !ok {
		return fmt.Errorf("search not found: %d", searchID)
	}
	search.MaxPrice = maxPrice
	search.AvgDiscountPct = avgDiscountPct
	return nil
}

func (f *fakeRepo) DeleteSearch(ctx context.Context, searchID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	search, ok := f.searches[searchID]
	if !ok {
		return fmt.Errorf("search not found: %d", searchID)
	}
	delete(f.searches, searchID)
	delete(f.searchIDByKeyword, search.Keyword)
	delete(f.links, searchID)
	return nil
}

func (f *fakeRepo) CreateProduct(ctx context.Context, product *domain2.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.createProductErr != nil {
		return f.createProductErr
	}

	f.nextProductID++
	product.ID = f.nextProductID
	stored := *product
	f.products[product.ID] = &stored
	f.productIDByURL[product.URL] = product.ID
	return nil
}

func (f *fakeRepo) UpdateProduct(ctx context.Context, product *domain2.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.products[product.ID]; !ok {
		return fmt.Errorf("product not found: %d", product.ID)
	}
	stored := *product
	f.products[product.ID] = &stored
	return nil
}

func (f *fakeRepo) GetProductByURL(ctx context.Context, url string) (*domain2.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id, ok := f.productIDByURL[url]
	if !ok {
		return nil, fmt.Errorf("product not found: %s", url)
	}
	p := *f.products[id]
	return &p, nil
}

func (f *fakeRepo) GetProductsBySearchID(ctx context.Context, searchID int64) ([]domain2.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.getProductsBySearchIDErr != nil {
		return nil, f.getProductsBySearchIDErr
	}

	var products []domain2.Product
	for productID := range f.links[searchID] {
		products = append(products, *f.products[productID])
	}
	return products, nil
}

func (f *fakeRepo) LinkProductToSearch(ctx context.Context, searchID, productID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.links[searchID] == nil {
		f.links[searchID] = make(map[int64]bool)
	}
	f.links[searchID][productID] = true
	if f.inactiveIDs[searchID] != nil {
		delete(f.inactiveIDs[searchID], productID)
	}
	return nil
}

func (f *fakeRepo) GetNewProductsSinceLastCheck(ctx context.Context, searchID int64) ([]domain2.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var products []domain2.Product
	for productID, isNew := range f.links[searchID] {
		if isNew {
			products = append(products, *f.products[productID])
		}
	}
	return products, nil
}

func (f *fakeRepo) MarkProductsAsChecked(ctx context.Context, searchID int64, productIDs []int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, id := range productIDs {
		if f.links[searchID] != nil {
			f.links[searchID][id] = false
		}
	}
	return nil
}

func (f *fakeRepo) MarkProductsInactive(ctx context.Context, searchID int64, productIDs []int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.markProductsInactiveErr != nil {
		return f.markProductsInactiveErr
	}

	if f.inactiveIDs[searchID] == nil {
		f.inactiveIDs[searchID] = make(map[int64]bool)
	}
	for _, id := range productIDs {
		f.inactiveIDs[searchID][id] = true
	}
	return nil
}

// fakeAdapter is a deterministic domain2.ShopAdapter for tests.
type fakeAdapter struct {
	name     string
	products []domain2.Product
	err      error
}

func (a *fakeAdapter) Name() string         { return a.name }
func (a *fakeAdapter) SupportsSearch() bool { return true }
func (a *fakeAdapter) Search(ctx context.Context, keyword string) ([]domain2.Product, error) {
	return a.products, a.err
}
