package main

import "testing"

func TestGoodOfferPointers(t *testing.T) {
	tests := []struct {
		name                     string
		maxPrice, avgDiscountPct float64
		wantMaxPriceNil          bool
		wantAvgDiscountNil       bool
	}{
		{"neither set", 0, 0, true, true},
		{"only max price set", 100, 0, false, true},
		{"only avg discount set", 0, 15.5, true, false},
		{"both set", 100, 15.5, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxPricePtr, avgDiscountPtr := goodOfferPointers(tt.maxPrice, tt.avgDiscountPct)

			if (maxPricePtr == nil) != tt.wantMaxPriceNil {
				t.Fatalf("maxPricePtr = %v, want nil=%v", maxPricePtr, tt.wantMaxPriceNil)
			}
			if maxPricePtr != nil && *maxPricePtr != tt.maxPrice {
				t.Fatalf("*maxPricePtr = %v, want %v", *maxPricePtr, tt.maxPrice)
			}

			if (avgDiscountPtr == nil) != tt.wantAvgDiscountNil {
				t.Fatalf("avgDiscountPtr = %v, want nil=%v", avgDiscountPtr, tt.wantAvgDiscountNil)
			}
			if avgDiscountPtr != nil && *avgDiscountPtr != tt.avgDiscountPct {
				t.Fatalf("*avgDiscountPtr = %v, want %v", *avgDiscountPtr, tt.avgDiscountPct)
			}
		})
	}
}
