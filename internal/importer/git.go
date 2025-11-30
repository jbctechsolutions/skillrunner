package importer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ImportFromGit imports skill(s) and/or agent(s) from a git repository
func (i *Importer) ImportFromGit(repoURL string) ([]*ImportedSkill, error) {
	// Create temporary clone directory
	tmpDir, err := os.MkdirTemp("", "skillrunner-git-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up temp clone

	// Clone repository
	if err := i.cloneRepo(repoURL, tmpDir); err != nil {
		return nil, fmt.Errorf("clone repository: %w", err)
	}

	// Find all SKILL.md and AGENT.md files in the repository
	itemPaths, err := i.findItemFiles(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("find item files: %w", err)
	}

	if len(itemPaths) == 0 {
		return nil, fmt.Errorf("no SKILL.md or AGENT.md files found in repository")
	}

	// Import each item found
	var items []*ImportedSkill
	for _, itemPath := range itemPaths {
		item, err := i.importItemFromGitClone(repoURL, tmpDir, itemPath)
		if err != nil {
			// Log error but continue with other items
			fmt.Fprintf(os.Stderr, "Warning: failed to import %s: %v\n", itemPath, err)
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no items successfully imported")
	}

	return items, nil
}

// cloneRepo clones a git repository
func (i *Importer) cloneRepo(repoURL, destDir string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// findItemFiles finds all SKILL.md and AGENT.md files in a directory recursively
func (i *Importer) findItemFiles(rootDir string) ([]string, error) {
	var itemPaths []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Check for SKILL.md or AGENT.md files
		if !info.IsDir() && (info.Name() == "SKILL.md" || info.Name() == "AGENT.md") {
			itemPaths = append(itemPaths, path)
		}

		return nil
	})

	return itemPaths, err
}

// importItemFromGitClone imports a single skill or agent from a cloned git repo
func (i *Importer) importItemFromGitClone(repoURL, cloneDir, itemPath string) (*ImportedSkill, error) {
	// Get item directory
	itemDir := filepath.Dir(itemPath)

	// Determine type based on filename
	itemType := "skill"
	if filepath.Base(itemPath) == "AGENT.md" {
		itemType = "agent"
	}

	// Parse metadata
	metadata, err := parseSkillMetadata(itemPath)
	if err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}

	// Generate item ID
	itemID := metadata.Name
	if itemID == "" {
		itemID = filepath.Base(itemDir)
	}
	// Add repo name prefix if multiple items
	repoName := extractRepoName(repoURL)
	if repoName != "" && itemID != repoName {
		itemID = fmt.Sprintf("%s-%s", repoName, itemID)
	}
	itemID = strings.ToLower(strings.ReplaceAll(itemID, " ", "-"))

	// Check if already exists
	if existing, err := i.GetSkill(itemID); err == nil {
		return nil, fmt.Errorf("%s '%s' already imported from %s (use update to refresh)",
			existing.Type, itemID, existing.Source.Path)
	}

	// Copy directory to cache
	destDir := filepath.Join(i.cacheDir, itemID)
	if err := copyDir(itemDir, destDir); err != nil {
		return nil, fmt.Errorf("copy directory: %w", err)
	}

	// Get git commit info
	commitHash, commitDate := i.getGitCommitInfo(cloneDir)

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
			Type:       "git",
			Path:       repoURL,
			GitCommit:  commitHash,
			GitRefresh: commitDate,
		},
		ImportedAt:  now,
		LastUpdated: now,
		LocalPath:   destDir,
	}

	// Add to registry
	if err := i.addToRegistry(item); err != nil {
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

// UpdateFromGit updates a skill or agent from its git source
func (i *Importer) UpdateFromGit(skillID string) (*ImportedSkill, error) {
	// Get existing item
	item, err := i.GetSkill(skillID)
	if err != nil {
		return nil, err
	}

	if item.Source.Type != "git" {
		return nil, fmt.Errorf("%s '%s' is not from a git source", item.Type, skillID)
	}

	// Remove old version
	if err := os.RemoveAll(item.LocalPath); err != nil {
		return nil, fmt.Errorf("remove old version: %w", err)
	}

	// Re-import
	delete(i.registry.Skills, skillID)

	// Import from git (may return multiple items)
	items, err := i.ImportFromGit(item.Source.Path)
	if err != nil {
		// Try to restore old version info
		i.registry.Skills[skillID] = item
		return nil, fmt.Errorf("update failed: %w", err)
	}

	// Find the item with matching ID
	for _, newItem := range items {
		if newItem.ID == skillID {
			return newItem, nil
		}
	}

	// If we got here, the item ID changed or wasn't found
	return nil, fmt.Errorf("%s '%s' not found in updated repository", item.Type, skillID)
}

// getGitCommitInfo gets the current commit hash and date
func (i *Importer) getGitCommitInfo(repoDir string) (hash string, date time.Time) {
	// Get commit hash
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
	if output, err := cmd.Output(); err == nil {
		hash = strings.TrimSpace(string(output))
	}

	// Get commit date
	cmd = exec.Command("git", "-C", repoDir, "log", "-1", "--format=%cI")
	if output, err := cmd.Output(); err == nil {
		dateStr := strings.TrimSpace(string(output))
		if parsedDate, err := time.Parse(time.RFC3339, dateStr); err == nil {
			date = parsedDate
		}
	}

	return hash, date
}

// extractRepoName extracts the repository name from a git URL
func extractRepoName(repoURL string) string {
	// Remove .git suffix
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Extract last component
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Handle git@ URLs
		if colonIdx := strings.LastIndex(name, ":"); colonIdx != -1 {
			name = name[colonIdx+1:]
		}
		return name
	}

	return ""
}
