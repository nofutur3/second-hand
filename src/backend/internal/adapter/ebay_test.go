package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"secondHand/internal/config"
	"secondHand/internal/domain"
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

// newAuctionAndShippingServer serves fixtures shaped exactly like the real
// Browse API responses captured against the live API while building this:
// a pure auction (price:null, currentBidPrice+bidCount set), a fixed-price
// item with a resolved shipping cost, and a fixed-price item whose shipping
// cost eBay hasn't resolved yet (CALCULATED with no shippingCost value).
func newAuctionAndShippingServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/identity/v1/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "test-token", "expires_in": 7200})
	})
	mux.HandleFunc("/buy/browse/v1/item_summary/search", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-EBAY-C-ENDUSERCTX"); got != "contextualLocation=country=CZ,zip=58601" {
			t.Errorf("X-EBAY-C-ENDUSERCTX = %q, want contextualLocation=country=CZ,zip=58601", got)
		}
		if got := r.Header.Get("Accept-Language"); got != "en-US" {
			t.Errorf("Accept-Language = %q, want en-US (otherwise eBay machine-translates titles for a CZ destination)", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"itemSummaries": []map[string]interface{}{
				{
					"itemWebUrl":      "https://www.ebay.com/itm/10",
					"title":           "Auction item",
					"price":           nil,
					"currentBidPrice": map[string]string{"value": "2068.22", "currency": "CZK"},
					"bidCount":        35,
					"buyingOptions":   []string{"AUCTION"},
					"itemEndDate":     "2026-08-18T14:08:58Z",
				},
				{
					"itemWebUrl":    "https://www.ebay.com/itm/11",
					"title":         "Fixed price with resolved shipping",
					"price":         map[string]string{"value": "100.00", "currency": "CZK"},
					"buyingOptions": []string{"FIXED_PRICE"},
					"shippingOptions": []map[string]interface{}{
						{"shippingCostType": "FIXED", "shippingCost": map[string]string{"value": "50.00", "currency": "CZK"}},
					},
				},
				{
					"itemWebUrl":      "https://www.ebay.com/itm/12",
					"title":           "Fixed price, shipping not yet resolved",
					"price":           map[string]string{"value": "200.00", "currency": "CZK"},
					"buyingOptions":   []string{"FIXED_PRICE"},
					"shippingOptions": []map[string]interface{}{{"shippingCostType": "CALCULATED"}},
				},
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestEbayAdapter_AuctionUsesCurrentBidPrice(t *testing.T) {
	server := newAuctionAndShippingServer(t)
	defer server.Close()

	adapter := NewEbayAdapter(server.URL, config.EbayConfig{
		ClientID: "id", ClientSecret: "secret", APIBase: server.URL,
		ShipToCountry: "CZ", ShipToPostalCode: "58601",
	}, 0, 5)

	products, err := adapter.Search(context.Background(), "auction item")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	auction := products[0]
	if auction.Price != 2068.22 {
		t.Errorf("auction Price = %v, want 2068.22 (from currentBidPrice, not the null price field)", auction.Price)
	}
	if auction.Currency != "CZK" {
		t.Errorf("auction Currency = %q, want CZK", auction.Currency)
	}
	if auction.BidCount != 35 {
		t.Errorf("auction BidCount = %d, want 35", auction.BidCount)
	}
}

func TestEbayAdapter_ShippingCost(t *testing.T) {
	server := newAuctionAndShippingServer(t)
	defer server.Close()

	adapter := NewEbayAdapter(server.URL, config.EbayConfig{
		ClientID: "id", ClientSecret: "secret", APIBase: server.URL,
		ShipToCountry: "CZ", ShipToPostalCode: "58601",
	}, 0, 5)

	products, err := adapter.Search(context.Background(), "shipping test")
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	resolved := products[1]
	if resolved.ShippingCost == nil || *resolved.ShippingCost != 50.0 {
		t.Errorf("resolved item ShippingCost = %v, want 50.0", resolved.ShippingCost)
	}

	unresolved := products[2]
	if unresolved.ShippingCost != nil {
		t.Errorf("unresolved CALCULATED item ShippingCost = %v, want nil", *unresolved.ShippingCost)
	}
}

func TestEbayAdapter_GetDescription(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/identity/v1/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "test-token", "expires_in": 7200})
	})
	mux.HandleFunc("/buy/browse/v1/item/get_item_by_legacy_id", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("legacy_item_id"); got != "147512920592" {
			t.Errorf("legacy_item_id = %q, want 147512920592", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"description": `<p style="margin:0">Selling <b>just the lens</b> &amp; case</p><br>No body included.`,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := NewEbayAdapter(server.URL, config.EbayConfig{
		ClientID: "id", ClientSecret: "secret", APIBase: server.URL,
	}, 0, 5)

	desc, err := adapter.GetDescription(context.Background(), "https://www.ebay.com/itm/147512920592?_skw=lens")
	if err != nil {
		t.Fatalf("GetDescription() error: %v", err)
	}
	if want := "Selling just the lens & case No body included."; desc != want {
		t.Errorf("GetDescription() = %q, want %q", desc, want)
	}
}

func TestEbayAdapter_GetDescription_UnrecognizedURL(t *testing.T) {
	adapter := NewEbayAdapter("", config.EbayConfig{ClientID: "id", ClientSecret: "secret", APIBase: "https://example.com"}, 0, 5)

	_, err := adapter.GetDescription(context.Background(), "https://www.ebay.com/some-other-page")
	if err != ErrEbayItemURLNotRecognized {
		t.Errorf("GetDescription() error = %v, want ErrEbayItemURLNotRecognized", err)
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"tags and entities", `<p>Hello &amp; welcome</p>`, "Hello & welcome"},
		{"collapses whitespace across tags", "<div>a</div>\n<div>b</div>", "a b"},
		{"plain text unchanged", "just text", "just text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripHTML(tt.input); got != tt.want {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
