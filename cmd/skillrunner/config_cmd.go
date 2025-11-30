package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Skillrunner configuration",
	Long:  `View and modify Skillrunner configuration settings.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Supported keys:
  workspace       - Path to the workspace directory
  default_model    - Default model to use (e.g., gpt-4, claude-3)
  output_format    - Output format (table|json)
  compact_output   - Compact output mode (true|false)

Examples:
  skill config set workspace /path/to/workspace
  skill config set default_model gpt-4
  skill config set output_format json
  skill config set compact_output true`,
	Args: cobra.ExactArgs(2),
	RunE: setConfig,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current Skillrunner configuration settings.`,
	RunE:  showConfig,
}

var configFormat string

func init() {
	configShowCmd.Flags().StringVarP(&configFormat, "format", "f", "table", "Output format (table|json)")
}

func setConfig(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])
	value := args[1]

	cm := NewConfigManager()
	config, err := cm.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update the specified key
	switch key {
	case "workspace":
		config.Workspace = value
	case "default_model":
		config.DefaultModel = value
	case "output_format":
		if value != "table" && value != "json" {
			return fmt.Errorf("invalid output_format: must be 'table' or 'json'")
		}
		config.OutputFormat = value
	case "compact_output":
		compact, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid compact_output value: must be 'true' or 'false'")
		}
		config.CompactOutput = compact
	default:
		return fmt.Errorf("invalid key: %s. Supported keys: workspace, default_model, output_format, compact_output", key)
	}

	// Save the updated config
	if err := cm.Save(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Configuration updated: %s = %s\n", key, value)
	return nil
}

func showConfig(cmd *cobra.Command, args []string) error {
	cm := NewConfigManager()
	config, err := cm.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if configFormat == "json" {
		jsonOutput, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling config: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		// Table format
		fmt.Println("\nSkillrunner Configuration:")
		fmt.Println("----------------------------------------------------------------------")
		fmt.Printf("  Workspace:       %s\n", getValueOrEmpty(config.Workspace))
		fmt.Printf("  Default Model:  %s\n", getValueOrEmpty(config.DefaultModel))
		fmt.Printf("  Output Format:  %s\n", getValueOrDefault(config.OutputFormat, "table"))
		fmt.Printf("  Compact Output: %v\n", config.CompactOutput)
		if len(config.Models) > 0 {
			fmt.Printf("  Models:          %s\n", strings.Join(config.Models, ", "))
		}
		fmt.Printf("  Config File:     %s\n", cm.GetConfigPath())
		fmt.Println()
	}

	return nil
}

func getValueOrEmpty(value string) string {
	if value == "" {
		return "(not set)"
	}
	return value
}

func getValueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
