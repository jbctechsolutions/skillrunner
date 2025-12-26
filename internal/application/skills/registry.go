// Package skills provides application-level skill management.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	infraSkills "github.com/jbctechsolutions/skillrunner/internal/infrastructure/skills"
)

// Registry provides caching and lookup for skill definitions.
// It manages skills from both built-in and user directories.
type Registry struct {
	loader *infraSkills.Loader
	skills map[string]*skill.Skill
	mu     sync.RWMutex
	loaded bool

	// Directory paths
	builtInDir string
	userDir    string
}

// NewRegistry creates a new SkillRegistry with the given loader.
func NewRegistry(loader *infraSkills.Loader) *Registry {
	return &Registry{
		loader: loader,
		skills: make(map[string]*skill.Skill),
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
}

// Clear removes all skills from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills = make(map[string]*skill.Skill)
	r.loaded = false
}
