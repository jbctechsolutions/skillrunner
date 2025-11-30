package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/jbctechsolutions/skillrunner/internal/converter"
)

// Importer handles importing skills and agents from various sources
type Importer struct {
	cacheDir     string // ~/.skillrunner/marketplace
	registryPath string // ~/.skillrunner/marketplace/registry.json
	registry     *Registry
}

// ImportedSkill represents a skill or agent that has been imported
type ImportedSkill struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"` // "skill" or "agent"
	Version     string     `json:"version"`
	Author      string     `json:"author"`
	Description string     `json:"description"`
	Source      SourceInfo `json:"source"`
	ImportedAt  time.Time  `json:"imported_at"`
	LastUpdated time.Time  `json:"last_updated"`
	LocalPath   string     `json:"local_path"`
}

// SourceInfo contains information about where a skill was imported from
type SourceInfo struct {
	Type       string    `json:"type"`                  // "web", "local", "git"
	Path       string    `json:"path,omitempty"`        // Source path/URL
	GitCommit  string    `json:"git_commit,omitempty"`  // Git commit hash
	GitRefresh time.Time `json:"git_refresh,omitempty"` // Git commit date
}

// Registry tracks all imported skills
type Registry struct {
	Version string                    `json:"version"`
	Skills  map[string]*ImportedSkill `json:"skills"`
}

// NewImporter creates a new importer
func NewImporter() (*Importer, error) {
	// Check if path is overridden in config
	var skillrunnerDir string
	cfgManager, err := config.NewManager("")
	if err == nil {
		cfg := cfgManager.Get()
		if cfg.Paths != nil && cfg.Paths.SkillrunnerDir != "" {
			skillrunnerDir = cfg.Paths.SkillrunnerDir
		}
	}

	// Default to ~/.skillrunner if not overridden
	if skillrunnerDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		skillrunnerDir = filepath.Join(home, ".skillrunner")
	}

	cacheDir := filepath.Join(skillrunnerDir, "marketplace")
	registryPath := filepath.Join(cacheDir, "registry.json")

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	importer := &Importer{
		cacheDir:     cacheDir,
		registryPath: registryPath,
	}

	// Load or create registry
	if err := importer.loadRegistry(); err != nil {
		return nil, err
	}

	return importer, nil
}

// loadRegistry loads the skill registry from disk
func (i *Importer) loadRegistry() error {
	// Check if registry exists
	if _, err := os.Stat(i.registryPath); os.IsNotExist(err) {
		// Create new registry
		i.registry = &Registry{
			Version: "1.0",
			Skills:  make(map[string]*ImportedSkill),
		}
		return i.saveRegistry()
	}

	// Load existing registry
	data, err := os.ReadFile(i.registryPath)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}

	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}

	i.registry = &registry
	return nil
}

// saveRegistry saves the skill registry to disk
func (i *Importer) saveRegistry() error {
	data, err := json.MarshalIndent(i.registry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	if err := os.WriteFile(i.registryPath, data, 0644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}

// addToRegistry adds a skill to the registry
func (i *Importer) addToRegistry(skill *ImportedSkill) error {
	i.registry.Skills[skill.ID] = skill
	return i.saveRegistry()
}

// GetSkill retrieves a skill from the registry
func (i *Importer) GetSkill(skillID string) (*ImportedSkill, error) {
	skill, ok := i.registry.Skills[skillID]
	if !ok {
		return nil, fmt.Errorf("skill not found in registry: %s", skillID)
	}
	return skill, nil
}

// ListSkills returns all imported skills and agents
func (i *Importer) ListSkills() []*ImportedSkill {
	items := make([]*ImportedSkill, 0, len(i.registry.Skills))
	for _, item := range i.registry.Skills {
		items = append(items, item)
	}
	return items
}

// RemoveSkill removes a skill from the registry and cache
func (i *Importer) RemoveSkill(skillID string) error {
	skill, ok := i.registry.Skills[skillID]
	if !ok {
		return fmt.Errorf("skill not found: %s", skillID)
	}

	// Remove from filesystem
	if err := os.RemoveAll(skill.LocalPath); err != nil {
		return fmt.Errorf("remove skill directory: %w", err)
	}

	// Remove from registry
	delete(i.registry.Skills, skillID)
	return i.saveRegistry()
}

// GetCacheDir returns the cache directory path
func (i *Importer) GetCacheDir() string {
	return i.cacheDir
}

// Import imports a skill or agent from any source (auto-detects type)
func (i *Importer) Import(source string) ([]*ImportedSkill, error) {
	sourceType := DetectSourceType(source)

	switch sourceType {
	case SourceTypeWebHTTP:
		item, err := i.ImportFromWeb(source)
		if err != nil {
			return nil, err
		}
		return []*ImportedSkill{item}, nil

	case SourceTypeLocalPath:
		item, err := i.ImportFromLocal(source)
		if err != nil {
			return nil, err
		}
		return []*ImportedSkill{item}, nil

	case SourceTypeGitHTTPS, SourceTypeGitSSH:
		return i.ImportFromGit(source)

	default:
		return nil, fmt.Errorf("unsupported source type: %s", source)
	}
}

// Update updates a skill from its original source
func (i *Importer) Update(skillID string) (*ImportedSkill, error) {
	skill, err := i.GetSkill(skillID)
	if err != nil {
		return nil, err
	}

	switch skill.Source.Type {
	case "web":
		return i.UpdateFromWeb(skillID)
	case "local":
		return i.UpdateFromLocal(skillID)
	case "git":
		return i.UpdateFromGit(skillID)
	default:
		return nil, fmt.Errorf("unknown source type: %s", skill.Source.Type)
	}
}

// convertToOrchestrated automatically converts an imported markdown skill to orchestrated format
// The skill.LocalPath is a directory containing SKILL.md or AGENT.md (and potentially other files)
func (i *Importer) convertToOrchestrated(skill *ImportedSkill) error {
	// FromMarkdown now accepts directories and will find SKILL.md/AGENT.md automatically
	// Convert markdown to orchestrated format
	orchestrated, err := converter.FromMarkdown(skill.LocalPath)
	if err != nil {
		// If conversion fails (e.g., no SKILL.md/AGENT.md found), skip conversion
		// The marketplace format is still usable without orchestrated conversion
		return nil
	}

	// Determine output directory (respects config override)
	var skillrunnerDir string
	cfgManager, err := config.NewManager("")
	if err == nil {
		cfg := cfgManager.Get()
		if cfg.Paths != nil && cfg.Paths.SkillrunnerDir != "" {
			skillrunnerDir = cfg.Paths.SkillrunnerDir
		}
	}

	// Default to ~/.skillrunner if not overridden
	if skillrunnerDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		skillrunnerDir = filepath.Join(home, ".skillrunner")
	}

	skillsDir := filepath.Join(skillrunnerDir, "skills", skill.ID)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("create skills directory: %w", err)
	}

	// Write skill.yaml
	outputPath := filepath.Join(skillsDir, "skill.yaml")
	// Use custom marshaling to format multi-line strings as literal blocks
	yamlData, err := converter.MarshalYAMLWithMultiLineStrings(orchestrated)
	if err != nil {
		return fmt.Errorf("marshal skill to YAML: %w", err)
	}

	if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
		return fmt.Errorf("write skill.yaml: %w", err)
	}

	return nil
}
