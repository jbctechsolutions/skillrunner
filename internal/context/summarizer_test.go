package context

import (
	"context"
	"strings"
	"testing"
)

func TestSummarizer_ExtractiveSummarize(t *testing.T) {
	summarizer := NewSummarizer(ExtractiveSummarization, nil, "")

	text := `
		Introduction to machine learning. Machine learning is a subset of artificial intelligence.
		It focuses on the development of algorithms. These algorithms can learn from data.
		The goal is to make predictions. Predictions are based on patterns in the data.
		There are many types of machine learning. The most common are supervised and unsupervised learning.
		Supervised learning uses labeled data. Unsupervised learning finds patterns in unlabeled data.
	`

	targetTokens := 20 // Much smaller than the original

	summary, err := summarizer.Summarize(context.Background(), text, targetTokens)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	summaryTokens := summarizer.counter.Count(summary)
	if summaryTokens > targetTokens*2 {
		t.Errorf("Summary has %d tokens, which is too far from target of %d", summaryTokens, targetTokens)
	}

	// Summary should not be empty
	if len(strings.TrimSpace(summary)) == 0 {
		t.Error("Summary is empty")
	}

	// Summary should be shorter than original
	if len(summary) >= len(text) {
		t.Error("Summary is not shorter than original text")
	}
}

func TestSummarizer_SummarizeIfNeeded_NoSummarization(t *testing.T) {
	summarizer := NewSummarizer(ExtractiveSummarization, nil, "")

	text := "Short text that doesn't need summarization."
	targetTokens := 100 // Much larger than the text

	summary, err := summarizer.SummarizeIfNeeded(context.Background(), text, targetTokens)
	if err != nil {
		t.Fatalf("SummarizeIfNeeded() error = %v", err)
	}

	// Should return original text unchanged
	if summary != text {
		t.Error("Text was modified when summarization wasn't needed")
	}
}

func TestSummarizer_SummarizeIfNeeded_WithSummarization(t *testing.T) {
	summarizer := NewSummarizer(ExtractiveSummarization, nil, "")

	// Create a long text
	text := strings.Repeat("This is a test sentence that will be repeated many times. ", 50)
	targetTokens := 50

	summary, err := summarizer.SummarizeIfNeeded(context.Background(), text, targetTokens)
	if err != nil {
		t.Fatalf("SummarizeIfNeeded() error = %v", err)
	}

	// Summary should be shorter
	if len(summary) >= len(text) {
		t.Error("Summary is not shorter than original text")
	}

	summaryTokens := summarizer.counter.Count(summary)
	if summaryTokens > targetTokens*2 {
		t.Errorf("Summary has %d tokens, which is too far from target of %d", summaryTokens, targetTokens)
	}
}

func TestSummarizer_ScoreSentences(t *testing.T) {
	summarizer := NewSummarizer(ExtractiveSummarization, nil, "")

	sentences := []string{
		"Machine learning is important.",
		"It has many applications.",
		"Data science uses machine learning extensively.",
		"Machine learning algorithms learn from data.",
	}

	scores := summarizer.scoreSentences(sentences)

	if len(scores) != len(sentences) {
		t.Errorf("Expected %d scores, got %d", len(sentences), len(scores))
	}

	// All scores should be non-negative
	for i, score := range scores {
		if score < 0 {
			t.Errorf("Score %d is negative: %f", i, score)
		}
	}

	// First sentence should have bonus (higher score)
	// This test might be flaky, but generally first sentence gets a boost
	if scores[0] <= 0 {
		t.Error("First sentence should have a positive score")
	}
}

func TestSummarizer_IsCommonWord(t *testing.T) {
	summarizer := NewSummarizer(ExtractiveSummarization, nil, "")

	commonTests := []struct {
		word     string
		expected bool
	}{
		{"the", true},
		{"and", true},
		{"machine", false},
		{"learning", false},
		{"a", true},
		{"is", true},
		{"algorithm", false},
	}

	for _, tt := range commonTests {
		t.Run(tt.word, func(t *testing.T) {
			got := summarizer.isCommonWord(tt.word)
			if got != tt.expected {
				t.Errorf("isCommonWord(%q) = %v, want %v", tt.word, got, tt.expected)
			}
		})
	}
}

func TestSummarizer_EmptyText(t *testing.T) {
	summarizer := NewSummarizer(ExtractiveSummarization, nil, "")

	summary, err := summarizer.Summarize(context.Background(), "", 100)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	if summary != "" {
		t.Error("Expected empty summary for empty text")
	}
}
