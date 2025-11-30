package context

import (
	"context"
	"strings"
	"testing"
)

func TestManager_PrepareContext_FitsWithinLimit(t *testing.T) {
	manager := NewManager(DefaultConfig(), nil)

	text := "Short text that fits."
	maxTokens := 100

	result, optimized, err := manager.PrepareContext(context.Background(), text, maxTokens)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}

	if optimized {
		t.Error("Expected no optimization for text within limit")
	}

	if result != text {
		t.Error("Text was modified when it should not have been")
	}
}

func TestManager_PrepareContext_ExceedsLimit(t *testing.T) {
	manager := NewManager(DefaultConfig(), nil)

	// Create long text
	text := strings.Repeat("This is a test sentence. ", 100)
	maxTokens := 50

	result, optimized, err := manager.PrepareContext(context.Background(), text, maxTokens)
	if err != nil {
		t.Fatalf("PrepareContext() error = %v", err)
	}

	if !optimized {
		t.Error("Expected optimization for text exceeding limit")
	}

	if len(result) >= len(text) {
		t.Error("Result should be shorter than original")
	}

	resultTokens := manager.CountTokens(result)
	if resultTokens > maxTokens*2 {
		t.Errorf("Result has %d tokens, which is too far from limit of %d", resultTokens, maxTokens)
	}
}

func TestManager_PreparePhaseContext_RequiredOnly(t *testing.T) {
	manager := NewManager(DefaultConfig(), nil)

	phaseInputs := map[string]string{
		"task":     "Build a web server",
		"language": "Go",
		"optional": strings.Repeat("Extra info. ", 100),
	}

	required := []string{"task", "language"}
	optional := []string{"optional"}

	result, err := manager.PreparePhaseContext(
		context.Background(),
		phaseInputs,
		required,
		optional,
		100, // Small budget
	)

	if err != nil {
		t.Fatalf("PreparePhaseContext() error = %v", err)
	}

	// Should contain required keys
	if !strings.Contains(result, "task") || !strings.Contains(result, "language") {
		t.Error("Result should contain required inputs")
	}

	// Should not contain full optional content due to budget
	if strings.Count(result, "Extra info") > 5 {
		t.Error("Result should not contain full optional content")
	}
}

func TestManager_ChunkText(t *testing.T) {
	config := DefaultConfig()
	config.ChunkingEnabled = true
	manager := NewManager(config, nil)

	text := strings.Repeat("This is a test sentence. ", 50)
	maxTokens := 50

	chunks, err := manager.ChunkText(text, maxTokens)
	if err != nil {
		t.Fatalf("ChunkText() error = %v", err)
	}

	if len(chunks) < 2 {
		t.Error("Expected multiple chunks for long text")
	}

	// Verify each chunk is within limit
	for i, chunk := range chunks {
		if chunk.Tokens > maxTokens*2 {
			t.Errorf("Chunk %d has %d tokens, exceeds reasonable limit", i, chunk.Tokens)
		}
	}
}

func TestManager_ChunkText_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.ChunkingEnabled = false
	manager := NewManager(config, nil)

	text := strings.Repeat("This is a test sentence. ", 50)
	maxTokens := 50

	chunks, err := manager.ChunkText(text, maxTokens)
	if err != nil {
		t.Fatalf("ChunkText() error = %v", err)
	}

	// Should return single chunk when disabled
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk when chunking disabled, got %d", len(chunks))
	}
}

func TestManager_CountTokens(t *testing.T) {
	manager := NewManager(DefaultConfig(), nil)

	text := "Hello, world!"
	tokens := manager.CountTokens(text)

	if tokens <= 0 {
		t.Error("Token count should be positive")
	}

	// Rough sanity check: ~4 chars per token
	expectedRange := [2]int{len(text) / 5, len(text) / 3}
	if tokens < expectedRange[0] || tokens > expectedRange[1] {
		t.Errorf("Token count %d seems unreasonable for text length %d", tokens, len(text))
	}
}

func TestManager_CreateBudget(t *testing.T) {
	manager := NewManager(DefaultConfig(), nil)

	budget := manager.CreateBudget(1000, 200)

	if budget.Total != 1000 {
		t.Errorf("Total should be 1000, got %d", budget.Total)
	}

	if budget.Reserved != 200 {
		t.Errorf("Reserved should be 200, got %d", budget.Reserved)
	}

	if budget.Available() != 800 {
		t.Errorf("Available should be 800, got %d", budget.Available())
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.ChunkingEnabled {
		t.Error("Chunking should be enabled by default")
	}

	if !config.SummarizationEnabled {
		t.Error("Summarization should be enabled by default")
	}

	if config.ChunkingStrategy != HierarchicalChunking {
		t.Errorf("Expected HierarchicalChunking, got %v", config.ChunkingStrategy)
	}

	if config.SummarizationStrategy != HybridSummarization {
		t.Errorf("Expected HybridSummarization, got %v", config.SummarizationStrategy)
	}
}
