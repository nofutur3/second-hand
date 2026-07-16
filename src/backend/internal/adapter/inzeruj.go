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

// InzerujAdapter is the adapter for inzeruj.cz
type InzerujAdapter struct {
	*BaseAdapter
}

// NewInzerujAdapter creates a new Inzeruj adapter
func NewInzerujAdapter(baseURL string, delayMS, timeoutSec int) *InzerujAdapter {
	return &InzerujAdapter{
		BaseAdapter: NewBaseAdapter("inzeruj.cz", baseURL, delayMS, timeoutSec),
	}
}

// SupportsSearch returns true as Inzeruj supports searching
func (a *InzerujAdapter) SupportsSearch() bool {
	return true
}

// Search performs a search on Inzeruj
// URL format: /search?title=keyword
// Note: No pagination on Inzeruj
func (a *InzerujAdapter) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
	products := []domain.Product{}

	// Inzeruj search URL format: /search?title=keyword
	searchURL := fmt.Sprintf("%s/search?title=%s",
		strings.TrimSuffix(a.baseURL, "/"),
		strings.ReplaceAll(keyword, " ", "+"))

	fmt.Printf("Inzeruj: Fetching %s\n", searchURL)

	c := a.collector.Clone()
	// Clone() shares the parent collector's visited-URL store by design
	// (colly v2.3.0), which would otherwise make every search after the
	// first one within this long-lived process fail with "already
	// visited" for the exact same search URL. Give each call truly
	// independent storage instead.
	_ = c.SetStorage(&storage.InMemoryStorage{})

	// Track product IDs
	visitedIDs := make(map[string]bool)
	productLinks := []string{}

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Inzeruj: Got response, status %d, size %d bytes\n", r.StatusCode, len(r.Body))

		// Save for debugging
		filename := fmt.Sprintf("temp/report/inzeruj_response_%d.html", time.Now().Unix())
		os.MkdirAll("temp/report", 0755)
		if err := os.WriteFile(filename, r.Body, 0644); err == nil {
			fmt.Printf("Inzeruj: Saved response to %s\n", filename)
		}

		// Extract product links from HTML/JavaScript
		// Pattern: /inzerat/category/subcategory/product-name-123456.html
		re := regexp.MustCompile(`/inzerat/([\w-]+)/([\w-]+)/([\w-]+)-(\d{6})\.html`)
		matches := re.FindAllStringSubmatch(string(r.Body), -1)

		for _, match := range matches {
			if len(match) >= 5 {
				productID := match[4]
				if !visitedIDs[productID] {
					visitedIDs[productID] = true
					link := match[0] // Full match
					productLinks = append(productLinks, link)
				}
			}
		}

		fmt.Printf("Inzeruj: Extracted %d product links\n", len(productLinks))
	})

	c.OnError(func(r *colly.Response, err error) {
		if r != nil {
			fmt.Printf("Inzeruj error: Status %d, Error: %v\n", r.StatusCode, err)
		} else {
			fmt.Printf("Inzeruj error: %v\n", err)
		}
	})

	// Visit the search page
	if err := c.Visit(searchURL); err != nil {
		return nil, fmt.Errorf("failed to visit inzeruj: %w", err)
	}

	c.Wait()

	fmt.Printf("Inzeruj: Processing %d products\n", len(productLinks))

	// Fetch details for each product
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

	fmt.Printf("Inzeruj: Found %d total products\n", len(products))
	return products, nil
}

// extractProductFromURL extracts basic product info from the URL
func (a *InzerujAdapter) extractProductFromURL(url string) domain.Product {
	product := domain.Product{
		ShopSource:  a.Name(),
		URL:         url,
		AuctionType: domain.AuctionTypeSale,
		Currency:    "CZK",
		Condition:   domain.ConditionUsed,
	}

	// Extract title from URL: /inzerat/stroje/stavebni-stroje/mikro-rypadlo-jansen-r-mb-500-645576.html
	// -> "mikro rypadlo jansen r mb 500"
	re := regexp.MustCompile(`/inzerat/([\w-]+)/([\w-]+)/([\w-]+)-(\d{6})\.html`)
	match := re.FindStringSubmatch(url)
	if len(match) >= 4 {
		titleSlug := match[3]
		// Convert slug to title
		product.Title = strings.ReplaceAll(titleSlug, "-", " ")
		product.Title = strings.Title(strings.ToLower(product.Title))
	}

	return product
}

// fetchProductDetails tries to fetch details from a product page
func (a *InzerujAdapter) fetchProductDetails(ctx context.Context, url string) (domain.Product, error) {
	product := domain.Product{
		ShopSource:  a.Name(),
		URL:         url,
		AuctionType: domain.AuctionTypeSale,
		Currency:    "CZK",
		Condition:   domain.ConditionUsed,
	}

	c := a.collector.Clone()
	_ = c.SetStorage(&storage.InMemoryStorage{}) // see Search()'s comment on why

	// Try to extract from HTML if available
	c.OnHTML("h1", func(e *colly.HTMLElement) {
		if product.Title == "" {
			product.Title = strings.TrimSpace(e.Text)
		}
	})

	// Look for price
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
