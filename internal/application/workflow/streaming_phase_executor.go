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
	"github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// PhaseStreamCallback is called for each chunk of streamed content.
type PhaseStreamCallback func(chunk string, inputTokens, outputTokens int) error

// streamingPhaseExecutor handles the execution of a single phase with streaming support.
type streamingPhaseExecutor struct {
	provider      ports.ProviderPort
	memoryContent string
	mcpRegistry   ports.MCPToolRegistryPort // nil when tool calling is disabled
	allowedTools  []string                  // skill-declared tool allowlist; empty = none
	compressor    *compression.Compressor   // nil means no compression
	modelHints    map[string]string         // v1.3: profile→model overrides; nil = use defaults
}

// newStreamingPhaseExecutor creates a new streaming phase executor.
func newStreamingPhaseExecutor(provider ports.ProviderPort, memoryContent string, mcpRegistry ports.MCPToolRegistryPort) *streamingPhaseExecutor {
	return &streamingPhaseExecutor{
		provider:      provider,
		memoryContent: memoryContent,
		mcpRegistry:   mcpRegistry,
	}
}

// ExecuteWithStreaming runs a single phase with streaming output.
// When MCP tool calling is needed, intermediate tool turns use non-streaming Complete(),
// and only the final response turn is streamed to the callback.
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

	// Apply context compression if enabled
	if e.compressor != nil {
		cr := e.compressor.Compress(prompt)
		prompt = cr.Compressed
		result.CompressionRatio = cr.Ratio
	}

	// Fetch MCP tools if this phase allows tool calling, filtered to the skill's declared allowlist.
	var tools []ports.Tool
	if phase.AllowTools && e.mcpRegistry != nil && len(e.allowedTools) > 0 {
		mcpTools, err := e.mcpRegistry.GetAllTools(ctx)
		if err == nil && len(mcpTools) > 0 {
			mcpTools = e.filterAllowedTools(mcpTools)
			if len(mcpTools) > 0 {
				tools = mcpAdapter.ToProviderTools(mcpTools, false)
			}
		}
	}

	messages := e.buildMessages(prompt, dependencyOutputs)

	req := ports.CompletionRequest{
		ModelID:     e.selectModel(ctx, phase.RoutingProfile),
		Messages:    messages,
		MaxTokens:   phase.MaxTokens,
		Temperature: phase.Temperature,
		Tools:       tools,
	}

	// If tool calling is not active for this phase, use the original streaming path.
	if !phase.AllowTools || e.mcpRegistry == nil || len(tools) == 0 {
		return e.executeStreaming(ctx, result, req, callback)
	}

	// Tool-calling path: use non-streaming Complete() for intermediate tool turns,
	// then stream the final response turn.
	var totalInput, totalOutput int
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

		// No tool calls: final response — replay via callback
		if resp.FinishReason != ports.FinishReasonToolUse || len(resp.ToolCalls) == 0 {
			if callback != nil && resp.Content != "" {
				_ = callback(resp.Content, totalInput, totalOutput)
			}
			if callback != nil {
				_ = callback("", totalInput, totalOutput)
			}
			result.Status = PhaseStatusCompleted
			result.Output = resp.Content
			result.InputTokens = totalInput
			result.OutputTokens = totalOutput
			result.ModelUsed = modelUsed
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		// Tool-use turn: execute tools and loop
		req.Messages = append(req.Messages, ports.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})
		toolResults := make([]ports.ToolResult, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			toolResults = append(toolResults, e.executeToolCall(ctx, tc))
		}
		req.Messages = append(req.Messages, ports.Message{
			Role:        "user",
			ToolResults: toolResults,
		})
	}

	// Max tool iterations exceeded without producing final content.
	result.Status = PhaseStatusFailed
	result.Error = fmt.Errorf("max tool iterations (%d) exceeded without producing final content", maxToolIterations)
	result.InputTokens = totalInput
	result.OutputTokens = totalOutput
	result.ModelUsed = modelUsed
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result
}

// executeStreaming is the original streaming path used when no tools are needed.
func (e *streamingPhaseExecutor) executeStreaming(
	ctx context.Context,
	result *PhaseResult,
	req ports.CompletionRequest,
	callback PhaseStreamCallback,
) *PhaseResult {
	var fullContent strings.Builder
	var lastInputTokens int

	streamCallback := func(chunk string) error {
		fullContent.WriteString(chunk)
		if callback != nil {
			return callback(chunk, lastInputTokens, fullContent.Len()/4)
		}
		return nil
	}

	resp, err := e.provider.Stream(ctx, req, streamCallback)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	result.Status = PhaseStatusCompleted
	result.Output = resp.Content
	result.InputTokens = resp.InputTokens
	result.OutputTokens = resp.OutputTokens
	result.ModelUsed = resp.ModelUsed
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if callback != nil {
		_ = callback("", resp.InputTokens, resp.OutputTokens)
	}

	return result
}

// executeToolCall executes a single tool call via the MCP registry and returns the result.
func (e *streamingPhaseExecutor) executeToolCall(ctx context.Context, tc ports.ToolCall) ports.ToolResult {
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

// Execute runs a single phase without streaming (for compatibility).
func (e *streamingPhaseExecutor) Execute(ctx context.Context, phase *skill.Phase, dependencyOutputs map[string]string) *PhaseResult {
	return e.ExecuteWithStreaming(ctx, phase, dependencyOutputs, nil)
}

// filterAllowedTools returns only the tools whose FullName matches the executor's allowedTools list.
func (e *streamingPhaseExecutor) filterAllowedTools(tools []*mcp.Tool) []*mcp.Tool {
	if len(e.allowedTools) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(e.allowedTools))
	for _, name := range e.allowedTools {
		allowed[name] = struct{}{}
	}
	var filtered []*mcp.Tool
	for _, t := range tools {
		if _, ok := allowed[t.FullName()]; ok {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// buildPrompt renders the phase's prompt template with the dependency outputs.
// Phase outputs are also available via {{.phases.phaseid}} for better organization.
func (e *streamingPhaseExecutor) buildPrompt(templateStr string, data map[string]string) (string, error) {
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
	messages := make([]ports.Message, 0, 3)

	// Add memory context first (if available) - highest priority
	if e.memoryContent != "" {
		messages = append(messages, ports.Message{
			Role:    "system",
			Content: "Project Memory:\n\n" + e.memoryContent,
		})
	}

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

// selectModel returns a model ID based on the routing profile, respecting skill-level hints.
func (e *streamingPhaseExecutor) selectModel(ctx context.Context, routingProfile string) string {
	defaultModel := e.defaultModel(routingProfile)

	if len(e.modelHints) > 0 {
		if hinted, ok := e.modelHints[routingProfile]; ok && hinted != "" {
			if ok, err := e.provider.SupportsModel(ctx, hinted); err == nil && ok {
				return hinted
			}
		}
	}

	return defaultModel
}

// defaultModel returns the default model for a routing profile.
func (e *streamingPhaseExecutor) defaultModel(routingProfile string) string {
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
