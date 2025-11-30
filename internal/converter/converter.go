package converter

import (
	"fmt"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// ClaudeSkillFormat represents a Claude Code marketplace skill format
type ClaudeSkillFormat struct {
	Name        string                 `yaml:"name"`
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description"`
	Phases      []ClaudePhase          `yaml:"phases"`
	Routing     *ClaudeRouting         `yaml:"routing,omitempty"`
	Context     map[string]interface{} `yaml:"context,omitempty"`
}

// ClaudePhase represents a phase in Claude skill format
type ClaudePhase struct {
	ID              string   `yaml:"id"`
	TaskType        string   `yaml:"task_type"`
	MaxInputTokens  int      `yaml:"max_input_tokens,omitempty"`
	MaxOutputTokens int      `yaml:"max_output_tokens,omitempty"`
	Parallelizable  bool     `yaml:"parallelizable,omitempty"`
	Description     string   `yaml:"description"`
	ContextFields   []string `yaml:"context_fields,omitempty"`
	DependsOn       []string `yaml:"depends_on,omitempty"`
}

// ClaudeRouting represents routing in Claude skill format
type ClaudeRouting struct {
	ByTaskType map[string][]string `yaml:"by_task_type"`
}

// ToSkillrunner converts Claude skill format to Skillrunner orchestrated format
func ToSkillrunner(claude *ClaudeSkillFormat) (*types.OrchestratedSkill, error) {
	skill := &types.OrchestratedSkill{
		Name:        claude.Name,
		Version:     claude.Version,
		Description: claude.Description,
		Type:        types.SkillTypeOrchestrated,
		Phases:      make([]types.Phase, 0, len(claude.Phases)),
		Context:     claude.Context,
	}

	// Convert global routing to skill-level routing
	if claude.Routing != nil {
		skill.Routing = &types.RoutingConfig{
			// Map the routing - this is simplified, may need enhancement
		}
	}

	// Convert each phase
	for _, cp := range claude.Phases {
		phase := types.Phase{
			ID:          cp.ID,
			Name:        formatPhaseName(cp.ID),
			TaskType:    types.TaskType(cp.TaskType),
			DependsOn:   cp.DependsOn,
			MaxTokens:   cp.MaxOutputTokens,
			OutputKey:   cp.ID, // Use phase ID as output key
			Temperature: 0.7,   // Default
		}

		// Generate prompt template based on task type and description
		phase.PromptTemplate = generatePromptTemplate(cp)

		// Add routing preferences if available
		if claude.Routing != nil && claude.Routing.ByTaskType != nil {
			if models, ok := claude.Routing.ByTaskType[cp.TaskType]; ok && len(models) > 0 {
				phase.Routing = &types.PhaseRouting{
					PreferredModels: models,
				}
			}
		}

		skill.Phases = append(skill.Phases, phase)
	}

	return skill, nil
}

// FromSkillrunner converts Skillrunner orchestrated format to Claude skill format
func FromSkillrunner(skill *types.OrchestratedSkill) (*ClaudeSkillFormat, error) {
	claude := &ClaudeSkillFormat{
		Name:        skill.Name,
		Version:     skill.Version,
		Description: skill.Description,
		Phases:      make([]ClaudePhase, 0, len(skill.Phases)),
		Context:     skill.Context,
	}

	// Build routing from phase preferences
	routingMap := make(map[string][]string)
	for _, phase := range skill.Phases {
		if phase.Routing != nil && len(phase.Routing.PreferredModels) > 0 {
			taskTypeKey := string(phase.TaskType)
			// Merge models for same task type
			if existing, ok := routingMap[taskTypeKey]; ok {
				// Add unique models
				for _, model := range phase.Routing.PreferredModels {
					if !contains(existing, model) {
						routingMap[taskTypeKey] = append(routingMap[taskTypeKey], model)
					}
				}
			} else {
				routingMap[taskTypeKey] = phase.Routing.PreferredModels
			}
		}
	}

	if len(routingMap) > 0 {
		claude.Routing = &ClaudeRouting{
			ByTaskType: routingMap,
		}
	}

	// Convert each phase
	for _, p := range skill.Phases {
		cp := ClaudePhase{
			ID:              p.ID,
			TaskType:        string(p.TaskType),
			MaxInputTokens:  estimateInputTokens(p.PromptTemplate),
			MaxOutputTokens: p.MaxTokens,
			Description:     extractDescription(p.PromptTemplate),
			DependsOn:       p.DependsOn,
			Parallelizable:  len(p.DependsOn) == 0, // Simplified
		}

		// Extract context fields from prompt template
		cp.ContextFields = extractContextFields(p.PromptTemplate)

		claude.Phases = append(claude.Phases, cp)
	}

	return claude, nil
}

// generatePromptTemplate creates a prompt template from Claude phase
func generatePromptTemplate(cp ClaudePhase) string {
	var sb strings.Builder

	// Add description as base
	sb.WriteString(cp.Description)
	sb.WriteString("\n\n")

	// Add context placeholders
	if len(cp.ContextFields) > 0 {
		sb.WriteString("Context:\n")
		for _, field := range cp.ContextFields {
			if field == "*" {
				sb.WriteString("{{request}}\n")
			} else {
				sb.WriteString(fmt.Sprintf("{{%s}}\n", field))
			}
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("Input: {{request}}\n\n")
	}

	// Add dependency outputs
	if len(cp.DependsOn) > 0 {
		sb.WriteString("Previous phase outputs:\n")
		for _, dep := range cp.DependsOn {
			sb.WriteString(fmt.Sprintf("- {{%s}}\n", dep))
		}
		sb.WriteString("\n")
	}

	// Add task-specific instructions
	switch cp.TaskType {
	case "extraction":
		sb.WriteString("Extract and summarize the key information from the above context.\n")
	case "generation":
		sb.WriteString("Generate the required output based on the above context.\n")
	case "verification":
		sb.WriteString("Review and validate the above information for correctness.\n")
	case "analysis":
		sb.WriteString("Analyze the above information and provide insights.\n")
	default:
		sb.WriteString("Process the above information as requested.\n")
	}

	return sb.String()
}

// extractDescription extracts a description from prompt template
func extractDescription(promptTemplate string) string {
	// Take first line or first 100 chars as description
	lines := strings.Split(promptTemplate, "\n")
	if len(lines) > 0 {
		desc := strings.TrimSpace(lines[0])
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		return desc
	}
	return "Process input and generate output"
}

// extractContextFields extracts context field names from prompt template
func extractContextFields(promptTemplate string) []string {
	fields := []string{}

	// Find all {{field}} placeholders
	start := 0
	for {
		idx := strings.Index(promptTemplate[start:], "{{")
		if idx == -1 {
			break
		}
		idx += start
		endIdx := strings.Index(promptTemplate[idx:], "}}")
		if endIdx == -1 {
			break
		}
		endIdx += idx

		field := strings.TrimSpace(promptTemplate[idx+2 : endIdx])
		if field != "" && !contains(fields, field) {
			fields = append(fields, field)
		}

		start = endIdx + 2
	}

	if len(fields) == 0 {
		fields = append(fields, "*")
	}

	return fields
}

// estimateInputTokens estimates input tokens from prompt template
func estimateInputTokens(promptTemplate string) int {
	// Rough estimate: 4 characters per token
	chars := len(promptTemplate)
	// Add buffer for context injection
	chars += 2000
	tokens := chars / 4

	// Round to nearest 1000
	tokens = ((tokens + 500) / 1000) * 1000

	// Clamp to reasonable range
	if tokens < 1000 {
		tokens = 1000
	}
	if tokens > 20000 {
		tokens = 20000
	}

	return tokens
}

// formatPhaseName converts phase ID to readable name
func formatPhaseName(id string) string {
	// Convert snake_case to Title Case
	words := strings.Split(id, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// contains checks if a string slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
