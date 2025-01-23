package zapx

import (
	"testing"
)

func TestSlowlogEncoder_LongestCommonPrefOffset(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: 0,
		},
		{
			name:     "single string",
			input:    []string{"test"},
			expected: 4,
		},
		{
			name:     "no common prefix",
			input:    []string{"abc", "def", "ghi"},
			expected: 0,
		},
		{
			name:     "partial common prefix",
			input:    []string{"prefix123", "prefix456", "prefix789"},
			expected: 6,
		},
		{
			name:     "full common string",
			input:    []string{"same", "same", "same"},
			expected: 4,
		},
		{
			name:     "mixed length strings with common prefix",
			input:    []string{"test123", "test", "test4567"},
			expected: 4,
		},
		{
			name:     "strings with spaces",
			input:    []string{"common prefix 1", "common prefix 2", "common prefix 3"},
			expected: 14,
		},
	}

	sle := &SlowlogEncoder{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sle.longestCommonPrefOffset(tt.input)
			if result != tt.expected {
				t.Errorf("longestCommonPrefOffset() = %v, want %v", result, tt.expected)
			}
		})
	}
}
