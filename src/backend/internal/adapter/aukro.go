package adapter

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"secondHand/src/backend/internal/domain"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/storage"
)

// AukroAdapter is the adapter for aukro.cz
type AukroAdapter struct {
	*BaseAdapter
}

// NewAukroAdapter creates a new Aukro adapter
func NewAukroAdapter(baseURL string, delayMS, timeoutSec int) *AukroAdapter {
	return &AukroAdapter{
		BaseAdapter: NewBaseAdapter("aukro.cz", baseURL, delayMS, timeoutSec),
	}
}

// SupportsSearch returns true as Aukro supports searching
func (a *AukroAdapter) SupportsSearch() bool {
	return true
}

// Search performs a search on Aukro
func (a *AukroAdapter) Search(_ context.Context, keyword string) ([]domain.Product, error) {
	products := []domain.Product{}
	seen := make(map[string]bool)

	// Create directories
	_ = os.MkdirAll("temp/report", 0755)

	pageCount := 0
	maxPages := 5 // Limit pagination

	// Base search URL
	searchURL := fmt.Sprintf("%s/vysledky-vyhledavani?text=%s&searchAll=true&subbrand=NOT_SPECIFIED",
		strings.TrimSuffix(a.baseURL, "/"),
		strings.ReplaceAll(keyword, " ", "+"))

	fmt.Printf("Aukro: Starting search for '%s'\n", keyword)

	for page := 1; page <= maxPages; page++ {
		pageURL := searchURL
		if page > 1 {
			pageURL = fmt.Sprintf("%s&page=%d", searchURL, page)
		}

		fmt.Printf("Aukro: Fetching page %d: %s\n", page, pageURL)

		c := a.collector.Clone()
		// Clone() shares the parent collector's visited-URL store by
		// design (colly v2.3.0), which would otherwise make every search
		// after the first one within this long-lived process fail with
		// "already visited" for the exact same page URLs. Give each call
		// truly independent storage instead.
		_ = c.SetStorage(&storage.InMemoryStorage{})
		pageProducts := 0

		// Look for all product cards using the custom element name
		c.OnHTML("auk-basic-item-card", func(e *colly.HTMLElement) {
			// Get the item ID
			id := e.Attr("id")
			if id == "" || !strings.HasPrefix(id, "item-") {
				return
			}

			// Check if we've seen this product
			if seen[id] {
				return
			}
			seen[id] = true

			product := domain.Product{
				ShopSource: a.Name(),
				Currency:   "CZK",
			}

			// Extract URL - it's in an <a> tag with specific attribute
			e.ForEach("a", func(_ int, a *colly.HTMLElement) {
				if href := a.Attr("href"); href != "" && product.URL == "" {
					if strings.HasPrefix(href, "http") {
						product.URL = href
					} else if strings.HasPrefix(href, "/") {
						product.URL = strings.TrimSuffix(a.Request.URL.Scheme+"://"+a.Request.URL.Host, "/") + href
					}
				}
			})

			// Extract title from h2
			e.ForEach("h2", func(_ int, h *colly.HTMLElement) {
				if title := strings.TrimSpace(h.Text); title != "" && product.Title == "" {
					product.Title = title
				}
			})

			// Extract price - isolate the dedicated price element rather than
			// using the whole card's text, which also contains the title,
			// labels, follower count, and auction countdown; cleanPriceString
			// has no way to tell "the price" apart from other digits in a
			// blob, so e.g. a title containing "31313" or a countdown like
			// "02:27" would get concatenated into the parsed price.
			e.ForEach("auk-item-card-price", func(_ int, p *colly.HTMLElement) {
				if price := parsePrice(p.Text); price != 0 && product.Price == 0 {
					product.Price = price
				}
			})

			// Try to determine type and condition from labels
			fullText := strings.ToLower(e.Text)
			if strings.Contains(fullText, "aukce") || strings.Contains(fullText, "auction") {
				product.AuctionType = domain.AuctionTypeAuction
			} else {
				product.AuctionType = domain.AuctionTypeSale
			}

			if strings.Contains(fullText, "použité") {
				product.Condition = domain.ConditionUsed
			} else if strings.Contains(fullText, "nové") {
				product.Condition = domain.ConditionNew
			} else if strings.Contains(fullText, "rozbaleno") {
				product.Condition = domain.ConditionUsed
			} else {
				product.Condition = domain.ConditionUnknown
			}

			if product.Title != "" && product.URL != "" {
				products = append(products, product)
				pageProducts++
			}
		})

		c.OnResponse(func(r *colly.Response) {
			fmt.Printf("Aukro: Page %d - Status %d, Size %d bytes\n", page, r.StatusCode, len(r.Body))

			// Save response for debugging
			filename := fmt.Sprintf("temp/report/aukro_page%d_%d.html", page, time.Now().Unix())
			_ = os.WriteFile(filename, r.Body, 0644)
			fmt.Printf("Aukro: Saved to %s\n", filename)
		})

		c.OnError(func(r *colly.Response, err error) {
			fmt.Printf("Aukro ERROR page %d: %v (Status: %d)\n", page, err, r.StatusCode)
		})

		if err := c.Visit(pageURL); err != nil {
			fmt.Printf("Aukro: Failed to visit page %d: %v\n", page, err)
			break
		}

		pageCount++

		fmt.Printf("Aukro: Page %d: Found %d products\n", page, pageProducts)

		// If no products found, we've reached the end
		if pageProducts == 0 {
			break
		}
	}

	fmt.Printf("Aukro: TOTAL: %d products across %d page(s)\n", len(products), pageCount)

	return products, nil
}

// extractProductID extracts the 10-digit product ID from Aukro URL
func extractAukroProductID(url string) string {
	re := regexp.MustCompile(`-(\d{10})$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
