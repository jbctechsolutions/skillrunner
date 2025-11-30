package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Skillrunner configuration",
	Long: `Initialize Skillrunner by creating configuration file and setting up API keys.

This command will:
- Create ~/.skillrunner/config.yaml
- Prompt for API keys (optional, can be set later)
- Test Docker connectivity
- Validate setup

Examples:
  sr init
  sr init --skip-docker-check`,
	RunE: runInit,
}

var (
	skipDockerCheck bool
	interactive     bool
)

func init() {
	initCmd.Flags().BoolVar(&skipDockerCheck, "skip-docker-check", false, "Skip Docker connectivity check")
	initCmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "Interactive mode (prompt for API keys)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing Skillrunner...")
	fmt.Println()

	// Create config manager
	configMgr, err := config.NewManager("")
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	cfg := configMgr.Get()
	configPath := configMgr.GetConfigPath()

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration already exists at: %s\n", configPath)
		fmt.Print("Do you want to overwrite it? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Initialization cancelled.")
			return nil
		}
	}

	// Set default router settings
	cfg.Router.LiteLLMURL = "http://localhost:18432"
	cfg.Router.OllamaURL = "http://localhost:18433"
	cfg.Router.AutoStart = true

	// Set default model defaults
	cfg.ModelDefaults = config.DefaultModelDefaults()

	// Interactive API key setup
	if interactive {
		fmt.Println("API Key Setup (press Enter to skip):")
		fmt.Println()

		// Anthropic
		fmt.Print("Anthropic API Key: ")
		var anthropicKey string
		fmt.Scanln(&anthropicKey)
		if strings.TrimSpace(anthropicKey) != "" {
			cfg.APIKeys.Anthropic = strings.TrimSpace(anthropicKey)
		}

		// OpenAI
		fmt.Print("OpenAI API Key: ")
		var openaiKey string
		fmt.Scanln(&openaiKey)
		if strings.TrimSpace(openaiKey) != "" {
			cfg.APIKeys.OpenAI = strings.TrimSpace(openaiKey)
		}

		// Google
		fmt.Print("Google API Key: ")
		var googleKey string
		fmt.Scanln(&googleKey)
		if strings.TrimSpace(googleKey) != "" {
			cfg.APIKeys.Google = strings.TrimSpace(googleKey)
		}
	}

	// Save config
	if err := configMgr.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✅ Configuration saved to: %s\n", configPath)
	fmt.Println()

	// Check Docker
	if !skipDockerCheck {
		fmt.Println("Checking Docker...")
		if err := checkDocker(); err != nil {
			fmt.Printf("⚠️  Docker check failed: %v\n", err)
			fmt.Println("   Make sure Docker is installed and running.")
		} else {
			fmt.Println("✅ Docker is available")
		}
		fmt.Println()
	}

	// Check docker-compose.yml
	fmt.Println("Checking for docker-compose.yml...")
	if _, err := os.Stat("docker-compose.yml"); err == nil {
		fmt.Println("✅ docker-compose.yml found")
	} else {
		fmt.Println("⚠️  docker-compose.yml not found in current directory")
		fmt.Println("   Make sure you're in the Skillrunner project directory.")
	}
	fmt.Println()

	fmt.Println("Initialization complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Start Docker services: docker-compose up -d")
	fmt.Println("  2. Pull Ollama models: docker exec skillrunner-ollama ollama pull deepseek-coder-v2:16b")
	fmt.Println("  3. Try a skill: sr run test 'hello world'")
	fmt.Println()

	return nil
}

func checkDocker() error {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
