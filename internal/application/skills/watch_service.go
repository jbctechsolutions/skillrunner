package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/logging"
	infraSkills "github.com/jbctechsolutions/skillrunner/internal/infrastructure/skills"
)

// WatchServiceConfig holds configuration for the WatchService.
type WatchServiceConfig struct {
	// UserDir is the user skills directory (~/.skillrunner/skills/)
	UserDir string
	// ProjectDir is the project skills directory (.skillrunner/skills/)
	ProjectDir string
	// DebounceDuration is the debounce window for file changes
	DebounceDuration time.Duration
	// OnReload is called when a skill is reloaded (optional)
	OnReload func(event SkillReloadEvent)
}

// SkillReloadEvent represents a skill reload event.
type SkillReloadEvent struct {
	SkillID   string
	FilePath  string
	EventType string // "created", "modified", "deleted", "failed"
	Error     error
}

// WatchService coordinates file watching with the skill registry.
// It monitors user and project directories for skill file changes
// and updates the registry accordingly.
type WatchService struct {
	registry *Registry
	loader   *infraSkills.Loader
	watcher  *infraSkills.Watcher
	logger   *logging.Logger
	config   WatchServiceConfig

	// State
	running bool
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewWatchService creates a new WatchService.
func NewWatchService(cfg WatchServiceConfig, registry *Registry, loader *infraSkills.Loader, logger *logging.Logger) (*WatchService, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is required")
	}
	if loader == nil {
		return nil, fmt.Errorf("loader is required")
	}

	// Set defaults
	if cfg.DebounceDuration <= 0 {
		cfg.DebounceDuration = 100 * time.Millisecond
	}

	watcherCfg := infraSkills.WatcherConfig{
		DebounceDuration: cfg.DebounceDuration,
		BufferSize:       100,
	}

	watcher, err := infraSkills.NewWatcher(watcherCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	// Use no-op logger if none provided
	if logger == nil {
		logger = logging.New(logging.Config{
			Level:  logging.LevelInfo,
			Format: logging.FormatText,
		})
	}

	return &WatchService{
		registry: registry,
		loader:   loader,
		watcher:  watcher,
		logger:   logger,
		config:   cfg,
	}, nil
}

// Start begins watching for skill file changes.
func (s *WatchService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil // Already running
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Collect directories to watch
	var dirs []string
	if s.config.UserDir != "" {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(s.config.UserDir, 0755); err == nil {
			dirs = append(dirs, s.config.UserDir)
		}
	}
	if s.config.ProjectDir != "" {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(s.config.ProjectDir, 0755); err == nil {
			dirs = append(dirs, s.config.ProjectDir)
		}
	}

	if len(dirs) == 0 {
		return fmt.Errorf("no directories to watch")
	}

	// Load existing skills from watched directories
	s.loadExistingSkills(dirs)

	// Start the watcher
	if err := s.watcher.Watch(s.ctx, dirs...); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Start event processing goroutine
	s.wg.Add(1)
	go s.processEvents()

	s.running = true
	s.logger.Info("skill watch service started", "directories", dirs)

	return nil
}

// Stop stops watching for skill file changes.
func (s *WatchService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil // Not running
	}

	s.cancel()
	if err := s.watcher.Close(); err != nil {
		s.logger.Warn("error closing watcher", "error", err)
	}
	s.wg.Wait()

	s.running = false
	s.logger.Info("skill watch service stopped")

	return nil
}

// IsRunning returns true if the service is currently running.
func (s *WatchService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// loadExistingSkills loads skills from the watched directories on startup.
func (s *WatchService) loadExistingSkills(dirs []string) {
	for _, dir := range dirs {
		sourceType := s.getSourceType(dir)

		// Walk the directory and load all YAML files
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if d.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}

			// Load and register the skill
			s.loadAndRegisterSkill(path, sourceType)
			return nil
		})

		if err != nil {
			s.logger.Warn("error walking directory", "dir", dir, "error", err)
		}
	}
}

// processEvents handles watch events from the watcher.
func (s *WatchService) processEvents() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return

		case event, ok := <-s.watcher.Events():
			if !ok {
				return
			}
			s.handleEvent(event)

		case err, ok := <-s.watcher.Errors():
			if !ok {
				return
			}
			s.logger.Error("watcher error", "error", err)
		}
	}
}

// handleEvent processes a single file system event.
func (s *WatchService) handleEvent(event infraSkills.WatchEvent) {
	sourceType := s.getSourceType(event.Path)

	switch event.Type {
	case infraSkills.WatchEventCreate, infraSkills.WatchEventWrite:
		s.handleCreateOrModify(event.Path, sourceType)

	case infraSkills.WatchEventRemove, infraSkills.WatchEventRename:
		s.handleRemove(event.Path)
	}
}

// handleCreateOrModify handles file creation or modification.
func (s *WatchService) handleCreateOrModify(path string, sourceType skill.SourceType) {
	loadedSkill := s.loadAndRegisterSkill(path, sourceType)

	if s.config.OnReload != nil {
		eventType := "modified"
		if loadedSkill != nil {
			// Check if this was a new skill
			source := s.registry.GetSource(loadedSkill.ID())
			if source != nil && time.Since(source.LoadedAt()) < 500*time.Millisecond {
				eventType = "created"
			}
		}

		event := SkillReloadEvent{
			FilePath:  path,
			EventType: eventType,
		}
		if loadedSkill != nil {
			event.SkillID = loadedSkill.ID()
		}
		s.config.OnReload(event)
	}
}

// handleRemove handles file removal.
func (s *WatchService) handleRemove(path string) {
	skillID, found := s.registry.UnregisterByPath(path)

	if found {
		s.logger.Info("skill removed", "skill_id", skillID, "path", path)

		if s.config.OnReload != nil {
			s.config.OnReload(SkillReloadEvent{
				SkillID:   skillID,
				FilePath:  path,
				EventType: "deleted",
			})
		}
	}
}

// loadAndRegisterSkill loads a skill from a file and registers it.
func (s *WatchService) loadAndRegisterSkill(path string, sourceType skill.SourceType) *skill.Skill {
	loadedSkill, err := s.loader.LoadSkill(path)
	if err != nil {
		s.logger.Warn("failed to load skill",
			"path", path,
			"error", err,
		)

		if s.config.OnReload != nil {
			s.config.OnReload(SkillReloadEvent{
				FilePath:  path,
				EventType: "failed",
				Error:     err,
			})
		}
		return nil
	}

	// Register with source tracking
	if err := s.registry.RegisterWithSource(loadedSkill, path, sourceType); err != nil {
		s.logger.Warn("failed to register skill",
			"skill_id", loadedSkill.ID(),
			"path", path,
			"error", err,
		)
		return nil
	}

	s.logger.Info("skill loaded",
		"skill_id", loadedSkill.ID(),
		"skill_name", loadedSkill.Name(),
		"path", path,
		"source", sourceType,
	)

	return loadedSkill
}

// getSourceType determines the source type based on the file path.
func (s *WatchService) getSourceType(path string) skill.SourceType {
	// Check if path is under project directory
	if s.config.ProjectDir != "" {
		absPath, err1 := filepath.Abs(path)
		absProject, err2 := filepath.Abs(s.config.ProjectDir)
		if err1 == nil && err2 == nil && strings.HasPrefix(absPath, absProject) {
			return skill.SourceProject
		}
	}

	// Default to user source
	return skill.SourceUser
}
