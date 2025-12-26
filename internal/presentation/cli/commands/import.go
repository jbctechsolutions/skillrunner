// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// importFlags holds the flags for the import command.
type importFlags struct {
	Name  string
	Force bool
}

var importOpts importFlags

// ImportResult represents the result of an import operation.
type ImportResult struct {
	Source      string   `json:"source"`
	SourceType  string   `json:"source_type"`
	Destination string   `json:"destination"`
	SkillName   string   `json:"skill_name"`
	Skills      []string `json:"skills,omitempty"`
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
}

// NewImportCmd creates the import command for importing skills from various sources.
func NewImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <source>",
		Short: "Import skills from web/git sources",
		Long: `Import skill definitions from URLs, git repositories, or local paths.

The import command supports multiple source types:
  • URL to a YAML file (e.g., https://example.com/skill.yaml)
  • Git repository URL (clones and finds skills in skills/ directory)
  • Local file path (copies to skillrunner skills directory)

Imported skills are saved to ~/.skillrunner/skills/ by default.

Examples:
  # Import a skill from a URL
  sr import https://example.com/skills/code-review.yaml

  # Import from a git repository
  sr import https://github.com/user/skillrunner-skills.git

  # Import a local skill file
  sr import ./my-skill.yaml

  # Import with a custom name
  sr import https://example.com/review.yaml --name my-review

  # Force overwrite existing skill
  sr import ./updated-skill.yaml --force`,
		Args: cobra.ExactArgs(1),
		RunE: runImport,
	}

	// Define flags
	cmd.Flags().StringVarP(&importOpts.Name, "name", "n", "", "rename the skill on import")
	cmd.Flags().BoolVarP(&importOpts.Force, "force", "f", false, "overwrite existing skill if it exists")

	return cmd
}

// runImport handles the import command execution.
func runImport(cmd *cobra.Command, args []string) error {
	source := args[0]
	formatter := GetFormatter()

	// Determine source type
	sourceType := detectSourceType(source)

	var result ImportResult
	var err error

	switch sourceType {
	case "url":
		result, err = importFromURL(source)
	case "git":
		result, err = importFromGit(source)
	case "local":
		result, err = importFromLocal(source)
	default:
		return fmt.Errorf("unknown source type for: %s", source)
	}

	if err != nil {
		return err
	}

	// Output based on format
	if formatter.Format() == "json" {
		return formatter.JSON(result)
	}

	// Text output
	if result.Success {
		formatter.Success("Skill imported successfully")
		formatter.Item("Source", result.Source)
		formatter.Item("Type", result.SourceType)
		formatter.Item("Skill Name", result.SkillName)
		formatter.Item("Destination", result.Destination)
		if len(result.Skills) > 0 {
			formatter.Println("")
			formatter.Info("Imported %d skill(s):", len(result.Skills))
			for _, s := range result.Skills {
				formatter.BulletItem(s)
			}
		}
	} else {
		formatter.Error("Import failed: %s", result.Message)
	}

	return nil
}

// detectSourceType determines the type of source (url, git, local).
func detectSourceType(source string) string {
	// Check if it's a URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Check if it's a git repo URL
		if strings.HasSuffix(source, ".git") ||
			strings.Contains(source, "github.com") ||
			strings.Contains(source, "gitlab.com") ||
			strings.Contains(source, "bitbucket.org") {
			// If it ends with .yaml or .yml, it's a URL
			if strings.HasSuffix(source, ".yaml") || strings.HasSuffix(source, ".yml") {
				return "url"
			}
			return "git"
		}
		return "url"
	}

	// Otherwise, treat as local path
	return "local"
}

// getSkillsDir returns the skillrunner skills directory path.
func getSkillsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".skillrunner", "skills"), nil
}

// ensureSkillsDir creates the skills directory if it doesn't exist.
func ensureSkillsDir() (string, error) {
	skillsDir, err := getSkillsDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return "", fmt.Errorf("could not create skills directory: %w", err)
	}

	return skillsDir, nil
}

// importFromURL imports a skill from a URL.
func importFromURL(source string) (ImportResult, error) {
	result := ImportResult{
		Source:     source,
		SourceType: "url",
	}

	// Validate URL
	parsedURL, err := url.Parse(source)
	if err != nil {
		result.Message = fmt.Sprintf("invalid URL: %v", err)
		return result, fmt.Errorf("invalid URL: %w", err)
	}

	// Ensure skills directory exists
	skillsDir, err := ensureSkillsDir()
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	// Determine filename
	filename := filepath.Base(parsedURL.Path)
	if importOpts.Name != "" {
		// Use custom name, preserving extension
		ext := filepath.Ext(filename)
		if ext == "" {
			ext = ".yaml"
		}
		filename = importOpts.Name + ext
	}

	// Check if file exists
	destPath := filepath.Join(skillsDir, filename)
	if _, err := os.Stat(destPath); err == nil && !importOpts.Force {
		result.Message = fmt.Sprintf("skill already exists: %s (use --force to overwrite)", filename)
		return result, fmt.Errorf("skill already exists: %s (use --force to overwrite)", filename)
	}

	// Download the file
	resp, err := http.Get(source)
	if err != nil {
		result.Message = fmt.Sprintf("failed to download: %v", err)
		return result, fmt.Errorf("failed to download skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Message = fmt.Sprintf("failed to download: HTTP %d", resp.StatusCode)
		return result, fmt.Errorf("failed to download skill: HTTP %d", resp.StatusCode)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Message = fmt.Sprintf("failed to read response: %v", err)
		return result, fmt.Errorf("failed to read response: %w", err)
	}

	// Write to file
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		result.Message = fmt.Sprintf("failed to write file: %v", err)
		return result, fmt.Errorf("failed to write skill file: %w", err)
	}

	// Extract skill name from filename
	skillName := strings.TrimSuffix(filename, filepath.Ext(filename))

	result.Success = true
	result.SkillName = skillName
	result.Destination = destPath
	result.Message = "Skill imported successfully"

	return result, nil
}

// importFromGit imports skills from a git repository.
func importFromGit(source string) (ImportResult, error) {
	result := ImportResult{
		Source:     source,
		SourceType: "git",
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		result.Message = "git is not installed or not in PATH"
		return result, fmt.Errorf("git is not installed or not in PATH")
	}

	// Create temp directory for cloning
	tempDir, err := os.MkdirTemp("", "skillrunner-import-*")
	if err != nil {
		result.Message = fmt.Sprintf("failed to create temp directory: %v", err)
		return result, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth", "1", source, tempDir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		result.Message = fmt.Sprintf("failed to clone repository: %v", err)
		return result, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Look for skills in skills/ directory
	skillsSourceDir := filepath.Join(tempDir, "skills")
	if _, err := os.Stat(skillsSourceDir); os.IsNotExist(err) {
		// Try root directory
		skillsSourceDir = tempDir
	}

	// Find all YAML files
	var skillFiles []string
	err = filepath.Walk(skillsSourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			skillFiles = append(skillFiles, path)
		}
		return nil
	})
	if err != nil {
		result.Message = fmt.Sprintf("failed to scan for skills: %v", err)
		return result, fmt.Errorf("failed to scan for skills: %w", err)
	}

	if len(skillFiles) == 0 {
		result.Message = "no skill files found in repository"
		return result, fmt.Errorf("no skill files found in repository")
	}

	// Ensure skills directory exists
	destDir, err := ensureSkillsDir()
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	// Copy each skill file
	var importedSkills []string
	for _, skillFile := range skillFiles {
		filename := filepath.Base(skillFile)
		destPath := filepath.Join(destDir, filename)

		// Check if file exists
		if _, err := os.Stat(destPath); err == nil && !importOpts.Force {
			// Skip existing files unless force is set
			continue
		}

		// Read and copy
		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			continue
		}

		skillName := strings.TrimSuffix(filename, filepath.Ext(filename))
		importedSkills = append(importedSkills, skillName)
	}

	if len(importedSkills) == 0 {
		result.Message = "no new skills imported (all skills already exist, use --force to overwrite)"
		return result, fmt.Errorf("no new skills imported (all skills already exist, use --force to overwrite)")
	}

	result.Success = true
	result.Skills = importedSkills
	result.SkillName = strings.Join(importedSkills, ", ")
	result.Destination = destDir
	result.Message = fmt.Sprintf("Imported %d skill(s)", len(importedSkills))

	return result, nil
}

// importFromLocal imports a skill from a local file or directory.
func importFromLocal(source string) (ImportResult, error) {
	result := ImportResult{
		Source:     source,
		SourceType: "local",
	}

	// Check if source exists
	info, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			result.Message = fmt.Sprintf("source not found: %s", source)
			return result, fmt.Errorf("source not found: %s", source)
		}
		result.Message = fmt.Sprintf("failed to access source: %v", err)
		return result, fmt.Errorf("failed to access source: %w", err)
	}

	// Ensure skills directory exists
	destDir, err := ensureSkillsDir()
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	if info.IsDir() {
		// Import all YAML files from directory
		return importFromLocalDir(source, destDir, result)
	}

	// Import single file
	return importFromLocalFile(source, destDir, result)
}

// importFromLocalFile imports a single local file.
func importFromLocalFile(source, destDir string, result ImportResult) (ImportResult, error) {
	// Validate file extension
	ext := filepath.Ext(source)
	if ext != ".yaml" && ext != ".yml" {
		result.Message = "source must be a YAML file (.yaml or .yml)"
		return result, fmt.Errorf("source must be a YAML file (.yaml or .yml)")
	}

	// Determine filename
	filename := filepath.Base(source)
	if importOpts.Name != "" {
		filename = importOpts.Name + ext
	}

	// Check if file exists
	destPath := filepath.Join(destDir, filename)
	if _, err := os.Stat(destPath); err == nil && !importOpts.Force {
		result.Message = fmt.Sprintf("skill already exists: %s (use --force to overwrite)", filename)
		return result, fmt.Errorf("skill already exists: %s (use --force to overwrite)", filename)
	}

	// Read source file
	content, err := os.ReadFile(source)
	if err != nil {
		result.Message = fmt.Sprintf("failed to read source file: %v", err)
		return result, fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		result.Message = fmt.Sprintf("failed to write skill file: %v", err)
		return result, fmt.Errorf("failed to write skill file: %w", err)
	}

	skillName := strings.TrimSuffix(filename, ext)

	result.Success = true
	result.SkillName = skillName
	result.Destination = destPath
	result.Message = "Skill imported successfully"

	return result, nil
}

// importFromLocalDir imports all YAML files from a local directory.
func importFromLocalDir(source, destDir string, result ImportResult) (ImportResult, error) {
	// Find all YAML files
	var skillFiles []string
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			skillFiles = append(skillFiles, path)
		}
		return nil
	})
	if err != nil {
		result.Message = fmt.Sprintf("failed to scan directory: %v", err)
		return result, fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(skillFiles) == 0 {
		result.Message = "no skill files found in directory"
		return result, fmt.Errorf("no skill files found in directory")
	}

	// Copy each skill file
	var importedSkills []string
	for _, skillFile := range skillFiles {
		filename := filepath.Base(skillFile)
		destPath := filepath.Join(destDir, filename)

		// Check if file exists
		if _, err := os.Stat(destPath); err == nil && !importOpts.Force {
			continue
		}

		// Read and copy
		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			continue
		}

		skillName := strings.TrimSuffix(filename, filepath.Ext(filename))
		importedSkills = append(importedSkills, skillName)
	}

	if len(importedSkills) == 0 {
		result.Message = "no new skills imported (all skills already exist, use --force to overwrite)"
		return result, fmt.Errorf("no new skills imported (all skills already exist, use --force to overwrite)")
	}

	result.Success = true
	result.Skills = importedSkills
	result.SkillName = strings.Join(importedSkills, ", ")
	result.Destination = destDir
	result.Message = fmt.Sprintf("Imported %d skill(s)", len(importedSkills))

	return result, nil
}
