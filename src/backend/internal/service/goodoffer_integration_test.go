package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"secondHand/src/backend/internal/config"
	"secondHand/src/backend/internal/domain"
	"secondHand/src/backend/internal/output"
	"secondHand/src/backend/internal/service"
)

// TestGoodOfferTriggersTelegramSend exercises the trigger-to-send path this
// feature's Definition of Done requires: a mock "good offer" eBay listing
// evaluates to true, and that true result actually drives an HTTP call to
// a fake Telegram endpoint - not just the two halves tested in isolation.
func TestGoodOfferTriggersTelegramSend(t *testing.T) {
	var requestReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
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

	notifier := output.NewTelegramNotifier(&config.TelegramConfig{
		BotToken: "testtoken",
		ChatID:   "12345",
		APIBase:  server.URL,
	})

	if err := notifier.SendGoodOffer(listing, search); err != nil {
		t.Fatalf("SendGoodOffer failed: %v", err)
	}

	if !requestReceived {
		t.Fatal("expected a Telegram API request to have been made")
	}
}
