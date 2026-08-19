package service

import (
	"regexp"
	"strconv"

	"secondHand/internal/domain"
)

// minPriorPricesForAverage is the minimum number of previously stored
// products required before the trailing-average discount check is
// considered meaningful; below this, the check is skipped rather than
// computed against noisy signal.
const minPriorPricesForAverage = 3

// TotalCost is a product's price plus its shipping cost when known - the
// real cost of acquiring it, which is what good-offer thresholds should be
// judged against rather than the item price alone. Falls back to just the
// price for shops (or unresolved eBay listings) with no shipping cost.
func TotalCost(product domain.Product) float64 {
	if product.ShippingCost != nil {
		return product.Price + *product.ShippingCost
	}
	return product.Price
}

// lotSizePatterns matches common ways sellers phrase a multi-item bundle in
// a title ("Lot of 6", "5 set", "Sada 10", "10-pack", "Bundle of 3"). Best-
// effort text heuristic, not a structured eBay field - titles are free
// text and phrasing is inconsistent, so this will occasionally miss a real
// lot (no false "good offer" results from that, just a missed per-unit
// discount) or, less often, misread an unrelated number in the title.
var lotSizePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\blot\s+of\s+(\d+)\b`),
	regexp.MustCompile(`(?i)\blot\s*[:\-]?\s*(\d+)\b`),
	regexp.MustCompile(`(?i)\bset\s+of\s+(\d+)\b`),
	regexp.MustCompile(`(?i)\bbundle\s+of\s+(\d+)\b`),
	regexp.MustCompile(`(?i)\b(\d+)[- ]pack\b`),
	regexp.MustCompile(`(?i)\b(\d+)\s*sets?\b`),
	regexp.MustCompile(`(?i)\bsada\s+(\d+)\b`),
}

// detectLotSize returns how many physical items a listing's title implies
// are bundled together, or 1 if no such pattern is found.
func detectLotSize(title string) int {
	for _, re := range lotSizePatterns {
		if m := re.FindStringSubmatch(title); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
				return n
			}
		}
	}
	return 1
}

// PerUnitCost is TotalCost divided by the title's detected lot size - a
// "Lot of 6" priced at 300 costs 50/unit, which is what a good-offer
// ceiling should actually be judged against, not the full lot price.
func PerUnitCost(product domain.Product) float64 {
	return TotalCost(product) / float64(detectLotSize(product.Title))
}

// EvaluateGoodOffer decides whether product is a "good offer" for search,
// per the two independently optional thresholds a saved search can
// configure: a flat price ceiling, or a discount against the trailing
// average of previously stored prices for that search. Either being met is
// sufficient; if neither is configured, this always returns false.
// priorPrices and the ceiling/average comparisons are all in terms of
// PerUnitCost (price + shipping, divided by detected lot size), not the
// bare listing price.
func EvaluateGoodOffer(search domain.Search, product domain.Product, priorPrices []float64) bool {
	cost := PerUnitCost(product)

	if search.MaxPrice != nil && cost <= *search.MaxPrice {
		return true
	}

	if search.AvgDiscountPct != nil && len(priorPrices) >= minPriorPricesForAverage {
		var sum float64
		for _, p := range priorPrices {
			sum += p
		}
		average := sum / float64(len(priorPrices))
		threshold := average * (1 - *search.AvgDiscountPct/100)
		if cost <= threshold {
			return true
		}
	}

	return false
}
