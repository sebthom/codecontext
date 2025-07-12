package cli

import (
	"testing"
)

func TestGetReductionFactor(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected float64
	}{
		{
			name:     "minimal level",
			level:    "minimal",
			expected: 0.3,
		},
		{
			name:     "balanced level",
			level:    "balanced",
			expected: 0.6,
		},
		{
			name:     "aggressive level",
			level:    "aggressive",
			expected: 0.15,
		},
		{
			name:     "unknown level defaults to balanced",
			level:    "unknown",
			expected: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getReductionFactor(tt.level)
			if result != tt.expected {
				t.Errorf("getReductionFactor(%s) = %f, expected %f", tt.level, result, tt.expected)
			}
		})
	}
}

func TestGetQualityScore(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected float64
	}{
		{
			name:     "minimal level",
			level:    "minimal",
			expected: 0.95,
		},
		{
			name:     "balanced level",
			level:    "balanced",
			expected: 0.85,
		},
		{
			name:     "aggressive level",
			level:    "aggressive",
			expected: 0.70,
		},
		{
			name:     "unknown level defaults to balanced",
			level:    "unknown",
			expected: 0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getQualityScore(tt.level)
			if result != tt.expected {
				t.Errorf("getQualityScore(%s) = %f, expected %f", tt.level, result, tt.expected)
			}
		})
	}
}

func TestCompactionCalculations(t *testing.T) {
	tests := []struct {
		name              string
		originalTokens    int
		level             string
		expectedTokens    int
		expectedReduction float64
	}{
		{
			name:              "minimal compaction",
			originalTokens:    150000,
			level:             "minimal",
			expectedTokens:    45000,
			expectedReduction: 70.0,
		},
		{
			name:              "balanced compaction",
			originalTokens:    150000,
			level:             "balanced",
			expectedTokens:    90000,
			expectedReduction: 40.0,
		},
		{
			name:              "aggressive compaction",
			originalTokens:    150000,
			level:             "aggressive",
			expectedTokens:    22500,
			expectedReduction: 85.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compactedTokens := int(float64(tt.originalTokens) * getReductionFactor(tt.level))
			reductionPercent := float64(tt.originalTokens-compactedTokens) / float64(tt.originalTokens) * 100

			if compactedTokens != tt.expectedTokens {
				t.Errorf("Compacted tokens = %d, expected %d", compactedTokens, tt.expectedTokens)
			}

			if reductionPercent != tt.expectedReduction {
				t.Errorf("Reduction percent = %f, expected %f", reductionPercent, tt.expectedReduction)
			}
		})
	}
}
