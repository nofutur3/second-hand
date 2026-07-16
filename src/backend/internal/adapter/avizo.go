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

// AvizoAdapter is the adapter for avizo.cz
// Note: Avizo.cz is heavily JavaScript-based
type AvizoAdapter struct {
	*BaseAdapter
}

// NewAvizoAdapter creates a new Avizo adapter
func NewAvizoAdapter(baseURL string, delayMS, timeoutSec int) *AvizoAdapter {
	return &AvizoAdapter{
		BaseAdapter: NewBaseAdapter("avizo.cz", baseURL, delayMS, timeoutSec),
	}
}

// SupportsSearch returns true as Avizo supports searching
func (a *AvizoAdapter) SupportsSearch() bool {
	return true
}

// Search performs a search on Avizo with pagination support
// URL format: /inzerce/keyword or /inzerce/keyword/2/ for page 2
func (a *AvizoAdapter) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
	products := []domain.Product{}

	// Avizo search URL format: /inzerce/keyword
	searchURL := fmt.Sprintf("%s/inzerce/%s",
		strings.TrimSuffix(a.baseURL, "/"),
		strings.ReplaceAll(keyword, " ", "-"))

	fmt.Printf("Avizo: Fetching %s\n", searchURL)

	c := a.collector.Clone()
	// Clone() shares the parent collector's visited-URL store by design
	// (colly v2.3.0), which would otherwise make every search after the
	// first one within this long-lived process fail with "already
	// visited" for the exact same search URL. Give each call truly
	// independent storage instead.
	_ = c.SetStorage(&storage.InMemoryStorage{})

	// Track product IDs across all pages
	visitedIDs := make(map[string]bool)
	productLinks := []string{}
	visitedPages := make(map[string]bool)
	pageCount := 0
	maxPages := 5

	c.OnResponse(func(r *colly.Response) {
		pageCount++
		fmt.Printf("Avizo: Got response for page %d, status %d, size %d bytes\n", pageCount, r.StatusCode, len(r.Body))

		// Save for debugging
		filename := fmt.Sprintf("temp/report/avizo_page%d_%d.html", pageCount, time.Now().Unix())
		os.MkdirAll("temp/report", 0755)
		if err := os.WriteFile(filename, r.Body, 0644); err == nil {
			fmt.Printf("Avizo: Saved response to %s\n", filename)
		}

		// Extract product links from HTML/JavaScript
		// Pattern: /category/product-name-12345678.html
		re := regexp.MustCompile(`/([\w-]+)/([\w-]+)-(\d{8})\.html`)
		matches := re.FindAllStringSubmatch(string(r.Body), -1)

		linksThisPage := 0
		for _, match := range matches {
			if len(match) >= 4 {
				productID := match[3]
				if !visitedIDs[productID] {
					visitedIDs[productID] = true
					link := match[0] // Full match: /prace/strojnik-pasoveho-rypadla-19725372.html
					productLinks = append(productLinks, link)
					linksThisPage++
				}
			}
		}

		fmt.Printf("Avizo: Page %d - extracted %d new products (total: %d)\n", pageCount, linksThisPage, len(productLinks))

		// Extract pagination links
		// Pattern: /inzerce/keyword/2/ or /inzerce/keyword/3/
		if pageCount < maxPages {
			paginationRe := regexp.MustCompile(`/inzerce/` + regexp.QuoteMeta(strings.ReplaceAll(keyword, " ", "-")) + `/(\d+)/?`)
			paginationMatches := paginationRe.FindAllStringSubmatch(string(r.Body), -1)

			for _, pmatch := range paginationMatches {
				if len(pmatch) >= 2 {
					pageLink := pmatch[0]

					if !visitedPages[pageLink] {
						visitedPages[pageLink] = true
						fullURL := strings.TrimSuffix(a.baseURL, "/") + pageLink
						fmt.Printf("Avizo: Found pagination link: %s\n", fullURL)
						c.Visit(fullURL)
					}
				}
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		if r != nil {
			fmt.Printf("Avizo error: Status %d, Error: %v\n", r.StatusCode, err)
		} else {
			fmt.Printf("Avizo error: %v\n", err)
		}
	})

	// Visit the search page
	if err := c.Visit(searchURL); err != nil {
		return nil, fmt.Errorf("failed to visit avizo: %w", err)
	}

	c.Wait()

	fmt.Printf("Avizo: Processing %d products from %d page(s)\n", len(productLinks), pageCount)

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

	fmt.Printf("Avizo: Found %d total products across %d page(s)\n", len(products), pageCount)
	return products, nil
}

// extractProductFromURL extracts basic product info from the URL
func (a *AvizoAdapter) extractProductFromURL(url string) domain.Product {
	product := domain.Product{
		ShopSource:  a.Name(),
		URL:         url,
		AuctionType: domain.AuctionTypeSale,
		Currency:    "CZK",
		Condition:   domain.ConditionUsed,
	}

	// Extract title from URL: /prace/strojnik-pasoveho-rypadla-19725372.html
	// -> "strojnik pasoveho rypadla"
	re := regexp.MustCompile(`/([\w-]+)/([\w-]+)-(\d{8})\.html`)
	match := re.FindStringSubmatch(url)
	if len(match) >= 3 {
		titleSlug := match[2]
		// Convert slug to title
		product.Title = strings.ReplaceAll(titleSlug, "-", " ")
		product.Title = strings.Title(strings.ToLower(product.Title))
	}

	return product
}

// fetchProductDetails tries to fetch details from a product page
func (a *AvizoAdapter) fetchProductDetails(ctx context.Context, url string) (domain.Product, error) {
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
