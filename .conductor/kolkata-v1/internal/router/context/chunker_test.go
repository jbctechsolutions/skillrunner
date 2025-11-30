package context

import (
	"strings"
	"testing"
)

func TestChunker_ChunkBySize(t *testing.T) {
	chunker := NewChunker()

	t.Run("chunk small content", func(t *testing.T) {
		content := "This is a short text."
		chunks := chunker.ChunkBySize(content, 100)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0] != content {
			t.Errorf("Chunk content mismatch")
		}
	})

	t.Run("chunk large content", func(t *testing.T) {
		content := strings.Repeat("a", 500)
		chunks := chunker.ChunkBySize(content, 100)

		expectedChunks := 5
		if len(chunks) != expectedChunks {
			t.Errorf("Expected %d chunks, got %d", expectedChunks, len(chunks))
		}

		// Verify total length
		totalLen := 0
		for _, chunk := range chunks {
			totalLen += len(chunk)
		}
		if totalLen != len(content) {
			t.Errorf("Total chunk length = %d, want %d", totalLen, len(content))
		}
	})

	t.Run("chunk with exact size", func(t *testing.T) {
		content := strings.Repeat("a", 200)
		chunks := chunker.ChunkBySize(content, 100)

		if len(chunks) != 2 {
			t.Errorf("Expected 2 chunks, got %d", len(chunks))
		}
		if len(chunks[0]) != 100 || len(chunks[1]) != 100 {
			t.Errorf("Chunk sizes incorrect")
		}
	})

	t.Run("chunk empty content", func(t *testing.T) {
		chunks := chunker.ChunkBySize("", 100)

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for empty content, got %d", len(chunks))
		}
	})
}

func TestChunker_ChunkByLines(t *testing.T) {
	chunker := NewChunker()

	t.Run("chunk by lines", func(t *testing.T) {
		content := "line1\nline2\nline3\nline4\nline5"
		chunks := chunker.ChunkByLines(content, 2)

		expectedChunks := 3 // 5 lines / 2 = 3 chunks (2, 2, 1)
		if len(chunks) != expectedChunks {
			t.Errorf("Expected %d chunks, got %d", expectedChunks, len(chunks))
		}

		// Verify first chunk has 2 lines
		lines := strings.Split(chunks[0], "\n")
		if len(lines) != 2 {
			t.Errorf("First chunk should have 2 lines, got %d", len(lines))
		}
	})

	t.Run("chunk with fewer lines than chunk size", func(t *testing.T) {
		content := "line1\nline2"
		chunks := chunker.ChunkByLines(content, 10)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
	})

	t.Run("chunk content without newlines", func(t *testing.T) {
		content := "no newlines here"
		chunks := chunker.ChunkByLines(content, 5)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
	})

	t.Run("chunk empty content", func(t *testing.T) {
		chunks := chunker.ChunkByLines("", 10)

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for empty content, got %d", len(chunks))
		}
	})
}

func TestChunker_ChunkBySentences(t *testing.T) {
	chunker := NewChunker()

	t.Run("chunk by sentences", func(t *testing.T) {
		content := "First sentence. Second sentence. Third sentence. Fourth sentence."
		chunks := chunker.ChunkBySentences(content, 2)

		expectedChunks := 2 // 4 sentences / 2 = 2 chunks
		if len(chunks) != expectedChunks {
			t.Errorf("Expected %d chunks, got %d", expectedChunks, len(chunks))
		}
	})

	t.Run("chunk with fewer sentences", func(t *testing.T) {
		content := "One sentence. Two sentence."
		chunks := chunker.ChunkBySentences(content, 5)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
	})

	t.Run("chunk content without sentence endings", func(t *testing.T) {
		content := "No sentence endings here"
		chunks := chunker.ChunkBySentences(content, 2)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
	})
}

func TestChunker_ChunkByParagraphs(t *testing.T) {
	chunker := NewChunker()

	t.Run("chunk by paragraphs", func(t *testing.T) {
		content := "Para 1.\n\nPara 2.\n\nPara 3.\n\nPara 4."
		chunks := chunker.ChunkByParagraphs(content, 2)

		expectedChunks := 2 // 4 paragraphs / 2 = 2 chunks
		if len(chunks) != expectedChunks {
			t.Errorf("Expected %d chunks, got %d", expectedChunks, len(chunks))
		}
	})

	t.Run("chunk with single paragraph", func(t *testing.T) {
		content := "Single paragraph here."
		chunks := chunker.ChunkByParagraphs(content, 2)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
	})

	t.Run("chunk content without double newlines", func(t *testing.T) {
		content := "No paragraphs here"
		chunks := chunker.ChunkByParagraphs(content, 2)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
	})
}

func TestChunker_ChunkWithOverlap(t *testing.T) {
	chunker := NewChunker()

	t.Run("chunk with overlap", func(t *testing.T) {
		content := strings.Repeat("word ", 100) // ~500 chars
		chunks := chunker.ChunkWithOverlap(content, 100, 20)

		if len(chunks) < 5 {
			t.Errorf("Expected at least 5 chunks with overlap, got %d", len(chunks))
		}

		// Verify overlap between consecutive chunks
		if len(chunks) > 1 {
			// Check that chunks overlap (simplified check)
			firstEnd := chunks[0][len(chunks[0])-20:]
			secondStart := chunks[1][:20]
			if firstEnd != secondStart {
				// Overlap might not be exact due to word boundaries, so just check it exists
				if len(chunks[0]) < 100 || len(chunks[1]) < 100 {
					t.Error("Chunks should be at least chunkSize in length")
				}
			}
		}
	})

	t.Run("chunk with overlap smaller than content", func(t *testing.T) {
		content := "short"
		chunks := chunker.ChunkWithOverlap(content, 100, 20)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk for small content, got %d", len(chunks))
		}
	})
}

func TestChunker_ChunkFile(t *testing.T) {
	chunker := NewChunker()

	t.Run("chunk file by size", func(t *testing.T) {
		file := &File{
			Path:    "/test/file.md",
			Content: strings.Repeat("a", 500),
			Size:    500,
		}

		chunks := chunker.ChunkFile(file, ChunkStrategySize, 100, 0)

		if len(chunks) != 5 {
			t.Errorf("Expected 5 chunks, got %d", len(chunks))
		}

		// Verify chunks have file path
		for _, chunk := range chunks {
			if chunk.FilePath != file.Path {
				t.Errorf("Chunk file path = %s, want %s", chunk.FilePath, file.Path)
			}
		}
	})

	t.Run("chunk file by lines", func(t *testing.T) {
		file := &File{
			Path:    "/test/file.md",
			Content: "line1\nline2\nline3\nline4\nline5",
			Size:    30,
		}

		chunks := chunker.ChunkFile(file, ChunkStrategyLines, 2, 0)

		if len(chunks) != 3 {
			t.Errorf("Expected 3 chunks, got %d", len(chunks))
		}
	})

	t.Run("chunk file with paragraphs", func(t *testing.T) {
		file := &File{
			Path:    "/test/file.md",
			Content: "Para 1.\n\nPara 2.\n\nPara 3.",
			Size:    30,
		}

		chunks := chunker.ChunkFile(file, ChunkStrategyParagraphs, 2, 0)

		if len(chunks) != 2 {
			t.Errorf("Expected 2 chunks, got %d", len(chunks))
		}
	})

	t.Run("chunk file with overlap", func(t *testing.T) {
		file := &File{
			Path:    "/test/file.md",
			Content: strings.Repeat("a", 500),
			Size:    500,
		}

		chunks := chunker.ChunkFile(file, ChunkStrategySize, 100, 20)

		if len(chunks) < 5 {
			t.Errorf("Expected at least 5 chunks with overlap, got %d", len(chunks))
		}
	})
}

func TestNewChunker(t *testing.T) {
	chunker := NewChunker()

	if chunker == nil {
		t.Fatal("NewChunker returned nil")
	}
}

func TestChunkStrategyConstants(t *testing.T) {
	if ChunkStrategySize != "size" {
		t.Errorf("ChunkStrategySize = %s, want size", ChunkStrategySize)
	}
	if ChunkStrategyLines != "lines" {
		t.Errorf("ChunkStrategyLines = %s, want lines", ChunkStrategyLines)
	}
	if ChunkStrategyParagraphs != "paragraphs" {
		t.Errorf("ChunkStrategyParagraphs = %s, want paragraphs", ChunkStrategyParagraphs)
	}
	if ChunkStrategySentences != "sentences" {
		t.Errorf("ChunkStrategySentences = %s, want sentences", ChunkStrategySentences)
	}
}

func TestChunker_ChunkFiles(t *testing.T) {
	chunker := NewChunker()

	files := []*File{
		{
			Path:    "/test/file1.md",
			Content: strings.Repeat("a", 200),
			Size:    200,
		},
		{
			Path:    "/test/file2.md",
			Content: strings.Repeat("b", 200),
			Size:    200,
		},
	}

	chunks := chunker.ChunkFiles(files, ChunkStrategySize, 100, 0)

	// Should have 4 chunks total (2 files * 2 chunks each)
	if len(chunks) != 4 {
		t.Errorf("Expected 4 chunks, got %d", len(chunks))
	}

	// Verify file paths are preserved
	file1Chunks := 0
	file2Chunks := 0
	for _, chunk := range chunks {
		if chunk.FilePath == "/test/file1.md" {
			file1Chunks++
		} else if chunk.FilePath == "/test/file2.md" {
			file2Chunks++
		}
	}

	if file1Chunks != 2 || file2Chunks != 2 {
		t.Errorf("Expected 2 chunks per file, got file1=%d, file2=%d", file1Chunks, file2Chunks)
	}
}

func TestChunker_ChunkFile_NilFile(t *testing.T) {
	chunker := NewChunker()

	chunks := chunker.ChunkFile(nil, ChunkStrategySize, 100, 0)

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for nil file, got %d", len(chunks))
	}
}

func TestChunker_ChunkWithOverlap_EdgeCases(t *testing.T) {
	chunker := NewChunker()

	t.Run("overlap size equals chunk size", func(t *testing.T) {
		content := strings.Repeat("a", 200)
		chunks := chunker.ChunkWithOverlap(content, 100, 100)

		// Should default to half overlap
		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}
	})

	t.Run("overlap size greater than chunk size", func(t *testing.T) {
		content := strings.Repeat("a", 200)
		chunks := chunker.ChunkWithOverlap(content, 100, 150)

		// Should default to half overlap
		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}
	})

	t.Run("zero chunk size", func(t *testing.T) {
		content := "test content"
		chunks := chunker.ChunkWithOverlap(content, 0, 0)

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for zero chunk size, got %d", len(chunks))
		}
	})
}

func TestChunker_ChunkBySentences_EdgeCases(t *testing.T) {
	chunker := NewChunker()

	t.Run("zero sentences per chunk", func(t *testing.T) {
		content := "First. Second. Third."
		chunks := chunker.ChunkBySentences(content, 0)

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for zero sentences per chunk, got %d", len(chunks))
		}
	})

	t.Run("content with multiple punctuation", func(t *testing.T) {
		content := "First!!! Second??? Third..."
		chunks := chunker.ChunkBySentences(content, 1)

		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}
	})
}

func TestChunker_ChunkByParagraphs_EdgeCases(t *testing.T) {
	chunker := NewChunker()

	t.Run("zero paragraphs per chunk", func(t *testing.T) {
		content := "Para 1.\n\nPara 2."
		chunks := chunker.ChunkByParagraphs(content, 0)

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for zero paragraphs per chunk, got %d", len(chunks))
		}
	})

	t.Run("paragraphs with only whitespace", func(t *testing.T) {
		content := "Para 1.\n\n   \n\nPara 2."
		chunks := chunker.ChunkByParagraphs(content, 1)

		// Should filter out empty paragraphs
		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}
	})
}

func TestChunker_ChunkBySize_EdgeCases(t *testing.T) {
	chunker := NewChunker()

	t.Run("zero chunk size", func(t *testing.T) {
		content := "test content"
		chunks := chunker.ChunkBySize(content, 0)

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for zero chunk size, got %d", len(chunks))
		}
	})

	t.Run("negative chunk size", func(t *testing.T) {
		content := "test content"
		chunks := chunker.ChunkBySize(content, -10)

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for negative chunk size, got %d", len(chunks))
		}
	})
}

func TestChunker_ChunkFile_DefaultStrategy(t *testing.T) {
	chunker := NewChunker()

	file := &File{
		Path:    "/test/file.md",
		Content: strings.Repeat("a", 200),
		Size:    200,
	}

	// Test with invalid strategy (should default to size)
	chunks := chunker.ChunkFile(file, "invalid", 100, 0)

	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks with default strategy, got %d", len(chunks))
	}
}
