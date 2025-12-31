package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/crypto"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// InitResult holds the result of the init command for JSON output.
type InitResult struct {
	ConfigDir   string `json:"config_dir"`
	ConfigFile  string `json:"config_file"`
	SkillsDir   string `json:"skills_dir"`
	Initialized bool   `json:"initialized"`
}

// NewInitCmd creates the init command.
func NewInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize skillrunner configuration",
		Long: `Initialize skillrunner configuration interactively.

This command creates the ~/.skillrunner/ directory structure and
generates a config.yaml file with your provider settings.

The initialization process will:
  • Create ~/.skillrunner/ directory
  • Create ~/.skillrunner/skills/ directory for skill definitions
  • Generate ~/.skillrunner/config.yaml with provider configurations
  • Prompt for Ollama endpoint and optional cloud provider API keys`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing configuration")

	return cmd
}

// prompter handles interactive user input.
type prompter struct {
	reader    *bufio.Reader
	formatter *output.Formatter
}

// newPrompter creates a new prompter.
func newPrompter(formatter *output.Formatter) *prompter {
	return &prompter{
		reader:    bufio.NewReader(os.Stdin),
		formatter: formatter,
	}
}

// prompt asks a question and returns the answer (or default if empty).
func (p *prompter) prompt(question, defaultValue string) (string, error) {
	if defaultValue != "" {
		p.formatter.Print("%s [%s]: ", question, defaultValue)
	} else {
		p.formatter.Print("%s: ", question)
	}

	answer, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultValue, nil
	}
	return answer, nil
}

// promptSecret asks for sensitive input (displayed as-is since Go stdlib doesn't support hidden input easily).
func (p *prompter) promptSecret(question string) (string, error) {
	p.formatter.Print("%s: ", question)

	answer, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(answer), nil
}

// promptYesNo asks a yes/no question and returns true for yes.
func (p *prompter) promptYesNo(question string, defaultYes bool) (bool, error) {
	defaultStr := "[y/N]"
	if defaultYes {
		defaultStr = "[Y/n]"
	}

	p.formatter.Print("%s %s: ", question, defaultStr)

	answer, err := p.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "" {
		return defaultYes, nil
	}

	return answer == "y" || answer == "yes", nil
}

func runInit(force bool) error {
	// Create formatter - don't use colors for prompts to keep it clean
	format := output.FormatText
	if globalFlags.Output == "json" {
		format = output.FormatJSON
	}

	formatter := output.NewFormatter(
		output.WithFormat(format),
		output.WithColor(format != output.FormatJSON),
	)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".skillrunner")
	configFile := filepath.Join(configDir, "config.yaml")
	skillsDir := filepath.Join(configDir, "skills")

	// Check if already initialized
	if _, err := os.Stat(configFile); err == nil && !force {
		if format == output.FormatJSON {
			return formatter.JSON(InitResult{
				ConfigDir:   configDir,
				ConfigFile:  configFile,
				SkillsDir:   skillsDir,
				Initialized: false,
			})
		}
		formatter.Warning("Configuration already exists at %s", configFile)
		formatter.Info("Use --force to overwrite existing configuration")
		return nil
	}

	// For JSON output, skip interactive prompts and use defaults
	if format == output.FormatJSON {
		cfg := config.NewDefaultConfig()
		if err := writeConfig(configDir, skillsDir, configFile, cfg); err != nil {
			return err
		}
		return formatter.JSON(InitResult{
			ConfigDir:   configDir,
			ConfigFile:  configFile,
			SkillsDir:   skillsDir,
			Initialized: true,
		})
	}

	// Interactive setup
	formatter.Header("Skillrunner Configuration")
	formatter.Println("")
	formatter.Info("This wizard will help you set up skillrunner.")
	formatter.Println("")

	p := newPrompter(formatter)

	// Create default config
	cfg := config.NewDefaultConfig()

	// Ollama configuration
	formatter.SubHeader("Local Provider (Ollama)")
	formatter.Println("")

	ollamaURL, err := p.prompt("Ollama URL", config.DefaultOllamaURL)
	if err != nil {
		return err
	}
	cfg.Providers.Ollama.URL = ollamaURL

	enableOllama, err := p.promptYesNo("Enable Ollama", true)
	if err != nil {
		return err
	}
	cfg.Providers.Ollama.Enabled = enableOllama

	formatter.Println("")

	// Cloud providers
	formatter.SubHeader("Cloud Providers (Optional)")
	formatter.Println("")
	formatter.Println("%s", formatter.Dim("API keys will be stored encrypted in config.yaml"))
	formatter.Println("")

	// Initialize encryptor for API keys
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		return fmt.Errorf("failed to initialize encryption: %w", err)
	}

	// Anthropic
	configureAnthropic, err := p.promptYesNo("Configure Anthropic (Claude)", false)
	if err != nil {
		return err
	}
	if configureAnthropic {
		apiKey, err := p.promptSecret("Anthropic API key")
		if err != nil {
			return err
		}
		if apiKey != "" {
			encryptedKey, err := encryptor.Encrypt(apiKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt Anthropic API key: %w", err)
			}
			cfg.Providers.Anthropic.APIKeyEncrypted = encryptedKey
			cfg.Providers.Anthropic.Enabled = true
		}
	}

	// OpenAI
	configureOpenAI, err := p.promptYesNo("Configure OpenAI", false)
	if err != nil {
		return err
	}
	if configureOpenAI {
		apiKey, err := p.promptSecret("OpenAI API key")
		if err != nil {
			return err
		}
		if apiKey != "" {
			encryptedKey, err := encryptor.Encrypt(apiKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt OpenAI API key: %w", err)
			}
			cfg.Providers.OpenAI.APIKeyEncrypted = encryptedKey
			cfg.Providers.OpenAI.Enabled = true
		}
	}

	// Groq
	configureGroq, err := p.promptYesNo("Configure Groq", false)
	if err != nil {
		return err
	}
	if configureGroq {
		apiKey, err := p.promptSecret("Groq API key")
		if err != nil {
			return err
		}
		if apiKey != "" {
			encryptedKey, err := encryptor.Encrypt(apiKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt Groq API key: %w", err)
			}
			cfg.Providers.Groq.APIKeyEncrypted = encryptedKey
			cfg.Providers.Groq.Enabled = true
		}
	}

	formatter.Println("")

	// Write configuration
	if err := writeConfig(configDir, skillsDir, configFile, cfg); err != nil {
		return err
	}

	formatter.Println("")
	formatter.Success("Configuration initialized successfully!")
	formatter.Println("")
	formatter.Item("Config directory", configDir)
	formatter.Item("Config file", configFile)
	formatter.Item("Skills directory", skillsDir)
	formatter.Println("")
	formatter.Info("Run 'sr list' to see available skills")
	formatter.Info("Run 'sr run <skill>' to execute a skill")

	return nil
}

// writeConfig creates directories and writes the configuration file.
func writeConfig(configDir, skillsDir, configFile string, cfg *config.Config) error {
	// Create config directory
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create skills directory
	if err := os.MkdirAll(skillsDir, 0750); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment
	header := `# Skillrunner Configuration
# Generated by 'sr init'
#
# Documentation: https://github.com/jbctechsolutions/skillrunner
#
`
	content := header + string(data)

	// Write config file with restricted permissions
	if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
