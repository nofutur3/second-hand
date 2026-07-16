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

// SbazarAdapter is the adapter for sbazar.cz
// Note: Sbazar.cz is heavily JavaScript-based and may require API access or headless browser
type SbazarAdapter struct {
	*BaseAdapter
}

// NewSbazarAdapter creates a new Sbazar adapter
func NewSbazarAdapter(baseURL string, delayMS, timeoutSec int) *SbazarAdapter {
	return &SbazarAdapter{
		BaseAdapter: NewBaseAdapter("sbazar.cz", baseURL, delayMS, timeoutSec),
	}
}

// SupportsSearch returns true as Sbazar supports searching
func (a *SbazarAdapter) SupportsSearch() bool {
	return true
}

// Search performs a search on Sbazar with pagination support
// Note: Sbazar.cz is JavaScript-heavy. This adapter extracts what it can from the raw HTML/JS
func (a *SbazarAdapter) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
	products := []domain.Product{}

	// Sbazar search URL format: /hledej/keyword
	searchURL := fmt.Sprintf("%s/hledej/%s",
		strings.TrimSuffix(a.baseURL, "/"),
		strings.ReplaceAll(keyword, " ", "-"))

	fmt.Printf("Sbazar: Fetching %s\n", searchURL)

	c := a.collector.Clone()
	// Clone() shares the parent collector's visited-URL store by design
	// (colly v2.3.0), which would otherwise make every search after the
	// first one within this long-lived process fail with "already
	// visited" for the exact same search URL. Give each call truly
	// independent storage instead.
	_ = c.SetStorage(&storage.InMemoryStorage{})

	// Track product IDs we've seen across all pages
	visitedIDs := make(map[string]bool)
	productLinks := []string{}
	visitedPages := make(map[string]bool)
	pageCount := 0
	maxPages := 5 // Limit to first 5 pages

	// Sbazar loads content via JavaScript, but we can extract links from the page source
	c.OnResponse(func(r *colly.Response) {
		pageCount++
		fmt.Printf("Sbazar: Got response for page %d, status %d, size %d bytes\n", pageCount, r.StatusCode, len(r.Body))

		// Save for debugging
		filename := fmt.Sprintf("temp/report/sbazar_page%d_%d.html", pageCount, time.Now().Unix())
		os.MkdirAll("temp/report", 0755)
		if err := os.WriteFile(filename, r.Body, 0644); err == nil {
			fmt.Printf("Sbazar: Saved response to %s\n", filename)
		}

		// Extract product links from the HTML source
		// Look for patterns like: /inzerat/183060347-hemingway
		re := regexp.MustCompile(`/inzerat/(\d+)-([^"'\s<>]+)`)
		matches := re.FindAllStringSubmatch(string(r.Body), -1)

		linksThisPage := 0
		for _, match := range matches {
			if len(match) >= 3 {
				productID := match[1]
				if !visitedIDs[productID] {
					visitedIDs[productID] = true
					link := match[0] // Full match: /inzerat/183060347-hemingway
					productLinks = append(productLinks, link)
					linksThisPage++
				}
			}
		}

		fmt.Printf("Sbazar: Page %d - extracted %d new products (total: %d)\n", pageCount, linksThisPage, len(productLinks))

		// Extract pagination links via regex since Sbazar is JavaScript-heavy
		// Pattern: /hledej/keyword/.../nejnovejsi/2, /nejnovejsi/3, etc.
		if pageCount < maxPages {
			paginationRe := regexp.MustCompile(`/hledej/[^"'\s]+/nejnovejsi/(\d+)`)
			paginationMatches := paginationRe.FindAllStringSubmatch(string(r.Body), -1)

			for _, pmatch := range paginationMatches {
				if len(pmatch) >= 2 {
					pageLink := pmatch[0]

					// Avoid duplicates
					if !visitedPages[pageLink] {
						visitedPages[pageLink] = true
						fullURL := strings.TrimSuffix(a.baseURL, "/") + pageLink
						fmt.Printf("Sbazar: Found pagination link: %s\n", fullURL)
						c.Visit(fullURL)
					}
				}
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		if r != nil {
			fmt.Printf("Sbazar error: Status %d, Error: %v\n", r.StatusCode, err)
		} else {
			fmt.Printf("Sbazar error: %v\n", err)
		}
	})

	// Visit the search page
	if err := c.Visit(searchURL); err != nil {
		return nil, fmt.Errorf("failed to visit sbazar: %w", err)
	}

	c.Wait()

	fmt.Printf("Sbazar: Processing %d products from %d page(s)\n", len(productLinks), pageCount)

	// Now fetch details for each product - NO LIMIT
	for i, link := range productLinks {
		fullURL := strings.TrimSuffix(a.baseURL, "/") + link
		fmt.Printf("  Product %d/%d: %s\n", i+1, len(productLinks), fullURL)

		// Extract basic info from URL
		product := a.extractProductFromURL(fullURL)

		// Try to fetch more details
		detailedProduct, err := a.fetchProductDetails(ctx, fullURL)
		if err == nil && detailedProduct.Title != "" {
			product = detailedProduct
		}

		products = append(products, product)
	}

	fmt.Printf("Sbazar: Found %d total products across %d page(s)\n", len(products), pageCount)
	return products, nil
}

// extractProductFromURL extracts basic product info from the URL
func (a *SbazarAdapter) extractProductFromURL(url string) domain.Product {
	product := domain.Product{
		ShopSource:  a.Name(),
		URL:         url,
		AuctionType: domain.AuctionTypeSale,
		Currency:    "CZK",
		Condition:   domain.ConditionUsed,
	}

	// Extract title from URL: /inzerat/183060347-hemingway -> "hemingway"
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove ID prefix: "183060347-hemingway" -> "hemingway"
		if idx := strings.Index(lastPart, "-"); idx > 0 {
			titleSlug := lastPart[idx+1:]
			// Convert slug to title
			product.Title = strings.ReplaceAll(titleSlug, "-", " ")
			product.Title = strings.Title(strings.ToLower(product.Title))
		}
	}

	return product
}

// fetchProductDetails tries to fetch details from a product page
func (a *SbazarAdapter) fetchProductDetails(ctx context.Context, url string) (domain.Product, error) {
	product := domain.Product{
		ShopSource:  a.Name(),
		URL:         url,
		AuctionType: domain.AuctionTypeSale,
		Currency:    "CZK",
		Condition:   domain.ConditionUsed,
	}

	c := a.collector.Clone()
	_ = c.SetStorage(&storage.InMemoryStorage{}) // see Search()'s comment on why

	// Try to extract what we can from the HTML
	c.OnHTML("h1", func(e *colly.HTMLElement) {
		if product.Title == "" {
			product.Title = strings.TrimSpace(e.Text)
		}
	})

	// Look for price in various formats
	c.OnHTML("*[class*='price'], *[class*='Price'], *[class*='cena']", func(e *colly.HTMLElement) {
		if product.Price == 0 {
			priceText := strings.TrimSpace(e.Text)
			product.Price = parsePrice(priceText)
		}
	})

	// Description
	c.OnHTML("*[class*='description'], *[class*='Description'], *[class*='popis']", func(e *colly.HTMLElement) {
		if product.Description == "" {
			product.Description = strings.TrimSpace(e.Text)
		}
	})

	// Location
	c.OnHTML("*[class*='location'], *[class*='Location'], *[class*='lokalita']", func(e *colly.HTMLElement) {
		if product.Location == "" {
			product.Location = strings.TrimSpace(e.Text)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		// Silently ignore errors
	})

	if err := c.Visit(url); err != nil {
		return product, err
	}

	c.Wait()

	// If we didn't get a title, extract from URL
	if product.Title == "" {
		product = a.extractProductFromURL(url)
	} else {
		// Detect condition
		product.Condition = detectCondition(product.Title + " " + product.Description)
	}

	return product, nil
}
