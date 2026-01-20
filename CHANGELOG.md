# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [1.1.0] - 2026-01-20

### Added

#### File Context Detection & Permissions
- Automatic file detection from natural language input
- Regex-based pattern matching for file paths and bare filenames
- Recursive file search up to 5 levels deep with smart ignore patterns
- Interactive permission system for file access approval
  - Four modes: approve all, deny all, individual approval, preview
  - Sensitive file detection (.env, credentials, keys)
  - `-y` flag for non-interactive auto-approval
- File size limits (1MB default) and binary detection
- Comprehensive deduplication by absolute path
- Documentation at `docs/file-permissions.md`

#### Crash Recovery ([#7](https://github.com/jbctechsolutions/skillrunner/pull/7))
- Checkpoint persistence for workflow state
- Automatic recovery from crashes and interruptions
- Resume workflows from last successful phase
- Persistent checkpoint storage in SQLite

#### Per-Phase Cost Tracking ([#8](https://github.com/jbctechsolutions/skillrunner/pull/8))
- Granular cost tracking at phase level
- Real-time cost display during execution
- Per-provider cost breakdown
- Token usage tracking per phase
- Cost estimation before workflow execution

#### Enhanced Documentation ([#6](https://github.com/jbctechsolutions/skillrunner/pull/6))
- Comprehensive CLI reference documentation
- Configuration guide with all available options
- Skills authoring guide with examples
- Architecture documentation for contributors

#### Memory System ([#1](https://github.com/jbctechsolutions/skillrunner/pull/1))
- Persistent context across sessions via MEMORY.md files
- Global memory at `~/.skillrunner/MEMORY.md` for user preferences
- Project memory at `.skillrunner/MEMORY.md` for project-specific context
- Automatic memory injection into prompts
- `sr memory edit` and `sr memory view` commands
- `--no-memory` flag to disable memory injection

#### MCP Server Support ([#2](https://github.com/jbctechsolutions/skillrunner/pull/2))
- Model Context Protocol (MCP) server integration
- Read MCP config from `.claude/mcp.json` for Claude Code compatibility
- MCP tool registry with dynamic tool discovery
- Tool execution via JSON-RPC protocol

#### New Default Skills ([#3](https://github.com/jbctechsolutions/skillrunner/pull/3))
- 7 additional built-in skills (10 total):
  - `changelog` - Generate changelog entries from git history
  - `commit-msg` - Generate conventional commit messages
  - `pr-description` - Generate pull request descriptions
  - `lint-fix` - Identify and fix linting errors
  - `test-fix` - Debug and fix failing tests
  - `refactor` - Apply refactoring patterns
  - `issue-breakdown` - Break down issues into subtasks

#### Skill Hot Reload ([#4](https://github.com/jbctechsolutions/skillrunner/pull/4))
- Automatic skill reload when YAML files change
- fsnotify-based file watching for `~/.skillrunner/skills/` and `.skillrunner/skills/`
- Debounced reload to handle rapid changes
- No CLI restart required for skill updates

#### Plan Mode ([#5](https://github.com/jbctechsolutions/skillrunner/pull/5))
- `sr plan` command to preview execution before running
- DAG visualization showing phase dependencies
- Cost estimation before execution
- `--approve` flag for automatic approval
- Token count and model selection preview

### Fixed

- **Multi-phase execution**: Replaced hardcoded placeholder model names with actual Ollama models (llama3.2:3b, llama3:8b, qwen2.5:14b)
- **Template rendering**: Fixed nested phase output access - templates now correctly support `{{.phases.phaseid}}` syntax
- **Logging configuration**: Fixed log level parsing to respect `logging.level` config value instead of checking wrong field
- **Skill templates**: Removed invalid `.output` suffix from doc-gen and test-gen skill phase references

### Changed

- Default log level changed from `info` to `warn` for cleaner output
- File permissions integrated into `sr ask` and `sr run` commands
- Enhanced error messages for template rendering failures

---

## [0.1.0] - 2025-01-01

### Added

#### Core Workflow Engine
- Multi-phase DAG (Directed Acyclic Graph) workflow execution with automatic dependency resolution
- Topological sorting using Kahn's algorithm for optimal execution order
- Parallel batch execution for independent phases
- Cycle detection to prevent invalid workflow definitions
- YAML-based skill definitions with template rendering and dependency output injection

#### Provider Integration
- Unified provider interface supporting multiple LLM backends:
  - **Ollama** - Local-first execution for privacy and cost savings
  - **Anthropic Claude** - Cloud-based Claude models
  - **OpenAI** - GPT model family support
  - **Groq** - High-speed inference
- Intelligent provider routing with three configurable profiles:
  - `cheap` - Prioritizes local models and cost-optimized providers
  - `balanced` - Mid-tier models balancing cost and capability (default)
  - `premium` - Best-available models for critical tasks
- Provider health monitoring with automatic failover
- Thread-safe provider registry with model discovery

#### Session Management
- Multi-backend AI coding session support:
  - **Aider** - Python-first AI pair programming integration
  - **Claude Code** - Anthropic's code editor integration
- Full session lifecycle management (start, attach, detach, terminate, peek)
- tmux integration for session multiplexing
- Context items management (files, code snippets, URLs)
- Focus management for guiding AI attention
- Checkpoint system for saving and restoring session state
- Rules engine for defining execution constraints and guidelines

#### Caching & Performance
- Two-tier composite caching architecture:
  - L1 Cache: In-memory cache for hot responses
  - L2 Cache: SQLite-backed persistent cache for long-term storage
- SHA256 request fingerprinting for deterministic cache keys
- TTL-based cache expiration with automatic cleanup
- LRU eviction when cache exceeds configured limits
- Cache hit rate tracking and cost savings metrics
- CLI commands: `sr cache stats`, `sr cache clear`, `sr cache list`

#### Observability
- Structured logging with slog:
  - Correlation ID tracking across requests
  - Multiple log levels (Debug, Info, Warn, Error)
  - Text and JSON output formats
- Distributed tracing with OpenTelemetry:
  - Workflow-level spans for complete executions
  - Phase-level spans for individual steps
  - Provider-level spans for LLM API calls
  - Multiple exporters (stdout, OTLP)
- SQLite-backed metrics persistence:
  - Execution records with token counts and duration
  - Phase-level execution metrics
  - Aggregated metrics for time periods
- Cost tracking with per-provider token pricing

#### Workspace Management
- Project-aware workspace initialization
- Git integration with automatic branch detection
- Git worktree support for parallel development
- Persistent workspace metadata in SQLite

#### CLI Interface
- Comprehensive command-line interface:
  - `sr run` - Execute multi-phase workflows with streaming
  - `sr ask` - Quick single-phase queries
  - `sr list` - List available skills
  - `sr status` - Provider health and system status
  - `sr init` - Initialize configuration
  - `sr chat` - Interactive chat sessions
  - `sr import` - Import external resources
- Session commands: `sr session start|list|attach|peek|kill`
- Context commands: `sr context items|focus|checkpoint|rules|init`
- Workspace commands: `sr workspace init|list|show`
- Metrics commands: `sr metrics` with aggregation options
- Multiple output formats (text with color, JSON, streaming)

#### Real-Time Streaming
- Stream LLM responses as they arrive
- Live token counting during generation
- Stream callbacks for handling chunks and progress events
- Workflow and phase-level streaming support

#### Configuration & Security
- YAML-based configuration with environment variable expansion
- API key encryption for secure credential storage
- Comprehensive configuration validation

#### Built-in Skills
- Code review skill with multi-phase analysis (patterns, security, reporting)
- Test generation skill with coverage analysis
- Documentation generation skill

#### Architecture
- Hexagonal (ports and adapters) architecture for clean separation of concerns
- Domain-driven design with skill aggregate and phase value objects
- Dependency injection container for service orchestration
- Typed error hierarchy with contextual information

### Infrastructure
- SQLite storage layer for persistence (sessions, workspaces, metrics, cache)
- Terminal and PTY spawning for session management
- Skill registry with filesystem-based loading and caching
- Comprehensive test suite with mocks and fixtures

---

## Version History

- **0.1.0** - Initial open-source beta release

[0.1.0]: https://github.com/jbctechsolutions/skillrunner/releases/tag/v0.1.0
