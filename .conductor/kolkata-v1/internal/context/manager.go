package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/llm"
)

// Config holds context management configuration
type Config struct {
	ChunkingEnabled       bool
	ChunkingStrategy      ChunkingStrategy
	ChunkOverlapTokens    int
	SummarizationEnabled  bool
	SummarizationStrategy SummarizationStrategy
	DefaultModel          string // Model to use for abstractive summarization
}

// DefaultConfig returns the default context management configuration
func DefaultConfig() *Config {
	return &Config{
		ChunkingEnabled:       true,
		ChunkingStrategy:      HierarchicalChunking,
		ChunkOverlapTokens:    100,
		SummarizationEnabled:  true,
		SummarizationStrategy: HybridSummarization,
		DefaultModel:          "ollama/qwen2.5:14b", // Fast local model for summarization
	}
}

// Manager manages context optimization through chunking and summarization
type Manager struct {
	config     *Config
	counter    *TokenCounter
	chunker    *Chunker
	summarizer *Summarizer
	llmClient  *llm.Client
}

// NewManager creates a new context manager
func NewManager(config *Config, llmClient *llm.Client) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	chunker := NewChunker(config.ChunkingStrategy)
	summarizer := NewSummarizer(config.SummarizationStrategy, llmClient, config.DefaultModel)

	return &Manager{
		config:     config,
		counter:    NewTokenCounter(),
		chunker:    chunker,
		summarizer: summarizer,
		llmClient:  llmClient,
	}
}

// PrepareContext prepares context for a model, optimizing if necessary
// Returns the optimized context and whether optimization was applied
func (m *Manager) PrepareContext(ctx context.Context, text string, maxTokens int) (string, bool, error) {
	// Count tokens in current text
	currentTokens := m.counter.Count(text)

	// If text fits, return as-is
	if currentTokens <= maxTokens {
		return text, false, nil
	}

	// Text exceeds limit, need to optimize
	optimized := false

	// Try chunking first if enabled
	if m.config.ChunkingEnabled {
		// For now, just return the first chunk
		// In a more sophisticated implementation, we could:
		// 1. Return all chunks for parallel processing
		// 2. Use semantic search to find most relevant chunk
		// 3. Combine multiple chunks intelligently
		chunks, err := m.chunker.ChunkIfNeeded(text, maxTokens)
		if err != nil {
			return "", false, fmt.Errorf("chunking failed: %w", err)
		}

		if len(chunks) == 1 {
			return chunks[0].Content, false, nil
		}

		// Multiple chunks created, select best chunk(s)
		selected := m.selectBestChunks(chunks, maxTokens)
		return selected, true, nil
	}

	// If summarization is enabled and chunking didn't work, summarize
	if m.config.SummarizationEnabled {
		summary, err := m.summarizer.Summarize(ctx, text, maxTokens)
		if err != nil {
			return "", false, fmt.Errorf("summarization failed: %w", err)
		}
		optimized = true
		return summary, optimized, nil
	}

	// No optimization available, truncate as last resort
	// This is not ideal but prevents complete failure
	return m.truncate(text, maxTokens), true, nil
}

// PreparePhaseContext prepares context for a specific phase
// Takes into account required vs optional inputs
func (m *Manager) PreparePhaseContext(
	ctx context.Context,
	phaseInputs map[string]string,
	requiredKeys []string,
	optionalKeys []string,
	maxTokens int,
) (string, error) {
	// Build full context
	parts := []string{}

	// Add required inputs first
	for _, key := range requiredKeys {
		if value, ok := phaseInputs[key]; ok {
			parts = append(parts, fmt.Sprintf("## %s\n%s\n", key, value))
		}
	}

	// Check if we have budget for optional inputs
	currentContext := ""
	if len(parts) > 0 {
		currentContext = join(parts)
	}

	currentTokens := m.counter.Count(currentContext)

	// If required inputs exceed budget, we need to compress them
	if currentTokens > maxTokens {
		// Try to summarize required inputs
		if m.config.SummarizationEnabled {
			compressed, err := m.summarizer.Summarize(ctx, currentContext, maxTokens)
			if err != nil {
				return "", fmt.Errorf("failed to compress required inputs: %w", err)
			}
			return compressed, nil
		}

		// Can't compress, return truncated
		return m.truncate(currentContext, maxTokens), nil
	}

	// Add optional inputs if space permits
	budget := maxTokens - currentTokens

	for _, key := range optionalKeys {
		if value, ok := phaseInputs[key]; ok {
			keySection := fmt.Sprintf("## %s\n%s\n", key, value)
			sectionTokens := m.counter.Count(keySection)

			if sectionTokens <= budget {
				// Fits, add it
				parts = append(parts, keySection)
				budget -= sectionTokens
			} else if budget > 100 && m.config.SummarizationEnabled {
				// Try to compress and fit
				compressed, err := m.summarizer.SummarizeIfNeeded(ctx, value, budget-20) // Leave buffer for header
				if err != nil {
					// Skip this optional input
					continue
				}
				compressedSection := fmt.Sprintf("## %s\n%s\n", key, compressed)
				parts = append(parts, compressedSection)
				budget = maxTokens - m.counter.Count(join(parts))
			}
			// else: Not enough budget, skip this optional input
		}
	}

	return join(parts), nil
}

// ChunkText splits text into chunks if it exceeds maxTokens
func (m *Manager) ChunkText(text string, maxTokens int) ([]Chunk, error) {
	if !m.config.ChunkingEnabled {
		// Return single chunk
		return []Chunk{
			{
				Content: text,
				Tokens:  m.counter.Count(text),
				Index:   0,
			},
		}, nil
	}

	return m.chunker.ChunkIfNeeded(text, maxTokens)
}

// SummarizeText summarizes text to fit within targetTokens
func (m *Manager) SummarizeText(ctx context.Context, text string, targetTokens int) (string, error) {
	if !m.config.SummarizationEnabled {
		// Return truncated text
		return m.truncate(text, targetTokens), nil
	}

	return m.summarizer.Summarize(ctx, text, targetTokens)
}

// CountTokens counts tokens in text
func (m *Manager) CountTokens(text string) int {
	return m.counter.Count(text)
}

// CountTokensForModel counts tokens for a specific model
func (m *Manager) CountTokensForModel(text string, modelName string) int {
	return m.counter.EstimateTokensForModel(text, modelName)
}

// truncate truncates text to approximately fit within maxTokens
// This is a fallback when other methods fail
func (m *Manager) truncate(text string, maxTokens int) string {
	// Simple truncation: ~4 chars per token
	maxChars := maxTokens * 4

	if len(text) <= maxChars {
		return text
	}

	// Truncate and add ellipsis
	return text[:maxChars-3] + "..."
}

// join is a helper to join strings
func join(parts []string) string {
	result := ""
	for _, part := range parts {
		result += part
	}
	return result
}

// CreateBudget creates a token budget for context management
func (m *Manager) CreateBudget(contextWindow int, outputTokens int) *TokenBudget {
	return NewTokenBudget(contextWindow, outputTokens)
}

// selectBestChunks selects the best chunk(s) from multiple chunks
// Scores chunks and returns the highest-scoring chunk or merges top chunks if they fit
func (m *Manager) selectBestChunks(chunks []Chunk, maxTokens int) string {
	if len(chunks) == 0 {
		return ""
	}

	// Score each chunk
	scores := make([]float64, len(chunks))
	for i, chunk := range chunks {
		scores[i] = m.scoreChunk(chunk, i, len(chunks))
	}

	// Find best chunk
	bestIdx := 0
	bestScore := scores[0]
	for i := 1; i < len(chunks); i++ {
		if scores[i] > bestScore {
			bestScore = scores[i]
			bestIdx = i
		}
	}

	// Try to merge top chunks if they fit within token limit
	merged := m.mergeTopChunks(chunks, scores, maxTokens)
	if merged != "" {
		return merged
	}

	// Return best single chunk
	return chunks[bestIdx].Content
}

// scoreChunk scores a chunk based on various factors
func (m *Manager) scoreChunk(chunk Chunk, index int, totalChunks int) float64 {
	score := 0.0

	// Factor 1: Token count (more tokens = more information, but not too much)
	// Prefer chunks that use most of the available space
	tokenScore := float64(chunk.Tokens) / 1000.0 // Normalize
	score += tokenScore * 0.3

	// Factor 2: Section metadata (structured content is better)
	if len(chunk.Metadata.Sections) > 0 {
		score += float64(len(chunk.Metadata.Sections)) * 0.2
	}

	// Factor 3: Position (earlier chunks often contain more important info)
	// But also give some weight to middle chunks
	positionRatio := float64(index) / float64(totalChunks)
	if positionRatio < 0.3 {
		// First 30% of chunks get bonus
		score += (1.0 - positionRatio) * 0.2
	} else if positionRatio < 0.7 {
		// Middle chunks get moderate score
		score += 0.1
	}

	// Factor 4: Content quality (longer content with structure)
	contentLength := float64(len(chunk.Content))
	if contentLength > 500 {
		score += 0.1 // Bonus for substantial content
	}

	// Factor 5: Check for common important markers
	content := chunk.Content
	if contains(content, "##") || contains(content, "# ") {
		score += 0.1 // Has markdown headers
	}
	if contains(content, "```") {
		score += 0.05 // Has code blocks
	}

	return score
}

// mergeTopChunks attempts to merge top-scoring chunks if they fit within token limit
func (m *Manager) mergeTopChunks(chunks []Chunk, scores []float64, maxTokens int) string {
	// Create index-score pairs and sort by score
	type chunkScore struct {
		index int
		score float64
	}
	scored := make([]chunkScore, len(chunks))
	for i := range chunks {
		scored[i] = chunkScore{index: i, score: scores[i]}
	}

	// Simple sort (bubble sort for small arrays)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Try to merge top chunks
	merged := ""
	totalTokens := 0

	for _, cs := range scored {
		chunk := chunks[cs.index]
		chunkTokens := chunk.Tokens

		// Check if adding this chunk would exceed limit
		if totalTokens+chunkTokens > maxTokens {
			break
		}

		// Add chunk to merged result
		if merged == "" {
			merged = chunk.Content
		} else {
			merged += "\n\n---\n\n" + chunk.Content
		}
		totalTokens += chunkTokens

		// Limit to top 3 chunks to avoid too much merging
		if float64(totalTokens) > float64(maxTokens)*0.8 {
			break
		}
	}

	// Only return merged if we got at least 2 chunks
	if totalTokens > 0 && strings.Contains(merged, "---") {
		return merged
	}

	return ""
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

// indexOf finds the index of a substring (simple implementation)
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
