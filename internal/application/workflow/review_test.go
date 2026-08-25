package workflow

import (
	"context"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// TestPhaseExecutor_PostCompletionReview verifies the review loop approves and retries correctly.
func TestPhaseExecutor_PostCompletionReview(t *testing.T) {
	t.Run("approved_on_first_review", func(t *testing.T) {
		callCount := 0
		provider := newMockProvider()
		provider.completeFunc = func(_ context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
			callCount++
			if callCount == 1 {
				return &ports.CompletionResponse{Content: "The answer is 42.", InputTokens: 5, OutputTokens: 5}, nil
			}
			// Review call: approve
			return &ports.CompletionResponse{Content: "APPROVED", InputTokens: 5, OutputTokens: 5}, nil
		}

		pe := &phaseExecutor{provider: provider}
		phase := skill.Phase{
			ID: "p1", Name: "P1", PromptTemplate: "hello",
			MaxTokens: 100, PostCompletionReview: true,
		}
		result := pe.Execute(context.Background(), &phase, map[string]string{"_input": "test"})
		if result.Status != PhaseStatusCompleted {
			t.Fatalf("expected completed, got %s", result.Status)
		}
		if result.Output != "The answer is 42." {
			t.Errorf("unexpected output: %q", result.Output)
		}
		if callCount != 2 {
			t.Errorf("expected 2 provider calls (generate + review), got %d", callCount)
		}
	})

	t.Run("rejected_then_corrected", func(t *testing.T) {
		callCount := 0
		provider := newMockProvider()
		provider.completeFunc = func(_ context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
			callCount++
			switch callCount {
			case 1:
				return &ports.CompletionResponse{Content: "Wrong answer", InputTokens: 5, OutputTokens: 5}, nil
			case 2:
				// First review: reject and provide correction
				return &ports.CompletionResponse{Content: "REJECTED\nCorrected answer", InputTokens: 5, OutputTokens: 5}, nil
			default:
				// Second review: approve
				return &ports.CompletionResponse{Content: "APPROVED", InputTokens: 5, OutputTokens: 5}, nil
			}
		}

		pe := &phaseExecutor{provider: provider}
		phase := skill.Phase{
			ID: "p1", Name: "P1", PromptTemplate: "hello",
			MaxTokens: 100, PostCompletionReview: true,
		}
		result := pe.Execute(context.Background(), &phase, map[string]string{"_input": "test"})
		if result.Status != PhaseStatusCompleted {
			t.Fatalf("expected completed, got %s", result.Status)
		}
		if result.Output != "Corrected answer" {
			t.Errorf("unexpected output: %q", result.Output)
		}
	})

	t.Run("skip_review_flag", func(t *testing.T) {
		callCount := 0
		provider := newMockProvider()
		provider.completeFunc = func(_ context.Context, _ ports.CompletionRequest) (*ports.CompletionResponse, error) {
			callCount++
			return &ports.CompletionResponse{Content: "content", InputTokens: 5, OutputTokens: 5}, nil
		}

		pe := &phaseExecutor{provider: provider, skipPostCompletionReview: true}
		phase := skill.Phase{
			ID: "p1", Name: "P1", PromptTemplate: "hello",
			MaxTokens: 100, PostCompletionReview: true,
		}
		result := pe.Execute(context.Background(), &phase, map[string]string{"_input": "test"})
		if result.Status != PhaseStatusCompleted {
			t.Fatalf("expected completed, got %s", result.Status)
		}
		// Only 1 call — review was skipped
		if callCount != 1 {
			t.Errorf("expected 1 provider call (skip_review), got %d", callCount)
		}
	})
}
