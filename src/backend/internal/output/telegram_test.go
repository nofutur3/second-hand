package output

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"secondHand/src/backend/internal/config"
	"secondHand/src/backend/internal/domain"
)

func TestTelegramNotifier_SendGoodOffer(t *testing.T) {
	var gotPath string
	var gotBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := &config.TelegramConfig{BotToken: "testtoken", ChatID: "12345", APIBase: server.URL}
	notifier := NewTelegramNotifier(cfg)

	product := domain.Product{
		ShopSource: "ebay.com",
		Title:      "Joy-Con Pair (Neon)",
		Price:      45.99,
		Currency:   "USD",
		URL:        "https://www.ebay.com/itm/12345",
	}
	search := domain.Search{Keyword: "joy-con pair"}

	if err := notifier.SendGoodOffer(product, search); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/bottesttoken/sendMessage" {
		t.Errorf("unexpected request path: got %q", gotPath)
	}
	if gotBody["chat_id"] != "12345" {
		t.Errorf("unexpected chat_id: got %q", gotBody["chat_id"])
	}
	if !strings.Contains(gotBody["text"], "Joy-Con Pair (Neon)") || !strings.Contains(gotBody["text"], product.URL) {
		t.Errorf("expected message text to reference product, got %q", gotBody["text"])
	}
}

func TestTelegramNotifier_NotConfigured(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	cfg := &config.TelegramConfig{BotToken: "", ChatID: "", APIBase: server.URL}
	notifier := NewTelegramNotifier(cfg)

	err := notifier.SendGoodOffer(domain.Product{}, domain.Search{})
	if err == nil {
		t.Fatal("expected error for unconfigured Telegram credentials")
	}
	if called {
		t.Fatal("expected no HTTP request to be made when not configured")
	}
}

func TestTelegramNotifier_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	cfg := &config.TelegramConfig{BotToken: "testtoken", ChatID: "12345", APIBase: server.URL}
	notifier := NewTelegramNotifier(cfg)

	err := notifier.SendGoodOffer(domain.Product{}, domain.Search{})
	if err == nil {
		t.Fatal("expected error for non-2xx telegram API response")
	}
}
