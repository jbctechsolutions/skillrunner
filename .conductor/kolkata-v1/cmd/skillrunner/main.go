package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/cache"
	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/jbctechsolutions/skillrunner/internal/converter"
	"github.com/jbctechsolutions/skillrunner/internal/docker"
	"github.com/jbctechsolutions/skillrunner/internal/engine"
	"github.com/jbctechsolutions/skillrunner/internal/importer"
	"github.com/jbctechsolutions/skillrunner/internal/llm"
	"github.com/jbctechsolutions/skillrunner/internal/marketplace"
	"github.com/jbctechsolutions/skillrunner/internal/orchestration"
	"github.com/jbctechsolutions/skillrunner/internal/routing"
	runreport "github.com/jbctechsolutions/skillrunner/internal/run"
	"github.com/jbctechsolutions/skillrunner/internal/skills"
	"github.com/jbctechsolutions/skillrunner/internal/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "sr",
	Short:   "Skillrunner - Orchestrate AI development workflows",
	Long:    `Skillrunner helps you execute structured AI-powered development tasks by converting natural language requests into multi-step workflows that tools like Continue can execute.`,
	Version: version,
}

var runCmd = &cobra.Command{
	Use:   "run <agent_name> <request>",
	Short: "Run an agent or workflow with a natural language request",
	Long: `Run an agent or workflow with a natural language request.

Examples:
  sr run test "hello world"
  sr run backend-architect "add user model with authentication"
  sr run backend-architect "implement OAuth" --model gpt-4`,
	Args: cobra.ExactArgs(2),
	RunE: runSkill,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available agents and workflows",
	Long:  `Shows all agents and workflows that can be executed with the 'sr run' command.`,
	RunE:  listSkills,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system status",
	Long:  `Displays information about the Skillrunner system including available agents/workflows, configured models, and system health.`,
	RunE:  showStatus,
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for skills in the HuggingFace marketplace",
	Long:  `Search for available skills in the HuggingFace marketplace.`,
	Args:  cobra.ExactArgs(1),
	RunE:  searchMarketplace,
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <skill-id>",
	Short: "Inspect a marketplace skill",
	Long:  `Get detailed information about a specific marketplace skill.`,
	Args:  cobra.ExactArgs(1),
	RunE:  inspectSkill,
}

var askCmd = &cobra.Command{
	Use:   "ask <skill-id> <question>",
	Short: "Ask a question using a marketplace skill with any model",
	Long: `Ask a question using a marketplace skill's expertise with intelligent model routing.

Examples:
  sr ask avl-expert "How do I design audio for a 500 person venue?"
  sr ask business-analyst "What KPIs should I track for SaaS?"
  sr ask avl-expert "Calculate power requirements for LED wall" --prefer-local

This uses the Model Router to select the best model (Ollama → Anthropic) for cost savings.`,
	Args: cobra.ExactArgs(2),
	RunE: askMarketplaceSkill,
}

var importCmd = &cobra.Command{
	Use:   "import <source>",
	Short: "Import an agent or skill from a marketplace, local path, or git repository",
	Long: `Import an agent or skill from various sources. The source type is automatically detected.
Supports both SKILL.md (marketplace skills) and AGENT.md (agents) files.

Examples:
  # Import from web marketplace
  sr import https://claudeskillshub.org/skills/my-skill.json
  sr import https://example.com/skills/SKILL.md
  sr import https://example.com/agents/AGENT.md

  # Import from local filesystem
  sr import ~/my-agents/custom-agent
  sr import /path/to/agent/AGENT.md
  sr import /path/to/skills/SKILL.md

  # Import from git repository
  sr import https://github.com/user/agent-repo.git
  sr import git@github.com:user/skill-repo.git

The agent or skill will be cached locally at ~/.skillrunner/marketplace/`,
	Args: cobra.ExactArgs(1),
	RunE: importSkill,
}

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an imported agent or skill from its source",
	Long: `Update an imported agent or skill by re-fetching from its original source.

Examples:
  sr update my-agent
  sr update my-skill
  sr update avl-expert`,
	Args: cobra.ExactArgs(1),
	RunE: updateSkill,
}

var importListCmd = &cobra.Command{
	Use:   "imported",
	Short: "List all imported agents and skills",
	Long:  `Shows all agents and skills that have been imported from various sources.`,
	RunE:  listImportedSkills,
}

var importRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove an imported agent or skill",
	Long: `Remove an imported agent or skill from the local cache.

Examples:
  sr remove my-agent
  sr remove my-skill`,
	Args: cobra.ExactArgs(1),
	RunE: removeSkill,
}

var convertCmd = &cobra.Command{
	Use:   "convert <input-file-or-directory> <output-file>",
	Short: "Convert skill format between Claude Code and Skillrunner",
	Long: `Convert skill YAML format bidirectionally:
- Claude Code marketplace format → Skillrunner orchestrated format
- Skillrunner orchestrated format → Claude Code marketplace format

The tool auto-detects the input format and converts to the other format.

Input can be:
- A local directory containing SKILL.md or AGENT.md (e.g., ../business-analyst/)
- A local markdown file (SKILL.md or AGENT.md)
- A local YAML file (Skillrunner or Claude format)
- A web URL (https://example.com/skill/SKILL.md)
- A GitHub repository (https://github.com/user/skill-repo.git)

Use --import flag to automatically import converted orchestrated skills to the skills directory.

Examples:
  # Convert skill directory to Skillrunner format
  sr convert ../business-analyst/ output.yaml

  # Convert and auto-import to skills directory
  sr convert ../business-analyst/ output.yaml --import

  # Convert from web URL
  sr convert https://example.com/skill/SKILL.md output.yaml

  # Convert from GitHub repository
  sr convert https://github.com/user/skill-repo.git output.yaml --import

  # Convert Claude skill markdown to Skillrunner format
  sr convert SKILL.md orchestrated-skill.yaml

  # Convert Skillrunner skill to Claude format
  sr convert orchestrated-skill.yaml marketplace-skill.yaml`,
	Args: cobra.ExactArgs(2),
	RunE: convertSkill,
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage result cache",
	Long:  `Manage the result cache for agent and workflow executions.`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the result cache",
	Long:  `Clear all cached execution results.`,
	RunE:  clearCache,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	Long:  `Show statistics about the result cache.`,
	RunE:  showCacheStats,
}

var (
	modelOverride string
	workspace     string
	outputFile    string
	compact       bool
	format        string
	modelPolicy   string
	preferLocal   bool
	forceCloud    bool
	taskType      string
	mcpEndpoint   string
	skillParams   string
	stream        bool
	profile       string
	showSavings   bool
	reportJSON    string
	noCache       bool
	convertImport bool // Auto-import after conversion
)

func init() {
	// Run command flags
	runCmd.Flags().StringVarP(&modelOverride, "model", "m", "", "Override the default model for this execution")
	runCmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Path to the workspace directory")
	runCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	runCmd.Flags().BoolVar(&compact, "compact", false, "Output compact JSON without indentation")
	runCmd.Flags().BoolVar(&preferLocal, "prefer-local", true, "Prefer local models (Ollama) over cloud models")
	runCmd.Flags().BoolVar(&forceCloud, "force-cloud", false, "Force cloud models (Anthropic/OpenAI)")
	runCmd.Flags().StringVarP(&taskType, "task-type", "t", "", "Task type for model routing (summarization, extraction, analysis, generation, verification, review, architecture)")
	runCmd.Flags().StringVar(&skillParams, "params", "", "JSON parameters for marketplace skill execution")
	runCmd.Flags().BoolVar(&stream, "stream", false, "Stream responses in real-time (Ollama only)")
	runCmd.Flags().StringVar(&profile, "profile", "balanced", "Routing profile (cheap|balanced|premium)")
	runCmd.Flags().BoolVar(&showSavings, "show-savings", false, "Display cost comparison (premium-only vs cheap-only)")
	runCmd.Flags().StringVar(&reportJSON, "report-json", "", "Write JSON cost report to file")
	runCmd.Flags().BoolVar(&noCache, "no-cache", false, "Skip cache and force fresh execution")

	// List command flags
	listCmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table|json)")

	// Status command flags
	statusCmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table|json)")

	// Search command flags
	searchCmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table|json)")

	// Inspect command flags
	inspectCmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table|json)")

	// Ask command flags
	askCmd.Flags().BoolVar(&preferLocal, "prefer-local", true, "Prefer local models (Ollama) over cloud models")
	askCmd.Flags().BoolVar(&forceCloud, "force-cloud", false, "Force cloud models (Anthropic/OpenAI)")
	askCmd.Flags().StringVarP(&taskType, "task-type", "t", "analysis", "Task type for model routing")
	askCmd.Flags().StringVarP(&modelOverride, "model", "m", "", "Override model selection")

	// Convert command flags
	convertCmd.Flags().BoolVar(&convertImport, "import", false, "Auto-import converted skill to skills directory (only for conversions to orchestrated format)")

	// Global flags
	rootCmd.PersistentFlags().StringVar(&modelPolicy, "model-policy", "", "Model selection policy (auto|local_first|performance_first|cost_optimized)")
	rootCmd.PersistentFlags().StringVar(&mcpEndpoint, "mcp-endpoint", "", "MCP server endpoint (default: http://localhost:3000)")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(importListCmd)
	rootCmd.AddCommand(importRemoveCmd)
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(configCmd)

	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	rootCmd.AddCommand(cacheCmd)

	// Add models command
	rootCmd.AddCommand(modelsCmd)

	// Add marketplace command
	rootCmd.AddCommand(marketplaceCmd)
}

func createSkillrunner(workspace string) (*engine.Skillrunner, error) {
	policy, err := engine.ResolveModelPolicy(modelPolicy)
	if err != nil {
		return nil, err
	}

	// Load configuration
	cfgManager, err := config.NewManager("")
	if err != nil {
		// If config fails to load, continue without auto_start
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
	} else {
		cfg := cfgManager.Get()

		// Auto-start Docker services if enabled
		if cfg.Router.AutoStart {
			projectDir := cfg.Docker.ProjectDir
			if projectDir == "" {
				// Try to find project root with docker-compose.yml
				if wd, err := os.Getwd(); err == nil {
					projectDir = wd
				}
			}

			sm, err := docker.NewServiceManager(projectDir, cfgManager.GetEnvVars())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Docker auto-start failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "You may need to manually start services: docker-compose up -d\n")
			} else {
				// Try to ensure services are running
				if err := sm.EnsureServicesRunning(true, cfg.Router.LiteLLMURL); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
					fmt.Fprintf(os.Stderr, "Continuing without Docker services...\n")
				}
			}
		}
	}

	return engine.NewSkillrunner(workspace, policy), nil
}

// getSkillsDir returns the directory where orchestrated skills are stored
func getSkillsDir() string {
	// Check if path is overridden in config
	cfgManager, err := config.NewManager("")
	if err == nil {
		cfg := cfgManager.Get()
		if cfg.Paths != nil && cfg.Paths.SkillrunnerDir != "" {
			return filepath.Join(cfg.Paths.SkillrunnerDir, "skills")
		}
	}

	// Default to ~/.skillrunner/skills
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./skills"
	}
	return filepath.Join(homeDir, ".skillrunner", "skills")
}

// isOrchestratedSkill checks if a skill is an orchestrated skill
func isOrchestratedSkill(skillName string) bool {
	skillsDir := getSkillsDir()

	// Check for skill.yaml
	yamlPath := filepath.Join(skillsDir, skillName, "skill.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return true
	}

	// Check for skill.yml
	ymlPath := filepath.Join(skillsDir, skillName, "skill.yml")
	if _, err := os.Stat(ymlPath); err == nil {
		return true
	}

	return false
}

func runSkill(cmd *cobra.Command, args []string) error {
	skillName := args[0]
	request := args[1]

	// Check if this is a marketplace skill (starts with "hf:")
	if strings.HasPrefix(skillName, "hf:") {
		return runMarketplaceSkill(skillName[3:], request)
	}

	// Check if this is an orchestrated skill
	if isOrchestratedSkill(skillName) {
		return runOrchestratedSkill(skillName, request)
	}

	// Fall back to legacy envelope-based execution
	eng, err := createSkillrunner(workspace)
	if err != nil {
		return err
	}

	// Determine which model to use
	selectedModel := modelOverride

	// Note: Task-type routing is now handled by profile-based routing in orchestrated skills
	// Legacy skills use the engine's default model selection

	// Execute the legacy skill
	envelope, err := eng.Run(skillName, request, selectedModel)
	if err != nil {
		return fmt.Errorf("error running skill: %w", err)
	}

	// Convert to JSON
	var jsonOutput []byte
	if compact {
		jsonOutput, err = json.Marshal(envelope)
	} else {
		jsonOutput, err = json.MarshalIndent(envelope, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("error marshaling envelope: %w", err)
	}

	// Output to file or stdout
	if outputFile != "" {
		err = os.WriteFile(outputFile, jsonOutput, 0644)
		if err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Envelope written to %s\n", outputFile)
	} else {
		fmt.Println(string(jsonOutput))
	}

	return nil
}

// processTemplateVariablesInText processes template variables like {{key}} in a string
// using only user context (for cached results where phase results aren't available)
func processTemplateVariablesInText(text string, userContext map[string]interface{}) string {
	result := text
	start := 0

	for {
		// Find the next {{ pattern
		openIdx := strings.Index(result[start:], "{{")
		if openIdx == -1 {
			break
		}
		openIdx += start

		// Find the matching }}
		closeIdx := strings.Index(result[openIdx:], "}}")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx

		// Extract the variable path
		varPath := strings.TrimSpace(result[openIdx+2 : closeIdx])

		// Resolve the variable value from user context only
		varValue := ""
		parts := strings.Split(varPath, ".")

		// Check user context
		if len(parts) >= 1 && parts[0] == "user" {
			if len(parts) > 1 {
				key := strings.Join(parts[1:], ".")
				if val, ok := userContext[key]; ok {
					varValue = fmt.Sprintf("%v", val)
				}
			}
		} else {
			// Try direct lookup in user context
			if val, ok := userContext[varPath]; ok {
				varValue = fmt.Sprintf("%v", val)
			}
		}

		// Replace the template variable with the resolved value (or empty if not found)
		result = result[:openIdx] + varValue + result[closeIdx+2:]

		// Continue searching from after the replacement
		start = openIdx + len(varValue)
	}

	return result
}

// runOrchestratedSkill executes an orchestrated multi-phase skill
func runOrchestratedSkill(skillName, request string) error {
	// Prepare user context (from request) - needed for processing cached results
	userContext := map[string]interface{}{
		"request": request,
	}

	// Parse additional context if provided via skillParams
	if skillParams != "" {
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(skillParams), &params); err == nil {
			for k, v := range params {
				userContext[k] = v
			}
		}
	}

	// Check cache first (unless --no-cache is set)
	var cacheMgr *cache.Manager
	if !noCache {
		var err error
		cacheMgr, err = cache.NewManager(24) // 24 hour TTL
		if err == nil {
			// Try to get from cache (use empty model for cache key since model is selected per phase)
			if cached, found := cacheMgr.Get(skillName, request, ""); found {
				// Process template variables in cached result (at least user context variables)
				processed := processTemplateVariablesInText(cached, userContext)
				fmt.Printf("=== Cached Result ===\n")
				fmt.Printf("Agent/Workflow: %s\n", skillName)
				fmt.Printf("Request: %s\n\n", request)
				fmt.Printf("%s\n", processed)
				return nil
			}
		}
	}

	skillsDir := getSkillsDir()

	fmt.Printf("=== Orchestrated Workflow Execution ===\n")
	fmt.Printf("Agent/Workflow: %s\n", skillName)
	fmt.Printf("Request: %s\n\n", request)

	// Load skill
	loader := orchestration.NewSkillLoader(skillsDir)
	skill, err := loader.LoadSkill(skillName)
	if err != nil {
		return fmt.Errorf("load workflow: %w", err)
	}

	fmt.Printf("Loaded workflow: %s v%s\n", skill.Name, skill.Version)
	fmt.Printf("Description: %s\n\n", skill.Description)

	// Create streaming callback if --stream flag is set
	var streamCallback orchestration.StreamCallback
	if stream {
		var currentPhase string
		streamCallback = func(phaseID string, chunk string) error {
			// Print phase header on first chunk of each phase
			if currentPhase != phaseID {
				if currentPhase != "" {
					fmt.Println() // Add newline after previous phase
				}
				fmt.Printf("\n--- Streaming output for phase: %s ---\n", phaseID)
				currentPhase = phaseID
			}
			// Print chunks immediately to stdout
			fmt.Print(chunk)
			return nil
		}
	}

	// Create phase executor
	configPath := "config/models.yaml"
	executor, err := orchestration.NewPhaseExecutor(skill, userContext, configPath, profile, streamCallback, modelOverride)
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	// Display profile if using profile routing
	if profile != "" {
		fmt.Printf("Using routing profile: %s\n\n", profile)
	}

	// Execute phases
	fmt.Println("🚀 Starting execution...")
	fmt.Println()
	ctx := context.Background()
	results, err := executor.Execute(ctx)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Display final output
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📋 Final Output")
	fmt.Println(strings.Repeat("=", 70))
	finalOutput := executor.GetFinalOutput()
	fmt.Println(finalOutput)

	// Cache the result
	if cacheMgr != nil {
		_ = cacheMgr.Set(skillName, request, "", finalOutput)
	}

	// Get cost summary if available
	costSummary := executor.GetCostSummary()

	// Display cost information if profile routing is enabled
	if costSummary != nil {
		fmt.Println("\n=== Cost Summary ===")
		fmt.Printf("Actual cost: $%.2f\n", costSummary.ActualCost)
		if showSavings || profile != "" {
			fmt.Printf("If premium-only: $%.2f\n", costSummary.PremiumOnlyCost)
			fmt.Printf("If cheap-only: $%.2f\n", costSummary.CheapOnlyCost)
		}

		// Write JSON report if requested
		if reportJSON != "" {
			report := runreport.RunReport{
				WorkflowName: skillName,
				ProfileUsed:  profile,
				CostSummary:  *costSummary,
			}
			if err := report.WriteJSON(reportJSON); err != nil {
				return fmt.Errorf("write report: %w", err)
			}
			fmt.Printf("\nCost report written to: %s\n", reportJSON)
		}
	}

	// Display summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📊 Execution Summary")
	fmt.Println(strings.Repeat("=", 70))
	var totalInputTokens, totalOutputTokens int
	var totalDuration int64
	for _, result := range results {
		totalInputTokens += result.InputTokens
		totalOutputTokens += result.OutputTokens
		totalDuration += result.DurationMs
	}

	fmt.Printf("Total phases:     %d\n", len(results))
	fmt.Printf("Input tokens:     %d\n", totalInputTokens)
	fmt.Printf("Output tokens:    %d\n", totalOutputTokens)
	fmt.Printf("Total duration:   %.2fs\n", float64(totalDuration)/1000.0)

	// Display completion message
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("✅ Execution Complete!")
	fmt.Println(strings.Repeat("=", 70))

	return nil
}

func listSkills(cmd *cobra.Command, args []string) error {
	eng, err := createSkillrunner("")
	if err != nil {
		return err
	}
	skills, err := eng.ListSkills()
	if err != nil {
		return fmt.Errorf("error listing skills: %w", err)
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(skills, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling skills: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		// Table format
		fmt.Println("\nAvailable Skills:")
		fmt.Println(strings.Repeat("=", 70))

		// Group skills by type - use proper categorization
		var builtinSkills []types.SkillConfig
		var orchestratedSkills []types.SkillConfig
		var importedSkills []types.SkillConfig

		// Get directories for checking
		skillsDir := getSkillsDir()

		// Check importer to see which skills are imported
		// We need to access the importer through reflection or add a getter method
		// For now, we'll use a simpler approach: check if skill exists in orchestrated dir
		// If not, and it's not a known built-in, it's imported

		// Get list of imported skill names from importer
		importedSkillNames := make(map[string]bool)
		if importer := eng.GetImporter(); importer != nil {
			for _, imported := range importer.ListSkills() {
				importedSkillNames[imported.Name] = true
			}
		}

		// Categorize skills properly
		for _, skill := range skills {
			// Check if it's an orchestrated skill (exists in ~/.skillrunner/skills/)
			yamlPath := filepath.Join(skillsDir, skill.Name, "skill.yaml")
			ymlPath := filepath.Join(skillsDir, skill.Name, "skill.yml")
			if _, err := os.Stat(yamlPath); err == nil {
				orchestratedSkills = append(orchestratedSkills, skill)
			} else if _, err := os.Stat(ymlPath); err == nil {
				orchestratedSkills = append(orchestratedSkills, skill)
			} else if importedSkillNames[skill.Name] {
				// It's in the importer registry, so it's imported
				importedSkills = append(importedSkills, skill)
			} else {
				// Not in orchestrated dir and not in importer - could be built-in or legacy
				// Since builtinSkills is currently empty, treat as imported for now
				importedSkills = append(importedSkills, skill)
			}
		}

		// Print built-in skills (only if we have any)
		// Note: builtinSkills is currently always empty, so this section won't show
		if len(builtinSkills) > 0 {
			fmt.Println("\n📦 Built-in Skills:")
			for _, skill := range builtinSkills {
				fmt.Printf("  • %-25s v%-10s\n", skill.Name, skill.Version)
				fmt.Printf("    %s\n\n", skill.Description)
			}
		}

		// Print orchestrated skills
		if len(orchestratedSkills) > 0 {
			fmt.Println("\n🎯 Orchestrated Skills (multi-phase workflows):")
			for _, skill := range orchestratedSkills {
				fmt.Printf("  • %-25s v%-10s\n", skill.Name, skill.Version)
				fmt.Printf("    %s\n\n", skill.Description)
			}
		}

		// Print imported marketplace skills
		if len(importedSkills) > 0 {
			fmt.Println("\n📚 Imported Marketplace Skills:")
			for _, skill := range importedSkills {
				fmt.Printf("  • %-25s v%-10s\n", skill.Name, skill.Version)
				fmt.Printf("    %s\n\n", skill.Description)
			}
		}

		if len(skills) == 0 {
			fmt.Println("  No skills found.")
			fmt.Println("\n  To get started:")
			fmt.Println("    • Import skills: sr import <source>")
			fmt.Println("    • List imported: sr imported")
		}
	}

	return nil
}

func showStatus(cmd *cobra.Command, args []string) error {
	eng, err := createSkillrunner("")
	if err != nil {
		return err
	}
	status, err := eng.GetStatus()
	if err != nil {
		return fmt.Errorf("error getting status: %w", err)
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling status: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		// Table format
		fmt.Println("\nSkillrunner Status:")
		fmt.Println("----------------------------------------------------------------------")
		fmt.Printf("  Version:          %s\n", status.Version)
		fmt.Printf("  Available Agents/Workflows: %d\n", status.SkillCount)
		fmt.Printf("  Workspace:        %s\n", status.Workspace)
		readyStr := "Ready"
		if !status.Ready {
			readyStr = "Not Ready"
		}
		fmt.Printf("  Status:           %s\n", readyStr)
		fmt.Println("\n  Configured Models:")
		for _, model := range status.ConfiguredModels {
			fmt.Printf("    - %s\n", model)
		}
	}

	return nil
}

func runMarketplaceSkill(skillID string, request string) error {
	// Create marketplace client
	client := marketplace.NewHuggingFaceClient(mcpEndpoint)

	// Parse parameters if provided
	var params map[string]interface{}
	if skillParams != "" {
		if err := json.Unmarshal([]byte(skillParams), &params); err != nil {
			return fmt.Errorf("invalid params JSON: %w", err)
		}
	} else {
		// Use request as the default parameter
		params = map[string]interface{}{
			"input": request,
		}
	}

	fmt.Fprintf(os.Stderr, "Executing marketplace skill: %s\n", skillID)

	// Execute the skill
	ctx := context.Background()
	result, err := client.ExecuteSkill(ctx, skillID, params)
	if err != nil {
		return fmt.Errorf("marketplace skill execution failed: %w", err)
	}

	// Output result
	var jsonOutput []byte
	if compact {
		jsonOutput, err = json.Marshal(result)
	} else {
		jsonOutput, err = json.MarshalIndent(result, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("error marshaling result: %w", err)
	}

	// Output to file or stdout
	if outputFile != "" {
		err = os.WriteFile(outputFile, jsonOutput, 0644)
		if err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Result written to %s\n", outputFile)
	} else {
		fmt.Println(string(jsonOutput))
	}

	return nil
}

func searchMarketplace(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Create marketplace client
	client := marketplace.NewHuggingFaceClient(mcpEndpoint)

	// Search for skills
	ctx := context.Background()
	skills, err := client.SearchSkills(ctx, query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(skills) == 0 {
		fmt.Printf("No skills found for query: %s\n", query)
		return nil
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(skills, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling results: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		// Table format
		fmt.Printf("\nFound %d skills matching '%s':\n", len(skills), query)
		fmt.Println("----------------------------------------------------------------------")
		for _, skill := range skills {
			fmt.Printf("\n  ID:          %s\n", skill.ID)
			fmt.Printf("  Name:        %s\n", skill.Name)
			fmt.Printf("  Author:      %s\n", skill.Author)
			fmt.Printf("  Description: %s\n", skill.Description)
			if len(skill.Tags) > 0 {
				fmt.Printf("  Tags:        %s\n", strings.Join(skill.Tags, ", "))
			}
			fmt.Printf("  Likes:       %d\n", skill.Likes)
		}
		fmt.Println("\nUse 'sr inspect <id>' to see more details")
		fmt.Println("Use 'sr run hf:<id> --params <json>' to execute")
	}

	return nil
}

func inspectSkill(cmd *cobra.Command, args []string) error {
	skillID := args[0]

	// Create marketplace client
	client := marketplace.NewHuggingFaceClient(mcpEndpoint)

	// Get skill details
	ctx := context.Background()
	skill, err := client.GetSkill(ctx, skillID)
	if err != nil {
		return fmt.Errorf("inspect failed: %w", err)
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(skill, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling skill: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		// Detailed format
		fmt.Println("\nSkill Details:")
		fmt.Println("======================================================================")
		fmt.Printf("  ID:          %s\n", skill.ID)
		fmt.Printf("  Name:        %s\n", skill.Name)
		fmt.Printf("  Author:      %s\n", skill.Author)
		fmt.Printf("  SDK:         %s\n", skill.SDK)
		if skill.Runtime != "" {
			fmt.Printf("  Runtime:     %s\n", skill.Runtime)
		}
		fmt.Printf("  Likes:       %d\n", skill.Likes)
		fmt.Printf("  Created:     %s\n", skill.CreatedAt)
		fmt.Printf("  Updated:     %s\n", skill.UpdatedAt)
		if skill.URL != "" {
			fmt.Printf("  URL:         %s\n", skill.URL)
		}
		fmt.Println("\n  Description:")
		fmt.Printf("    %s\n", skill.Description)
		if len(skill.Tags) > 0 {
			fmt.Println("\n  Tags:")
			fmt.Printf("    %s\n", strings.Join(skill.Tags, ", "))
		}
		fmt.Println("\n  Usage:")
		fmt.Printf("    sr run hf:%s --params '{\"key\": \"value\"}'\n", skill.ID)
	}

	return nil
}

func askMarketplaceSkill(cmd *cobra.Command, args []string) error {
	skillID := args[0]
	question := args[1]

	// Load marketplace skills
	loader, err := skills.NewMarketplaceLoader("")
	if err != nil {
		return fmt.Errorf("failed to load marketplace skills: %w", err)
	}

	// Build prompt with skill knowledge
	prompt, err := loader.BuildPromptWithSkill(skillID, question)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	// Determine which model to use
	var selectedModel string

	if modelOverride != "" {
		// User specified a model
		selectedModel = modelOverride
	} else {
		// Use profile-based routing (default to balanced profile)
		profile := "balanced"
		if preferLocal {
			profile = "cheap" // Prefer cheaper models when preferLocal is true
		} else if forceCloud {
			profile = "premium" // Use premium models when forcing cloud
		}

		configPath := "config/models.yaml"
		routingConfig, err := routing.LoadRoutingConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load routing config: %w", err)
		}

		profileRouter, err := routing.NewRouter(routingConfig)
		if err != nil {
			return fmt.Errorf("failed to create profile router: %w", err)
		}

		_, modelConfig, err := profileRouter.Route(profile)
		if err != nil {
			return fmt.Errorf("model selection failed: %w", err)
		}

		selectedModel = fmt.Sprintf("%s/%s", modelConfig.Provider, modelConfig.Model)
		fmt.Printf("Using model: %s (profile: %s)\n\n", selectedModel, profile)
	}

	// Execute with LLM client
	fmt.Printf("=== Marketplace Skill Execution ===\n")
	fmt.Printf("Skill:    %s\n", skillID)
	fmt.Printf("Model:    %s\n", selectedModel)
	fmt.Printf("Question: %s\n\n", question)
	fmt.Printf("Generating response...\n\n")

	// Create LLM client
	client := llm.NewClient()

	// Build completion request
	req := llm.CompletionRequest{
		Model:       selectedModel,
		Prompt:      prompt,
		MaxTokens:   4000,
		Temperature: 0.7,
		Stream:      stream,
	}

	// Execute
	ctx := context.Background()

	var resp *llm.CompletionResponse
	if stream {
		// Stream response in real-time
		fmt.Printf("--- Response (streaming) ---\n\n")
		resp, err = client.StreamCompletion(ctx, req, func(chunk string) error {
			fmt.Print(chunk)
			return nil
		})
		if err != nil {
			return fmt.Errorf("model execution failed: %w", err)
		}
		fmt.Println()
	} else {
		resp, err = client.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("model execution failed: %w", err)
		}

		// Display response
		fmt.Printf("--- Response ---\n\n")
		fmt.Printf("%s\n\n", resp.Content)
	}

	// Display metrics
	fmt.Printf("--- Metrics ---\n")
	fmt.Printf("Provider:      %s\n", resp.Provider)
	fmt.Printf("Model:         %s\n", resp.Model)
	fmt.Printf("Input tokens:  %d\n", resp.InputTokens)
	fmt.Printf("Output tokens: %d\n", resp.OutputTokens)
	fmt.Printf("Duration:      %v\n", resp.Duration)

	// Calculate cost (if cloud model)
	if strings.HasPrefix(selectedModel, "anthropic/") {
		inputCost := float64(resp.InputTokens) / 1000.0 * 0.003   // $0.003/1K input
		outputCost := float64(resp.OutputTokens) / 1000.0 * 0.015 // $0.015/1K output
		totalCost := inputCost + outputCost
		fmt.Printf("Cost:          $%.4f\n", totalCost)

		// Show savings vs cloud-only
		if resp.Provider == "ollama" {
			fmt.Printf("\nSavings: $%.4f (100%% - using free local model)\n", totalCost)
		}
	} else if resp.Provider == "ollama" {
		fmt.Printf("Cost:          $0.0000 (FREE - local model)\n")

		// Estimate what it would cost on Claude
		claudeInputCost := float64(resp.InputTokens) / 1000.0 * 0.003
		claudeOutputCost := float64(resp.OutputTokens) / 1000.0 * 0.015
		claudeTotalCost := claudeInputCost + claudeOutputCost
		fmt.Printf("\nSavings: $%.4f (vs Claude Sonnet)\n", claudeTotalCost)
	}

	return nil
}

func importSkill(cmd *cobra.Command, args []string) error {
	source := args[0]

	fmt.Printf("Importing from: %s\n", source)

	// Create importer
	imp, err := importer.NewImporter()
	if err != nil {
		return fmt.Errorf("create importer: %w", err)
	}

	// Import item (agent or skill)
	items, err := imp.Import(source)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Display results
	fmt.Printf("\n✓ Successfully imported %d item(s):\n\n", len(items))
	for _, item := range items {
		fmt.Printf("  Type:        %s\n", item.Type)
		fmt.Printf("  ID:          %s\n", item.ID)
		fmt.Printf("  Name:        %s\n", item.Name)
		fmt.Printf("  Version:     %s\n", item.Version)
		fmt.Printf("  Author:      %s\n", item.Author)
		fmt.Printf("  Description: %s\n", item.Description)
		fmt.Printf("  Source:      %s (%s)\n", item.Source.Path, item.Source.Type)
		fmt.Printf("  Local path:  %s\n", item.LocalPath)
		fmt.Printf("\n")
	}

	fmt.Printf("Items cached at: %s\n", imp.GetCacheDir())

	return nil
}

func updateSkill(cmd *cobra.Command, args []string) error {
	itemID := args[0]

	fmt.Printf("Updating: %s\n", itemID)

	// Create importer
	imp, err := importer.NewImporter()
	if err != nil {
		return fmt.Errorf("create importer: %w", err)
	}

	// Update item
	item, err := imp.Update(itemID)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	// Display results
	fmt.Printf("\n✓ Successfully updated %s:\n\n", item.Type)
	fmt.Printf("  Type:        %s\n", item.Type)
	fmt.Printf("  ID:          %s\n", item.ID)
	fmt.Printf("  Name:        %s\n", item.Name)
	fmt.Printf("  Version:     %s\n", item.Version)
	fmt.Printf("  Updated at:  %s\n", item.LastUpdated.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Source:      %s (%s)\n", item.Source.Path, item.Source.Type)

	return nil
}

func listImportedSkills(cmd *cobra.Command, args []string) error {
	// Create importer
	imp, err := importer.NewImporter()
	if err != nil {
		return fmt.Errorf("create importer: %w", err)
	}

	// Get all imported items
	items := imp.ListSkills()

	if len(items) == 0 {
		fmt.Println("No imported agents or skills found.")
		fmt.Printf("\nTo import agents or skills, use: sr import <source>\n")
		return nil
	}

	// Display items
	fmt.Printf("Imported Agents and Skills (%d total):\n\n", len(items))

	for i, item := range items {
		fmt.Printf("%d. [%s] %s\n", i+1, item.Type, item.ID)
		fmt.Printf("   Name:        %s\n", item.Name)
		fmt.Printf("   Version:     %s\n", item.Version)
		fmt.Printf("   Author:      %s\n", item.Author)
		fmt.Printf("   Source:      %s (%s)\n", item.Source.Path, item.Source.Type)
		fmt.Printf("   Imported:    %s\n", item.ImportedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Last update: %s\n", item.LastUpdated.Format("2006-01-02 15:04:05"))
		if item.Source.Type == "git" && item.Source.GitCommit != "" {
			fmt.Printf("   Git commit:  %s\n", item.Source.GitCommit[:8])
		}
		fmt.Printf("\n")
	}

	fmt.Printf("Cache directory: %s\n", imp.GetCacheDir())

	return nil
}

func removeSkill(cmd *cobra.Command, args []string) error {
	itemID := args[0]

	fmt.Printf("Removing: %s\n", itemID)

	// Create importer
	imp, err := importer.NewImporter()
	if err != nil {
		return fmt.Errorf("create importer: %w", err)
	}

	// Get item to show type
	item, err := imp.GetSkill(itemID)
	if err != nil {
		return fmt.Errorf("get item: %w", err)
	}

	// Remove item
	if err := imp.RemoveSkill(itemID); err != nil {
		return fmt.Errorf("remove failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully removed %s: %s\n", item.Type, itemID)

	return nil
}

func convertSkill(cmd *cobra.Command, args []string) error {
	inputFile := args[0]
	outputFile := args[1]

	fmt.Printf("Converting skill from %s to %s\n", inputFile, outputFile)

	// Check if input is a remote source (URL or git repo)
	sourceType := importer.DetectSourceType(inputFile)

	if sourceType == importer.SourceTypeWebHTTP || sourceType == importer.SourceTypeGitHTTPS || sourceType == importer.SourceTypeGitSSH {
		// Remote source - download/clone using importer
		fmt.Printf("Detected remote source: %s\n", sourceType.String())

		// Create importer to handle remote sources
		imp, err := importer.NewImporter()
		if err != nil {
			return fmt.Errorf("create importer: %w", err)
		}

		var downloadedPath string
		var cleanupPath string
		switch sourceType {
		case importer.SourceTypeWebHTTP:
			// Download from web
			fmt.Println("Downloading from web...")
			item, err := imp.ImportFromWeb(inputFile)
			if err != nil {
				return fmt.Errorf("download from web: %w", err)
			}
			downloadedPath = item.LocalPath
			// Note: Imported items are stored in ~/.skillrunner/marketplace/, we'll leave them there
			// User can clean up with `sr remove` if needed
		case importer.SourceTypeGitHTTPS, importer.SourceTypeGitSSH:
			// Clone git repo
			fmt.Println("Cloning git repository...")
			items, err := imp.ImportFromGit(inputFile)
			if err != nil {
				return fmt.Errorf("clone repository: %w", err)
			}
			if len(items) == 0 {
				return fmt.Errorf("no skills found in repository")
			}
			if len(items) > 1 {
				fmt.Printf("Warning: Repository contains %d skills, converting first one: %s\n", len(items), items[0].ID)
			}
			downloadedPath = items[0].LocalPath
		default:
			return fmt.Errorf("unsupported remote source type")
		}

		// Use downloaded path as input
		inputFile = downloadedPath
		fmt.Printf("Downloaded to: %s\n", inputFile)

		// Cleanup will be handled by defer if we created a temp path
		if cleanupPath != "" {
			defer os.RemoveAll(cleanupPath)
		}
	}

	// Check if input is a markdown file or directory
	inputInfo, err := os.Stat(inputFile)
	if err != nil {
		return fmt.Errorf("input path not found: %w", err)
	}

	if inputInfo.IsDir() || strings.HasSuffix(strings.ToLower(inputFile), ".md") {
		// This is a directory or markdown file (SKILL.md or AGENT.md)
		if inputInfo.IsDir() {
			fmt.Println("Detected: Skill directory (will find SKILL.md/AGENT.md)")
		} else {
			fmt.Println("Detected: Marketplace markdown format (SKILL.md/AGENT.md)")
		}
		fmt.Println("Converting to: Skillrunner orchestrated format")

		sr, err := converter.FromMarkdown(inputFile)
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		// Use custom marshaling to format multi-line strings as literal blocks
		outputData, err := converter.MarshalYAMLWithMultiLineStrings(sr)
		if err != nil {
			return fmt.Errorf("marshal output: %w", err)
		}

		// If input is a directory, preserve extra files and create output directory
		if inputInfo.IsDir() {
			// Determine output directory
			outputDir := outputFile
			outputInfo, err := os.Stat(outputFile)
			if err == nil && outputInfo.IsDir() {
				// Output is already a directory, use it
				outputDir = outputFile
			} else if strings.HasSuffix(outputFile, ".yaml") || strings.HasSuffix(outputFile, ".yml") {
				// Output is a file, create directory with same name (without extension)
				outputDir = strings.TrimSuffix(outputFile, filepath.Ext(outputFile))
			} else {
				// Output doesn't exist, treat as directory name
				outputDir = outputFile
			}

			// Create output directory
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}

			// Copy all files/directories except SKILL.md and AGENT.md
			entries, err := os.ReadDir(inputFile)
			if err != nil {
				return fmt.Errorf("read input directory: %w", err)
			}

			for _, entry := range entries {
				// Skip SKILL.md and AGENT.md (we're converting those)
				if entry.Name() == "SKILL.md" || entry.Name() == "AGENT.md" {
					continue
				}

				srcPath := filepath.Join(inputFile, entry.Name())
				dstPath := filepath.Join(outputDir, entry.Name())

				if entry.IsDir() {
					// Copy directory recursively
					if err := copyDirectoryRecursive(srcPath, dstPath); err != nil {
						return fmt.Errorf("copy directory %s: %w", entry.Name(), err)
					}
				} else {
					// Copy file
					if err := copyFileSimple(srcPath, dstPath); err != nil {
						return fmt.Errorf("copy file %s: %w", entry.Name(), err)
					}
				}
			}

			// Write skill.yaml in the output directory
			skillYamlPath := filepath.Join(outputDir, "skill.yaml")
			if err := os.WriteFile(skillYamlPath, outputData, 0644); err != nil {
				return fmt.Errorf("write skill.yaml: %w", err)
			}

			fmt.Printf("✓ Converted successfully to %s/\n", outputDir)
			fmt.Printf("  - skill.yaml (converted from SKILL.md/AGENT.md)\n")

			// Auto-import if flag is set
			if convertImport {
				if err := importConvertedSkill(sr, outputDir); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to auto-import skill: %v\n", err)
				} else {
					fmt.Printf("✓ Auto-imported to skills directory\n")
				}
			}

			return nil
		}

		// Input is a single file, just write the output file
		if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		fmt.Printf("✓ Converted successfully to %s\n", outputFile)

		// Auto-import if flag is set (converting TO orchestrated format from markdown)
		if convertImport {
			if err := importConvertedSkill(sr, outputFile); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to auto-import skill: %v\n", err)
			} else {
				fmt.Printf("✓ Auto-imported to skills directory\n")
			}
		}

		return nil
	}

	// Read input file for YAML formats
	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("read input file: %w", err)
	}

	// Try to detect format by parsing as both types
	var claudeSkill converter.ClaudeSkillFormat
	var srSkill types.OrchestratedSkill

	claudeErr := yaml.Unmarshal(inputData, &claudeSkill)
	srErr := yaml.Unmarshal(inputData, &srSkill)

	var outputData []byte

	// Determine which format it is based on structure
	if srErr == nil && len(srSkill.Phases) > 0 && srSkill.Phases[0].PromptTemplate != "" {
		// This is Skillrunner format → Convert to Claude format
		fmt.Println("Detected: Skillrunner orchestrated format")
		fmt.Println("Converting to: Claude Code marketplace format")
		// Note: --import flag is ignored when converting FROM orchestrated format

		claude, err := converter.FromSkillrunner(&srSkill)
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		outputData, err = yaml.Marshal(claude)
		if err != nil {
			return fmt.Errorf("marshal output: %w", err)
		}

	} else if claudeErr == nil && len(claudeSkill.Phases) > 0 {
		// This is Claude format → Convert to Skillrunner format
		fmt.Println("Detected: Claude Code marketplace format")
		fmt.Println("Converting to: Skillrunner orchestrated format")

		sr, err := converter.ToSkillrunner(&claudeSkill)
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		outputData, err = yaml.Marshal(sr)
		if err != nil {
			return fmt.Errorf("marshal output: %w", err)
		}

		// Auto-import if flag is set (converting TO orchestrated format)
		if convertImport {
			// Write to temp location first, then import
			tempFile := outputFile
			if err := importConvertedSkill(sr, tempFile); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to auto-import skill: %v\n", err)
			} else {
				fmt.Printf("✓ Auto-imported to skills directory\n")
			}
		}

	} else {
		return fmt.Errorf("unable to detect skill format (tried Skillrunner, Claude, and Markdown formats)")
	}

	// Write output file
	if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	fmt.Printf("\n✓ Successfully converted skill\n")
	fmt.Printf("Output written to: %s\n", outputFile)

	return nil
}

// importConvertedSkill imports a converted orchestrated skill to the skills directory
func importConvertedSkill(skill *types.OrchestratedSkill, sourcePath string) error {
	// Get skills directory (respects config override)
	skillsDir := getSkillsDir()

	// Use skill name as directory name
	skillDir := filepath.Join(skillsDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("create skill directory: %w", err)
	}

	// Determine source file path
	var sourceFile string
	if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
		// Source is a directory, look for skill.yaml
		sourceFile = filepath.Join(sourcePath, "skill.yaml")
	} else {
		// Source is a file
		sourceFile = sourcePath
	}

	// Copy skill.yaml to skills directory
	destFile := filepath.Join(skillDir, "skill.yaml")

	// Read source file
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destFile, data, 0644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	// If source is a directory, copy any additional files (templates, helpers, etc.)
	if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
		entries, err := os.ReadDir(sourcePath)
		if err == nil {
			for _, entry := range entries {
				// Skip skill.yaml (already copied) and hidden files
				if entry.Name() == "skill.yaml" || entry.Name() == "skill.yml" || strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				srcPath := filepath.Join(sourcePath, entry.Name())
				dstPath := filepath.Join(skillDir, entry.Name())

				if entry.IsDir() {
					if err := copyDirectoryRecursive(srcPath, dstPath); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to copy directory %s: %v\n", entry.Name(), err)
					}
				} else {
					if err := copyFileSimple(srcPath, dstPath); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to copy file %s: %v\n", entry.Name(), err)
					}
				}
			}
		}
	}

	return nil
}

// copyDirectoryRecursive recursively copies a directory
func copyDirectoryRecursive(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectoryRecursive(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFileSimple(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFileSimple copies a single file
func copyFileSimple(src, dst string) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, srcData, srcInfo.Mode())
}

func clearCache(cmd *cobra.Command, args []string) error {
	cacheMgr, err := cache.NewManager(24)
	if err != nil {
		return fmt.Errorf("create cache manager: %w", err)
	}

	if err := cacheMgr.Clear(); err != nil {
		return fmt.Errorf("clear cache: %w", err)
	}

	fmt.Println("Cache cleared successfully")
	return nil
}

func showCacheStats(cmd *cobra.Command, args []string) error {
	cacheMgr, err := cache.NewManager(24)
	if err != nil {
		return fmt.Errorf("create cache manager: %w", err)
	}

	total, valid, expired := cacheMgr.Stats()
	fmt.Printf("Cache Statistics:\n")
	fmt.Printf("  Total entries:  %d\n", total)
	fmt.Printf("  Valid entries:  %d\n", valid)
	fmt.Printf("  Expired entries: %d\n", expired)

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
