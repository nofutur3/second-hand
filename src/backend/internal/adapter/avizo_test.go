package adapter

import (
	"testing"

	"secondHand/src/backend/internal/domain"
)

func TestTitleMatchesKeyword(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		keyword  string
		expected bool
	}{
		{"Exact phrase", "Lego Mindstorms EV3", "lego mindstorms", true},
		{"Only one word present", "Lego 8547 Mindstorms NXT 2.0", "lego mindstorms", true},
		{"Unrelated promoted listing", "Dámské červené sako zn. OILILY", "lego mindstorms", false},
		{"Unrelated job posting", "Referentka odboru stavební úřad", "lego mindstorms", false},
		{"Case insensitive", "LEGO MINDSTORMS", "lego mindstorms", true},
		{"Short keyword falls back to whole phrase", "Ford Ka 1.3", "ka", true},
		{"Short keyword, no match", "Lego Mindstorms", "xy", false},
		{"Numeric word alone doesn't match unrelated tire listing", "Sada letnich pneu 255/60 R18 Nexen", "forerunner 255", false},
		{"Numeric word still matches when the distinctive word is present", "Garmin Forerunner 255 hodinky", "forerunner 255", true},
		{"Bare numeric keyword falls back to literal phrase", "Pneu 255/60 R18", "255", true},
		{"Bare numeric keyword, no literal match", "Garmin Forerunner 255", "256", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := titleMatchesKeyword(tt.title, tt.keyword)
			if result != tt.expected {
				t.Errorf("titleMatchesKeyword(%q, %q) = %v, want %v", tt.title, tt.keyword, result, tt.expected)
			}
		})
	}
}

func TestAvizoConditionFromSchema(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		expected  domain.Condition
	}{
		{"New", "https://schema.org/NewCondition", domain.ConditionNew},
		{"Used", "https://schema.org/UsedCondition", domain.ConditionUsed},
		{"Refurbished maps to like-new", "https://schema.org/RefurbishedCondition", domain.ConditionLikeNew},
		{"Damaged", "https://schema.org/DamagedCondition", domain.ConditionDamaged},
		{"Unknown/empty", "", domain.ConditionUnknown},
		{"Unrecognized value", "https://schema.org/SomethingElse", domain.ConditionUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := avizoConditionFromSchema(tt.condition)
			if result != tt.expected {
				t.Errorf("avizoConditionFromSchema(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}
