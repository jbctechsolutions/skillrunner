// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application/workflow"
	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	domainWorkflow "github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
	infraMemory "github.com/jbctechsolutions/skillrunner/internal/infrastructure/memory"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/tokenizer"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// planFlags holds the flags for the plan command.
type planFlags struct {
	Profile  string
	Approve  bool
	SaveOnly bool
	Output   string
	NoMemory bool
}

var planOpts planFlags

// NewPlanCmd creates the plan command for showing execution plans.
func NewPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan <skill> <request>",
		Short: "Show the execution plan for a skill before running",
		Long: `Display the execution plan for a multi-phase AI workflow skill.

The plan command shows the execution plan including:
  - DAG visualization with phase dependencies
  - Model selection per phase based on routing profile
  - Token estimates (input and output)
  - Cost estimates per phase and total

After displaying the plan, you can approve to execute or cancel.

Examples:
  # Show execution plan and prompt for approval
  sr plan code-review "Review this pull request"

  # Auto-approve and execute immediately
  sr plan code-review "Review this PR" --approve

  # Show plan only, don't execute
  sr plan code-review "Review this PR" --save-only

  # Save plan to a JSON file
  sr plan code-review "Review this PR" --save-only --output plan.json

  # Use a specific routing profile
  sr plan code-review "Review this PR" --profile premium`,
		Args: cobra.ExactArgs(2),
		RunE: runPlan,
	}

	// Define flags
	cmd.Flags().StringVarP(&planOpts.Profile, "profile", "p", skill.ProfileBalanced,
		fmt.Sprintf("routing profile: %s, %s, %s", skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium))
	cmd.Flags().BoolVar(&planOpts.Approve, "approve", false, "auto-approve and execute without confirmation")
	cmd.Flags().BoolVar(&planOpts.SaveOnly, "save-only", false, "show plan only, do not execute")
	cmd.Flags().StringVarP(&planOpts.Output, "output", "O", "", "save plan to file (JSON format)")
	cmd.Flags().BoolVar(&planOpts.NoMemory, "no-memory", false, "disable memory injection (MEMORY.md/CLAUDE.md)")

	return cmd
}

// runPlan executes the plan command.
func runPlan(cmd *cobra.Command, args []string) error {
	skillName := args[0]
	request := args[1]

	// Validate profile
	if err := validateProfile(planOpts.Profile); err != nil {
		return err
	}

	formatter := GetFormatter()
	container := GetContainer()

	if container == nil {
		return fmt.Errorf("application not initialized")
	}

	// Get skill registry and load skill
	registry := container.SkillRegistry()
	if registry == nil {
		return fmt.Errorf("skill registry not available")
	}

	// Try to find skill by ID first, then by name
	sk := registry.GetSkill(skillName)
	if sk == nil {
		sk = registry.GetSkillByName(skillName)
	}
	if sk == nil {
		return fmt.Errorf("skill not found: %s", skillName)
	}

	ctx := context.Background()

	// Load memory content (unless disabled)
	var memoryContent string
	appCtx := GetAppContext()
	memoryEnabled := appCtx != nil && appCtx.Config != nil && appCtx.Config.Memory.Enabled
	if memoryEnabled && !planOpts.NoMemory {
		cwd, err := os.Getwd()
		if err == nil {
			maxTokens := appCtx.Config.Memory.MaxTokens
			loader := infraMemory.NewLoader(maxTokens)
			mem, err := loader.Load(cwd)
			if err == nil && !mem.IsEmpty() {
				memoryContent = mem.Combined()
			}
		}
	}

	// Create planner
	planner := createPlanner(container)

	// Generate the execution plan
	plan, err := planner.GeneratePlan(ctx, sk, request, memoryContent)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	// JSON output format
	if formatter.Format() == output.FormatJSON {
		return outputPlanJSON(formatter, plan, planOpts.Output)
	}

	// Render the plan with DAG visualization
	dagRenderer := output.NewDAGRenderer(formatter)
	dagRenderer.RenderPlan(plan)

	// Save to file if requested
	if planOpts.Output != "" {
		if err := savePlanToFile(plan, planOpts.Output); err != nil {
			_ = formatter.Error("Failed to save plan: %v", err)
		} else {
			dagRenderer.RenderPlanSaved(planOpts.Output)
		}
	}

	// If save-only, exit without executing
	if planOpts.SaveOnly {
		return nil
	}

	// Prompt for approval if not auto-approved
	if !planOpts.Approve {
		dagRenderer.RenderApprovalPrompt()

		approved, err := promptApproval()
		if err != nil {
			return err
		}
		if !approved {
			_ = formatter.Warning("Execution cancelled")
			return nil
		}
	}

	// Execute the skill
	return executePlanSkill(ctx, sk, request, memoryContent, formatter)
}

// createPlanner creates a planner with available dependencies.
func createPlanner(container interface {
	CostCalculator() *provider.CostCalculator
}) *workflow.Planner {
	// Get cost calculator from container
	costCalculator := container.CostCalculator()

	// Create token estimator
	tokenEstimator, err := tokenizer.NewEstimator()
	if err != nil {
		// Fall back to nil tokenEstimator (will use defaults)
		tokenEstimator = nil
	}

	// Create planner without router (it will use placeholder model names)
	// This is acceptable since we're just generating the plan for visualization
	config := workflow.DefaultPlannerConfig()
	return workflow.NewPlanner(nil, costCalculator, tokenEstimator, config)
}

// outputPlanJSON outputs the plan as JSON.
func outputPlanJSON(formatter *output.Formatter, plan *domainWorkflow.ExecutionPlan, outputPath string) error {
	if outputPath != "" {
		return savePlanToFile(plan, outputPath)
	}
	return formatter.JSON(plan)
}

// savePlanToFile saves the execution plan to a JSON file.
func savePlanToFile(plan *domainWorkflow.ExecutionPlan, path string) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// promptApproval prompts the user for approval to proceed.
func promptApproval() (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Default to yes on empty input or explicit yes
	return input == "" || input == "y" || input == "yes", nil
}

// executePlanSkill runs the skill after plan approval.
func executePlanSkill(ctx context.Context, sk *skill.Skill, request string, memoryContent string, formatter *output.Formatter) error {
	container := GetContainer()
	if container == nil {
		return fmt.Errorf("application not initialized")
	}

	// Get a provider for execution
	providerRegistry := container.ProviderRegistry()
	providers := providerRegistry.ListProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no providers configured. Run 'sr init' to set up providers")
	}

	// Select provider based on profile
	selectedProvider := selectProvider(providers, planOpts.Profile)
	if selectedProvider == nil {
		return fmt.Errorf("no suitable provider found for profile: %s", planOpts.Profile)
	}

	// Create executor with memory content
	executorConfig := workflow.DefaultExecutorConfig()
	executorConfig.MemoryContent = memoryContent
	executor := workflow.NewExecutor(selectedProvider, executorConfig)

	// Get cost calculator for pricing
	costCalc := container.CostCalculator()

	// Execute using the standard text output (similar to run.go)
	return runSkillText(ctx, executor, sk, request, selectedProvider, formatter, costCalc)
}
