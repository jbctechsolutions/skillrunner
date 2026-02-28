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
	"github.com/jbctechsolutions/skillrunner/internal/domain/budget"
	"github.com/jbctechsolutions/skillrunner/internal/domain/complexity"
	domainExport "github.com/jbctechsolutions/skillrunner/internal/domain/export"
	"github.com/jbctechsolutions/skillrunner/internal/domain/isolation"
	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	fileContext "github.com/jbctechsolutions/skillrunner/internal/infrastructure/context"
	infraGit "github.com/jbctechsolutions/skillrunner/internal/infrastructure/git"
	infraMemory "github.com/jbctechsolutions/skillrunner/internal/infrastructure/memory"
	infraStorage "github.com/jbctechsolutions/skillrunner/internal/infrastructure/storage"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// runFlags holds the flags for the run command.
type runFlags struct {
	Profile         string
	Stream          bool
	NoMemory        bool
	Resume          bool
	NoCheckpoint    bool
	Force           bool
	AutoApprove     bool    // skip tool permission prompts (-y / --yes)
	Budget          float64 // per-workflow spend cap in USD (0 = use global config)
	SkipEscalation  bool    // disable auto-escalation on low-confidence responses
	SkipReview      bool    // disable post-completion review phases
	ExportFormat    string  // export result in this format (json, claude-code, aider, cursor)
	Isolate         bool    // run in a git worktree, show diff and prompt merge/discard
	profileExplicit bool    // set to true when --profile was provided by the user
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
		fmt.Sprintf("routing profile: %s, %s, %s (default: auto-detected from complexity)", skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium))
	// Track whether the user explicitly set --profile so we know not to override it.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		runOpts.profileExplicit = cmd.Flags().Changed("profile")
		return nil
	}
	cmd.Flags().BoolVarP(&runOpts.Stream, "stream", "s", false, "enable streaming output")
	cmd.Flags().BoolVar(&runOpts.NoMemory, "no-memory", false, "disable memory injection (MEMORY.md/CLAUDE.md)")
	cmd.Flags().BoolVar(&runOpts.Resume, "resume", false, "resume from last checkpoint if available")
	cmd.Flags().BoolVar(&runOpts.NoCheckpoint, "no-checkpoint", false, "disable checkpoint persistence")
	cmd.Flags().BoolVarP(&runOpts.Force, "force", "f", false, "start new execution even if checkpoint exists")
	cmd.Flags().BoolVarP(&runOpts.AutoApprove, "yes", "y", false, "auto-approve MCP tool execution (skip permission prompts)")
	cmd.Flags().Float64Var(&runOpts.Budget, "budget", 0, "per-workflow spend cap in USD (overrides global config)")
	cmd.Flags().BoolVar(&runOpts.SkipEscalation, "skip-escalation", false, "disable auto-escalation on low-confidence responses")
	cmd.Flags().BoolVar(&runOpts.SkipReview, "skip-review", false, "disable post-completion review on phases that declare it")
	cmd.Flags().StringVar(&runOpts.ExportFormat, "export-format", "", "export result format: json, claude-code, aider, cursor")
	cmd.Flags().BoolVar(&runOpts.Isolate, "isolate", false, "run in a git worktree; show diff and prompt merge or discard")

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

	// Auto-detect complexity profile unless the user set --profile explicitly.
	if !runOpts.profileExplicit {
		analyzer := complexity.NewAnalyzer()
		score, _ := analyzer.Analyze(request)
		autoProfile := score.Profile()
		if autoProfile != runOpts.Profile {
			runOpts.Profile = autoProfile
		}
		formatter.Info("Complexity: %.2f → %s profile", float64(score), runOpts.Profile)
	}

	// Select provider based on profile
	provider := selectProvider(providers, runOpts.Profile)
	if provider == nil {
		return fmt.Errorf("no suitable provider found for profile: %s", runOpts.Profile)
	}

	ctx := context.Background()

	// Show pre-execution budget alerts (non-fatal — informational only).
	showBudgetAlerts(ctx, formatter, request, runOpts.Profile)

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

	// Worktree isolation: create a temporary git worktree for the execution.
	var isolMgr *infraGit.IsolationManager
	var isolSess *isolation.Session
	if runOpts.Isolate {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fmt.Errorf("--isolate requires a working directory: %w", cwdErr)
		}
		im, imErr := infraGit.NewIsolationManager("")
		if imErr != nil {
			return fmt.Errorf("--isolate requires git to be installed: %w", imErr)
		}
		wm, wmErr := infraGit.NewWorktreeManager()
		if wmErr != nil {
			return fmt.Errorf("--isolate requires git: %w", wmErr)
		}
		repoRoot, rootErr := wm.GetRepositoryRoot(ctx, cwd)
		if rootErr != nil {
			return fmt.Errorf("--isolate requires a git repository: %w", rootErr)
		}
		sess, sessErr := im.Setup(ctx, repoRoot, sk.Name())
		if sessErr != nil {
			return fmt.Errorf("failed to create isolation worktree: %w", sessErr)
		}
		isolMgr = im
		isolSess = sess
		formatter.Info("Isolation worktree: %s", sess.WorktreePath)
		// Inject worktree path as context so LLM tools operate on the isolated copy.
		if memoryContent != "" {
			memoryContent += "\n\n"
		}
		memoryContent += fmt.Sprintf("Working directory for file operations: %s", sess.WorktreePath)
	}

	// Check tool permissions if the skill declares MCP tools
	if sk.HasTools() {
		mcpReg := container.MCPRegistry()
		var toolInfos []fileContext.ToolInfo
		if mcpReg != nil {
			// Collect descriptions from the registry for declared tools
			mcpTools, _ := mcpReg.GetAllTools(ctx)
			descByName := make(map[string]string, len(mcpTools))
			for _, t := range mcpTools {
				descByName[t.FullName()] = t.Description()
			}
			for _, name := range sk.Tools() {
				toolInfos = append(toolInfos, fileContext.ToolInfo{
					Name:        name,
					Description: descByName[name],
				})
			}
		} else {
			for _, name := range sk.Tools() {
				toolInfos = append(toolInfos, fileContext.ToolInfo{Name: name})
			}
		}
		prompter := fileContext.NewToolPermissionPrompt(runOpts.AutoApprove)
		if err := prompter.PromptForTools(toolInfos); err != nil {
			return fmt.Errorf("tool permission denied: %w", err)
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

	// Get cost calculator for pricing
	costCalc := container.CostCalculator()

	// Budget check: enforce global and per-workflow limits
	if err := checkBudget(ctx, formatter, runOpts.Budget); err != nil {
		return err
	}

	// Build executor config (shared base)
	baseConfig := workflow.DefaultExecutorConfig()
	baseConfig.MemoryContent = memoryContent
	baseConfig.AutoApproveTools = runOpts.AutoApprove
	baseConfig.RoutingProfile = runOpts.Profile
	baseConfig.SkillID = sk.ID()
	baseConfig.SkillName = sk.Name()
	if appCtx != nil && appCtx.Config != nil {
		baseConfig.CompressionEnabled = appCtx.Config.Context.CompressionEnabled
		// Resolve skill-level model hints for this skill
		if hints := appCtx.Config.Routing.SkillModelHints; len(hints) > 0 {
			skillID := sk.ID()
			skillName := sk.Name()
			if perSkill, ok := hints[skillID]; ok {
				baseConfig.ModelHints = perSkill
			} else if perSkill, ok := hints[skillName]; ok {
				baseConfig.ModelHints = perSkill
			}
		}
		baseConfig.ConfidenceThresholds = appCtx.Config.Routing.ConfidenceThreshold
	}
	baseConfig.SkipConfidenceEscalation = runOpts.SkipEscalation
	baseConfig.SkipPostCompletionReview = runOpts.SkipReview
	if outcomeRepo := container.OutcomeRepository(); outcomeRepo != nil {
		baseConfig.OutcomePort = outcomeRepo
	}
	if mcpReg := container.MCPRegistry(); mcpReg != nil {
		baseConfig.MCPRegistry = mcpReg
	}
	baseConfig.AllowedTools = sk.Tools()

	// Validate export format early
	if runOpts.ExportFormat != "" {
		if !domainExport.Format(runOpts.ExportFormat).IsValid() {
			return fmt.Errorf("invalid --export-format %q: must be one of json, claude-code, aider, cursor", runOpts.ExportFormat)
		}
	}

	// JSON output for scripting (non-streaming)
	if formatter.Format() == output.FormatJSON {
		executor := workflow.NewCheckpointingExecutor(provider, baseConfig, cpConfig)
		execErr := runSkillJSON(ctx, executor, sk, request, provider, costCalc)
		if isolSess != nil {
			handleIsolationResult(ctx, formatter, isolMgr, isolSess, execErr)
		}
		return execErr
	}

	// Streaming output mode
	// Note: Checkpointing is not supported in streaming mode. For long-running
	// tasks that need crash recovery, use standard (non-streaming) mode.
	if runOpts.Stream {
		streamingExecutor := workflow.NewStreamingExecutor(provider, baseConfig)
		execErr := runSkillStreaming(ctx, streamingExecutor, sk, request, provider, formatter)
		if isolSess != nil {
			handleIsolationResult(ctx, formatter, isolMgr, isolSess, execErr)
		}
		return execErr
	}

	// Standard text output with progress display
	executor := workflow.NewCheckpointingExecutor(provider, baseConfig, cpConfig)
	result, execErr := runSkillTextWithResult(ctx, executor, sk, request, provider, formatter, costCalc)
	if isolSess != nil {
		handleIsolationResult(ctx, formatter, isolMgr, isolSess, execErr)
	}

	// Export result if requested
	if execErr == nil && result != nil && runOpts.ExportFormat != "" {
		if exportErr := exportResult(result, sk.Name(), sk.ID(), request, runOpts.Profile, runOpts.ExportFormat, formatter); exportErr != nil {
			formatter.Warning("Export failed: %v", exportErr)
		}
	}
	return execErr
}

// exportResult serialises the execution result in the requested format and prints to stdout.
func exportResult(result *workflow.ExecutionResult, skillName, skillID, input, profile, format string, formatter *output.Formatter) error {
	we := &domainExport.WorkflowExport{
		SkillID:     skillID,
		SkillName:   skillName,
		Input:       input,
		Output:      result.FinalOutput,
		Profile:     profile,
		Status:      string(result.Status),
		TotalCost:   result.TotalCost,
		TotalTokens: result.TotalTokens,
		Duration:    fmt.Sprintf("%d", result.Duration.Milliseconds()),
		ExportedAt:  result.EndTime,
	}
	for _, pr := range result.PhaseResults {
		we.Phases = append(we.Phases, domainExport.PhaseExport{
			ID:     pr.PhaseID,
			Name:   pr.PhaseName,
			Model:  pr.ModelUsed,
			Output: pr.Output,
			Tokens: pr.InputTokens + pr.OutputTokens,
			Cost:   pr.Cost,
			Status: string(pr.Status),
		})
	}

	data, err := we.Marshal(domainExport.Format(format))
	if err != nil {
		return err
	}

	formatter.Println("")
	formatter.SubHeader(fmt.Sprintf("Export (%s)", format))
	fmt.Println(string(data))
	return nil
}

// handleIsolationResult presents the worktree diff and prompts the user to merge or discard.
// In JSON output mode, changes are auto-discarded to avoid blocking non-interactive scripts.
func handleIsolationResult(ctx context.Context, formatter *output.Formatter, im *infraGit.IsolationManager, sess *isolation.Session, execErr error) {
	if execErr != nil {
		_ = formatter.Warning("Execution failed — discarding worktree (%s)", sess.WorktreePath)
		_ = im.Discard(ctx, sess)
		return
	}

	diff, err := im.Diff(ctx, sess)
	if err != nil || strings.TrimSpace(diff) == "" {
		_ = formatter.Info("No file changes detected in isolation worktree.")
		_ = im.Discard(ctx, sess)
		return
	}

	// In JSON mode, auto-discard to avoid blocking non-interactive scripts.
	if formatter.Format() == output.FormatJSON {
		_ = im.Discard(ctx, sess)
		return
	}

	_ = formatter.Println("")
	_ = formatter.SubHeader("Isolation Diff")
	_ = formatter.Println(diff)

	// Prompt user to apply or discard
	fmt.Print("Apply changes to working tree? [y/N] ")
	var answer string
	if _, scanErr := fmt.Scanln(&answer); scanErr != nil {
		answer = "n"
	}

	if strings.ToLower(strings.TrimSpace(answer)) == "y" {
		if applyErr := im.Apply(ctx, sess); applyErr != nil {
			_ = formatter.Error("Failed to apply changes: %v", applyErr)
		} else {
			_ = formatter.Success("Changes applied to working tree.")
		}
		_ = im.Discard(ctx, sess)
	} else {
		_ = formatter.Info("Changes discarded. Worktree removed.")
		_ = im.Discard(ctx, sess)
	}
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
func runSkillJSON(ctx context.Context, executor workflow.Executor, sk *skill.Skill, request string, prov ports.ProviderPort, costCalc *provider.CostCalculator) error {
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

	// Calculate costs for each phase using model pricing
	calculateCostsForResult(result, costCalc)

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
			"cost":          pr.Cost,
		})
	}

	jsonResult := map[string]any{
		"skill":        sk.Name(),
		"status":       string(result.Status),
		"profile":      runOpts.Profile,
		"provider":     prov.Info().Name,
		"duration_ms":  result.Duration.Milliseconds(),
		"total_tokens": result.TotalTokens,
		"total_cost":   result.TotalCost,
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
// runSkillTextWithResult executes the skill with text output and returns both the result and any error.
func runSkillTextWithResult(ctx context.Context, executor workflow.Executor, sk *skill.Skill, request string, prov ports.ProviderPort, formatter *output.Formatter, costCalc *provider.CostCalculator) (*workflow.ExecutionResult, error) {
	return runSkillTextImpl(ctx, executor, sk, request, prov, formatter, costCalc)
}

func runSkillText(ctx context.Context, executor workflow.Executor, sk *skill.Skill, request string, prov ports.ProviderPort, formatter *output.Formatter, costCalc *provider.CostCalculator) error {
	_, err := runSkillTextImpl(ctx, executor, sk, request, prov, formatter, costCalc)
	return err
}

func runSkillTextImpl(ctx context.Context, executor workflow.Executor, sk *skill.Skill, request string, prov ports.ProviderPort, formatter *output.Formatter, costCalc *provider.CostCalculator) (*workflow.ExecutionResult, error) {
	// Display execution header
	formatter.Header("Skill Execution")
	formatter.Item("Skill", sk.Name())
	formatter.Item("Version", sk.Version())
	formatter.Item("Profile", runOpts.Profile)
	formatter.Item("Provider", prov.Info().Name)
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
		return nil, err
	}

	// Calculate costs for each phase using model pricing
	calculateCostsForResult(result, costCalc)

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
	formatter.Item("Total Cost", formatCost(result.TotalCost))
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

	return result, nil
}

// displayPhaseResults displays the results of each phase in a table with cost breakdown.
func displayPhaseResults(formatter *output.Formatter, result *workflow.ExecutionResult) {
	// Sort phase results by completion order
	sortedPhases := make([]*workflow.PhaseResult, 0, len(result.PhaseResults))
	for _, pr := range result.PhaseResults {
		sortedPhases = append(sortedPhases, pr)
	}
	sort.Slice(sortedPhases, func(i, j int) bool {
		return sortedPhases[i].StartTime.Before(sortedPhases[j].StartTime)
	})

	// Create table data with Cost column
	tableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: "Phase", Width: 15, Align: output.AlignLeft},
			{Header: "Model", Width: 25, Align: output.AlignLeft}, // 25 chars to fit model names like "claude-opus-4-5-20251101"
			{Header: "Time", Width: 8, Align: output.AlignRight},
			{Header: "Tokens", Width: 8, Align: output.AlignRight},
			{Header: "Cost", Width: 10, Align: output.AlignRight}, // 10 chars for costs like "$0.0175"
			{Header: "Status", Width: 6, Align: output.AlignCenter},
		},
		Rows: make([][]string, 0, len(sortedPhases)+3), // +3 for separator, total, vs premium
	}

	// Track totals for input/output tokens separately (for premium calculation)
	var totalInputTokens, totalOutputTokens int

	for _, pr := range sortedPhases {
		totalTokens := pr.InputTokens + pr.OutputTokens
		totalInputTokens += pr.InputTokens
		totalOutputTokens += pr.OutputTokens

		tableData.Rows = append(tableData.Rows, []string{
			pr.PhaseName,
			pr.ModelUsed,
			formatDuration(pr.Duration),
			fmt.Sprintf("%d", totalTokens),
			formatCost(pr.Cost),
			formatStatusIcon(pr.Status),
		})
	}

	// Add separator row
	tableData.Rows = append(tableData.Rows, []string{"───────────────", "─────────────────────────", "────────", "────────", "──────────", "──────"})

	// Add TOTAL row
	tableData.Rows = append(tableData.Rows, []string{
		"TOTAL",
		"",
		formatDuration(result.Duration),
		fmt.Sprintf("%d", result.TotalTokens),
		formatCost(result.TotalCost),
		"",
	})

	// Calculate premium equivalent cost (using Claude Sonnet 3.5 pricing as reference)
	// Get pricing from DefaultModelPricing for consistency
	premiumRate := getPremiumModelPricing()
	premiumCost := (float64(totalInputTokens) / 1000.0 * premiumRate.InputRate) +
		(float64(totalOutputTokens) / 1000.0 * premiumRate.OutputRate)

	// Calculate savings percentage
	savingsPercent := 0.0
	if premiumCost > 0 {
		savingsPercent = ((premiumCost - result.TotalCost) / premiumCost) * 100
	}

	// Add vs premium comparison row
	savingsDisplay := ""
	if savingsPercent > 0 {
		savingsDisplay = fmt.Sprintf("-%.0f%%", savingsPercent)
	} else if savingsPercent < 0 {
		savingsDisplay = fmt.Sprintf("+%.0f%%", -savingsPercent)
	}

	tableData.Rows = append(tableData.Rows, []string{
		"vs premium",
		"",
		"",
		"",
		formatCost(premiumCost),
		savingsDisplay,
	})

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

// formatCost formats a cost in USD with appropriate precision.
func formatCost(cost float64) string {
	if cost == 0 {
		return "$0.00"
	}
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// formatStatusIcon returns a status icon for display.
func formatStatusIcon(status workflow.PhaseStatus) string {
	switch status {
	case workflow.PhaseStatusCompleted:
		return "✓"
	case workflow.PhaseStatusFailed:
		return "✗"
	case workflow.PhaseStatusSkipped:
		return "○"
	case workflow.PhaseStatusRunning:
		return "◐"
	case workflow.PhaseStatusPending:
		return "·"
	default:
		return "?"
	}
}

// getPremiumModelPricing returns the pricing for the premium reference model (Claude Sonnet 3.5).
// This is used for the "vs premium" comparison in the output.
func getPremiumModelPricing() provider.ModelCostRate {
	// Default fallback (Claude Sonnet 3.5 pricing: $3/MTok input, $15/MTok output)
	defaultRate := provider.ModelCostRate{
		ModelID:    "claude-3-5-sonnet-20241022",
		Provider:   provider.ProviderAnthropic,
		InputRate:  0.003, // $3/MTok = 0.003 per 1K
		OutputRate: 0.015, // $15/MTok = 0.015 per 1K
		IsLocal:    false,
	}

	// Try to get from DefaultModelPricing for consistency
	for _, rate := range provider.DefaultModelPricing() {
		if rate.ModelID == "claude-3-5-sonnet-20241022" {
			return rate
		}
	}

	return defaultRate
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

// calculateCostsForResult populates cost data for each phase in the execution result.
// It uses the provided CostCalculator to look up model pricing.
// If costCalc is nil, costs will remain at zero.
func calculateCostsForResult(result *workflow.ExecutionResult, costCalc *provider.CostCalculator) {
	if costCalc == nil || result == nil {
		return
	}

	// Calculate cost for each phase
	var totalCost float64
	for _, pr := range result.PhaseResults {
		breakdown := costCalc.CalculateOrZero(pr.ModelUsed, pr.InputTokens, pr.OutputTokens)
		pr.Cost = breakdown.TotalCost
		totalCost += breakdown.TotalCost
	}

	result.TotalCost = totalCost
}

// checkBudget enforces configured budget limits before execution.
// It warns at 80% of any limit and blocks at 100%.
// workflowCap > 0 is treated as a per-workflow daily cap for this check.
func checkBudget(ctx context.Context, formatter *output.Formatter, workflowCap float64) error {
	appContainer := GetContainer()
	if appContainer == nil {
		return nil
	}
	appCtxVal := GetAppContext()
	if appCtxVal == nil || appCtxVal.Config == nil {
		return nil
	}

	budgetCfg := appCtxVal.Config.Budget
	limits := budget.NewLimits(budgetCfg.DailyLimit, budgetCfg.MonthlyLimit)
	if workflowCap > 0 {
		limits.DailyLimit = workflowCap
	}
	if !limits.Enabled() {
		return nil
	}

	repo, err := infraStorage.NewBudgetRepository(appContainer.DB())
	if err != nil {
		formatter.Warning("Budget check unavailable: %v", err)
		return nil
	}

	usage, err := repo.GetUsage(ctx)
	if err != nil {
		formatter.Warning("Could not retrieve budget usage: %v", err)
		return nil
	}

	if limits.DailyLimit > 0 {
		pct := (usage.DailySpend / limits.DailyLimit) * 100
		formatter.Item("Daily budget", fmt.Sprintf("$%.4f / $%.2f (%.0f%%)", usage.DailySpend, limits.DailyLimit, pct))
	}
	if limits.MonthlyLimit > 0 {
		pct := (usage.MonthlySpend / limits.MonthlyLimit) * 100
		formatter.Item("Monthly budget", fmt.Sprintf("$%.4f / $%.2f (%.0f%%)", usage.MonthlySpend, limits.MonthlyLimit, pct))
	}

	if err := budget.CheckLimit(limits, usage, 0); err != nil {
		switch err {
		case budget.ErrBudgetExceeded:
			formatter.Println("")
			return fmt.Errorf("budget limit exceeded — use 'sr config set budget.daily_limit' to adjust or --budget=0 to disable")
		case budget.ErrBudgetWarning:
			formatter.Warning("Budget warning: spending is at ≥80%% of configured limit")
		}
	}
	return nil
}

// showBudgetAlerts prints pre-execution budget alerts and cost estimates.
// Non-fatal — execution proceeds regardless.
func showBudgetAlerts(ctx context.Context, formatter *output.Formatter, request, profile string) {
	appCtxVal := GetAppContext()
	appContainer := GetContainer()
	if appCtxVal == nil || appContainer == nil {
		return
	}

	budgetCfg := appCtxVal.Config.Budget
	limits := budget.NewLimits(budgetCfg.DailyLimit, budgetCfg.MonthlyLimit)
	if !limits.Enabled() {
		return
	}

	repo, err := infraStorage.NewBudgetRepository(appContainer.DB())
	if err != nil {
		return
	}
	usage, err := repo.GetUsage(ctx)
	if err != nil {
		return
	}

	// Show threshold alerts
	for _, alert := range budget.CheckAlerts(limits, usage) {
		if alert.IsError() {
			formatter.Error("Budget Alert: %s", alert.Message())
		} else {
			formatter.Warning("Budget Alert: %s", alert.Message())
		}
	}

	// Show cost estimate + savings tip for non-cheap profiles
	est := budget.EstimateCost(len(request), profile)
	if est.EstimatedUSD > 0 {
		formatter.Info("Estimated cost: ~$%.4f", est.EstimatedUSD)
		if profile != "cheap" && est.CheapSavingsPct > 10 {
			formatter.Info("Tip: --profile cheap saves ~%.0f%% (~$%.4f)", est.CheapSavingsPct, est.CheapSavingsUSD)
		}
	}
}

// init registers the run command with the root command.
func init() {
	// This will be called when the package is imported
}
