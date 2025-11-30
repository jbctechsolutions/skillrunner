package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/jbctechsolutions/skillrunner/internal/marketplace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	sourceType      string
	sourcePath      string
	sourceOwner     string
	sourceRepo      string
	sourceBranch    string
	sourcePackage   string
	sourceURL       string
	sourcePriority  int
	sourceAuthToken string
)

var marketplaceSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage marketplace sources",
	Long: `Manage marketplace sources for skills, agents, and commands.

Skillrunner supports multiple marketplace sources including:
  - local:    Local filesystem directories
  - github:   GitHub repositories
  - npm:      NPM packages (like aitmpl.com templates)
  - http:     Generic HTTP/HTTPS endpoints
  - registry: Marketplace registries`,
}

var marketplaceSourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured sources",
	Long: `List all configured marketplace sources.

Examples:
  sr marketplace source list
  sr marketplace source list --format json`,
	RunE: listMarketplaceSources,
}

var marketplaceSourceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new marketplace source",
	Long: `Add a new marketplace source.

Examples:
  # Add a local source
  sr marketplace source add my-skills --type local --path ~/my-skills

  # Add a GitHub source
  sr marketplace source add community --type github --owner anthropics --repo claude-code-templates

  # Add an NPM source (aitmpl.com compatible)
  sr marketplace source add aitmpl --type npm --package claude-code-templates

  # Add an HTTP source
  sr marketplace source add custom --type http --url https://example.com/marketplace/manifest.json`,
	Args: cobra.ExactArgs(1),
	RunE: addMarketplaceSource,
}

var marketplaceSourceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a marketplace source",
	Long: `Remove a marketplace source from the configuration.

Examples:
  sr marketplace source remove my-skills`,
	Args: cobra.ExactArgs(1),
	RunE: removeMarketplaceSource,
}

var marketplaceSourceRefreshCmd = &cobra.Command{
	Use:   "refresh [name]",
	Short: "Refresh marketplace source(s)",
	Long: `Refresh data from marketplace sources.

Examples:
  sr marketplace source refresh           # Refresh all sources
  sr marketplace source refresh my-source # Refresh specific source`,
	RunE: refreshMarketplaceSources,
}

var marketplaceSourceHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health of marketplace sources",
	Long: `Check the health and accessibility of all configured sources.

Examples:
  sr marketplace source health`,
	RunE: checkMarketplaceSourceHealth,
}

func init() {
	// List command flags
	marketplaceSourceListCmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table|json)")

	// Add command flags
	marketplaceSourceAddCmd.Flags().StringVar(&sourceType, "type", "", "Source type (local|github|npm|http|registry)")
	marketplaceSourceAddCmd.Flags().StringVar(&sourcePath, "path", "", "Local filesystem path (for local type)")
	marketplaceSourceAddCmd.Flags().StringVar(&sourceOwner, "owner", "", "GitHub owner/org (for github type)")
	marketplaceSourceAddCmd.Flags().StringVar(&sourceRepo, "repo", "", "GitHub repository (for github type)")
	marketplaceSourceAddCmd.Flags().StringVar(&sourceBranch, "branch", "main", "GitHub branch (for github type)")
	marketplaceSourceAddCmd.Flags().StringVar(&sourcePackage, "package", "", "NPM package name (for npm type)")
	marketplaceSourceAddCmd.Flags().StringVar(&sourceURL, "url", "", "HTTP URL (for http type)")
	marketplaceSourceAddCmd.Flags().IntVar(&sourcePriority, "priority", 50, "Source priority (lower = higher priority)")
	marketplaceSourceAddCmd.Flags().StringVar(&sourceAuthToken, "auth-token", "", "Authentication token")
	marketplaceSourceAddCmd.MarkFlagRequired("type")

	// Add subcommands
	marketplaceSourceCmd.AddCommand(marketplaceSourceListCmd)
	marketplaceSourceCmd.AddCommand(marketplaceSourceAddCmd)
	marketplaceSourceCmd.AddCommand(marketplaceSourceRemoveCmd)
	marketplaceSourceCmd.AddCommand(marketplaceSourceRefreshCmd)
	marketplaceSourceCmd.AddCommand(marketplaceSourceHealthCmd)

	// Add source command to marketplace
	marketplaceCmd.AddCommand(marketplaceSourceCmd)
}

// getSourcesConfigPath returns the path to the sources configuration file
func getSourcesConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".skillrunner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sources.yaml"), nil
}

// SourcesConfig represents the sources configuration file
type SourcesConfig struct {
	Sources []marketplace.SourceConfig `yaml:"sources"`
}

// loadSourcesConfig loads the sources configuration
func loadSourcesConfig() (*SourcesConfig, error) {
	configPath, err := getSourcesConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default configuration
		registry := marketplace.NewRegistry()
		return &SourcesConfig{
			Sources: registry.DefaultConfigs(),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config SourcesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// saveSourcesConfig saves the sources configuration
func saveSourcesConfig(config *SourcesConfig) error {
	configPath, err := getSourcesConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func listMarketplaceSources(cmd *cobra.Command, args []string) error {
	config, err := loadSourcesConfig()
	if err != nil {
		return fmt.Errorf("failed to load sources config: %w", err)
	}

	if len(config.Sources) == 0 {
		fmt.Println("No marketplace sources configured.")
		return nil
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(config.Sources, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling sources: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tPRIORITY\tENABLED\tLOCATION")
		fmt.Fprintln(w, "----\t----\t--------\t-------\t--------")

		for _, source := range config.Sources {
			enabled := "Yes"
			if !source.Enabled {
				enabled = "No"
			}

			location := ""
			switch source.Type {
			case marketplace.SourceTypeLocal:
				location = source.Path
				if location == "" {
					location = "(default)"
				}
			case marketplace.SourceTypeGitHub:
				location = fmt.Sprintf("%s/%s", source.Owner, source.Repo)
			case marketplace.SourceTypeNPM:
				location = source.Package
			case marketplace.SourceTypeHTTP, marketplace.SourceTypeRegistry:
				location = source.URL
			}

			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				source.Name,
				source.Type,
				source.Priority,
				enabled,
				location)
		}

		w.Flush()
		fmt.Printf("\nTotal: %d sources\n", len(config.Sources))
	}

	return nil
}

func addMarketplaceSource(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate source type
	var st marketplace.SourceType
	switch sourceType {
	case "local":
		st = marketplace.SourceTypeLocal
		if sourcePath == "" {
			return fmt.Errorf("--path is required for local sources")
		}
	case "github":
		st = marketplace.SourceTypeGitHub
		if sourceOwner == "" || sourceRepo == "" {
			return fmt.Errorf("--owner and --repo are required for github sources")
		}
	case "npm":
		st = marketplace.SourceTypeNPM
		if sourcePackage == "" {
			return fmt.Errorf("--package is required for npm sources")
		}
	case "http":
		st = marketplace.SourceTypeHTTP
		if sourceURL == "" {
			return fmt.Errorf("--url is required for http sources")
		}
	case "registry":
		st = marketplace.SourceTypeRegistry
		if sourceURL == "" {
			return fmt.Errorf("--url is required for registry sources")
		}
	default:
		return fmt.Errorf("invalid source type: %s (valid types: local, github, npm, http, registry)", sourceType)
	}

	config, err := loadSourcesConfig()
	if err != nil {
		return fmt.Errorf("failed to load sources config: %w", err)
	}

	// Check if source already exists
	for _, s := range config.Sources {
		if s.Name == name {
			return fmt.Errorf("source already exists: %s", name)
		}
	}

	// Create new source config
	newSource := marketplace.SourceConfig{
		Name:      name,
		Type:      st,
		Priority:  sourcePriority,
		Enabled:   true,
		Path:      sourcePath,
		Owner:     sourceOwner,
		Repo:      sourceRepo,
		Branch:    sourceBranch,
		Package:   sourcePackage,
		URL:       sourceURL,
		AuthToken: sourceAuthToken,
	}

	config.Sources = append(config.Sources, newSource)

	if err := saveSourcesConfig(config); err != nil {
		return fmt.Errorf("failed to save sources config: %w", err)
	}

	fmt.Printf("Successfully added source '%s' (%s)\n", name, st)
	return nil
}

func removeMarketplaceSource(cmd *cobra.Command, args []string) error {
	name := args[0]

	config, err := loadSourcesConfig()
	if err != nil {
		return fmt.Errorf("failed to load sources config: %w", err)
	}

	found := false
	var newSources []marketplace.SourceConfig
	for _, s := range config.Sources {
		if s.Name == name {
			found = true
			continue
		}
		newSources = append(newSources, s)
	}

	if !found {
		return fmt.Errorf("source not found: %s", name)
	}

	config.Sources = newSources

	if err := saveSourcesConfig(config); err != nil {
		return fmt.Errorf("failed to save sources config: %w", err)
	}

	fmt.Printf("Successfully removed source '%s'\n", name)
	return nil
}

func refreshMarketplaceSources(cmd *cobra.Command, args []string) error {
	config, err := loadSourcesConfig()
	if err != nil {
		return fmt.Errorf("failed to load sources config: %w", err)
	}

	registry := marketplace.NewRegistry()

	// Filter to specific source if name provided
	var sourcesToRefresh []marketplace.SourceConfig
	if len(args) > 0 {
		name := args[0]
		for _, s := range config.Sources {
			if s.Name == name {
				sourcesToRefresh = append(sourcesToRefresh, s)
				break
			}
		}
		if len(sourcesToRefresh) == 0 {
			return fmt.Errorf("source not found: %s", name)
		}
	} else {
		sourcesToRefresh = config.Sources
	}

	fmt.Printf("Refreshing %d source(s)...\n\n", len(sourcesToRefresh))

	for _, sourceConfig := range sourcesToRefresh {
		if !sourceConfig.Enabled {
			fmt.Printf("  %-20s SKIPPED (disabled)\n", sourceConfig.Name)
			continue
		}

		source, err := marketplace.NewSource(sourceConfig)
		if err != nil {
			fmt.Printf("  %-20s ERROR: %v\n", sourceConfig.Name, err)
			continue
		}

		if err := registry.AddSource(source); err != nil {
			fmt.Printf("  %-20s ERROR: %v\n", sourceConfig.Name, err)
			continue
		}

		fmt.Printf("  %-20s OK\n", sourceConfig.Name)
	}

	fmt.Println("\nRefresh complete.")
	return nil
}

func checkMarketplaceSourceHealth(cmd *cobra.Command, args []string) error {
	config, err := loadSourcesConfig()
	if err != nil {
		return fmt.Errorf("failed to load sources config: %w", err)
	}

	fmt.Println("Checking marketplace source health...")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tSTATUS\tDETAILS")
	fmt.Fprintln(w, "----\t----\t------\t-------")

	for _, sourceConfig := range config.Sources {
		status := "HEALTHY"
		details := ""

		if !sourceConfig.Enabled {
			status = "DISABLED"
			details = "Source is disabled"
		} else {
			source, err := marketplace.NewSource(sourceConfig)
			if err != nil {
				status = "ERROR"
				details = err.Error()
			} else {
				if !source.IsHealthy(cmd.Context()) {
					status = "UNHEALTHY"
					details = "Source is not accessible"
				}
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			sourceConfig.Name,
			sourceConfig.Type,
			status,
			details)
	}

	w.Flush()
	return nil
}
