package confidence_test

import (
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/confidence"
)

func TestDetect_HighConfidence(t *testing.T) {
	cases := []string{
		"The function should return the sum of both arguments.",
		"Use a mutex to protect concurrent map writes.",
		"The test passes because the mock returns the expected value.",
	}
	for _, text := range cases {
		s := confidence.Detect(text)
		if s < 0.85 {
			t.Errorf("expected high confidence for %q, got %.2f", text, s)
		}
	}
}

func TestDetect_LowConfidence(t *testing.T) {
	cases := []string{
		"I don't know the exact value, approximately 42.",
		"I'm unsure whether this is correct — it's possible but uncertain.",
		"I'm not sure; I cannot confirm this and I'm uncertain about the approach.",
	}
	for _, text := range cases {
		s := confidence.Detect(text)
		if s >= 0.70 {
			t.Errorf("expected low confidence (<0.70) for %q, got %.4f", text, s)
		}
	}
}

func TestDetect_EmptyInput(t *testing.T) {
	s := confidence.Detect("")
	if s != 1.0 {
		t.Errorf("empty input should score 1.0, got %.2f", s)
	}
}

func TestDetect_ScoreInRange(t *testing.T) {
	texts := []string{
		"Definitely use channel-based signalling.",
		"I think this could maybe work, probably.",
		"As far as I know, I believe this might be approximately right.",
	}
	for _, text := range texts {
		s := confidence.Detect(text)
		if s < 0 || s > 1 {
			t.Errorf("score %.2f out of [0,1] for %q", s, text)
		}
	}
}

func TestNeedsEscalation(t *testing.T) {
	// Below threshold → escalate
	if !confidence.NeedsEscalation(0.5, "cheap", 0) {
		t.Error("score 0.5 on cheap profile should need escalation (threshold 0.70)")
	}
	// Above threshold → no escalate
	if confidence.NeedsEscalation(0.9, "cheap", 0) {
		t.Error("score 0.9 on cheap profile should NOT need escalation")
	}
}

func TestEscalateProfile(t *testing.T) {
	cases := map[string]string{
		"cheap":    "balanced",
		"balanced": "premium",
		"premium":  "premium",
	}
	for from, want := range cases {
		got := confidence.EscalateProfile(from)
		if got != want {
			t.Errorf("EscalateProfile(%q) = %q, want %q", from, got, want)
		}
	}
}

func TestThresholdForProfile(t *testing.T) {
	if confidence.ThresholdForProfile("cheap") != confidence.DefaultThresholdCheap {
		t.Error("cheap profile should return DefaultThresholdCheap")
	}
	if confidence.ThresholdForProfile("premium") != confidence.DefaultThresholdPremium {
		t.Error("premium profile should return DefaultThresholdPremium")
	}
}
