package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_LoadFolder(t *testing.T) {
	// Create a temporary test directory structure
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.md":        "# Test File 1\nContent here",
		"file2.md":        "# Test File 2\nMore content",
		"file3.go":        "package main\nfunc main() {}",
		"subdir/file4.md": "# Subdir File\nContent",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	loader := NewLoader(tmpDir)

	t.Run("load all markdown files", func(t *testing.T) {
		files, err := loader.LoadFolder(tmpDir, "*.md")
		if err != nil {
			t.Fatalf("LoadFolder failed: %v", err)
		}

		// Should find file1.md, file2.md, and subdir/file4.md
		if len(files) != 3 {
			t.Errorf("Expected 3 markdown files, got %d", len(files))
		}

		// Verify file contents
		found := make(map[string]bool)
		for _, file := range files {
			found[filepath.Base(file.Path)] = true
			if file.Content == "" {
				t.Errorf("File %s has empty content", file.Path)
			}
		}

		if !found["file1.md"] || !found["file2.md"] {
			t.Error("Missing expected markdown files")
		}
	})

	t.Run("load go files", func(t *testing.T) {
		files, err := loader.LoadFolder(tmpDir, "*.go")
		if err != nil {
			t.Fatalf("LoadFolder failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 go file, got %d", len(files))
		}

		if filepath.Base(files[0].Path) != "file3.go" {
			t.Errorf("Expected file3.go, got %s", files[0].Path)
		}
	})

	t.Run("load all files with no filter", func(t *testing.T) {
		files, err := loader.LoadFolder(tmpDir, "")
		if err != nil {
			t.Fatalf("LoadFolder failed: %v", err)
		}

		if len(files) != 4 {
			t.Errorf("Expected 4 files, got %d", len(files))
		}
	})
}

func TestLoader_LoadFolder_Nonexistent(t *testing.T) {
	loader := NewLoader("/nonexistent")

	files, err := loader.LoadFolder("/nonexistent/path", "*.md")
	if err == nil {
		t.Error("Expected error for nonexistent path, got nil")
	}
	if files != nil {
		t.Error("Expected nil files for nonexistent path")
	}
}

func TestLoader_LoadFolder_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	files, err := loader.LoadFolder(tmpDir, "*.md")
	if err != nil {
		t.Fatalf("LoadFolder failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
}

func TestLoader_LoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := "# Test Content\nThis is a test file."
	testPath := filepath.Join(tmpDir, "test.md")
	os.WriteFile(testPath, []byte(testContent), 0644)

	loader := NewLoader(tmpDir)

	file, err := loader.LoadFile(testPath)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if file == nil {
		t.Fatal("LoadFile returned nil")
	}

	if file.Content != testContent {
		t.Errorf("Content mismatch: got %q, want %q", file.Content, testContent)
	}

	if file.Path != testPath {
		t.Errorf("Path mismatch: got %q, want %q", file.Path, testPath)
	}
}

func TestLoader_LoadFile_Nonexistent(t *testing.T) {
	loader := NewLoader("/tmp")

	file, err := loader.LoadFile("/nonexistent/file.md")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
	if file != nil {
		t.Error("Expected nil file for nonexistent path")
	}
}

func TestLoader_FilterFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various file types
	files := map[string]string{
		"doc1.md":    "content",
		"doc2.md":    "content",
		"code1.go":   "package main",
		"code2.go":   "package main",
		"readme.txt": "readme",
	}

	for name, content := range files {
		os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
	}

	loader := NewLoader(tmpDir)

	t.Run("filter markdown files", func(t *testing.T) {
		allFiles, _ := loader.LoadFolder(tmpDir, "")
		filtered := loader.FilterFiles(allFiles, "*.md")

		if len(filtered) != 2 {
			t.Errorf("Expected 2 markdown files, got %d", len(filtered))
		}
	})

	t.Run("filter go files", func(t *testing.T) {
		allFiles, _ := loader.LoadFolder(tmpDir, "")
		filtered := loader.FilterFiles(allFiles, "*.go")

		if len(filtered) != 2 {
			t.Errorf("Expected 2 go files, got %d", len(filtered))
		}
	})

	t.Run("filter with empty pattern", func(t *testing.T) {
		allFiles, _ := loader.LoadFolder(tmpDir, "")
		filtered := loader.FilterFiles(allFiles, "")

		if len(filtered) != len(allFiles) {
			t.Errorf("Empty pattern should return all files")
		}
	})
}

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/test/workspace")

	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}

	if loader.workspacePath != "/test/workspace" {
		t.Errorf("workspacePath = %s, want /test/workspace", loader.workspacePath)
	}
}

func TestLoader_LoadFolder_Recursive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "level1", "level2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "root.md"), []byte("root"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "level1", "mid.md"), []byte("mid"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "level1", "level2", "deep.md"), []byte("deep"), 0644)

	loader := NewLoader(tmpDir)

	files, err := loader.LoadFolder(tmpDir, "*.md")
	if err != nil {
		t.Fatalf("LoadFolder failed: %v", err)
	}

	// Should find all 3 markdown files recursively
	if len(files) != 3 {
		t.Errorf("Expected 3 markdown files recursively, got %d", len(files))
	}
}

func TestLoader_GetFileExtension(t *testing.T) {
	loader := NewLoader("/tmp")

	tests := []struct {
		path     string
		expected string
	}{
		{"file.md", "md"},
		{"file.go", "go"},
		{"file.test.js", "js"},
		{"file", ""},
		{".hidden", ""},
		{"file.tar.gz", "gz"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ext := loader.GetFileExtension(tt.path)
			if ext != tt.expected {
				t.Errorf("GetFileExtension(%q) = %q, want %q", tt.path, ext, tt.expected)
			}
		})
	}
}

func TestLoader_IsTextFile(t *testing.T) {
	loader := NewLoader("/tmp")

	tests := []struct {
		path     string
		expected bool
	}{
		{"file.md", true},
		{"file.go", true},
		{"file.js", true},
		{"file.txt", true},
		{"file.json", true},
		{"file.bin", false},
		{"file.exe", false},
		{"file", false},
		{"file.MD", true}, // case insensitive
		{"file.GO", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := loader.IsTextFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsTextFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestLoader_LoadFolder_NotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "notadir.txt")
	os.WriteFile(filePath, []byte("content"), 0644)

	loader := NewLoader(tmpDir)

	_, err := loader.LoadFolder(filePath, "")
	if err == nil {
		t.Error("Expected error when loading file as folder, got nil")
	}
}

func TestLoader_LoadFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	loader := NewLoader(tmpDir)

	_, err := loader.LoadFile(tmpDir)
	if err == nil {
		t.Error("Expected error when loading directory as file, got nil")
	}
}

func TestLoader_LoadFolder_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("content"), 0644)

	loader := NewLoader(tmpDir)

	// Use relative path
	files, err := loader.LoadFolder("", "*.md")
	if err != nil {
		t.Fatalf("LoadFolder failed: %v", err)
	}

	// Should find the file in workspace
	if len(files) == 0 {
		t.Error("Expected to find files in workspace")
	}
}

func TestLoader_LoadFile_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := "test.md"
	os.WriteFile(filepath.Join(tmpDir, testFile), []byte("content"), 0644)

	loader := NewLoader(tmpDir)

	file, err := loader.LoadFile(testFile)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if file == nil {
		t.Fatal("LoadFile returned nil")
	}

	if file.Content != "content" {
		t.Errorf("Content mismatch: got %q, want %q", file.Content, "content")
	}
}
