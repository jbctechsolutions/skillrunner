package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// File represents a loaded file with its content and metadata.
// It contains the file path, content as a string, and file size in bytes.
type File struct {
	Path    string `json:"path"`    // Absolute path to the file
	Content string `json:"content"` // File content as a string
	Size    int64  `json:"size"`    // File size in bytes
}

// Loader handles loading files from the filesystem.
// It supports loading files and folders, with pattern-based filtering.
// Paths can be absolute or relative to the workspace path.
type Loader struct {
	workspacePath string // Base workspace path for resolving relative paths
}

// NewLoader creates a new Loader instance with the specified workspace path.
// The workspace path is used to resolve relative file and folder paths.
func NewLoader(workspacePath string) *Loader {
	return &Loader{
		workspacePath: workspacePath,
	}
}

// LoadFolder loads all files from a folder matching the given pattern.
// The folderPath can be absolute or relative to the workspace.
// If pattern is empty, all files are loaded recursively.
// Pattern uses filepath.Match syntax (e.g., "*.md", "*.go").
// Returns an error if the folder doesn't exist or is not a directory.
func (l *Loader) LoadFolder(folderPath string, pattern string) ([]*File, error) {
	var files []*File

	// Resolve path relative to workspace if not absolute
	absPath := folderPath
	if !filepath.IsAbs(folderPath) {
		absPath = filepath.Join(l.workspacePath, folderPath)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", folderPath)
	}

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/directories we can't access
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Apply pattern filter if provided
		if pattern != "" {
			matched, err := filepath.Match(pattern, info.Name())
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			// Skip files we can't read
			return nil
		}

		files = append(files, &File{
			Path:    path,
			Content: string(content),
			Size:    info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", folderPath, err)
	}

	return files, nil
}

// LoadFile loads a single file from the filesystem.
// The filePath can be absolute or relative to the workspace.
// Returns an error if the file doesn't exist or is a directory.
func (l *Loader) LoadFile(filePath string) (*File, error) {
	// Resolve path relative to workspace if not absolute
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(l.workspacePath, filePath)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &File{
		Path:    absPath,
		Content: string(content),
		Size:    info.Size(),
	}, nil
}

// FilterFiles filters a list of files by the given pattern.
// Pattern uses filepath.Match syntax (e.g., "*.md", "*.go").
// If pattern is empty, returns all files unchanged.
// Files that don't match the pattern are excluded from the result.
func (l *Loader) FilterFiles(files []*File, pattern string) []*File {
	if pattern == "" {
		return files
	}

	var filtered []*File
	for _, file := range files {
		matched, err := filepath.Match(pattern, filepath.Base(file.Path))
		if err != nil {
			continue
		}
		if matched {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

// GetFileExtension returns the file extension (without the dot)
// Files starting with a dot (like .hidden) are considered to have no extension
func (l *Loader) GetFileExtension(filePath string) string {
	base := filepath.Base(filePath)
	// Handle hidden files (starting with dot)
	if strings.HasPrefix(base, ".") && !strings.Contains(base[1:], ".") {
		return ""
	}
	ext := filepath.Ext(filePath)
	if ext == "" {
		return ""
	}
	return strings.TrimPrefix(ext, ".")
}

// IsTextFile checks if a file is likely a text file based on its extension.
// Returns true for common text file extensions (md, txt, go, js, py, etc.).
// The check is case-insensitive.
func (l *Loader) IsTextFile(filePath string) bool {
	ext := strings.ToLower(l.GetFileExtension(filePath))
	textExtensions := map[string]bool{
		"md":   true,
		"txt":  true,
		"go":   true,
		"js":   true,
		"ts":   true,
		"py":   true,
		"rs":   true,
		"java": true,
		"c":    true,
		"cpp":  true,
		"h":    true,
		"hpp":  true,
		"json": true,
		"yaml": true,
		"yml":  true,
		"toml": true,
		"xml":  true,
		"html": true,
		"css":  true,
		"sh":   true,
		"bash": true,
		"zsh":  true,
	}
	return textExtensions[ext]
}
