package context

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/llm"
)

// SummarizationStrategy defines the summarization approach
type SummarizationStrategy string

const (
	// ExtractiveSummarization extracts key sentences
	ExtractiveSummarization SummarizationStrategy = "extractive"
	// AbstractiveSummarization generates new summary using LLM
	AbstractiveSummarization SummarizationStrategy = "abstractive"
	// HybridSummarization combines both approaches
	HybridSummarization SummarizationStrategy = "hybrid"
)

// Summarizer handles text summarization
type Summarizer struct {
	counter   *TokenCounter
	strategy  SummarizationStrategy
	llmClient *llm.Client
	modelName string
}

// NewSummarizer creates a new summarizer
func NewSummarizer(strategy SummarizationStrategy, llmClient *llm.Client, modelName string) *Summarizer {
	return &Summarizer{
		counter:   NewTokenCounter(),
		strategy:  strategy,
		llmClient: llmClient,
		modelName: modelName,
	}
}

// Summarize summarizes text to fit within target token count
func (s *Summarizer) Summarize(ctx context.Context, text string, targetTokens int) (string, error) {
	// Check if summarization is needed
	currentTokens := s.counter.Count(text)
	if currentTokens <= targetTokens {
		return text, nil
	}

	switch s.strategy {
	case ExtractiveSummarization:
		return s.extractiveSummarize(text, targetTokens), nil
	case AbstractiveSummarization:
		return s.abstractiveSummarize(ctx, text, targetTokens)
	case HybridSummarization:
		return s.hybridSummarize(ctx, text, targetTokens)
	default:
		return s.extractiveSummarize(text, targetTokens), nil
	}
}

// extractiveSummarize implements extractive summarization
// Selects the most important sentences based on word frequency and position
func (s *Summarizer) extractiveSummarize(text string, targetTokens int) string {
	sentences := s.counter.SplitSentences(text)
	if len(sentences) == 0 {
		return text
	}

	// If only one sentence, return it (possibly truncated)
	if len(sentences) == 1 {
		return text
	}

	// Calculate target compression ratio
	currentTokens := s.counter.Count(text)
	ratio := float64(targetTokens) / float64(currentTokens)

	// If we need to keep 90%+, just return the text
	if ratio >= 0.9 {
		return text
	}

	// Score each sentence
	scores := s.scoreSentences(sentences)

	// Sort sentences by score (descending)
	type scoredSentence struct {
		sentence string
		score    float64
		index    int
	}

	scored := make([]scoredSentence, len(sentences))
	for i, sentence := range sentences {
		scored[i] = scoredSentence{
			sentence: sentence,
			score:    scores[i],
			index:    i,
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Select sentences until we reach target tokens
	selected := []scoredSentence{}
	tokensUsed := 0

	for _, s := range scored {
		sentenceTokens := s.score // Approximate, we already computed
		if tokensUsed+int(sentenceTokens*10) <= targetTokens {
			selected = append(selected, s)
			tokensUsed += int(sentenceTokens * 10)
		}
		if tokensUsed >= targetTokens*9/10 {
			break
		}
	}

	// If no sentences selected, take the first one
	if len(selected) == 0 && len(scored) > 0 {
		selected = append(selected, scored[0])
	}

	// Sort selected sentences by original order
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].index < selected[j].index
	})

	// Build summary
	summaryParts := make([]string, len(selected))
	for i, s := range selected {
		summaryParts[i] = s.sentence
	}

	return strings.Join(summaryParts, " ")
}

// scoreSentences scores sentences based on word frequency and position
func (s *Summarizer) scoreSentences(sentences []string) []float64 {
	scores := make([]float64, len(sentences))

	// Build word frequency map (excluding common words)
	wordFreq := make(map[string]int)
	totalWords := 0

	for _, sentence := range sentences {
		words := strings.Fields(strings.ToLower(sentence))
		for _, word := range words {
			// Skip very short words and common words
			if len(word) <= 2 || s.isCommonWord(word) {
				continue
			}
			wordFreq[word]++
			totalWords++
		}
	}

	// Score each sentence
	for i, sentence := range sentences {
		words := strings.Fields(strings.ToLower(sentence))
		score := 0.0

		// Word frequency score
		for _, word := range words {
			if freq, ok := wordFreq[word]; ok {
				score += float64(freq)
			}
		}

		// Normalize by sentence length
		if len(words) > 0 {
			score /= float64(len(words))
		}

		// Position bonus (first and last sentences are usually important)
		if i == 0 {
			score *= 1.5
		} else if i == len(sentences)-1 {
			score *= 1.2
		}

		scores[i] = score
	}

	return scores
}

// isCommonWord checks if a word is a common word
func (s *Summarizer) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "be": true, "to": true, "of": true, "and": true,
		"a": true, "in": true, "that": true, "have": true, "i": true,
		"it": true, "for": true, "not": true, "on": true, "with": true,
		"he": true, "as": true, "you": true, "do": true, "at": true,
		"this": true, "but": true, "his": true, "by": true, "from": true,
		"they": true, "we": true, "say": true, "her": true, "she": true,
		"or": true, "an": true, "will": true, "my": true, "one": true,
		"all": true, "would": true, "there": true, "their": true, "what": true,
		"so": true, "up": true, "out": true, "if": true, "about": true,
		"who": true, "get": true, "which": true, "go": true, "me": true,
		"when": true, "make": true, "can": true, "like": true, "time": true,
		"no": true, "just": true, "him": true, "know": true, "take": true,
		"people": true, "into": true, "year": true, "your": true, "good": true,
		"some": true, "could": true, "them": true, "see": true, "other": true,
		"than": true, "then": true, "now": true, "look": true, "only": true,
		"come": true, "its": true, "over": true, "think": true, "also": true,
		"back": true, "after": true, "use": true, "two": true, "how": true,
		"our": true, "work": true, "first": true, "well": true, "way": true,
		"even": true, "new": true, "want": true, "because": true, "any": true,
		"these": true, "give": true, "day": true, "most": true, "us": true,
		"is": true, "was": true, "are": true, "been": true, "has": true,
		"had": true, "were": true, "said": true, "did": true, "having": true,
	}
	return commonWords[word]
}

// abstractiveSummarize implements abstractive summarization using LLM
func (s *Summarizer) abstractiveSummarize(ctx context.Context, text string, targetTokens int) (string, error) {
	if s.llmClient == nil {
		return "", fmt.Errorf("LLM client not configured for abstractive summarization")
	}

	// Build prompt for summarization
	prompt := fmt.Sprintf(`Summarize the following text in approximately %d tokens.
Focus on key information and maintain technical accuracy.
Be concise and preserve important details.

Text:
%s

Summary:`, targetTokens, text)

	// Create request
	req := llm.CompletionRequest{
		Model:       s.modelName,
		Prompt:      prompt,
		MaxTokens:   targetTokens,
		Temperature: 0.3, // Lower temperature for more focused summarization
	}

	// Call LLM
	resp, err := s.llmClient.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return resp.Content, nil
}

// hybridSummarize implements hybrid summarization
// First uses extractive to reduce by 50%, then abstractive to reach target
func (s *Summarizer) hybridSummarize(ctx context.Context, text string, targetTokens int) (string, error) {
	currentTokens := s.counter.Count(text)

	// Step 1: Extractive to reduce by ~50%
	intermediateTokens := int(math.Max(float64(currentTokens)/2, float64(targetTokens)))
	extracted := s.extractiveSummarize(text, intermediateTokens)

	// Step 2: If we're close enough, return extractive result
	extractedTokens := s.counter.Count(extracted)
	if extractedTokens <= targetTokens {
		return extracted, nil
	}

	// Step 3: Use abstractive to reach target
	return s.abstractiveSummarize(ctx, extracted, targetTokens)
}

// SummarizeIfNeeded only summarizes if text exceeds target tokens
func (s *Summarizer) SummarizeIfNeeded(ctx context.Context, text string, targetTokens int) (string, error) {
	tokens := s.counter.Count(text)
	if tokens <= targetTokens {
		return text, nil
	}
	return s.Summarize(ctx, text, targetTokens)
}
