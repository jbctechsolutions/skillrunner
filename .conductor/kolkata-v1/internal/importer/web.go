package importer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MarketplaceAPI represents a web marketplace API response
type MarketplaceAPI struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Author      string            `json:"author"`
	Description string            `json:"description"`
	Files       map[string]string `json:"files"` // filename -> URL
}

// ImportFromWeb imports a skill from a web marketplace
func (i *Importer) ImportFromWeb(sourceURL string) (*ImportedSkill, error) {
	// Try to fetch as marketplace API first
	skill, err := i.fetchFromMarketplaceAPI(sourceURL)
	if err == nil {
		return skill, nil
	}

	// Fall back to direct SKILL.md or AGENT.md fetch
	return i.fetchDirectItem(sourceURL)
}

// fetchFromMarketplaceAPI fetches skill from a marketplace API endpoint
func (i *Importer) fetchFromMarketplaceAPI(apiURL string) (*ImportedSkill, error) {
	// Add .json if not present
	if !strings.HasSuffix(apiURL, ".json") && !strings.Contains(apiURL, "/api/") {
		apiURL = apiURL + ".json"
	}

	// Fetch API response
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetch marketplace API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace API returned %d", resp.StatusCode)
	}

	// Parse JSON response
	var apiData MarketplaceAPI
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, fmt.Errorf("parse marketplace API response: %w", err)
	}

	// Download skill files
	skillID := apiData.ID
	if skillID == "" {
		skillID = strings.ToLower(strings.ReplaceAll(apiData.Name, " ", "-"))
	}

	destDir := filepath.Join(i.cacheDir, skillID)

	// Check if already exists
	if existing, err := i.GetSkill(skillID); err == nil {
		return nil, fmt.Errorf("skill '%s' already imported from %s (use update to refresh)",
			skillID, existing.Source.Path)
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create skill directory: %w", err)
	}

	// Download all files
	for filename, fileURL := range apiData.Files {
		if err := i.downloadFile(fileURL, filepath.Join(destDir, filename)); err != nil {
			os.RemoveAll(destDir) // Cleanup on failure
			return nil, fmt.Errorf("download file %s: %w", filename, err)
		}
	}

	// Create imported skill record
	now := time.Now()
	skill := &ImportedSkill{
		ID:          skillID,
		Name:        apiData.Name,
		Type:        "skill", // Marketplace items are typically skills
		Version:     apiData.Version,
		Author:      apiData.Author,
		Description: apiData.Description,
		Source: SourceInfo{
			Type: "web",
			Path: apiURL,
		},
		ImportedAt:  now,
		LastUpdated: now,
		LocalPath:   destDir,
	}

	// Add to registry
	if err := i.addToRegistry(skill); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("add to registry: %w", err)
	}

	// Auto-convert to orchestrated format
	if err := i.convertToOrchestrated(skill); err != nil {
		// Log warning but don't fail import - marketplace format still usable
		fmt.Fprintf(os.Stderr, "Warning: Failed to auto-convert %s to orchestrated format: %v\n", skill.ID, err)
	}

	return skill, nil
}

// fetchDirectItem fetches a skill or agent from a direct SKILL.md or AGENT.md URL
func (i *Importer) fetchDirectItem(itemURL string) (*ImportedSkill, error) {
	// Fetch file
	resp, err := http.Get(itemURL)
	if err != nil {
		return nil, fmt.Errorf("fetch file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch returned %d", resp.StatusCode)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Determine type from URL
	itemType := "skill"
	filename := filepath.Base(itemURL)
	if filename == "AGENT.md" {
		itemType = "agent"
	}

	// Parse metadata from frontmatter
	metadata, err := parseSkillMetadataFromContent(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}

	// Generate item ID
	itemID := metadata.Name
	if itemID == "" {
		// Extract from URL
		parts := strings.Split(itemURL, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" && parts[i] != "SKILL.md" && parts[i] != "AGENT.md" {
				itemID = parts[i]
				break
			}
		}
	}
	itemID = strings.ToLower(strings.ReplaceAll(itemID, " ", "-"))

	// Check if already exists
	if existing, err := i.GetSkill(itemID); err == nil {
		return nil, fmt.Errorf("%s '%s' already imported from %s (use update to refresh)",
			existing.Type, itemID, existing.Source.Path)
	}

	// Create destination directory
	destDir := filepath.Join(i.cacheDir, itemID)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	// Write file
	filePath := filepath.Join(destDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		os.RemoveAll(destDir)
		return nil, fmt.Errorf("write file: %w", err)
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
			Type: "web",
			Path: itemURL,
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

// UpdateFromWeb updates a skill or agent from its web source
func (i *Importer) UpdateFromWeb(skillID string) (*ImportedSkill, error) {
	// Get existing item
	item, err := i.GetSkill(skillID)
	if err != nil {
		return nil, err
	}

	if item.Source.Type != "web" {
		return nil, fmt.Errorf("%s '%s' is not from a web source", item.Type, skillID)
	}

	// Remove old version
	if err := os.RemoveAll(item.LocalPath); err != nil {
		return nil, fmt.Errorf("remove old version: %w", err)
	}

	// Re-import
	delete(i.registry.Skills, skillID)

	newItem, err := i.ImportFromWeb(item.Source.Path)
	if err != nil {
		// Try to restore old version info
		i.registry.Skills[skillID] = item
		return nil, fmt.Errorf("update failed: %w", err)
	}

	return newItem, nil
}

// downloadFile downloads a file from a URL to a local path
func (i *Importer) downloadFile(url, destPath string) error {
	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer destFile.Close()

	// Fetch URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch returned %d", resp.StatusCode)
	}

	// Copy content
	if _, err := io.Copy(destFile, resp.Body); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	return nil
}

// parseSkillMetadataFromContent parses metadata from SKILL.md content
func parseSkillMetadataFromContent(content string) (*SkillMetadata, error) {
	// Check for YAML frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return &SkillMetadata{}, nil // No frontmatter
	}

	// Find end of frontmatter
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return nil, fmt.Errorf("unterminated frontmatter")
	}

	// Extract frontmatter
	frontmatterText := content[4 : endIdx+4]

	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatterText), &metadata); err != nil {
		return nil, fmt.Errorf("parse YAML frontmatter: %w", err)
	}

	return &metadata, nil
}
