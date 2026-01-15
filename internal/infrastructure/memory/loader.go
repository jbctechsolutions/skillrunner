// Package memory provides infrastructure for loading memory files (MEMORY.md/CLAUDE.md).
package memory

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/domain/memory"
)

// Memory file names in priority order.
const (
	MemoryFileName = "MEMORY.md"
	ClaudeFileName = "CLAUDE.md"
)

// includePattern matches @include: ./path/to/file.md directives.
var includePattern = regexp.MustCompile(`^@include:\s*(.+)$`)

// Loader loads memory files from global and project locations.
type Loader struct {
	maxTokens int
	homeDir   string
}

// NewLoader creates a new memory loader with the specified max tokens.
func NewLoader(maxTokens int) *Loader {
	homeDir, _ := os.UserHomeDir()
	return &Loader{
		maxTokens: maxTokens,
		homeDir:   homeDir,
	}
}

// NewLoaderWithHomeDir creates a loader with a custom home directory (for testing).
func NewLoaderWithHomeDir(maxTokens int, homeDir string) *Loader {
	return &Loader{
		maxTokens: maxTokens,
		homeDir:   homeDir,
	}
}

// Load loads memory from both global and project locations.
// Returns a Memory domain object combining all sources.
func (l *Loader) Load(projectDir string) (*memory.Memory, error) {
	// Load global memory
	globalContent, globalSource, err := l.loadGlobalMemory()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Load project memory
	projectContent, projectSource, err := l.loadProjectMemory(projectDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Parse includes from both sources
	var includes []memory.IncludedFile

	// Parse global includes
	// Note: parseIncludes only returns scanner errors which are unlikely;
	// missing include files are gracefully skipped within parseIncludes itself.
	if globalContent != "" && globalSource != "" {
		globalDir := filepath.Dir(globalSource)
		globalIncludes, _ := l.parseIncludes(globalContent, globalDir, "global")
		includes = append(includes, globalIncludes...)
	}

	// Parse project includes
	if projectContent != "" && projectSource != "" {
		projectDirPath := filepath.Dir(projectSource)
		projectIncludes, _ := l.parseIncludes(projectContent, projectDirPath, "project")
		includes = append(includes, projectIncludes...)
	}

	// Create memory object
	mem := memory.NewMemory(globalContent, projectContent, includes)

	// Check token limit and truncate if needed
	if l.maxTokens > 0 && mem.EstimatedTokens() > l.maxTokens {
		// Truncate combined content to fit within limit
		mem = l.truncateMemory(mem)
	}

	return mem, nil
}

// loadGlobalMemory loads memory from ~/.skillrunner/MEMORY.md or CLAUDE.md.
func (l *Loader) loadGlobalMemory() (content, source string, err error) {
	if l.homeDir == "" {
		return "", "", os.ErrNotExist
	}

	skillrunnerDir := filepath.Join(l.homeDir, ".skillrunner")

	// Try MEMORY.md first
	memoryPath := filepath.Join(skillrunnerDir, MemoryFileName)
	content, err = l.readFile(memoryPath)
	if err == nil {
		return content, memoryPath, nil
	}

	// Fall back to CLAUDE.md
	claudePath := filepath.Join(skillrunnerDir, ClaudeFileName)
	content, err = l.readFile(claudePath)
	if err == nil {
		return content, claudePath, nil
	}

	return "", "", os.ErrNotExist
}

// loadProjectMemory loads memory from project root MEMORY.md or CLAUDE.md.
func (l *Loader) loadProjectMemory(projectDir string) (content, source string, err error) {
	if projectDir == "" {
		return "", "", os.ErrNotExist
	}

	// Try MEMORY.md first
	memoryPath := filepath.Join(projectDir, MemoryFileName)
	content, err = l.readFile(memoryPath)
	if err == nil {
		return content, memoryPath, nil
	}

	// Fall back to CLAUDE.md
	claudePath := filepath.Join(projectDir, ClaudeFileName)
	content, err = l.readFile(claudePath)
	if err == nil {
		return content, claudePath, nil
	}

	return "", "", os.ErrNotExist
}

// parseIncludes extracts and loads @include directives from content.
func (l *Loader) parseIncludes(content, baseDir, source string) ([]memory.IncludedFile, error) {
	var includes []memory.IncludedFile
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := includePattern.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		// Resolve the path
		includePath := strings.TrimSpace(matches[1])
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(baseDir, includePath)
		}
		includePath = filepath.Clean(includePath)

		// Skip if already seen (cycle detection)
		if seen[includePath] {
			continue
		}
		seen[includePath] = true

		// Read the included file (graceful handling of missing files)
		includeContent, err := l.readFile(includePath)
		if err != nil {
			// Skip missing includes gracefully
			continue
		}

		includes = append(includes, memory.IncludedFile{
			Path:    includePath,
			Content: includeContent,
			Source:  source,
		})
	}

	return includes, scanner.Err()
}

// readFile reads the content of a file.
// The path is constructed from known base directories (homeDir, projectDir).
func (l *Loader) readFile(path string) (string, error) {
	// #nosec G304 -- path is constructed from controlled inputs (homeDir + known filenames)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// truncateMemory creates a new Memory with content truncated to fit maxTokens.
func (l *Loader) truncateMemory(mem *memory.Memory) *memory.Memory {
	// Calculate approximate character limit based on token limit
	// Using ~4 chars per token
	charLimit := l.maxTokens * 4

	combined := mem.Combined()
	if len(combined) <= charLimit {
		return mem
	}

	// Truncate - prioritize project content
	projectContent := mem.ProjectContent()
	globalContent := mem.GlobalContent()

	// Calculate available space
	remaining := charLimit

	// Keep as much project content as possible
	if len(projectContent) > remaining {
		projectContent = truncateAtWordBoundary(projectContent, remaining)
		globalContent = ""
	} else {
		remaining -= len(projectContent)
		if len(globalContent) > remaining {
			globalContent = truncateAtWordBoundary(globalContent, remaining)
		}
	}

	// Create new memory without includes (they would exceed the limit)
	return memory.NewMemory(globalContent, projectContent, nil)
}

// truncateAtWordBoundary truncates content at a word boundary, adding ellipsis if truncated.
func truncateAtWordBoundary(content string, limit int) string {
	if len(content) <= limit {
		return content
	}

	// Reserve space for ellipsis
	const ellipsis = "..."
	if limit <= len(ellipsis) {
		return ellipsis[:limit]
	}

	targetLen := limit - len(ellipsis)

	// Find the last space before the limit
	lastSpace := strings.LastIndex(content[:targetLen], " ")
	if lastSpace > 0 {
		return content[:lastSpace] + ellipsis
	}

	// No space found, truncate at the limit
	return content[:targetLen] + ellipsis
}
