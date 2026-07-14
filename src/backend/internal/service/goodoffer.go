package service

import "secondHand/src/backend/internal/domain"

// minPriorPricesForAverage is the minimum number of previously stored
// products required before the trailing-average discount check is
// considered meaningful; below this, the check is skipped rather than
// computed against noisy signal.
const minPriorPricesForAverage = 3

// EvaluateGoodOffer decides whether product is a "good offer" for search,
// per the two independently optional thresholds a saved search can
// configure: a flat price ceiling, or a discount against the trailing
// average of previously stored prices for that search. Either being met is
// sufficient; if neither is configured, this always returns false.
func EvaluateGoodOffer(search domain.Search, product domain.Product, priorPrices []float64) bool {
	if search.MaxPrice != nil && product.Price <= *search.MaxPrice {
		return true
	}

	if search.AvgDiscountPct != nil && len(priorPrices) >= minPriorPricesForAverage {
		var sum float64
		for _, p := range priorPrices {
			sum += p
		}
		average := sum / float64(len(priorPrices))
		threshold := average * (1 - *search.AvgDiscountPct/100)
		if product.Price <= threshold {
			return true
		}
	}

	return false
}
