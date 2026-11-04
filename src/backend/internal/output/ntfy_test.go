package output

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"secondHand/internal/config"
	"secondHand/internal/domain"
)

func TestNtfyNotifier_SendGoodOffer(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody ntfyMessage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.NtfyConfig{URL: server.URL + "/", Topic: "snoopy", Token: "tk_test"}
	notifier := NewNtfyNotifier(cfg)

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

	if gotPath != "/" && gotPath != "" {
		t.Errorf("unexpected request path: got %q", gotPath)
	}
	if gotAuth != "Bearer tk_test" {
		t.Errorf("unexpected Authorization header: got %q", gotAuth)
	}
	if gotBody.Topic != "snoopy" {
		t.Errorf("unexpected topic: got %q", gotBody.Topic)
	}
	if !strings.Contains(gotBody.Title, "joy-con pair") {
		t.Errorf("expected title to reference search, got %q", gotBody.Title)
	}
	if !strings.Contains(gotBody.Message, "Joy-Con Pair (Neon)") || gotBody.Click != product.URL {
		t.Errorf("expected message/click to reference product, got %+v", gotBody)
	}
}

func TestNtfyNotifier_NotConfigured(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	cfg := &config.NtfyConfig{URL: server.URL, Topic: "snoopy", Token: ""}
	notifier := NewNtfyNotifier(cfg)

	err := notifier.SendGoodOffer(domain.Product{}, domain.Search{})
	if err == nil {
		t.Fatal("expected error for unconfigured ntfy token")
	}
	if called {
		t.Fatal("expected no HTTP request to be made when not configured")
	}
}

func TestNtfyNotifier_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	cfg := &config.NtfyConfig{URL: server.URL, Topic: "snoopy", Token: "tk_test"}
	notifier := NewNtfyNotifier(cfg)

	err := notifier.SendGoodOffer(domain.Product{}, domain.Search{})
	if err == nil {
		t.Fatal("expected error for non-2xx ntfy response")
	}
}
