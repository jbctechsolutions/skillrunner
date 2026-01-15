package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/logging"
	infraSkills "github.com/jbctechsolutions/skillrunner/internal/infrastructure/skills"
)

func TestNewWatchService(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
	})

	t.Run("creates watch service with valid config", func(t *testing.T) {
		cfg := WatchServiceConfig{
			UserDir:          t.TempDir(),
			DebounceDuration: 100 * time.Millisecond,
		}

		service, err := NewWatchService(cfg, registry, loader, logger)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer service.Stop()

		if service == nil {
			t.Fatal("expected non-nil service")
		}
	})

	t.Run("returns error for nil registry", func(t *testing.T) {
		cfg := WatchServiceConfig{
			UserDir: t.TempDir(),
		}

		_, err := NewWatchService(cfg, nil, loader, logger)
		if err == nil {
			t.Error("expected error for nil registry")
		}
	})

	t.Run("returns error for nil loader", func(t *testing.T) {
		cfg := WatchServiceConfig{
			UserDir: t.TempDir(),
		}

		_, err := NewWatchService(cfg, registry, nil, logger)
		if err == nil {
			t.Error("expected error for nil loader")
		}
	})
}

func TestWatchServiceStartStop(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
	})
	userDir := t.TempDir()

	cfg := WatchServiceConfig{
		UserDir:          userDir,
		DebounceDuration: 50 * time.Millisecond,
	}

	service, err := NewWatchService(cfg, registry, loader, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	t.Run("starts successfully", func(t *testing.T) {
		ctx := context.Background()
		err := service.Start(ctx)
		if err != nil {
			t.Fatalf("failed to start service: %v", err)
		}

		if !service.IsRunning() {
			t.Error("expected service to be running")
		}
	})

	t.Run("stops successfully", func(t *testing.T) {
		err := service.Stop()
		if err != nil {
			t.Fatalf("failed to stop service: %v", err)
		}

		if service.IsRunning() {
			t.Error("expected service to not be running")
		}
	})

	t.Run("can be restarted", func(t *testing.T) {
		ctx := context.Background()
		err := service.Start(ctx)
		if err != nil {
			t.Fatalf("failed to restart service: %v", err)
		}
		defer service.Stop()

		if !service.IsRunning() {
			t.Error("expected service to be running after restart")
		}
	})
}

func TestWatchServiceFileCreation(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
	})
	userDir := t.TempDir()

	cfg := WatchServiceConfig{
		UserDir:          userDir,
		DebounceDuration: 50 * time.Millisecond,
	}

	service, err := NewWatchService(cfg, registry, loader, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	defer service.Stop()

	t.Run("detects new skill file", func(t *testing.T) {
		skillYAML := `
id: new-skill
name: New Skill
version: "1.0.0"
phases:
  - id: main
    name: Main Phase
    prompt_template: "Process: {{.input}}"
`
		filePath := filepath.Join(userDir, "new-skill.yaml")
		if err := os.WriteFile(filePath, []byte(skillYAML), 0644); err != nil {
			t.Fatalf("failed to write skill file: %v", err)
		}

		// Wait for detection and processing
		time.Sleep(200 * time.Millisecond)

		// Verify skill was registered
		s := registry.GetSkill("new-skill")
		if s == nil {
			t.Fatal("expected new skill to be registered")
		}
		if s.Name() != "New Skill" {
			t.Errorf("expected name 'New Skill', got %q", s.Name())
		}

		// Verify source tracking
		source := registry.GetSource("new-skill")
		if source == nil {
			t.Fatal("expected source to be tracked")
		}
		if source.Source() != skill.SourceUser {
			t.Errorf("expected source type User, got %q", source.Source())
		}
	})
}

func TestWatchServiceFileModification(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
	})
	userDir := t.TempDir()

	// Create initial skill file before starting service
	initialYAML := `
id: modify-skill
name: Initial Name
version: "1.0.0"
phases:
  - id: main
    name: Main Phase
    prompt_template: "Initial: {{.input}}"
`
	filePath := filepath.Join(userDir, "modify-skill.yaml")
	if err := os.WriteFile(filePath, []byte(initialYAML), 0644); err != nil {
		t.Fatalf("failed to write initial skill file: %v", err)
	}

	cfg := WatchServiceConfig{
		UserDir:          userDir,
		DebounceDuration: 50 * time.Millisecond,
	}

	service, err := NewWatchService(cfg, registry, loader, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	defer service.Stop()

	// Wait for initial file to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify initial skill
	s := registry.GetSkill("modify-skill")
	if s == nil {
		t.Fatal("expected initial skill to be registered")
	}

	t.Run("detects modified skill file", func(t *testing.T) {
		modifiedYAML := `
id: modify-skill
name: Modified Name
version: "2.0.0"
phases:
  - id: main
    name: Main Phase
    prompt_template: "Modified: {{.input}}"
`
		if err := os.WriteFile(filePath, []byte(modifiedYAML), 0644); err != nil {
			t.Fatalf("failed to modify skill file: %v", err)
		}

		// Wait for detection and processing
		time.Sleep(200 * time.Millisecond)

		// Verify skill was updated
		s := registry.GetSkill("modify-skill")
		if s == nil {
			t.Fatal("expected skill to still exist")
		}
		if s.Name() != "Modified Name" {
			t.Errorf("expected name 'Modified Name', got %q", s.Name())
		}
		if s.Version() != "2.0.0" {
			t.Errorf("expected version '2.0.0', got %q", s.Version())
		}
	})
}

func TestWatchServiceFileDeletion(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
	})
	userDir := t.TempDir()

	// Create skill file before starting service
	skillYAML := `
id: delete-skill
name: To Delete
version: "1.0.0"
phases:
  - id: main
    name: Main Phase
    prompt_template: "Delete: {{.input}}"
`
	filePath := filepath.Join(userDir, "delete-skill.yaml")
	if err := os.WriteFile(filePath, []byte(skillYAML), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	cfg := WatchServiceConfig{
		UserDir:          userDir,
		DebounceDuration: 50 * time.Millisecond,
	}

	service, err := NewWatchService(cfg, registry, loader, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	defer service.Stop()

	// Wait for initial file to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify skill exists
	if registry.GetSkill("delete-skill") == nil {
		t.Fatal("expected skill to exist before deletion")
	}

	t.Run("detects deleted skill file", func(t *testing.T) {
		if err := os.Remove(filePath); err != nil {
			t.Fatalf("failed to delete skill file: %v", err)
		}

		// Wait for detection and processing
		time.Sleep(200 * time.Millisecond)

		// Verify skill was removed
		s := registry.GetSkill("delete-skill")
		if s != nil {
			t.Error("expected skill to be removed after file deletion")
		}
	})
}

func TestWatchServiceInvalidYAML(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
	})
	userDir := t.TempDir()

	cfg := WatchServiceConfig{
		UserDir:          userDir,
		DebounceDuration: 50 * time.Millisecond,
	}

	service, err := NewWatchService(cfg, registry, loader, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	defer service.Stop()

	t.Run("handles invalid YAML without crashing", func(t *testing.T) {
		invalidYAML := `
id:
name: No ID
invalid yaml structure
`
		filePath := filepath.Join(userDir, "invalid.yaml")
		if err := os.WriteFile(filePath, []byte(invalidYAML), 0644); err != nil {
			t.Fatalf("failed to write invalid file: %v", err)
		}

		// Wait - should not crash
		time.Sleep(200 * time.Millisecond)

		// Service should still be running
		if !service.IsRunning() {
			t.Error("expected service to still be running after invalid YAML")
		}

		// No skill should be registered
		if registry.Count() > 0 {
			t.Errorf("expected no skills, got %d", registry.Count())
		}
	})
}

func TestWatchServiceProjectDirectory(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
	})
	projectDir := t.TempDir()

	cfg := WatchServiceConfig{
		ProjectDir:       projectDir,
		DebounceDuration: 50 * time.Millisecond,
	}

	service, err := NewWatchService(cfg, registry, loader, logger)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	defer service.Stop()

	t.Run("detects skill in project directory with project source type", func(t *testing.T) {
		skillYAML := `
id: project-skill
name: Project Skill
version: "1.0.0"
phases:
  - id: main
    name: Main Phase
    prompt_template: "Project: {{.input}}"
`
		filePath := filepath.Join(projectDir, "project-skill.yaml")
		if err := os.WriteFile(filePath, []byte(skillYAML), 0644); err != nil {
			t.Fatalf("failed to write skill file: %v", err)
		}

		// Wait for detection and processing
		time.Sleep(200 * time.Millisecond)

		// Verify skill was registered
		s := registry.GetSkill("project-skill")
		if s == nil {
			t.Fatal("expected project skill to be registered")
		}

		// Verify source type
		source := registry.GetSource("project-skill")
		if source == nil {
			t.Fatal("expected source to be tracked")
		}
		if source.Source() != skill.SourceProject {
			t.Errorf("expected source type Project, got %q", source.Source())
		}
	})
}
