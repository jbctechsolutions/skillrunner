// Package sqlite provides SQLite-based sync backend implementation.
package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Connection manages the SQLite database connection.
type Connection struct {
	db       *sql.DB
	dbPath   string
	mu       sync.RWMutex
	isClosed bool
}

// NewConnection creates a new SQLite connection.
// If dbPath is empty, it uses the default location: ~/.skillrunner/skillrunner.db
func NewConnection(dbPath string) (*Connection, error) {
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".skillrunner", "skillrunner.db")
	}

	conn := &Connection{
		dbPath: dbPath,
	}

	return conn, nil
}

// Open opens the database connection and creates the necessary directory structure.
func (c *Connection) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		return fmt.Errorf("database already open")
	}

	// Ensure the directory exists
	dir := filepath.Dir(c.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create database directory: %w", err)
	}

	// Open the database
	db, err := sql.Open("sqlite3", c.dbPath)
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite works best with a single connection
	db.SetMaxIdleConns(1)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("could not ping database: %w", err)
	}

	c.db = db
	c.isClosed = false

	// Run migrations
	if err := c.runMigrations(); err != nil {
		db.Close()
		c.db = nil
		return fmt.Errorf("could not run migrations: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db == nil {
		return nil
	}

	if err := c.db.Close(); err != nil {
		return fmt.Errorf("could not close database: %w", err)
	}

	c.db = nil
	c.isClosed = true
	return nil
}

// DB returns the underlying database connection.
// Returns an error if the connection is not open.
func (c *Connection) DB() (*sql.DB, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	if c.isClosed {
		return nil, fmt.Errorf("database is closed")
	}

	return c.db, nil
}

// Path returns the database file path.
func (c *Connection) Path() string {
	return c.dbPath
}

// IsClosed returns whether the connection is closed.
func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isClosed
}

// Ping tests the database connection.
// Returns an error if the connection is not open or the ping fails.
func (c *Connection) Ping() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.db == nil {
		return fmt.Errorf("database not open")
	}

	if c.isClosed {
		return fmt.Errorf("database is closed")
	}

	return c.db.Ping()
}

// runMigrations runs the database migrations.
func (c *Connection) runMigrations() error {
	// This method assumes the lock is already held
	if c.db == nil {
		return fmt.Errorf("database not open")
	}

	// Apply migrations
	return applyMigrations(c.db)
}
