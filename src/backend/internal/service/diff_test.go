package service

import (
	"context"
	"errors"
	"testing"

	"secondHand/internal/adapter"
	domain2 "secondHand/internal/domain"
)

func diffByType(diffs []domain2.ProductDiff, url string) *domain2.ProductDiff {
	for i := range diffs {
		if diffs[i].Product.URL == url {
			return &diffs[i]
		}
	}
	return nil
}

func TestGenerateDiff(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()

	search, err := repo.CreateSearch(ctx, "hemingway")
	if err != nil {
		t.Fatalf("seed CreateSearch: %v", err)
	}

	unchanged := &domain2.Product{URL: "https://shop/unchanged", Price: 100}
	priceUp := &domain2.Product{URL: "https://shop/price-up", Price: 100}
	priceDown := &domain2.Product{URL: "https://shop/price-down", Price: 100}
	removed := &domain2.Product{URL: "https://shop/removed", Price: 100}
	for _, p := range []*domain2.Product{unchanged, priceUp, priceDown, removed} {
		if err := repo.CreateProduct(ctx, p); err != nil {
			t.Fatalf("seed CreateProduct: %v", err)
		}
		if err := repo.LinkProductToSearch(ctx, search.ID, p.ID); err != nil {
			t.Fatalf("seed LinkProductToSearch: %v", err)
		}
	}

	previous, err := repo.GetProductsBySearchID(ctx, search.ID)
	if err != nil {
		t.Fatalf("seed GetProductsBySearchID: %v", err)
	}

	current := []domain2.Product{
		{ID: unchanged.ID, URL: unchanged.URL, Price: 100},
		{ID: priceUp.ID, URL: priceUp.URL, Price: 150},
		{ID: priceDown.ID, URL: priceDown.URL, Price: 50},
		{URL: "https://shop/new", Price: 999}, // not previously seen
	}

	svc := NewDiffService(repo)
	diffs, err := svc.GenerateDiff(ctx, search.ID, previous, current)
	if err != nil {
		t.Fatalf("GenerateDiff: %v", err)
	}

	if d := diffByType(diffs, unchanged.URL); d != nil {
		t.Fatalf("unchanged product should produce no diff, got %+v", d)
	}

	up := diffByType(diffs, priceUp.URL)
	if up == nil || up.DiffType != domain2.DiffTypePriceUp || !up.PriceChanged {
		t.Fatalf("price-up diff = %+v, want DiffTypePriceUp/PriceChanged=true", up)
	}
	if up.OldPrice == nil || *up.OldPrice != 100 || up.NewPrice == nil || *up.NewPrice != 150 {
		t.Fatalf("price-up diff prices = old:%v new:%v, want 100/150", up.OldPrice, up.NewPrice)
	}

	down := diffByType(diffs, priceDown.URL)
	if down == nil || down.DiffType != domain2.DiffTypePriceDown || !down.PriceChanged {
		t.Fatalf("price-down diff = %+v, want DiffTypePriceDown/PriceChanged=true", down)
	}

	newDiff := diffByType(diffs, "https://shop/new")
	if newDiff == nil || newDiff.DiffType != domain2.DiffTypeNew || newDiff.PriceChanged {
		t.Fatalf("new-product diff = %+v, want DiffTypeNew/PriceChanged=false", newDiff)
	}

	removedDiff := diffByType(diffs, removed.URL)
	if removedDiff == nil || removedDiff.DiffType != domain2.DiffTypeRemoved {
		t.Fatalf("removed-product diff = %+v, want DiffTypeRemoved", removedDiff)
	}
	if !repo.inactiveIDs[search.ID][removed.ID] {
		t.Fatal("removed product was not marked inactive")
	}
}

func TestGetDiffForAllSearches_SkipsSearchOnPreviousProductsError(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()

	if _, err := repo.CreateSearch(ctx, "hemingway"); err != nil {
		t.Fatalf("seed CreateSearch: %v", err)
	}
	repo.getProductsBySearchIDErr = errors.New("boom")

	searchSvc := NewSearchService(repo, adapter.NewRegistryFromAdapters())
	diffSvc := NewDiffService(repo)

	results, err := diffSvc.GetDiffForAllSearches(ctx, searchSvc)
	if err != nil {
		t.Fatalf("GetDiffForAllSearches: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %+v, want none (search skipped after snapshot error)", results)
	}
}

func TestGenerateDiff_MarkInactiveError(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()

	search, err := repo.CreateSearch(ctx, "hemingway")
	if err != nil {
		t.Fatalf("seed CreateSearch: %v", err)
	}
	product := &domain2.Product{URL: "https://shop/a", Price: 10}
	if err := repo.CreateProduct(ctx, product); err != nil {
		t.Fatalf("seed CreateProduct: %v", err)
	}
	if err := repo.LinkProductToSearch(ctx, search.ID, product.ID); err != nil {
		t.Fatalf("seed LinkProductToSearch: %v", err)
	}
	repo.markProductsInactiveErr = errors.New("boom")

	previous, err := repo.GetProductsBySearchID(ctx, search.ID)
	if err != nil {
		t.Fatalf("seed GetProductsBySearchID: %v", err)
	}

	svc := NewDiffService(repo)
	if _, err := svc.GenerateDiff(ctx, search.ID, previous, nil); err == nil {
		t.Fatal("expected an error when MarkProductsInactive fails, got nil")
	}
}

func TestGetDiffForAllSearches(t *testing.T) {
	repo := newFakeRepo()
	ctx := context.Background()

	if _, err := repo.CreateSearch(ctx, "hemingway"); err != nil {
		t.Fatalf("seed CreateSearch: %v", err)
	}
	if _, err := repo.CreateSearch(ctx, "steinbeck"); err != nil {
		t.Fatalf("seed CreateSearch: %v", err)
	}

	shop := &fakeAdapter{
		name: "mock-shop",
		products: []domain2.Product{
			{ShopSource: "mock-shop", Title: "new find", Price: 42, URL: "https://shop/new-find"},
		},
	}
	searchSvc := NewSearchService(repo, adapter.NewRegistryFromAdapters(shop))
	diffSvc := NewDiffService(repo)

	results, err := diffSvc.GetDiffForAllSearches(ctx, searchSvc)
	if err != nil {
		t.Fatalf("GetDiffForAllSearches: %v", err)
	}

	// GenerateDiff scopes "previous products" by search ID, so on this
	// first run both searches see the adapter's product as new to *them*
	// individually, even though it's the same underlying product row
	// (reused via GetProductByURL, not recreated).
	for _, keyword := range []string{"hemingway", "steinbeck"} {
		diffs, ok := results[keyword]
		if !ok || len(diffs) != 1 || diffs[0].DiffType != domain2.DiffTypeNew {
			t.Fatalf("results[%q] = %+v, want one DiffTypeNew entry", keyword, diffs)
		}
	}
	if len(repo.products) != 1 {
		t.Fatalf("repo has %d products, want 1 (shared across both searches)", len(repo.products))
	}
}
