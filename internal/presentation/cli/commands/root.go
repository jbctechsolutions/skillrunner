// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
	Config     *config.Config
	Formatter  *output.Formatter
	Flags      *GlobalFlags
	Container  *application.Container
	cancelFunc context.CancelFunc
}

var (
	globalFlags GlobalFlags
	appCtx      *AppContext
	appCtxMu    sync.RWMutex // Protects appCtx for thread-safe access
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
	rootCmd.AddCommand(NewMemoryCmd())

	// Session and workspace management
	rootCmd.AddCommand(NewSessionCmd())
	rootCmd.AddCommand(NewWorkspaceCmd())

	// Wave 10: Cache management
	rootCmd.AddCommand(NewCacheCmd())

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

	// Load or create default config using the new loader
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

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Store app context with mutex protection
	appCtxMu.Lock()
	appCtx = &AppContext{
		Config:     cfg,
		Formatter:  formatter,
		Flags:      &globalFlags,
		Container:  container,
		cancelFunc: cancel,
	}
	appCtxMu.Unlock()

	// Start skill hot reload watcher in background
	if err := container.StartSkillWatching(ctx); err != nil {
		if globalFlags.Verbose {
			formatter.Warning("Could not start skill hot reload: %v", err)
		}
	}

	return nil
}

// loadConfig loads configuration from the specified file or default location.
func loadConfig(configPath string) (*config.Config, error) {
	loader, err := config.NewLoader("")
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	return loader.Load(configPath)
}

// GetAppContext returns the current application context.
// Returns nil if the app hasn't been initialized.
// Thread-safe via mutex protection.
func GetAppContext() *AppContext {
	appCtxMu.RLock()
	defer appCtxMu.RUnlock()
	return appCtx
}

// GetFormatter returns the output formatter.
// Creates a default formatter if app context is not initialized.
// Thread-safe via mutex protection.
func GetFormatter() *output.Formatter {
	appCtxMu.RLock()
	ctx := appCtx
	appCtxMu.RUnlock()

	if ctx != nil {
		return ctx.Formatter
	}
	return output.NewFormatter()
}

// GetContainer returns the application container.
// Returns nil if the app hasn't been initialized.
// Thread-safe via mutex protection.
func GetContainer() *application.Container {
	appCtxMu.RLock()
	ctx := appCtx
	appCtxMu.RUnlock()

	if ctx != nil {
		return ctx.Container
	}
	return nil
}

// Shutdown performs graceful shutdown of the application.
// Cancels the context and cleans up resources.
func Shutdown() {
	appCtxMu.Lock()
	defer appCtxMu.Unlock()

	if appCtx != nil && appCtx.cancelFunc != nil {
		appCtx.cancelFunc()
	}
}

// Execute runs the root command with graceful shutdown support.
func Execute() {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run command in a goroutine
	errChan := make(chan error, 1)
	go func() {
		rootCmd := NewRootCmd()
		errChan <- rootCmd.Execute()
	}()

	// Wait for either command completion or signal
	select {
	case err := <-errChan:
		if err != nil {
			formatter := GetFormatter()
			formatter.Error("%s", err.Error())
			Shutdown()
			os.Exit(1)
		}
	case sig := <-sigChan:
		formatter := GetFormatter()
		formatter.Warning("Received signal %v, shutting down...", sig)
		Shutdown()
		os.Exit(130) // Standard exit code for SIGINT
	}

	Shutdown()
}
