package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"secondHand/internal/config"
	database2 "secondHand/internal/database"
	domain2 "secondHand/internal/domain"
	output2 "secondHand/internal/output"
)

// fakeRepo implements database2.Repository. notifyGoodOffers only ever
// calls GetProductsBySearchID; every other method is an unused stub.
type fakeRepo struct {
	mu               sync.Mutex
	productsBySearch map[int64][]domain2.Product
}

func (f *fakeRepo) GetProductsBySearchID(ctx context.Context, searchID int64) ([]domain2.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.productsBySearch[searchID], nil
}

func (f *fakeRepo) CreateSearch(ctx context.Context, keyword string) (*domain2.Search, error) {
	return nil, nil
}
func (f *fakeRepo) GetSearchByID(ctx context.Context, id int64) (*domain2.Search, error) {
	return nil, nil
}
func (f *fakeRepo) GetSearchByKeyword(ctx context.Context, keyword string) (*domain2.Search, error) {
	return nil, nil
}
func (f *fakeRepo) GetAllSearches(ctx context.Context) ([]domain2.Search, error) { return nil, nil }
func (f *fakeRepo) GetAllSearchesWithCounts(ctx context.Context) ([]database2.SearchWithCount, error) {
	return nil, nil
}
func (f *fakeRepo) UpdateSearchLastChecked(ctx context.Context, searchID int64) error { return nil }
func (f *fakeRepo) SetGoodOfferConfig(ctx context.Context, searchID int64, maxPrice, avgDiscountPct *float64) error {
	return nil
}
func (f *fakeRepo) DeleteSearch(ctx context.Context, searchID int64) error { return nil }
func (f *fakeRepo) CreateProduct(ctx context.Context, product *domain2.Product) error {
	return nil
}
func (f *fakeRepo) UpdateProduct(ctx context.Context, product *domain2.Product) error {
	return nil
}
func (f *fakeRepo) GetProductByURL(ctx context.Context, url string) (*domain2.Product, error) {
	return nil, nil
}
func (f *fakeRepo) GetProductsBySearchIDWithStatus(ctx context.Context, searchID int64) ([]database2.ProductWithStatus, error) {
	return nil, nil
}
func (f *fakeRepo) LinkProductToSearch(ctx context.Context, searchID, productID int64) error {
	return nil
}
func (f *fakeRepo) MarkProductsInactive(ctx context.Context, searchID int64, productIDs []int64) error {
	return nil
}
func (f *fakeRepo) SetProductHidden(ctx context.Context, searchID, productID int64, hidden bool) error {
	return nil
}
func (f *fakeRepo) Close() {}

// countingTelegramServer returns a *output2.TelegramNotifier wired to a
// test server, plus a function reporting how many requests it received.
func countingTelegramServer(t *testing.T) (*output2.TelegramNotifier, func() int) {
	t.Helper()

	var mu sync.Mutex
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	notifier := output2.NewTelegramNotifier(&config.TelegramConfig{
		BotToken: "testtoken",
		ChatID:   "12345",
		APIBase:  server.URL,
	})

	return notifier, func() int {
		mu.Lock()
		defer mu.Unlock()
		return count
	}
}

func TestNotifyGoodOffers_SendsForMatchingNewEbayListing(t *testing.T) {
	notifier, requestCount := countingTelegramServer(t)
	maxPrice := 100.0
	search := domain2.Search{ID: 1, Keyword: "joy-con", MaxPrice: &maxPrice}

	diffs := map[string][]domain2.ProductDiff{
		"joy-con": {
			{Product: domain2.Product{ShopSource: "ebay.com", Price: 50, URL: "https://ebay.com/1"}, DiffType: domain2.DiffTypeNew},
		},
	}

	notifyGoodOffers(context.Background(), &fakeRepo{}, notifier, []domain2.Search{search}, diffs)

	if got := requestCount(); got != 1 {
		t.Fatalf("Telegram requests = %d, want 1", got)
	}
}

func TestNotifyGoodOffers_SkipsSearchWithoutThresholds(t *testing.T) {
	notifier, requestCount := countingTelegramServer(t)
	search := domain2.Search{ID: 1, Keyword: "joy-con"}

	diffs := map[string][]domain2.ProductDiff{
		"joy-con": {
			{Product: domain2.Product{ShopSource: "ebay.com", Price: 1, URL: "https://ebay.com/1"}, DiffType: domain2.DiffTypeNew},
		},
	}

	notifyGoodOffers(context.Background(), &fakeRepo{}, notifier, []domain2.Search{search}, diffs)

	if got := requestCount(); got != 0 {
		t.Fatalf("Telegram requests = %d, want 0 (no thresholds configured)", got)
	}
}

func TestNotifyGoodOffers_SkipsNonEbayShop(t *testing.T) {
	notifier, requestCount := countingTelegramServer(t)
	maxPrice := 100.0
	search := domain2.Search{ID: 1, Keyword: "joy-con", MaxPrice: &maxPrice}

	diffs := map[string][]domain2.ProductDiff{
		"joy-con": {
			{Product: domain2.Product{ShopSource: "bazos.cz", Price: 50, URL: "https://bazos.cz/1"}, DiffType: domain2.DiffTypeNew},
		},
	}

	notifyGoodOffers(context.Background(), &fakeRepo{}, notifier, []domain2.Search{search}, diffs)

	if got := requestCount(); got != 0 {
		t.Fatalf("Telegram requests = %d, want 0 (non-ebay shop)", got)
	}
}

func TestNotifyGoodOffers_SkipsUnqualifyingDiffTypes(t *testing.T) {
	notifier, requestCount := countingTelegramServer(t)
	maxPrice := 100.0
	search := domain2.Search{ID: 1, Keyword: "joy-con", MaxPrice: &maxPrice}

	diffs := map[string][]domain2.ProductDiff{
		"joy-con": {
			{Product: domain2.Product{ShopSource: "ebay.com", Price: 50, URL: "https://ebay.com/1"}, DiffType: domain2.DiffTypeRemoved},
			{Product: domain2.Product{ShopSource: "ebay.com", Price: 50, URL: "https://ebay.com/2"}, DiffType: domain2.DiffTypePriceUp},
		},
	}

	notifyGoodOffers(context.Background(), &fakeRepo{}, notifier, []domain2.Search{search}, diffs)

	if got := requestCount(); got != 0 {
		t.Fatalf("Telegram requests = %d, want 0 (neither Removed nor PriceUp qualify)", got)
	}
}

func TestNotifyGoodOffers_PriceDownQualifies(t *testing.T) {
	notifier, requestCount := countingTelegramServer(t)
	maxPrice := 100.0
	search := domain2.Search{ID: 1, Keyword: "joy-con", MaxPrice: &maxPrice}

	diffs := map[string][]domain2.ProductDiff{
		"joy-con": {
			{Product: domain2.Product{ShopSource: "ebay.com", Price: 50, URL: "https://ebay.com/1"}, DiffType: domain2.DiffTypePriceDown},
		},
	}

	notifyGoodOffers(context.Background(), &fakeRepo{}, notifier, []domain2.Search{search}, diffs)

	if got := requestCount(); got != 1 {
		t.Fatalf("Telegram requests = %d, want 1", got)
	}
}

func TestNotifyGoodOffers_DoesNotMatchAboveCeiling(t *testing.T) {
	notifier, requestCount := countingTelegramServer(t)
	maxPrice := 10.0
	search := domain2.Search{ID: 1, Keyword: "joy-con", MaxPrice: &maxPrice}

	diffs := map[string][]domain2.ProductDiff{
		"joy-con": {
			{Product: domain2.Product{ShopSource: "ebay.com", Price: 999, URL: "https://ebay.com/1"}, DiffType: domain2.DiffTypeNew},
		},
	}

	notifyGoodOffers(context.Background(), &fakeRepo{}, notifier, []domain2.Search{search}, diffs)

	if got := requestCount(); got != 0 {
		t.Fatalf("Telegram requests = %d, want 0 (price above ceiling)", got)
	}
}

func TestNotifyGoodOffers_UnknownKeywordIsSkipped(t *testing.T) {
	notifier, requestCount := countingTelegramServer(t)

	diffs := map[string][]domain2.ProductDiff{
		"no-such-search": {
			{Product: domain2.Product{ShopSource: "ebay.com", Price: 1, URL: "https://ebay.com/1"}, DiffType: domain2.DiffTypeNew},
		},
	}

	notifyGoodOffers(context.Background(), &fakeRepo{}, notifier, nil, diffs)

	if got := requestCount(); got != 0 {
		t.Fatalf("Telegram requests = %d, want 0 (keyword not in searches)", got)
	}
}
