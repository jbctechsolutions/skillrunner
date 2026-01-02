// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"strings"
	"text/template"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// PhaseStreamCallback is called for each chunk of streamed content.
type PhaseStreamCallback func(chunk string, inputTokens, outputTokens int) error

// streamingPhaseExecutor handles the execution of a single phase with streaming support.
type streamingPhaseExecutor struct {
	provider ports.ProviderPort
}

// newStreamingPhaseExecutor creates a new streaming phase executor.
func newStreamingPhaseExecutor(provider ports.ProviderPort) *streamingPhaseExecutor {
	return &streamingPhaseExecutor{
		provider: provider,
	}
}

// ExecuteWithStreaming runs a single phase with streaming output.
func (e *streamingPhaseExecutor) ExecuteWithStreaming(
	ctx context.Context,
	phase *skill.Phase,
	dependencyOutputs map[string]string,
	callback PhaseStreamCallback,
) *PhaseResult {
	result := &PhaseResult{
		PhaseID:   phase.ID,
		PhaseName: phase.Name,
		Status:    PhaseStatusRunning,
		StartTime: time.Now(),
	}

	// Build the prompt from the template
	prompt, err := e.buildPrompt(phase.PromptTemplate, dependencyOutputs)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Build the completion request
	req := ports.CompletionRequest{
		ModelID:     e.selectModel(phase.RoutingProfile),
		Messages:    e.buildMessages(prompt, dependencyOutputs),
		MaxTokens:   phase.MaxTokens,
		Temperature: phase.Temperature,
	}

	// Accumulate the full content for the result
	var fullContent strings.Builder
	var lastInputTokens int

	// Create streaming callback
	streamCallback := func(chunk string) error {
		fullContent.WriteString(chunk)
		if callback != nil {
			// For now, we estimate output tokens based on accumulated content
			// The actual token counts come at the end of the stream
			return callback(chunk, lastInputTokens, fullContent.Len()/4) // rough estimate
		}
		return nil
	}

	// Call the provider with streaming
	resp, err := e.provider.Stream(ctx, req, streamCallback)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Use the response content (which should match accumulated content)
	result.Status = PhaseStatusCompleted
	result.Output = resp.Content
	result.InputTokens = resp.InputTokens
	result.OutputTokens = resp.OutputTokens
	result.ModelUsed = resp.ModelUsed
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Final callback with accurate token counts
	if callback != nil {
		_ = callback("", resp.InputTokens, resp.OutputTokens)
	}

	return result
}

// Execute runs a single phase without streaming (for compatibility).
func (e *streamingPhaseExecutor) Execute(ctx context.Context, phase *skill.Phase, dependencyOutputs map[string]string) *PhaseResult {
	return e.ExecuteWithStreaming(ctx, phase, dependencyOutputs, nil)
}

// buildPrompt renders the phase's prompt template with the dependency outputs.
func (e *streamingPhaseExecutor) buildPrompt(templateStr string, data map[string]string) (string, error) {
	templateData := make(map[string]any, len(data))
	for k, v := range data {
		templateData[k] = v
	}

	funcMap := template.FuncMap{
		"get": func(key string) string {
			if v, ok := data[key]; ok {
				return v
			}
			return ""
		},
	}

	tmpl, err := template.New("prompt").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// buildMessages constructs the message array for the LLM request.
func (e *streamingPhaseExecutor) buildMessages(prompt string, dependencyOutputs map[string]string) []ports.Message {
	messages := make([]ports.Message, 0, 2)

	if len(dependencyOutputs) > 0 {
		var contextParts []string

		if input, ok := dependencyOutputs["_input"]; ok && input != "" {
			contextParts = append(contextParts, "Original Input:\n"+input)
		}

		for id, output := range dependencyOutputs {
			if id != "_input" && output != "" {
				contextParts = append(contextParts, "Previous Phase ("+id+"):\n"+output)
			}
		}

		if len(contextParts) > 0 {
			contextMsg := strings.Join(contextParts, "\n\n---\n\n")
			messages = append(messages, ports.Message{
				Role:    "system",
				Content: "Context from previous phases:\n\n" + contextMsg,
			})
		}
	}

	messages = append(messages, ports.Message{
		Role:    "user",
		Content: prompt,
	})

	return messages
}

// selectModel returns a model ID based on the routing profile.
func (e *streamingPhaseExecutor) selectModel(routingProfile string) string {
	switch routingProfile {
	case skill.RoutingProfileCheap:
		return "cheap-model"
	case skill.RoutingProfilePremium:
		return "premium-model"
	case skill.RoutingProfileBalanced:
		fallthrough
	default:
		return "balanced-model"
	}
}
