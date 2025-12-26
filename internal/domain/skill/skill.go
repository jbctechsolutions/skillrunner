// Package skill provides the Skill aggregate root and related domain types.
package skill

import (
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// Skill is the aggregate root representing a skill definition.
// A skill consists of one or more phases that execute in order based on dependencies,
// with routing configuration to control model selection and fallback behavior.
type Skill struct {
	id          string
	name        string
	version     string
	description string
	phases      []Phase
	routing     RoutingConfig
	metadata    map[string]any
}

// NewSkill creates a new Skill with the required fields.
// Returns an error if validation fails:
//   - id is required
//   - name is required
//   - phases must have at least one element
func NewSkill(id, name, version string, phases []Phase) (*Skill, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)

	if id == "" {
		return nil, errors.ErrSkillIDRequired
	}
	if name == "" {
		return nil, errors.ErrSkillNameRequired
	}
	if len(phases) == 0 {
		return nil, errors.ErrNoPhasesDefied
	}

	// Make a copy of phases to avoid external mutation
	phasesCopy := make([]Phase, len(phases))
	copy(phasesCopy, phases)

	return &Skill{
		id:       id,
		name:     name,
		version:  version,
		phases:   phasesCopy,
		routing:  *NewRoutingConfig(),
		metadata: make(map[string]any),
	}, nil
}

// ID returns the skill's unique identifier.
func (s *Skill) ID() string {
	return s.id
}

// Name returns the skill's human-readable name.
func (s *Skill) Name() string {
	return s.name
}

// Version returns the skill's version string.
func (s *Skill) Version() string {
	return s.version
}

// Description returns the skill's description.
func (s *Skill) Description() string {
	return s.description
}

// Phases returns a copy of the skill's phases.
func (s *Skill) Phases() []Phase {
	phases := make([]Phase, len(s.phases))
	copy(phases, s.phases)
	return phases
}

// Routing returns a copy of the skill's routing configuration.
func (s *Skill) Routing() RoutingConfig {
	return s.routing
}

// Metadata returns a copy of the skill's metadata.
func (s *Skill) Metadata() map[string]any {
	meta := make(map[string]any, len(s.metadata))
	for k, v := range s.metadata {
		meta[k] = v
	}
	return meta
}

// SetDescription sets the skill's description.
func (s *Skill) SetDescription(desc string) {
	s.description = desc
}

// SetRouting sets the skill's routing configuration.
func (s *Skill) SetRouting(r RoutingConfig) {
	s.routing = r
}

// SetMetadata sets a metadata value for the skill.
func (s *Skill) SetMetadata(key string, value any) {
	s.metadata[key] = value
}

// GetPhase returns the phase with the given ID, or an error if not found.
func (s *Skill) GetPhase(id string) (*Phase, error) {
	for i := range s.phases {
		if s.phases[i].ID == id {
			return &s.phases[i], nil
		}
	}
	return nil, errors.ErrPhaseNotFound
}

// Validate checks if the Skill is in a valid state.
// It validates:
//   - All required fields are present
//   - All phases are valid
//   - Routing configuration is valid
//   - All phase dependencies exist
//   - No cycles in phase dependencies
func (s *Skill) Validate() error {
	if strings.TrimSpace(s.id) == "" {
		return errors.ErrSkillIDRequired
	}
	if strings.TrimSpace(s.name) == "" {
		return errors.ErrSkillNameRequired
	}
	if len(s.phases) == 0 {
		return errors.ErrNoPhasesDefied
	}

	// Build a map of phase IDs for dependency validation
	phaseIDs := make(map[string]bool, len(s.phases))
	for i := range s.phases {
		phaseIDs[s.phases[i].ID] = true
	}

	// Validate each phase and check dependencies
	for i := range s.phases {
		if err := s.phases[i].Validate(); err != nil {
			return err
		}

		// Check that all dependencies exist
		for _, depID := range s.phases[i].DependsOn {
			if !phaseIDs[depID] {
				return errors.ErrDependencyNotFound
			}
		}
	}

	// Check for cycles in dependencies
	if hasCycle(s.phases) {
		return errors.ErrCycleDetected
	}

	// Validate routing configuration
	if err := s.routing.Validate(); err != nil {
		return err
	}

	return nil
}

// hasCycle detects if there's a cycle in phase dependencies using DFS.
func hasCycle(phases []Phase) bool {
	// Build adjacency list
	graph := make(map[string][]string)
	for i := range phases {
		graph[phases[i].ID] = phases[i].DependsOn
	}

	// Track visited and recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(id string) bool
	dfs = func(id string) bool {
		visited[id] = true
		recStack[id] = true

		for _, dep := range graph[id] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for i := range phases {
		if !visited[phases[i].ID] {
			if dfs(phases[i].ID) {
				return true
			}
		}
	}

	return false
}
