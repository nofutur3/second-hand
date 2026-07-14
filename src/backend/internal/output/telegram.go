package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"secondHand/src/backend/internal/config"
	"secondHand/src/backend/internal/domain"
)

// TelegramNotifier sends "good offer" notifications via a Telegram bot.
type TelegramNotifier struct {
	cfg    *config.TelegramConfig
	client *http.Client
}

// NewTelegramNotifier creates a new Telegram notifier.
func NewTelegramNotifier(cfg *config.TelegramConfig) *TelegramNotifier {
	return &TelegramNotifier{cfg: cfg, client: &http.Client{}}
}

type telegramSendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// SendGoodOffer notifies about a product that matched a search's
// "good offer" thresholds.
func (t *TelegramNotifier) SendGoodOffer(product domain.Product, search domain.Search) error {
	if t.cfg.BotToken == "" || t.cfg.ChatID == "" {
		return fmt.Errorf("Telegram credentials not configured")
	}

	text := fmt.Sprintf(
		"Good offer for \"%s\": %s - %.2f %s\n%s",
		search.Keyword, product.Title, product.Price, product.Currency, product.URL,
	)

	body, err := json.Marshal(telegramSendMessageRequest{
		ChatID: t.cfg.ChatID,
		Text:   text,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", t.cfg.APIBase, t.cfg.BotToken)
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
