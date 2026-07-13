package domain

import (
	"testing"
	"time"
)

func TestConditions(t *testing.T) {
	tests := []struct {
		name     string
		input    Condition
		expected Condition
	}{
		{"New condition", ConditionNew, ConditionNew},
		{"Used condition", ConditionUsed, ConditionUsed},
		{"Like new condition", ConditionLikeNew, ConditionLikeNew},
		{"Unknown condition", ConditionUnknown, ConditionUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}

func TestAuctionTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    AuctionType
		expected AuctionType
	}{
		{"Sale type", AuctionTypeSale, AuctionTypeSale},
		{"Auction type", AuctionTypeAuction, AuctionTypeAuction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}

func TestProductCreation(t *testing.T) {
	now := time.Now()
	product := Product{
		ID:          1,
		ShopSource:  "test.cz",
		Title:       "Test Product",
		Description: "Test Description",
		Price:       100.0,
		Currency:    "CZK",
		AuctionType: AuctionTypeSale,
		Condition:   ConditionUsed,
		URL:         "https://test.cz/product/1",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if product.Title != "Test Product" {
		t.Errorf("Expected title 'Test Product', got '%s'", product.Title)
	}

	if product.Price != 100.0 {
		t.Errorf("Expected price 100.0, got %f", product.Price)
	}

	if product.ShopSource != "test.cz" {
		t.Errorf("Expected shop 'test.cz', got '%s'", product.ShopSource)
	}
}

func TestSearchCreation(t *testing.T) {
	now := time.Now()
	search := Search{
		ID:        1,
		Keyword:   "laptop",
		CreatedAt: now,
	}

	if search.Keyword != "laptop" {
		t.Errorf("Expected keyword 'laptop', got '%s'", search.Keyword)
	}

	if search.LastCheckedAt != nil {
		t.Error("Expected LastCheckedAt to be nil")
	}
}

func TestDiffTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    DiffType
		expected DiffType
	}{
		{"New diff", DiffTypeNew, DiffTypeNew},
		{"Removed diff", DiffTypeRemoved, DiffTypeRemoved},
		{"Price up diff", DiffTypePriceUp, DiffTypePriceUp},
		{"Price down diff", DiffTypePriceDown, DiffTypePriceDown},
		{"Unchanged diff", DiffTypeUnchanged, DiffTypeUnchanged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}
