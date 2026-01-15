// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/application/workflow"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	infraMemory "github.com/jbctechsolutions/skillrunner/internal/infrastructure/memory"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// runFlags holds the flags for the run command.
type runFlags struct {
	Profile      string
	Stream       bool
	NoMemory     bool
	Resume       bool
	NoCheckpoint bool
	Force        bool
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

  # Resume from last checkpoint
  sr run long-analysis "Complex analysis" --resume

  # Run without checkpoint persistence
  sr run quick-task "Simple task" --no-checkpoint

  # Force new execution even if checkpoint exists
  sr run analysis "Data analysis" --force

Routing Profiles:
  cheap     - Prioritize cost, use local/cheaper models
  balanced  - Balance between cost and quality (default)
  premium   - Prioritize quality, use best available models

Crash Recovery:
  By default, execution state is checkpointed after each phase batch.
  Use --resume to continue from the last checkpoint if available.
  Use --no-checkpoint to disable checkpointing (for testing or short tasks).
  Use --force to start a new execution even if a checkpoint exists.

Note: Streaming mode (--stream) does not support checkpointing. Use standard
mode for long-running tasks that may need crash recovery.`,
		Args: cobra.ExactArgs(2),
		RunE: runSkill,
	}

	// Define flags
	cmd.Flags().StringVarP(&runOpts.Profile, "profile", "p", skill.ProfileBalanced,
		fmt.Sprintf("routing profile: %s, %s, %s", skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium))
	cmd.Flags().BoolVarP(&runOpts.Stream, "stream", "s", false, "enable streaming output")
	cmd.Flags().BoolVar(&runOpts.NoMemory, "no-memory", false, "disable memory injection (MEMORY.md/CLAUDE.md)")
	cmd.Flags().BoolVar(&runOpts.Resume, "resume", false, "resume from last checkpoint if available")
	cmd.Flags().BoolVar(&runOpts.NoCheckpoint, "no-checkpoint", false, "disable checkpoint persistence")
	cmd.Flags().BoolVarP(&runOpts.Force, "force", "f", false, "start new execution even if checkpoint exists")

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

	// Get a provider for execution
	providerRegistry := container.ProviderRegistry()
	providers := providerRegistry.ListProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no providers configured. Run 'sr init' to set up providers")
	}

	// Select provider based on profile
	provider := selectProvider(providers, runOpts.Profile)
	if provider == nil {
		return fmt.Errorf("no suitable provider found for profile: %s", runOpts.Profile)
	}

	ctx := context.Background()

	// Load memory content (unless disabled)
	var memoryContent string
	appCtx := GetAppContext()
	memoryEnabled := appCtx != nil && appCtx.Config != nil && appCtx.Config.Memory.Enabled
	if memoryEnabled && !runOpts.NoMemory {
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

	// Build checkpoint config
	cpConfig := workflow.CheckpointConfig{
		Enabled:   !runOpts.NoCheckpoint,
		Port:      container.WorkflowCheckpointRepository(),
		Resume:    runOpts.Resume,
		MachineID: container.MachineID(),
	}

	// Check for existing checkpoint if not resuming and not forcing
	if cpConfig.Enabled && !runOpts.Resume && !runOpts.Force && cpConfig.Port != nil {
		existingCP, _ := workflow.GetExistingCheckpoint(ctx, cpConfig.Port, sk.ID(), request)
		if existingCP != nil {
			formatter.Warning("An incomplete execution exists for this skill/input (progress: %s).", existingCP.Progress())
			formatter.Warning("Use --resume to continue, or --force to start fresh.")
			return fmt.Errorf("checkpoint exists; use --resume or --force")
		}
	}

	// JSON output for scripting (non-streaming)
	if formatter.Format() == output.FormatJSON {
		executorConfig := workflow.DefaultExecutorConfig()
		executorConfig.MemoryContent = memoryContent
		executor := workflow.NewCheckpointingExecutor(provider, executorConfig, cpConfig)
		return runSkillJSON(ctx, executor, sk, request, provider)
	}

	// Streaming output mode
	// Note: Checkpointing is not supported in streaming mode. For long-running
	// tasks that need crash recovery, use standard (non-streaming) mode.
	if runOpts.Stream {
		streamingConfig := workflow.DefaultExecutorConfig()
		streamingConfig.MemoryContent = memoryContent
		streamingExecutor := workflow.NewStreamingExecutor(provider, streamingConfig)
		return runSkillStreaming(ctx, streamingExecutor, sk, request, provider, formatter)
	}

	// Standard text output with progress display
	executorConfig := workflow.DefaultExecutorConfig()
	executorConfig.MemoryContent = memoryContent
	executor := workflow.NewCheckpointingExecutor(provider, executorConfig, cpConfig)
	return runSkillText(ctx, executor, sk, request, provider, formatter)
}

// selectProvider chooses a provider based on the routing profile.
func selectProvider(providers []ports.ProviderPort, profile string) ports.ProviderPort {
	if len(providers) == 0 {
		return nil
	}

	// Sort providers based on profile preference
	switch profile {
	case skill.ProfileCheap:
		// Prefer local providers for cheap profile
		for _, p := range providers {
			if p.Info().IsLocal {
				return p
			}
		}
		// Fall back to first available
		return providers[0]

	case skill.ProfilePremium:
		// Prefer cloud providers for premium profile
		for _, p := range providers {
			if !p.Info().IsLocal {
				return p
			}
		}
		// Fall back to first available
		return providers[0]

	default: // balanced
		// Return first available provider
		return providers[0]
	}
}

// runSkillJSON executes the skill and outputs results as JSON.
func runSkillJSON(ctx context.Context, executor workflow.Executor, sk *skill.Skill, request string, provider ports.ProviderPort) error {
	formatter := GetFormatter()

	result, err := executor.Execute(ctx, sk, request)
	if err != nil {
		errorResult := map[string]any{
			"skill":   sk.Name(),
			"status":  "error",
			"error":   err.Error(),
			"profile": runOpts.Profile,
		}
		return formatter.JSON(errorResult)
	}

	// Build phase results for JSON output
	phaseResults := make([]map[string]any, 0, len(result.PhaseResults))
	for _, pr := range result.PhaseResults {
		phaseResults = append(phaseResults, map[string]any{
			"id":            pr.PhaseID,
			"name":          pr.PhaseName,
			"status":        string(pr.Status),
			"duration_ms":   pr.Duration.Milliseconds(),
			"input_tokens":  pr.InputTokens,
			"output_tokens": pr.OutputTokens,
			"model":         pr.ModelUsed,
		})
	}

	jsonResult := map[string]any{
		"skill":        sk.Name(),
		"status":       string(result.Status),
		"profile":      runOpts.Profile,
		"provider":     provider.Info().Name,
		"duration_ms":  result.Duration.Milliseconds(),
		"total_tokens": result.TotalTokens,
		"phases":       phaseResults,
		"final_output": result.FinalOutput,
		"streaming":    runOpts.Stream,
	}

	if result.Error != nil {
		jsonResult["error"] = result.Error.Error()
	}

	return formatter.JSON(jsonResult)
}

// runSkillStreaming executes the skill with streaming output.
func runSkillStreaming(ctx context.Context, executor workflow.StreamingExecutor, sk *skill.Skill, request string, _ ports.ProviderPort, formatter *output.Formatter) error {
	// Create streaming output handler
	streamOut := output.NewStreamingOutput(
		output.WithStreamingColor(formatter.Format() != output.FormatJSON),
		output.WithShowTokenCounts(true),
		output.WithShowPhaseInfo(true),
	)

	phases := sk.Phases()
	streamOut.StartWorkflow(sk.Name(), sk.Version(), len(phases))

	// Create streaming callback
	callback := func(event workflow.StreamEvent) error {
		switch event.Type {
		case workflow.EventPhaseStarted:
			streamOut.StartPhase(event.PhaseID, event.PhaseName, event.PhaseIndex)
		case workflow.EventPhaseProgress:
			if event.Content != "" {
				streamOut.WriteChunk(event.Content)
			}
		case workflow.EventPhaseCompleted:
			streamOut.CompletePhase(event.InputTokens, event.OutputTokens, "")
		case workflow.EventPhaseFailed:
			streamOut.FailPhase(event.Error)
		case workflow.EventTokenUpdate:
			streamOut.UpdateTokens(event.InputTokens, event.OutputTokens)
		case workflow.EventWorkflowCompleted:
			// Final completion is handled after the result is returned
		}
		return nil
	}

	// Execute with streaming
	result, err := executor.ExecuteWithStreaming(ctx, sk, request, callback)
	if err != nil {
		streamOut.CompleteWorkflow(false)
		return err
	}

	// Complete workflow
	streamOut.CompleteWorkflow(result.Status == workflow.PhaseStatusCompleted)

	return nil
}

// runSkillText executes the skill with text output and progress display.
func runSkillText(ctx context.Context, executor workflow.Executor, sk *skill.Skill, request string, provider ports.ProviderPort, formatter *output.Formatter) error {
	// Display execution header
	formatter.Header("Skill Execution")
	formatter.Item("Skill", sk.Name())
	formatter.Item("Version", sk.Version())
	formatter.Item("Profile", runOpts.Profile)
	formatter.Item("Provider", provider.Info().Name)
	if runOpts.Stream {
		formatter.Item("Mode", "streaming")
	}
	formatter.Println("")

	// Display the request (truncate if too long)
	requestDisplay := request
	if len(requestDisplay) > 100 {
		requestDisplay = requestDisplay[:97] + "..."
	}
	formatter.Item("Request", requestDisplay)
	formatter.Println("")

	// Show phase information
	phases := sk.Phases()
	formatter.SubHeader(fmt.Sprintf("Phases (%d)", len(phases)))
	for i, phase := range phases {
		deps := ""
		if len(phase.DependsOn) > 0 {
			deps = fmt.Sprintf(" (depends: %s)", strings.Join(phase.DependsOn, ", "))
		}
		formatter.BulletItem(fmt.Sprintf("%d. %s%s", i+1, phase.Name, deps))
	}
	formatter.Println("")

	// Start spinner for execution
	spinner := output.NewSpinner("Executing workflow...")
	spinner.Start()

	// Execute the workflow
	startTime := time.Now()
	result, err := executor.Execute(ctx, sk, request)
	executionTime := time.Since(startTime)

	spinner.Stop()

	if err != nil {
		formatter.Error("Execution failed: %v", err)
		return err
	}

	// Display results
	formatter.Println("")
	formatter.Header("Execution Results")

	// Phase results
	formatter.SubHeader("Phase Results")
	displayPhaseResults(formatter, result)
	formatter.Println("")

	// Summary statistics
	formatter.SubHeader("Summary")
	formatter.Item("Status", formatStatus(result.Status))
	formatter.Item("Total Duration", formatDuration(executionTime))
	formatter.Item("Total Tokens", fmt.Sprintf("%d", result.TotalTokens))
	formatter.Println("")

	// Final output
	if result.FinalOutput != "" {
		formatter.SubHeader("Output")
		formatter.Println("")
		// Print output with proper formatting
		outputLines := strings.Split(result.FinalOutput, "\n")
		for _, line := range outputLines {
			formatter.Println("%s", line)
		}
	}

	// Success message
	if result.Status == workflow.PhaseStatusCompleted {
		formatter.Println("")
		formatter.Success("Skill execution completed successfully")
	} else if result.Error != nil {
		formatter.Println("")
		formatter.Error("Skill execution failed: %v", result.Error)
	}

	return nil
}

// displayPhaseResults displays the results of each phase in a table.
func displayPhaseResults(formatter *output.Formatter, result *workflow.ExecutionResult) {
	// Sort phase results by completion order
	sortedPhases := make([]*workflow.PhaseResult, 0, len(result.PhaseResults))
	for _, pr := range result.PhaseResults {
		sortedPhases = append(sortedPhases, pr)
	}
	sort.Slice(sortedPhases, func(i, j int) bool {
		return sortedPhases[i].StartTime.Before(sortedPhases[j].StartTime)
	})

	// Create table data
	tableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: "Phase", Width: 20, Align: output.AlignLeft},
			{Header: "Status", Width: 10, Align: output.AlignLeft},
			{Header: "Duration", Width: 12, Align: output.AlignRight},
			{Header: "Tokens", Width: 10, Align: output.AlignRight},
			{Header: "Model", Width: 20, Align: output.AlignLeft},
		},
		Rows: make([][]string, 0, len(sortedPhases)),
	}

	for _, pr := range sortedPhases {
		totalTokens := pr.InputTokens + pr.OutputTokens
		tableData.Rows = append(tableData.Rows, []string{
			pr.PhaseName,
			formatStatus(pr.Status),
			formatDuration(pr.Duration),
			fmt.Sprintf("%d", totalTokens),
			pr.ModelUsed,
		})
	}

	_ = formatter.Table(tableData)
}

// formatStatus returns a human-readable status string.
func formatStatus(status workflow.PhaseStatus) string {
	switch status {
	case workflow.PhaseStatusCompleted:
		return "completed"
	case workflow.PhaseStatusFailed:
		return "failed"
	case workflow.PhaseStatusRunning:
		return "running"
	case workflow.PhaseStatusSkipped:
		return "skipped"
	case workflow.PhaseStatusPending:
		return "pending"
	default:
		return string(status)
	}
}

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// validateProfile checks if the profile is valid.
func validateProfile(profile string) error {
	profile = strings.ToLower(strings.TrimSpace(profile))
	validProfiles := []string{skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium}

	if slices.Contains(validProfiles, profile) {
		return nil
	}

	return fmt.Errorf("invalid profile %q: must be one of %s", profile, strings.Join(validProfiles, ", "))
}

// init registers the run command with the root command.
func init() {
	// This will be called when the package is imported
}
