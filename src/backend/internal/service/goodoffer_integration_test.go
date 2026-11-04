package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"secondHand/internal/config"
	"secondHand/internal/domain"
	"secondHand/internal/output"
	"secondHand/internal/service"
)

// TestGoodOfferTriggersNtfySend exercises the trigger-to-send path this
// feature's Definition of Done requires: a mock "good offer" eBay listing
// evaluates to true, and that true result actually drives an HTTP call to
// a fake ntfy endpoint - not just the two halves tested in isolation.
func TestGoodOfferTriggersNtfySend(t *testing.T) {
	var requestReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	maxPrice := 60.0
	search := domain.Search{Keyword: "joy-con pair", MaxPrice: &maxPrice}

	listing := domain.Product{
		ShopSource: "ebay.com",
		Title:      "Nintendo Switch Joy-Con Pair - Neon",
		Price:      49.99,
		Currency:   "USD",
		URL:        "https://www.ebay.com/itm/999",
	}

	isGoodOffer := service.EvaluateGoodOffer(search, listing, nil)
	if !isGoodOffer {
		t.Fatal("expected mock listing to evaluate as a good offer")
	}

	notifier := output.NewNtfyNotifier(&config.NtfyConfig{
		URL:   server.URL,
		Topic: "snoopy",
		Token: "tk_test",
	})

	if err := notifier.SendGoodOffer(listing, search); err != nil {
		t.Fatalf("SendGoodOffer failed: %v", err)
	}

	if !requestReceived {
		t.Fatal("expected an ntfy request to have been made")
	}
}
