package adapter

import (
	"fmt"
	"secondHand/src/backend/internal/config"
	"secondHand/src/backend/internal/domain"
	"strings"
)

// Registry manages all shop adapters
type Registry struct {
	adapters map[string]domain.ShopAdapter
	cfg      *config.Config
}

// NewRegistry creates a new adapter registry
func NewRegistry(cfg *config.Config) *Registry {
	registry := &Registry{
		adapters: make(map[string]domain.ShopAdapter),
		cfg:      cfg,
	}

	// Initialize adapters based on config
	for _, shop := range cfg.Shops {
		if !shop.Enabled {
			continue
		}

		adapter := registry.createAdapter(shop.URL, cfg.Scraping.DelayMS, cfg.Scraping.RequestTimeout)
		if adapter != nil {
			registry.adapters[adapter.Name()] = adapter
		}
	}

	return registry
}

// createAdapter creates an adapter based on URL
func (r *Registry) createAdapter(url string, delayMS, timeoutSec int) domain.ShopAdapter {
	urlLower := strings.ToLower(url)

	// Check for mock adapters (for testing)
	if strings.Contains(urlLower, "mock-") {
		// Extract shop name from URL
		parts := strings.Split(urlLower, "mock-")
		if len(parts) > 1 {
			shopName := strings.Split(parts[1], ".")[0]
			return NewMockAdapter("mock-"+shopName, url, delayMS, timeoutSec)
		}
	}

	if strings.Contains(urlLower, "bazos.cz") {
		return NewBazosAdapter(url, delayMS, timeoutSec)
	} else if strings.Contains(urlLower, "sbazar.cz") {
		return NewSbazarAdapter(url, delayMS, timeoutSec)
	} else if strings.Contains(urlLower, "avizo.cz") {
		return NewAvizoAdapter(url, delayMS, timeoutSec)
	} else if strings.Contains(urlLower, "inzeruj.cz") {
		return NewInzerujAdapter(url, delayMS, timeoutSec)
	} else if strings.Contains(urlLower, "aukro.cz") {
		return NewAukroAdapter(url, delayMS, timeoutSec)
	} else if strings.Contains(urlLower, "ebay.com") {
		return NewEbayAdapter(url, r.cfg.Ebay, delayMS, timeoutSec)
	}

	fmt.Printf("Warning: No adapter found for URL: %s\n", url)
	return nil
}

// GetAdapter returns an adapter by name
func (r *Registry) GetAdapter(name string) (domain.ShopAdapter, error) {
	adapter, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("adapter not found: %s", name)
	}
	return adapter, nil
}

// GetAllAdapters returns all registered adapters
func (r *Registry) GetAllAdapters() []domain.ShopAdapter {
	adapters := make([]domain.ShopAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	return adapters
}

// GetAdapterNames returns all registered adapter names
func (r *Registry) GetAdapterNames() []string {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}
