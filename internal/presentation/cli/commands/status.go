package commands

import (
	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// ProviderStatus represents the health status of a single provider.
type ProviderStatus struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Endpoint string   `json:"endpoint,omitempty"`
	Models   []string `json:"models,omitempty"`
	Latency  string   `json:"latency,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// SystemStatus represents the overall system health status.
type SystemStatus struct {
	Status       string           `json:"status"`
	Version      string           `json:"version"`
	Providers    []ProviderStatus `json:"providers"`
	ConfigLoaded bool             `json:"config_loaded"`
	ConfigPath   string           `json:"config_path,omitempty"`
	SkillsDir    string           `json:"skills_dir,omitempty"`
	SkillCount   int              `json:"skill_count"`
}

// NewStatusCmd creates the status command.
func NewStatusCmd() *cobra.Command {
	var detailed bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show system health status",
		Long: `Display the health status of the skillrunner system.

This includes:
  • Provider connectivity and health (Ollama, Anthropic, OpenAI, Groq)
  • Available models per provider
  • Configuration status
  • Skill availability

Use --detailed for additional diagnostic information.`,
		Example: `  # Show basic status
  sr status

  # Show detailed status with latency info
  sr status --detailed

  # Get status as JSON for scripting
  sr status -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(detailed)
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "show detailed status with latency and model info")

	return cmd
}

func runStatus(detailed bool) error {
	formatter := GetFormatter()

	// Get mock status data (simulated provider health)
	status := getMockSystemStatus()

	// Handle JSON output
	if formatter.Format() == output.FormatJSON {
		return formatter.JSON(status)
	}

	// Print text output
	return printStatusText(formatter, status, detailed)
}

// getMockSystemStatus returns simulated system status data.
func getMockSystemStatus() SystemStatus {
	return SystemStatus{
		Status:       "healthy",
		Version:      Version,
		ConfigLoaded: true,
		ConfigPath:   "~/.skillrunner/config.yaml",
		SkillsDir:    "~/.skillrunner/skills",
		SkillCount:   3,
		Providers: []ProviderStatus{
			{
				Name:     "ollama",
				Type:     "local",
				Status:   "healthy",
				Endpoint: "http://localhost:11434",
				Models:   []string{"llama3.2:latest", "codellama:13b", "mistral:latest"},
				Latency:  "12ms",
			},
			{
				Name:     "anthropic",
				Type:     "cloud",
				Status:   "healthy",
				Endpoint: "https://api.anthropic.com",
				Models:   []string{"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"},
				Latency:  "145ms",
			},
			{
				Name:     "openai",
				Type:     "cloud",
				Status:   "degraded",
				Endpoint: "https://api.openai.com",
				Models:   []string{"gpt-4o", "gpt-4o-mini"},
				Latency:  "890ms",
				Error:    "high latency detected",
			},
			{
				Name:     "groq",
				Type:     "cloud",
				Status:   "unavailable",
				Endpoint: "https://api.groq.com",
				Error:    "API key not configured",
			},
		},
	}
}

// printStatusText prints the status in human-readable format.
func printStatusText(formatter *output.Formatter, status SystemStatus, detailed bool) error {
	// System header
	formatter.Header("Skillrunner Status")
	formatter.Println("")

	// Overall status with color
	statusIndicator := getStatusIndicator(formatter, status.Status)
	formatter.Println("  %s  %s", formatter.Dim("System:"), statusIndicator)
	formatter.Println("  %s  %s", formatter.Dim("Version:"), status.Version)
	formatter.Println("")

	// Configuration
	formatter.SubHeader("Configuration")
	if status.ConfigLoaded {
		formatter.Success("Config loaded from %s", status.ConfigPath)
	} else {
		formatter.Warning("Using default configuration")
	}
	formatter.Println("  %s  %s (%d skills)", formatter.Dim("Skills Dir:"), status.SkillsDir, status.SkillCount)
	formatter.Println("")

	// Providers
	formatter.SubHeader("Providers")
	formatter.Println("")

	for _, provider := range status.Providers {
		printProviderStatus(formatter, provider, detailed)
	}

	// Summary
	formatter.Println("")
	healthy, degraded, unavailable := countProviderStatuses(status.Providers)
	formatter.Println("%s %d healthy, %d degraded, %d unavailable",
		formatter.Dim("Summary:"),
		healthy, degraded, unavailable)

	return nil
}

// printProviderStatus prints a single provider's status.
func printProviderStatus(formatter *output.Formatter, provider ProviderStatus, detailed bool) {
	statusIndicator := getStatusIndicator(formatter, provider.Status)
	typeLabel := formatter.Dim("[" + provider.Type + "]")

	formatter.Println("  %s %s %s", statusIndicator, formatter.Bold(provider.Name), typeLabel)

	if detailed {
		if provider.Endpoint != "" {
			formatter.Println("      %s %s", formatter.Dim("Endpoint:"), provider.Endpoint)
		}
		if provider.Latency != "" {
			formatter.Println("      %s %s", formatter.Dim("Latency:"), provider.Latency)
		}
		if len(provider.Models) > 0 {
			formatter.Println("      %s", formatter.Dim("Models:"))
			for _, model := range provider.Models {
				formatter.Println("        • %s", model)
			}
		}
	}

	if provider.Error != "" {
		formatter.Println("      %s", formatter.Colorize("Error: "+provider.Error, output.ColorRed))
	}
}

// getStatusIndicator returns a colored status indicator.
func getStatusIndicator(formatter *output.Formatter, status string) string {
	switch status {
	case "healthy":
		return formatter.Colorize("●", output.ColorGreen) + " " + formatter.Colorize("healthy", output.ColorGreen)
	case "degraded":
		return formatter.Colorize("●", output.ColorYellow) + " " + formatter.Colorize("degraded", output.ColorYellow)
	case "unavailable":
		return formatter.Colorize("●", output.ColorRed) + " " + formatter.Colorize("unavailable", output.ColorRed)
	default:
		return formatter.Colorize("●", output.ColorDim) + " " + formatter.Colorize("unknown", output.ColorDim)
	}
}

// countProviderStatuses counts providers by their status.
func countProviderStatuses(providers []ProviderStatus) (healthy, degraded, unavailable int) {
	for _, p := range providers {
		switch p.Status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		case "unavailable":
			unavailable++
		}
	}
	return
}
