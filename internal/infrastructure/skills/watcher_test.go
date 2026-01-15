package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	t.Run("creates watcher with default config", func(t *testing.T) {
		w, err := NewWatcher(DefaultWatcherConfig())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer w.Close()

		if w == nil {
			t.Fatal("expected watcher to be non-nil")
		}
	})

	t.Run("creates watcher with custom debounce duration", func(t *testing.T) {
		cfg := WatcherConfig{
			DebounceDuration: 200 * time.Millisecond,
			BufferSize:       50,
		}
		w, err := NewWatcher(cfg)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer w.Close()

		if w == nil {
			t.Fatal("expected watcher to be non-nil")
		}
	})
}

func TestWatcherConfig(t *testing.T) {
	t.Run("default config has sensible values", func(t *testing.T) {
		cfg := DefaultWatcherConfig()
		if cfg.DebounceDuration != 100*time.Millisecond {
			t.Errorf("expected DebounceDuration 100ms, got %v", cfg.DebounceDuration)
		}
		if cfg.BufferSize != 100 {
			t.Errorf("expected BufferSize 100, got %d", cfg.BufferSize)
		}
	})
}

func TestWatchEventType(t *testing.T) {
	t.Run("event types are defined correctly", func(t *testing.T) {
		if WatchEventCreate != "create" {
			t.Errorf("expected 'create', got %q", WatchEventCreate)
		}
		if WatchEventWrite != "write" {
			t.Errorf("expected 'write', got %q", WatchEventWrite)
		}
		if WatchEventRemove != "remove" {
			t.Errorf("expected 'remove', got %q", WatchEventRemove)
		}
		if WatchEventRename != "rename" {
			t.Errorf("expected 'rename', got %q", WatchEventRename)
		}
	})
}

func TestWatcherWatch(t *testing.T) {
	t.Run("watches directory and detects file creation", func(t *testing.T) {
		dir := t.TempDir()
		w, err := NewWatcher(WatcherConfig{
			DebounceDuration: 50 * time.Millisecond,
			BufferSize:       10,
		})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start watching
		if err := w.Watch(ctx, dir); err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Create a YAML file
		filePath := filepath.Join(dir, "test-skill.yaml")
		if err := os.WriteFile(filePath, []byte("id: test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Wait for event with timeout
		select {
		case event := <-w.Events():
			if event.Path != filePath {
				t.Errorf("expected path %q, got %q", filePath, event.Path)
			}
			// Event type could be Create or Write depending on timing
			if event.Type != WatchEventCreate && event.Type != WatchEventWrite {
				t.Errorf("expected Create or Write event, got %q", event.Type)
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("watches directory and detects file modification", func(t *testing.T) {
		dir := t.TempDir()

		// Create file before starting watcher
		filePath := filepath.Join(dir, "test-skill.yaml")
		if err := os.WriteFile(filePath, []byte("id: test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		w, err := NewWatcher(WatcherConfig{
			DebounceDuration: 50 * time.Millisecond,
			BufferSize:       10,
		})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start watching
		if err := w.Watch(ctx, dir); err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Give watcher time to start
		time.Sleep(50 * time.Millisecond)

		// Modify the file
		if err := os.WriteFile(filePath, []byte("id: test-modified"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		// Wait for event with timeout
		select {
		case event := <-w.Events():
			if event.Path != filePath {
				t.Errorf("expected path %q, got %q", filePath, event.Path)
			}
			if event.Type != WatchEventWrite {
				t.Errorf("expected Write event, got %q", event.Type)
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("watches directory and detects file deletion", func(t *testing.T) {
		dir := t.TempDir()

		// Create file before starting watcher
		filePath := filepath.Join(dir, "test-skill.yaml")
		if err := os.WriteFile(filePath, []byte("id: test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		w, err := NewWatcher(WatcherConfig{
			DebounceDuration: 50 * time.Millisecond,
			BufferSize:       10,
		})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start watching
		if err := w.Watch(ctx, dir); err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Give watcher time to start
		time.Sleep(50 * time.Millisecond)

		// Delete the file
		if err := os.Remove(filePath); err != nil {
			t.Fatalf("failed to remove file: %v", err)
		}

		// Wait for event with timeout
		select {
		case event := <-w.Events():
			if event.Path != filePath {
				t.Errorf("expected path %q, got %q", filePath, event.Path)
			}
			if event.Type != WatchEventRemove {
				t.Errorf("expected Remove event, got %q", event.Type)
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("ignores non-YAML files", func(t *testing.T) {
		dir := t.TempDir()
		w, err := NewWatcher(WatcherConfig{
			DebounceDuration: 50 * time.Millisecond,
			BufferSize:       10,
		})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// Start watching
		if err := w.Watch(ctx, dir); err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Create a non-YAML file
		filePath := filepath.Join(dir, "test.txt")
		if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Wait a bit - should not receive any event
		select {
		case event := <-w.Events():
			t.Errorf("unexpected event for non-YAML file: %+v", event)
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-ctx.Done():
			// Expected - no event should be received
		}
	})

	t.Run("supports both .yaml and .yml extensions", func(t *testing.T) {
		dir := t.TempDir()
		w, err := NewWatcher(WatcherConfig{
			DebounceDuration: 50 * time.Millisecond,
			BufferSize:       10,
		})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start watching
		if err := w.Watch(ctx, dir); err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Create a .yml file
		filePath := filepath.Join(dir, "test-skill.yml")
		if err := os.WriteFile(filePath, []byte("id: test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Wait for event with timeout
		select {
		case event := <-w.Events():
			if event.Path != filePath {
				t.Errorf("expected path %q, got %q", filePath, event.Path)
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("debounces rapid changes", func(t *testing.T) {
		dir := t.TempDir()
		w, err := NewWatcher(WatcherConfig{
			DebounceDuration: 100 * time.Millisecond,
			BufferSize:       10,
		})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start watching
		if err := w.Watch(ctx, dir); err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Create file and rapidly modify it multiple times
		filePath := filepath.Join(dir, "test-skill.yaml")
		for i := 0; i < 5; i++ {
			if err := os.WriteFile(filePath, []byte("id: test-"+string(rune('0'+i))), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			time.Sleep(10 * time.Millisecond) // Rapid writes
		}

		// Should receive only one debounced event
		eventCount := 0
		timeout := time.After(300 * time.Millisecond)
		for {
			select {
			case <-w.Events():
				eventCount++
			case err := <-w.Errors():
				t.Fatalf("unexpected error: %v", err)
			case <-timeout:
				// Allow 1-2 events due to timing variability
				if eventCount == 0 {
					t.Error("expected at least one event")
				}
				if eventCount > 2 {
					t.Errorf("expected 1-2 debounced events, got %d", eventCount)
				}
				return
			}
		}
	})

	t.Run("watches multiple directories", func(t *testing.T) {
		dir1 := t.TempDir()
		dir2 := t.TempDir()

		w, err := NewWatcher(WatcherConfig{
			DebounceDuration: 50 * time.Millisecond,
			BufferSize:       10,
		})
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start watching both directories
		if err := w.Watch(ctx, dir1, dir2); err != nil {
			t.Fatalf("failed to watch directories: %v", err)
		}

		// Create file in second directory
		filePath := filepath.Join(dir2, "test-skill.yaml")
		if err := os.WriteFile(filePath, []byte("id: test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Wait for event with timeout
		select {
		case event := <-w.Events():
			if event.Path != filePath {
				t.Errorf("expected path %q, got %q", filePath, event.Path)
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("skips non-existent directories without error", func(t *testing.T) {
		dir := t.TempDir()
		nonExistent := "/non/existent/path"

		w, err := NewWatcher(DefaultWatcherConfig())
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}
		defer w.Close()

		ctx := context.Background()

		// Should succeed - skipping non-existent directory
		if err := w.Watch(ctx, dir, nonExistent); err != nil {
			t.Fatalf("expected no error when skipping non-existent dir, got %v", err)
		}
	})
}

func TestWatcherClose(t *testing.T) {
	t.Run("close stops watching", func(t *testing.T) {
		dir := t.TempDir()
		w, err := NewWatcher(DefaultWatcherConfig())
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		ctx := context.Background()
		if err := w.Watch(ctx, dir); err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Close should not error
		if err := w.Close(); err != nil {
			t.Errorf("expected no error from Close, got %v", err)
		}
	})

	t.Run("close can be called multiple times", func(t *testing.T) {
		w, err := NewWatcher(DefaultWatcherConfig())
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// Multiple closes should not panic
		_ = w.Close()
		_ = w.Close()
	})
}
