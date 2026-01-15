// Package memory provides domain models for memory/context persistence across sessions.
package memory

import (
	"strings"
)

// IncludedFile represents a file that was included via @include directive.
type IncludedFile struct {
	// Path is the resolved path to the included file.
	Path string

	// Content is the content of the included file.
	Content string

	// Source indicates where the include directive was found ("global" or "project").
	Source string
}

// Memory represents the aggregated memory content from global and project sources.
// Memory is immutable after creation - use the constructor to create new instances.
type Memory struct {
	globalContent  string
	projectContent string
	includes       []IncludedFile
	sources        []string
}

// NewMemory creates a new Memory instance with the given content.
// The includes slice is defensively copied to prevent external mutation.
func NewMemory(global, project string, includes []IncludedFile) *Memory {
	// Build sources list
	sources := make([]string, 0, 2+len(includes))
	if project != "" {
		sources = append(sources, "project")
	}
	if global != "" {
		sources = append(sources, "global")
	}
	for _, inc := range includes {
		sources = append(sources, "include:"+inc.Path)
	}

	// Defensive copy of includes
	includesCopy := make([]IncludedFile, len(includes))
	copy(includesCopy, includes)

	return &Memory{
		globalContent:  global,
		projectContent: project,
		includes:       includesCopy,
		sources:        sources,
	}
}

// GlobalContent returns the global memory content.
func (m *Memory) GlobalContent() string {
	return m.globalContent
}

// ProjectContent returns the project-specific memory content.
func (m *Memory) ProjectContent() string {
	return m.projectContent
}

// Includes returns a copy of the included files.
func (m *Memory) Includes() []IncludedFile {
	if len(m.includes) == 0 {
		return nil
	}
	result := make([]IncludedFile, len(m.includes))
	copy(result, m.includes)
	return result
}

// Sources returns a list of sources that contributed to this memory.
// Sources are listed in priority order: project, global, includes.
func (m *Memory) Sources() []string {
	if len(m.sources) == 0 {
		return nil
	}
	result := make([]string, len(m.sources))
	copy(result, m.sources)
	return result
}

// Combined returns the merged memory content in priority order:
// project content first (most specific), then global, then includes.
// Each section is separated by a divider line.
func (m *Memory) Combined() string {
	var parts []string

	// Project content first (highest priority)
	if m.projectContent != "" {
		parts = append(parts, m.projectContent)
	}

	// Global content second
	if m.globalContent != "" {
		parts = append(parts, m.globalContent)
	}

	// Includes last
	for _, inc := range m.includes {
		if inc.Content != "" {
			parts = append(parts, inc.Content)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// IsEmpty returns true if there is no memory content from any source.
func (m *Memory) IsEmpty() bool {
	if m.globalContent != "" || m.projectContent != "" {
		return false
	}
	for _, inc := range m.includes {
		if inc.Content != "" {
			return false
		}
	}
	return true
}

// EstimatedTokens returns an approximate token count for the combined content.
// Uses a simple heuristic: ~4 characters per token (common for English text).
func (m *Memory) EstimatedTokens() int {
	combined := m.Combined()
	if combined == "" {
		return 0
	}
	// Simple heuristic: approximately 4 characters per token
	// This is a rough estimate - actual tokenization varies by model
	return (len(combined) + 3) / 4
}
