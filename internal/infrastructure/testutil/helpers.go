// Package testutil provides testing utilities and helpers for the skillrunner project.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory for tests.
// The directory is automatically cleaned up when the test completes.
func TempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// WriteFile writes content to a file in the given directory.
// Returns the full path to the created file.
func WriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
	return path
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// AssertEqual fails the test if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// AssertContains checks if slice contains the given element.
// Fails the test if the element is not found.
func AssertContains[T comparable](t *testing.T, slice []T, elem T) {
	t.Helper()
	for _, v := range slice {
		if v == elem {
			return
		}
	}
	t.Fatalf("slice does not contain %v", elem)
}
