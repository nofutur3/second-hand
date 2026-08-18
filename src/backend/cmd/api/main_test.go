package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"secondHand/internal/adapter"
	"secondHand/internal/config"
	database2 "secondHand/internal/database"
	"secondHand/internal/domain"
	"secondHand/internal/service"
)

// fakeRepository is an in-memory stand-in for database2.Repository (and
// domain.Repository, so it can back a real *service.SearchService too),
// used to exercise cmd/api's HTTP handlers without a real database.
type fakeRepository struct {
	mu sync.Mutex

	searches       map[int64]*domain.Search
	searchesByKey  map[string]int64
	nextSearchID   int64
	products       map[int64]*domain.Product
	nextProductID  int64
	searchProducts map[int64]map[int64]*spStatus // searchID -> productID -> status

	// error injection
	createSearchErr             error
	getAllSearchesWithCountsErr error
	getSearchByIDErr            error
	getProductsWithStatusErr    error
	deleteSearchErr             error
	setProductHiddenErr         error
}

type spStatus struct {
	isHidden bool
	isActive bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		searches:       make(map[int64]*domain.Search),
		searchesByKey:  make(map[string]int64),
		products:       make(map[int64]*domain.Product),
		searchProducts: make(map[int64]map[int64]*spStatus),
	}
}

func (f *fakeRepository) CreateSearch(ctx context.Context, keyword string) (*domain.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.createSearchErr != nil {
		return nil, f.createSearchErr
	}

	if id, ok := f.searchesByKey[keyword]; ok {
		s := *f.searches[id]
		return &s, nil
	}

	f.nextSearchID++
	id := f.nextSearchID
	s := &domain.Search{ID: id, Keyword: keyword, CreatedAt: time.Now()}
	f.searches[id] = s
	f.searchesByKey[keyword] = id
	f.searchProducts[id] = make(map[int64]*spStatus)

	cp := *s
	return &cp, nil
}

func (f *fakeRepository) GetSearchByID(ctx context.Context, id int64) (*domain.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.getSearchByIDErr != nil {
		return nil, f.getSearchByIDErr
	}
	s, ok := f.searches[id]
	if !ok {
		return nil, database2.ErrSearchNotFound
	}
	cp := *s
	return &cp, nil
}

func (f *fakeRepository) GetSearchByKeyword(ctx context.Context, keyword string) (*domain.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id, ok := f.searchesByKey[keyword]
	if !ok {
		return nil, database2.ErrSearchNotFound
	}
	cp := *f.searches[id]
	return &cp, nil
}

func (f *fakeRepository) GetAllSearches(ctx context.Context) ([]domain.Search, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]domain.Search, 0, len(f.searches))
	for _, s := range f.searches {
		out = append(out, *s)
	}
	return out, nil
}

func (f *fakeRepository) GetAllSearchesWithCounts(ctx context.Context) ([]database2.SearchWithCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.getAllSearchesWithCountsErr != nil {
		return nil, f.getAllSearchesWithCountsErr
	}

	out := make([]database2.SearchWithCount, 0, len(f.searches))
	for _, s := range f.searches {
		count := 0
		for _, st := range f.searchProducts[s.ID] {
			if !st.isHidden && st.isActive {
				count++
			}
		}
		out = append(out, database2.SearchWithCount{Search: *s, ProductCount: count})
	}
	return out, nil
}

func (f *fakeRepository) UpdateSearchLastChecked(ctx context.Context, searchID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.searches[searchID]; ok {
		now := time.Now()
		s.LastCheckedAt = &now
	}
	return nil
}

func (f *fakeRepository) SetGoodOfferConfig(ctx context.Context, searchID int64, maxPrice *float64, avgDiscountPct *float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.searches[searchID]; ok {
		s.MaxPrice = maxPrice
		s.AvgDiscountPct = avgDiscountPct
	}
	return nil
}

func (f *fakeRepository) DeleteSearch(ctx context.Context, searchID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.deleteSearchErr != nil {
		return f.deleteSearchErr
	}
	s, ok := f.searches[searchID]
	if !ok {
		return database2.ErrSearchNotFound
	}
	delete(f.searches, searchID)
	delete(f.searchesByKey, s.Keyword)
	delete(f.searchProducts, searchID)
	return nil
}

func (f *fakeRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextProductID++
	product.ID = f.nextProductID
	product.CreatedAt = time.Now()
	product.UpdatedAt = product.CreatedAt
	cp := *product
	f.products[product.ID] = &cp
	return nil
}

func (f *fakeRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.products[product.ID]; !ok {
		return nil
	}
	cp := *product
	f.products[product.ID] = &cp
	return nil
}

func (f *fakeRepository) GetProductByURL(ctx context.Context, url string) (*domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.products {
		if p.URL == url {
			cp := *p
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeRepository) GetProductsBySearchID(ctx context.Context, searchID int64) ([]domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Product
	for pid := range f.searchProducts[searchID] {
		if p, ok := f.products[pid]; ok {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *fakeRepository) GetProductsBySearchIDWithStatus(ctx context.Context, searchID int64) ([]database2.ProductWithStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.getProductsWithStatusErr != nil {
		return nil, f.getProductsWithStatusErr
	}

	var out []database2.ProductWithStatus
	for pid, st := range f.searchProducts[searchID] {
		p, ok := f.products[pid]
		if !ok {
			continue
		}
		out = append(out, database2.ProductWithStatus{Product: *p, IsHidden: st.isHidden, IsActive: st.isActive})
	}
	return out, nil
}

func (f *fakeRepository) LinkProductToSearch(ctx context.Context, searchID, productID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.searchProducts[searchID] == nil {
		f.searchProducts[searchID] = make(map[int64]*spStatus)
	}
	f.searchProducts[searchID][productID] = &spStatus{isActive: true}
	return nil
}

func (f *fakeRepository) MarkProductsInactive(ctx context.Context, searchID int64, productIDs []int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, pid := range productIDs {
		if st, ok := f.searchProducts[searchID][pid]; ok {
			st.isActive = false
		}
	}
	return nil
}

func (f *fakeRepository) GetNewProductsSinceLastCheck(ctx context.Context, searchID int64) ([]domain.Product, error) {
	return nil, nil
}

func (f *fakeRepository) MarkProductsAsChecked(ctx context.Context, searchID int64, productIDs []int64) error {
	return nil
}

func (f *fakeRepository) SetProductHidden(ctx context.Context, searchID, productID int64, hidden bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.setProductHiddenErr != nil {
		return f.setProductHiddenErr
	}
	st, ok := f.searchProducts[searchID][productID]
	if !ok {
		return database2.ErrSearchProductNotFound
	}
	st.isHidden = hidden
	return nil
}

func (f *fakeRepository) Close() {}

// seedSearch inserts a search directly (bypassing CreateSearch) so tests
// can control its ID.
func (f *fakeRepository) seedSearch(s *domain.Search) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searches[s.ID] = s
	f.searchesByKey[s.Keyword] = s.ID
	if f.searchProducts[s.ID] == nil {
		f.searchProducts[s.ID] = make(map[int64]*spStatus)
	}
	if s.ID > f.nextSearchID {
		f.nextSearchID = s.ID
	}
}

// seedProduct inserts a product and links it to a search with the given
// visibility status.
func (f *fakeRepository) seedProduct(searchID int64, p *domain.Product, hidden, active bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.products[p.ID] = p
	if p.ID > f.nextProductID {
		f.nextProductID = p.ID
	}
	if f.searchProducts[searchID] == nil {
		f.searchProducts[searchID] = make(map[int64]*spStatus)
	}
	f.searchProducts[searchID][p.ID] = &spStatus{isHidden: hidden, isActive: active}
}

// newTestAPI builds an API backed by a fresh fakeRepository and a
// SearchService with zero configured adapters, so handleCreateSearch's
// background goroutine completes instantly without touching the network.
func newTestAPI() (*API, *fakeRepository) {
	repo := newFakeRepository()
	registry := adapter.NewRegistry(&config.Config{})
	searchService := service.NewSearchService(repo, registry)
	return &API{repo: repo, searchService: searchService}, repo
}

func decodeJSON(t *testing.T, body *bytes.Buffer, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

func TestHandleHealthCheck(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	decodeJSON(t, rec.Body, &body)
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want %q", body["status"], "ok")
	}
}

func TestHandleGetSearches(t *testing.T) {
	api, repo := newTestAPI()
	repo.seedSearch(&domain.Search{ID: 1, Keyword: "nintendo switch", CreatedAt: time.Now()})
	repo.seedProduct(1, &domain.Product{ID: 1, Title: "visible", URL: "https://x/1"}, false, true)
	repo.seedProduct(1, &domain.Product{ID: 2, Title: "hidden", URL: "https://x/2"}, true, true)
	repo.seedProduct(1, &domain.Product{ID: 3, Title: "inactive", URL: "https://x/3"}, false, false)

	router := newRouter(api)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/searches", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var searches []SearchResponse
	decodeJSON(t, rec.Body, &searches)
	if len(searches) != 1 {
		t.Fatalf("len(searches) = %d, want 1", len(searches))
	}
	if searches[0].ProductCount == nil || *searches[0].ProductCount != 1 {
		t.Fatalf("product_count = %v, want 1 (only the visible+active product)", searches[0].ProductCount)
	}
}

func TestHandleGetSearches_RepositoryError(t *testing.T) {
	api, repo := newTestAPI()
	repo.getAllSearchesWithCountsErr = context.DeadlineExceeded

	router := newRouter(api)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/searches", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleCreateSearch(t *testing.T) {
	api, repo := newTestAPI()
	router := newRouter(api)

	body := `{"keyword":"nintendo switch joycon"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/searches", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp SearchResponse
	decodeJSON(t, rec.Body, &resp)
	if resp.Keyword != "nintendo switch joycon" {
		t.Fatalf("keyword = %q, want %q", resp.Keyword, "nintendo switch joycon")
	}
	if resp.ID == 0 {
		t.Fatalf("expected non-zero ID")
	}

	if _, ok := repo.searchesByKey["nintendo switch joycon"]; !ok {
		t.Fatalf("expected search to be persisted in repository")
	}
}

func TestHandleCreateSearch_TrimsWhitespace(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/searches", bytes.NewBufferString(`{"keyword":"  spaced out  "}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp SearchResponse
	decodeJSON(t, rec.Body, &resp)
	if resp.Keyword != "spaced out" {
		t.Fatalf("keyword = %q, want trimmed %q", resp.Keyword, "spaced out")
	}
}

func TestHandleCreateSearch_EmptyKeyword(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/searches", bytes.NewBufferString(`{"keyword":"   "}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSearch_TooLong(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	longKeyword := make([]byte, maxKeywordLength+1)
	for i := range longKeyword {
		longKeyword[i] = 'a'
	}
	reqBody, err := json.Marshal(CreateSearchRequest{Keyword: string(longKeyword)})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/searches", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSearch_InvalidJSON(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/searches", bytes.NewBufferString(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateSearch_RepositoryError(t *testing.T) {
	api, repo := newTestAPI()
	repo.createSearchErr = context.DeadlineExceeded
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/searches", bytes.NewBufferString(`{"keyword":"x"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleDeleteSearch(t *testing.T) {
	api, repo := newTestAPI()
	repo.seedSearch(&domain.Search{ID: 7, Keyword: "gamecube"})
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/searches/7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if _, ok := repo.searches[7]; ok {
		t.Fatalf("expected search 7 to be deleted")
	}
}

func TestHandleDeleteSearch_NotFound(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/searches/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteSearch_InvalidID(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/searches/not-a-number", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleGetSearchProducts(t *testing.T) {
	api, repo := newTestAPI()
	repo.seedSearch(&domain.Search{ID: 3, Keyword: "n64"})
	repo.seedProduct(3, &domain.Product{ID: 10, Title: "visible", URL: "https://x/10"}, false, true)
	repo.seedProduct(3, &domain.Product{ID: 11, Title: "hidden", URL: "https://x/11"}, true, true)
	repo.seedProduct(3, &domain.Product{ID: 12, Title: "delisted", URL: "https://x/12"}, false, false)
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/searches/3/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp SearchWithProductsResponse
	decodeJSON(t, rec.Body, &resp)

	if len(resp.Products) != 3 {
		t.Fatalf("len(products) = %d, want 3 (hidden/inactive still included in payload)", len(resp.Products))
	}
	if resp.Total != 1 {
		t.Fatalf("total = %d, want 1 (only the visible+active product counts)", resp.Total)
	}
}

func TestHandleGetSearchProducts_SearchNotFound(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/searches/404/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleGetSearchProducts_InvalidID(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/searches/abc/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSetProductHidden(t *testing.T) {
	api, repo := newTestAPI()
	repo.seedSearch(&domain.Search{ID: 5, Keyword: "wii"})
	repo.seedProduct(5, &domain.Product{ID: 20, Title: "spam", URL: "https://x/20"}, false, true)
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/searches/5/products/20", bytes.NewBufferString(`{"hidden":true}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if !repo.searchProducts[5][20].isHidden {
		t.Fatalf("expected product 20 to be marked hidden")
	}
}

func TestHandleSetProductHidden_NotFound(t *testing.T) {
	api, repo := newTestAPI()
	repo.seedSearch(&domain.Search{ID: 5, Keyword: "wii"})
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/searches/5/products/999", bytes.NewBufferString(`{"hidden":true}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	_ = repo
}

func TestHandleSetProductHidden_InvalidSearchID(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/searches/abc/products/20", bytes.NewBufferString(`{"hidden":true}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSetProductHidden_InvalidProductID(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/searches/5/products/abc", bytes.NewBufferString(`{"hidden":true}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSetProductHidden_InvalidBody(t *testing.T) {
	api, repo := newTestAPI()
	repo.seedSearch(&domain.Search{ID: 5, Keyword: "wii"})
	repo.seedProduct(5, &domain.Product{ID: 20, Title: "spam", URL: "https://x/20"}, false, true)
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/searches/5/products/20", bytes.NewBufferString(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRouter_UnknownRouteReturns404(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// gorilla/mux only emits 405 when a MethodNotAllowedHandler is configured;
// this router doesn't set one, so an existing path with the wrong method
// falls through to the same 404 as an unknown path.
func TestRouter_WrongMethodReturns404(t *testing.T) {
	api, _ := newTestAPI()
	router := newRouter(api)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/searches", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
