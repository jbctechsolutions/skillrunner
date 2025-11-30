package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/jbctechsolutions/skillrunner/internal/embedded"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Skillrunner configuration",
	Long: `Initialize Skillrunner by creating configuration file and setting up API keys.

This command will:
- Create ~/.skillrunner/ directory structure
- Create ~/.skillrunner/config.yaml
- Install bundled starter skills
- Prompt for API keys (optional, can be set later)
- Test Docker connectivity

Examples:
  sr init
  sr init --skip-docker-check
  sr init --non-interactive`,
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

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".skillrunner")

	// Create all required directories
	fmt.Println("Creating directory structure...")
	directories := []string{
		baseDir,
		filepath.Join(baseDir, "skills"),
		filepath.Join(baseDir, "cache"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	fmt.Printf("  Created: %s/\n", baseDir)
	fmt.Printf("  Created: %s/skills/\n", baseDir)
	fmt.Printf("  Created: %s/cache/\n", baseDir)
	fmt.Println()

	// Create config manager
	configMgr, err := config.NewManager("")
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	cfg := configMgr.Get()
	configPath := configMgr.GetConfigPath()

	// Check if config already exists
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
		if interactive {
			fmt.Printf("Configuration already exists at: %s\n", configPath)
			fmt.Print("Do you want to overwrite it? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Println("Keeping existing configuration.")
			} else {
				configExists = false // Will overwrite
			}
		}
	}

	if !configExists {
		// Set default router settings
		cfg.Router.LiteLLMURL = "http://localhost:18432"
		cfg.Router.OllamaURL = "http://localhost:18433"
		cfg.Router.AutoStart = false // User must explicitly enable auto-start

		// Set default model defaults
		cfg.ModelDefaults = config.DefaultModelDefaults()

		// Interactive API key setup
		if interactive {
			fmt.Println()
			fmt.Println("API Key Setup (press Enter to skip):")
			fmt.Println("Note: Ollama (local) is enabled by default. Anthropic is optional for cloud models.")
			fmt.Println()

			// Anthropic (optional cloud provider)
			fmt.Print("Anthropic API Key (optional): ")
			var anthropicKey string
			fmt.Scanln(&anthropicKey)
			if strings.TrimSpace(anthropicKey) != "" {
				cfg.APIKeys.Anthropic = strings.TrimSpace(anthropicKey)
			}
		}

		// Save config
		if err := configMgr.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\n  Configuration saved to: %s\n", configPath)
	}
	fmt.Println()

	// Extract bundled skills
	fmt.Println("Installing starter skills...")
	skillsDir := filepath.Join(baseDir, "skills")
	installedSkills, err := extractBundledSkills(skillsDir)
	if err != nil {
		fmt.Printf("  Warning: Failed to install bundled skills: %v\n", err)
	} else if len(installedSkills) > 0 {
		for _, skill := range installedSkills {
			fmt.Printf("  Installed: %s\n", skill)
		}
	} else {
		fmt.Println("  No new skills to install (already present)")
	}
	fmt.Println()

	// Check Docker
	if !skipDockerCheck {
		fmt.Println("Checking Docker...")
		if err := checkDocker(); err != nil {
			fmt.Printf("  Warning: Docker check failed: %v\n", err)
			fmt.Println("  Make sure Docker is installed and running.")
		} else {
			fmt.Println("  Docker is available")
		}
		fmt.Println()
	}

	fmt.Println("Initialization complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Start Ollama: ollama serve")
	fmt.Println("  2. Pull a model: ollama pull qwen2.5-coder:14b")
	fmt.Println("  3. Try a skill: sr run code-review 'func add(a, b int) { return a + b }'")
	fmt.Println()
	fmt.Println("List available skills with: sr list")
	fmt.Println()

	return nil
}

// extractBundledSkills extracts embedded skills to the skills directory
func extractBundledSkills(skillsDir string) ([]string, error) {
	skillsFS, err := embedded.GetSkillsFS()
	if err != nil {
		return nil, fmt.Errorf("get embedded skills: %w", err)
	}

	var installed []string

	err = fs.WalkDir(skillsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process YAML files
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		destPath := filepath.Join(skillsDir, path)

		// Check if skill already exists
		if _, err := os.Stat(destPath); err == nil {
			// Skill already exists, skip
			return nil
		}

		// Read embedded file
		srcFile, err := skillsFS.Open(path)
		if err != nil {
			return fmt.Errorf("open embedded file %s: %w", path, err)
		}
		defer srcFile.Close()

		content, err := io.ReadAll(srcFile)
		if err != nil {
			return fmt.Errorf("read embedded file %s: %w", path, err)
		}

		// Write to destination
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("write skill file %s: %w", destPath, err)
		}

		// Extract skill name from filename
		skillName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		installed = append(installed, skillName)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return installed, nil
}

func checkDocker() error {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
