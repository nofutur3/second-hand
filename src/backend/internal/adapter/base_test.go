package adapter

import (
	"secondHand/src/backend/internal/domain"
	"testing"
)

func TestParsePrice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"Simple price", "100", 100.0},
		{"Price with CZK", "100 Kč", 100.0},
		{"Price with spaces", "1 000 Kč", 1000.0},
		{"Price with comma decimal", "1 500,50", 1500.50},
		{"Price with dot decimal", "1500.50", 1500.50},
		{"Price no decimal", "1500", 1500.0},
		{"Empty string", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePrice(tt.input)
			if result != tt.expected {
				t.Errorf("parsePrice(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectCondition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected domain.Condition
	}{
		{"New in Czech", "Novy produkt", domain.ConditionNew},
		{"Used in Czech", "Pouzity notebook", domain.ConditionUsed},
		{"Like new", "Jako novy telefon", domain.ConditionLikeNew},
		{"Damaged", "Poskozeny display", domain.ConditionDamaged},
		{"Unknown", "Some random text", domain.ConditionUnknown},
		{"Empty", "", domain.ConditionUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCondition(tt.input)
			if result != tt.expected {
				t.Errorf("detectCondition(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"HTTPS URL", "https://www.bazos.cz", "www.bazos.cz"},
		{"HTTP URL", "http://www.bazos.cz", "www.bazos.cz"},
		{"URL with path", "https://www.bazos.cz/search", "www.bazos.cz"},
		{"Simple domain", "www.bazos.cz", "www.bazos.cz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDomain(tt.input)
			if result != tt.expected {
				t.Errorf("extractDomain(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanPriceString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Digits only", "12345", "12345"},
		{"With currency", "100 Kč", "100"},
		{"With comma decimal", "1,500.50", "1500.50"}, // .50 is detected as decimal
		{"With spaces", "1 000 Kč", "1000"},
		{"Mixed", "Price: 1,500 CZK", "1500"}, // No decimal part, comma ignored
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanPriceString(tt.input)
			if result != tt.expected {
				t.Errorf("cleanPriceString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"All uppercase", "HELLO", "hello"},
		{"Mixed case", "Hello World", "hello world"},
		{"All lowercase", "hello", "hello"},
		{"With numbers", "Test123", "test123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLower(tt.input)
			if result != tt.expected {
				t.Errorf("toLower(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"Contains substring", "hello world", "world", true},
		{"Does not contain", "hello world", "foo", false},
		{"Empty substring", "hello", "", true},
		{"Empty string", "", "hello", false},
		{"Same strings", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.str, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.str, tt.substr, result, tt.expected)
			}
		})
	}
}
