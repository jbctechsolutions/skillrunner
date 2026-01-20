// Package context provides context detection and injection for skills.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileReference represents a detected file reference in user input.
type FileReference struct {
	Original string // Original text from input
	Path     string // Resolved file path
	Exists   bool   // Whether file exists
	Size     int64  // File size in bytes
}

// FileDetector detects file references in user input.
type FileDetector struct {
	workingDir string
	maxSize    int64 // Max file size to read (default 1MB)
}

// NewFileDetector creates a new file detector.
func NewFileDetector() *FileDetector {
	wd, _ := os.Getwd()
	return &FileDetector{
		workingDir: wd,
		maxSize:    1024 * 1024, // 1MB default
	}
}

// DetectFiles finds file references in the input string.
func (d *FileDetector) DetectFiles(input string) []FileReference {
	var refs []FileReference
	seen := make(map[string]bool) // Maps absolute paths to prevent duplicates

	// Pattern 1: Explicit file paths with extensions
	// Matches: file.go, ./file.go, path/to/file.go, *.go
	filePathPattern := regexp.MustCompile(`(?:^|\s)(\.{0,2}/?[\w\-./]+\.\w+)(?:\s|$|[,;:)])`)
	matches := filePathPattern.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		if len(match) > 1 {
			candidate := strings.TrimSpace(match[1])

			ref := d.resolveReference(candidate)
			if ref.Exists {
				// Deduplicate by absolute path, not original candidate string
				if seen[ref.Path] {
					continue
				}
				seen[ref.Path] = true
				refs = append(refs, ref)
			}
		}
	}

	// Pattern 2: Bare filenames with common code extensions
	// Matches: main.go, phase_executor.go, config.yaml (without path)
	// Uses word boundaries for reliable matching
	bareFilePattern := regexp.MustCompile(`\b([\w\-]+\.(go|yaml|yml|json|js|ts|py|rb|java|rs|md|txt))\b`)
	matches = bareFilePattern.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		if len(match) > 1 {
			candidate := strings.TrimSpace(match[1])

			// Try to find the file in current dir or recursively
			ref := d.findFile(candidate)
			if ref.Exists {
				// Deduplicate by absolute path
				if seen[ref.Path] {
					continue
				}
				seen[ref.Path] = true
				refs = append(refs, ref)
			}
		}
	}

	return refs
}

// resolveReference resolves a file reference to an absolute path.
func (d *FileDetector) resolveReference(ref string) FileReference {
	result := FileReference{
		Original: ref,
	}

	// Handle relative paths
	var absPath string
	if filepath.IsAbs(ref) {
		absPath = ref
	} else {
		absPath = filepath.Join(d.workingDir, ref)
	}

	// Clean the path
	absPath = filepath.Clean(absPath)

	// Check if file exists
	info, err := os.Stat(absPath)
	if err == nil && !info.IsDir() {
		result.Path = absPath
		result.Exists = true
		result.Size = info.Size()
	}

	return result
}

// findFile searches for a file by name in the working directory and subdirectories.
func (d *FileDetector) findFile(filename string) FileReference {
	result := FileReference{
		Original: filename,
	}

	// First check current directory
	currentPath := filepath.Join(d.workingDir, filename)
	if info, err := os.Stat(currentPath); err == nil && !info.IsDir() {
		result.Path = currentPath
		result.Exists = true
		result.Size = info.Size()
		return result
	}

	// Search in subdirectories (limit depth to 5 for better coverage)
	var foundPath string
	workingDir := d.workingDir // Capture for closure
	maxDepth := 5

	_ = filepath.WalkDir(workingDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir // Skip inaccessible directories
		}

		// Skip hidden directories and common ignore patterns
		if entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" || name == "build" {
				return filepath.SkipDir
			}
			// Limit depth
			rel, _ := filepath.Rel(workingDir, path)
			depth := strings.Count(rel, string(os.PathSeparator))
			if depth > maxDepth {
				return filepath.SkipDir
			}
		}

		// Check if this is the file we're looking for
		if !entry.IsDir() && entry.Name() == filename {
			foundPath = path
			return filepath.SkipAll // Found it, stop walking
		}
		return nil
	})

	if foundPath != "" {
		if info, err := os.Stat(foundPath); err == nil {
			result.Path = foundPath
			result.Exists = true
			result.Size = info.Size()
		}
	}

	return result
}

// InjectFileContext reads files and injects their content into the input.
func (d *FileDetector) InjectFileContext(input string, refs []FileReference) (string, error) {
	if len(refs) == 0 {
		return input, nil
	}

	var builder strings.Builder

	// Add file context section
	builder.WriteString("=== FILE CONTEXT ===\n\n")

	for i, ref := range refs {
		// Check size limit
		if ref.Size > d.maxSize {
			builder.WriteString(fmt.Sprintf("--- %s ---\n", ref.Path))
			builder.WriteString(fmt.Sprintf("[File too large: %d bytes, limit: %d bytes]\n\n", ref.Size, d.maxSize))
			continue
		}

		// Read file content
		content, err := os.ReadFile(ref.Path)
		if err != nil {
			builder.WriteString(fmt.Sprintf("--- %s ---\n", ref.Path))
			builder.WriteString(fmt.Sprintf("[Error reading file: %v]\n\n", err))
			continue
		}

		// Check if binary (heuristic: high ratio of non-printable chars)
		if isBinary(content) {
			builder.WriteString(fmt.Sprintf("--- %s ---\n", ref.Path))
			builder.WriteString("[Binary file, content not shown]\n\n")
			continue
		}

		// Write file content
		relPath, _ := filepath.Rel(d.workingDir, ref.Path)
		builder.WriteString(fmt.Sprintf("--- File %d: %s ---\n", i+1, relPath))
		builder.Write(content)
		builder.WriteString("\n\n")
	}

	// Add original user query
	builder.WriteString("=== USER QUERY ===\n")
	builder.WriteString(input)

	return builder.String(), nil
}

// isBinary checks if content appears to be binary (heuristic).
func isBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 512 bytes for non-printable characters
	checkLen := 512
	if len(content) < checkLen {
		checkLen = len(content)
	}

	nonPrintable := 0
	for i := 0; i < checkLen; i++ {
		b := content[i]
		// Allow common whitespace
		if b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		// Count non-printable ASCII
		if b < 32 || b > 126 {
			nonPrintable++
		}
	}

	// If more than 30% non-printable, consider binary
	return float64(nonPrintable)/float64(checkLen) > 0.3
}
