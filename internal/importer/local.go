package importer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ImportFromLocal imports a skill or agent from a local filesystem path
func (i *Importer) ImportFromLocal(sourcePath string) (*ImportedSkill, error) {
	// Normalize path
	sourcePath = NormalizeSource(sourcePath)

	// Verify source exists
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("source path not found: %w", err)
	}

	// If it's a file, get the directory
	itemDir := sourcePath
	var itemType string
	var metadataPath string

	if !sourceInfo.IsDir() {
		// Check if it's a SKILL.md or AGENT.md file
		baseName := filepath.Base(sourcePath)
		if baseName == "SKILL.md" {
			itemDir = filepath.Dir(sourcePath)
			itemType = "skill"
			metadataPath = sourcePath
		} else if baseName == "AGENT.md" {
			itemDir = filepath.Dir(sourcePath)
			itemType = "agent"
			metadataPath = sourcePath
		} else {
			return nil, fmt.Errorf("source must be a directory, SKILL.md, or AGENT.md file")
		}
	} else {
		// Check for SKILL.md or AGENT.md in directory
		skillPath := filepath.Join(itemDir, "SKILL.md")
		agentPath := filepath.Join(itemDir, "AGENT.md")

		hasSkill, _ := os.Stat(skillPath)
		hasAgent, _ := os.Stat(agentPath)

		if hasSkill != nil && hasAgent != nil {
			return nil, fmt.Errorf("directory contains both SKILL.md and AGENT.md - only one is allowed")
		} else if hasSkill != nil {
			itemType = "skill"
			metadataPath = skillPath
		} else if hasAgent != nil {
			itemType = "agent"
			metadataPath = agentPath
		} else {
			return nil, fmt.Errorf("neither SKILL.md nor AGENT.md found in %s", itemDir)
		}
	}

	// Parse metadata
	metadata, err := parseSkillMetadata(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}

	// Generate item ID from directory name or metadata
	itemID := metadata.Name
	if itemID == "" {
		itemID = filepath.Base(itemDir)
	}
	// Normalize ID (lowercase, replace spaces with dashes)
	itemID = strings.ToLower(strings.ReplaceAll(itemID, " ", "-"))

	// Create destination directory
	destDir := filepath.Join(i.cacheDir, itemID)

	// Check if item already exists
	if existing, err := i.GetSkill(itemID); err == nil {
		return nil, fmt.Errorf("%s '%s' already imported from %s (use update to refresh)",
			existing.Type, itemID, existing.Source.Path)
	}

	// Copy directory to cache
	if err := copyDir(itemDir, destDir); err != nil {
		return nil, fmt.Errorf("copy directory: %w", err)
	}

	// Create imported item record
	now := time.Now()
	item := &ImportedSkill{
		ID:          itemID,
		Name:        metadata.Name,
		Type:        itemType,
		Version:     metadata.Version,
		Author:      metadata.Author,
		Description: metadata.Description,
		Source: SourceInfo{
			Type: "local",
			Path: sourcePath,
		},
		ImportedAt:  now,
		LastUpdated: now,
		LocalPath:   destDir,
	}

	// Add to registry
	if err := i.addToRegistry(item); err != nil {
		// Cleanup on failure
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("add to registry: %w", err)
	}

	// Auto-convert to orchestrated format
	if err := i.convertToOrchestrated(item); err != nil {
		// Log warning but don't fail import - marketplace format still usable
		fmt.Fprintf(os.Stderr, "Warning: Failed to auto-convert %s to orchestrated format: %v\n", item.ID, err)
	}

	return item, nil
}

// UpdateFromLocal updates a skill or agent from its local source
func (i *Importer) UpdateFromLocal(skillID string) (*ImportedSkill, error) {
	// Get existing item
	item, err := i.GetSkill(skillID)
	if err != nil {
		return nil, err
	}

	if item.Source.Type != "local" {
		return nil, fmt.Errorf("%s '%s' is not from a local source", item.Type, skillID)
	}

	// Remove old version
	if err := os.RemoveAll(item.LocalPath); err != nil {
		return nil, fmt.Errorf("remove old version: %w", err)
	}

	// Re-import
	delete(i.registry.Skills, skillID) // Remove from registry temporarily

	newItem, err := i.ImportFromLocal(item.Source.Path)
	if err != nil {
		// Try to restore old version info
		i.registry.Skills[skillID] = item
		return nil, fmt.Errorf("update failed: %w", err)
	}

	return newItem, nil
}

// SkillMetadata represents metadata from SKILL.md frontmatter
type SkillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Keywords    string `yaml:"keywords"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
	LastUpdated string `yaml:"last_updated"`
}

// parseSkillMetadata extracts metadata from SKILL.md frontmatter
func parseSkillMetadata(path string) (*SkillMetadata, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Check for YAML frontmatter
	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "---\n") {
		return &SkillMetadata{}, nil // No frontmatter, return empty metadata
	}

	// Find end of frontmatter
	endIdx := strings.Index(contentStr[4:], "\n---\n")
	if endIdx == -1 {
		return nil, fmt.Errorf("unterminated frontmatter")
	}

	// Extract and parse frontmatter
	frontmatterText := contentStr[4 : endIdx+4]

	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatterText), &metadata); err != nil {
		return nil, fmt.Errorf("parse YAML frontmatter: %w", err)
	}

	return &metadata, nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}
