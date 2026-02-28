package commands

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// NewModelsCmd creates the `sr models` command group.
func NewModelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Model management and recommendations",
		Long:  "Inspect available models and get task-specific recommendations.",
	}

	cmd.AddCommand(newModelsRecommendCmd())

	return cmd
}

// newModelsRecommendCmd creates `sr models recommend`.
func newModelsRecommendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "recommend",
		Short: "Show model recommendations per skill",
		Long: `Show which Ollama model will be used for each skill and routing profile.

Hints configured under routing.skill_model_hints in ~/.skillrunner/config.yaml
override the defaults. An asterisk (*) marks an active hint; [unavailable] means
the hinted model could not be found in the running Ollama instance.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsRecommend(cmd.Context())
		},
	}
}

// runModelsRecommend prints skill → profile → model suggestions.
func runModelsRecommend(ctx context.Context) error {
	container := GetContainer()
	if container == nil {
		return fmt.Errorf("application not initialized")
	}

	appCtx := GetAppContext()
	if appCtx == nil || appCtx.Config == nil {
		return fmt.Errorf("configuration not available")
	}

	// Load skill registry
	registry := container.SkillRegistry()
	if registry == nil {
		return fmt.Errorf("skill registry not available")
	}
	skills := registry.ListSkills()

	// Get local (Ollama) provider for availability checks
	var availableModels []string
	providerReg := container.ProviderRegistry()
	providers := providerReg.ListProviders()
	for _, p := range providers {
		if p.Info().IsLocal {
			if models, err := p.ListModels(ctx); err == nil {
				availableModels = models
			}
			break
		}
	}

	hints := appCtx.Config.Routing.SkillModelHints

	// Print header
	fmt.Printf("Model recommendations (Ollama)\n")
	fmt.Printf("Available models: %s\n\n", formatAvailableModels(availableModels))

	profiles := []string{
		skill.RoutingProfileCheap,
		skill.RoutingProfileBalanced,
		skill.RoutingProfilePremium,
	}
	defaults := map[string]string{
		skill.RoutingProfileCheap:    "llama3.2:3b",
		skill.RoutingProfileBalanced: "llama3:8b",
		skill.RoutingProfilePremium:  "qwen2.5:14b",
	}

	// Sort skills for deterministic output
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name() < skills[j].Name()
	})

	for _, sk := range skills {
		fmt.Printf("Skill: %s (%s)\n", sk.Name(), sk.ID())

		// Resolve hints for this skill (try ID then name)
		var perSkill map[string]string
		if h, ok := hints[sk.ID()]; ok {
			perSkill = h
		} else if h, ok := hints[sk.Name()]; ok {
			perSkill = h
		}

		for _, profile := range profiles {
			model := defaults[profile]
			tag := ""

			if hinted, ok := perSkill[profile]; ok && hinted != "" {
				avail := isModelAvailable(hinted, availableModels)
				if avail {
					model = hinted
					tag = " *"
				} else {
					tag = fmt.Sprintf(" [hint: %s — unavailable, using default]", hinted)
				}
			}

			fmt.Printf("  %-10s → %s%s\n", profile, model, tag)
		}
		fmt.Println()
	}

	fmt.Println("* = active skill hint from routing.skill_model_hints")

	return nil
}

// isModelAvailable checks if a model name appears in the available models list.
func isModelAvailable(model string, available []string) bool {
	norm := func(s string) string {
		// strip ":latest" suffix for comparison
		if len(s) > 7 && s[len(s)-7:] == ":latest" {
			s = s[:len(s)-7]
		}
		return s
	}
	target := norm(model)
	for _, m := range available {
		if norm(m) == target {
			return true
		}
	}
	return false
}

// formatAvailableModels formats the available models list for display.
func formatAvailableModels(models []string) string {
	if len(models) == 0 {
		return "(none — Ollama not reachable)"
	}
	if len(models) > 5 {
		return fmt.Sprintf("%s … (+%d more)", joinStrings(models[:5], ", "), len(models)-5)
	}
	return joinStrings(models, ", ")
}

// joinStrings joins a string slice with a separator.
func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
