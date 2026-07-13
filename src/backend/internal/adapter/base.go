package adapter

import (
	"context"
	"fmt"
	"net/http"
	"secondHand/src/backend/internal/domain"
	"time"

	"github.com/gocolly/colly/v2"
)

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	name           string
	baseURL        string
	collector      *colly.Collector
	requestDelay   time.Duration
	requestTimeout time.Duration
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(name, baseURL string, delayMS, timeoutSec int) *BaseAdapter {
	c := colly.NewCollector(
		colly.AllowedDomains(extractDomain(baseURL)),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	c.SetRequestTimeout(time.Duration(timeoutSec) * time.Second)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       time.Duration(delayMS) * time.Millisecond,
		RandomDelay: time.Duration(delayMS/2) * time.Millisecond,
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request to %s failed: %v\n", r.Request.URL, err)
	})

	return &BaseAdapter{
		name:           name,
		baseURL:        baseURL,
		collector:      c,
		requestDelay:   time.Duration(delayMS) * time.Millisecond,
		requestTimeout: time.Duration(timeoutSec) * time.Second,
	}
}

// Name returns the adapter name
func (a *BaseAdapter) Name() string {
	return a.name
}

// extractDomain extracts domain from URL
func extractDomain(url string) string {
	// Simple domain extraction
	if len(url) > 8 && url[:8] == "https://" {
		url = url[8:]
	} else if len(url) > 7 && url[:7] == "http://" {
		url = url[7:]
	}

	// Remove path
	for i, c := range url {
		if c == '/' {
			url = url[:i]
			break
		}
	}

	return url
}

// makeRequest makes an HTTP request with timeout
func makeRequest(ctx context.Context, url string, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return client.Do(req)
}

// parsePrice attempts to parse a price string
func parsePrice(priceStr string) float64 {
	// Remove common currency symbols and whitespace
	priceStr = cleanPriceString(priceStr)

	var price float64
	fmt.Sscanf(priceStr, "%f", &price)
	return price
}

// cleanPriceString removes currency symbols and handles Czech number format
// Czech format: "1 500,50" or "1 500.50" or "1500"
func cleanPriceString(s string) string {
	result := ""
	decimalSeparator := ""

	// First pass: identify if there's a decimal separator (last comma or dot with <=2 digits after)
	digitCount := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] >= '0' && s[i] <= '9' {
			digitCount++
		} else if s[i] == ',' || s[i] == '.' {
			if digitCount <= 2 && digitCount > 0 {
				decimalSeparator = string(s[i])
			}
			break
		} else if s[i] == ' ' {
			// Continue looking for separator
			continue
		} else {
			break
		}
	}

	// Second pass: build the number
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result += string(c)
		} else if (c == ',' || c == '.') && string(c) == decimalSeparator {
			result += "."
		}
		// Skip spaces and other characters
	}

	return result
}

// detectCondition attempts to detect condition from text
func detectCondition(text string) domain.Condition {
	lowerText := toLower(text)

	// Check for "jako nový" first (more specific)
	if contains(lowerText, "jako nov") {
		return domain.ConditionLikeNew
	}
	if contains(lowerText, "nov") {
		return domain.ConditionNew
	}
	if contains(lowerText, "pou") || contains(lowerText, "bazar") {
		return domain.ConditionUsed
	}
	if contains(lowerText, "po") && contains(lowerText, "zen") {
		return domain.ConditionDamaged
	}

	return domain.ConditionUnknown
}

func toLower(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0
}

func indexOfSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
