package sqlite

import (
	"database/sql"
	"fmt"
)

// applyMigrations applies all database migrations in order.
func applyMigrations(db *sql.DB) error {
	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("could not enable foreign keys: %w", err)
	}

	// Create migrations table
	if err := createMigrationsTable(db); err != nil {
		return err
	}

	// Apply each migration
	migrations := []struct {
		version int
		name    string
		sql     string
	}{
		{1, "create_workspaces_table", createWorkspacesTable},
		{2, "create_sessions_table", createSessionsTable},
		{3, "create_checkpoints_table", createCheckpointsTable},
		{4, "create_context_items_table", createContextItemsTable},
		{5, "create_rules_table", createRulesTable},
		{6, "create_drift_log_table", createDriftLogTable},
		{7, "create_indices", createIndices},
		{8, "create_response_cache_table", createResponseCacheTable},
		{9, "create_cache_stats_table", createCacheStatsTable},
		{10, "create_cache_indices", createCacheIndices},
		// Wave 11: Observability
		{11, "create_execution_records_table", createExecutionRecordsTable},
		{12, "create_phase_execution_records_table", createPhaseExecutionRecordsTable},
		{13, "create_metrics_indices", createMetricsIndices},
		// Crash Recovery: Workflow checkpoints
		{14, "create_workflow_checkpoints_table", createWorkflowCheckpointsTable},
		{15, "create_workflow_checkpoint_indices", createWorkflowCheckpointIndices},
	}

	for _, m := range migrations {
		applied, err := isMigrationApplied(db, m.version)
		if err != nil {
			return fmt.Errorf("could not check migration %d: %w", m.version, err)
		}

		if applied {
			continue
		}

		// Apply migration
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("could not apply migration %d (%s): %w", m.version, m.name, err)
		}

		// Record migration
		if err := recordMigration(db, m.version, m.name); err != nil {
			return fmt.Errorf("could not record migration %d: %w", m.version, err)
		}
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table.
func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// isMigrationApplied checks if a migration has been applied.
func isMigrationApplied(db *sql.DB, version int) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// recordMigration records that a migration has been applied.
func recordMigration(db *sql.DB, version int, name string) error {
	_, err := db.Exec("INSERT INTO migrations (version, name) VALUES (?, ?)", version, name)
	return err
}

// Migration SQL statements

const createWorkspacesTable = `
CREATE TABLE workspaces (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	repo_path TEXT NOT NULL,
	worktree_path TEXT,
	branch TEXT,
	focus TEXT,
	status TEXT NOT NULL DEFAULT 'active',
	default_backend TEXT,
	last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

const createSessionsTable = `
CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL,
	backend TEXT,
	model TEXT,
	profile TEXT,
	status TEXT NOT NULL DEFAULT 'active',
	tokens_used INTEGER DEFAULT 0,
	tokens_limit INTEGER DEFAULT 0,
	pid INTEGER,
	machine_id TEXT,
	started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
`

const createCheckpointsTable = `
CREATE TABLE checkpoints (
	id TEXT PRIMARY KEY,
	workspace_id TEXT NOT NULL,
	session_id TEXT NOT NULL,
	summary TEXT NOT NULL,
	details TEXT,
	files_modified TEXT,
	decisions TEXT,
	machine_id TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
`

const createContextItemsTable = `
CREATE TABLE context_items (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	content TEXT,
	tags TEXT,
	token_estimate INTEGER DEFAULT 0,
	last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

const createRulesTable = `
CREATE TABLE rules (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	content TEXT NOT NULL,
	scope TEXT NOT NULL,
	is_active BOOLEAN DEFAULT 1,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

const createDriftLogTable = `
CREATE TABLE drift_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	workspace_id TEXT NOT NULL,
	session_id TEXT,
	original_focus TEXT,
	detected_topic TEXT,
	prompt_snippet TEXT,
	action_taken TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE SET NULL
);
`

const createIndices = `
CREATE INDEX IF NOT EXISTS idx_workspaces_status ON workspaces(status);
CREATE INDEX IF NOT EXISTS idx_workspaces_last_active ON workspaces(last_active_at);
CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_last_active ON sessions(last_active_at);
CREATE INDEX IF NOT EXISTS idx_sessions_backend ON sessions(backend);
CREATE INDEX IF NOT EXISTS idx_checkpoints_workspace ON checkpoints(workspace_id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON checkpoints(session_id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_created ON checkpoints(created_at);
CREATE INDEX IF NOT EXISTS idx_context_items_name ON context_items(name);
CREATE INDEX IF NOT EXISTS idx_context_items_type ON context_items(type);
CREATE INDEX IF NOT EXISTS idx_context_items_last_used ON context_items(last_used_at);
CREATE INDEX IF NOT EXISTS idx_rules_scope ON rules(scope);
CREATE INDEX IF NOT EXISTS idx_rules_active ON rules(is_active);
CREATE INDEX IF NOT EXISTS idx_drift_log_workspace ON drift_log(workspace_id);
CREATE INDEX IF NOT EXISTS idx_drift_log_session ON drift_log(session_id);
CREATE INDEX IF NOT EXISTS idx_drift_log_created ON drift_log(created_at);
`

// Wave 10: Response cache table for LLM responses
const createResponseCacheTable = `
CREATE TABLE response_cache (
	key TEXT PRIMARY KEY,
	fingerprint TEXT NOT NULL,
	model_id TEXT NOT NULL,
	response_content TEXT NOT NULL,
	input_tokens INTEGER DEFAULT 0,
	output_tokens INTEGER DEFAULT 0,
	finish_reason TEXT,
	model_used TEXT,
	duration_ns INTEGER DEFAULT 0,
	size_bytes INTEGER DEFAULT 0,
	hit_count INTEGER DEFAULT 0,
	ttl_seconds INTEGER NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP NOT NULL,
	last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

// Wave 10: Cache statistics table for tracking performance
const createCacheStatsTable = `
CREATE TABLE cache_stats (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	stat_type TEXT NOT NULL,
	stat_value INTEGER DEFAULT 0,
	model_id TEXT,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO cache_stats (stat_type, stat_value) VALUES
	('hit_count', 0),
	('miss_count', 0),
	('eviction_count', 0),
	('expired_count', 0),
	('input_tokens_saved', 0),
	('output_tokens_saved', 0);
`

// Wave 10: Cache indices for performance
const createCacheIndices = `
CREATE INDEX IF NOT EXISTS idx_response_cache_fingerprint ON response_cache(fingerprint);
CREATE INDEX IF NOT EXISTS idx_response_cache_model ON response_cache(model_id);
CREATE INDEX IF NOT EXISTS idx_response_cache_expires ON response_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_response_cache_created ON response_cache(created_at);
CREATE INDEX IF NOT EXISTS idx_response_cache_last_accessed ON response_cache(last_accessed_at);
CREATE INDEX IF NOT EXISTS idx_cache_stats_type ON cache_stats(stat_type);
CREATE INDEX IF NOT EXISTS idx_cache_stats_model ON cache_stats(model_id);
`

// Wave 11: Execution records table for workflow execution metrics
const createExecutionRecordsTable = `
CREATE TABLE execution_records (
	id TEXT PRIMARY KEY,
	skill_id TEXT NOT NULL,
	skill_name TEXT NOT NULL,
	status TEXT NOT NULL,
	input_tokens INTEGER DEFAULT 0,
	output_tokens INTEGER DEFAULT 0,
	total_cost REAL DEFAULT 0,
	duration_ns INTEGER DEFAULT 0,
	phase_count INTEGER DEFAULT 0,
	cache_hits INTEGER DEFAULT 0,
	cache_misses INTEGER DEFAULT 0,
	primary_model TEXT,
	started_at TIMESTAMP NOT NULL,
	completed_at TIMESTAMP NOT NULL,
	correlation_id TEXT
);
`

// Wave 11: Phase execution records table for individual phase metrics
const createPhaseExecutionRecordsTable = `
CREATE TABLE phase_execution_records (
	id TEXT PRIMARY KEY,
	execution_id TEXT NOT NULL,
	phase_id TEXT NOT NULL,
	phase_name TEXT NOT NULL,
	status TEXT NOT NULL,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	input_tokens INTEGER DEFAULT 0,
	output_tokens INTEGER DEFAULT 0,
	cost REAL DEFAULT 0,
	duration_ns INTEGER DEFAULT 0,
	cache_hit BOOLEAN DEFAULT 0,
	started_at TIMESTAMP NOT NULL,
	completed_at TIMESTAMP NOT NULL,
	error_message TEXT,
	FOREIGN KEY (execution_id) REFERENCES execution_records(id) ON DELETE CASCADE
);
`

// Wave 11: Metrics indices for performance
const createMetricsIndices = `
CREATE INDEX IF NOT EXISTS idx_execution_records_skill ON execution_records(skill_id);
CREATE INDEX IF NOT EXISTS idx_execution_records_status ON execution_records(status);
CREATE INDEX IF NOT EXISTS idx_execution_records_started ON execution_records(started_at);
CREATE INDEX IF NOT EXISTS idx_execution_records_completed ON execution_records(completed_at);
CREATE INDEX IF NOT EXISTS idx_execution_records_correlation ON execution_records(correlation_id);
CREATE INDEX IF NOT EXISTS idx_phase_records_execution ON phase_execution_records(execution_id);
CREATE INDEX IF NOT EXISTS idx_phase_records_provider ON phase_execution_records(provider);
CREATE INDEX IF NOT EXISTS idx_phase_records_model ON phase_execution_records(model);
CREATE INDEX IF NOT EXISTS idx_phase_records_started ON phase_execution_records(started_at);
CREATE INDEX IF NOT EXISTS idx_phase_records_cache_hit ON phase_execution_records(cache_hit);
`

// Crash Recovery: Workflow checkpoints table for storing execution state
const createWorkflowCheckpointsTable = `
CREATE TABLE workflow_checkpoints (
	id TEXT PRIMARY KEY,
	execution_id TEXT NOT NULL,
	skill_id TEXT NOT NULL,
	skill_name TEXT NOT NULL,
	input TEXT NOT NULL,
	input_hash TEXT NOT NULL,
	completed_batch INTEGER DEFAULT -1,
	total_batches INTEGER NOT NULL,
	phase_results TEXT,
	phase_outputs TEXT,
	status TEXT NOT NULL DEFAULT 'in_progress',
	input_tokens INTEGER DEFAULT 0,
	output_tokens INTEGER DEFAULT 0,
	machine_id TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

// Crash Recovery: Workflow checkpoint indices for performance
const createWorkflowCheckpointIndices = `
CREATE INDEX IF NOT EXISTS idx_wf_checkpoint_skill_input ON workflow_checkpoints(skill_id, input_hash);
CREATE INDEX IF NOT EXISTS idx_wf_checkpoint_execution ON workflow_checkpoints(execution_id);
CREATE INDEX IF NOT EXISTS idx_wf_checkpoint_status ON workflow_checkpoints(status);
CREATE INDEX IF NOT EXISTS idx_wf_checkpoint_machine ON workflow_checkpoints(machine_id);
CREATE INDEX IF NOT EXISTS idx_wf_checkpoint_updated ON workflow_checkpoints(updated_at);
CREATE INDEX IF NOT EXISTS idx_wf_checkpoint_created ON workflow_checkpoints(created_at);
`
