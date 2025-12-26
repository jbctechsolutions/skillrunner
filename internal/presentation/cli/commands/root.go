// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// Version information - set at build time via ldflags.
var (
	Version   = "0.1.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// GlobalFlags holds the global CLI flags.
type GlobalFlags struct {
	ConfigFile string
	Output     string
	Verbose    bool
}

// AppContext holds the application runtime context.
type AppContext struct {
	Config    *config.Config
	Formatter *output.Formatter
	Flags     *GlobalFlags
	Container *application.Container
}

var (
	globalFlags GlobalFlags
	appCtx      *AppContext
)

// NewRootCmd creates the root command for the skillrunner CLI.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sr",
		Short: "Skillrunner - Local-first AI workflow orchestration",
		Long: `Skillrunner (sr) is a local-first AI workflow orchestration tool.

It enables multi-phase AI workflows that prioritize local LLM providers
(like Ollama) while seamlessly falling back to cloud providers when needed.

Key features:
  • Multi-phase workflow execution with DAG support
  • Intelligent provider routing (local-first, cost-aware, performance-based)
  • Skill-based workflow definitions
  • Provider health monitoring and automatic failover`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip initialization for help, version, init, and completion commands
			if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" || cmd.Name() == "init" {
				return nil
			}
			return initializeApp()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&globalFlags.ConfigFile, "config", "c", "", "config file path (default: ~/.skillrunner/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&globalFlags.Output, "output", "o", "text", "output format: text, json")
	rootCmd.PersistentFlags().BoolVarP(&globalFlags.Verbose, "verbose", "v", false, "enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewListCmd())
	rootCmd.AddCommand(NewRunCmd())
	rootCmd.AddCommand(NewStatusCmd())
	rootCmd.AddCommand(NewAskCmd())
	rootCmd.AddCommand(NewChatCmd())
	rootCmd.AddCommand(NewImportCmd())
	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewMetricsCmd())
	rootCmd.AddCommand(NewContextCmd())

	// Session and workspace management
	rootCmd.AddCommand(NewSessionCmd())
	rootCmd.AddCommand(NewWorkspaceCmd())

	return rootCmd
}

// initializeApp initializes the application context.
func initializeApp() error {
	// Determine output format
	format := output.FormatText
	if globalFlags.Output == "json" {
		format = output.FormatJSON
	}

	// Create formatter
	formatter := output.NewFormatter(
		output.WithFormat(format),
		output.WithColor(format != output.FormatJSON),
	)

	// Load or create default config
	cfg, err := loadConfig(globalFlags.ConfigFile)
	if err != nil {
		if globalFlags.Verbose {
			formatter.Warning("Could not load config: %v, using defaults", err)
		}
		cfg = config.NewDefaultConfig()
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize the application container with all dependencies
	container, err := application.NewContainer(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Store app context
	appCtx = &AppContext{
		Config:    cfg,
		Formatter: formatter,
		Flags:     &globalFlags,
		Container: container,
	}

	return nil
}

// loadConfig loads configuration from the specified file or default location.
func loadConfig(configPath string) (*config.Config, error) {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".skillrunner", "config.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// TODO: Implement actual YAML loading when we have a config loader
	// For now, return default config
	return config.NewDefaultConfig(), nil
}

// GetAppContext returns the current application context.
// Returns nil if the app hasn't been initialized.
func GetAppContext() *AppContext {
	return appCtx
}

// GetFormatter returns the output formatter.
// Creates a default formatter if app context is not initialized.
func GetFormatter() *output.Formatter {
	if appCtx != nil {
		return appCtx.Formatter
	}
	return output.NewFormatter()
}

// GetContainer returns the application container.
// Returns nil if the app hasn't been initialized.
func GetContainer() *application.Container {
	if appCtx != nil {
		return appCtx.Container
	}
	return nil
}

// Execute runs the root command.
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		formatter := GetFormatter()
		formatter.Error("%s", err.Error())
		os.Exit(1)
	}
}
