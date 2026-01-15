# Skillrunner: Complete Project Overview

**Version:** 2.0 MVP
**Status:** Wave 11 Complete (Unreleased)
**Last Updated:** 2026-01-15

---

## Table of Contents

1. [What is Skillrunner?](#what-is-skillrunner)
2. [Core Features](#core-features)
3. [Architecture Overview](#architecture-overview)
4. [Implementation Roadmap](#implementation-roadmap)
5. [Current Status](#current-status)
6. [Future Roadmap](#future-roadmap)
7. [Quick Start](#quick-start)
8. [Technical Details](#technical-details)

---

## What is Skillrunner?

**Skillrunner** is a **local-first AI workflow orchestration tool** that enables complex, multi-phase AI workflows with intelligent provider routing. It's designed to:

- **Execute complex AI tasks** as directed acyclic graphs (DAGs) of dependent phases
- **Prioritize local LLM execution** (Ollama) for privacy and cost savings
- **Seamlessly fallback to cloud providers** (Anthropic Claude, OpenAI, Groq) when needed
- **Provide cost-aware routing** through configurable profiles (cheap/balanced/premium)

### The Problem It Solves

Traditional AI integrations are limited to single-shot prompts. Real-world AI tasks often require:
- Multiple processing steps with dependencies
- Different models for different subtasks (analysis vs. generation vs. review)
- Intelligent routing based on cost/quality tradeoffs
- Local execution for sensitive data

**Skillrunner** solves this by treating AI workflows as first-class citizens, defined in YAML "skills" that can be shared, versioned, and executed consistently.

### Example Use Case

A **code-review skill** might have 3 phases:
1. **Pattern Analysis** (cheap profile) - Scan for common patterns
2. **Security Review** (premium profile) - Deep security analysis with best model
3. **Report Generation** (balanced profile) - Synthesize findings into a report

Each phase can depend on outputs from previous phases, and the system automatically handles execution order, parallelization, and provider selection.

---

## Core Features

### Implemented Features (MVP)

| Feature | Description | Status |
|---------|-------------|--------|
| **Multi-Phase Workflows** | DAG-based execution with dependencies between phases | ✅ Complete |
| **Local-First Execution** | Prioritizes Ollama for privacy and cost savings | ✅ Complete |
| **Multi-Provider Support** | Ollama, Anthropic Claude, OpenAI, Groq | ✅ Complete |
| **Routing Profiles** | cheap/balanced/premium profiles for cost-quality tradeoffs | ✅ Complete |
| **YAML Skill Definitions** | Declarative workflow definitions | ✅ Complete |
| **CLI Interface** | Full-featured command-line tool | ✅ Complete |
| **Session Management** | Start, attach, detach AI coding sessions (Aider, Claude Code) | ✅ Complete |
| **Context Management** | Manage context items, focus, checkpoints, rules | ✅ Complete |
| **Workspace Management** | Initialize and manage project workspaces | ✅ Complete |
| **SQLite Storage** | Persistent storage for sessions, workspaces, checkpoints | ✅ Complete |
| **Provider Health Checks** | Real-time provider status monitoring | ✅ Complete |
| **Streaming Output** | Real-time LLM response streaming with live token counts | ✅ Complete |
| **Memory System** | Persistent context via MEMORY.md files (global + project) | ✅ Complete |
| **MCP Support** | Model Context Protocol server integration for tool extensibility | ✅ Complete |
| **Plan Mode** | Preview execution plan with cost estimates before running | ✅ Complete |
| **Skill Hot Reload** | Automatic skill reload when YAML files change | ✅ Complete |
| **10 Built-in Skills** | Expanded skill library for common development tasks | ✅ Complete |

### Routing Profiles

| Profile | Description | Best For |
|---------|-------------|----------|
| `cheap` | Local-first, cost optimization | Drafts, exploration, high-volume tasks |
| `balanced` | Quality-to-cost ratio (default) | General use, production workloads |
| `premium` | Best available models | Critical tasks, final outputs |

### Built-in Skills (10 Total)

| Skill | Description | Phases |
|-------|-------------|--------|
| `code-review` | Comprehensive code review | 3 phases (patterns → security → report) |
| `test-gen` | Generate unit tests | Coverage analysis + test generation |
| `doc-gen` | Generate documentation | Extract structure + generate docs |
| `changelog` | Generate changelog entries | Git analysis + formatting |
| `commit-msg` | Generate commit messages | Diff analysis + conventional commit |
| `pr-description` | Generate PR descriptions | Change analysis + summary |
| `lint-fix` | Fix linting errors | Identify → fix → verify |
| `test-fix` | Debug failing tests | Analyze → diagnose → fix |
| `refactor` | Apply refactoring patterns | Analyze → transform → validate |
| `issue-breakdown` | Break down issues | Complexity analysis + subtasks |

---

## Architecture Overview

Skillrunner uses **Hexagonal Architecture** (Ports & Adapters) for clean separation of concerns.

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                       │
│                         (CLI)                               │
│  Commands: run, ask, list, status, session, context, etc.  │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                   APPLICATION LAYER                         │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Container                          │   │
│  │  (Dependency Injection - Central Service Registry)  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Services:                                                  │
│  • WorkflowExecutor - DAG-based skill execution            │
│  • PhaseExecutor - Single phase execution                  │
│  • SessionManager - AI session lifecycle                   │
│  • ProviderRouter - Intelligent model selection            │
│  • SkillRegistry - Skill discovery and caching             │
│                                                             │
│  Ports (Interfaces):                                        │
│  • ProviderPort - LLM provider interface                   │
│  • SessionStoragePort - Session persistence                │
│  • WorkspaceStoragePort - Workspace persistence            │
│  • ContextStoragePort - Context item persistence           │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                    DOMAIN LAYER                             │
│                 (Pure Business Logic)                       │
│                                                             │
│  • Skill - Multi-phase workflow definition                 │
│  • Phase - Single execution step                           │
│  • WorkflowDAG - Dependency graph algorithms               │
│  • Session - AI coding session entity                      │
│  • Context - Context items, focus, rules                   │
│  • RoutingConfig - Cost-aware model selection              │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                    ADAPTER LAYER                            │
│                                                             │
│  Providers:                                                 │
│  ┌────────────┬────────────┬────────────┬────────────┐     │
│  │  Ollama    │ Anthropic  │  OpenAI    │   Groq     │     │
│  │  (local)   │  (cloud)   │  (cloud)   │  (cloud)   │     │
│  └────────────┴────────────┴────────────┴────────────┘     │
│                                                             │
│  Backends (Session Management):                             │
│  ┌────────────┬────────────┬────────────┐                  │
│  │   Aider    │Claude Code │  OpenCode  │                  │
│  └────────────┴────────────┴────────────┘                  │
│                                                             │
│  Storage:                                                   │
│  • SQLite repositories for all persistence                 │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                 INFRASTRUCTURE LAYER                        │
│                                                             │
│  • Configuration (YAML, env vars)                          │
│  • Skill loading from filesystem                           │
│  • Terminal/tmux integration                               │
│  • Git integration                                         │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Principles

1. **Local-First** - Prioritize local models for privacy and cost
2. **Minimal Dependencies** - Go standard library where possible
3. **Thread-Safe** - All shared components use proper synchronization
4. **Cost-Aware** - Explicit routing profiles for cost control
5. **Dependency Injection** - Ports/interfaces for testability
6. **Immutable Value Objects** - Phases and configs are immutable

---

## Implementation Roadmap

### Wave 1: Domain Layer Foundation
**Status:** ✅ Complete

- Skill aggregate with validation
- Phase value object
- Workflow DAG with cycle detection
- Routing configuration
- Domain error hierarchy

### Wave 2: Provider System
**Status:** ✅ Complete

- ProviderPort interface definition
- Ollama provider adapter
- Anthropic provider adapter
- OpenAI provider adapter
- Groq provider adapter
- Thread-safe provider registry

### Wave 3: Workflow Execution
**Status:** ✅ Complete

- WorkflowExecutor with DAG-based execution
- PhaseExecutor for single phase execution
- Parallel phase execution with semaphores
- Template rendering with dependency outputs
- Result aggregation

### Wave 4: Provider Routing
**Status:** ✅ Complete

- Router with profile-based selection
- Resolver with fallback logic
- Health check integration
- Cost-aware model selection

### Wave 5: Session Management
**Status:** ✅ Complete

- Session entity and lifecycle
- Backend registry (Aider, Claude Code, OpenCode)
- Tmux integration for session multiplexing
- Session storage port and repository

### Wave 6: Context Management
**Status:** ✅ Complete

- Context items (files, snippets, URLs)
- Focus management (current working set)
- Checkpoint system (save/restore state)
- Rules engine (constraints and guidelines)

### Wave 7: Storage Infrastructure
**Status:** ✅ Complete

- SQLite connection management
- Session repository
- Workspace repository
- Checkpoint repository
- Context item repository
- Rule repository

### Wave 8: CLI Wiring & Integration
**Status:** ✅ Complete

- Application container (dependency injection)
- Skill registry with caching
- Provider initializer
- All CLI commands wired to real services:
  - `sr run` - Execute multi-phase workflows
  - `sr ask` - Single-phase queries
  - `sr list` - List available skills
  - `sr status` - Provider health status
  - `sr session *` - Session management
  - `sr context *` - Context management
  - `sr workspace *` - Workspace management

### Wave 9: Streaming & Real-time Output
**Status:** ✅ Complete

- StreamingExecutor interface for real-time LLM output
- StreamingPhaseExecutor with streaming callbacks
- Real-time phase progress events (started, progress, completed, failed)
- Live token counting during execution
- StreamingOutput formatter for CLI display
- Workflow-level events (started, completed)
- `--stream` flag wired in `sr run` command
- `--stream` flag wired in `sr ask` command
- Comprehensive test coverage for streaming

### Wave 10: Caching & Performance
**Status:** ✅ Complete

- Two-tier cache architecture (in-memory L1 + SQLite L2)
- Response caching for repeated LLM requests
- SHA256 request fingerprinting for deterministic cache keys
- TTL-based cache expiration with automatic cleanup
- LRU eviction when cache exceeds size limits
- Batch request aggregator for provider efficiency
- Cache CLI commands (`sr cache stats`, `sr cache clear`, `sr cache list`)
- SQLite migrations for persistent cache storage
- Comprehensive test coverage for all cache adapters

### Wave 11: Memory, MCP, Plan Mode & Hot Reload
**Status:** ✅ Complete

- **Memory System** - Persistent context via MEMORY.md files
  - Global memory at `~/.skillrunner/MEMORY.md`
  - Project memory at `.skillrunner/MEMORY.md`
  - Automatic injection into prompts
  - `sr memory edit` and `sr memory view` commands
  - `--no-memory` flag to disable
- **MCP Server Support** - Model Context Protocol integration
  - Read config from `.claude/mcp.json`
  - Dynamic tool discovery and registration
  - JSON-RPC tool execution
- **Plan Mode** - Execution preview
  - `sr plan` command for DAG visualization
  - Cost estimation before execution
  - `--approve` flag for automatic approval
- **Skill Hot Reload** - Live skill updates
  - fsnotify-based file watching
  - Automatic reload on YAML changes
  - No restart required
- **7 New Default Skills** - Expanded skill library
  - changelog, commit-msg, pr-description
  - lint-fix, test-fix, refactor
  - issue-breakdown

---

## Current Status

### What Works Now

```bash
# Check system status and provider health
sr status

# List available skills
sr list

# Run a multi-phase workflow
sr run code-review "Review this code for issues"

# Run with streaming output (real-time)
sr run code-review "Review this PR" --stream

# Quick single-phase query
sr ask code-review "What are common security issues?"

# Ask with streaming output
sr ask explain "Explain this code" --stream

# Session management
sr session start --backend aider
sr session list
sr session attach <session-id>
sr session peek <session-id>
sr session kill <session-id>

# Context management
sr context items add file.go
sr context items list
sr context focus set "authentication module"
sr context checkpoint create "before refactor"

# Workspace management
sr workspace init
sr workspace list
sr workspace show

# Cache management (Wave 10)
sr cache stats              # View cache statistics
sr cache clear              # Clear all cached responses
sr cache list               # List cached entries
sr cache config             # View cache configuration

# Memory management (Wave 11)
sr memory view              # View current memory content
sr memory edit              # Edit memory files

# Plan mode (Wave 11)
sr plan code-review "Review this code"  # Preview execution plan
sr plan code-review "Review" --approve  # Auto-approve and run
```

### Test Coverage

All tests pass with the following coverage:
- Domain layer: High coverage (pure logic, easy to test)
- Application layer: Good coverage (mocked ports)
- Adapters: Integration tests with mocked HTTP
- CLI commands: E2E tests for all commands

---

## Future Roadmap

### Wave 12: Reliability & Security (v1.1 - Feb 2026)
- AES-256-GCM encryption for API keys
- Circuit breakers for provider failover
- Health checks with remediation
- Hooks system for workflow customization
- Slash commands support
- Test coverage to 80%+

### Wave 13: Advanced Features (v1.2+ - Future)
- Crash recovery with checkpoint persistence
- Execution history (`sr history`)
- Per-phase cost/time display
- Skill marketplace
- Web dashboard
- REST API

---

## Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/jbctechsolutions/skillrunner.git
cd skillrunner
make build

# Or install directly
go install github.com/jbctechsolutions/skillrunner/cmd/skillrunner@latest
```

### Configuration

```bash
# Initialize configuration
sr init

# Edit config at ~/.skillrunner/config.yaml
```

**Example Configuration:**

```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    enabled: true
  openai:
    api_key: ${OPENAI_API_KEY}
    enabled: false
  groq:
    api_key: ${GROQ_API_KEY}
    enabled: false

routing:
  default_profile: balanced

skills:
  directory: ~/.skillrunner/skills

logging:
  level: info
  format: text

cache:
  enabled: true
  max_memory_size: 104857600  # 100MB L1 cache
  max_disk_size: 1073741824   # 1GB L2 cache
  default_ttl: 24h
  cleanup_period: 5m
```

### First Workflow

```bash
# Check providers are healthy
sr status

# List available skills
sr list

# Run a code review
sr run code-review "$(cat main.go)" --profile balanced

# Quick question with premium model
sr ask code-review "What's the best way to handle errors in Go?" --profile premium
```

---

## Technical Details

### Directory Structure

```
skillrunner/
├── cmd/skillrunner/           # CLI entry point
├── internal/
│   ├── domain/                # Pure business logic
│   │   ├── skill/             # Skill aggregate
│   │   ├── workflow/          # DAG execution
│   │   ├── session/           # Session entity
│   │   ├── context/           # Context management
│   │   └── errors/            # Domain errors
│   ├── application/           # Use cases and orchestration
│   │   ├── container.go       # Dependency injection
│   │   ├── ports/             # Interface definitions
│   │   ├── workflow/          # Workflow executor
│   │   ├── provider/          # Provider routing
│   │   ├── session/           # Session manager
│   │   └── skills/            # Skill registry
│   ├── adapters/              # External integrations
│   │   ├── provider/          # LLM providers
│   │   │   ├── ollama/
│   │   │   ├── anthropic/
│   │   │   ├── openai/
│   │   │   └── groq/
│   │   ├── backend/           # AI session backends
│   │   └── sync/sqlite/       # SQLite storage
│   ├── infrastructure/        # Cross-cutting concerns
│   │   ├── config/            # Configuration
│   │   ├── skills/            # Skill loading
│   │   └── storage/           # Storage repositories
│   └── presentation/          # User interface
│       └── cli/               # CLI commands
├── skills/                    # Built-in skill definitions
├── configs/                   # Example configurations
└── docs/                      # Documentation
```

### Skill Definition Format

```yaml
id: code-review
name: Code Review
version: 1.0.0
description: Multi-phase code review workflow

routing:
  default_profile: balanced
  generation_model: claude-3-5-sonnet-20241022
  review_model: gpt-4o
  fallback_model: llama3.1:8b

phases:
  - id: patterns
    name: Pattern Analysis
    routing_profile: cheap
    prompt_template: |
      Analyze the following code for patterns and anti-patterns:

      {{._input}}
    max_tokens: 2048
    temperature: 0.3

  - id: security
    name: Security Review
    routing_profile: premium
    depends_on: [patterns]
    prompt_template: |
      Based on the pattern analysis:
      {{.patterns}}

      Perform a security review of:
      {{._input}}
    max_tokens: 4096
    temperature: 0.2

  - id: report
    name: Final Report
    routing_profile: balanced
    depends_on: [patterns, security]
    prompt_template: |
      Synthesize findings into a report:

      Pattern Analysis:
      {{.patterns}}

      Security Review:
      {{.security}}
    max_tokens: 4096
    temperature: 0.4
```

### Requirements

- **Go 1.23+** - For building from source
- **Ollama** - Recommended for local execution
- **API Keys** - For cloud providers (optional)
- **tmux** - For session management features

---

## Contributing

Contributions are welcome! The codebase follows:

- **Go Code Review Comments** style guidelines
- **Hexagonal Architecture** patterns
- **Domain-Driven Design** principles
- **High test coverage** requirements (75%+)

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines.

---

## License

MIT License - see [LICENSE](../LICENSE) for details.

---

**Document Status:** Complete
**Last Review:** 2025-12-26
**Maintainer:** JBC Tech Solutions
