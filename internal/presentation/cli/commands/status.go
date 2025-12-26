package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	appProvider "github.com/jbctechsolutions/skillrunner/internal/application/provider"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// ProviderStatus represents the health status of a single provider.
type ProviderStatus struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Status    string   `json:"status"`
	Endpoint  string   `json:"endpoint,omitempty"`
	Models    []string `json:"models,omitempty"`
	Latency   string   `json:"latency,omitempty"`
	Error     string   `json:"error,omitempty"`
	APIKeySet bool     `json:"api_key_set,omitempty"`
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
	var checkHealth bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show system health status",
		Long: `Display the health status of the skillrunner system.

This includes:
  • Provider connectivity and health (Ollama, Anthropic, OpenAI, Groq)
  • Available models per provider
  • Configuration status
  • Skill availability

Use --detailed for additional diagnostic information.
Use --check to perform live health checks on providers.`,
		Example: `  # Show basic status
  sr status

  # Show detailed status with latency info
  sr status --detailed

  # Perform live health checks
  sr status --check

  # Get status as JSON for scripting
  sr status -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(detailed, checkHealth)
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "show detailed status with latency and model info")
	cmd.Flags().BoolVar(&checkHealth, "check", false, "perform live health checks on providers")

	return cmd
}

func runStatus(detailed bool, checkHealth bool) error {
	formatter := GetFormatter()

	// Get real status from container
	status := getSystemStatus(checkHealth)

	// Handle JSON output
	if formatter.Format() == output.FormatJSON {
		return formatter.JSON(status)
	}

	// Print text output
	return printStatusText(formatter, status, detailed)
}

// getSystemStatus returns the actual system status from the container.
func getSystemStatus(checkHealth bool) SystemStatus {
	container := GetContainer()

	status := SystemStatus{
		Status:       "healthy",
		Version:      Version,
		ConfigLoaded: true,
		ConfigPath:   "~/.skillrunner/config.yaml",
		SkillsDir:    "~/.skillrunner/skills",
		SkillCount:   0,
	}

	// Get skill count if container is available
	if container != nil {
		if registry := container.SkillRegistry(); registry != nil {
			status.SkillCount = registry.Count()
		}

		// Get config info
		if cfg := container.Config(); cfg != nil {
			status.SkillsDir = cfg.Skills.Directory
		}
	}

	// Get provider status
	status.Providers = getProviderStatuses(container, checkHealth)

	// Determine overall status based on providers
	status.Status = determineOverallStatus(status.Providers)

	return status
}

// getProviderStatuses returns the status of all providers.
func getProviderStatuses(container interface {
	ProviderInitializer() *appProvider.Initializer
}, checkHealth bool) []ProviderStatus {
	// Define known providers in order
	knownProviders := []string{"ollama", "anthropic", "openai", "groq"}
	providerTypes := map[string]string{
		"ollama":    "local",
		"anthropic": "cloud",
		"openai":    "cloud",
		"groq":      "cloud",
	}

	// If container is nil, return all providers as unavailable
	if container == nil {
		providers := make([]ProviderStatus, 0, len(knownProviders))
		for _, name := range knownProviders {
			providers = append(providers, ProviderStatus{
				Name:   name,
				Type:   providerTypes[name],
				Status: "unavailable",
				Error:  "container not initialized",
			})
		}
		return providers
	}

	initializer := container.ProviderInitializer()
	if initializer == nil {
		providers := make([]ProviderStatus, 0, len(knownProviders))
		for _, name := range knownProviders {
			providers = append(providers, ProviderStatus{
				Name:   name,
				Type:   providerTypes[name],
				Status: "unavailable",
				Error:  "provider initializer not available",
			})
		}
		return providers
	}

	// Perform health checks if requested
	var healthData map[string]*appProvider.ProviderHealth
	if checkHealth {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		healthData = initializer.CheckHealth(ctx)
	} else {
		healthData = initializer.GetAllHealth()
	}

	providers := make([]ProviderStatus, 0, len(knownProviders))
	for _, name := range knownProviders {
		ps := ProviderStatus{
			Name: name,
			Type: providerTypes[name],
		}

		health, exists := healthData[name]
		if !exists || health == nil {
			ps.Status = "unavailable"
			ps.Error = "not configured"
			providers = append(providers, ps)
			continue
		}

		ps.Type = health.Type
		ps.Endpoint = health.Endpoint
		ps.Models = health.Models
		ps.APIKeySet = health.APIKeySet

		if !health.Enabled {
			ps.Status = "unavailable"
			ps.Error = "disabled in configuration"
			if health.Type == "cloud" && !health.APIKeySet {
				ps.Error = "API key not configured"
			}
		} else if health.Healthy {
			ps.Status = "healthy"
			if health.Latency > 0 {
				ps.Latency = formatLatency(health.Latency)
			}
		} else if health.Error != "" {
			// Check if it's a connection issue vs degraded performance
			if health.Latency > 500*time.Millisecond {
				ps.Status = "degraded"
				ps.Latency = formatLatency(health.Latency)
				ps.Error = "high latency detected"
			} else {
				ps.Status = "unavailable"
				ps.Error = health.Error
			}
		} else {
			// Enabled but health not yet checked
			ps.Status = "unknown"
		}

		providers = append(providers, ps)
	}

	return providers
}

// formatLatency formats a duration as a human-readable string.
func formatLatency(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

// determineOverallStatus determines the overall system status based on providers.
func determineOverallStatus(providers []ProviderStatus) string {
	healthy := 0
	degraded := 0
	unavailable := 0

	for _, p := range providers {
		switch p.Status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		case "unavailable", "unknown":
			unavailable++
		}
	}

	// If no providers are healthy, system is unhealthy
	if healthy == 0 {
		return "unhealthy"
	}

	// If any providers are degraded, system is degraded
	if degraded > 0 {
		return "degraded"
	}

	return "healthy"
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
		// Show API key status for cloud providers
		if provider.Type == "cloud" {
			if provider.APIKeySet {
				formatter.Println("      %s %s", formatter.Dim("API Key:"), formatter.Colorize("configured", output.ColorGreen))
			} else {
				formatter.Println("      %s %s", formatter.Dim("API Key:"), formatter.Colorize("not configured", output.ColorRed))
			}
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
	case "unhealthy":
		return formatter.Colorize("●", output.ColorRed) + " " + formatter.Colorize("unhealthy", output.ColorRed)
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
		case "unavailable", "unknown":
			unavailable++
		}
	}
	return
}
