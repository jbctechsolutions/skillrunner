// Package complexity provides domain types for request complexity scoring.
// Scores range from 0.0 (trivial) to 1.0 (expert-level) and map to routing profiles.
package complexity

import "github.com/jbctechsolutions/skillrunner/internal/domain/skill"

// Thresholds separate the three routing tiers.
const (
	ThresholdCheapMax    = 0.40 // 0.00–0.40 → cheap
	ThresholdBalancedMax = 0.70 // 0.40–0.70 → balanced
	// above ThresholdBalancedMax → premium
)

// Score is a normalised complexity value in [0.0, 1.0].
type Score float64

// Profile maps the score to a routing profile name.
func (s Score) Profile() string {
	switch {
	case float64(s) < ThresholdCheapMax:
		return skill.ProfileCheap
	case float64(s) < ThresholdBalancedMax:
		return skill.ProfileBalanced
	default:
		return skill.ProfilePremium
	}
}

// Clamp returns a Score clamped to [0.0, 1.0].
func (s Score) Clamp() Score {
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

// Signals holds the individual signal values that contributed to a score.
type Signals struct {
	LengthScore  float64 // 0–1: normalised input length
	KeywordScore float64 // 0–1: technical keyword density
	CodeScore    float64 // 0–1: proportion of content that is code
	Final        Score   // weighted composite
}
