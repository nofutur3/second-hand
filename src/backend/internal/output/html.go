package output

import (
	"fmt"
	"html/template"
	"secondHand/src/backend/internal/domain"
	"strings"
)

// HTMLFormatter formats output as HTML
type HTMLFormatter struct{}

// NewHTMLFormatter creates a new HTML formatter
func NewHTMLFormatter() *HTMLFormatter {
	return &HTMLFormatter{}
}

const productsTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Second Hand Products</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        h1 { color: #333; }
        .product { background: white; padding: 15px; margin: 10px 0; border-radius: 5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .product h3 { margin: 0 0 10px 0; color: #2c3e50; }
        .price { font-size: 1.2em; color: #27ae60; font-weight: bold; }
        .shop { color: #7f8c8d; font-size: 0.9em; }
        .location { color: #95a5a6; font-size: 0.9em; }
        .description { margin-top: 10px; color: #555; }
        .url { margin-top: 10px; }
        .url a { color: #3498db; text-decoration: none; }
        .url a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>Found {{ .Count }} Products</h1>
    {{ range .Products }}
    <div class="product">
        <h3>{{ .Title }}</h3>
        <div class="price">{{ printf "%.2f" .Price }} {{ .Currency }}</div>
        <div class="shop">Shop: {{ .ShopSource }}</div>
        {{ if .Location }}<div class="location">Location: {{ .Location }}</div>{{ end }}
        {{ if .Description }}<div class="description">{{ .Description }}</div>{{ end }}
        <div class="url"><a href="{{ .URL }}" target="_blank">View Product</a></div>
    </div>
    {{ end }}
</body>
</html>
`

const diffTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Product Changes</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        h1, h2 { color: #333; }
        .section { margin: 20px 0; }
        .product { background: white; padding: 15px; margin: 10px 0; border-radius: 5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .product h3 { margin: 0 0 10px 0; color: #2c3e50; }
        .new { border-left: 4px solid #27ae60; }
        .price-down { border-left: 4px solid #2ecc71; }
        .price-up { border-left: 4px solid #e74c3c; }
        .removed { border-left: 4px solid #95a5a6; opacity: 0.7; }
        .price { font-size: 1.2em; font-weight: bold; }
        .price-change { color: #7f8c8d; font-size: 0.9em; }
        .shop { color: #7f8c8d; font-size: 0.9em; }
        .url { margin-top: 10px; }
        .url a { color: #3498db; text-decoration: none; }
        .url a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>Product Changes ({{ .TotalCount }})</h1>
    
    {{ if .NewProducts }}
    <div class="section">
        <h2>📦 New Products ({{ len .NewProducts }})</h2>
        {{ range .NewProducts }}
        <div class="product new">
            <h3>{{ .Product.Title }}</h3>
            <div class="price" style="color: #27ae60;">{{ printf "%.2f" .Product.Price }} {{ .Product.Currency }}</div>
            <div class="shop">Shop: {{ .Product.ShopSource }}</div>
            {{ if .Product.Location }}<div class="shop">Location: {{ .Product.Location }}</div>{{ end }}
            <div class="url"><a href="{{ .Product.URL }}" target="_blank">View Product</a></div>
        </div>
        {{ end }}
    </div>
    {{ end }}

    {{ if .PriceDownProducts }}
    <div class="section">
        <h2>📉 Price Drops ({{ len .PriceDownProducts }})</h2>
        {{ range .PriceDownProducts }}
        <div class="product price-down">
            <h3>{{ .Product.Title }}</h3>
            <div class="price" style="color: #2ecc71;">{{ printf "%.2f" .NewPrice }} {{ .Product.Currency }}</div>
            <div class="price-change">Was: {{ printf "%.2f" .OldPrice }} {{ .Product.Currency }} (saved {{ printf "%.2f" (sub .OldPrice .NewPrice) }} {{ .Product.Currency }})</div>
            <div class="shop">Shop: {{ .Product.ShopSource }}</div>
            <div class="url"><a href="{{ .Product.URL }}" target="_blank">View Product</a></div>
        </div>
        {{ end }}
    </div>
    {{ end }}

    {{ if .PriceUpProducts }}
    <div class="section">
        <h2>📈 Price Increases ({{ len .PriceUpProducts }})</h2>
        {{ range .PriceUpProducts }}
        <div class="product price-up">
            <h3>{{ .Product.Title }}</h3>
            <div class="price" style="color: #e74c3c;">{{ printf "%.2f" .NewPrice }} {{ .Product.Currency }}</div>
            <div class="price-change">Was: {{ printf "%.2f" .OldPrice }} {{ .Product.Currency }} (+{{ printf "%.2f" (sub .NewPrice .OldPrice) }} {{ .Product.Currency }})</div>
            <div class="shop">Shop: {{ .Product.ShopSource }}</div>
            <div class="url"><a href="{{ .Product.URL }}" target="_blank">View Product</a></div>
        </div>
        {{ end }}
    </div>
    {{ end }}

    {{ if .RemovedProducts }}
    <div class="section">
        <h2>❌ Removed Products ({{ len .RemovedProducts }})</h2>
        {{ range .RemovedProducts }}
        <div class="product removed">
            <h3>{{ .Product.Title }}</h3>
            <div class="price" style="color: #95a5a6;">Last Price: {{ printf "%.2f" .Product.Price }} {{ .Product.Currency }}</div>
            <div class="shop">Shop: {{ .Product.ShopSource }}</div>
        </div>
        {{ end }}
    </div>
    {{ end }}
</body>
</html>
`

// FormatProducts formats products as HTML
func (f *HTMLFormatter) FormatProducts(products []domain.Product, verbose bool) (string, error) {
	tmpl, err := template.New("products").Parse(productsTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := map[string]interface{}{
		"Count":    len(products),
		"Products": products,
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}

// FormatDiff formats product diffs as HTML
func (f *HTMLFormatter) FormatDiff(diffs []domain.ProductDiff, verbose bool) (string, error) {
	funcMap := template.FuncMap{
		"sub": func(a, b float64) float64 { return a - b },
	}

	tmpl, err := template.New("diff").Funcs(funcMap).Parse(diffTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Group diffs by type
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

	data := map[string]interface{}{
		"TotalCount":        len(diffs),
		"NewProducts":       newProducts,
		"RemovedProducts":   removedProducts,
		"PriceUpProducts":   priceUpProducts,
		"PriceDownProducts": priceDownProducts,
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}
