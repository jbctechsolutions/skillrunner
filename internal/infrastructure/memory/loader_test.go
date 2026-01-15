package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainmemory "github.com/jbctechsolutions/skillrunner/internal/domain/memory"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader(2000)
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if loader.maxTokens != 2000 {
		t.Errorf("maxTokens = %d, want 2000", loader.maxTokens)
	}
}

func TestNewLoaderWithHomeDir(t *testing.T) {
	loader := NewLoaderWithHomeDir(1000, "/custom/home")
	if loader == nil {
		t.Fatal("NewLoaderWithHomeDir returned nil")
	}
	if loader.maxTokens != 1000 {
		t.Errorf("maxTokens = %d, want 1000", loader.maxTokens)
	}
	if loader.homeDir != "/custom/home" {
		t.Errorf("homeDir = %q, want %q", loader.homeDir, "/custom/home")
	}
}

func TestLoader_LoadGlobalMemory_MEMORY(t *testing.T) {
	// Create temp directory structure
	homeDir := t.TempDir()
	skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create MEMORY.md
	memoryPath := filepath.Join(skillrunnerDir, "MEMORY.md")
	if err := os.WriteFile(memoryPath, []byte("# Global Memory"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoaderWithHomeDir(2000, homeDir)
	content, source, err := loader.loadGlobalMemory()

	if err != nil {
		t.Fatalf("loadGlobalMemory() error = %v", err)
	}
	if content != "# Global Memory" {
		t.Errorf("content = %q, want %q", content, "# Global Memory")
	}
	if source != memoryPath {
		t.Errorf("source = %q, want %q", source, memoryPath)
	}
}

func TestLoader_LoadGlobalMemory_CLAUDE_Fallback(t *testing.T) {
	homeDir := t.TempDir()
	skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create only CLAUDE.md (no MEMORY.md)
	claudePath := filepath.Join(skillrunnerDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Claude Memory"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoaderWithHomeDir(2000, homeDir)
	content, source, err := loader.loadGlobalMemory()

	if err != nil {
		t.Fatalf("loadGlobalMemory() error = %v", err)
	}
	if content != "# Claude Memory" {
		t.Errorf("content = %q, want %q", content, "# Claude Memory")
	}
	if source != claudePath {
		t.Errorf("source = %q, want %q", source, claudePath)
	}
}

func TestLoader_LoadGlobalMemory_NotFound(t *testing.T) {
	homeDir := t.TempDir()
	loader := NewLoaderWithHomeDir(2000, homeDir)

	_, _, err := loader.loadGlobalMemory()
	if !os.IsNotExist(err) {
		t.Errorf("loadGlobalMemory() error = %v, want os.ErrNotExist", err)
	}
}

func TestLoader_LoadGlobalMemory_EmptyHomeDir(t *testing.T) {
	loader := NewLoaderWithHomeDir(2000, "")

	_, _, err := loader.loadGlobalMemory()
	if !os.IsNotExist(err) {
		t.Errorf("loadGlobalMemory() error = %v, want os.ErrNotExist", err)
	}
}

func TestLoader_LoadProjectMemory_MEMORY(t *testing.T) {
	projectDir := t.TempDir()

	// Create MEMORY.md
	memoryPath := filepath.Join(projectDir, "MEMORY.md")
	if err := os.WriteFile(memoryPath, []byte("# Project Memory"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(2000)
	content, source, err := loader.loadProjectMemory(projectDir)

	if err != nil {
		t.Fatalf("loadProjectMemory() error = %v", err)
	}
	if content != "# Project Memory" {
		t.Errorf("content = %q, want %q", content, "# Project Memory")
	}
	if source != memoryPath {
		t.Errorf("source = %q, want %q", source, memoryPath)
	}
}

func TestLoader_LoadProjectMemory_CLAUDE_Fallback(t *testing.T) {
	projectDir := t.TempDir()

	// Create only CLAUDE.md
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Claude Project"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(2000)
	content, source, err := loader.loadProjectMemory(projectDir)

	if err != nil {
		t.Fatalf("loadProjectMemory() error = %v", err)
	}
	if content != "# Claude Project" {
		t.Errorf("content = %q, want %q", content, "# Claude Project")
	}
	if source != claudePath {
		t.Errorf("source = %q, want %q", source, claudePath)
	}
}

func TestLoader_LoadProjectMemory_NotFound(t *testing.T) {
	projectDir := t.TempDir()
	loader := NewLoader(2000)

	_, _, err := loader.loadProjectMemory(projectDir)
	if !os.IsNotExist(err) {
		t.Errorf("loadProjectMemory() error = %v, want os.ErrNotExist", err)
	}
}

func TestLoader_LoadProjectMemory_EmptyDir(t *testing.T) {
	loader := NewLoader(2000)

	_, _, err := loader.loadProjectMemory("")
	if !os.IsNotExist(err) {
		t.Errorf("loadProjectMemory() error = %v, want os.ErrNotExist", err)
	}
}

func TestLoader_Load_GlobalAndProject(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	// Create global MEMORY.md
	skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillrunnerDir, "MEMORY.md"), []byte("Global content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project MEMORY.md
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("Project content"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoaderWithHomeDir(2000, homeDir)
	mem, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mem.GlobalContent() != "Global content" {
		t.Errorf("GlobalContent() = %q, want %q", mem.GlobalContent(), "Global content")
	}
	if mem.ProjectContent() != "Project content" {
		t.Errorf("ProjectContent() = %q, want %q", mem.ProjectContent(), "Project content")
	}
}

func TestLoader_Load_GlobalOnly(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	// Create only global MEMORY.md
	skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillrunnerDir, "MEMORY.md"), []byte("Global only"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoaderWithHomeDir(2000, homeDir)
	mem, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mem.GlobalContent() != "Global only" {
		t.Errorf("GlobalContent() = %q, want %q", mem.GlobalContent(), "Global only")
	}
	if mem.ProjectContent() != "" {
		t.Errorf("ProjectContent() = %q, want empty", mem.ProjectContent())
	}
}

func TestLoader_Load_ProjectOnly(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	// Create only project MEMORY.md
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte("Project only"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoaderWithHomeDir(2000, homeDir)
	mem, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mem.GlobalContent() != "" {
		t.Errorf("GlobalContent() = %q, want empty", mem.GlobalContent())
	}
	if mem.ProjectContent() != "Project only" {
		t.Errorf("ProjectContent() = %q, want %q", mem.ProjectContent(), "Project only")
	}
}

func TestLoader_Load_Neither(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	loader := NewLoaderWithHomeDir(2000, homeDir)
	mem, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !mem.IsEmpty() {
		t.Error("Memory should be empty")
	}
}

func TestLoader_ParseIncludes_SingleFile(t *testing.T) {
	baseDir := t.TempDir()

	// Create included file
	includePath := filepath.Join(baseDir, "extra.md")
	if err := os.WriteFile(includePath, []byte("Extra content"), 0644); err != nil {
		t.Fatal(err)
	}

	content := "# Main\n@include: ./extra.md\nMore content"
	loader := NewLoader(2000)
	includes, err := loader.parseIncludes(content, baseDir, "project")

	if err != nil {
		t.Fatalf("parseIncludes() error = %v", err)
	}
	if len(includes) != 1 {
		t.Fatalf("len(includes) = %d, want 1", len(includes))
	}
	if includes[0].Content != "Extra content" {
		t.Errorf("includes[0].Content = %q, want %q", includes[0].Content, "Extra content")
	}
	if includes[0].Source != "project" {
		t.Errorf("includes[0].Source = %q, want %q", includes[0].Source, "project")
	}
}

func TestLoader_ParseIncludes_MultipleFiles(t *testing.T) {
	baseDir := t.TempDir()

	// Create included files
	if err := os.WriteFile(filepath.Join(baseDir, "file1.md"), []byte("Content 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "file2.md"), []byte("Content 2"), 0644); err != nil {
		t.Fatal(err)
	}

	content := "@include: ./file1.md\n@include: ./file2.md"
	loader := NewLoader(2000)
	includes, err := loader.parseIncludes(content, baseDir, "global")

	if err != nil {
		t.Fatalf("parseIncludes() error = %v", err)
	}
	if len(includes) != 2 {
		t.Fatalf("len(includes) = %d, want 2", len(includes))
	}
}

func TestLoader_ParseIncludes_MissingFile_Graceful(t *testing.T) {
	baseDir := t.TempDir()

	content := "@include: ./missing.md"
	loader := NewLoader(2000)
	includes, err := loader.parseIncludes(content, baseDir, "project")

	if err != nil {
		t.Fatalf("parseIncludes() error = %v", err)
	}
	// Missing files should be skipped gracefully
	if len(includes) != 0 {
		t.Errorf("len(includes) = %d, want 0 (missing files skipped)", len(includes))
	}
}

func TestLoader_ParseIncludes_RelativePath(t *testing.T) {
	baseDir := t.TempDir()

	// Create subdirectory with file
	subDir := filepath.Join(baseDir, "docs")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "readme.md"), []byte("Docs content"), 0644); err != nil {
		t.Fatal(err)
	}

	content := "@include: ./docs/readme.md"
	loader := NewLoader(2000)
	includes, err := loader.parseIncludes(content, baseDir, "project")

	if err != nil {
		t.Fatalf("parseIncludes() error = %v", err)
	}
	if len(includes) != 1 {
		t.Fatalf("len(includes) = %d, want 1", len(includes))
	}
	if includes[0].Content != "Docs content" {
		t.Errorf("includes[0].Content = %q, want %q", includes[0].Content, "Docs content")
	}
}

func TestLoader_ParseIncludes_NoIncludes(t *testing.T) {
	baseDir := t.TempDir()

	content := "# Just regular content\nNo includes here"
	loader := NewLoader(2000)
	includes, err := loader.parseIncludes(content, baseDir, "project")

	if err != nil {
		t.Fatalf("parseIncludes() error = %v", err)
	}
	if len(includes) != 0 {
		t.Errorf("len(includes) = %d, want 0", len(includes))
	}
}

func TestLoader_ParseIncludes_DuplicatePrevention(t *testing.T) {
	baseDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(baseDir, "same.md"), []byte("Same content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Same file included twice
	content := "@include: ./same.md\n@include: ./same.md"
	loader := NewLoader(2000)
	includes, err := loader.parseIncludes(content, baseDir, "project")

	if err != nil {
		t.Fatalf("parseIncludes() error = %v", err)
	}
	// Should only include once
	if len(includes) != 1 {
		t.Errorf("len(includes) = %d, want 1 (duplicates prevented)", len(includes))
	}
}

func TestLoader_Load_WithIncludes(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	// Create project MEMORY.md with include
	memoryContent := "# Project\n@include: ./extra.md"
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte(memoryContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "extra.md"), []byte("Extra content"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoaderWithHomeDir(2000, homeDir)
	mem, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	includes := mem.Includes()
	if len(includes) != 1 {
		t.Fatalf("len(Includes()) = %d, want 1", len(includes))
	}
	if includes[0].Content != "Extra content" {
		t.Errorf("includes[0].Content = %q, want %q", includes[0].Content, "Extra content")
	}
}

func TestLoader_Load_RespectsTokenLimit(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	// Create large content (much larger than limit)
	largeContent := strings.Repeat("word ", 1000) // ~5000 chars
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte(largeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Very small token limit
	loader := NewLoaderWithHomeDir(100, homeDir)
	mem, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Content should be truncated
	if mem.EstimatedTokens() > 100 {
		t.Errorf("EstimatedTokens() = %d, should be <= 100", mem.EstimatedTokens())
	}
}

func TestLoader_Load_TruncatesWhenOverLimit(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	// Create content that will exceed limit
	globalContent := strings.Repeat("G", 500)  // 500 chars
	projectContent := strings.Repeat("P", 500) // 500 chars

	// Create global
	skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillrunnerDir, "MEMORY.md"), []byte(globalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project
	if err := os.WriteFile(filepath.Join(projectDir, "MEMORY.md"), []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Limit to ~200 tokens (800 chars)
	loader := NewLoaderWithHomeDir(200, homeDir)
	mem, err := loader.Load(projectDir)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	combined := mem.Combined()
	// Should be truncated to fit within limit (allowing some slack for separator)
	if len(combined) > 850 {
		t.Errorf("Combined length = %d, should be <= 850", len(combined))
	}
}

func TestLoader_TruncateMemory_NoTruncationNeeded(t *testing.T) {
	loader := NewLoaderWithHomeDir(1000, "")

	// Create memory that fits within limit
	mem := newTestMemory("content", "short", nil)

	result := loader.truncateMemory(mem)

	if result.ProjectContent() != "short" {
		t.Error("Project content should not be truncated")
	}
	if result.GlobalContent() != "content" {
		t.Error("Global content should not be truncated")
	}
}

func TestLoader_TruncateMemory_TruncatesGlobal(t *testing.T) {
	loader := NewLoaderWithHomeDir(50, "") // ~200 chars limit

	// Project fits, global needs truncation
	mem := newTestMemory(strings.Repeat("G", 300), "Project content", nil)

	result := loader.truncateMemory(mem)

	if result.ProjectContent() != "Project content" {
		t.Error("Project content should be preserved")
	}
	// Global should be truncated or removed
	if len(result.GlobalContent()) >= 300 {
		t.Error("Global content should be truncated")
	}
}

// newTestMemory creates a domainmemory.Memory for testing using the domain constructor.
func newTestMemory(global, project string, includes []domainmemory.IncludedFile) *domainmemory.Memory {
	return domainmemory.NewMemory(global, project, includes)
}

func TestTruncateAtWordBoundary(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		limit    int
		expected string
	}{
		{
			name:     "no truncation needed",
			content:  "short text",
			limit:    100,
			expected: "short text",
		},
		{
			name:     "truncate at word boundary",
			content:  "hello world foo bar",
			limit:    15,
			expected: "hello world...",
		},
		{
			name:     "truncate with no space",
			content:  "verylongwordwithoutspaces",
			limit:    15,
			expected: "verylongword...",
		},
		{
			name:     "very short limit",
			content:  "hello world",
			limit:    3,
			expected: "...",
		},
		{
			name:     "limit equals ellipsis length",
			content:  "hello",
			limit:    3,
			expected: "...",
		},
		{
			name:     "limit less than ellipsis",
			content:  "hello",
			limit:    2,
			expected: "..",
		},
		{
			name:     "empty content",
			content:  "",
			limit:    10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateAtWordBoundary(tt.content, tt.limit)
			if result != tt.expected {
				t.Errorf("truncateAtWordBoundary(%q, %d) = %q, want %q",
					tt.content, tt.limit, result, tt.expected)
			}
			if len(result) > tt.limit {
				t.Errorf("result length %d exceeds limit %d", len(result), tt.limit)
			}
		})
	}
}
