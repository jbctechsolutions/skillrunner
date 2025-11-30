package context

import (
	"testing"
)

func TestTokenCounter_Count(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "simple sentence",
			text:     "Hello, world!",
			expected: 3, // 13 chars / 4 = 3
		},
		{
			name:     "longer text",
			text:     "This is a longer sentence with multiple words.",
			expected: 11, // 47 chars / 4 = 11
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tc.Count(tt.text)
			if got != tt.expected {
				t.Errorf("Count() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTokenCounter_CountWords(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "single word",
			text:     "Hello",
			expected: 1,
		},
		{
			name:     "multiple words",
			text:     "Hello world this is a test",
			expected: 6,
		},
		{
			name:     "words with punctuation",
			text:     "Hello, world! How are you?",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tc.CountWords(tt.text)
			if got != tt.expected {
				t.Errorf("CountWords() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTokenCounter_CountSentences(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "single sentence",
			text:     "Hello, world!",
			expected: 1,
		},
		{
			name:     "multiple sentences",
			text:     "Hello. How are you? I am fine!",
			expected: 3,
		},
		{
			name:     "no punctuation",
			text:     "Hello world",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tc.CountSentences(tt.text)
			if got != tt.expected {
				t.Errorf("CountSentences() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTokenCounter_SplitSentences(t *testing.T) {
	tc := NewTokenCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "single sentence",
			text:     "Hello, world!",
			expected: 1,
		},
		{
			name:     "multiple sentences",
			text:     "Hello. How are you? I am fine!",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tc.SplitSentences(tt.text)
			if len(got) != tt.expected {
				t.Errorf("SplitSentences() returned %v sentences, want %v", len(got), tt.expected)
			}
		})
	}
}

func TestTokenBudget(t *testing.T) {
	// Create a budget with 1000 total tokens, 200 reserved for output
	budget := NewTokenBudget(1000, 200)

	// Available should be 800 (1000 - 0 - 200)
	if budget.Available() != 800 {
		t.Errorf("Available() = %v, want 800", budget.Available())
	}

	// Use 300 tokens
	if !budget.Use(300) {
		t.Error("Use(300) failed, should succeed")
	}

	// Available should be 500 (800 - 300)
	if budget.Available() != 500 {
		t.Errorf("Available() = %v, want 500", budget.Available())
	}

	// Try to use 600 tokens (should fail)
	if budget.Use(600) {
		t.Error("Use(600) succeeded, should fail")
	}

	// Use 500 tokens (should succeed)
	if !budget.Use(500) {
		t.Error("Use(500) failed, should succeed")
	}

	// Available should be 0
	if budget.Available() != 0 {
		t.Errorf("Available() = %v, want 0", budget.Available())
	}

	// Reset
	budget.Reset()

	// Available should be 800 again
	if budget.Available() != 800 {
		t.Errorf("Available() = %v, want 800", budget.Available())
	}
}

func TestTokenCounter_EstimateTokensForModel(t *testing.T) {
	tc := NewTokenCounter()
	text := "Hello, world! This is a test."

	tests := []struct {
		name      string
		modelName string
		wantRange [2]int // min and max expected tokens
	}{
		{
			name:      "gpt model",
			modelName: "gpt-4",
			wantRange: [2]int{6, 8},
		},
		{
			name:      "claude model",
			modelName: "claude-3-sonnet",
			wantRange: [2]int{7, 10},
		},
		{
			name:      "qwen model",
			modelName: "qwen2.5:14b",
			wantRange: [2]int{6, 8},
		},
		{
			name:      "deepseek model",
			modelName: "deepseek-coder:33b",
			wantRange: [2]int{6, 8},
		},
		{
			name:      "unknown model",
			modelName: "unknown-model",
			wantRange: [2]int{6, 8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tc.EstimateTokensForModel(text, tt.modelName)
			if got < tt.wantRange[0] || got > tt.wantRange[1] {
				t.Errorf("EstimateTokensForModel() = %v, want between %v and %v",
					got, tt.wantRange[0], tt.wantRange[1])
			}
		})
	}
}
