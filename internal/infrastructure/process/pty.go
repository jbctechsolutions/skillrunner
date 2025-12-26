// Package process provides process and terminal management utilities.
package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// PTYSession represents a PTY-based session (fallback for systems without tmux).
type PTYSession struct {
	ID      string
	Cmd     *exec.Cmd
	PTY     *os.File
	WorkDir string
	mu      sync.Mutex
	running bool
}

// PTYManager manages PTY-based sessions.
type PTYManager struct {
	mu       sync.RWMutex
	sessions map[string]*PTYSession
}

// NewPTYManager creates a new PTY manager.
func NewPTYManager() *PTYManager {
	return &PTYManager{
		sessions: make(map[string]*PTYSession),
	}
}

// CreateSession creates a new PTY session.
func (pm *PTYManager) CreateSession(ctx context.Context, sessionID, command, workDir string) (*PTYSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	if command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if session already exists
	if _, exists := pm.sessions[sessionID]; exists {
		return nil, fmt.Errorf("session already exists: %s", sessionID)
	}

	// Create command
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Start with PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	session := &PTYSession{
		ID:      sessionID,
		Cmd:     cmd,
		PTY:     ptmx,
		WorkDir: workDir,
		running: true,
	}

	pm.sessions[sessionID] = session

	// Monitor process in background
	go pm.monitorSession(session)

	return session, nil
}

// monitorSession monitors a session and cleans up when it exits.
func (pm *PTYManager) monitorSession(session *PTYSession) {
	session.Cmd.Wait()

	session.mu.Lock()
	session.running = false
	session.mu.Unlock()

	// Close PTY
	if session.PTY != nil {
		session.PTY.Close()
	}
}

// GetSession retrieves a session by ID.
func (pm *PTYManager) GetSession(sessionID string) (*PTYSession, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	session, exists := pm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// KillSession terminates a PTY session.
func (pm *PTYManager) KillSession(sessionID string, force bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	session, exists := pm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.running {
		delete(pm.sessions, sessionID)
		return nil
	}

	// Kill process
	if session.Cmd.Process != nil {
		signal := syscall.SIGTERM
		if force {
			signal = syscall.SIGKILL
		}
		if err := session.Cmd.Process.Signal(signal); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Close PTY
	if session.PTY != nil {
		session.PTY.Close()
	}

	delete(pm.sessions, sessionID)
	return nil
}

// SendInput sends input to a PTY session.
func (pm *PTYManager) SendInput(sessionID, input string) error {
	session, err := pm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.running {
		return fmt.Errorf("session not running: %s", sessionID)
	}

	_, err = session.PTY.WriteString(input + "\n")
	if err != nil {
		return fmt.Errorf("failed to send input: %w", err)
	}

	return nil
}

// ReadOutput reads output from a PTY session.
func (pm *PTYManager) ReadOutput(sessionID string, maxBytes int) (string, error) {
	session, err := pm.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.PTY == nil {
		return "", fmt.Errorf("PTY not available")
	}

	buf := make([]byte, maxBytes)
	n, err := session.PTY.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read output: %w", err)
	}

	return string(buf[:n]), nil
}

// IsRunning checks if a session is running.
func (pm *PTYManager) IsRunning(sessionID string) bool {
	session, err := pm.GetSession(sessionID)
	if err != nil {
		return false
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	return session.running
}

// ListSessions returns all session IDs.
func (pm *PTYManager) ListSessions() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	sessions := make([]string, 0, len(pm.sessions))
	for id := range pm.sessions {
		sessions = append(sessions, id)
	}

	return sessions
}

// Resize resizes the PTY.
func (pm *PTYManager) Resize(sessionID string, rows, cols uint16) error {
	session, err := pm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.PTY == nil {
		return fmt.Errorf("PTY not available")
	}

	size := &pty.Winsize{
		Rows: rows,
		Cols: cols,
	}

	if err := pty.Setsize(session.PTY, size); err != nil {
		return fmt.Errorf("failed to resize PTY: %w", err)
	}

	return nil
}

// Close closes all sessions.
func (pm *PTYManager) Close() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for id := range pm.sessions {
		// Kill each session (unlock mutex during operation)
		pm.mu.Unlock()
		pm.KillSession(id, true)
		pm.mu.Lock()
	}

	return nil
}
