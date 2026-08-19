package adapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"secondHand/internal/config"
	"secondHand/internal/domain"
)

// ebayTokenExpiryBuffer is how long before actual expiry a cached OAuth2 token is treated as stale.
const ebayTokenExpiryBuffer = 60 * time.Second

// EbayAdapter searches eBay via the Browse API (OAuth2 client-credentials), not scraping.
type EbayAdapter struct {
	apiBase          string
	clientID         string
	clientSecret     string
	shipToCountry    string
	shipToPostalCode string
	httpClient       *http.Client
	delay            time.Duration

	tokenMu     sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewEbayAdapter creates a new eBay Browse API adapter. url is accepted for
// signature consistency with the other adapter constructors (registry
// dispatch matches on it) but requests go to cfg.APIBase, not url.
func NewEbayAdapter(url string, cfg config.EbayConfig, delayMS, timeoutSec int) *EbayAdapter {
	return &EbayAdapter{
		apiBase:          cfg.APIBase,
		clientID:         cfg.ClientID,
		clientSecret:     cfg.ClientSecret,
		shipToCountry:    cfg.ShipToCountry,
		shipToPostalCode: cfg.ShipToPostalCode,
		httpClient:       &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		delay:            time.Duration(delayMS) * time.Millisecond,
	}
}

// Name returns the adapter's shop identifier.
func (a *EbayAdapter) Name() string {
	return "ebay.com"
}

// SupportsSearch returns true.
func (a *EbayAdapter) SupportsSearch() bool {
	return true
}

type ebayTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// getToken returns a cached OAuth2 app token, refreshing it if missing or near expiry.
func (a *EbayAdapter) getToken(ctx context.Context) (string, error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.accessToken != "" && time.Now().Before(a.tokenExpiry) {
		return a.accessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "https://api.ebay.com/oauth/api_scope")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiBase+"/identity/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("building eBay OAuth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	auth := base64.StdEncoding.EncodeToString([]byte(a.clientID + ":" + a.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting eBay OAuth token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading eBay OAuth token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("eBay OAuth token request failed: status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp ebayTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing eBay OAuth token response: %w", err)
	}

	a.accessToken = tokenResp.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn)*time.Second - ebayTokenExpiryBuffer)

	return a.accessToken, nil
}

type ebaySearchResponse struct {
	ItemSummaries []ebayItemSummary `json:"itemSummaries"`
}

type ebayItemSummary struct {
	ItemWebURL      string               `json:"itemWebUrl"`
	Title           string               `json:"title"`
	Price           *ebayPrice           `json:"price"`
	CurrentBidPrice *ebayPrice           `json:"currentBidPrice"`
	BidCount        int                  `json:"bidCount"`
	Condition       string               `json:"condition"`
	Image           ebayImage            `json:"image"`
	ItemLocation    ebayItemLocation     `json:"itemLocation"`
	Seller          ebaySeller           `json:"seller"`
	BuyingOptions   []string             `json:"buyingOptions"`
	ItemEndDate     string               `json:"itemEndDate"`
	ShippingOptions []ebayShippingOption `json:"shippingOptions"`
}

type ebayPrice struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type ebayShippingOption struct {
	ShippingCostType string     `json:"shippingCostType"`
	ShippingCost     *ebayPrice `json:"shippingCost"`
}

type ebayImage struct {
	ImageURL string `json:"imageUrl"`
}

type ebayItemLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

type ebaySeller struct {
	Username string `json:"username"`
}

// Search queries eBay Browse API's item_summary/search endpoint for keyword.
func (a *EbayAdapter) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
	if a.delay > 0 {
		time.Sleep(a.delay)
	}

	token, err := a.getToken(ctx)
	if err != nil {
		return nil, err
	}

	searchURL := fmt.Sprintf("%s/buy/browse/v1/item_summary/search?q=%s", a.apiBase, url.QueryEscape(keyword))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building eBay search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	a.setBuyerContextHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting eBay search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading eBay search response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eBay search request failed: status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp ebaySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("parsing eBay search response: %w", err)
	}

	products := make([]domain.Product, 0, len(searchResp.ItemSummaries))
	for _, item := range searchResp.ItemSummaries {
		products = append(products, mapEbayItem(item))
	}

	return products, nil
}

// setBuyerContextHeaders tells the Browse API which marketplace and buyer
// destination to resolve price/shipping figures for - without this,
// calculated shipping costs come back unresolved (no value at all) and
// price stays in the seller's own currency instead of the buyer's.
// Accept-Language pins titles/descriptions to English: contextualLocation
// alone makes eBay machine-translate them into the destination country's
// language (confirmed live - CZ destination without this header yields
// Czech-translated titles), which is more confusing than helpful here.
func (a *EbayAdapter) setBuyerContextHeaders(req *http.Request) {
	req.Header.Set("X-EBAY-C-MARKETPLACE-ID", "EBAY_US")
	req.Header.Set("Accept-Language", "en-US")
	if a.shipToCountry != "" && a.shipToPostalCode != "" {
		req.Header.Set("X-EBAY-C-ENDUSERCTX", fmt.Sprintf("contextualLocation=country=%s,zip=%s", a.shipToCountry, a.shipToPostalCode))
	}
}

// ebayLegacyItemIDPattern extracts the numeric legacy item ID from an eBay
// item URL (e.g. "https://www.ebay.com/itm/147512920592?...").
var ebayLegacyItemIDPattern = regexp.MustCompile(`/itm/(\d+)`)

// ebayHTMLTagPattern strips HTML tags from a raw item description; used
// instead of a full HTML sanitizer since the result is only ever rendered
// as plain text, never as HTML.
var ebayHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)

type ebayGetItemResponse struct {
	Description string `json:"description"`
}

// ErrEbayItemURLNotRecognized is returned by GetDescription when itemURL
// doesn't contain a recognizable eBay legacy item ID.
var ErrEbayItemURLNotRecognized = fmt.Errorf("eBay item URL not recognized")

// GetDescription fetches an eBay listing's full description on demand (the
// item_summary/search response used by Search doesn't include it - only a
// per-item call does). Not used during bulk scraping/cron runs to avoid
// burning through the Browse API's per-app daily call quota.
func (a *EbayAdapter) GetDescription(ctx context.Context, itemURL string) (string, error) {
	match := ebayLegacyItemIDPattern.FindStringSubmatch(itemURL)
	if match == nil {
		return "", ErrEbayItemURLNotRecognized
	}
	legacyItemID := match[1]

	token, err := a.getToken(ctx)
	if err != nil {
		return "", err
	}

	getItemURL := fmt.Sprintf("%s/buy/browse/v1/item/get_item_by_legacy_id?legacy_item_id=%s", a.apiBase, url.QueryEscape(legacyItemID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getItemURL, nil)
	if err != nil {
		return "", fmt.Errorf("building eBay get-item request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	a.setBuyerContextHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting eBay item detail: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading eBay item detail response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("eBay item detail request failed: status %d: %s", resp.StatusCode, string(body))
	}

	var itemResp ebayGetItemResponse
	if err := json.Unmarshal(body, &itemResp); err != nil {
		return "", fmt.Errorf("parsing eBay item detail response: %w", err)
	}

	return stripHTML(itemResp.Description), nil
}

// stripHTML reduces a raw HTML description to plain text - the frontend
// only ever displays this as text, never renders it as HTML, so there's no
// need to sanitize and preserve markup, just discard it.
func stripHTML(raw string) string {
	withoutTags := ebayHTMLTagPattern.ReplaceAllString(raw, " ")
	unescaped := html.UnescapeString(withoutTags)
	return strings.Join(strings.Fields(unescaped), " ")
}

func mapEbayItem(item ebayItemSummary) domain.Product {
	auctionType := domain.AuctionTypeSale
	var endingTime *time.Time
	for _, opt := range item.BuyingOptions {
		if opt == "AUCTION" {
			auctionType = domain.AuctionTypeAuction
			if t, err := time.Parse(time.RFC3339, item.ItemEndDate); err == nil {
				endingTime = &t
			}
			break
		}
	}

	// Pure auctions return price:null and carry the live figure in
	// currentBidPrice instead; fall back to price for the rare case of an
	// auction that also has one (e.g. a Buy-It-Now option) with no bids yet.
	priceInfo := item.Price
	if auctionType == domain.AuctionTypeAuction && item.CurrentBidPrice != nil {
		priceInfo = item.CurrentBidPrice
	}
	var price float64
	var currency string
	if priceInfo != nil {
		price, _ = strconv.ParseFloat(priceInfo.Value, 64)
		currency = priceInfo.Currency
	}

	var shippingCost *float64
	if len(item.ShippingOptions) > 0 && item.ShippingOptions[0].ShippingCost != nil {
		if v, err := strconv.ParseFloat(item.ShippingOptions[0].ShippingCost.Value, 64); err == nil {
			shippingCost = &v
		}
	}

	location := item.ItemLocation.City
	if location == "" {
		location = item.ItemLocation.Country
	}

	return domain.Product{
		ShopSource:   "ebay.com",
		Title:        item.Title,
		Price:        price,
		Currency:     currency,
		AuctionType:  auctionType,
		EndingTime:   endingTime,
		Condition:    mapEbayCondition(item.Condition),
		URL:          item.ItemWebURL,
		ImageURL:     item.Image.ImageURL,
		Location:     location,
		SellerName:   item.Seller.Username,
		ShippingCost: shippingCost,
		BidCount:     item.BidCount,
	}
}

// mapEbayCondition maps eBay Browse API's human-readable condition strings
// onto this app's existing domain.Condition enum (no distinct "refurbished"
// value exists, so refurbished conditions map to the closest existing one).
func mapEbayCondition(condition string) domain.Condition {
	c := strings.ToLower(condition)
	switch {
	case c == "new" || c == "new with tags":
		return domain.ConditionNew
	case strings.Contains(c, "refurbished"):
		return domain.ConditionLikeNew
	case c == "used" || c == "pre-owned":
		return domain.ConditionUsed
	case c == "for parts or not working":
		return domain.ConditionDamaged
	default:
		return domain.ConditionUnknown
	}
}
