// Package compression provides context compression strategies to reduce
// token counts before LLM provider calls.
package compression

import (
	"regexp"
	"strings"
)

// Result holds the compressed text and statistics.
type Result struct {
	Original   string
	Compressed string
	// Ratio is the fraction of characters removed (0 = nothing removed, 1 = everything removed).
	Ratio float64
}

// Mode controls how aggressively compression is applied.
type Mode int

const (
	// ModeOff disables compression entirely.
	ModeOff Mode = iota
	// ModeConservative targets <30% reduction; safe for all profiles.
	ModeConservative
	// ModeAggressive targets >70% reduction; suitable for the cheap routing profile.
	ModeAggressive
)

// Compressor applies a pipeline of compression strategies.
type Compressor struct {
	mode Mode
}

// New creates a Compressor for the given mode.
func New(mode Mode) *Compressor {
	return &Compressor{mode: mode}
}

// NewFromProfile returns a Compressor calibrated for the routing profile.
func NewFromProfile(profile string) *Compressor {
	switch profile {
	case "cheap":
		return New(ModeAggressive)
	case "premium":
		return New(ModeConservative)
	default:
		return New(ModeConservative)
	}
}

// Compress applies the configured pipeline to text and returns a Result.
func (c *Compressor) Compress(text string) Result {
	if c.mode == ModeOff || strings.TrimSpace(text) == "" {
		return Result{Original: text, Compressed: text, Ratio: 0}
	}

	out := text

	// Conservative strategies (always applied when not Off)
	out = normalizeWhitespace(out)
	out = deduplicateBlankLines(out)

	if c.mode == ModeAggressive {
		out = deduplicateRepeatedBlocks(out)
		out = stripLineComments(out)
	}

	ratio := 0.0
	if len(text) > 0 {
		ratio = 1.0 - float64(len(out))/float64(len(text))
	}
	if ratio < 0 {
		ratio = 0
	}

	return Result{Original: text, Compressed: out, Ratio: ratio}
}

// normalizeWhitespace collapses runs of spaces/tabs within each line to a single space.
func normalizeWhitespace(text string) string {
	lines := strings.Split(text, "\n")
	spaceRun := regexp.MustCompile(`[ \t]{2,}`)
	for i, line := range lines {
		// Preserve leading indentation; only collapse internal runs.
		trimmed := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimmed)]
		lines[i] = indent + spaceRun.ReplaceAllString(trimmed, " ")
	}
	return strings.Join(lines, "\n")
}

// deduplicateBlankLines collapses runs of 3+ blank lines into 2.
func deduplicateBlankLines(text string) string {
	threeBlank := regexp.MustCompile(`\n{4,}`) // 4+ newlines = 3+ blank lines
	return threeBlank.ReplaceAllString(text, "\n\n\n")
}

// deduplicateRepeatedBlocks removes exact-duplicate non-empty lines that appear
// more than twice in a row (aggressive mode only).
func deduplicateRepeatedBlocks(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	var prev string
	consecutive := 0

	for _, line := range lines {
		if line == prev && strings.TrimSpace(line) != "" {
			consecutive++
			if consecutive >= 2 {
				// Already emitted this line twice; skip further repetitions.
				continue
			}
		} else {
			consecutive = 0
		}
		out = append(out, line)
		prev = line
	}
	return strings.Join(out, "\n")
}

// stripLineComments removes single-line comments (// …) from non-blank lines.
// Only applied in aggressive mode and only when the line is otherwise code.
// This is deliberately conservative: it only strips pure comment lines
// (where the entire non-whitespace content is a comment), not inline comments.
func stripLineComments(text string) string {
	lines := strings.Split(text, "\n")
	pureComment := regexp.MustCompile(`^\s*(//|#)\s`)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if pureComment.MatchString(line) {
			continue // drop the comment line
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
