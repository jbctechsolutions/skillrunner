package context

import (
	"regexp"
	"strings"
)

// Summarizer handles summarizing content to fit within token limits.
// It provides methods to reduce content length while preserving important information.
// Summarization strategies include truncation, key sentence extraction, and key point extraction.
type Summarizer struct{}

// NewSummarizer creates a new Summarizer instance.
func NewSummarizer() *Summarizer {
	return &Summarizer{}
}

// Summarize reduces content to fit within the specified maximum length
// It attempts to preserve important information by keeping the beginning
// and extracting key sentences
func (s *Summarizer) Summarize(content string, maxLength int) string {
	if content == "" || maxLength <= 0 {
		return ""
	}

	if len(content) <= maxLength {
		return content
	}

	// Strategy: Take first part + extract key sentences from the rest
	firstPartLength := maxLength / 2
	if firstPartLength > len(content) {
		firstPartLength = len(content)
	}

	firstPart := content[:firstPartLength]
	remaining := content[firstPartLength:]

	// Extract key sentences from remaining content
	keySentences := s.extractKeySentences(remaining, maxLength-firstPartLength)

	summary := firstPart
	if keySentences != "" {
		summary += " " + keySentences
	}

	// If still too long, truncate to sentence boundary
	if len(summary) > maxLength {
		summary = s.TruncateToSentence(summary, maxLength)
	}

	return summary
}

// SummarizeChunk creates a summary of a chunk.
// Returns a new Chunk with summarized content, preserving metadata (FilePath, Index, Start, End).
// Returns nil if chunk is nil.
func (s *Summarizer) SummarizeChunk(chunk *Chunk, maxLength int) *Chunk {
	if chunk == nil {
		return nil
	}

	summaryContent := s.Summarize(chunk.Content, maxLength)

	return &Chunk{
		Content:  summaryContent,
		FilePath: chunk.FilePath,
		Index:    chunk.Index,
		Start:    chunk.Start,
		End:      chunk.End,
	}
}

// SummarizeChunks creates summaries for multiple chunks.
// This is a convenience method that summarizes all chunks.
// Returns an empty slice if chunks is empty or nil.
func (s *Summarizer) SummarizeChunks(chunks []*Chunk, maxLength int) []*Chunk {
	if len(chunks) == 0 {
		return []*Chunk{}
	}

	summaries := make([]*Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		summary := s.SummarizeChunk(chunk, maxLength)
		if summary != nil {
			summaries = append(summaries, summary)
		}
	}

	return summaries
}

// ExtractKeyPoints extracts the most important points from content.
// The strategy includes the first sentence, last sentence, and evenly distributed middle sentences.
// Returns at most maxPoints key points.
// Returns an empty slice if content is empty or maxPoints is <= 0.
func (s *Summarizer) ExtractKeyPoints(content string, maxPoints int) []string {
	if content == "" || maxPoints <= 0 {
		return []string{}
	}

	// Split into sentences
	sentences := s.splitSentences(content)

	if len(sentences) == 0 {
		return []string{}
	}

	// If we have fewer sentences than max points, return all
	if len(sentences) <= maxPoints {
		return sentences
	}

	// Strategy: Take first sentence, last sentence, and evenly distributed middle sentences
	keyPoints := make([]string, 0, maxPoints)

	// Always include first sentence
	keyPoints = append(keyPoints, strings.TrimSpace(sentences[0]))

	if maxPoints > 1 {
		// Include last sentence
		keyPoints = append(keyPoints, strings.TrimSpace(sentences[len(sentences)-1]))

		// Distribute remaining points across middle sentences
		if maxPoints > 2 {
			step := (len(sentences) - 2) / (maxPoints - 2)
			if step < 1 {
				step = 1
			}

			for i := step; i < len(sentences)-1 && len(keyPoints) < maxPoints; i += step {
				trimmed := strings.TrimSpace(sentences[i])
				if trimmed != "" {
					keyPoints = append(keyPoints, trimmed)
				}
			}
		}
	}

	// Trim to max points if we exceeded
	if len(keyPoints) > maxPoints {
		keyPoints = keyPoints[:maxPoints]
	}

	return keyPoints
}

// TruncateToSentence truncates content to the nearest sentence boundary.
// Sentence boundaries are identified by periods, exclamation marks, or question marks.
// If no sentence boundary is found, truncates at the nearest word boundary.
// Adds ellipsis (...) if content is truncated and doesn't end with punctuation.
// Returns an empty string if content is empty or maxLength is <= 0.
func (s *Summarizer) TruncateToSentence(content string, maxLength int) string {
	if content == "" || maxLength <= 0 {
		return ""
	}

	if len(content) <= maxLength {
		return content
	}

	// Find the last sentence ending before maxLength
	// Look for sentence endings followed by space or end of string
	sentenceEndRegex := regexp.MustCompile(`[.!?]+(\s+|$)`)
	matches := sentenceEndRegex.FindAllStringIndex(content[:maxLength], -1)

	if len(matches) > 0 {
		// Use the last sentence boundary found
		lastMatch := matches[len(matches)-1]
		truncated := content[:lastMatch[1]]
		// Trim trailing whitespace
		return strings.TrimSpace(truncated)
	}

	// If no sentence boundary found, truncate at word boundary
	truncated := content[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 {
		result := strings.TrimSpace(truncated[:lastSpace])
		if !strings.HasSuffix(result, ".") && !strings.HasSuffix(result, "!") && !strings.HasSuffix(result, "?") {
			result += "..."
		}
		return result
	}

	// If no good word boundary, just truncate and add ellipsis
	result := strings.TrimSpace(truncated)
	if !strings.HasSuffix(result, ".") && !strings.HasSuffix(result, "!") && !strings.HasSuffix(result, "?") {
		result += "..."
	}
	return result
}

// extractKeySentences extracts important sentences from content
func (s *Summarizer) extractKeySentences(content string, maxLength int) string {
	if content == "" || maxLength <= 0 {
		return ""
	}

	sentences := s.splitSentences(content)
	if len(sentences) == 0 {
		return ""
	}

	// Take first few sentences that fit
	var result strings.Builder
	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if trimmed == "" {
			continue
		}

		// Check if adding this sentence would exceed maxLength
		candidate := result.String()
		if candidate != "" {
			candidate += " "
		}
		candidate += trimmed

		if len(candidate) > maxLength {
			break
		}

		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString(trimmed)
	}

	return result.String()
}

// splitSentences splits content into sentences
func (s *Summarizer) splitSentences(content string) []string {
	// Split by sentence endings (. ! ? followed by space or end of string)
	sentenceRegex := regexp.MustCompile(`([.!?]+\s+|$)`)

	var sentences []string
	lastIndex := 0

	matches := sentenceRegex.FindAllStringIndex(content, -1)
	for _, match := range matches {
		sentence := content[lastIndex:match[1]]
		trimmed := strings.TrimSpace(sentence)
		if trimmed != "" {
			sentences = append(sentences, trimmed)
		}
		lastIndex = match[1]
	}

	// If no sentence endings found, return the whole content as one sentence
	if len(sentences) == 0 {
		trimmed := strings.TrimSpace(content)
		if trimmed != "" {
			sentences = append(sentences, trimmed)
		}
	}

	return sentences
}
