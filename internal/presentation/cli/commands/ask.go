// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	appProvider "github.com/jbctechsolutions/skillrunner/internal/application/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
	fileContext "github.com/jbctechsolutions/skillrunner/internal/infrastructure/context"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// askFlags holds the flags for the ask command.
type askFlags struct {
	Model       string
	Profile     string
	Phase       string
	Stream      bool
	AutoApprove bool // Skip file permission prompts
}

var askOpts askFlags

// NewAskCmd creates the ask command for quick single-phase queries.
func NewAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask <skill> <question>",
		Short: "Quick one-shot query using a skill's single phase",
		Long: `Execute a quick one-shot query using a single phase from a skill.

The ask command provides a simplified interface for running a single phase
from a skill definition without executing the full multi-phase workflow.

Examples:
  # Ask using the first phase of a skill
  sr ask code-review "Review this function for bugs"

  # Ask with a specific phase
  sr ask code-review "Review security" --phase security-check

  # Ask with a specific model override
  sr ask summarize "Summarize this document" --model claude-3-opus

  # Ask with streaming output
  sr ask explain "Explain this code" --stream

  # Ask with a routing profile
  sr ask translate "Translate this to Spanish" --profile premium`,
		Args: cobra.ExactArgs(2),
		RunE: runAsk,
	}

	// Define flags
	cmd.Flags().StringVarP(&askOpts.Model, "model", "m", "",
		"override model selection (e.g., claude-3-opus, gpt-4, llama3)")
	cmd.Flags().StringVarP(&askOpts.Profile, "profile", "p", skill.ProfileBalanced,
		fmt.Sprintf("routing profile: %s, %s, %s", skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium))
	cmd.Flags().StringVar(&askOpts.Phase, "phase", "",
		"specific phase to execute (defaults to first phase)")
	cmd.Flags().BoolVarP(&askOpts.Stream, "stream", "s", false, "enable streaming output")
	cmd.Flags().BoolVarP(&askOpts.AutoApprove, "yes", "y", false, "auto-approve file access (skip permission prompts)")

	return cmd
}

// runAsk executes the quick one-shot query using a skill's single phase.
func runAsk(cmd *cobra.Command, args []string) error {
	skillName := args[0]
	question := args[1]

	// Detect and inject file context
	detector := fileContext.NewFileDetector()
	fileRefs := detector.DetectFiles(question)

	if len(fileRefs) > 0 {
		// Prompt for permission
		prompter := fileContext.NewPermissionPrompt(askOpts.AutoApprove)
		approvedFiles, err := prompter.PromptForFiles(fileRefs)
		if err != nil {
			return fmt.Errorf("file access denied: %w", err)
		}

		// Inject file content into question
		enrichedQuestion, err := detector.InjectFileContext(question, approvedFiles)
		if err != nil {
			return fmt.Errorf("failed to inject file context: %w", err)
		}
		question = enrichedQuestion

		// Print feedback to user (only if auto-approved, otherwise already shown)
		if askOpts.AutoApprove {
			fmt.Printf("📄 Detected %d file(s), injecting context...\n", len(approvedFiles))
			for _, ref := range approvedFiles {
				fmt.Printf("  • %s\n", ref.Path)
			}
			fmt.Println()
		}
	}

	// Validate profile
	if err := validateAskProfile(askOpts.Profile); err != nil {
		return err
	}

	formatter := GetFormatter()
	ctx := context.Background()

	// Get the application container
	container := GetContainer()
	if container == nil {
		return fmt.Errorf("application not initialized")
	}

	// Get the skill registry and load the skill
	registry := container.SkillRegistry()
	if registry == nil {
		return fmt.Errorf("skill registry not initialized")
	}

	// Try to find skill by ID first, then by name
	s := registry.GetSkill(skillName)
	if s == nil {
		s = registry.GetSkillByName(skillName)
	}
	if s == nil {
		return fmt.Errorf("skill not found: %s", skillName)
	}

	// Get the phase to execute
	phase, err := getPhaseToExecute(s, askOpts.Phase)
	if err != nil {
		return err
	}

	// Create provider router for model selection
	providerRegistry := container.ProviderRegistry()
	routingCfg := config.NewRoutingConfiguration()
	router, err := appProvider.NewRouter(routingCfg, providerRegistry)
	if err != nil {
		return fmt.Errorf("could not create router: %w", err)
	}

	// Select model based on profile (or use override)
	var modelSelection *appProvider.ModelSelection
	if askOpts.Model != "" {
		// Use the specified model override
		modelSelection = &appProvider.ModelSelection{
			ModelID:      askOpts.Model,
			ProviderName: "", // Will be determined by registry
			IsFallback:   false,
		}
	} else {
		// Use router to select model based on profile
		profile := askOpts.Profile
		if phase.RoutingProfile != "" {
			profile = phase.RoutingProfile
		}
		modelSelection, err = router.SelectModel(ctx, profile)
		if err != nil {
			return fmt.Errorf("could not select model: %w", err)
		}
	}

	// Get the provider for the selected model
	var provider ports.ProviderPort
	if modelSelection.ProviderName != "" {
		provider = providerRegistry.Get(modelSelection.ProviderName)
	} else if askOpts.Model != "" {
		// Find provider by model
		provider, err = providerRegistry.FindByModel(ctx, askOpts.Model)
		if err != nil {
			return fmt.Errorf("no provider found for model %s: %w", askOpts.Model, err)
		}
	}

	if provider == nil {
		return fmt.Errorf("no provider available for model selection")
	}

	// Build the prompt from the phase template
	prompt := buildPromptFromPhase(phase, question)

	// Build completion request
	maxTokens := phase.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}
	temperature := phase.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	req := ports.CompletionRequest{
		ModelID:     modelSelection.ModelID,
		Messages:    buildMessagesForAsk(prompt, question),
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// Execute the request (with or without streaming)
	var response *ports.CompletionResponse
	if askOpts.Stream {
		response, err = executeWithStreaming(ctx, provider, req, formatter)
	} else {
		response, err = provider.Complete(ctx, req)
	}

	if err != nil {
		return fmt.Errorf("failed to get response: %w", err)
	}

	// Output results
	if formatter.Format() == "json" {
		result := map[string]any{
			"skill":         s.Name(),
			"phase":         phase.Name,
			"question":      question,
			"answer":        response.Content,
			"model":         response.ModelUsed,
			"provider":      provider.Info().Name,
			"input_tokens":  response.InputTokens,
			"output_tokens": response.OutputTokens,
			"profile":       askOpts.Profile,
			"is_fallback":   modelSelection.IsFallback,
		}
		return formatter.JSON(result)
	}

	// Text output for terminal (skip if streaming already printed)
	if !askOpts.Stream {
		formatter.Header("Ask")
		formatter.Item("Skill", s.Name())
		formatter.Item("Phase", phase.Name)
		formatter.Item("Profile", askOpts.Profile)
		formatter.Item("Model", response.ModelUsed)
		formatter.Item("Provider", provider.Info().Name)
		if modelSelection.IsFallback {
			formatter.Warning("Using fallback model")
		}
		formatter.Println("")
		formatter.Success("Answer:")
		formatter.Println(response.Content)
		formatter.Println("")
		formatter.Item("Tokens", fmt.Sprintf("in=%d out=%d", response.InputTokens, response.OutputTokens))
	} else {
		// For streaming, just show summary at the end
		formatter.Println("")
		formatter.Item("Tokens", fmt.Sprintf("in=%d out=%d", response.InputTokens, response.OutputTokens))
	}

	return nil
}

// getPhaseToExecute returns the specified phase or the first phase if not specified.
func getPhaseToExecute(s *skill.Skill, phaseID string) (*skill.Phase, error) {
	phases := s.Phases()
	if len(phases) == 0 {
		return nil, fmt.Errorf("skill %s has no phases", s.Name())
	}

	if phaseID == "" {
		// Return the first phase
		return &phases[0], nil
	}

	// Find the specified phase
	phase, err := s.GetPhase(phaseID)
	if err != nil {
		// Try matching by name
		for i := range phases {
			if strings.EqualFold(phases[i].Name, phaseID) {
				return &phases[i], nil
			}
		}
		return nil, fmt.Errorf("phase not found: %s", phaseID)
	}
	return phase, nil
}

// buildPromptFromPhase builds a prompt from the phase template and question.
func buildPromptFromPhase(phase *skill.Phase, question string) string {
	// If the phase has a prompt template, use it with the question as input
	if phase.PromptTemplate != "" {
		// Simple substitution - replace {{.input}} or {{._input}} with question
		prompt := phase.PromptTemplate
		prompt = strings.ReplaceAll(prompt, "{{.input}}", question)
		prompt = strings.ReplaceAll(prompt, "{{._input}}", question)
		prompt = strings.ReplaceAll(prompt, "{{index . \"_input\"}}", question)
		return prompt
	}

	// If no template, just use the question directly
	return question
}

// buildMessagesForAsk constructs messages for the ask request.
func buildMessagesForAsk(prompt, question string) []ports.Message {
	messages := make([]ports.Message, 0, 2)

	// Add original input as context if the prompt differs from question
	if prompt != question {
		messages = append(messages, ports.Message{
			Role:    "system",
			Content: "Original request: " + question,
		})
	}

	// Add the main prompt as user message
	messages = append(messages, ports.Message{
		Role:    "user",
		Content: prompt,
	})

	return messages
}

// executeWithStreaming executes the request with streaming output.
func executeWithStreaming(ctx context.Context, provider ports.ProviderPort, req ports.CompletionRequest, formatter *output.Formatter) (*ports.CompletionResponse, error) {
	formatter.Header("Ask (Streaming)")
	formatter.Success("Answer:")

	// Create streaming callback
	callback := func(chunk string) error {
		// Print each chunk without newline (streaming output)
		fmt.Print(chunk)
		return nil
	}

	// Execute with streaming
	response, err := provider.Stream(ctx, req, callback)
	if err != nil {
		return nil, err
	}

	// Add newline after streaming completes
	fmt.Println()

	return response, nil
}

// validateAskProfile checks if the profile is valid.
func validateAskProfile(profile string) error {
	profile = strings.ToLower(strings.TrimSpace(profile))
	validProfiles := []string{skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium}

	if slices.Contains(validProfiles, profile) {
		return nil
	}

	return fmt.Errorf("invalid profile %q: must be one of %s", profile, strings.Join(validProfiles, ", "))
}
