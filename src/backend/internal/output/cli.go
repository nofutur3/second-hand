package output

import (
	"fmt"
	"secondHand/src/backend/internal/domain"
	"strings"
)

// CLIFormatter formats output for command line
type CLIFormatter struct{}

// NewCLIFormatter creates a new CLI formatter
func NewCLIFormatter() *CLIFormatter {
	return &CLIFormatter{}
}

// FormatProducts formats products for CLI output
func (f *CLIFormatter) FormatProducts(products []domain.Product, verbose bool) (string, error) {
	if len(products) == 0 {
		return "No products found.\n", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n=== Found %d products ===\n\n", len(products)))

	for i, p := range products {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, p.Title))
		sb.WriteString(fmt.Sprintf("   Shop: %s\n", p.ShopSource))
		sb.WriteString(fmt.Sprintf("   Price: %.2f %s\n", p.Price, p.Currency))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", p.URL))

		if verbose {
			if p.Location != "" {
				sb.WriteString(fmt.Sprintf("   Location: %s\n", p.Location))
			}
			if p.Description != "" {
				desc := p.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("   Description: %s\n", desc))
			}
			if p.Condition != domain.ConditionUnknown {
				sb.WriteString(fmt.Sprintf("   Condition: %s\n", p.Condition))
			}
			if p.SellerName != "" {
				sb.WriteString(fmt.Sprintf("   Seller: %s\n", p.SellerName))
			}
			sb.WriteString(fmt.Sprintf("   Type: %s\n", p.AuctionType))
			sb.WriteString(fmt.Sprintf("   Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04:05")))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// FormatDiff formats product diffs for CLI output
func (f *CLIFormatter) FormatDiff(diffs []domain.ProductDiff, verbose bool) (string, error) {
	if len(diffs) == 0 {
		return "No changes found.\n", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n=== Found %d changes ===\n\n", len(diffs)))

	// Group by diff type
	newProducts := []domain.ProductDiff{}
	removedProducts := []domain.ProductDiff{}
	priceUpProducts := []domain.ProductDiff{}
	priceDownProducts := []domain.ProductDiff{}

	for _, diff := range diffs {
		switch diff.DiffType {
		case domain.DiffTypeNew:
			newProducts = append(newProducts, diff)
		case domain.DiffTypeRemoved:
			removedProducts = append(removedProducts, diff)
		case domain.DiffTypePriceUp:
			priceUpProducts = append(priceUpProducts, diff)
		case domain.DiffTypePriceDown:
			priceDownProducts = append(priceDownProducts, diff)
		}
	}

	// Format new products
	if len(newProducts) > 0 {
		sb.WriteString(fmt.Sprintf("📦 NEW PRODUCTS (%d):\n", len(newProducts)))
		for i, diff := range newProducts {
			p := diff.Product
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, p.Title))
			sb.WriteString(fmt.Sprintf("   Price: %.2f %s | Shop: %s\n", p.Price, p.Currency, p.ShopSource))
			sb.WriteString(fmt.Sprintf("   URL: %s\n", p.URL))
			if verbose && p.Location != "" {
				sb.WriteString(fmt.Sprintf("   Location: %s\n", p.Location))
			}
			sb.WriteString("\n")
		}
	}

	// Format price drops
	if len(priceDownProducts) > 0 {
		sb.WriteString(fmt.Sprintf("📉 PRICE DROPS (%d):\n", len(priceDownProducts)))
		for i, diff := range priceDownProducts {
			p := diff.Product
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, p.Title))
			sb.WriteString(fmt.Sprintf("   Price: %.2f %s → %.2f %s (%.2f %s)\n",
				*diff.OldPrice, p.Currency, *diff.NewPrice, p.Currency,
				*diff.OldPrice-*diff.NewPrice, p.Currency))
			sb.WriteString(fmt.Sprintf("   Shop: %s | URL: %s\n", p.ShopSource, p.URL))
			sb.WriteString("\n")
		}
	}

	// Format price increases
	if len(priceUpProducts) > 0 {
		sb.WriteString(fmt.Sprintf("📈 PRICE INCREASES (%d):\n", len(priceUpProducts)))
		for i, diff := range priceUpProducts {
			p := diff.Product
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, p.Title))
			sb.WriteString(fmt.Sprintf("   Price: %.2f %s → %.2f %s (+%.2f %s)\n",
				*diff.OldPrice, p.Currency, *diff.NewPrice, p.Currency,
				*diff.NewPrice-*diff.OldPrice, p.Currency))
			sb.WriteString(fmt.Sprintf("   Shop: %s | URL: %s\n", p.ShopSource, p.URL))
			sb.WriteString("\n")
		}
	}

	// Format removed products
	if len(removedProducts) > 0 {
		sb.WriteString(fmt.Sprintf("❌ REMOVED PRODUCTS (%d):\n", len(removedProducts)))
		for i, diff := range removedProducts {
			p := diff.Product
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, p.Title))
			sb.WriteString(fmt.Sprintf("   Last Price: %.2f %s | Shop: %s\n", p.Price, p.Currency, p.ShopSource))
			if verbose {
				sb.WriteString(fmt.Sprintf("   URL: %s\n", p.URL))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
