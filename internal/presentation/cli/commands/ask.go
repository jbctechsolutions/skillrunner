// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/application/chat"
	appProvider "github.com/jbctechsolutions/skillrunner/internal/application/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

// askFlags holds the flags for the ask command.
type askFlags struct {
	Model   string
	Profile string
}

var askOpts askFlags

// NewAskCmd creates the ask command for quick single-phase queries.
func NewAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Quick one-shot query",
		Long: `Execute a quick one-shot query using the chat service.

The ask command provides a simplified interface for asking quick questions
without the need for a full interactive chat session.

Examples:
  # Ask a quick question using the default model
  sr ask "What are the key points of this document?"

  # Ask with a specific model override
  sr ask "Explain this code" --model claude-3-opus

  # Ask with a routing profile
  sr ask "Translate this to Spanish" --profile premium`,
		Args: cobra.ExactArgs(1),
		RunE: runAsk,
	}

	// Define flags
	cmd.Flags().StringVarP(&askOpts.Model, "model", "m", "",
		"override model selection (e.g., claude-3-opus, gpt-4, llama3)")
	cmd.Flags().StringVarP(&askOpts.Profile, "profile", "p", skill.ProfileBalanced,
		fmt.Sprintf("routing profile: %s, %s, %s", skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium))

	return cmd
}

// runAsk executes the quick one-shot query.
func runAsk(cmd *cobra.Command, args []string) error {
	question := args[0]

	// Validate profile
	if err := validateAskProfile(askOpts.Profile); err != nil {
		return err
	}

	formatter := GetFormatter()
	ctx := context.Background()

	// Initialize chat service
	chatService, err := initChatService()
	if err != nil {
		return fmt.Errorf("could not initialize chat service: %w", err)
	}

	// Build ask request
	askReq := &chat.AskRequest{
		Question:      question,
		Profile:       askOpts.Profile,
		ModelOverride: askOpts.Model,
		MaxTokens:     2048,
		Temperature:   0.7,
	}

	// Execute the ask
	response, err := chatService.Ask(ctx, askReq)
	if err != nil {
		return fmt.Errorf("failed to get response: %w", err)
	}

	// Output results
	if formatter.Format() == "json" {
		result := map[string]any{
			"question":      question,
			"answer":        response.Answer,
			"model":         response.ModelUsed,
			"provider":      response.Provider,
			"input_tokens":  response.InputTokens,
			"output_tokens": response.OutputTokens,
			"is_fallback":   response.IsFallback,
			"profile":       askOpts.Profile,
		}
		return formatter.JSON(result)
	}

	// Text output for terminal
	formatter.Header("Ask")
	formatter.Item("Question", question)
	formatter.Item("Profile", askOpts.Profile)
	formatter.Item("Model", response.ModelUsed)
	formatter.Item("Provider", response.Provider)
	if response.IsFallback {
		formatter.Warning("Using fallback model")
	}
	formatter.Println("")
	formatter.Success("Answer:")
	formatter.Println(response.Answer)
	formatter.Println("")
	formatter.Item("Tokens", fmt.Sprintf("in=%d out=%d", response.InputTokens, response.OutputTokens))

	return nil
}

// validateAskProfile checks if the profile is valid.
func validateAskProfile(profile string) error {
	profile = strings.ToLower(strings.TrimSpace(profile))
	validProfiles := []string{skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium}

	for _, valid := range validProfiles {
		if profile == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid profile %q: must be one of %s", profile, strings.Join(validProfiles, ", "))
}

// initChatService initializes the chat service with provider registry and router.
func initChatService() (*chat.Service, error) {
	appCtx := GetAppContext()
	if appCtx == nil {
		return nil, fmt.Errorf("app context not initialized")
	}

	// Create provider registry
	registry := adapterProvider.NewRegistry()

	// TODO: Register providers based on configuration
	// For now, this is a stub - providers should be registered from config

	// Create routing configuration from app config
	// Use default RoutingConfiguration with sensible defaults
	routingCfg := config.NewRoutingConfiguration()

	// Create router
	router, err := appProvider.NewRouter(routingCfg, registry)
	if err != nil {
		return nil, fmt.Errorf("could not create router: %w", err)
	}

	// Create chat service
	chatService, err := chat.NewService(router, registry)
	if err != nil {
		return nil, fmt.Errorf("could not create chat service: %w", err)
	}

	return chatService, nil
}
