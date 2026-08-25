// Package outcome provides domain types for tracking skill execution outcomes
// and generating routing profile recommendations based on history.
package outcome

import "time"

// Outcome records the result of a single phase execution.
type Outcome struct {
	ID           string    // Unique ID (UUID)
	SkillID      string    // Skill that was executed
	SkillName    string    // Human-readable skill name
	PhaseID      string    // Phase within the skill
	PhaseName    string    // Human-readable phase name
	Profile      string    // Routing profile used (cheap, balanced, premium)
	Model        string    // Specific model used
	Success      bool      // Whether the phase completed successfully
	QualityScore float64   // Optional quality score 0.0–1.0 (0 = unscored)
	RetryCount   int       // Number of retries before success/failure
	DurationMS   int64     // Execution duration in milliseconds
	RecordedAt   time.Time // When the outcome was recorded
}

// ProfileStats aggregates outcomes for a skill+phase+profile combination.
type ProfileStats struct {
	SkillID       string
	PhaseID       string
	PhaseName     string
	Profile       string
	Model         string
	TotalRuns     int64
	SuccessCount  int64
	SuccessRate   float64 // 0.0–1.0
	AvgQuality    float64 // Average quality score (0 = unscored)
	AvgRetries    float64 // Average retry count
	AvgDurationMS int64
}

// Recommendation suggests a routing profile change based on outcome history.
type Recommendation struct {
	PhaseID          string
	PhaseName        string
	CurrentProfile   string
	SuggestedProfile string
	Reason           string
	Confidence       float64 // 0.0–1.0 (higher = more data)
}

// Analyze inspects a set of ProfileStats and returns routing recommendations.
// It needs at least minSamples data points to make a recommendation.
func Analyze(stats []ProfileStats) []Recommendation {
	const minSamples = 5

	// Group by phaseID
	byPhase := make(map[string][]ProfileStats)
	for _, s := range stats {
		byPhase[s.PhaseID] = append(byPhase[s.PhaseID], s)
	}

	var recs []Recommendation
	for _, phaseStats := range byPhase {
		for _, s := range phaseStats {
			if s.TotalRuns < minSamples {
				continue
			}

			// Suggest upgrade: cheap with low success rate → balanced
			if s.Profile == "cheap" && s.SuccessRate < 0.80 {
				recs = append(recs, Recommendation{
					PhaseID:          s.PhaseID,
					PhaseName:        s.PhaseName,
					CurrentProfile:   "cheap",
					SuggestedProfile: "balanced",
					Reason:           reasonLowSuccess(s.SuccessRate),
					Confidence:       confidence(s.TotalRuns, minSamples),
				})
			}

			// Suggest downgrade: premium with perfect success → balanced (save cost)
			if s.Profile == "premium" && s.SuccessRate >= 0.95 && s.TotalRuns >= 20 {
				recs = append(recs, Recommendation{
					PhaseID:          s.PhaseID,
					PhaseName:        s.PhaseName,
					CurrentProfile:   "premium",
					SuggestedProfile: "balanced",
					Reason:           "Excellent success rate — balanced profile may suffice at lower cost",
					Confidence:       confidence(s.TotalRuns, minSamples),
				})
			}

			// Suggest downgrade: balanced with perfect success → cheap (save cost)
			if s.Profile == "balanced" && s.SuccessRate >= 0.95 && s.TotalRuns >= 20 {
				recs = append(recs, Recommendation{
					PhaseID:          s.PhaseID,
					PhaseName:        s.PhaseName,
					CurrentProfile:   "balanced",
					SuggestedProfile: "cheap",
					Reason:           "Excellent success rate — cheap profile may suffice at lowest cost",
					Confidence:       confidence(s.TotalRuns, minSamples),
				})
			}
		}
	}

	return recs
}

func reasonLowSuccess(rate float64) string {
	if rate < 0.60 {
		return "Very low success rate — upgrade to balanced profile recommended"
	}
	return "Below-target success rate — consider balanced profile for reliability"
}

func confidence(runs, minSamples int64) float64 {
	// Confidence grows from 0 at minSamples to 1.0 at 5×minSamples
	if runs <= minSamples {
		return 0.1
	}
	c := float64(runs-minSamples) / float64(4*minSamples)
	if c > 1.0 {
		return 1.0
	}
	return c
}
