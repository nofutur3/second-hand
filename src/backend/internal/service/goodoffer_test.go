package service

import (
	"testing"

	"secondHand/src/backend/internal/domain"
)

func floatPtr(f float64) *float64 { return &f }

func TestEvaluateGoodOffer(t *testing.T) {
	tests := []struct {
		name        string
		search      domain.Search
		product     domain.Product
		priorPrices []float64
		want        bool
	}{
		{
			name:    "ceiling met",
			search:  domain.Search{MaxPrice: floatPtr(50)},
			product: domain.Product{Price: 45},
			want:    true,
		},
		{
			name:    "ceiling not met, nothing else configured",
			search:  domain.Search{MaxPrice: floatPtr(50)},
			product: domain.Product{Price: 55},
			want:    false,
		},
		{
			name:        "discount met with >=3 priors",
			search:      domain.Search{AvgDiscountPct: floatPtr(20)},
			product:     domain.Product{Price: 79},
			priorPrices: []float64{100, 100, 100},
			want:        true, // threshold = 100 * 0.8 = 80, 79 <= 80
		},
		{
			name:        "discount not met with >=3 priors",
			search:      domain.Search{AvgDiscountPct: floatPtr(20)},
			product:     domain.Product{Price: 85},
			priorPrices: []float64{100, 100, 100},
			want:        false, // threshold = 80, 85 > 80
		},
		{
			name:        "discount configured but fewer than 3 priors is skipped",
			search:      domain.Search{AvgDiscountPct: floatPtr(20)},
			product:     domain.Product{Price: 1},
			priorPrices: []float64{100, 100},
			want:        false,
		},
		{
			name:    "neither configured never triggers",
			search:  domain.Search{},
			product: domain.Product{Price: 1},
			want:    false,
		},
		{
			name: "both configured, discount sufficient even though ceiling isn't met",
			search: domain.Search{
				MaxPrice:       floatPtr(50),
				AvgDiscountPct: floatPtr(20),
			},
			product:     domain.Product{Price: 79},
			priorPrices: []float64{100, 100, 100},
			want:        true,
		},
		{
			name: "both configured, neither met",
			search: domain.Search{
				MaxPrice:       floatPtr(50),
				AvgDiscountPct: floatPtr(20),
			},
			product:     domain.Product{Price: 90},
			priorPrices: []float64{100, 100, 100},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateGoodOffer(tt.search, tt.product, tt.priorPrices)
			if got != tt.want {
				t.Errorf("EvaluateGoodOffer() = %v, want %v", got, tt.want)
			}
		})
	}
}
