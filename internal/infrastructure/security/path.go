// Package security provides security utilities for path validation and sanitization.
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// PathValidator provides path validation and sanitization functions.
type PathValidator struct {
	allowedRoots   []string
	criticalPaths  []string
	protectedPaths []string
}

// NewPathValidator creates a new path validator with default settings.
func NewPathValidator() *PathValidator {
	return &PathValidator{
		allowedRoots: []string{"/tmp/", "/var/tmp/", "/projects/", "/work/"},
		criticalPaths: []string{
			"/",
			"/bin",
			"/sbin",
			"/usr",
			"/etc",
			"/var",
			"/tmp",
			"/opt",
			"/lib",
			"/System",
			"/Library",
			"/Applications",
		},
		protectedPaths: []string{}, // populated dynamically
	}
}

// SanitizePathForDeletion validates that a path is safe to delete.
// It ensures the path:
// - Is absolute
// - Does not traverse outside allowed directories
// - Is not a system directory or home directory itself
// Returns an error if the path is unsafe.
func SanitizePathForDeletion(path string) error {
	return NewPathValidator().ValidateForDeletion(path)
}

// ValidateForDeletion validates that a path is safe to delete.
func (v *PathValidator) ValidateForDeletion(path string) error {
	// Must be absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Check for path traversal after cleaning
	if cleanPath != path && strings.Contains(path, "..") {
		return fmt.Errorf("path contains traversal components: %s", path)
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Don't allow deleting the home directory itself
	if cleanPath == homeDir {
		return fmt.Errorf("cannot delete home directory")
	}

	// Don't allow deleting critical system directories
	if slices.Contains(v.criticalPaths, cleanPath) {
		return fmt.Errorf("cannot delete system directory: %s", path)
	}

	// Don't allow deleting paths too close to critical paths
	for _, critical := range v.criticalPaths {
		if strings.HasPrefix(cleanPath, critical+"/") && len(cleanPath) <= len(critical)+5 {
			return fmt.Errorf("cannot delete system directory: %s", path)
		}
	}

	// Don't allow deleting .skillrunner config directory itself
	skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
	if cleanPath == skillrunnerDir {
		return fmt.Errorf("cannot delete skillrunner config directory")
	}

	// Path should be within home directory or a recognizable project directory
	if !strings.HasPrefix(cleanPath, homeDir) {
		// Allow paths in common project locations
		allowed := false
		for _, root := range v.allowedRoots {
			if strings.HasPrefix(cleanPath, root) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path is outside allowed directories: %s", path)
		}
	}

	return nil
}

// AddAllowedRoot adds an allowed root directory for deletion.
func (v *PathValidator) AddAllowedRoot(root string) {
	v.allowedRoots = append(v.allowedRoots, root)
}

// AddProtectedPath adds a path that cannot be deleted.
func (v *PathValidator) AddProtectedPath(path string) {
	v.protectedPaths = append(v.protectedPaths, path)
}
