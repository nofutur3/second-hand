package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"secondHand/src/backend/internal/config"
	"secondHand/src/backend/internal/domain"
)

func newTestEbayServer(t *testing.T, tokenHits *int32) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/identity/v1/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(tokenHits, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"expires_in":   7200,
		})
	})

	mux.HandleFunc("/buy/browse/v1/item_summary/search", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"itemSummaries": []map[string]interface{}{
				{
					"itemWebUrl": "https://www.ebay.com/itm/1",
					"title":      "Nintendo Switch Joy-Con Pair",
					"price":      map[string]string{"value": "899.99", "currency": "USD"},
					"condition":  "New",
					"image":      map[string]string{"imageUrl": "https://example.com/img1.jpg"},
					"itemLocation": map[string]string{
						"city":    "Seattle",
						"country": "US",
					},
					"seller":        map[string]string{"username": "retro_parts_shop"},
					"buyingOptions": []string{"FIXED_PRICE"},
				},
				{
					"itemWebUrl":    "https://www.ebay.com/itm/2",
					"title":         "Game Boy Color Shell - For Parts",
					"price":         map[string]string{"value": "150.00", "currency": "USD"},
					"condition":     "For parts or not working",
					"buyingOptions": []string{"AUCTION"},
					"itemEndDate":   "2026-08-01T12:00:00Z",
				},
				{
					"itemWebUrl":    "https://www.ebay.com/itm/3",
					"title":         "Seller Refurbished Switch Dock",
					"price":         map[string]string{"value": "45.50", "currency": "USD"},
					"condition":     "Seller refurbished",
					"buyingOptions": []string{"FIXED_PRICE"},
				},
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestEbayAdapter_SearchAndTokenCaching(t *testing.T) {
	var tokenHits int32
	server := newTestEbayServer(t, &tokenHits)
	defer server.Close()

	cfg := config.EbayConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		APIBase:      server.URL,
	}
	adapter := NewEbayAdapter(server.URL, cfg, 0, 5)

	if adapter.Name() != "ebay.com" {
		t.Errorf("Name() = %q, want %q", adapter.Name(), "ebay.com")
	}

	products, err := adapter.Search(context.Background(), "nintendo parts")
	if err != nil {
		t.Fatalf("first Search() error: %v", err)
	}
	if len(products) != 3 {
		t.Fatalf("expected 3 products, got %d", len(products))
	}

	// Second call within the token's validity window must not re-hit the token endpoint.
	if _, err := adapter.Search(context.Background(), "nintendo parts"); err != nil {
		t.Fatalf("second Search() error: %v", err)
	}
	if got := atomic.LoadInt32(&tokenHits); got != 1 {
		t.Errorf("token endpoint hit %d times across two Search calls, want 1 (cached)", got)
	}

	newItem := products[0]
	if newItem.Price != 899.99 {
		t.Errorf("item 0 Price = %v, want 899.99", newItem.Price)
	}
	if newItem.Currency != "USD" {
		t.Errorf("item 0 Currency = %q, want USD", newItem.Currency)
	}
	if newItem.Condition != domain.ConditionNew {
		t.Errorf("item 0 Condition = %q, want %q", newItem.Condition, domain.ConditionNew)
	}
	if newItem.AuctionType != domain.AuctionTypeSale {
		t.Errorf("item 0 AuctionType = %q, want %q", newItem.AuctionType, domain.AuctionTypeSale)
	}
	if newItem.SellerName != "retro_parts_shop" {
		t.Errorf("item 0 SellerName = %q, want retro_parts_shop", newItem.SellerName)
	}
	if newItem.Location != "Seattle" {
		t.Errorf("item 0 Location = %q, want Seattle", newItem.Location)
	}

	forPartsItem := products[1]
	if forPartsItem.Condition != domain.ConditionDamaged {
		t.Errorf("item 1 Condition = %q, want %q", forPartsItem.Condition, domain.ConditionDamaged)
	}
	if forPartsItem.AuctionType != domain.AuctionTypeAuction {
		t.Errorf("item 1 AuctionType = %q, want %q", forPartsItem.AuctionType, domain.AuctionTypeAuction)
	}
	if forPartsItem.EndingTime == nil {
		t.Fatal("item 1 EndingTime should be set for an auction listing")
	}

	refurbishedItem := products[2]
	if refurbishedItem.Condition != domain.ConditionLikeNew {
		t.Errorf("item 2 Condition = %q, want %q (refurbished maps to closest existing enum)", refurbishedItem.Condition, domain.ConditionLikeNew)
	}
}

func TestMapEbayCondition(t *testing.T) {
	tests := []struct {
		input string
		want  domain.Condition
	}{
		{"New", domain.ConditionNew},
		{"New with tags", domain.ConditionNew},
		{"Certified refurbished", domain.ConditionLikeNew},
		{"Seller refurbished", domain.ConditionLikeNew},
		{"Used", domain.ConditionUsed},
		{"Pre-owned", domain.ConditionUsed},
		{"For parts or not working", domain.ConditionDamaged},
		{"New other", domain.ConditionUnknown},
		{"New with defects", domain.ConditionUnknown},
		{"something else entirely", domain.ConditionUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := mapEbayCondition(tt.input); got != tt.want {
				t.Errorf("mapEbayCondition(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEbayAdapter_OAuthTokenRequest(t *testing.T) {
	var tokenHits int32
	server := newTestEbayServer(t, &tokenHits)
	defer server.Close()

	adapter := NewEbayAdapter(server.URL, config.EbayConfig{
		ClientID:     "myid",
		ClientSecret: "mysecret",
		APIBase:      server.URL,
	}, 0, 5)

	token, err := adapter.getToken(context.Background())
	if err != nil {
		t.Fatalf("getToken() error: %v", err)
	}
	if token != "test-token" {
		t.Errorf("getToken() = %q, want test-token", token)
	}
	if got := atomic.LoadInt32(&tokenHits); got != 1 {
		t.Errorf("token endpoint hit %d times, want 1", got)
	}
}
