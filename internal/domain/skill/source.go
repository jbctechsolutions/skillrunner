package skill

import (
	"errors"
	"strings"
	"time"
)

// SourceType represents where a skill was loaded from.
type SourceType string

// SourceType constants for skill origins.
const (
	// SourceBuiltIn indicates a skill bundled with the application.
	SourceBuiltIn SourceType = "built-in"
	// SourceUser indicates a skill from the user's ~/.skillrunner/skills/ directory.
	SourceUser SourceType = "user"
	// SourceProject indicates a skill from the project's .skillrunner/skills/ directory.
	SourceProject SourceType = "project"
)

// IsValid returns true if the source type is a recognized value.
func (s SourceType) IsValid() bool {
	switch s {
	case SourceBuiltIn, SourceUser, SourceProject:
		return true
	default:
		return false
	}
}

// Priority returns the priority level for this source type.
// Higher values indicate higher priority.
// Project > User > BuiltIn (3 > 2 > 1).
func (s SourceType) Priority() int {
	switch s {
	case SourceProject:
		return 3
	case SourceUser:
		return 2
	case SourceBuiltIn:
		return 1
	default:
		return 0
	}
}

// Sentinel errors for SkillSource validation.
var (
	ErrSourceSkillIDRequired  = errors.New("skill source: skill ID is required")
	ErrSourceFilePathRequired = errors.New("skill source: file path is required")
	ErrSourceTypeInvalid      = errors.New("skill source: invalid source type")
	ErrSourceLoadedAtRequired = errors.New("skill source: loaded at time is required")
)

// SkillSource tracks the origin of a loaded skill.
// This enables proper handling of skill overrides and deletions.
type SkillSource struct {
	skillID  string
	filePath string
	source   SourceType
	loadedAt time.Time
}

// NewSkillSource creates a new SkillSource with validation.
// Returns an error if any required field is missing or invalid.
func NewSkillSource(skillID, filePath string, source SourceType, loadedAt time.Time) (*SkillSource, error) {
	skillID = strings.TrimSpace(skillID)
	filePath = strings.TrimSpace(filePath)

	if skillID == "" {
		return nil, ErrSourceSkillIDRequired
	}
	if filePath == "" {
		return nil, ErrSourceFilePathRequired
	}
	if !source.IsValid() {
		return nil, ErrSourceTypeInvalid
	}
	if loadedAt.IsZero() {
		return nil, ErrSourceLoadedAtRequired
	}

	return &SkillSource{
		skillID:  skillID,
		filePath: filePath,
		source:   source,
		loadedAt: loadedAt,
	}, nil
}

// SkillID returns the skill's unique identifier.
func (s *SkillSource) SkillID() string {
	return s.skillID
}

// FilePath returns the absolute path to the skill definition file.
func (s *SkillSource) FilePath() string {
	return s.filePath
}

// Source returns the source type indicating where the skill was loaded from.
func (s *SkillSource) Source() SourceType {
	return s.source
}

// LoadedAt returns the time when the skill was loaded.
func (s *SkillSource) LoadedAt() time.Time {
	return s.loadedAt
}
