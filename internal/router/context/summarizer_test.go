package context

import (
	"strings"
	"testing"
)

func TestSummarizer_Summarize(t *testing.T) {
	summarizer := NewSummarizer()

	t.Run("summarize short content", func(t *testing.T) {
		content := "This is a short text that doesn't need summarization."
		summary := summarizer.Summarize(content, 100)

		if summary == "" {
			t.Error("Summary should not be empty")
		}
		if len(summary) > len(content) {
			t.Error("Summary should not be longer than original")
		}
	})

	t.Run("summarize long content", func(t *testing.T) {
		content := strings.Repeat("This is a sentence. ", 100)
		maxLength := 200
		summary := summarizer.Summarize(content, maxLength)

		if summary == "" {
			t.Error("Summary should not be empty")
		}
		if len(summary) > maxLength {
			t.Errorf("Summary length %d exceeds max length %d", len(summary), maxLength)
		}
	})

	t.Run("summarize with very small max length", func(t *testing.T) {
		content := "This is a longer piece of content that needs to be summarized significantly."
		maxLength := 20
		summary := summarizer.Summarize(content, maxLength)

		if summary == "" {
			t.Error("Summary should not be empty")
		}
		if len(summary) > maxLength {
			t.Errorf("Summary length %d exceeds max length %d", len(summary), maxLength)
		}
	})

	t.Run("summarize empty content", func(t *testing.T) {
		summary := summarizer.Summarize("", 100)

		if summary != "" {
			t.Errorf("Expected empty summary for empty content, got %q", summary)
		}
	})

	t.Run("summarize with zero max length", func(t *testing.T) {
		content := "Some content here."
		summary := summarizer.Summarize(content, 0)

		if summary != "" {
			t.Errorf("Expected empty summary for zero max length, got %q", summary)
		}
	})
}

func TestSummarizer_SummarizeChunk(t *testing.T) {
	summarizer := NewSummarizer()

	chunk := &Chunk{
		Content:  strings.Repeat("This is a sentence. ", 50),
		FilePath: "/test/file.md",
		Index:    0,
		Start:    0,
		End:      1000,
	}

	maxLength := 100
	summary := summarizer.SummarizeChunk(chunk, maxLength)

	if summary == nil {
		t.Fatal("SummarizeChunk returned nil")
	}

	if summary.Content == "" {
		t.Error("Summary content should not be empty")
	}

	if len(summary.Content) > maxLength {
		t.Errorf("Summary length %d exceeds max length %d", len(summary.Content), maxLength)
	}

	if summary.FilePath != chunk.FilePath {
		t.Errorf("Summary file path = %s, want %s", summary.FilePath, chunk.FilePath)
	}

	if summary.Index != chunk.Index {
		t.Errorf("Summary index = %d, want %d", summary.Index, chunk.Index)
	}
}

func TestSummarizer_SummarizeChunk_NilChunk(t *testing.T) {
	summarizer := NewSummarizer()

	summary := summarizer.SummarizeChunk(nil, 100)

	if summary != nil {
		t.Error("Expected nil summary for nil chunk")
	}
}

func TestSummarizer_SummarizeChunks(t *testing.T) {
	summarizer := NewSummarizer()

	chunks := []*Chunk{
		{
			Content:  strings.Repeat("Sentence one. ", 20),
			FilePath: "/test/file1.md",
			Index:    0,
		},
		{
			Content:  strings.Repeat("Sentence two. ", 20),
			FilePath: "/test/file1.md",
			Index:    1,
		},
		{
			Content:  strings.Repeat("Sentence three. ", 20),
			FilePath: "/test/file2.md",
			Index:    0,
		},
	}

	maxLength := 50
	summaries := summarizer.SummarizeChunks(chunks, maxLength)

	if len(summaries) != len(chunks) {
		t.Errorf("Expected %d summaries, got %d", len(chunks), len(summaries))
	}

	for i, summary := range summaries {
		if summary == nil {
			t.Errorf("Summary %d is nil", i)
			continue
		}
		if len(summary.Content) > maxLength {
			t.Errorf("Summary %d length %d exceeds max length %d", i, len(summary.Content), maxLength)
		}
		if summary.FilePath != chunks[i].FilePath {
			t.Errorf("Summary %d file path mismatch", i)
		}
	}
}

func TestSummarizer_SummarizeChunks_Empty(t *testing.T) {
	summarizer := NewSummarizer()

	summaries := summarizer.SummarizeChunks([]*Chunk{}, 100)

	if len(summaries) != 0 {
		t.Errorf("Expected 0 summaries, got %d", len(summaries))
	}
}

func TestSummarizer_ExtractKeyPoints(t *testing.T) {
	summarizer := NewSummarizer()

	content := `
	This document discusses several important topics.
	First, we have topic A which is very important.
	Second, topic B is also significant.
	Third, topic C completes the discussion.
	Finally, we conclude with topic D.
	`

	keyPoints := summarizer.ExtractKeyPoints(content, 3)

	if len(keyPoints) == 0 {
		t.Error("Expected at least one key point")
	}

	if len(keyPoints) > 3 {
		t.Errorf("Expected at most 3 key points, got %d", len(keyPoints))
	}

	for _, point := range keyPoints {
		if point == "" {
			t.Error("Key point should not be empty")
		}
	}
}

func TestSummarizer_ExtractKeyPoints_Empty(t *testing.T) {
	summarizer := NewSummarizer()

	keyPoints := summarizer.ExtractKeyPoints("", 5)

	if len(keyPoints) != 0 {
		t.Errorf("Expected 0 key points for empty content, got %d", len(keyPoints))
	}
}

func TestSummarizer_ExtractKeyPoints_ShortContent(t *testing.T) {
	summarizer := NewSummarizer()

	content := "Short content here."
	keyPoints := summarizer.ExtractKeyPoints(content, 10)

	// Should return at least the content itself or a single point
	if len(keyPoints) == 0 {
		t.Error("Expected at least one key point")
	}
}

func TestNewSummarizer(t *testing.T) {
	summarizer := NewSummarizer()

	if summarizer == nil {
		t.Fatal("NewSummarizer returned nil")
	}
}

func TestSummarizer_TruncateToSentence(t *testing.T) {
	summarizer := NewSummarizer()

	t.Run("truncate at sentence boundary", func(t *testing.T) {
		content := "First sentence. Second sentence. Third sentence."
		result := summarizer.TruncateToSentence(content, 30)

		// Should end at a sentence boundary (period, exclamation, or question mark)
		hasSentenceEnding := strings.HasSuffix(result, ".") ||
			strings.HasSuffix(result, "!") ||
			strings.HasSuffix(result, "?") ||
			strings.HasSuffix(result, "...")
		if !hasSentenceEnding {
			t.Errorf("Truncated content should end at sentence boundary, got %q", result)
		}
		// Allow some flexibility for trimmed whitespace
		if len(result) > 35 {
			t.Errorf("Truncated content length %d exceeds reasonable max %d", len(result), 35)
		}
	})

	t.Run("truncate content without sentence endings", func(t *testing.T) {
		content := "No sentence endings here just text"
		result := summarizer.TruncateToSentence(content, 20)

		// Should truncate and add ellipsis, so might be slightly over due to "..."
		if len(result) > 25 {
			t.Errorf("Truncated content length %d exceeds reasonable max %d", len(result), 25)
		}
		if result == "" {
			t.Error("Result should not be empty")
		}
	})

	t.Run("truncate empty content", func(t *testing.T) {
		result := summarizer.TruncateToSentence("", 100)

		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("truncate with zero max length", func(t *testing.T) {
		content := "Some content here."
		result := summarizer.TruncateToSentence(content, 0)

		if result != "" {
			t.Errorf("Expected empty string for zero max length, got %q", result)
		}
	})

	t.Run("truncate with no word boundary", func(t *testing.T) {
		content := "VeryLongWordWithoutSpacesHere"
		result := summarizer.TruncateToSentence(content, 10)

		if result == "" {
			t.Error("Result should not be empty")
		}
	})

	t.Run("truncate with sentence at end", func(t *testing.T) {
		content := "First sentence. Second sentence."
		result := summarizer.TruncateToSentence(content, 100)

		// Should return full content since it fits
		if result != content {
			t.Errorf("Expected full content, got %q", result)
		}
	})
}

func TestSummarizer_Summarize_EdgeCases(t *testing.T) {
	summarizer := NewSummarizer()

	t.Run("summarize with exact length", func(t *testing.T) {
		content := strings.Repeat("a", 100)
		summary := summarizer.Summarize(content, 100)

		if len(summary) > 100 {
			t.Errorf("Summary length %d exceeds max %d", len(summary), 100)
		}
	})

	t.Run("summarize with very long content", func(t *testing.T) {
		content := strings.Repeat("This is a sentence. ", 1000)
		summary := summarizer.Summarize(content, 200)

		if len(summary) > 200 {
			t.Errorf("Summary length %d exceeds max %d", len(summary), 200)
		}
		if summary == "" {
			t.Error("Summary should not be empty")
		}
	})
}

func TestSummarizer_ExtractKeyPoints_EdgeCases(t *testing.T) {
	summarizer := NewSummarizer()

	t.Run("extract with zero max points", func(t *testing.T) {
		content := "First. Second. Third."
		points := summarizer.ExtractKeyPoints(content, 0)

		if len(points) != 0 {
			t.Errorf("Expected 0 points for zero max, got %d", len(points))
		}
	})

	t.Run("extract with single sentence", func(t *testing.T) {
		content := "Only one sentence here."
		points := summarizer.ExtractKeyPoints(content, 5)

		if len(points) != 1 {
			t.Errorf("Expected 1 point, got %d", len(points))
		}
	})

	t.Run("extract with many sentences", func(t *testing.T) {
		content := strings.Repeat("Sentence. ", 100)
		points := summarizer.ExtractKeyPoints(content, 3)

		if len(points) > 3 {
			t.Errorf("Expected at most 3 points, got %d", len(points))
		}
		if len(points) == 0 {
			t.Error("Expected at least one point")
		}
	})
}

func TestSummarizer_SplitSentences_EdgeCases(t *testing.T) {
	summarizer := NewSummarizer()

	t.Run("split sentences with no endings", func(t *testing.T) {
		content := "No sentence endings here"
		sentences := summarizer.splitSentences(content)

		if len(sentences) != 1 {
			t.Errorf("Expected 1 sentence, got %d", len(sentences))
		}
	})

	t.Run("split sentences with multiple punctuation", func(t *testing.T) {
		content := "First!!! Second??? Third..."
		sentences := summarizer.splitSentences(content)

		if len(sentences) == 0 {
			t.Error("Expected at least one sentence")
		}
	})

	t.Run("split sentences with only whitespace", func(t *testing.T) {
		content := "   \n\n   "
		sentences := summarizer.splitSentences(content)

		// Should handle whitespace-only content
		if len(sentences) > 1 {
			t.Errorf("Expected 0 or 1 sentence for whitespace, got %d", len(sentences))
		}
	})
}
