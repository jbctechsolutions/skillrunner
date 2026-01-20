package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileDetector_DetectFiles_Deduplication(t *testing.T) {
	// Create temp dir with test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create detector with temp dir as working directory
	detector := &FileDetector{
		workingDir: tmpDir,
		maxSize:    1024 * 1024,
	}

	// Test case: Same file referenced with different formats
	input := "Check ./test.go and also test.go"
	refs := detector.DetectFiles(input)

	// Should only return 1 file (deduplicated by absolute path)
	if len(refs) != 1 {
		t.Errorf("expected 1 file after deduplication, got %d", len(refs))
		for i, ref := range refs {
			t.Logf("  [%d] %s (original: %s)", i, ref.Path, ref.Original)
		}
	}

	// Verify it's the correct file
	if len(refs) > 0 && !strings.Contains(refs[0].Path, "test.go") {
		t.Errorf("expected test.go, got %s", refs[0].Path)
	}
}

func TestFileDetector_DetectFiles_MultipleFiles(t *testing.T) {
	// Create temp dir with test files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.go")
	file2 := filepath.Join(tmpDir, "file2.go")
	if err := os.WriteFile(file1, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	detector := &FileDetector{
		workingDir: tmpDir,
		maxSize:    1024 * 1024,
	}

	input := "Compare file1.go and file2.go"
	refs := detector.DetectFiles(input)

	// Should return 2 distinct files
	if len(refs) != 2 {
		t.Errorf("expected 2 files, got %d", len(refs))
	}

	// Verify both files are different
	if len(refs) == 2 && refs[0].Path == refs[1].Path {
		t.Errorf("files should be different but got same path: %s", refs[0].Path)
	}
}

func TestFileDetector_DetectFiles_NonexistentFile(t *testing.T) {
	// Create temp dir with only one file
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.go")
	if err := os.WriteFile(existingFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	detector := &FileDetector{
		workingDir: tmpDir,
		maxSize:    1024 * 1024,
	}

	input := "Review nonexistent.go and existing.go"
	refs := detector.DetectFiles(input)

	// Should skip nonexistent and only return existing.go
	if len(refs) != 1 {
		t.Errorf("expected 1 file, got %d", len(refs))
	}

	if len(refs) > 0 && !strings.Contains(refs[0].Path, "existing.go") {
		t.Errorf("expected existing.go, got %s", refs[0].Path)
	}
}
