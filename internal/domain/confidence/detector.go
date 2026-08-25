// Package confidence provides detection of low-confidence LLM responses
// and determines whether escalation to a higher-tier model is warranted.
package confidence

import (
	"math"
	"regexp"
	"strings"
)

// Score represents a confidence level in [0.0, 1.0].
// Lower scores indicate more uncertainty in the response.
type Score float64

// Default thresholds by routing profile.
const (
	DefaultThresholdCheap    = 0.70
	DefaultThresholdBalanced = 0.55
	DefaultThresholdPremium  = 0.40 // premium rarely escalates
)

// uncertaintyPhrases are hedging markers that indicate low confidence.
// Ordered roughly by severity (strongest first).
var uncertaintyPhrases = []struct {
	pattern string
	weight  float64
}{
	// Strong hedges
	{`\bi('m| am) not sure\b`, 0.35},
	{`\bi('m| am) unsure\b`, 0.35},
	{`\bi don't know\b`, 0.35},
	{`\bi cannot (confirm|verify|determine)\b`, 0.30},
	{`\bi('m| am) unable to (confirm|determine)\b`, 0.30},
	{`\bcannot (say|confirm|determine) (with|for)\b`, 0.30},
	// Moderate hedges
	{`\bapproximately\b`, 0.15},
	{`\bprobably\b`, 0.15},
	{`\blikely\b`, 0.10},
	{`\bit('s| is) possible\b`, 0.15},
	{`\bit('s| is) unclear\b`, 0.20},
	{`\bmight\b`, 0.08},
	{`\bmay or may not\b`, 0.20},
	{`\bcould be\b`, 0.08},
	{`\bnot certain\b`, 0.25},
	{`\buncertain\b`, 0.25},
	// Softer hedges
	{`\bas far as i know\b`, 0.12},
	{`\bto my knowledge\b`, 0.10},
	{`\bi believe\b`, 0.08},
	{`\bi think\b`, 0.06},
	{`\bif i recall\b`, 0.12},
	{`\bif memory serves\b`, 0.12},
}

// compiledPhrases are pre-compiled regexps for performance.
var compiledPhrases []struct {
	re     *regexp.Regexp
	weight float64
}

func init() {
	for _, p := range uncertaintyPhrases {
		compiledPhrases = append(compiledPhrases, struct {
			re     *regexp.Regexp
			weight float64
		}{
			re:     regexp.MustCompile(`(?i)` + p.pattern),
			weight: p.weight,
		})
	}
}

// Detect analyses text and returns a confidence Score.
// Score of 1.0 = fully confident; 0.0 = completely uncertain.
func Detect(text string) Score {
	if strings.TrimSpace(text) == "" {
		return 1.0 // empty is treated as neutral
	}

	penalty := 0.0
	for _, p := range compiledPhrases {
		matches := p.re.FindAllString(text, -1)
		if len(matches) > 0 {
			// Diminishing returns for repeated occurrences
			penalty += p.weight * math.Log(1+float64(len(matches)))
		}
	}

	score := 1.0 - penalty
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return Score(score)
}

// ThresholdForProfile returns the default confidence threshold for a routing profile.
func ThresholdForProfile(profile string) float64 {
	switch profile {
	case "cheap":
		return DefaultThresholdCheap
	case "balanced":
		return DefaultThresholdBalanced
	case "premium":
		return DefaultThresholdPremium
	default:
		return DefaultThresholdBalanced
	}
}

// NeedsEscalation reports whether a response with the given score and profile
// should be retried with a higher-tier model.
func NeedsEscalation(score Score, profile string, threshold float64) bool {
	if threshold <= 0 {
		threshold = ThresholdForProfile(profile)
	}
	return float64(score) < threshold
}

// EscalateProfile returns the next-tier profile for escalation.
// premium has no higher tier and returns itself.
func EscalateProfile(profile string) string {
	switch profile {
	case "cheap":
		return "balanced"
	case "balanced":
		return "premium"
	default:
		return profile // premium stays premium
	}
}
