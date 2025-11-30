package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/jbctechsolutions/skillrunner/internal/llm"
	"github.com/jbctechsolutions/skillrunner/internal/routing"
	"github.com/jbctechsolutions/skillrunner/internal/skills"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	orchestrationEnabled bool
	enableOrchestration  bool
	contextFile          string
	exportOutput         string
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Marketplace agent management",
	Long:  `Commands for discovering and executing marketplace agents.`,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all marketplace agents",
	Long: `List all agents from the marketplace.

Examples:
  sr marketplace list
  sr marketplace list --orchestration-enabled
  sr marketplace list --format json`,
	RunE: listMarketplaceAgents,
}

var marketplaceInspectCmd = &cobra.Command{
	Use:   "inspect <agent-name>",
	Short: "Inspect a marketplace agent",
	Long: `Get detailed information about a specific marketplace agent.

Examples:
  sr marketplace inspect backend-architect
  sr marketplace inspect frontend-specialist`,
	Args: cobra.ExactArgs(1),
	RunE: inspectMarketplaceAgent,
}

var marketplaceRunCmd = &cobra.Command{
	Use:   "run <agent-name> [request]",
	Short: "Run a marketplace agent",
	Long: `Run a marketplace agent with a request.

Examples:
  sr marketplace run backend-architect "design user authentication API"
  sr marketplace run backend-architect "add caching layer" --context ./context.md
  sr marketplace run backend-architect "optimize database" --prefer-local --enable-orchestration`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMarketplaceAgent,
}

var marketplaceExportCmd = &cobra.Command{
	Use:   "export <agent-name>",
	Short: "Export agent to workflow format",
	Long: `Export a marketplace agent to Skillrunner orchestrated workflow format.

Examples:
  sr marketplace export backend-architect --output backend-architect.yaml
  sr marketplace export frontend-specialist --output workflows/frontend.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: exportMarketplaceAgent,
}

func init() {
	// List command flags
	marketplaceListCmd.Flags().BoolVar(&orchestrationEnabled, "orchestration-enabled", false, "Show only agents with orchestration config")
	marketplaceListCmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table|json)")

	// Inspect command flags
	marketplaceInspectCmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table|json)")

	// Run command flags
	marketplaceRunCmd.Flags().StringVar(&contextFile, "context", "", "Path to context file")
	marketplaceRunCmd.Flags().BoolVar(&preferLocal, "prefer-local", true, "Prefer local models over cloud models")
	marketplaceRunCmd.Flags().BoolVar(&enableOrchestration, "enable-orchestration", false, "Enable orchestration if agent supports it")
	marketplaceRunCmd.Flags().StringVarP(&taskType, "task-type", "t", "analysis", "Task type for model routing")
	marketplaceRunCmd.Flags().StringVarP(&modelOverride, "model", "m", "", "Override model selection")

	// Export command flags
	marketplaceExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (required)")
	marketplaceExportCmd.MarkFlagRequired("output")

	// Add subcommands
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceInspectCmd)
	marketplaceCmd.AddCommand(marketplaceRunCmd)
	marketplaceCmd.AddCommand(marketplaceExportCmd)
}

func listMarketplaceAgents(cmd *cobra.Command, args []string) error {
	// Load marketplace agents
	loader, err := skills.NewMarketplaceLoader("")
	if err != nil {
		return fmt.Errorf("failed to load marketplace: %w", err)
	}

	agents := loader.ListAgents()
	if len(agents) == 0 {
		fmt.Println("No marketplace agents found.")
		return nil
	}

	// Filter by orchestration if requested
	var filteredAgents []*skills.MarketplaceAgent
	if orchestrationEnabled {
		for _, agent := range agents {
			if agent.Orchestration != nil && agent.Orchestration.Enabled {
				filteredAgents = append(filteredAgents, agent)
			}
		}
	} else {
		filteredAgents = agents
	}

	if len(filteredAgents) == 0 {
		fmt.Println("No agents found matching criteria.")
		return nil
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(filteredAgents, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling agents: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		// Table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tPRIMARY SKILL\tMODEL\tORCHESTRATION")
		fmt.Fprintln(w, "----\t-----------\t-------------\t-----\t-------------")

		for _, agent := range filteredAgents {
			orchStatus := "No"
			if agent.Orchestration != nil && agent.Orchestration.Enabled {
				orchStatus = "Yes"
			}

			// Truncate description if too long
			desc := agent.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				agent.Name,
				desc,
				agent.PrimarySkill,
				agent.Model,
				orchStatus)
		}

		w.Flush()
		fmt.Printf("\nTotal: %d agents\n", len(filteredAgents))
	}

	return nil
}

func inspectMarketplaceAgent(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	// Load marketplace agents
	loader, err := skills.NewMarketplaceLoader("")
	if err != nil {
		return fmt.Errorf("failed to load marketplace: %w", err)
	}

	agent, err := loader.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(agent, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling agent: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		// Detailed format
		fmt.Println("\nMarketplace Agent Details:")
		fmt.Println("======================================================================")
		fmt.Printf("  Name:         %s\n", agent.Name)
		fmt.Printf("  Description:  %s\n", agent.Description)
		fmt.Printf("  Model:        %s\n", agent.Model)
		fmt.Printf("  Primary Skill: %s\n", agent.PrimarySkill)

		if len(agent.SupportingSkills) > 0 {
			fmt.Printf("  Supporting Skills:\n")
			for _, skill := range agent.SupportingSkills {
				fmt.Printf("    - %s\n", skill)
			}
		}

		if len(agent.Tools) > 0 {
			fmt.Printf("  Tools:        %s\n", strings.Join(agent.Tools, ", "))
		}

		// Routing configuration
		if agent.Routing != nil {
			fmt.Println("\n  Routing Configuration:")
			fmt.Printf("    Defer to skill:  %v\n", agent.Routing.DeferToSkill)
			if agent.Routing.FallbackModel != "" {
				fmt.Printf("    Fallback model:  %s\n", agent.Routing.FallbackModel)
			}
		}

		// Orchestration configuration
		if agent.Orchestration != nil {
			fmt.Println("\n  Orchestration Configuration:")
			fmt.Printf("    Enabled:         %v\n", agent.Orchestration.Enabled)
			if len(agent.Orchestration.DefaultPhases) > 0 {
				fmt.Printf("    Default phases:\n")
				for _, phase := range agent.Orchestration.DefaultPhases {
					fmt.Printf("      - %s\n", phase)
				}
			}
			if agent.Orchestration.RoutingStrategy != "" {
				fmt.Printf("    Routing strategy: %s\n", agent.Orchestration.RoutingStrategy)
			}
			fmt.Printf("    Cost optimization: %v\n", agent.Orchestration.CostOptimization)
		}

		// Content preview
		if agent.Content != "" {
			fmt.Println("\n  Agent Expertise (Summary):")
			lines := strings.Split(agent.Content, "\n")
			maxLines := 10
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Printf("    %s\n", lines[i])
			}
			if len(lines) > maxLines {
				fmt.Printf("    ... (%d more lines)\n", len(lines)-maxLines)
			}
		}

		fmt.Println("\n  Usage:")
		fmt.Printf("    sr marketplace run %s \"your request here\"\n", agentID)
	}

	return nil
}

func runMarketplaceAgent(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	var request string
	if len(args) > 1 {
		request = args[1]
	} else {
		// If no request provided, read from stdin or use interactive mode
		return fmt.Errorf("request is required")
	}

	// Load marketplace agents
	loader, err := skills.NewMarketplaceLoader("")
	if err != nil {
		return fmt.Errorf("failed to load marketplace: %w", err)
	}

	agent, err := loader.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	fmt.Printf("=== Running Marketplace Agent ===\n")
	fmt.Printf("Agent:   %s\n", agent.Name)
	fmt.Printf("Request: %s\n\n", request)

	// Load context if provided
	var contextContent string
	if contextFile != "" {
		content, err := os.ReadFile(contextFile)
		if err != nil {
			return fmt.Errorf("failed to read context file: %w", err)
		}
		contextContent = string(content)
	}

	// Build prompt with agent expertise
	prompt, err := loader.BuildPromptWithAgent(agentID, request)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	// Add context if provided
	if contextContent != "" {
		prompt = fmt.Sprintf("%s\n\nAdditional Context:\n%s", prompt, contextContent)
	}

	// Check if orchestration is requested and supported
	if enableOrchestration && agent.Orchestration != nil && agent.Orchestration.Enabled {
		fmt.Printf("Note: Orchestration enabled for this agent.\n")
		fmt.Printf("Default phases: %s\n", strings.Join(agent.Orchestration.DefaultPhases, ", "))
		fmt.Printf("\nMulti-phase orchestration execution will be available in Phase 2.\n")
		fmt.Printf("Proceeding with single-phase execution...\n\n")
	}

	// Determine which model to use
	var selectedModel string

	if modelOverride != "" {
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
		fmt.Printf("Selected model: %s (profile: %s)\n\n", selectedModel, profile)
	}

	// Execute with LLM client
	fmt.Printf("Generating response...\n\n")

	client := llm.NewClient()

	req := llm.CompletionRequest{
		Model:       selectedModel,
		Prompt:      prompt,
		MaxTokens:   4000,
		Temperature: 0.7,
		Stream:      stream,
	}

	ctx := context.Background()

	var resp *llm.CompletionResponse
	if stream {
		fmt.Printf("--- Response (streaming) ---\n\n")
		resp, err = client.StreamCompletion(ctx, req, func(chunk string) error {
			fmt.Print(chunk)
			return nil
		})
		if err != nil {
			return fmt.Errorf("agent execution failed: %w", err)
		}
		fmt.Println()
	} else {
		resp, err = client.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("agent execution failed: %w", err)
		}

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

	// Calculate cost
	if strings.HasPrefix(selectedModel, "anthropic/") {
		inputCost := float64(resp.InputTokens) / 1000.0 * 0.003
		outputCost := float64(resp.OutputTokens) / 1000.0 * 0.015
		totalCost := inputCost + outputCost
		fmt.Printf("Cost:          $%.4f\n", totalCost)
	} else if resp.Provider == "ollama" {
		fmt.Printf("Cost:          $0.0000 (FREE - local model)\n")
	}

	return nil
}

func exportMarketplaceAgent(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	// Load marketplace agents
	loader, err := skills.NewMarketplaceLoader("")
	if err != nil {
		return fmt.Errorf("failed to load marketplace: %w", err)
	}

	agent, err := loader.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	fmt.Printf("Exporting agent '%s' to orchestrated workflow format...\n", agent.Name)

	// Create exported workflow structure
	workflow := ExportedWorkflow{
		Name:        agent.Name,
		Version:     "1.0.0",
		Description: agent.Description,
		Type:        "orchestrated",
		Phases:      []ExportedPhase{},
	}

	// If agent has orchestration config, use default phases
	if agent.Orchestration != nil && agent.Orchestration.Enabled && len(agent.Orchestration.DefaultPhases) > 0 {
		for i, phaseName := range agent.Orchestration.DefaultPhases {
			phase := ExportedPhase{
				ID:          fmt.Sprintf("phase_%d_%s", i+1, strings.ReplaceAll(phaseName, " ", "_")),
				TaskType:    "analysis", // Default, can be customized
				Description: fmt.Sprintf("Execute %s phase", phaseName),
				PromptTemplate: fmt.Sprintf(`You are executing the '%s' phase of the %s workflow.

Agent Expertise:
%s

Phase: %s

Request: {{.request}}

Provide a detailed response for this phase.`, phaseName, agent.Name, agent.Content, phaseName),
			}
			workflow.Phases = append(workflow.Phases, phase)
		}
	} else {
		// Create a simple single-phase workflow
		phase := ExportedPhase{
			ID:          "phase_1_execution",
			TaskType:    "analysis",
			Description: fmt.Sprintf("Execute %s agent", agent.Name),
			PromptTemplate: fmt.Sprintf(`You are a %s.

%s

Request: {{.request}}

Provide a comprehensive response based on your expertise.`, agent.Name, agent.Content),
		}
		workflow.Phases = append(workflow.Phases, phase)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(exportOutput)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(exportOutput, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("\nSuccessfully exported agent to: %s\n", exportOutput)
	fmt.Printf("Workflow contains %d phase(s)\n", len(workflow.Phases))

	// Display summary
	fmt.Println("\nWorkflow Summary:")
	fmt.Printf("  Name:        %s\n", workflow.Name)
	fmt.Printf("  Version:     %s\n", workflow.Version)
	fmt.Printf("  Type:        %s\n", workflow.Type)
	fmt.Printf("  Phases:      %d\n", len(workflow.Phases))

	fmt.Println("\nTo run this workflow:")
	fmt.Printf("  1. Copy to skills directory: cp %s ~/.skillrunner/skills/%s/skill.yaml\n", exportOutput, agent.Name)
	fmt.Printf("  2. Execute: sr run %s \"your request here\"\n", agent.Name)

	return nil
}

// ExportedWorkflow represents an orchestrated workflow for export
type ExportedWorkflow struct {
	Name        string          `yaml:"name"`
	Version     string          `yaml:"version"`
	Description string          `yaml:"description"`
	Type        string          `yaml:"type"`
	Phases      []ExportedPhase `yaml:"phases"`
}

// ExportedPhase represents a phase in the exported workflow
type ExportedPhase struct {
	ID             string `yaml:"id"`
	TaskType       string `yaml:"task_type"`
	Description    string `yaml:"description"`
	PromptTemplate string `yaml:"prompt_template"`
}
