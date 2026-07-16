package adapter

import (
	"context"
	"encoding/json"
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
	skipped := 0
	for i, link := range productLinks {
		fullURL := strings.TrimSuffix(a.baseURL, "/") + link
		fmt.Printf("  Product %d/%d: %s\n", i+1, len(productLinks), fullURL)

		// Extract basic info from URL
		product := a.extractProductFromURL(fullURL)

		// Avizo pads every search's results with paid "TOP" listings from
		// unrelated categories regardless of the search term (confirmed by
		// inspecting the real site: a clothing ad shows up in the exact
		// same markup as a genuine match for "lego mindstorms"). The site
		// itself never filters these out, so do it here instead of
		// surfacing obvious spam - and skip the detail fetch entirely for
		// anything that's clearly irrelevant, since there's no point
		// fetching a page we're about to discard.
		if !titleMatchesKeyword(product.Title, keyword) {
			skipped++
			continue
		}

		// Try to fetch more details
		detailedProduct, err := a.fetchProductDetails(ctx, fullURL)
		if err == nil && detailedProduct.Title != "" {
			product = detailedProduct
		}

		products = append(products, product)
	}

	if skipped > 0 {
		fmt.Printf("Avizo: Skipped %d unrelated result(s) (not matching '%s')\n", skipped, keyword)
	}

	fmt.Printf("Avizo: Found %d total products across %d page(s)\n", len(products), pageCount)
	return products, nil
}

// titleMatchesKeyword reports whether title plausibly relates to keyword -
// true if any distinctive keyword word (longer than two characters, not
// purely numeric) appears in the title. Purely-numeric words are excluded
// from counting on their own (a bare "255" matches tire sizes, apartment
// sizes, model numbers... across totally unrelated listings), but if
// that's genuinely the whole keyword, fall back to requiring the literal
// phrase instead of accepting everything.
func titleMatchesKeyword(title, keyword string) bool {
	title = toLower(title)
	words := strings.Fields(toLower(keyword))

	hasDistinctiveWord := false
	for _, word := range words {
		if len(word) <= 2 || isAllDigits(word) {
			continue
		}
		hasDistinctiveWord = true
		if contains(title, word) {
			return true
		}
	}

	if !hasDistinctiveWord {
		return contains(title, toLower(keyword))
	}
	return false
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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

	// The page's own schema.org JSON-LD is the authoritative source for
	// this listing - every field in it genuinely describes the product on
	// this page, unlike loose "match any element whose class contains
	// price/description/location" selectors, which also match unrelated
	// "similar listings" widgets elsewhere on the page. That's not
	// hypothetical: it's exactly how a real listing here ended up showing
	// 400 Kč instead of its actual 3 400 Kč price - the wildcard price
	// selector grabbed a sidebar recommendation's price first.
	jsonLDFound := false
	c.OnHTML(`script[type="application/ld+json"]`, func(e *colly.HTMLElement) {
		var data avizoProductLD
		if err := json.Unmarshal([]byte(e.Text), &data); err != nil {
			return
		}
		// avizo emits several ld+json blocks per page (breadcrumbs, etc.);
		// only the Product one has a real offer price.
		if data.Type != "Product" || data.Offers.Price <= 0 {
			return
		}
		jsonLDFound = true
		product.Title = data.Name
		product.Description = data.Description
		product.Price = data.Offers.Price
		if data.Offers.PriceCurrency != "" {
			product.Currency = data.Offers.PriceCurrency
		}
		product.Location = strings.TrimSpace(data.Offers.AvailableAtOrFrom.Address.AddressLocality)
		product.Condition = avizoConditionFromSchema(data.Offers.ItemCondition)
	})

	// Fallback only for the title, if the structured data above is ever
	// missing or malformed - deliberately not a fallback for price, since
	// a wrong price is worse than a missing one.
	c.OnHTML("h1", func(e *colly.HTMLElement) {
		if product.Title == "" {
			product.Title = strings.TrimSpace(e.Text)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		// Silently ignore errors
	})

	if err := c.Visit(url); err != nil {
		return product, err
	}

	c.Wait()

	if jsonLDFound {
		if product.Condition == domain.ConditionUnknown {
			product.Condition = detectCondition(product.Title + " " + product.Description)
		}
	} else if product.Title == "" {
		// No structured data and no <h1> either - fall back to guessing
		// from the URL slug.
		product = a.extractProductFromURL(url)
	} else {
		product.Condition = detectCondition(product.Title)
	}

	return product, nil
}

// avizoProductLD mirrors the subset of avizo.cz's schema.org Product+Offer
// JSON-LD block this adapter cares about.
type avizoProductLD struct {
	Type        string `json:"@type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Offers      struct {
		Price             float64 `json:"price"`
		PriceCurrency     string  `json:"priceCurrency"`
		ItemCondition     string  `json:"itemCondition"`
		AvailableAtOrFrom struct {
			Address struct {
				AddressLocality string `json:"addressLocality"`
			} `json:"address"`
		} `json:"availableAtOrFrom"`
	} `json:"offers"`
}

// avizoConditionFromSchema maps schema.org's OfferItemCondition values to
// this app's domain.Condition enum.
func avizoConditionFromSchema(schemaCondition string) domain.Condition {
	switch {
	case contains(schemaCondition, "NewCondition"):
		return domain.ConditionNew
	case contains(schemaCondition, "UsedCondition"):
		return domain.ConditionUsed
	case contains(schemaCondition, "RefurbishedCondition"):
		// No distinct "refurbished" value in this app's enum - closest match.
		return domain.ConditionLikeNew
	case contains(schemaCondition, "DamagedCondition"):
		return domain.ConditionDamaged
	default:
		return domain.ConditionUnknown
	}
}
