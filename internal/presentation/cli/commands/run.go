// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// runFlags holds the flags for the run command.
type runFlags struct {
	Profile string
	Stream  bool
}

var runOpts runFlags

// NewRunCmd creates the run command for executing skills.
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <skill> <request>",
		Short: "Execute a skill with the given request",
		Long: `Execute a multi-phase AI workflow skill with the specified request.

The run command executes a skill definition, orchestrating the multi-phase
workflow and managing provider selection based on the routing profile.

Examples:
  # Run a skill with default settings
  sr run code-review "Review this pull request for security issues"

  # Run with a specific profile
  sr run code-review "Review this PR" --profile premium

  # Run with streaming output
  sr run summarize "Summarize this document" --stream

Routing Profiles:
  cheap     - Prioritize cost, use local/cheaper models
  balanced  - Balance between cost and quality (default)
  premium   - Prioritize quality, use best available models`,
		Args: cobra.ExactArgs(2),
		RunE: runSkill,
	}

	// Define flags
	cmd.Flags().StringVarP(&runOpts.Profile, "profile", "p", skill.ProfileBalanced,
		fmt.Sprintf("routing profile: %s, %s, %s", skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium))
	cmd.Flags().BoolVarP(&runOpts.Stream, "stream", "s", false, "enable streaming output")

	return cmd
}

// runSkill executes the skill workflow.
func runSkill(cmd *cobra.Command, args []string) error {
	skillName := args[0]
	request := args[1]

	// Validate profile
	if err := validateProfile(runOpts.Profile); err != nil {
		return err
	}

	formatter := GetFormatter()

	// For now, implement stub execution that prints the execution info
	if formatter.Format() == "json" {
		// JSON output for scripting
		result := map[string]any{
			"skill":   skillName,
			"request": request,
			"profile": runOpts.Profile,
			"stream":  runOpts.Stream,
			"status":  "stub",
			"message": fmt.Sprintf("Executing skill: %s with request: %s", skillName, request),
		}
		return formatter.JSON(result)
	}

	// Text output for terminal
	formatter.Header("Skill Execution")
	formatter.Item("Skill", skillName)
	formatter.Item("Profile", runOpts.Profile)
	formatter.Item("Streaming", fmt.Sprintf("%t", runOpts.Stream))
	formatter.Println("")
	formatter.Item("Request", request)
	formatter.Println("")

	// Stub message
	formatter.Info("Executing skill: %s with request: %s", skillName, request)

	// TODO: Implement actual workflow execution with:
	// 1. Load skill definition from registry
	// 2. Initialize workflow executor
	// 3. Execute phases with provider routing
	// 4. Handle streaming output if enabled
	// 5. Return results

	return nil
}

// validateProfile checks if the profile is valid.
func validateProfile(profile string) error {
	profile = strings.ToLower(strings.TrimSpace(profile))
	validProfiles := []string{skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium}

	for _, valid := range validProfiles {
		if profile == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid profile %q: must be one of %s", profile, strings.Join(validProfiles, ", "))
}

// init registers the run command with the root command.
func init() {
	// This will be called when the package is imported
}
