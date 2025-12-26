// Package context provides domain entities for workspace and context management.
package context

import (
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// Checkpoint represents a saved state of a skill execution session.
// It captures important context including what was accomplished, files modified,
// and decisions made during the session.
type Checkpoint struct {
	id            string
	workspaceID   string
	sessionID     string
	summary       string
	details       string
	filesModified []string
	decisions     map[string]string
	machineID     string
	createdAt     time.Time
}

// NewCheckpoint creates a new Checkpoint with the required fields.
// Returns an error if validation fails:
//   - id is required
//   - workspaceID is required
//   - sessionID is required
//   - summary is required
func NewCheckpoint(id, workspaceID, sessionID, summary string) (*Checkpoint, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	sessionID = strings.TrimSpace(sessionID)
	summary = strings.TrimSpace(summary)

	if id == "" {
		return nil, errors.New("checkpoint", "checkpoint ID is required")
	}
	if workspaceID == "" {
		return nil, errors.New("checkpoint", "workspace ID is required")
	}
	if sessionID == "" {
		return nil, errors.New("checkpoint", "session ID is required")
	}
	if summary == "" {
		return nil, errors.New("checkpoint", "summary is required")
	}

	return &Checkpoint{
		id:            id,
		workspaceID:   workspaceID,
		sessionID:     sessionID,
		summary:       summary,
		filesModified: make([]string, 0),
		decisions:     make(map[string]string),
		createdAt:     time.Now(),
	}, nil
}

// ID returns the checkpoint's unique identifier.
func (c *Checkpoint) ID() string {
	return c.id
}

// WorkspaceID returns the associated workspace ID.
func (c *Checkpoint) WorkspaceID() string {
	return c.workspaceID
}

// SessionID returns the associated session ID.
func (c *Checkpoint) SessionID() string {
	return c.sessionID
}

// Summary returns the checkpoint summary.
func (c *Checkpoint) Summary() string {
	return c.summary
}

// Details returns detailed information about the checkpoint.
func (c *Checkpoint) Details() string {
	return c.details
}

// FilesModified returns a copy of the files modified list.
func (c *Checkpoint) FilesModified() []string {
	files := make([]string, len(c.filesModified))
	copy(files, c.filesModified)
	return files
}

// Decisions returns a copy of the decisions map.
func (c *Checkpoint) Decisions() map[string]string {
	decisions := make(map[string]string, len(c.decisions))
	for k, v := range c.decisions {
		decisions[k] = v
	}
	return decisions
}

// MachineID returns the machine ID where the checkpoint was created.
func (c *Checkpoint) MachineID() string {
	return c.machineID
}

// CreatedAt returns when the checkpoint was created.
func (c *Checkpoint) CreatedAt() time.Time {
	return c.createdAt
}

// SetDetails sets detailed information about the checkpoint.
func (c *Checkpoint) SetDetails(details string) {
	c.details = details
}

// AddFile adds a file to the list of modified files.
func (c *Checkpoint) AddFile(filePath string) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return
	}

	// Avoid duplicates
	for _, f := range c.filesModified {
		if f == filePath {
			return
		}
	}

	c.filesModified = append(c.filesModified, filePath)
}

// SetFiles replaces the list of modified files.
func (c *Checkpoint) SetFiles(files []string) {
	c.filesModified = make([]string, 0, len(files))
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f != "" {
			c.filesModified = append(c.filesModified, f)
		}
	}
}

// AddDecision records a decision made during the session.
func (c *Checkpoint) AddDecision(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key != "" {
		c.decisions[key] = value
	}
}

// SetDecisions replaces the decisions map.
func (c *Checkpoint) SetDecisions(decisions map[string]string) {
	c.decisions = make(map[string]string, len(decisions))
	for k, v := range decisions {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" {
			c.decisions[k] = v
		}
	}
}

// SetMachineID sets the machine ID for the checkpoint.
func (c *Checkpoint) SetMachineID(machineID string) {
	c.machineID = strings.TrimSpace(machineID)
}

// Validate checks if the Checkpoint is in a valid state.
func (c *Checkpoint) Validate() error {
	if strings.TrimSpace(c.id) == "" {
		return errors.New("checkpoint", "checkpoint ID is required")
	}
	if strings.TrimSpace(c.workspaceID) == "" {
		return errors.New("checkpoint", "workspace ID is required")
	}
	if strings.TrimSpace(c.sessionID) == "" {
		return errors.New("checkpoint", "session ID is required")
	}
	if strings.TrimSpace(c.summary) == "" {
		return errors.New("checkpoint", "summary is required")
	}

	return nil
}
