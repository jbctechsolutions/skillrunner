package context

import (
	"regexp"
	"strings"
)

// ChunkStrategy defines how content should be chunked.
// Available strategies:
//   - ChunkStrategySize: Split by character count
//   - ChunkStrategyLines: Split by number of lines
//   - ChunkStrategyParagraphs: Split by number of paragraphs
//   - ChunkStrategySentences: Split by number of sentences
const (
	ChunkStrategySize       = "size"       // Chunk by character count
	ChunkStrategyLines      = "lines"      // Chunk by line count
	ChunkStrategyParagraphs = "paragraphs" // Chunk by paragraph count
	ChunkStrategySentences  = "sentences"  // Chunk by sentence count
)

// Chunk represents a piece of chunked content with metadata.
// It contains the chunk content, source file path, index, and position information.
type Chunk struct {
	Content  string `json:"content"`             // The chunk content
	FilePath string `json:"file_path,omitempty"` // Source file path (if applicable)
	Index    int    `json:"index"`               // Zero-based index of this chunk
	Start    int    `json:"start"`               // Character position in original content
	End      int    `json:"end"`                 // Character position in original content
}

// Chunker handles chunking content into smaller pieces.
// It supports multiple chunking strategies: by size, lines, paragraphs, or sentences.
// Chunking is useful for processing large files that exceed token limits.
type Chunker struct{}

// NewChunker creates a new Chunker instance.
func NewChunker() *Chunker {
	return &Chunker{}
}

// ChunkBySize splits content into chunks of approximately the given size.
// Each chunk will be at most chunkSize characters long.
// If content is smaller than chunkSize, returns a single chunk.
// Returns an empty slice if content is empty or chunkSize is <= 0.
func (c *Chunker) ChunkBySize(content string, chunkSize int) []string {
	if content == "" || chunkSize <= 0 {
		return []string{}
	}

	if len(content) <= chunkSize {
		return []string{content}
	}

	var chunks []string
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunks = append(chunks, content[i:end])
	}

	return chunks
}

// ChunkByLines splits content into chunks with approximately the given number of lines.
// Each chunk will contain at most linesPerChunk lines.
// If content has fewer lines than linesPerChunk, returns a single chunk.
// Returns an empty slice if content is empty or linesPerChunk is <= 0.
func (c *Chunker) ChunkByLines(content string, linesPerChunk int) []string {
	if content == "" || linesPerChunk <= 0 {
		return []string{}
	}

	lines := strings.Split(content, "\n")
	if len(lines) <= linesPerChunk {
		return []string{content}
	}

	var chunks []string
	for i := 0; i < len(lines); i += linesPerChunk {
		end := i + linesPerChunk
		if end > len(lines) {
			end = len(lines)
		}
		chunkLines := lines[i:end]
		chunks = append(chunks, strings.Join(chunkLines, "\n"))
	}

	return chunks
}

// ChunkBySentences splits content into chunks with approximately the given number of sentences.
// Sentences are identified by periods, exclamation marks, or question marks followed by whitespace.
// Each chunk will contain at most sentencesPerChunk sentences.
// If content has fewer sentences than sentencesPerChunk, returns a single chunk.
// Returns an empty slice if content is empty or sentencesPerChunk is <= 0.
func (c *Chunker) ChunkBySentences(content string, sentencesPerChunk int) []string {
	if content == "" || sentencesPerChunk <= 0 {
		return []string{}
	}

	// Split by sentence endings (. ! ? followed by space or end of string)
	sentenceRegex := regexp.MustCompile(`([.!?]+\s+|$)`)
	matches := sentenceRegex.FindAllStringIndex(content, -1)

	if len(matches) <= sentencesPerChunk {
		return []string{content}
	}

	var chunks []string
	var lastEnd int

	for i := 0; i < len(matches); i += sentencesPerChunk {
		endIdx := i + sentencesPerChunk
		if endIdx > len(matches) {
			endIdx = len(matches)
		}

		var end int
		if endIdx < len(matches) {
			end = matches[endIdx-1][1]
		} else {
			end = len(content)
		}

		chunks = append(chunks, content[lastEnd:end])
		lastEnd = end
	}

	return chunks
}

// ChunkByParagraphs splits content into chunks with approximately the given number of paragraphs.
// Paragraphs are identified by double newlines (\n\n).
// Empty paragraphs (whitespace only) are filtered out.
// Each chunk will contain at most paragraphsPerChunk paragraphs.
// If content has fewer paragraphs than paragraphsPerChunk, returns a single chunk.
// Returns an empty slice if content is empty or paragraphsPerChunk is <= 0.
func (c *Chunker) ChunkByParagraphs(content string, paragraphsPerChunk int) []string {
	if content == "" || paragraphsPerChunk <= 0 {
		return []string{}
	}

	// Split by double newlines (paragraph breaks)
	paragraphs := strings.Split(content, "\n\n")

	// Filter out empty paragraphs
	var nonEmptyParagraphs []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			nonEmptyParagraphs = append(nonEmptyParagraphs, trimmed)
		}
	}

	if len(nonEmptyParagraphs) <= paragraphsPerChunk {
		return []string{content}
	}

	var chunks []string
	for i := 0; i < len(nonEmptyParagraphs); i += paragraphsPerChunk {
		end := i + paragraphsPerChunk
		if end > len(nonEmptyParagraphs) {
			end = len(nonEmptyParagraphs)
		}
		chunkParagraphs := nonEmptyParagraphs[i:end]
		chunks = append(chunks, strings.Join(chunkParagraphs, "\n\n"))
	}

	return chunks
}

// ChunkWithOverlap splits content into chunks with overlap between consecutive chunks.
// This is useful for preserving context at chunk boundaries.
// Each chunk will be approximately chunkSize characters long.
// Consecutive chunks will overlap by overlapSize characters.
// If overlapSize >= chunkSize, it defaults to chunkSize/2.
// Returns an empty slice if content is empty or chunkSize is <= 0.
func (c *Chunker) ChunkWithOverlap(content string, chunkSize int, overlapSize int) []string {
	if content == "" || chunkSize <= 0 {
		return []string{}
	}

	if len(content) <= chunkSize {
		return []string{content}
	}

	if overlapSize >= chunkSize {
		overlapSize = chunkSize / 2 // Default to half overlap if invalid
	}

	var chunks []string
	step := chunkSize - overlapSize

	for i := 0; i < len(content); i += step {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunks = append(chunks, content[i:end])

		// If we've reached the end, break
		if end >= len(content) {
			break
		}
	}

	return chunks
}

// ChunkFile chunks a file using the specified strategy.
// Strategy can be one of: ChunkStrategySize, ChunkStrategyLines, ChunkStrategyParagraphs, ChunkStrategySentences.
// If strategy is invalid or empty, defaults to ChunkStrategySize.
// chunkSize specifies the size/number for the chunking strategy.
// overlapSize is only used with ChunkStrategySize and must be > 0 to enable overlap.
// Returns an empty slice if file is nil.
func (c *Chunker) ChunkFile(file *File, strategy string, chunkSize int, overlapSize int) []*Chunk {
	if file == nil {
		return []*Chunk{}
	}

	var contentChunks []string

	switch strategy {
	case ChunkStrategyLines:
		contentChunks = c.ChunkByLines(file.Content, chunkSize)
	case ChunkStrategyParagraphs:
		contentChunks = c.ChunkByParagraphs(file.Content, chunkSize)
	case ChunkStrategySentences:
		contentChunks = c.ChunkBySentences(file.Content, chunkSize)
	case ChunkStrategySize:
		fallthrough
	default:
		if overlapSize > 0 {
			contentChunks = c.ChunkWithOverlap(file.Content, chunkSize, overlapSize)
		} else {
			contentChunks = c.ChunkBySize(file.Content, chunkSize)
		}
	}

	// Convert to Chunk structs with metadata
	chunks := make([]*Chunk, 0, len(contentChunks))
	var offset int

	for i, content := range contentChunks {
		chunk := &Chunk{
			Content:  content,
			FilePath: file.Path,
			Index:    i,
			Start:    offset,
			End:      offset + len(content),
		}
		chunks = append(chunks, chunk)
		offset += len(content)

		// Adjust for overlap if using overlap strategy
		if overlapSize > 0 && strategy == ChunkStrategySize && i < len(contentChunks)-1 {
			offset -= overlapSize
		}
	}

	return chunks
}

// ChunkFiles chunks multiple files using the specified strategy.
// This is a convenience method that chunks all files and combines the results.
// Strategy and parameters are the same as ChunkFile.
// Returns an empty slice if files is empty or nil.
func (c *Chunker) ChunkFiles(files []*File, strategy string, chunkSize int, overlapSize int) []*Chunk {
	var allChunks []*Chunk

	for _, file := range files {
		chunks := c.ChunkFile(file, strategy, chunkSize, overlapSize)
		allChunks = append(allChunks, chunks...)
	}

	return allChunks
}
