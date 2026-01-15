package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchEventType represents the type of file system event.
type WatchEventType string

// Watch event types.
const (
	WatchEventCreate WatchEventType = "create"
	WatchEventWrite  WatchEventType = "write"
	WatchEventRemove WatchEventType = "remove"
	WatchEventRename WatchEventType = "rename"
)

// WatchEvent represents a file system event for a skill file.
type WatchEvent struct {
	Path      string
	Type      WatchEventType
	Timestamp time.Time
}

// WatcherConfig holds configuration for the file watcher.
type WatcherConfig struct {
	DebounceDuration time.Duration
	BufferSize       int
}

// DefaultWatcherConfig returns sensible default configuration.
func DefaultWatcherConfig() WatcherConfig {
	return WatcherConfig{
		DebounceDuration: 100 * time.Millisecond,
		BufferSize:       100,
	}
}

// Watcher monitors directories for skill file changes.
// It wraps fsnotify with debouncing and YAML file filtering.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	config    WatcherConfig
	events    chan WatchEvent
	errors    chan error

	// Debouncing state
	pending   map[string]pendingEvent
	pendingMu sync.Mutex

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

// pendingEvent tracks a pending file event for debouncing.
type pendingEvent struct {
	eventType WatchEventType
	timestamp time.Time
}

// NewWatcher creates a new file watcher with the given configuration.
func NewWatcher(cfg WatcherConfig) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 100
	}
	if cfg.DebounceDuration <= 0 {
		cfg.DebounceDuration = 100 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		fsWatcher: fsWatcher,
		config:    cfg,
		events:    make(chan WatchEvent, cfg.BufferSize),
		errors:    make(chan error, cfg.BufferSize),
		pending:   make(map[string]pendingEvent),
		ctx:       ctx,
		cancel:    cancel,
	}

	return w, nil
}

// Watch starts watching the given directories for skill file changes.
// Non-existent directories are skipped without error.
func (w *Watcher) Watch(ctx context.Context, dirs ...string) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	// Add directories to watcher
	for _, dir := range dirs {
		// Skip non-existent directories
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		if err := w.fsWatcher.Add(dir); err != nil {
			return err
		}
	}

	// Start event processing goroutine
	w.wg.Add(1)
	go w.processEvents()

	// Start debounce ticker goroutine
	w.wg.Add(1)
	go w.debounceProcessor()

	return nil
}

// Events returns the channel for receiving watch events.
func (w *Watcher) Events() <-chan WatchEvent {
	return w.events
}

// Errors returns the channel for receiving watcher errors.
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	w.cancel()
	err := w.fsWatcher.Close()
	w.wg.Wait()

	close(w.events)
	close(w.errors)

	return err
}

// processEvents reads from fsnotify and queues events for debouncing.
func (w *Watcher) processEvents() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Filter for YAML files only
			if !isYAMLFile(event.Name) {
				continue
			}

			// Convert event type
			eventType := convertEventType(event.Op)
			if eventType == "" {
				continue
			}

			// Queue for debouncing
			w.pendingMu.Lock()
			w.pending[event.Name] = pendingEvent{
				eventType: eventType,
				timestamp: time.Now(),
			}
			w.pendingMu.Unlock()

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			select {
			case w.errors <- err:
			default:
				// Drop error if channel is full
			}
		}
	}
}

// debounceProcessor periodically checks for stable events and emits them.
func (w *Watcher) debounceProcessor() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.DebounceDuration / 2)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return

		case <-ticker.C:
			w.emitStableEvents()
		}
	}
}

// emitStableEvents checks for events that have been stable long enough and emits them.
func (w *Watcher) emitStableEvents() {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	now := time.Now()
	stable := make([]string, 0)

	for path, pending := range w.pending {
		if now.Sub(pending.timestamp) >= w.config.DebounceDuration {
			stable = append(stable, path)
		}
	}

	for _, path := range stable {
		pending := w.pending[path]
		delete(w.pending, path)

		event := WatchEvent{
			Path:      path,
			Type:      pending.eventType,
			Timestamp: pending.timestamp,
		}

		select {
		case w.events <- event:
		default:
			// Drop event if channel is full
		}
	}
}

// isYAMLFile returns true if the file has a .yaml or .yml extension.
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// convertEventType converts fsnotify event operation to WatchEventType.
func convertEventType(op fsnotify.Op) WatchEventType {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return WatchEventCreate
	case op&fsnotify.Write == fsnotify.Write:
		return WatchEventWrite
	case op&fsnotify.Remove == fsnotify.Remove:
		return WatchEventRemove
	case op&fsnotify.Rename == fsnotify.Rename:
		return WatchEventRename
	default:
		return ""
	}
}
