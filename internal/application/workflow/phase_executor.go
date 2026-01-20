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

// phaseExecutor handles the execution of a single phase.
type phaseExecutor struct {
	provider      ports.ProviderPort
	memoryContent string
}

// newPhaseExecutor creates a new phase executor with the given provider and memory content.
func newPhaseExecutor(provider ports.ProviderPort, memoryContent string) *phaseExecutor {
	return &phaseExecutor{
		provider:      provider,
		memoryContent: memoryContent,
	}
}

// Execute runs a single phase with the given dependency outputs.
// It returns a PhaseResult containing the execution outcome.
func (e *phaseExecutor) Execute(ctx context.Context, phase *skill.Phase, dependencyOutputs map[string]string) *PhaseResult {
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

	// Call the provider
	resp, err := e.provider.Complete(ctx, req)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Populate the result
	result.Status = PhaseStatusCompleted
	result.Output = resp.Content
	result.InputTokens = resp.InputTokens
	result.OutputTokens = resp.OutputTokens
	result.ModelUsed = resp.ModelUsed
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// buildPrompt renders the phase's prompt template with the dependency outputs.
// The template can access values using {{.key}} syntax or {{index . "key-name"}} for keys with special chars.
// Phase outputs are also available via {{.phases.phaseid}} for better organization.
func (e *phaseExecutor) buildPrompt(templateStr string, data map[string]string) (string, error) {
	// Convert to a generic map for template rendering with nested structure
	templateData := make(map[string]any, len(data)+1)
	phases := make(map[string]string)

	for k, v := range data {
		templateData[k] = v
		// Add non-special keys to the phases map for nested access
		if !strings.HasPrefix(k, "_") {
			phases[k] = v
		}
	}

	// Add phases map for nested template access: {{.phases.phaseid}}
	if len(phases) > 0 {
		templateData["phases"] = phases
	}

	// Create template with custom function to access map values by key
	funcMap := template.FuncMap{
		"get": func(key string) string {
			if v, ok := data[key]; ok {
				return v
			}
			return ""
		},
	}

	// Parse and execute the template
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
func (e *phaseExecutor) buildMessages(prompt string, dependencyOutputs map[string]string) []ports.Message {
	messages := make([]ports.Message, 0, 3)

	// Add memory context first (if available) - highest priority
	if e.memoryContent != "" {
		messages = append(messages, ports.Message{
			Role:    "system",
			Content: "Project Memory:\n\n" + e.memoryContent,
		})
	}

	// Add context from dependencies if available
	if len(dependencyOutputs) > 0 {
		var contextParts []string

		// Add original input first
		if input, ok := dependencyOutputs["_input"]; ok && input != "" {
			contextParts = append(contextParts, "Original Input:\n"+input)
		}

		// Add outputs from dependencies
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

	// Add the main prompt as user message
	messages = append(messages, ports.Message{
		Role:    "user",
		Content: prompt,
	})

	return messages
}

// selectModel returns a model ID based on the routing profile.
// Maps routing profiles to actual Ollama model names.
func (e *phaseExecutor) selectModel(routingProfile string) string {
	switch routingProfile {
	case skill.RoutingProfileCheap:
		return "llama3.2:3b"
	case skill.RoutingProfilePremium:
		return "qwen2.5:14b"
	case skill.RoutingProfileBalanced:
		fallthrough
	default:
		return "llama3:8b"
	}
}
