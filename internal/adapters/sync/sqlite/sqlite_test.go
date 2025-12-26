package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestNewConnection(t *testing.T) {
	t.Run("creates connection with custom path", func(t *testing.T) {
		conn, err := NewConnection("/tmp/test.db")
		if err != nil {
			t.Fatalf("NewConnection() error = %v", err)
		}
		if conn.Path() != "/tmp/test.db" {
			t.Errorf("Path() = %q, want %q", conn.Path(), "/tmp/test.db")
		}
	})

	t.Run("creates connection with default path", func(t *testing.T) {
		conn, err := NewConnection("")
		if err != nil {
			t.Fatalf("NewConnection() error = %v", err)
		}
		homeDir, _ := os.UserHomeDir()
		expectedPath := filepath.Join(homeDir, ".skillrunner", "skillrunner.db")
		if conn.Path() != expectedPath {
			t.Errorf("Path() = %q, want %q", conn.Path(), expectedPath)
		}
	})
}

func TestConnection_OpenClose(t *testing.T) {
	// Use in-memory database for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	conn, err := NewConnection(dbPath)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}

	t.Run("open creates database and runs migrations", func(t *testing.T) {
		if err := conn.Open(); err != nil {
			t.Fatalf("Open() error = %v", err)
		}

		// Verify database file exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("Open() did not create database file")
		}

		// Verify DB() returns connection
		db, err := conn.DB()
		if err != nil {
			t.Fatalf("DB() error = %v", err)
		}
		if db == nil {
			t.Error("DB() returned nil")
		}
	})

	t.Run("open on already open connection returns error", func(t *testing.T) {
		err := conn.Open()
		if err == nil {
			t.Error("Open() on already open connection should return error")
		}
	})

	t.Run("IsClosed returns false when open", func(t *testing.T) {
		if conn.IsClosed() {
			t.Error("IsClosed() = true, want false")
		}
	})

	t.Run("close closes the connection", func(t *testing.T) {
		if err := conn.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	t.Run("IsClosed returns true after close", func(t *testing.T) {
		if !conn.IsClosed() {
			t.Error("IsClosed() = false, want true")
		}
	})

	t.Run("DB returns error when closed", func(t *testing.T) {
		_, err := conn.DB()
		if err == nil {
			t.Error("DB() on closed connection should return error")
		}
	})

	t.Run("close on closed connection is idempotent", func(t *testing.T) {
		if err := conn.Close(); err != nil {
			t.Errorf("Close() on closed connection error = %v", err)
		}
	})
}

func TestConnection_Ping(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	conn, err := NewConnection(dbPath)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}

	t.Run("ping returns error when not open", func(t *testing.T) {
		err := conn.Ping()
		if err == nil {
			t.Error("Ping() on unopened connection should return error")
		}
	})

	if err := conn.Open(); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	t.Run("ping succeeds when open", func(t *testing.T) {
		if err := conn.Ping(); err != nil {
			t.Errorf("Ping() error = %v", err)
		}
	})
}

func TestConnection_InMemory(t *testing.T) {
	// Test with in-memory database using file::memory:?cache=shared
	conn, err := NewConnection(":memory:")
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}

	if err := conn.Open(); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	db, err := conn.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}

	// Verify tables were created
	tables := []string{"workspaces", "sessions", "checkpoints", "context_items", "rules", "drift_log", "migrations"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("table %q was not created", table)
		} else if err != nil {
			t.Errorf("error checking table %q: %v", table, err)
		}
	}
}

func TestConnection_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	conn, err := NewConnection(dbPath)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}

	if err := conn.Open(); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := conn.DB()
			if err != nil {
				t.Errorf("concurrent DB() error = %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
