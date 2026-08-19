package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"secondHand/internal/adapter"
	domain2 "secondHand/internal/domain"
)

func TestSearchWithFilter_NewProducts(t *testing.T) {
	repo := newFakeRepo()
	shop := &fakeAdapter{
		name: "mock-shop",
		products: []domain2.Product{
			{ShopSource: "mock-shop", Title: "A", Price: 10, URL: "https://shop/a"},
			{ShopSource: "mock-shop", Title: "B", Price: 20, URL: "https://shop/b"},
		},
	}
	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters(shop))

	products, err := svc.SearchWithFilter(context.Background(), "hemingway", "")
	if err != nil {
		t.Fatalf("SearchWithFilter: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("got %d products, want 2", len(products))
	}
	if len(repo.products) != 2 {
		t.Fatalf("repo has %d products stored, want 2", len(repo.products))
	}

	search, err := repo.GetSearchByKeyword(context.Background(), "hemingway")
	if err != nil {
		t.Fatalf("GetSearchByKeyword: %v", err)
	}
	if search.LastCheckedAt == nil {
		t.Fatal("LastCheckedAt was not updated")
	}
	if len(repo.links[search.ID]) != 2 {
		t.Fatalf("search has %d linked products, want 2", len(repo.links[search.ID]))
	}
}

func TestSearchWithFilter_ExistingProductPriceUnchangedButOtherFieldsRefresh(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()

	existing := &domain2.Product{Title: "A", Price: 10, URL: "https://shop/a"}
	if err := repo.CreateProduct(ctx, existing); err != nil {
		t.Fatalf("seed CreateProduct: %v", err)
	}

	shop := &fakeAdapter{
		name: "mock-shop",
		products: []domain2.Product{
			{ShopSource: "mock-shop", Title: "A (rescraped)", Price: 10, URL: "https://shop/a"},
		},
	}
	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters(shop))

	products, err := svc.SearchWithFilter(ctx, "hemingway", "")
	if err != nil {
		t.Fatalf("SearchWithFilter: %v", err)
	}
	if len(products) != 1 || products[0].ID != existing.ID {
		t.Fatalf("products = %+v, want existing product ID %d reused", products, existing.ID)
	}
	// A field like title can change on rescrape independent of price (e.g.
	// eBay's title translation depends on request headers, not the
	// listing) - the stored row must reflect the latest scrape either way.
	if repo.products[existing.ID].Title != "A (rescraped)" {
		t.Fatalf("title should refresh even when price is unchanged, got %q", repo.products[existing.ID].Title)
	}
}

func TestSearchWithFilter_ExistingProductPriceChanged(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()

	existing := &domain2.Product{Title: "A", Price: 10, URL: "https://shop/a"}
	if err := repo.CreateProduct(ctx, existing); err != nil {
		t.Fatalf("seed CreateProduct: %v", err)
	}

	shop := &fakeAdapter{
		name: "mock-shop",
		products: []domain2.Product{
			{ShopSource: "mock-shop", Title: "A", Price: 25, URL: "https://shop/a"},
		},
	}
	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters(shop))

	products, err := svc.SearchWithFilter(ctx, "hemingway", "")
	if err != nil {
		t.Fatalf("SearchWithFilter: %v", err)
	}
	if len(products) != 1 || products[0].ID != existing.ID {
		t.Fatalf("products = %+v, want existing product ID %d reused", products, existing.ID)
	}
	if repo.products[existing.ID].Price != 25 {
		t.Fatalf("price was not updated, got %v, want 25", repo.products[existing.ID].Price)
	}
}

func TestSearchWithFilter_FiltersToOneAdapter(t *testing.T) {
	repo := newFakeRepo()
	shopA := &fakeAdapter{name: "shop-a", products: []domain2.Product{{ShopSource: "shop-a", URL: "https://a/1"}}}
	shopB := &fakeAdapter{name: "shop-b", products: []domain2.Product{{ShopSource: "shop-b", URL: "https://b/1"}, {ShopSource: "shop-b", URL: "https://b/2"}}}
	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters(shopA, shopB))

	products, err := svc.SearchWithFilter(context.Background(), "keyword", "shop-b")
	if err != nil {
		t.Fatalf("SearchWithFilter: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("got %d products, want 2 (shop-b only)", len(products))
	}
	for _, p := range products {
		if p.ShopSource != "shop-b" {
			t.Fatalf("got product from %q, want only shop-b", p.ShopSource)
		}
	}
}

func TestSearchWithFilter_UnknownAdapterFilter(t *testing.T) {
	repo := newFakeRepo()
	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters(&fakeAdapter{name: "shop-a"}))

	_, err := svc.SearchWithFilter(context.Background(), "keyword", "no-such-shop")
	if err == nil || !strings.Contains(err.Error(), "adapter not found") {
		t.Fatalf("err = %v, want an 'adapter not found' error", err)
	}
}

func TestSearchWithFilter_AllAdaptersFail(t *testing.T) {
	repo := newFakeRepo()
	shop := &fakeAdapter{name: "shop-a", err: errors.New("boom")}
	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters(shop))

	products, err := svc.SearchWithFilter(context.Background(), "keyword", "")
	if err == nil {
		t.Fatal("expected an error when every adapter fails, got nil")
	}
	if products != nil {
		t.Fatalf("products = %v, want nil", products)
	}
}

func TestSearchWithFilter_PartialFailureStillReturnsResults(t *testing.T) {
	repo := newFakeRepo()
	failing := &fakeAdapter{name: "shop-fail", err: errors.New("boom")}
	working := &fakeAdapter{name: "shop-ok", products: []domain2.Product{{ShopSource: "shop-ok", URL: "https://ok/1"}}}
	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters(failing, working))

	products, err := svc.SearchWithFilter(context.Background(), "keyword", "")
	if err != nil {
		t.Fatalf("SearchWithFilter: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("got %d products, want 1 from the working adapter", len(products))
	}
}

func TestGetSearchProducts(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()

	search, err := repo.CreateSearch(ctx, "hemingway")
	if err != nil {
		t.Fatalf("seed CreateSearch: %v", err)
	}
	product := &domain2.Product{URL: "https://shop/a"}
	if err := repo.CreateProduct(ctx, product); err != nil {
		t.Fatalf("seed CreateProduct: %v", err)
	}
	if err := repo.LinkProductToSearch(ctx, search.ID, product.ID); err != nil {
		t.Fatalf("seed LinkProductToSearch: %v", err)
	}

	svc := NewSearchService(repo, adapter.NewRegistryFromAdapters())

	products, err := svc.GetSearchProducts(ctx, "hemingway")
	if err != nil {
		t.Fatalf("GetSearchProducts: %v", err)
	}
	if len(products) != 1 || products[0].URL != product.URL {
		t.Fatalf("products = %+v, want just %+v", products, product)
	}

	if _, err := svc.GetSearchProducts(ctx, "no-such-keyword"); err == nil {
		t.Fatal("expected an error for an unknown keyword, got nil")
	}
}
