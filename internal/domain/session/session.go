// Package session defines domain models for AI coding assistant sessions.
package session

import (
	"time"
)

// Status represents the current state of a session.
type Status string

const (
	StatusActive    Status = "active"    // Session is running
	StatusIdle      Status = "idle"      // Session is running but idle
	StatusDetached  Status = "detached"  // Session is running in background
	StatusCompleted Status = "completed" // Session finished successfully
	StatusFailed    Status = "failed"    // Session encountered an error
	StatusKilled    Status = "killed"    // Session was forcefully terminated
)

// Session represents an active or historical AI coding assistant session.
type Session struct {
	ID          string            // Unique session identifier
	WorkspaceID string            // Associated workspace ID
	Backend     string            // Backend name (aider, claude, opencode)
	Model       string            // LLM model being used
	Status      Status            // Current session status
	StartedAt   time.Time         // When the session was started
	EndedAt     *time.Time        // When the session ended (nil if active)
	MachineID   string            // Machine identifier for remote sessions
	ProcessID   int               // OS process ID (if available)
	TmuxSession string            // Tmux session name (if applicable)
	Metadata    map[string]string // Additional session metadata
	TokenUsage  *TokenUsage       // Token usage statistics
	Context     *Context          // Session context and enrichment
}

// TokenUsage tracks token consumption for a session.
type TokenUsage struct {
	InputTokens   int       // Total input tokens consumed
	OutputTokens  int       // Total output tokens generated
	TotalTokens   int       // Total tokens (input + output)
	EstimatedCost float64   // Estimated cost in USD
	LastUpdated   time.Time // When usage was last updated
}

// Context holds contextual information for a session.
type Context struct {
	WorkingDirectory string            // Current working directory
	GitBranch        string            // Active git branch
	Files            []string          // Files in context
	Documentation    []DocReference    // Documentation references
	Protocols        []ProtocolBinding // Applied governance protocols
	Prepared         []PreparedContext // Pre-processed context blocks
}

// DocReference represents a documentation reference.
type DocReference struct {
	Type      string // file, url, snippet
	Path      string // File path, URL, or identifier
	Title     string // Human-readable title
	Section   string // Specific section within the document
	Relevance string // Why this doc is relevant
}

// ProtocolBinding represents a governance protocol applied to a session.
type ProtocolBinding struct {
	ProtocolID string                 // Reference to the protocol in registry
	Scope      string                 // pre_execution, post_execution, continuous, all
	Priority   int                    // Higher priority protocols enforced first
	Version    string                 // Optional version constraint
	Parameters map[string]interface{} // Protocol-specific parameters
	Overrides  map[string]interface{} // Override default protocol settings
}

// PreparedContext represents a pre-processed context block.
type PreparedContext struct {
	Key           string // Unique identifier for this context block
	Content       string // The actual context content
	Source        string // Where this context came from
	Priority      string // required, recommended, optional
	TokenEstimate int    // Estimated token count
}

// IsActive returns true if the session is currently active.
func (s *Session) IsActive() bool {
	return s.Status == StatusActive || s.Status == StatusIdle || s.Status == StatusDetached
}

// Duration returns the total duration of the session.
func (s *Session) Duration() time.Duration {
	if s.EndedAt == nil {
		return time.Since(s.StartedAt)
	}
	return s.EndedAt.Sub(s.StartedAt)
}

// Filter defines criteria for querying sessions.
type Filter struct {
	WorkspaceID string   // Filter by workspace
	Backend     string   // Filter by backend
	Status      []Status // Filter by status (empty for all)
	MachineID   string   // Filter by machine (empty for current machine)
	Limit       int      // Maximum results (0 for all)
}

// StartOptions contains parameters for starting a new session.
type StartOptions struct {
	WorkspaceID   string            // Workspace to run in
	Backend       string            // Backend to use (aider, claude, opencode)
	Model         string            // LLM model to use
	Profile       string            // Profile name (if supported by backend)
	Background    bool              // Run in background (detached)
	Task          string            // Initial task/prompt
	ContextFiles  []string          // Files to include in context
	Documentation []DocReference    // Documentation to include
	Protocols     []ProtocolBinding // Governance protocols to apply
	Config        BackendConfig     // Backend-specific configuration
}

// BackendConfig holds backend-specific configuration.
type BackendConfig struct {
	// Aider-specific
	AiderEditFormat   string // edit format: diff, whole, udiff
	AiderAutoCommit   bool   // auto-commit changes
	AiderDirtyCommits bool   // allow dirty working directory
	AiderMapTokens    int    // tokens for repository map
	AiderCachePrompts bool   // enable prompt caching
	AiderWeakModel    string // weak model for simple tasks
	AiderTestCmd      string // test command to run
	AiderLintCmd      string // lint command to run
	AiderAutoTest     bool   // auto-run tests
	AiderAutoLint     bool   // auto-run lint

	// Claude-specific
	ClaudeRulesFile   string // path to CLAUDE.md
	ClaudeHooksDir    string // directory for hooks
	ClaudeSessionHook string // session start hook script
	ClaudePreCompact  string // pre-compaction hook script

	// OpenCode-specific
	OpenCodeWorkspace string // workspace configuration
	OpenCodeMode      string // mode: chat, edit, review

	// Common
	Timeout     time.Duration     // session timeout
	MaxTokens   int               // max tokens per request
	Environment map[string]string // environment variables
}

// InjectContent represents content to inject into a session.
type InjectContent struct {
	Type    string   // prompt, file, item
	Content string   // The content to inject
	Files   []string // Files to inject (for file type)
}
