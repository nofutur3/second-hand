package adapter

import (
	"context"
	"fmt"
	"os"
	"secondHand/src/backend/internal/domain"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/storage"
)

// BazosAdapter is the adapter for bazos.cz
type BazosAdapter struct {
	*BaseAdapter
}

// NewBazosAdapter creates a new Bazos adapter
func NewBazosAdapter(baseURL string, delayMS, timeoutSec int) *BazosAdapter {
	return &BazosAdapter{
		BaseAdapter: NewBaseAdapter("bazos.cz", baseURL, delayMS, timeoutSec),
	}
}

// SupportsSearch returns true as Bazos supports searching
func (a *BazosAdapter) SupportsSearch() bool {
	return true
}

// Search performs a search on Bazos with pagination support
// Bazos.cz structure: Each product is in a div.inzeraty.inzeratyflex container
// Pagination: uses crz parameter (crz=20, crz=40, crz=60, etc.)
func (a *BazosAdapter) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
	products := []domain.Product{}

	// Bazos search URL - simplified
	searchURL := fmt.Sprintf("%s/search.php?hledat=%s",
		strings.TrimSuffix(a.baseURL, "/"),
		strings.ReplaceAll(keyword, " ", "+"))

	fmt.Printf("Bazos: Fetching %s\n", searchURL)

	c := a.collector.Clone()
	// Clone() shares the parent collector's visited-URL store by design
	// (colly v2.3.0), which would otherwise make every search after the
	// first one within this long-lived process fail with "already
	// visited" for the exact same search URL. Give each call truly
	// independent storage instead.
	_ = c.SetStorage(&storage.InMemoryStorage{})

	// Track visited URLs to avoid duplicates
	visitedURLs := make(map[string]bool)

	// Bazos uses div.inzeraty.inzeratyflex for each product listing
	c.OnHTML("div.inzeraty.inzeratyflex", func(e *colly.HTMLElement) {
		product := domain.Product{
			ShopSource:  a.Name(),
			AuctionType: domain.AuctionTypeSale,
			Currency:    "CZK",
			Condition:   domain.ConditionUsed,
		}

		// Title - in h2.nadpis a
		product.Title = strings.TrimSpace(e.ChildText("h2.nadpis a"))
		if product.Title == "" {
			product.Title = strings.TrimSpace(e.ChildText("h2.nadpis"))
		}
		if product.Title == "" {
			return
		}

		// URL - from h2.nadpis a href
		href := e.ChildAttr("h2.nadpis a", "href")
		if href == "" {
			href = e.ChildAttr(".inzeratynadpis a", "href")
		}

		if href != "" {
			if strings.HasPrefix(href, "http") {
				product.URL = href
			} else if strings.HasPrefix(href, "/") {
				product.URL = "https://www.bazos.cz" + href
			} else {
				product.URL = "https://www.bazos.cz/" + href
			}
		} else {
			return
		}

		// Skip if already seen this URL
		if visitedURLs[product.URL] {
			return
		}
		visitedURLs[product.URL] = true

		// Price - in div.inzeratycena span
		priceText := strings.TrimSpace(e.ChildText("div.inzeratycena span"))
		if priceText == "" {
			priceText = strings.TrimSpace(e.ChildText("div.inzeratycena b"))
		}
		if priceText == "" {
			priceText = strings.TrimSpace(e.ChildText("div.inzeratycena"))
		}
		if priceText != "" {
			product.Price = parsePrice(priceText)
		}

		// Description - in div.popis
		product.Description = strings.TrimSpace(e.ChildText("div.popis"))

		// Location - in div.inzeratylok
		location := strings.TrimSpace(e.ChildText("div.inzeratylok"))
		product.Location = location

		// Image - in .inzeratynadpis img
		imgSrc := e.ChildAttr("img.obrazek", "src")
		if imgSrc == "" {
			imgSrc = e.ChildAttr(".inzeratynadpis img", "src")
		}
		if imgSrc != "" && !strings.Contains(imgSrc, "noimg") && !strings.Contains(imgSrc, "placeholder") {
			if strings.HasPrefix(imgSrc, "http") {
				product.ImageURL = imgSrc
			} else if strings.HasPrefix(imgSrc, "//") {
				product.ImageURL = "https:" + imgSrc
			} else if strings.HasPrefix(imgSrc, "/") {
				product.ImageURL = "https://www.bazos.cz" + imgSrc
			} else {
				product.ImageURL = "https://www.bazos.cz/" + imgSrc
			}
		}

		// Debug log
		fmt.Printf("  Found: %s (%.2f CZK)\n", product.Title, product.Price)

		products = append(products, product)
	})

	// Handle pagination - look for "další" (next) link or page numbers
	// Bazos pagination format: search.php?hledat=keyword&crz=20, crz=40, etc.
	pageCount := 0
	maxPages := 5 // Limit to 5 pages (100 results) to avoid too many requests

	c.OnHTML("a[href*='crz=']", func(e *colly.HTMLElement) {
		if pageCount >= maxPages {
			return
		}

		// Check if this is a "next page" link
		linkText := strings.ToLower(strings.TrimSpace(e.Text))
		href := e.Attr("href")

		// Look for pagination links (numbers or "další" which means "next")
		if (linkText >= "1" && linkText <= "9") || strings.Contains(linkText, "dal") {
			if !strings.HasPrefix(href, "http") {
				if strings.HasPrefix(href, "/") {
					href = "https://www.bazos.cz" + href
				} else {
					href = "https://www.bazos.cz/" + href
				}
			}

			// Only follow if we haven't reached max pages
			pageCount++
			fmt.Printf("Bazos: Following pagination to page %d: %s\n", pageCount+1, href)
			c.Visit(href)
		}
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Bazos: Got response, status %d, size %d bytes\n", r.StatusCode, len(r.Body))
		// Save for debugging to temp/report directory
		filename := fmt.Sprintf("temp/report/bazos_response_%d.html", time.Now().Unix())
		if err := os.WriteFile(filename, r.Body, 0644); err == nil {
			fmt.Printf("Bazos: Saved response to %s\n", filename)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		if r != nil {
			fmt.Printf("Bazos error: Status %d, Error: %v\n", r.StatusCode, err)
		} else {
			fmt.Printf("Bazos error: %v\n", err)
		}
	})

	// Set timeout context
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(a.requestTimeout)*time.Second)
	defer cancel()

	_ = ctxWithTimeout

	if err := c.Visit(searchURL); err != nil {
		return nil, fmt.Errorf("failed to visit bazos: %w", err)
	}

	c.Wait()

	fmt.Printf("Bazos: Found %d total products across all pages\n", len(products))

	return products, nil
}
