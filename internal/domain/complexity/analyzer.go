package complexity

import (
	"math"
	"strings"
)

// technicalKeywords are terms associated with complex, expert-level tasks.
// Higher density → higher complexity score.
var technicalKeywords = []string{
	// Systems / concurrency
	"distributed", "concurrency", "race condition", "deadlock", "mutex",
	"goroutine", "channel", "semaphore", "atomic", "memory barrier",
	// Security
	"vulnerability", "exploit", "injection", "xss", "csrf", "authentication",
	"authorization", "encryption", "cryptography", "certificate", "oauth",
	// Architecture
	"microservice", "event-driven", "saga", "cqrs", "circuit breaker",
	"backpressure", "sharding", "partitioning", "consensus", "raft",
	// Performance
	"latency", "throughput", "bottleneck", "profiling", "optimization",
	"cache invalidation", "indexing", "query plan", "execution plan",
	// Algorithms
	"dynamic programming", "graph traversal", "dijkstra", "complexity", "big-o",
	"recursion", "memoization", "topological", "binary search",
	// Async / networking
	"async", "await", "callback", "promise", "websocket", "grpc",
	"protocol buffer", "serialization", "deserialization",
}

// weights control how much each signal contributes to the final score.
// Keyword dominates: 3+ distinct technical keywords alone push a request
// into premium territory, regardless of length.
const (
	weightKeyword = 0.75
	weightLength  = 0.20
	weightCode    = 0.05

	// keywordCountForMax: this many distinct technical keywords → keyword score 1.0
	keywordCountForMax = 3.0
)

// Analyzer scores a text input for complexity.
type Analyzer struct{}

// NewAnalyzer creates an Analyzer with default configuration.
func NewAnalyzer() *Analyzer { return &Analyzer{} }

// Analyze scores the input and returns the composite Score plus per-signal breakdown.
func (a *Analyzer) Analyze(input string) (Score, Signals) {
	if strings.TrimSpace(input) == "" {
		return 0, Signals{}
	}

	ls := lengthScore(input)
	ks := keywordScore(input)
	cs := codeScore(input)

	final := Score(ls*weightLength + ks*weightKeyword + cs*weightCode).Clamp()

	return final, Signals{
		LengthScore:  ls,
		KeywordScore: ks,
		CodeScore:    cs,
		Final:        final,
	}
}

// lengthScore maps character count to [0, 1].
func lengthScore(input string) float64 {
	chars := float64(len(input))
	// Logarithmic scaling so a 500-char request isn't already 0.5.
	// ln(chars/512) / ln(32) normalises 512–16384 chars to roughly 0–1.
	if chars < 64 {
		return 0
	}
	raw := math.Log(chars/512) / math.Log(32)
	return clamp01(raw)
}

// keywordScore counts distinct technical keyword matches (count-based, not density).
// keywordCountForMax or more distinct matches → score 1.0.
func keywordScore(input string) float64 {
	lower := strings.ToLower(input)
	var hits int
	for _, kw := range technicalKeywords {
		if strings.Contains(lower, kw) {
			hits++
		}
	}
	return clamp01(float64(hits) / keywordCountForMax)
}

// codeScore estimates what fraction of the input is source code.
func codeScore(input string) float64 {
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return 0
	}

	// Count lines that look like code: significant indentation or fenced blocks.
	var codeLines, inFence int
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence ^= 1
			continue
		}
		if inFence == 1 {
			codeLines++
			continue
		}
		// Heuristic: leading whitespace of 4+ spaces or a tab (and non-empty)
		if len(line) > 0 && (line[0] == '\t' || strings.HasPrefix(line, "    ")) && len(trimmed) > 0 {
			codeLines++
		}
	}

	return clamp01(float64(codeLines) / float64(len(lines)))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
