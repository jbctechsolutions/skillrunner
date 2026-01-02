package testutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTempDir(t *testing.T) {
	dir := TempDir(t)

	// Verify directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("TempDir returned non-existent directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("TempDir did not return a directory")
	}
}

func TestWriteFile(t *testing.T) {
	dir := TempDir(t)
	content := "test content"
	name := "test.txt"

	path := WriteFile(t, dir, name, content)

	// Verify path is correct
	expectedPath := filepath.Join(dir, name)
	if path != expectedPath {
		t.Fatalf("WriteFile returned wrong path: got %s, want %s", path, expectedPath)
	}

	// Verify file contents
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(data) != content {
		t.Fatalf("file content mismatch: got %q, want %q", string(data), content)
	}
}

func TestAssertNoError(t *testing.T) {
	// Should not panic with nil error
	AssertNoError(t, nil)
}

func TestAssertError(t *testing.T) {
	// Should not panic with non-nil error
	AssertError(t, errors.New("test error"))
}

func TestAssertEqual(t *testing.T) {
	// Test with integers
	AssertEqual(t, 42, 42)

	// Test with strings
	AssertEqual(t, "hello", "hello")

	// Test with booleans
	AssertEqual(t, true, true)
}

func TestAssertContains(t *testing.T) {
	// Test with int slice
	intSlice := []int{1, 2, 3, 4, 5}
	AssertContains(t, intSlice, 3)

	// Test with string slice
	strSlice := []string{"a", "b", "c"}
	AssertContains(t, strSlice, "b")
}

func TestAssertContainsEmpty(t *testing.T) {
	// Test with single element slice
	slice := []int{42}
	AssertContains(t, slice, 42)
}
