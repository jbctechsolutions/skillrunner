package tokenizer

import (
	"testing"
)

func TestNewEstimator(t *testing.T) {
	estimator, err := NewEstimator()
	if err != nil {
		t.Fatalf("NewEstimator() error: %v", err)
	}
	if estimator == nil {
		t.Fatal("expected non-nil Estimator")
	}
}

func TestEstimator_CountTokens(t *testing.T) {
	estimator, err := NewEstimator()
	if err != nil {
		t.Fatalf("NewEstimator() error: %v", err)
	}

	tests := []struct {
		name      string
		text      string
		minTokens int
		maxTokens int
	}{
		{
			name:      "empty string",
			text:      "",
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "single word",
			text:      "hello",
			minTokens: 1,
			maxTokens: 2,
		},
		{
			name:      "simple sentence",
			text:      "Hello, world!",
			minTokens: 3,
			maxTokens: 6,
		},
		{
			name:      "longer text",
			text:      "The quick brown fox jumps over the lazy dog.",
			minTokens: 8,
			maxTokens: 15,
		},
		{
			name:      "code snippet",
			text:      "func main() { fmt.Println(\"Hello, World!\") }",
			minTokens: 10,
			maxTokens: 25,
		},
		{
			name:      "json data",
			text:      `{"name": "test", "value": 123, "nested": {"key": "value"}}`,
			minTokens: 15,
			maxTokens: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := estimator.CountTokens(tt.text)
			if count < tt.minTokens || count > tt.maxTokens {
				t.Errorf("CountTokens(%q) = %d, expected between %d and %d",
					tt.text, count, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestEstimator_CountTokens_Consistency(t *testing.T) {
	estimator, err := NewEstimator()
	if err != nil {
		t.Fatalf("NewEstimator() error: %v", err)
	}

	text := "This is a test sentence for token counting."

	// Count tokens multiple times to ensure consistency
	count1 := estimator.CountTokens(text)
	count2 := estimator.CountTokens(text)
	count3 := estimator.CountTokens(text)

	if count1 != count2 || count2 != count3 {
		t.Errorf("Token counts are inconsistent: %d, %d, %d", count1, count2, count3)
	}
}

func TestEstimator_CountTokens_ThreadSafety(t *testing.T) {
	estimator, err := NewEstimator()
	if err != nil {
		t.Fatalf("NewEstimator() error: %v", err)
	}

	text := "Thread safety test text."
	done := make(chan bool)

	// Run multiple goroutines counting tokens simultaneously
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = estimator.CountTokens(text)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEstimateOutputTokens(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
		fraction  float64
		expected  int
	}{
		{
			name:      "default fraction",
			maxTokens: 1000,
			fraction:  0.5,
			expected:  500,
		},
		{
			name:      "custom fraction",
			maxTokens: 1000,
			fraction:  0.25,
			expected:  250,
		},
		{
			name:      "zero max tokens uses default",
			maxTokens: 0,
			fraction:  0.5,
			expected:  500,
		},
		{
			name:      "negative max tokens uses default",
			maxTokens: -100,
			fraction:  0.5,
			expected:  500,
		},
		{
			name:      "zero fraction defaults to 0.5",
			maxTokens: 1000,
			fraction:  0,
			expected:  500,
		},
		{
			name:      "negative fraction defaults to 0.5",
			maxTokens: 1000,
			fraction:  -0.5,
			expected:  500,
		},
		{
			name:      "fraction > 1 defaults to 0.5",
			maxTokens: 1000,
			fraction:  1.5,
			expected:  500,
		},
		{
			name:      "fraction = 1",
			maxTokens: 1000,
			fraction:  1.0,
			expected:  1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateOutputTokens(tt.maxTokens, tt.fraction)
			if result != tt.expected {
				t.Errorf("EstimateOutputTokens(%d, %f) = %d, expected %d",
					tt.maxTokens, tt.fraction, result, tt.expected)
			}
		})
	}
}

func TestNewSimpleEstimator(t *testing.T) {
	estimator := NewSimpleEstimator()
	if estimator == nil {
		t.Fatal("expected non-nil SimpleEstimator")
	}
}

func TestSimpleEstimator_CountTokens(t *testing.T) {
	estimator := NewSimpleEstimator()

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
			name:     "4 characters",
			text:     "test",
			expected: 1,
		},
		{
			name:     "5 characters",
			text:     "hello",
			expected: 2,
		},
		{
			name:     "8 characters",
			text:     "12345678",
			expected: 2,
		},
		{
			name:     "12 characters",
			text:     "123456789012",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := estimator.CountTokens(tt.text)
			if count != tt.expected {
				t.Errorf("CountTokens(%q) = %d, expected %d", tt.text, count, tt.expected)
			}
		})
	}
}

func TestEstimator_ImplementsInterface(t *testing.T) {
	// This test ensures both estimators implement the TokenEstimator interface
	estimator, err := NewEstimator()
	if err != nil {
		t.Fatalf("NewEstimator() error: %v", err)
	}

	simpleEstimator := NewSimpleEstimator()

	// Both should return positive values for non-empty text
	text := "Test text"

	if estimator.CountTokens(text) <= 0 {
		t.Error("Estimator.CountTokens should return positive for non-empty text")
	}
	if simpleEstimator.CountTokens(text) <= 0 {
		t.Error("SimpleEstimator.CountTokens should return positive for non-empty text")
	}
}

func BenchmarkEstimator_CountTokens(b *testing.B) {
	estimator, err := NewEstimator()
	if err != nil {
		b.Fatalf("NewEstimator() error: %v", err)
	}

	text := "This is a benchmark test for token counting performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = estimator.CountTokens(text)
	}
}

func BenchmarkSimpleEstimator_CountTokens(b *testing.B) {
	estimator := NewSimpleEstimator()
	text := "This is a benchmark test for token counting performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = estimator.CountTokens(text)
	}
}
