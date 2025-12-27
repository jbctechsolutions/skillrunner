package sqlite

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return db
}

func TestApplyMigrations(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Verify migrations table was created and populated
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count)
	if err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 13 {
		t.Errorf("migrations count = %d, want 13", count)
	}
}

func TestApplyMigrations_Idempotent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Run migrations twice
	if err := applyMigrations(db); err != nil {
		t.Fatalf("first applyMigrations() error = %v", err)
	}
	if err := applyMigrations(db); err != nil {
		t.Fatalf("second applyMigrations() error = %v", err)
	}

	// Verify migrations count is still 13 (not duplicated)
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count)
	if err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	if count != 13 {
		t.Errorf("migrations count = %d after idempotent run, want 13", count)
	}
}

func TestWorkspacesTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert a workspace
	_, err := db.Exec(`
		INSERT INTO workspaces (id, name, repo_path, worktree_path, branch, focus, status, default_backend)
		VALUES ('ws-1', 'Test Workspace', '/path/to/repo', '/path/to/worktree', 'main', 'feature-1', 'active', 'claude')
	`)
	if err != nil {
		t.Fatalf("INSERT workspaces error = %v", err)
	}

	// Verify the workspace was inserted
	var id, name, repoPath, worktreePath, branch, focus, status, defaultBackend string
	err = db.QueryRow(`SELECT id, name, repo_path, worktree_path, branch, focus, status, default_backend FROM workspaces WHERE id = 'ws-1'`).
		Scan(&id, &name, &repoPath, &worktreePath, &branch, &focus, &status, &defaultBackend)
	if err != nil {
		t.Fatalf("SELECT workspaces error = %v", err)
	}

	if id != "ws-1" || name != "Test Workspace" || repoPath != "/path/to/repo" || status != "active" {
		t.Errorf("workspace data mismatch: got id=%s, name=%s, repo_path=%s, status=%s", id, name, repoPath, status)
	}
}

func TestSessionsTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert a workspace first (foreign key)
	_, err := db.Exec(`INSERT INTO workspaces (id, name, repo_path) VALUES ('ws-1', 'Test', '/path')`)
	if err != nil {
		t.Fatalf("INSERT workspaces error = %v", err)
	}

	// Insert a session with all new fields
	_, err = db.Exec(`
		INSERT INTO sessions (id, workspace_id, backend, model, profile, status, tokens_used, tokens_limit, pid, machine_id)
		VALUES ('sess-1', 'ws-1', 'claude', 'claude-3-sonnet', 'balanced', 'active', 1000, 10000, 12345, 'machine-1')
	`)
	if err != nil {
		t.Fatalf("INSERT sessions error = %v", err)
	}

	// Verify the session was inserted with all fields
	var id, workspaceID, backend, model, profile, status, machineID string
	var tokensUsed, tokensLimit, pid int
	err = db.QueryRow(`
		SELECT id, workspace_id, backend, model, profile, status, tokens_used, tokens_limit, pid, machine_id
		FROM sessions WHERE id = 'sess-1'
	`).Scan(&id, &workspaceID, &backend, &model, &profile, &status, &tokensUsed, &tokensLimit, &pid, &machineID)
	if err != nil {
		t.Fatalf("SELECT sessions error = %v", err)
	}

	if backend != "claude" || model != "claude-3-sonnet" || profile != "balanced" {
		t.Errorf("session data mismatch: got backend=%s, model=%s, profile=%s", backend, model, profile)
	}
	if tokensUsed != 1000 || tokensLimit != 10000 || pid != 12345 {
		t.Errorf("session data mismatch: got tokens_used=%d, tokens_limit=%d, pid=%d", tokensUsed, tokensLimit, pid)
	}
}

func TestCheckpointsTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert workspace and session first (foreign keys)
	_, err := db.Exec(`INSERT INTO workspaces (id, name, repo_path) VALUES ('ws-1', 'Test', '/path')`)
	if err != nil {
		t.Fatalf("INSERT workspaces error = %v", err)
	}
	_, err = db.Exec(`INSERT INTO sessions (id, workspace_id) VALUES ('sess-1', 'ws-1')`)
	if err != nil {
		t.Fatalf("INSERT sessions error = %v", err)
	}

	// Insert a checkpoint
	_, err = db.Exec(`
		INSERT INTO checkpoints (id, workspace_id, session_id, summary, details, files_modified, decisions, machine_id)
		VALUES ('cp-1', 'ws-1', 'sess-1', 'Test summary', 'Test details', '["file1.go","file2.go"]', '["decision1"]', 'machine-1')
	`)
	if err != nil {
		t.Fatalf("INSERT checkpoints error = %v", err)
	}

	// Verify
	var summary, details, filesModified, decisions, machineID string
	err = db.QueryRow(`SELECT summary, details, files_modified, decisions, machine_id FROM checkpoints WHERE id = 'cp-1'`).
		Scan(&summary, &details, &filesModified, &decisions, &machineID)
	if err != nil {
		t.Fatalf("SELECT checkpoints error = %v", err)
	}

	if summary != "Test summary" || machineID != "machine-1" {
		t.Errorf("checkpoint data mismatch: got summary=%s, machine_id=%s", summary, machineID)
	}
}

func TestContextItemsTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert a context item
	_, err := db.Exec(`
		INSERT INTO context_items (id, name, type, content, tags, token_estimate)
		VALUES ('ctx-1', 'Test Item', 'file', 'Test content', '["tag1","tag2"]', 100)
	`)
	if err != nil {
		t.Fatalf("INSERT context_items error = %v", err)
	}

	// Verify
	var id, name, itemType, content, tags string
	var tokenEstimate int
	err = db.QueryRow(`SELECT id, name, type, content, tags, token_estimate FROM context_items WHERE id = 'ctx-1'`).
		Scan(&id, &name, &itemType, &content, &tags, &tokenEstimate)
	if err != nil {
		t.Fatalf("SELECT context_items error = %v", err)
	}

	if name != "Test Item" || itemType != "file" || tokenEstimate != 100 {
		t.Errorf("context_item data mismatch: got name=%s, type=%s, token_estimate=%d", name, itemType, tokenEstimate)
	}
}

func TestRulesTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert a rule
	_, err := db.Exec(`
		INSERT INTO rules (id, name, content, scope, is_active)
		VALUES ('rule-1', 'Test Rule', 'Do not use global variables', 'global', 1)
	`)
	if err != nil {
		t.Fatalf("INSERT rules error = %v", err)
	}

	// Verify
	var id, name, content, scope string
	var isActive bool
	err = db.QueryRow(`SELECT id, name, content, scope, is_active FROM rules WHERE id = 'rule-1'`).
		Scan(&id, &name, &content, &scope, &isActive)
	if err != nil {
		t.Fatalf("SELECT rules error = %v", err)
	}

	if name != "Test Rule" || scope != "global" || !isActive {
		t.Errorf("rule data mismatch: got name=%s, scope=%s, is_active=%v", name, scope, isActive)
	}
}

func TestDriftLogTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert workspace first (foreign key)
	_, err := db.Exec(`INSERT INTO workspaces (id, name, repo_path) VALUES ('ws-1', 'Test', '/path')`)
	if err != nil {
		t.Fatalf("INSERT workspaces error = %v", err)
	}
	_, err = db.Exec(`INSERT INTO sessions (id, workspace_id) VALUES ('sess-1', 'ws-1')`)
	if err != nil {
		t.Fatalf("INSERT sessions error = %v", err)
	}

	// Insert a drift log entry with the new schema fields
	_, err = db.Exec(`
		INSERT INTO drift_log (workspace_id, session_id, original_focus, detected_topic, prompt_snippet, action_taken)
		VALUES ('ws-1', 'sess-1', 'implement auth', 'database migration', 'can you help with...', 'redirected')
	`)
	if err != nil {
		t.Fatalf("INSERT drift_log error = %v", err)
	}

	// Verify
	var workspaceID, sessionID, originalFocus, detectedTopic, promptSnippet, actionTaken string
	err = db.QueryRow(`
		SELECT workspace_id, session_id, original_focus, detected_topic, prompt_snippet, action_taken
		FROM drift_log WHERE workspace_id = 'ws-1'
	`).Scan(&workspaceID, &sessionID, &originalFocus, &detectedTopic, &promptSnippet, &actionTaken)
	if err != nil {
		t.Fatalf("SELECT drift_log error = %v", err)
	}

	if originalFocus != "implement auth" || detectedTopic != "database migration" || actionTaken != "redirected" {
		t.Errorf("drift_log data mismatch: got original_focus=%s, detected_topic=%s, action_taken=%s",
			originalFocus, detectedTopic, actionTaken)
	}
}

func TestForeignKeyConstraints(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Try to insert a session without a valid workspace_id
	_, err := db.Exec(`INSERT INTO sessions (id, workspace_id) VALUES ('sess-1', 'nonexistent')`)
	if err == nil {
		t.Error("expected foreign key constraint violation, got nil error")
	}

	// Try to insert a checkpoint without a valid workspace_id or session_id
	_, err = db.Exec(`INSERT INTO checkpoints (id, workspace_id, session_id, summary) VALUES ('cp-1', 'ws-1', 'sess-1', 'summary')`)
	if err == nil {
		t.Error("expected foreign key constraint violation, got nil error")
	}
}

func TestCascadeDelete(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert workspace, session, and checkpoint
	_, err := db.Exec(`INSERT INTO workspaces (id, name, repo_path) VALUES ('ws-1', 'Test', '/path')`)
	if err != nil {
		t.Fatalf("INSERT workspaces error = %v", err)
	}
	_, err = db.Exec(`INSERT INTO sessions (id, workspace_id) VALUES ('sess-1', 'ws-1')`)
	if err != nil {
		t.Fatalf("INSERT sessions error = %v", err)
	}
	_, err = db.Exec(`INSERT INTO checkpoints (id, workspace_id, session_id, summary) VALUES ('cp-1', 'ws-1', 'sess-1', 'summary')`)
	if err != nil {
		t.Fatalf("INSERT checkpoints error = %v", err)
	}

	// Delete workspace - should cascade to sessions and checkpoints
	_, err = db.Exec(`DELETE FROM workspaces WHERE id = 'ws-1'`)
	if err != nil {
		t.Fatalf("DELETE workspaces error = %v", err)
	}

	// Verify session was deleted
	var sessionCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = 'sess-1'`).Scan(&sessionCount)
	if err != nil {
		t.Fatalf("SELECT sessions error = %v", err)
	}
	if sessionCount != 0 {
		t.Errorf("expected session to be deleted, got count=%d", sessionCount)
	}

	// Verify checkpoint was deleted
	var checkpointCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM checkpoints WHERE id = 'cp-1'`).Scan(&checkpointCount)
	if err != nil {
		t.Fatalf("SELECT checkpoints error = %v", err)
	}
	if checkpointCount != 0 {
		t.Errorf("expected checkpoint to be deleted, got count=%d", checkpointCount)
	}
}

func TestIndices(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Verify indices were created
	expectedIndices := []string{
		"idx_workspaces_status",
		"idx_workspaces_last_active",
		"idx_sessions_workspace",
		"idx_sessions_status",
		"idx_sessions_last_active",
		"idx_sessions_backend",
		"idx_checkpoints_workspace",
		"idx_checkpoints_session",
		"idx_checkpoints_created",
		"idx_context_items_name",
		"idx_context_items_type",
		"idx_context_items_last_used",
		"idx_rules_scope",
		"idx_rules_active",
		"idx_drift_log_workspace",
		"idx_drift_log_session",
		"idx_drift_log_created",
	}

	for _, idx := range expectedIndices {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
		if err == sql.ErrNoRows {
			t.Errorf("index %q was not created", idx)
		} else if err != nil {
			t.Errorf("error checking index %q: %v", idx, err)
		}
	}
}

func TestDefaultValues(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	// Insert workspace with minimal fields
	_, err := db.Exec(`INSERT INTO workspaces (id, name, repo_path) VALUES ('ws-1', 'Test', '/path')`)
	if err != nil {
		t.Fatalf("INSERT workspaces error = %v", err)
	}

	// Verify default values
	var status string
	err = db.QueryRow(`SELECT status FROM workspaces WHERE id = 'ws-1'`).Scan(&status)
	if err != nil {
		t.Fatalf("SELECT workspaces error = %v", err)
	}
	if status != "active" {
		t.Errorf("default status = %q, want 'active'", status)
	}

	// Insert session with minimal fields
	_, err = db.Exec(`INSERT INTO sessions (id, workspace_id) VALUES ('sess-1', 'ws-1')`)
	if err != nil {
		t.Fatalf("INSERT sessions error = %v", err)
	}

	// Verify default values
	var sessStatus string
	var tokensUsed, tokensLimit int
	err = db.QueryRow(`SELECT status, tokens_used, tokens_limit FROM sessions WHERE id = 'sess-1'`).
		Scan(&sessStatus, &tokensUsed, &tokensLimit)
	if err != nil {
		t.Fatalf("SELECT sessions error = %v", err)
	}
	if sessStatus != "active" {
		t.Errorf("default session status = %q, want 'active'", sessStatus)
	}
	if tokensUsed != 0 || tokensLimit != 0 {
		t.Errorf("default tokens_used=%d, tokens_limit=%d, want both 0", tokensUsed, tokensLimit)
	}

	// Insert rule with minimal fields
	_, err = db.Exec(`INSERT INTO rules (id, name, content, scope) VALUES ('rule-1', 'Test', 'content', 'global')`)
	if err != nil {
		t.Fatalf("INSERT rules error = %v", err)
	}

	// Verify default values
	var isActive bool
	err = db.QueryRow(`SELECT is_active FROM rules WHERE id = 'rule-1'`).Scan(&isActive)
	if err != nil {
		t.Fatalf("SELECT rules error = %v", err)
	}
	if !isActive {
		t.Error("default is_active = false, want true")
	}
}

func TestIsMigrationApplied(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Create migrations table
	if err := createMigrationsTable(db); err != nil {
		t.Fatalf("createMigrationsTable() error = %v", err)
	}

	// Check migration not applied
	applied, err := isMigrationApplied(db, 1)
	if err != nil {
		t.Fatalf("isMigrationApplied() error = %v", err)
	}
	if applied {
		t.Error("isMigrationApplied() = true for non-existent migration")
	}

	// Record migration
	if err := recordMigration(db, 1, "test_migration"); err != nil {
		t.Fatalf("recordMigration() error = %v", err)
	}

	// Check migration applied
	applied, err = isMigrationApplied(db, 1)
	if err != nil {
		t.Fatalf("isMigrationApplied() error = %v", err)
	}
	if !applied {
		t.Error("isMigrationApplied() = false for applied migration")
	}
}
