package output

import (
	"secondHand/src/backend/internal/domain"
	"strings"
	"testing"
	"time"
)

func TestCLIFormatterFormatProducts(t *testing.T) {
	formatter := NewCLIFormatter()

	products := []domain.Product{
		{
			ID:          1,
			ShopSource:  "test.cz",
			Title:       "Test Product",
			Description: "Test Description",
			Price:       100.0,
			Currency:    "CZK",
			AuctionType: domain.AuctionTypeSale,
			Condition:   domain.ConditionUsed,
			URL:         "https://test.cz/1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	t.Run("Non-verbose output", func(t *testing.T) {
		result, err := formatter.FormatProducts(products, false)
		if err != nil {
			t.Fatalf("FormatProducts() error = %v", err)
		}

		if !strings.Contains(result, "Test Product") {
			t.Error("Output should contain product title")
		}

		if !strings.Contains(result, "100.00 CZK") {
			t.Error("Output should contain price")
		}

		if !strings.Contains(result, "test.cz") {
			t.Error("Output should contain shop source")
		}
	})

	t.Run("Verbose output", func(t *testing.T) {
		result, err := formatter.FormatProducts(products, true)
		if err != nil {
			t.Fatalf("FormatProducts() error = %v", err)
		}

		if !strings.Contains(result, "Test Description") {
			t.Error("Verbose output should contain description")
		}

		if !strings.Contains(result, "used") {
			t.Error("Verbose output should contain condition")
		}
	})

	t.Run("Empty products", func(t *testing.T) {
		result, err := formatter.FormatProducts([]domain.Product{}, false)
		if err != nil {
			t.Fatalf("FormatProducts() error = %v", err)
		}

		if !strings.Contains(result, "No products found") {
			t.Error("Output should indicate no products found")
		}
	})
}

func TestCLIFormatterFormatDiff(t *testing.T) {
	formatter := NewCLIFormatter()

	oldPrice := 100.0
	newPrice := 80.0

	diffs := []domain.ProductDiff{
		{
			Product: domain.Product{
				Title:       "New Product",
				Price:       100.0,
				Currency:    "CZK",
				ShopSource:  "test.cz",
				URL:         "https://test.cz/1",
				AuctionType: domain.AuctionTypeSale,
			},
			DiffType: domain.DiffTypeNew,
			NewPrice: &newPrice,
		},
		{
			Product: domain.Product{
				Title:       "Price Drop Product",
				Price:       80.0,
				Currency:    "CZK",
				ShopSource:  "test.cz",
				URL:         "https://test.cz/2",
				AuctionType: domain.AuctionTypeSale,
			},
			DiffType:     domain.DiffTypePriceDown,
			OldPrice:     &oldPrice,
			NewPrice:     &newPrice,
			PriceChanged: true,
		},
	}

	t.Run("Format diff", func(t *testing.T) {
		result, err := formatter.FormatDiff(diffs, false)
		if err != nil {
			t.Fatalf("FormatDiff() error = %v", err)
		}

		if !strings.Contains(result, "NEW PRODUCTS") {
			t.Error("Output should contain NEW PRODUCTS section")
		}

		if !strings.Contains(result, "PRICE DROPS") {
			t.Error("Output should contain PRICE DROPS section")
		}

		if !strings.Contains(result, "New Product") {
			t.Error("Output should contain new product title")
		}

		if !strings.Contains(result, "Price Drop Product") {
			t.Error("Output should contain price drop product title")
		}
	})

	t.Run("Empty diffs", func(t *testing.T) {
		result, err := formatter.FormatDiff([]domain.ProductDiff{}, false)
		if err != nil {
			t.Fatalf("FormatDiff() error = %v", err)
		}

		if !strings.Contains(result, "No changes found") {
			t.Error("Output should indicate no changes found")
		}
	})
}
