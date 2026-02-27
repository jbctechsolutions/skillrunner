package complexity_test

import (
	"strings"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/complexity"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

func TestScore_Profile(t *testing.T) {
	tests := []struct {
		score   complexity.Score
		profile string
	}{
		{0.00, skill.ProfileCheap},
		{0.20, skill.ProfileCheap},
		{0.39, skill.ProfileCheap},
		{0.40, skill.ProfileBalanced},
		{0.55, skill.ProfileBalanced},
		{0.69, skill.ProfileBalanced},
		{0.70, skill.ProfilePremium},
		{0.90, skill.ProfilePremium},
		{1.00, skill.ProfilePremium},
	}
	for _, tt := range tests {
		if got := tt.score.Profile(); got != tt.profile {
			t.Errorf("Score(%v).Profile() = %q, want %q", tt.score, got, tt.profile)
		}
	}
}

func TestScore_Clamp(t *testing.T) {
	if complexity.Score(-0.5).Clamp() != 0 {
		t.Error("negative score should clamp to 0")
	}
	if complexity.Score(1.5).Clamp() != 1 {
		t.Error("score >1 should clamp to 1")
	}
	if complexity.Score(0.5).Clamp() != 0.5 {
		t.Error("in-range score should be unchanged")
	}
}

func TestAnalyzer_EmptyInput(t *testing.T) {
	a := complexity.NewAnalyzer()
	score, sigs := a.Analyze("")
	if score != 0 {
		t.Errorf("empty input: score = %v, want 0", score)
	}
	if sigs.Final != 0 {
		t.Errorf("empty input: signals.Final = %v, want 0", sigs.Final)
	}
}

func TestAnalyzer_SimpleRequest(t *testing.T) {
	a := complexity.NewAnalyzer()
	input := "What is 2 + 2?"
	score, _ := a.Analyze(input)
	if score.Profile() != skill.ProfileCheap {
		t.Errorf("trivial question should map to cheap, got %q (score %.2f)", score.Profile(), score)
	}
}

func TestAnalyzer_TechnicalRequest(t *testing.T) {
	a := complexity.NewAnalyzer()
	input := `Implement a distributed consensus algorithm using the Raft protocol.
The system must handle network partitions, leader election, log replication,
and membership changes. Consider race conditions, deadlocks, and memory barriers.
The implementation should be concurrent, secure against Byzantine faults,
and handle encryption of all inter-node communication using mTLS certificates.`
	score, _ := a.Analyze(input)
	if score.Profile() != skill.ProfilePremium {
		t.Errorf("highly technical request should map to premium, got %q (score %.2f)", score.Profile(), score)
	}
}

func TestAnalyzer_ModerateRequest(t *testing.T) {
	a := complexity.NewAnalyzer()
	input := `Write a function that reads a file, parses its JSON content,
and returns a structured response. Handle errors appropriately
and add basic validation for the input path.`
	score, _ := a.Analyze(input)
	// Should be cheap or balanced — definitely not premium
	if score.Profile() == skill.ProfilePremium {
		t.Errorf("moderate request should not map to premium, got %q (score %.2f)", score.Profile(), score)
	}
}

func TestAnalyzer_LongInputScoresHigher(t *testing.T) {
	a := complexity.NewAnalyzer()
	short := "fix typo"
	long := strings.Repeat("review this code carefully and suggest improvements ", 200)

	sShort, _ := a.Analyze(short)
	sLong, _ := a.Analyze(long)

	if sLong <= sShort {
		t.Errorf("long input (%.2f) should score higher than short (%.2f)", sLong, sShort)
	}
}

func TestAnalyzer_CodeHeavyInput(t *testing.T) {
	a := complexity.NewAnalyzer()
	// 80% fenced code
	codeInput := "Review this:\n```go\n" + strings.Repeat("    x := doSomething()\n", 40) + "```\n"
	noCodeInput := strings.Repeat("review this function for correctness ", 10)

	sCode, sigsCode := a.Analyze(codeInput)
	sNoCode, _ := a.Analyze(noCodeInput)

	if sigsCode.CodeScore == 0 {
		t.Error("code-heavy input should have non-zero code score")
	}
	_ = sCode
	_ = sNoCode
}

func TestAnalyzer_ScoreInRange(t *testing.T) {
	a := complexity.NewAnalyzer()
	inputs := []string{
		"hi",
		"What is the meaning of life?",
		"Implement distributed tracing with OpenTelemetry across 10 microservices",
		strings.Repeat("word ", 1000),
		"fix bug",
	}
	for _, inp := range inputs {
		score, _ := a.Analyze(inp)
		if score < 0 || score > 1 {
			t.Errorf("score %.3f out of [0,1] for input %q", score, inp[:min(len(inp), 30)])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
