// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	mcpAdapter "github.com/jbctechsolutions/skillrunner/internal/adapters/mcp"
	"github.com/jbctechsolutions/skillrunner/internal/application/compression"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// maxToolIterations caps the number of tool-calling turns per phase to avoid infinite loops.
const maxToolIterations = 10

// phaseExecutor handles the execution of a single phase.
type phaseExecutor struct {
	provider      ports.ProviderPort
	memoryContent string
	mcpRegistry   ports.MCPToolRegistryPort // nil when tool calling is disabled
	compressor    *compression.Compressor   // nil means no compression
	modelHints    map[string]string         // v1.3: profile→model overrides; nil = use defaults
}

// newPhaseExecutor creates a new phase executor with the given provider, memory content, and optional MCP registry.
func newPhaseExecutor(provider ports.ProviderPort, memoryContent string, mcpRegistry ports.MCPToolRegistryPort) *phaseExecutor {
	return &phaseExecutor{
		provider:      provider,
		memoryContent: memoryContent,
		mcpRegistry:   mcpRegistry,
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

	// Apply context compression if enabled
	if e.compressor != nil {
		cr := e.compressor.Compress(prompt)
		prompt = cr.Compressed
		result.CompressionRatio = cr.Ratio
	}

	// Fetch MCP tools if this phase allows tool calling
	var tools []ports.Tool
	if phase.AllowTools && e.mcpRegistry != nil {
		mcpTools, err := e.mcpRegistry.GetAllTools(ctx)
		if err == nil && len(mcpTools) > 0 {
			tools = mcpAdapter.ToProviderTools(mcpTools, false)
		}
	}

	// Build initial messages
	messages := e.buildMessages(prompt, dependencyOutputs)

	// Build the completion request
	req := ports.CompletionRequest{
		ModelID:     e.selectModel(ctx, phase.RoutingProfile),
		Messages:    messages,
		MaxTokens:   phase.MaxTokens,
		Temperature: phase.Temperature,
		Tools:       tools,
	}

	// Execute with tool calling loop
	var totalInput, totalOutput int
	var finalContent string
	var modelUsed string

	for i := 0; i < maxToolIterations; i++ {
		resp, err := e.provider.Complete(ctx, req)
		if err != nil {
			result.Status = PhaseStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		totalInput += resp.InputTokens
		totalOutput += resp.OutputTokens
		if modelUsed == "" {
			modelUsed = resp.ModelUsed
		}

		// If the model didn't call any tools, we're done
		if resp.FinishReason != ports.FinishReasonToolUse || len(resp.ToolCalls) == 0 {
			finalContent = resp.Content
			break
		}

		// Append the assistant's tool_use message to conversation
		req.Messages = append(req.Messages, ports.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool and collect results
		toolResults := make([]ports.ToolResult, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			toolResult := e.executeToolCall(ctx, tc)
			toolResults = append(toolResults, toolResult)
		}

		// Append the tool results as a user message
		req.Messages = append(req.Messages, ports.Message{
			Role:        "user",
			ToolResults: toolResults,
		})

		// If this is the last iteration, capture whatever was in last response
		if i == maxToolIterations-1 {
			finalContent = resp.Content
		}
	}

	// Populate the result
	result.Status = PhaseStatusCompleted
	result.Output = finalContent
	result.InputTokens = totalInput
	result.OutputTokens = totalOutput
	result.ModelUsed = modelUsed
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// executeToolCall executes a single tool call via the MCP registry and returns the result.
func (e *phaseExecutor) executeToolCall(ctx context.Context, tc ports.ToolCall) ports.ToolResult {
	if e.mcpRegistry == nil {
		return ports.ToolResult{
			ToolCallID: tc.ID,
			Content:    "error: MCP registry not available",
			IsError:    true,
		}
	}

	callResult, err := e.mcpRegistry.CallToolByFullName(ctx, tc.Name, tc.Arguments)
	if err != nil {
		return ports.ToolResult{
			ToolCallID: tc.ID,
			Content:    fmt.Sprintf("error: %v", err),
			IsError:    true,
		}
	}

	return ports.ToolResult{
		ToolCallID: tc.ID,
		Content:    callResult.TextContent(),
		IsError:    callResult.IsError,
	}
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
// If model hints are configured for this executor, the hinted model is used with
// automatic fallback to the default if the hint is not available.
func (e *phaseExecutor) selectModel(ctx context.Context, routingProfile string) string {
	defaultModel := e.defaultModel(routingProfile)

	// Check skill-level hint for this profile
	if len(e.modelHints) > 0 {
		if hinted, ok := e.modelHints[routingProfile]; ok && hinted != "" {
			// Verify the hinted model is actually available; fall back if not.
			if ok, err := e.provider.SupportsModel(ctx, hinted); err == nil && ok {
				return hinted
			}
		}
	}

	return defaultModel
}

// defaultModel returns the default model for a routing profile.
func (e *phaseExecutor) defaultModel(routingProfile string) string {
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
