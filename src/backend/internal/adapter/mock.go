package adapter

import (
	"context"
	"fmt"
	"math/rand"
	"secondHand/src/backend/internal/domain"
	"time"
)

// MockAdapter is a mock adapter for testing purposes
type MockAdapter struct {
	*BaseAdapter
	shopName string
}

// NewMockAdapter creates a new mock adapter
func NewMockAdapter(shopName, baseURL string, delayMS, timeoutSec int) *MockAdapter {
	return &MockAdapter{
		BaseAdapter: NewBaseAdapter(shopName, baseURL, delayMS, timeoutSec),
		shopName:    shopName,
	}
}

// SupportsSearch returns true
func (a *MockAdapter) SupportsSearch() bool {
	return true
}

// Search returns mock products
func (a *MockAdapter) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
	// Simulate network delay
	time.Sleep(time.Duration(100+rand.Intn(500)) * time.Millisecond)

	// Generate 2-5 mock products
	numProducts := 2 + rand.Intn(4)
	products := make([]domain.Product, numProducts)

	for i := 0; i < numProducts; i++ {
		price := float64(100+rand.Intn(9900)) + float64(rand.Intn(100))/100.0

		products[i] = domain.Product{
			ShopSource:  a.shopName,
			Title:       fmt.Sprintf("%s - Product %d from %s", keyword, i+1, a.shopName),
			Description: fmt.Sprintf("Great %s in good condition. This is a mock product for testing.", keyword),
			Price:       price,
			Currency:    "CZK",
			AuctionType: domain.AuctionTypeSale,
			Condition:   []domain.Condition{domain.ConditionNew, domain.ConditionUsed, domain.ConditionLikeNew}[rand.Intn(3)],
			URL:         fmt.Sprintf("%s/product/%s-%d-%d", a.baseURL, keyword, i+1, time.Now().Unix()),
			ImageURL:    fmt.Sprintf("%s/images/product-%d.jpg", a.baseURL, i+1),
			Location:    []string{"Praha", "Brno", "Ostrava", "Plzeň"}[rand.Intn(4)],
			SellerName:  fmt.Sprintf("Seller%d", rand.Intn(100)),
		}
	}

	return products, nil
}
