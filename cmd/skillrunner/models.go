package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/jbctechsolutions/skillrunner/internal/models"
	"github.com/spf13/cobra"
)

var (
	providerFilter string
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage and discover models",
	Long:  `Commands for listing, checking, and validating AI models across all providers.`,
}

var modelsListCmd = &cobra.Command{
	Use:   "list [--provider=name]",
	Short: "List all available models",
	Long: `List all available models from all providers (or filtered by provider).

Examples:
  sr models list
  sr models list --provider=ollama
  sr models list --provider=anthropic`,
	RunE: listModels,
}

var modelsCheckCmd = &cobra.Command{
	Use:   "check <model>",
	Short: "Check specific model health",
	Long: `Check the health and availability of a specific model.

The model format is: [provider/]model-name
- ollama/qwen2.5:14b
- anthropic/claude-3-5-sonnet-20241022
- qwen2.5:14b (searches all providers)

Examples:
  sr models check ollama/qwen2.5:14b
  sr models check anthropic/claude-3-5-sonnet-20241022
  sr models check qwen2.5:14b`,
	Args: cobra.ExactArgs(1),
	RunE: checkModel,
}

var modelsValidateCmd = &cobra.Command{
	Use:   "validate <skill>",
	Short: "Validate models for a skill",
	Long: `Load a skill and check all its preferred_models to recommend the best one.

Examples:
  sr models validate golang-pro
  sr models validate backend-architect`,
	Args: cobra.ExactArgs(1),
	RunE: validateSkill,
}

var modelsRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh provider caches",
	Long: `Clear provider caches and force re-fetch of model lists.

This is useful when you've added new models to Ollama or want to
ensure you have the latest model information.`,
	RunE: refreshModels,
}

func init() {
	// Add flags
	modelsListCmd.Flags().StringVar(&providerFilter, "provider", "", "Filter by provider name")

	// Add subcommands
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsCheckCmd)
	modelsCmd.AddCommand(modelsValidateCmd)
	modelsCmd.AddCommand(modelsRefreshCmd)
}

// createOrchestrator sets up the model orchestrator with all configured providers
func createOrchestrator() (*models.Orchestrator, error) {
	ctx := context.Background()

	// Load configuration
	cfgManager, err := config.NewManager("")
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	cfg := cfgManager.Get()

	// Create orchestrator
	orchestrator := models.NewOrchestrator()

	// Register Ollama provider
	ollamaURL := cfg.Router.OllamaURL
	if ollamaURL == "" {
		ollamaURL = "http://localhost:18433"
	}
	ollamaProvider, err := models.NewOllamaProvider(ollamaURL, nil)
	if err == nil {
		orchestrator.RegisterProvider(ollamaProvider)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize Ollama provider: %v\n", err)
	}

	// Register Anthropic provider if API key is available
	anthropicKey := cfgManager.GetAPIKey("anthropic")
	if anthropicKey == "" {
		anthropicKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if anthropicKey != "" {
		anthropicProvider, err := models.NewAnthropicProvider(anthropicKey, "", nil)
		if err == nil {
			orchestrator.RegisterProvider(anthropicProvider)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize Anthropic provider: %v\n", err)
		}
	}

	// Test connectivity
	_, err = orchestrator.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to providers: %w", err)
	}

	return orchestrator, nil
}

func listModels(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	orchestrator, err := createOrchestrator()
	if err != nil {
		return err
	}

	allModels, err := orchestrator.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}

	// Filter by provider if specified
	var filteredModels []models.RegisteredModel
	if providerFilter != "" {
		for _, model := range allModels {
			if strings.EqualFold(model.Provider.Name, providerFilter) {
				filteredModels = append(filteredModels, model)
			}
		}
		if len(filteredModels) == 0 {
			fmt.Fprintf(os.Stderr, "No models found for provider: %s\n", providerFilter)
			return nil
		}
	} else {
		filteredModels = allModels
	}

	// Display table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tMODEL\tCONTEXT\tMEMORY\tSTATUS")
	fmt.Fprintln(w, "--------\t-----\t-------\t------\t------")

	for _, model := range filteredModels {
		provider := model.Provider.Name
		modelName := model.Name

		// Context window (placeholder - would need model-specific data)
		context := "-"
		if strings.Contains(strings.ToLower(modelName), "claude") {
			context = "200K"
		} else if strings.Contains(strings.ToLower(modelName), "gpt-4") {
			context = "128K"
		} else if strings.Contains(strings.ToLower(modelName), "qwen") {
			context = "32K"
		} else if strings.Contains(strings.ToLower(modelName), "llama") {
			context = "8K"
		}

		// Memory (placeholder - would need actual model inspection)
		memory := "-"
		if model.Provider.Type == models.ProviderTypeLocal {
			if strings.Contains(strings.ToLower(modelName), "32b") {
				memory = "19GB"
			} else if strings.Contains(strings.ToLower(modelName), "14b") {
				memory = "9GB"
			} else if strings.Contains(strings.ToLower(modelName), "7b") {
				memory = "5GB"
			} else if strings.Contains(strings.ToLower(modelName), "3b") {
				memory = "2GB"
			}
		}

		// Status
		status := "\u2713 Ready"
		if !model.Available {
			status = "\u2717 Unavailable"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", provider, modelName, context, memory, status)
	}

	w.Flush()

	// Summary
	fmt.Printf("\nTotal: %d models", len(filteredModels))
	if providerFilter != "" {
		fmt.Printf(" (provider: %s)", providerFilter)
	}
	fmt.Println()

	return nil
}

func checkModel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	modelSpec := args[0]

	orchestrator, err := createOrchestrator()
	if err != nil {
		return err
	}

	// Parse model specification
	var providerName, modelName string
	if parts := strings.SplitN(modelSpec, "/", 2); len(parts) == 2 {
		providerName = parts[0]
		modelName = parts[1]
	} else {
		modelName = modelSpec
	}

	// Get all models
	allModels, err := orchestrator.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}

	// Find matching model
	var found *models.RegisteredModel
	for i, model := range allModels {
		nameMatch := strings.EqualFold(model.Name, modelName)
		providerMatch := providerName == "" || strings.EqualFold(model.Provider.Name, providerName)

		if nameMatch && providerMatch {
			found = &allModels[i]
			break
		}
	}

	if found == nil {
		fmt.Fprintf(os.Stderr, "\u2717 Model not found: %s\n", modelSpec)
		os.Exit(2)
	}

	// Check availability
	available, err := orchestrator.IsAvailable(ctx, found.Name)
	if err != nil {
		return fmt.Errorf("check availability: %w", err)
	}

	// Display status
	if available {
		fmt.Printf("\u2713 Model available\n")
	} else {
		fmt.Printf("\u2717 Model unavailable\n")
	}

	fmt.Printf("  Provider:    %s (%s)\n", found.Provider.Name, found.Provider.Type)
	fmt.Printf("  Model:       %s\n", found.Name)

	// Context window (estimate)
	context := "unknown"
	if strings.Contains(strings.ToLower(found.Name), "claude") {
		context = "200,000 tokens"
	} else if strings.Contains(strings.ToLower(found.Name), "gpt-4") {
		context = "128,000 tokens"
	} else if strings.Contains(strings.ToLower(found.Name), "qwen") {
		context = "32,000 tokens"
	} else if strings.Contains(strings.ToLower(found.Name), "llama") {
		context = "8,000 tokens"
	}
	fmt.Printf("  Context window: %s\n", context)

	// Memory requirements (for local models)
	if found.Provider.Type == models.ProviderTypeLocal {
		memory := "unknown"
		if strings.Contains(strings.ToLower(found.Name), "32b") {
			memory = "19GB"
		} else if strings.Contains(strings.ToLower(found.Name), "14b") {
			memory = "9GB"
		} else if strings.Contains(strings.ToLower(found.Name), "7b") {
			memory = "5GB"
		} else if strings.Contains(strings.ToLower(found.Name), "3b") {
			memory = "2GB"
		}
		fmt.Printf("  Memory required: %s\n", memory)
		// TODO: Get actual available memory from system
		fmt.Printf("  Available memory: (system check not implemented)\n")
	}

	// Cost info
	if found.CostPer1KTokens > 0 {
		fmt.Printf("  Cost per 1K tokens: $%.4f\n", found.CostPer1KTokens)
	} else {
		fmt.Printf("  Cost: Free (local model)\n")
	}

	// Status
	if available {
		fmt.Printf("  Status: Ready to use\n")
	} else {
		fmt.Printf("  Status: Not available\n")
		os.Exit(2)
	}

	return nil
}

func validateSkill(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	skillName := args[0]

	// This is a placeholder implementation
	// In a full implementation, this would:
	// 1. Load the skill file
	// 2. Parse preferred_models
	// 3. Check each model's availability
	// 4. Recommend the best one based on performance/cost/availability

	orchestrator, err := createOrchestrator()
	if err != nil {
		return err
	}

	fmt.Printf("Checking models for skill '%s'...\n\n", skillName)

	// Example model preferences (in a real implementation, load from skill file)
	preferredModels := []string{
		"ollama/qwen2.5-coder:32b",
		"ollama/qwen2.5:14b",
		"ollama/llama3.2:3b",
		"anthropic/claude-3-5-sonnet-20241022",
	}

	var availableModels []string
	var recommendedModel string

	for _, modelSpec := range preferredModels {
		parts := strings.SplitN(modelSpec, "/", 2)
		if len(parts) != 2 {
			continue
		}
		modelName := parts[1]

		available, err := orchestrator.IsAvailable(ctx, modelName)
		if err != nil {
			fmt.Printf("  \u2717 %s - Error checking: %v\n", modelSpec, err)
			continue
		}

		if available {
			fmt.Printf("  \u2713 %s - Available\n", modelSpec)
			availableModels = append(availableModels, modelSpec)
			if recommendedModel == "" {
				recommendedModel = modelSpec
			}
		} else {
			fmt.Printf("  \u2717 %s - Not available\n", modelSpec)
		}
	}

	fmt.Println()

	if recommendedModel != "" {
		fmt.Printf("Recommended model: %s\n", recommendedModel)
	} else {
		fmt.Printf("No available models found for skill '%s'\n", skillName)
		fmt.Printf("Please install one of the preferred models or configure alternatives.\n")
		os.Exit(2)
	}

	return nil
}

func refreshModels(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	orchestrator, err := createOrchestrator()
	if err != nil {
		return err
	}

	fmt.Println("Refreshing model caches...")

	// Force refresh by re-listing models
	allModels, err := orchestrator.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("refresh failed: %w", err)
	}

	// Count models per provider
	providerCounts := make(map[string]int)
	for _, model := range allModels {
		providerCounts[model.Provider.Name]++
	}

	fmt.Println("\nUpdated model counts per provider:")
	for provider, count := range providerCounts {
		fmt.Printf("  %s: %d models\n", provider, count)
	}

	fmt.Printf("\nTotal: %d models\n", len(allModels))

	return nil
}
