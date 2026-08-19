package service

import (
	"testing"

	"secondHand/internal/domain"
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
		{
			name:    "ceiling: shipping pushes total cost over the ceiling",
			search:  domain.Search{MaxPrice: floatPtr(50)},
			product: domain.Product{Price: 45, ShippingCost: floatPtr(10)},
			want:    false, // total = 55, over the 50 ceiling even though item price alone isn't
		},
		{
			name:    "ceiling: met once shipping is included in total cost",
			search:  domain.Search{MaxPrice: floatPtr(50)},
			product: domain.Product{Price: 30, ShippingCost: floatPtr(10)},
			want:    true, // total = 40
		},
		{
			name:        "discount: total cost (incl. shipping) compared against ceiling",
			search:      domain.Search{AvgDiscountPct: floatPtr(20)},
			product:     domain.Product{Price: 75, ShippingCost: floatPtr(10)},
			priorPrices: []float64{100, 100, 100},
			want:        false, // total = 85, threshold = 80
		},
		{
			name:    "lot: ceiling judged per-unit, not against the full lot price",
			search:  domain.Search{MaxPrice: floatPtr(50)},
			product: domain.Product{Title: "Gameboy Advance GBC Motherboard Lot of 6 For Parts Not Working", Price: 300},
			want:    true, // 300 / 6 = 50/unit
		},
		{
			name:    "lot: full lot price alone would wrongly fail the ceiling",
			search:  domain.Search{MaxPrice: floatPtr(50)},
			product: domain.Product{Title: "Lot of 6 Gameboy Color consoles for parts", Price: 360},
			want:    false, // 360 / 6 = 60/unit, over ceiling
		},
		{
			name:    "no detected lot defaults to a single unit",
			search:  domain.Search{MaxPrice: floatPtr(50)},
			product: domain.Product{Title: "Nintendo Game Boy Color Clear for parts", Price: 55},
			want:    false,
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

func TestDetectLotSize(t *testing.T) {
	tests := []struct {
		title string
		want  int
	}{
		{"Lot 2 Nintendo Gameboy Pocket GBP console Random color Japanese Junk for parts", 2},
		{"Gameboy Advance GBC Motherboard Lot of 6 For Parts Not Working", 6},
		{"Nintendo GameBoy Color Console GBC Random 5 set for parts repair", 5},
		{"Sada 10 GameBoy Color GBC sada 10 konzolí náhodných barev pro díly", 10},
		{"10-pack Game Boy Color replacement screens", 10},
		{"Bundle of 3 Game Boy Color shells", 3},
		{"PARTS: Nintendo Game Boy Color Clear (Shell, Buttons, Silicon, OEM Screen)", 1},
		{"Nintendo Game Boy Color CGB-001 Atomic Purple console tested partial fault", 1},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			if got := detectLotSize(tt.title); got != tt.want {
				t.Errorf("detectLotSize(%q) = %d, want %d", tt.title, got, tt.want)
			}
		})
	}
}
