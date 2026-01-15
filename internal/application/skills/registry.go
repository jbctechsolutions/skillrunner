// Package skills provides application-level skill management.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	infraSkills "github.com/jbctechsolutions/skillrunner/internal/infrastructure/skills"
)

// Registry provides caching and lookup for skill definitions.
// It manages skills from built-in, user, and project directories.
type Registry struct {
	loader *infraSkills.Loader
	skills map[string]*skill.Skill
	mu     sync.RWMutex
	loaded bool

	// Source tracking for hot reload support
	sourceMap map[string]*skill.SkillSource // skillID -> source info
	pathMap   map[string]string             // filePath -> skillID (for deletion handling)

	// Directory paths
	builtInDir string
	userDir    string
	projectDir string
}

// NewRegistry creates a new SkillRegistry with the given loader.
func NewRegistry(loader *infraSkills.Loader) *Registry {
	return &Registry{
		loader:    loader,
		skills:    make(map[string]*skill.Skill),
		sourceMap: make(map[string]*skill.SkillSource),
		pathMap:   make(map[string]string),
	}
}

// SetBuiltInDir sets the path to the built-in skills directory.
func (r *Registry) SetBuiltInDir(dir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builtInDir = dir
}

// SetUserDir sets the path to the user skills directory.
func (r *Registry) SetUserDir(dir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userDir = dir
}

// SetProjectDir sets the path to the project skills directory.
func (r *Registry) SetProjectDir(dir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projectDir = dir
}

// ProjectDir returns the project skills directory path.
func (r *Registry) ProjectDir() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.projectDir
}

// UserDir returns the user skills directory path.
func (r *Registry) UserDir() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.userDir != "" {
		return r.userDir
	}
	// Default: ~/.skillrunner/skills
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".skillrunner", "skills")
}

// BuiltInDir returns the built-in skills directory path.
func (r *Registry) BuiltInDir() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.builtInDir != "" {
		return r.builtInDir
	}
	return "skills"
}

// LoadBuiltInSkills loads skills from the built-in skills directory.
// These are skills bundled with the application.
func (r *Registry) LoadBuiltInSkills() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.builtInDir == "" {
		// Default: use ./skills relative to executable or current working dir
		r.builtInDir = "skills"
	}

	// Check if directory exists
	if _, err := os.Stat(r.builtInDir); os.IsNotExist(err) {
		// Not an error - built-in skills are optional
		return nil
	}

	skills, err := r.loader.LoadSkillsDir(r.builtInDir)
	if err != nil {
		return fmt.Errorf("failed to load built-in skills: %w", err)
	}

	// Add to cache
	for id, s := range skills {
		r.skills[id] = s
	}

	return nil
}

// LoadUserSkills loads skills from the user's custom skills directory.
// Default location is ~/.skillrunner/skills if dir is empty.
func (r *Registry) LoadUserSkills(dir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if dir == "" {
		if r.userDir != "" {
			dir = r.userDir
		} else {
			// Default: ~/.skillrunner/skills
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			dir = filepath.Join(homeDir, ".skillrunner", "skills")
		}
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Not an error - user skills are optional
		return nil
	}

	skills, err := r.loader.LoadSkillsDir(dir)
	if err != nil {
		return fmt.Errorf("failed to load user skills: %w", err)
	}

	// Add to cache (user skills override built-in skills with same ID)
	for id, s := range skills {
		r.skills[id] = s
	}

	return nil
}

// LoadAll loads skills from both built-in and user directories.
// User skills take precedence over built-in skills with the same ID.
func (r *Registry) LoadAll() error {
	// Clear existing cache
	r.mu.Lock()
	r.skills = make(map[string]*skill.Skill)
	r.sourceMap = make(map[string]*skill.SkillSource)
	r.pathMap = make(map[string]string)
	r.loaded = false
	r.mu.Unlock()

	// Load built-in skills first
	if err := r.LoadBuiltInSkills(); err != nil {
		return err
	}

	// Load user skills (they can override built-in)
	if err := r.LoadUserSkills(""); err != nil {
		return err
	}

	r.mu.Lock()
	r.loaded = true
	r.mu.Unlock()

	return nil
}

// GetSkill retrieves a skill by its ID.
// Returns nil if the skill is not found.
func (r *Registry) GetSkill(id string) *skill.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.skills[id]
}

// GetSkillByName retrieves a skill by its name (case-sensitive).
// Returns nil if no skill matches the name.
func (r *Registry) GetSkillByName(name string) *skill.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.skills {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

// ListSkills returns all loaded skills.
// The returned slice is a copy to prevent external modification.
func (r *Registry) ListSkills() []*skill.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*skill.Skill, 0, len(r.skills))
	for _, s := range r.skills {
		skills = append(skills, s)
	}
	return skills
}

// Count returns the number of loaded skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// IsLoaded returns true if skills have been loaded.
func (r *Registry) IsLoaded() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loaded
}

// Reload clears the cache and reloads all skills.
func (r *Registry) Reload() error {
	return r.LoadAll()
}

// Register manually registers a skill in the registry.
// This is useful for testing or programmatically created skills.
func (r *Registry) Register(s *skill.Skill) error {
	if s == nil {
		return fmt.Errorf("cannot register nil skill")
	}
	if s.ID() == "" {
		return fmt.Errorf("skill must have an ID")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[s.ID()] = s
	return nil
}

// Unregister removes a skill from the registry.
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.skills, id)
	// Also clean up source tracking
	if source, ok := r.sourceMap[id]; ok {
		delete(r.pathMap, source.FilePath())
		delete(r.sourceMap, id)
	}
}

// Clear removes all skills from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills = make(map[string]*skill.Skill)
	r.sourceMap = make(map[string]*skill.SkillSource)
	r.pathMap = make(map[string]string)
	r.loaded = false
}

// RegisterWithSource registers a skill with its source information.
// This enables proper handling of skill overrides and deletions.
func (r *Registry) RegisterWithSource(s *skill.Skill, filePath string, sourceType skill.SourceType) error {
	if s == nil {
		return fmt.Errorf("cannot register nil skill")
	}
	if s.ID() == "" {
		return fmt.Errorf("skill must have an ID")
	}
	if filePath == "" {
		return fmt.Errorf("file path is required for source tracking")
	}

	source, err := skill.NewSkillSource(s.ID(), filePath, sourceType, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create skill source: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we should register based on priority
	if existing, ok := r.sourceMap[s.ID()]; ok {
		// Only replace if new source has higher or equal priority
		if sourceType.Priority() < existing.Source().Priority() {
			return nil // Don't replace higher-priority skill
		}
		// Remove old path mapping
		delete(r.pathMap, existing.FilePath())
	}

	r.skills[s.ID()] = s
	r.sourceMap[s.ID()] = source
	r.pathMap[filePath] = s.ID()
	return nil
}

// GetSource returns the source information for a skill.
// Returns nil if the skill is not found or has no source tracking.
func (r *Registry) GetSource(skillID string) *skill.SkillSource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sourceMap[skillID]
}

// UnregisterByPath removes a skill by its file path.
// Returns the skill ID that was removed, and true if a skill was found.
func (r *Registry) UnregisterByPath(filePath string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	skillID, ok := r.pathMap[filePath]
	if !ok {
		return "", false
	}

	// Remove the skill and its source tracking
	delete(r.skills, skillID)
	delete(r.sourceMap, skillID)
	delete(r.pathMap, filePath)

	return skillID, true
}

// GetSkillIDByPath returns the skill ID for a given file path.
// Returns empty string if no skill is registered from that path.
func (r *Registry) GetSkillIDByPath(filePath string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.pathMap[filePath]
}

// HasSourceTracking returns true if the registry has source tracking data.
func (r *Registry) HasSourceTracking() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sourceMap) > 0
}

// Loader returns the skill loader used by the registry.
func (r *Registry) Loader() *infraSkills.Loader {
	return r.loader
}
