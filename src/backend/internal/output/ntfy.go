package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"secondHand/internal/config"
	"secondHand/internal/domain"
)

// NtfyNotifier sends "good offer" notifications to an ntfy topic.
type NtfyNotifier struct {
	cfg    *config.NtfyConfig
	client *http.Client
}

// NewNtfyNotifier creates a new ntfy notifier.
func NewNtfyNotifier(cfg *config.NtfyConfig) *NtfyNotifier {
	return &NtfyNotifier{cfg: cfg, client: &http.Client{}}
}

// ntfyMessage is ntfy's JSON publish format (POST to the server root), used
// instead of header-based publishing so non-ASCII titles survive intact.
type ntfyMessage struct {
	Topic   string   `json:"topic"`
	Title   string   `json:"title"`
	Message string   `json:"message"`
	Click   string   `json:"click,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

// SendGoodOffer notifies about a product that matched a search's
// "good offer" thresholds.
func (n *NtfyNotifier) SendGoodOffer(product domain.Product, search domain.Search) error {
	if n.cfg.Token == "" || n.cfg.Topic == "" {
		return fmt.Errorf("ntfy token/topic not configured")
	}

	body, err := json.Marshal(ntfyMessage{
		Topic:   n.cfg.Topic,
		Title:   fmt.Sprintf("Good offer: %s", search.Keyword),
		Message: fmt.Sprintf("%s - %.2f %s\n%s", product.Title, product.Price, product.Currency, product.URL),
		Click:   product.URL,
		Tags:    []string{"moneybag"},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal ntfy message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(n.cfg.URL, "/"), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build ntfy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+n.cfg.Token)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send ntfy message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	return nil
}
