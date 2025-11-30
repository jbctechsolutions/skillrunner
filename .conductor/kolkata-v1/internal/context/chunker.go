package context

import (
	"regexp"
	"strings"
)

// ChunkingStrategy defines the chunking approach
type ChunkingStrategy string

const (
	// SimpleChunking splits by token limit with overlap
	SimpleChunking ChunkingStrategy = "simple"
	// HierarchicalChunking preserves document structure
	HierarchicalChunking ChunkingStrategy = "hierarchical"
	// SemanticChunking groups by semantic similarity (future)
	SemanticChunking ChunkingStrategy = "semantic"
)

// Chunk represents a chunk of text with metadata
type Chunk struct {
	Content  string   // The actual text content
	Tokens   int      // Estimated token count
	Index    int      // Chunk index in the sequence
	Metadata Metadata // Additional metadata
}

// Metadata contains chunk metadata
type Metadata struct {
	Sections []string // Section titles (for hierarchical chunks)
	Start    int      // Start position in original text
	End      int      // End position in original text
}

// Chunker handles text chunking
type Chunker struct {
	counter  *TokenCounter
	strategy ChunkingStrategy
}

// NewChunker creates a new chunker
func NewChunker(strategy ChunkingStrategy) *Chunker {
	return &Chunker{
		counter:  NewTokenCounter(),
		strategy: strategy,
	}
}

// Chunk splits text into chunks based on the configured strategy
func (c *Chunker) Chunk(text string, maxTokens int) ([]Chunk, error) {
	switch c.strategy {
	case SimpleChunking:
		return c.simpleChunk(text, maxTokens, 100) // 100 token overlap
	case HierarchicalChunking:
		return c.hierarchicalChunk(text, maxTokens)
	case SemanticChunking:
		return c.semanticChunk(text, maxTokens)
	default:
		return c.simpleChunk(text, maxTokens, 100)
	}
}

// simpleChunk implements simple chunking with overlap
func (c *Chunker) simpleChunk(text string, maxTokens int, overlapTokens int) ([]Chunk, error) {
	if text == "" {
		return []Chunk{}, nil
	}

	// If text fits in one chunk, return it
	totalTokens := c.counter.Count(text)
	if totalTokens <= maxTokens {
		return []Chunk{
			{
				Content: text,
				Tokens:  totalTokens,
				Index:   0,
				Metadata: Metadata{
					Start: 0,
					End:   len(text),
				},
			},
		}, nil
	}

	// Split text into words for chunking
	words := strings.Fields(text)
	chunks := []Chunk{}
	chunkIndex := 0
	i := 0

	for i < len(words) {
		currentChunk := []string{}
		chunkText := ""
		startIdx := i

		// Add words until we hit the limit
		for i < len(words) {
			testChunk := append(currentChunk, words[i])
			testText := strings.Join(testChunk, " ")
			testTokens := c.counter.Count(testText)

			if testTokens > maxTokens && len(currentChunk) > 0 {
				// This word would exceed limit, don't add it
				break
			}

			// Add the word
			currentChunk = testChunk
			chunkText = testText
			i++
		}

		// If we couldn't add even one word, force add it
		// (handles case where single word exceeds limit)
		if len(currentChunk) == 0 && i < len(words) {
			currentChunk = []string{words[i]}
			chunkText = words[i]
			i++
		}

		// Save chunk
		if len(currentChunk) > 0 {
			chunks = append(chunks, Chunk{
				Content: chunkText,
				Tokens:  c.counter.Count(chunkText),
				Index:   chunkIndex,
				Metadata: Metadata{
					Start: 0,
					End:   0,
				},
			})
			chunkIndex++
		}

		// Calculate overlap for next chunk
		// Only if there are more words to process and we added more than 1 word
		if i < len(words) && overlapTokens > 0 && len(currentChunk) > 1 {
			// Calculate how many words to go back for overlap
			overlapWords := 0

			// Check from end of current chunk backwards
			for j := len(currentChunk) - 1; j >= 0; j-- {
				testOverlap := strings.Join(currentChunk[j:], " ")
				testSize := c.counter.Count(testOverlap)

				if testSize <= overlapTokens {
					overlapWords = len(currentChunk) - j
				} else {
					break
				}
			}

			// Move index back by overlap amount, but ensure we advance at least 1 word
			if overlapWords > 0 && overlapWords < len(currentChunk) {
				nextIdx := i - overlapWords
				// Ensure we're advancing
				if nextIdx > startIdx {
					i = nextIdx
				}
			}
		}
	}

	return chunks, nil
}

// hierarchicalChunk implements hierarchical chunking preserving structure
func (c *Chunker) hierarchicalChunk(text string, maxTokens int) ([]Chunk, error) {
	if text == "" {
		return []Chunk{}, nil
	}

	// If text fits in one chunk, return it
	totalTokens := c.counter.Count(text)
	if totalTokens <= maxTokens {
		return []Chunk{
			{
				Content: text,
				Tokens:  totalTokens,
				Index:   0,
				Metadata: Metadata{
					Start: 0,
					End:   len(text),
				},
			},
		}, nil
	}

	// Parse document structure (markdown sections)
	sections := c.parseMarkdownSections(text)

	chunks := []Chunk{}
	currentChunk := []Section{}
	currentTokens := 0
	chunkIndex := 0

	for _, section := range sections {
		sectionTokens := c.counter.Count(section.Content)

		// If a single section exceeds maxTokens, we need to split it
		if sectionTokens > maxTokens {
			// Flush current chunk if not empty
			if len(currentChunk) > 0 {
				chunks = append(chunks, c.buildChunkFromSections(currentChunk, chunkIndex))
				chunkIndex++
				currentChunk = []Section{}
				currentTokens = 0
			}

			// Split the large section into smaller chunks
			subChunks, _ := c.simpleChunk(section.Content, maxTokens, 100)
			for _, subChunk := range subChunks {
				chunks = append(chunks, Chunk{
					Content: subChunk.Content,
					Tokens:  subChunk.Tokens,
					Index:   chunkIndex,
					Metadata: Metadata{
						Sections: []string{section.Title},
					},
				})
				chunkIndex++
			}
			continue
		}

		// Check if adding this section exceeds the limit
		if currentTokens+sectionTokens > maxTokens && len(currentChunk) > 0 {
			// Create chunk from current sections
			chunks = append(chunks, c.buildChunkFromSections(currentChunk, chunkIndex))
			chunkIndex++
			currentChunk = []Section{}
			currentTokens = 0
		}

		// Add section to current chunk
		currentChunk = append(currentChunk, section)
		currentTokens += sectionTokens
	}

	// Add final chunk if it has content
	if len(currentChunk) > 0 {
		chunks = append(chunks, c.buildChunkFromSections(currentChunk, chunkIndex))
	}

	return chunks, nil
}

// Section represents a document section
type Section struct {
	Title   string
	Content string
	Level   int
}

// parseMarkdownSections parses markdown into sections
func (c *Chunker) parseMarkdownSections(text string) []Section {
	sections := []Section{}

	// Split by markdown headers (# ## ###)
	headerPattern := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	lines := strings.Split(text, "\n")

	currentSection := Section{
		Title:   "",
		Content: "",
		Level:   0,
	}

	for _, line := range lines {
		matches := headerPattern.FindStringSubmatch(line)
		if matches != nil {
			// This is a header line
			// Save previous section if it has content
			if currentSection.Content != "" {
				sections = append(sections, currentSection)
			}

			// Start new section
			level := len(matches[1])
			title := strings.TrimSpace(matches[2])
			currentSection = Section{
				Title:   title,
				Content: line + "\n",
				Level:   level,
			}
		} else {
			// Regular line, add to current section
			currentSection.Content += line + "\n"
		}
	}

	// Add final section
	if currentSection.Content != "" {
		sections = append(sections, currentSection)
	}

	// If no sections found, treat entire text as one section
	if len(sections) == 0 {
		sections = append(sections, Section{
			Title:   "",
			Content: text,
			Level:   0,
		})
	}

	return sections
}

// buildChunkFromSections builds a chunk from multiple sections
func (c *Chunker) buildChunkFromSections(sections []Section, index int) Chunk {
	content := ""
	sectionTitles := []string{}

	for _, section := range sections {
		content += section.Content
		if section.Title != "" {
			sectionTitles = append(sectionTitles, section.Title)
		}
	}

	tokens := c.counter.Count(content)

	return Chunk{
		Content: strings.TrimSpace(content),
		Tokens:  tokens,
		Index:   index,
		Metadata: Metadata{
			Sections: sectionTitles,
		},
	}
}

// semanticChunk implements semantic chunking by grouping sentences with similar topics
// Uses sentence-based similarity heuristics (keyword overlap, sentence structure)
func (c *Chunker) semanticChunk(text string, maxTokens int) ([]Chunk, error) {
	if text == "" {
		return []Chunk{}, nil
	}

	// If text fits in one chunk, return it
	totalTokens := c.counter.Count(text)
	if totalTokens <= maxTokens {
		return []Chunk{
			{
				Content: text,
				Tokens:  totalTokens,
				Index:   0,
				Metadata: Metadata{
					Start: 0,
					End:   len(text),
				},
			},
		}, nil
	}

	// Split into sentences
	sentences := c.splitSentences(text)
	if len(sentences) == 0 {
		return []Chunk{}, nil
	}

	// Extract keywords from each sentence (simple approach: common words)
	sentenceKeywords := make([][]string, len(sentences))
	for i, sent := range sentences {
		sentenceKeywords[i] = c.extractKeywords(sent)
	}

	// Group sentences by semantic similarity
	chunks := []Chunk{}
	chunkIndex := 0
	currentChunk := []string{}
	currentTokens := 0
	startPos := 0

	for i, sentence := range sentences {
		sentTokens := c.counter.Count(sentence)

		// If single sentence exceeds limit, create a chunk for it and continue
		if sentTokens > maxTokens {
			// Flush current chunk if not empty
			if len(currentChunk) > 0 {
				chunkText := strings.Join(currentChunk, " ")
				chunks = append(chunks, Chunk{
					Content: chunkText,
					Tokens:  currentTokens,
					Index:   chunkIndex,
					Metadata: Metadata{
						Start: startPos,
						End:   startPos + len(chunkText),
					},
				})
				chunkIndex++
				currentChunk = []string{}
				currentTokens = 0
			}

			// Split the large sentence using simple chunking
			subChunks, _ := c.simpleChunk(sentence, maxTokens, 50)
			for _, subChunk := range subChunks {
				chunks = append(chunks, Chunk{
					Content: subChunk.Content,
					Tokens:  subChunk.Tokens,
					Index:   chunkIndex,
					Metadata: Metadata{
						Start: startPos,
						End:   startPos + len(subChunk.Content),
					},
				})
				chunkIndex++
				startPos += len(subChunk.Content)
			}
			continue
		}

		// Check if adding this sentence would exceed the limit
		if currentTokens+sentTokens > maxTokens && len(currentChunk) > 0 {
			// Check semantic similarity with current chunk
			shouldGroup := c.isSemanticallySimilar(sentenceKeywords[i], currentChunk, sentenceKeywords)

			maxTokensFloat := float64(maxTokens)
			if !shouldGroup || float64(currentTokens+sentTokens) > maxTokensFloat*1.2 {
				// Create chunk from current sentences
				chunkText := strings.Join(currentChunk, " ")
				chunks = append(chunks, Chunk{
					Content: chunkText,
					Tokens:  currentTokens,
					Index:   chunkIndex,
					Metadata: Metadata{
						Start: startPos,
						End:   startPos + len(chunkText),
					},
				})
				chunkIndex++
				startPos += len(chunkText)
				currentChunk = []string{}
				currentTokens = 0
			}
		}

		// Add sentence to current chunk
		currentChunk = append(currentChunk, sentence)
		currentTokens += sentTokens
	}

	// Add final chunk if it has content
	if len(currentChunk) > 0 {
		chunkText := strings.Join(currentChunk, " ")
		chunks = append(chunks, Chunk{
			Content: chunkText,
			Tokens:  currentTokens,
			Index:   chunkIndex,
			Metadata: Metadata{
				Start: startPos,
				End:   startPos + len(chunkText),
			},
		})
	}

	return chunks, nil
}

// splitSentences splits text into sentences
func (c *Chunker) splitSentences(text string) []string {
	// Simple sentence splitting by common sentence endings
	re := regexp.MustCompile(`[.!?]+\s+`)
	parts := re.Split(text, -1)
	sentences := []string{}

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Add sentence ending back (except for last part)
		if i < len(parts)-1 {
			// Find the sentence ending
			matches := re.FindAllString(text, -1)
			if i < len(matches) {
				part += matches[i]
			}
		}

		sentences = append(sentences, part)
	}

	// If no sentence endings found, treat entire text as one sentence
	if len(sentences) == 0 {
		sentences = []string{text}
	}

	return sentences
}

// extractKeywords extracts important keywords from a sentence
func (c *Chunker) extractKeywords(sentence string) []string {
	// Remove punctuation and convert to lowercase
	re := regexp.MustCompile(`[^\w\s]`)
	cleaned := re.ReplaceAllString(strings.ToLower(sentence), " ")

	// Split into words
	words := strings.Fields(cleaned)

	// Filter out common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true,
		"has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true,
		"might": true, "must": true, "can": true, "this": true, "that": true,
		"these": true, "those": true, "it": true, "its": true, "they": true,
	}

	keywords := []string{}
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// isSemanticallySimilar checks if a sentence is semantically similar to current chunk
func (c *Chunker) isSemanticallySimilar(
	newKeywords []string,
	currentChunk []string,
	allKeywords [][]string,
) bool {
	if len(currentChunk) == 0 {
		return true
	}

	// Extract keywords from current chunk
	chunkText := strings.Join(currentChunk, " ")
	chunkKeywords := c.extractKeywords(chunkText)

	// Calculate keyword overlap
	overlap := c.calculateOverlap(newKeywords, chunkKeywords)

	// Consider similar if overlap > 20% or if chunk is small
	return overlap > 0.2 || len(currentChunk) < 3
}

// calculateOverlap calculates the overlap ratio between two keyword sets
func (c *Chunker) calculateOverlap(keywords1, keywords2 []string) float64 {
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	// Create sets
	set1 := make(map[string]bool)
	for _, k := range keywords1 {
		set1[k] = true
	}

	set2 := make(map[string]bool)
	for _, k := range keywords2 {
		set2[k] = true
	}

	// Count overlaps
	overlaps := 0
	for k := range set1 {
		if set2[k] {
			overlaps++
		}
	}

	// Return overlap ratio (average of both sets)
	total := len(set1) + len(set2)
	if total == 0 {
		return 0.0
	}

	return float64(overlaps*2) / float64(total)
}

// ChunkIfNeeded chunks text only if it exceeds the max tokens
func (c *Chunker) ChunkIfNeeded(text string, maxTokens int) ([]Chunk, error) {
	tokens := c.counter.Count(text)

	if tokens <= maxTokens {
		// No chunking needed
		return []Chunk{
			{
				Content: text,
				Tokens:  tokens,
				Index:   0,
			},
		}, nil
	}

	// Chunking needed
	return c.Chunk(text, maxTokens)
}
