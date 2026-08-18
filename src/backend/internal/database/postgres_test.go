package database

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"secondHand/internal/config"
	"secondHand/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupTestRepo connects to a real Postgres (DB_HOST defaults to what
// compose.yaml's backend-dev service sets, "postgres", when run via
// `make test`) and ensures migrations are applied. Skips instead of
// failing when no database is reachable, so `go test` still works for
// someone running it outside that container.
func setupTestRepo(t *testing.T) *PostgresRepository {
	t.Helper()

	cfg := testDatabaseConfig()
	connStr := cfg.ConnectionString()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("no database reachable, skipping functional test: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no database reachable, skipping functional test: %v", err)
	}

	if err := Migrate(pool, "../../migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	t.Cleanup(pool.Close)
	return &PostgresRepository{pool: pool}
}

func testDatabaseConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:     getenvDefault("DB_HOST", "localhost"),
		Port:     getenvDefault("DB_PORT", "5432"),
		User:     getenvDefault("DB_USER", "secondhand"),
		Password: getenvDefault("DB_PASSWORD", "secondhand_dev"),
		DBName:   getenvDefault("DB_NAME", "secondhand"),
		SSLMode:  getenvDefault("DB_SSLMODE", "disable"),
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var testSeq int64

// uniqueName returns a value that's unique both within a test run and
// across separate runs, so tests never collide with leftover rows from
// a previous run against the same shared dev database.
func uniqueName(prefix string) string {
	n := atomic.AddInt64(&testSeq, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), n)
}

func testProduct(url string) *domain.Product {
	return &domain.Product{
		ShopSource:  "mock-shop",
		Title:       "Test Product",
		Description: "A product for functional tests",
		Price:       123.45,
		Currency:    "CZK",
		AuctionType: domain.AuctionTypeSale,
		Condition:   domain.ConditionUsed,
		URL:         url,
		ImageURL:    "https://example.com/image.jpg",
		Location:    "Prague",
		SellerName:  "seller",
	}
}

func TestNewPostgresRepository(t *testing.T) {
	cfg := testDatabaseConfig()
	if _, err := pgxpool.New(context.Background(), cfg.ConnectionString()); err != nil {
		t.Skip("no database configured, skipping")
	}

	repo, err := NewPostgresRepository(&cfg)
	if err != nil {
		t.Skipf("no database reachable, skipping: %v", err)
	}
	defer repo.Close()

	badCfg := cfg
	badCfg.Host = "invalid.invalid"
	if _, err := NewPostgresRepository(&badCfg); err == nil {
		t.Fatal("expected an error connecting to an unreachable host, got nil")
	}
}

func TestPostgresRepository_SearchLifecycle(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()
	keyword := uniqueName("keyword")

	search, err := repo.CreateSearch(ctx, keyword)
	if err != nil {
		t.Fatalf("CreateSearch: %v", err)
	}
	if search.Keyword != keyword {
		t.Fatalf("Keyword = %q, want %q", search.Keyword, keyword)
	}

	// CreateSearch is idempotent: calling it again with the same keyword
	// returns the same row rather than erroring on the UNIQUE constraint.
	again, err := repo.CreateSearch(ctx, keyword)
	if err != nil {
		t.Fatalf("CreateSearch (repeat): %v", err)
	}
	if again.ID != search.ID {
		t.Fatalf("repeat CreateSearch ID = %d, want %d", again.ID, search.ID)
	}

	byID, err := repo.GetSearchByID(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetSearchByID: %v", err)
	}
	if byID.Keyword != keyword {
		t.Fatalf("GetSearchByID keyword = %q, want %q", byID.Keyword, keyword)
	}

	byKeyword, err := repo.GetSearchByKeyword(ctx, keyword)
	if err != nil {
		t.Fatalf("GetSearchByKeyword: %v", err)
	}
	if byKeyword.ID != search.ID {
		t.Fatalf("GetSearchByKeyword ID = %d, want %d", byKeyword.ID, search.ID)
	}

	if _, err := repo.GetSearchByKeyword(ctx, uniqueName("no-such-keyword")); err == nil {
		t.Fatal("expected an error for a nonexistent keyword, got nil")
	}

	if err := repo.UpdateSearchLastChecked(ctx, search.ID); err != nil {
		t.Fatalf("UpdateSearchLastChecked: %v", err)
	}
	afterCheck, err := repo.GetSearchByID(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetSearchByID after check: %v", err)
	}
	if afterCheck.LastCheckedAt == nil {
		t.Fatal("LastCheckedAt is nil after UpdateSearchLastChecked")
	}

	maxPrice, avgDiscount := 500.0, 15.5
	if err := repo.SetGoodOfferConfig(ctx, search.ID, &maxPrice, &avgDiscount); err != nil {
		t.Fatalf("SetGoodOfferConfig: %v", err)
	}
	withConfig, err := repo.GetSearchByID(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetSearchByID after config: %v", err)
	}
	if withConfig.MaxPrice == nil || *withConfig.MaxPrice != maxPrice {
		t.Fatalf("MaxPrice = %v, want %v", withConfig.MaxPrice, maxPrice)
	}
	if withConfig.AvgDiscountPct == nil || *withConfig.AvgDiscountPct != avgDiscount {
		t.Fatalf("AvgDiscountPct = %v, want %v", withConfig.AvgDiscountPct, avgDiscount)
	}

	all, err := repo.GetAllSearches(ctx)
	if err != nil {
		t.Fatalf("GetAllSearches: %v", err)
	}
	found := false
	for _, s := range all {
		if s.ID == search.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("GetAllSearches did not include the created search")
	}

	if err := repo.DeleteSearch(ctx, search.ID); err != nil {
		t.Fatalf("DeleteSearch: %v", err)
	}
	if _, err := repo.GetSearchByID(ctx, search.ID); err == nil {
		t.Fatal("expected an error fetching a deleted search, got nil")
	}
	if err := repo.DeleteSearch(ctx, search.ID); err != ErrSearchNotFound {
		t.Fatalf("DeleteSearch (already deleted) = %v, want ErrSearchNotFound", err)
	}
}

func TestPostgresRepository_ProductLifecycle(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	product := testProduct(uniqueName("https://example.com/product"))
	if err := repo.CreateProduct(ctx, product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if product.ID == 0 {
		t.Fatal("CreateProduct did not populate ID")
	}

	fetched, err := repo.GetProductByURL(ctx, product.URL)
	if err != nil {
		t.Fatalf("GetProductByURL: %v", err)
	}
	if fetched.Title != product.Title || fetched.Price != product.Price {
		t.Fatalf("GetProductByURL = %+v, want title/price matching %+v", fetched, product)
	}

	fetched.Title = "Updated Title"
	fetched.Price = 999.99
	if err := repo.UpdateProduct(ctx, fetched); err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	updated, err := repo.GetProductByURL(ctx, product.URL)
	if err != nil {
		t.Fatalf("GetProductByURL after update: %v", err)
	}
	if updated.Title != "Updated Title" || updated.Price != 999.99 {
		t.Fatalf("after update: title=%q price=%v, want %q / %v", updated.Title, updated.Price, "Updated Title", 999.99)
	}

	if _, err := repo.GetProductByURL(ctx, uniqueName("https://example.com/missing")); err == nil {
		t.Fatal("expected an error for a nonexistent product URL, got nil")
	}
}

func TestPostgresRepository_SearchProductLinking(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	search, err := repo.CreateSearch(ctx, uniqueName("keyword"))
	if err != nil {
		t.Fatalf("CreateSearch: %v", err)
	}

	p1 := testProduct(uniqueName("https://example.com/p1"))
	p2 := testProduct(uniqueName("https://example.com/p2"))
	for _, p := range []*domain.Product{p1, p2} {
		if err := repo.CreateProduct(ctx, p); err != nil {
			t.Fatalf("CreateProduct: %v", err)
		}
		if err := repo.LinkProductToSearch(ctx, search.ID, p.ID); err != nil {
			t.Fatalf("LinkProductToSearch: %v", err)
		}
	}

	products, err := repo.GetProductsBySearchID(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetProductsBySearchID: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("GetProductsBySearchID returned %d products, want 2", len(products))
	}

	withStatus, err := repo.GetProductsBySearchIDWithStatus(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetProductsBySearchIDWithStatus: %v", err)
	}
	if len(withStatus) != 2 {
		t.Fatalf("GetProductsBySearchIDWithStatus returned %d products, want 2", len(withStatus))
	}
	for _, p := range withStatus {
		if p.IsHidden || !p.IsActive {
			t.Fatalf("newly linked product has IsHidden=%v IsActive=%v, want false/true", p.IsHidden, p.IsActive)
		}
	}

	counts, err := repo.GetAllSearchesWithCounts(ctx)
	if err != nil {
		t.Fatalf("GetAllSearchesWithCounts: %v", err)
	}
	if c := findCount(counts, search.ID); c != 2 {
		t.Fatalf("GetAllSearchesWithCounts count = %d, want 2", c)
	}

	if err := repo.SetProductHidden(ctx, search.ID, p1.ID, true); err != nil {
		t.Fatalf("SetProductHidden: %v", err)
	}
	counts, err = repo.GetAllSearchesWithCounts(ctx)
	if err != nil {
		t.Fatalf("GetAllSearchesWithCounts after hide: %v", err)
	}
	if c := findCount(counts, search.ID); c != 1 {
		t.Fatalf("GetAllSearchesWithCounts count after hiding one = %d, want 1", c)
	}

	if err := repo.SetProductHidden(ctx, search.ID, 999999999, true); err != ErrSearchProductNotFound {
		t.Fatalf("SetProductHidden (unlinked pair) = %v, want ErrSearchProductNotFound", err)
	}

	if err := repo.MarkProductsInactive(ctx, search.ID, []int64{p2.ID}); err != nil {
		t.Fatalf("MarkProductsInactive: %v", err)
	}
	// Empty ID slice must be a no-op, not an error.
	if err := repo.MarkProductsInactive(ctx, search.ID, nil); err != nil {
		t.Fatalf("MarkProductsInactive (empty): %v", err)
	}

	counts, err = repo.GetAllSearchesWithCounts(ctx)
	if err != nil {
		t.Fatalf("GetAllSearchesWithCounts after inactive: %v", err)
	}
	if c := findCount(counts, search.ID); c != 0 {
		t.Fatalf("GetAllSearchesWithCounts count after hiding+deactivating both = %d, want 0", c)
	}

	// Re-linking an inactive product reactivates it (e.g. it reappeared
	// in a later scrape).
	if err := repo.LinkProductToSearch(ctx, search.ID, p2.ID); err != nil {
		t.Fatalf("LinkProductToSearch (reactivate): %v", err)
	}
	withStatus, err = repo.GetProductsBySearchIDWithStatus(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetProductsBySearchIDWithStatus after reactivate: %v", err)
	}
	p2Active := false
	for _, p := range withStatus {
		if p.ID == p2.ID {
			p2Active = p.IsActive
		}
	}
	if !p2Active {
		t.Fatal("re-linking an inactive product did not reactivate it")
	}
}

func TestPostgresRepository_NewAndCheckedProducts(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	search, err := repo.CreateSearch(ctx, uniqueName("keyword"))
	if err != nil {
		t.Fatalf("CreateSearch: %v", err)
	}
	product := testProduct(uniqueName("https://example.com/new-product"))
	if err := repo.CreateProduct(ctx, product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if err := repo.LinkProductToSearch(ctx, search.ID, product.ID); err != nil {
		t.Fatalf("LinkProductToSearch: %v", err)
	}

	newProducts, err := repo.GetNewProductsSinceLastCheck(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetNewProductsSinceLastCheck: %v", err)
	}
	if len(newProducts) != 1 || newProducts[0].ID != product.ID {
		t.Fatalf("GetNewProductsSinceLastCheck = %+v, want just %+v", newProducts, product)
	}

	if err := repo.MarkProductsAsChecked(ctx, search.ID, []int64{product.ID}); err != nil {
		t.Fatalf("MarkProductsAsChecked: %v", err)
	}
	newProducts, err = repo.GetNewProductsSinceLastCheck(ctx, search.ID)
	if err != nil {
		t.Fatalf("GetNewProductsSinceLastCheck after check: %v", err)
	}
	if len(newProducts) != 0 {
		t.Fatalf("GetNewProductsSinceLastCheck after MarkProductsAsChecked = %+v, want none", newProducts)
	}
}

// DeleteSearch cascades to search_products but must leave the underlying
// product row alone, since another search may still reference it.
func TestPostgresRepository_DeleteSearchCascadesLinkNotProduct(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	search, err := repo.CreateSearch(ctx, uniqueName("keyword"))
	if err != nil {
		t.Fatalf("CreateSearch: %v", err)
	}
	product := testProduct(uniqueName("https://example.com/cascade-product"))
	if err := repo.CreateProduct(ctx, product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if err := repo.LinkProductToSearch(ctx, search.ID, product.ID); err != nil {
		t.Fatalf("LinkProductToSearch: %v", err)
	}

	if err := repo.DeleteSearch(ctx, search.ID); err != nil {
		t.Fatalf("DeleteSearch: %v", err)
	}

	if _, err := repo.GetProductByURL(ctx, product.URL); err != nil {
		t.Fatalf("product should survive its search being deleted, got: %v", err)
	}
}

func findCount(counts []SearchWithCount, searchID int64) int {
	for _, c := range counts {
		if c.ID == searchID {
			return c.ProductCount
		}
	}
	return -1
}
