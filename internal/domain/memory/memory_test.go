package memory

import (
	"slices"
	"strings"
	"testing"
)

func TestNewMemory(t *testing.T) {
	tests := []struct {
		name     string
		global   string
		project  string
		includes []IncludedFile
	}{
		{
			name:     "empty memory",
			global:   "",
			project:  "",
			includes: nil,
		},
		{
			name:     "global only",
			global:   "global content",
			project:  "",
			includes: nil,
		},
		{
			name:     "project only",
			global:   "",
			project:  "project content",
			includes: nil,
		},
		{
			name:     "global and project",
			global:   "global content",
			project:  "project content",
			includes: nil,
		},
		{
			name:    "with includes",
			global:  "global",
			project: "project",
			includes: []IncludedFile{
				{Path: "file1.md", Content: "include1", Source: "project"},
				{Path: "file2.md", Content: "include2", Source: "global"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemory(tt.global, tt.project, tt.includes)
			if m == nil {
				t.Fatal("NewMemory returned nil")
			}
			if m.GlobalContent() != tt.global {
				t.Errorf("GlobalContent() = %q, want %q", m.GlobalContent(), tt.global)
			}
			if m.ProjectContent() != tt.project {
				t.Errorf("ProjectContent() = %q, want %q", m.ProjectContent(), tt.project)
			}
		})
	}
}

func TestMemory_Combined_GlobalOnly(t *testing.T) {
	m := NewMemory("global content", "", nil)
	combined := m.Combined()

	if combined != "global content" {
		t.Errorf("Combined() = %q, want %q", combined, "global content")
	}
}

func TestMemory_Combined_ProjectOnly(t *testing.T) {
	m := NewMemory("", "project content", nil)
	combined := m.Combined()

	if combined != "project content" {
		t.Errorf("Combined() = %q, want %q", combined, "project content")
	}
}

func TestMemory_Combined_GlobalAndProject(t *testing.T) {
	m := NewMemory("global content", "project content", nil)
	combined := m.Combined()

	// Project should come first, then global
	if !strings.HasPrefix(combined, "project content") {
		t.Error("Combined() should start with project content")
	}
	if !strings.Contains(combined, "global content") {
		t.Error("Combined() should contain global content")
	}
	if !strings.Contains(combined, "---") {
		t.Error("Combined() should contain separator")
	}
}

func TestMemory_Combined_WithIncludes(t *testing.T) {
	includes := []IncludedFile{
		{Path: "file1.md", Content: "include content 1", Source: "project"},
		{Path: "file2.md", Content: "include content 2", Source: "global"},
	}
	m := NewMemory("global", "project", includes)
	combined := m.Combined()

	if !strings.Contains(combined, "project") {
		t.Error("Combined() should contain project content")
	}
	if !strings.Contains(combined, "global") {
		t.Error("Combined() should contain global content")
	}
	if !strings.Contains(combined, "include content 1") {
		t.Error("Combined() should contain first include")
	}
	if !strings.Contains(combined, "include content 2") {
		t.Error("Combined() should contain second include")
	}
}

func TestMemory_Combined_OrderIsProjectGlobalIncludes(t *testing.T) {
	includes := []IncludedFile{
		{Path: "inc.md", Content: "INCLUDE", Source: "project"},
	}
	m := NewMemory("GLOBAL", "PROJECT", includes)
	combined := m.Combined()

	projectIdx := strings.Index(combined, "PROJECT")
	globalIdx := strings.Index(combined, "GLOBAL")
	includeIdx := strings.Index(combined, "INCLUDE")

	if projectIdx == -1 || globalIdx == -1 || includeIdx == -1 {
		t.Fatal("Combined() missing expected content")
	}

	if projectIdx > globalIdx {
		t.Error("Project content should come before global content")
	}
	if globalIdx > includeIdx {
		t.Error("Global content should come before includes")
	}
}

func TestMemory_Combined_Empty(t *testing.T) {
	m := NewMemory("", "", nil)
	combined := m.Combined()

	if combined != "" {
		t.Errorf("Combined() = %q, want empty string", combined)
	}
}

func TestMemory_Combined_EmptyInclude(t *testing.T) {
	includes := []IncludedFile{
		{Path: "empty.md", Content: "", Source: "project"},
	}
	m := NewMemory("", "", includes)
	combined := m.Combined()

	if combined != "" {
		t.Errorf("Combined() with empty include = %q, want empty string", combined)
	}
}

func TestMemory_IsEmpty_True(t *testing.T) {
	tests := []struct {
		name     string
		global   string
		project  string
		includes []IncludedFile
	}{
		{
			name:     "all empty",
			global:   "",
			project:  "",
			includes: nil,
		},
		{
			name:    "empty includes only",
			global:  "",
			project: "",
			includes: []IncludedFile{
				{Path: "empty.md", Content: "", Source: "project"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemory(tt.global, tt.project, tt.includes)
			if !m.IsEmpty() {
				t.Error("IsEmpty() should return true")
			}
		})
	}
}

func TestMemory_IsEmpty_False(t *testing.T) {
	tests := []struct {
		name     string
		global   string
		project  string
		includes []IncludedFile
	}{
		{
			name:     "global only",
			global:   "content",
			project:  "",
			includes: nil,
		},
		{
			name:     "project only",
			global:   "",
			project:  "content",
			includes: nil,
		},
		{
			name:    "include only",
			global:  "",
			project: "",
			includes: []IncludedFile{
				{Path: "file.md", Content: "content", Source: "project"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemory(tt.global, tt.project, tt.includes)
			if m.IsEmpty() {
				t.Error("IsEmpty() should return false")
			}
		})
	}
}

func TestMemory_EstimatedTokens(t *testing.T) {
	tests := []struct {
		name    string
		global  string
		project string
		wantMin int
		wantMax int
	}{
		{
			name:    "empty",
			global:  "",
			project: "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "short content",
			global:  "test",
			project: "",
			wantMin: 1,
			wantMax: 2,
		},
		{
			name:    "longer content",
			global:  strings.Repeat("word ", 100), // ~500 chars
			project: "",
			wantMin: 100,
			wantMax: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemory(tt.global, tt.project, nil)
			tokens := m.EstimatedTokens()
			if tokens < tt.wantMin || tokens > tt.wantMax {
				t.Errorf("EstimatedTokens() = %d, want between %d and %d", tokens, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestMemory_Sources(t *testing.T) {
	tests := []struct {
		name         string
		global       string
		project      string
		includes     []IncludedFile
		wantSources  []string
		wantContains []string
	}{
		{
			name:        "empty",
			global:      "",
			project:     "",
			includes:    nil,
			wantSources: nil,
		},
		{
			name:         "project only",
			global:       "",
			project:      "content",
			includes:     nil,
			wantContains: []string{"project"},
		},
		{
			name:         "global only",
			global:       "content",
			project:      "",
			includes:     nil,
			wantContains: []string{"global"},
		},
		{
			name:         "both",
			global:       "global",
			project:      "project",
			includes:     nil,
			wantContains: []string{"project", "global"},
		},
		{
			name:    "with includes",
			global:  "global",
			project: "project",
			includes: []IncludedFile{
				{Path: "file.md", Content: "inc", Source: "project"},
			},
			wantContains: []string{"project", "global", "include:file.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemory(tt.global, tt.project, tt.includes)
			sources := m.Sources()

			if tt.wantSources != nil {
				if sources != nil {
					t.Errorf("Sources() = %v, want nil", sources)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !slices.Contains(sources, want) {
					t.Errorf("Sources() missing %q, got %v", want, sources)
				}
			}
		})
	}
}

func TestMemory_Includes_DefensiveCopy(t *testing.T) {
	original := []IncludedFile{
		{Path: "file.md", Content: "content", Source: "project"},
	}
	m := NewMemory("", "", original)

	// Modify the original
	original[0].Path = "modified.md"

	// Memory should not be affected
	includes := m.Includes()
	if includes[0].Path != "file.md" {
		t.Error("Memory should have a defensive copy of includes")
	}

	// Modify the returned includes
	includes[0].Path = "also-modified.md"

	// Memory should not be affected
	includes2 := m.Includes()
	if includes2[0].Path != "file.md" {
		t.Error("Includes() should return a defensive copy")
	}
}

func TestMemory_Sources_DefensiveCopy(t *testing.T) {
	m := NewMemory("global", "project", nil)
	sources := m.Sources()

	// Modify the returned sources
	sources[0] = "modified"

	// Memory should not be affected
	sources2 := m.Sources()
	if sources2[0] == "modified" {
		t.Error("Sources() should return a defensive copy")
	}
}

func TestMemory_Includes_Nil(t *testing.T) {
	m := NewMemory("global", "", nil)
	includes := m.Includes()

	if includes != nil {
		t.Errorf("Includes() = %v, want nil for empty includes", includes)
	}
}

func TestErrors(t *testing.T) {
	// Test that error variables are defined and have expected messages
	tests := []struct {
		err     error
		wantMsg string
	}{
		{ErrMemoryEmpty, "memory content is empty"},
		{ErrIncludeNotFound, "included file not found"},
		{ErrIncludeCycle, "circular include detected"},
		{ErrTokenLimitExceeded, "memory exceeds token limit"},
	}

	for _, tt := range tests {
		t.Run(tt.wantMsg, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("Error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}
