# Skillrunner: Planned Features & Implementation Roadmap

**Version:** 2.0 MVP → v3.0 Vision  
**Status:** Planning & Specification  
**Last Updated:** 2025-12-26

---

## Executive Summary

This document consolidates all planned features for Skillrunner identified through strategic planning sessions. These features transform Skillrunner from a workflow orchestration MVP into a comprehensive AI development platform with intelligent routing, parallel agent orchestration, and cloud-hosted services.

### Feature Categories

| Category | Waves | Priority | Value |
|----------|-------|----------|-------|
| **Plano Intelligent Routing** | 9A-9C | High | Cost optimization, better model selection |
| **CLI Provider Adapters** | 10 | High | Leverage existing subscriptions (Claude Max, ChatGPT Plus) |
| **Conductor Mode** | 11-15 | High | Parallel agent orchestration, competitive differentiator |
| **Metrics & Observability** | 16-17 | Medium | Insights, cost tracking, model improvement |
| **Cloud Services** | 9B, 18 | Medium | Paid tier, reliability for non-local users |

---

## Table of Contents

1. [Wave 9A: Plano-Orchestrator Local Integration](#wave-9a-plano-orchestrator-local-integration)
2. [Wave 9B: Cloud Orchestrator Fallback](#wave-9b-cloud-orchestrator-fallback)
3. [Wave 9C: Analytics Pipeline for Model Improvement](#wave-9c-analytics-pipeline-for-model-improvement)
4. [Wave 10: CLI Provider Adapters](#wave-10-cli-provider-adapters)
5. [Waves 11-15: Conductor Mode](#waves-11-15-conductor-mode)
6. [Waves 16-17: Metrics & Observability](#waves-16-17-metrics--observability)
7. [Implementation Timeline](#implementation-timeline)

---

## Wave 9A: Plano-Orchestrator Local Integration

### Overview

Replace static routing profiles with an intelligent routing model (Plano-Orchestrator) that analyzes incoming prompts to predict optimal provider/model selection.

### Problem Statement

Current routing uses static profiles (cheap/balanced/premium) that map to predefined provider/model combinations. This doesn't consider:
- Task complexity
- Expected token count
- Latency requirements
- Historical performance data

### Architecture

```
User Request
    ↓
Routing Strategy Selector
    ↓
┌─────────────────────────────────────────────────┐
│            Intelligent Orchestrator              │
│                                                  │
│  1. Analyze request characteristics              │
│  2. Query Plano-Orchestrator model               │
│  3. Generate OrchestrationPlan                   │
│  4. Route to optimal providers per phase         │
└─────────────────────────────────────────────────┘
    ↓
Workflow Executor
```

### Domain Model

```go
// internal/domain/orchestration/plan.go

// OrchestrationPlan is the output from the intelligent orchestrator
type OrchestrationPlan struct {
    ID              string                  `json:"id"`
    SkillID         string                  `json:"skill_id"`
    CreatedAt       time.Time               `json:"created_at"`
    Source          PlanSource              `json:"source"`           // "intelligent", "static", "cloud"
    Confidence      float64                 `json:"confidence"`       // 0.0-1.0
    Reasoning       string                  `json:"reasoning"`        // Why this plan was chosen
    PhaseDecisions  []PhaseRoutingDecision  `json:"phase_decisions"`
    EstimatedCost   *CostEstimate           `json:"estimated_cost,omitempty"`
    EstimatedLatency time.Duration          `json:"estimated_latency,omitempty"`
}

// PhaseRoutingDecision captures the routing decision for a single phase
type PhaseRoutingDecision struct {
    PhaseID         string  `json:"phase_id"`
    Provider        string  `json:"provider"`         // "ollama", "anthropic", "openai", "groq"
    Model           string  `json:"model"`            // Specific model identifier
    Confidence      float64 `json:"confidence"`       // 0.0-1.0 for this phase
    Reasoning       string  `json:"reasoning"`        // Why this provider/model
    FallbackProvider string `json:"fallback_provider,omitempty"`
    FallbackModel   string  `json:"fallback_model,omitempty"`
}
```

### Orchestrator Port Interface

```go
// internal/application/ports/orchestrator.go

type OrchestratorPort interface {
    // Plan generates an orchestration plan for a skill execution
    Plan(ctx context.Context, req *OrchestrationRequest) (*OrchestrationPlan, error)
    
    // Replan generates a new plan after a failure
    Replan(ctx context.Context, req *OrchestrationRequest, failedPhase string, err error) (*OrchestrationPlan, error)
    
    // HealthCheck returns the orchestrator's health status
    HealthCheck(ctx context.Context) (*HealthStatus, error)
}
```

### Implementation Strategy

1. **Plano Adapter**: Calls local Ollama with Plano-Orchestrator model
2. **Static Adapter**: Wraps existing profile-based routing
3. **Hybrid Adapter**: Tries Plano first, falls back to static

### CLI Updates

```bash
# Use intelligent routing
sr run code-review "Review this code" --routing intelligent

# Show the orchestration plan before execution
sr run code-review "Review this code" --show-plan

# Override with static routing
sr run code-review "Review this code" --routing static --profile premium
```

### Configuration

```yaml
orchestrator:
  default_strategy: hybrid
  
  intelligent:
    provider: ollama
    model: "plano-orchestrator:3b"
    max_tokens: 1024
    temperature: 0.1
    timeout: 10s
    
  hybrid:
    min_confidence: 0.7
    fallback_to_static: true
    intelligent_timeout: 5s
```

### Success Criteria

- [ ] Domain types for orchestration plan and request
- [ ] OrchestratorPort interface defined
- [ ] Plano adapter with prompt templates and parsing
- [ ] Static adapter wrapping existing profile logic
- [ ] Hybrid adapter with fallback
- [ ] Workflow executor uses orchestration plan
- [ ] CLI supports --routing and --show-plan flags
- [ ] All tests pass with >80% coverage

---

## Wave 9B: Cloud Orchestrator Fallback

### Overview

Add cloud fallback for the intelligent orchestrator, enabling a paid tier where users get reliable orchestration even without local Ollama setup.

### Architecture

```
User Request
    ↓
Routing Strategy Selector
    ↓
┌─────────────────────────────────────────────────┐
│            Hybrid Orchestrator                   │
│                                                  │
│  1. Try Local (Ollama + Plano)                  │
│     ↓ (if unavailable or error)                 │
│  2. Try Cloud (skillrunner-cloud endpoint)      │
│     ↓ (if unavailable or no API key)            │
│  3. Fall back to Static Routing                 │
└─────────────────────────────────────────────────┘
```

### Cloud API Specification

**Base URL:** `https://api.skillrunner.io`

#### POST /v1/plan
Generate an orchestration plan.

```json
// Request
{
  "skill_id": "code-review",
  "user_request": "Review this Python code for security issues",
  "available_providers": [
    {"name": "ollama", "models": ["llama3.2:8b"], "healthy": true},
    {"name": "groq", "models": ["llama-3.2-70b"], "healthy": true}
  ],
  "constraints": {
    "max_latency_ms": 5000,
    "max_budget_usd": 0.10,
    "require_local": false
  }
}

// Response
{
  "plan_id": "plan_abc123",
  "source": "cloud",
  "confidence": 0.92,
  "phase_decisions": [...],
  "estimated_cost_usd": 0.02,
  "reasoning": "Selected groq for speed, ollama for security analysis"
}
```

#### POST /v1/replan
Replan after a failure.

#### GET /v1/health
Health check endpoint.

### Digital Ocean Infrastructure (MVP)

| Component | DO Service | Cost/Month |
|-----------|------------|------------|
| API Server | Basic Droplet ($12) or App Platform | $12-20 |
| Database | Managed PostgreSQL (Basic) | $15 |
| Object Storage | Spaces (for model artifacts) | $5 |
| **Total MVP** | | **~$35/month** |

### Authentication & Tiers

```yaml
# User config
orchestrator:
  cloud:
    enabled: true
    endpoint: "https://api.skillrunner.io"
    api_key: ${SKILLRUNNER_API_KEY}
```

| Tier | Rate Limit | Features |
|------|------------|----------|
| Free | Local only | Ollama-based routing |
| Basic ($10/mo) | 1000 req/day | Cloud fallback |
| Pro ($25/mo) | Unlimited | Priority routing, analytics |

### CLI Commands

```bash
# Authenticate with cloud service
sr auth login

# Check cloud status
sr status
# Shows: Cloud Orchestrator: ✓ Connected (Basic tier)

# View API usage
sr auth usage
```

### Success Criteria

- [ ] Cloud orchestrator adapter with retry logic
- [ ] Hybrid orchestrator supports three-tier fallback
- [ ] API key management via config and `sr auth login`
- [ ] `sr status` shows cloud orchestrator status
- [ ] Plan source visible in `--show-plan` output
- [ ] Graceful degradation when cloud unavailable

---

## Wave 9C: Analytics Pipeline for Model Improvement

### Overview

Capture execution telemetry to train/improve routing models over time.

### Telemetry Schema

```go
// internal/domain/telemetry/execution.go

type ExecutionTelemetry struct {
    // Identifiers (no PII)
    ExecutionID     string    `json:"execution_id"`
    SkillID         string    `json:"skill_id"`
    PhaseID         string    `json:"phase_id"`
    Timestamp       time.Time `json:"timestamp"`
    
    // Routing Decision
    RoutingSource   string    `json:"routing_source"`   // "intelligent", "static", "cloud"
    SelectedProvider string   `json:"selected_provider"`
    SelectedModel   string    `json:"selected_model"`
    Confidence      float64   `json:"confidence"`
    
    // Features (for training)
    EstimatedTokens int       `json:"estimated_tokens"`
    TaskComplexity  float64   `json:"task_complexity"`  // 0.0-1.0
    
    // Outcomes
    ActualLatencyMs int64     `json:"actual_latency_ms"`
    ActualTokens    int       `json:"actual_tokens"`
    ActualCostUSD   float64   `json:"actual_cost_usd"`
    Success         bool      `json:"success"`
    ErrorType       string    `json:"error_type,omitempty"`
    
    // Optional Quality Signal
    UserRating      *int      `json:"user_rating,omitempty"` // 1-5
}
```

### Local SQLite Storage

```sql
CREATE TABLE execution_telemetry (
    id INTEGER PRIMARY KEY,
    execution_id TEXT NOT NULL,
    skill_id TEXT NOT NULL,
    phase_id TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    routing_source TEXT,
    selected_provider TEXT,
    selected_model TEXT,
    confidence REAL,
    estimated_tokens INTEGER,
    actual_latency_ms INTEGER,
    actual_tokens INTEGER,
    actual_cost_usd REAL,
    success BOOLEAN,
    error_type TEXT,
    user_rating INTEGER,
    synced BOOLEAN DEFAULT FALSE
);
```

### Data Pipeline (Digital Ocean)

```
Skillrunner CLI
    ↓
SQLite (local)
    ↓ (opt-in sync)
API (DO Droplet)
    ↓
PostgreSQL (DO Managed)
    ↓ (nightly export)
DO Spaces (Parquet)
    ↓ (GitHub Actions trigger)
Modal GPU (training)
    ↓
Model artifacts → DO Spaces
    ↓
API reload
```

### Privacy Controls

- **No prompt content** transmitted to cloud
- **Opt-in only** telemetry collection
- **Local-first** - all data stored locally by default
- **Anonymized** aggregates only for cloud training

### Configuration

```yaml
telemetry:
  enabled: true                    # Local collection
  cloud_sync: false                # Opt-in cloud sync
  retention_days: 90               # Local retention
  
  # Opt-in consent
  consent_given: false
  consent_timestamp: null
```

### Success Criteria

- [ ] Telemetry schema defined
- [ ] SQLite storage for local analytics
- [ ] Opt-in cloud sync mechanism
- [ ] Privacy-preserving data export
- [ ] Training pipeline specification
- [ ] A/B testing framework for model versions

---

## Wave 10: CLI Provider Adapters

### Overview

Add CLI-based provider adapters for Claude Code, OpenAI Codex CLI, and Gemini CLI. This allows users to leverage their existing AI subscriptions instead of paying for API calls separately.

### The Economics

| Approach | User Cost | Experience |
|----------|-----------|------------|
| Anthropic API Key | $3-15/M tokens | Simple but expensive |
| Claude Pro ($20/mo) | Fixed cost | Great value if subscribed |
| Claude Max ($100-200/mo) | Fixed cost | Best for power users |
| Groq (free tier) | Free / cheap | Fast, good for cheap profile |

### Subscription Tier Domain Model

```go
// internal/domain/subscription/subscription.go

type Tier string

const (
    TierFree       Tier = "free"
    TierBasic      Tier = "basic"      // Pro, Plus, Advanced
    TierPremium    Tier = "premium"    // Max, Pro
    TierEnterprise Tier = "enterprise"
    TierAPIKey     Tier = "api_key"    // Pay per use
    TierUnknown    Tier = "unknown"
)

type SubscriptionInfo struct {
    Provider    string
    Tier        Tier
    DisplayName string        // "Claude Max (20x)", "ChatGPT Plus"
    RateLimit   *RateLimit
    DetectedAt  time.Time
}

type RateLimit struct {
    RequestsPerMinute int
    TokensPerDay      int
}

func (s *SubscriptionInfo) UsageBudget() UsageBudget {
    switch s.Tier {
    case TierPremium:
        return UsageBudgetHigh
    case TierBasic:
        return UsageBudgetMedium
    default:
        return UsageBudgetLow
    }
}
```

### Target CLIs

| CLI | Command | Auth |
|-----|---------|------|
| Claude Code | `claude` | OAuth (Claude Pro/Max) or API key |
| Codex CLI | `codex` | OpenAI API key or ChatGPT Plus |
| Gemini CLI | `gemini` | Google OAuth or API key |

### Tier Detection

```go
// Claude Code tier detection
func detectClaudeTier(ctx context.Context) (*SubscriptionInfo, error) {
    // Check ~/.claude/ config
    // Run claude --version --json
    // Parse account info from response
}

// Codex CLI tier detection
func detectCodexTier(ctx context.Context) (*SubscriptionInfo, error) {
    // Check ~/.codex/ or ~/.config/codex/
    // Run codex --account-info if available
}

// Gemini CLI tier detection
func detectGeminiTier(ctx context.Context) (*SubscriptionInfo, error) {
    // Check Google OAuth token info
    // Check ~/.config/gemini/
}
```

### Tier-Aware Routing

```go
type TierAwareRouter struct {
    providers         []ports.ProviderPort
    subscriptionCache map[string]*SubscriptionInfo
}

func (r *TierAwareRouter) SelectProvider(ctx context.Context, profile RoutingProfile) (ports.ProviderPort, error) {
    candidates := r.getProvidersForProfile(profile)
    
    // Score each candidate based on:
    // 1. Subscription tier (prefer higher tiers)
    // 2. Current rate limit status
    // 3. Estimated remaining daily budget
    // 4. Historical performance
}
```

### Configuration

```yaml
providers:
  claude-code:
    enabled: true
    tier_override: ""  # Auto-detect, or "free", "pro", "max_5x", "max_20x"
    
  codex-cli:
    enabled: true
    tier_override: ""
    
  gemini-cli:
    enabled: true
    tier_override: ""

routing:
  auto_configure: true  # Auto-configure based on detected tiers
```

### Enhanced Status Command

```bash
$ sr status

Provider Status:

  CLI Providers (use your existing subscriptions):
    ✓ claude-code    Claude Max (20x) - High usage budget
                     Rate limit: ~1000 req/day estimated
    ✓ codex-cli      ChatGPT Plus - Moderate usage budget
    ✗ gemini-cli     Not installed
                     → npm install -g @google/gemini-cli

  API Providers:
    ✗ anthropic      Not configured
    ✓ groq           Available (free tier)

  Local Providers:
    ✓ ollama         Available (llama3.1:8b, mistral)

Routing Intelligence:
  ✓ Claude Max detected - using as primary provider for all profiles
  ✓ Estimated daily budget: ~$200 worth of API calls included
```

### Enhanced Init Command

```bash
$ sr init

Detecting available providers...

Found CLI tools:
  ✓ claude (Claude Code)
    Checking subscription... Claude Max (20x) detected!
    Estimated daily budget: ~$200 API equivalent
    
  ✓ codex (OpenAI Codex CLI)  
    Checking subscription... ChatGPT Plus detected
    Estimated daily budget: ~$20 API equivalent

Found local providers:
  ✓ ollama running at localhost:11434
    Models: llama3.1:8b, mistral, codellama

Configuring optimal routing...

Based on your Claude Max subscription, I recommend:
  • Use Claude Code as primary provider (high limits included)
  • Groq as secondary for parallel requests
  • Ollama for offline/private work

Generated config at ~/.skillrunner/config.yaml
```

### Success Criteria

- [ ] Base CLI provider abstraction
- [ ] Claude Code provider with tier detection
- [ ] Codex CLI provider with tier detection
- [ ] Gemini CLI provider with tier detection
- [ ] Tier-aware router implementation
- [ ] Usage tracking to avoid hitting limits
- [ ] Enhanced status and init commands
- [ ] Tests for all providers

---

## Waves 11-15: Conductor Mode

### Overview

Transform Skillrunner into a parallel agent orchestration platform similar to conductor.build, enabling users to run multiple AI coding agents simultaneously on different tasks with git worktree isolation.

### Gap Analysis: Current State vs Conductor

| Component | Current Status | Needed |
|-----------|---------------|--------|
| Session lifecycle | ✅ Start, attach, detach, kill | - |
| Multiple backends | ✅ Aider, Claude Code, OpenCode | - |
| Tmux integration | ✅ Session multiplexing | - |
| Workspace isolation | ✅ SQLite-backed | Git worktree binding |
| Context management | ✅ Files, focus, checkpoints | Per-agent isolation |
| Git worktree management | ❌ | **Wave 11** |
| Visual dashboard | ❌ | **Wave 13** |
| Parallel agent spawning | ❌ | **Wave 12** |
| PR/merge workflow | ❌ | **Wave 14** |
| Issue tracker integration | ❌ | **Wave 15** |

---

### Wave 11: Git Worktree Integration

**Goal:** Each workspace becomes an isolated git worktree.

```bash
# Create workspace with dedicated worktree
sr workspace create --worktree feature-auth
sr workspace create --worktree fix-bug-123

# List workspaces with worktree info
sr workspace list
# ID          Branch          Path                    Status
# ws-abc123   feature-auth    .worktrees/feature-auth active
# ws-def456   fix-bug-123     .worktrees/fix-bug-123  active

# Delete workspace and cleanup worktree
sr workspace delete ws-abc123 --cleanup
```

**Domain Model:**

```go
type Workspace struct {
    ID          string
    Name        string
    Path        string
    GitWorktree *GitWorktreeInfo
    Sessions    []string  // Active session IDs
    CreatedAt   time.Time
}

type GitWorktreeInfo struct {
    Branch     string
    BaseBranch string
    Path       string
    Clean      bool
    Commits    int  // Commits ahead of base
}
```

**Success Criteria:**
- [ ] Workspace creates git worktree on --worktree flag
- [ ] Branch auto-created from workspace name
- [ ] Worktree path managed in ~/.skillrunner/worktrees/
- [ ] Cleanup removes worktree and optionally branch
- [ ] Status shows worktree state (clean/dirty, commits ahead)

---

### Wave 12: Parallel Session Orchestration

**Goal:** Manage multiple sessions as a cohort with task binding.

```bash
# Launch conductor mode
sr conductor start

# Spawn agents on tasks
sr conductor spawn --task "Implement user authentication" --backend claude-code
sr conductor spawn --task "Fix issue #123" --backend aider
sr conductor spawn --task "Add unit tests for auth module" --backend claude-code

# View all active agents
sr conductor status
# Agent  Task                              Backend      Worktree        Status
# 1      Implement user authentication     claude-code  feature-auth    working
# 2      Fix issue #123                    aider        fix-123         waiting
# 3      Add unit tests for auth module   claude-code  auth-tests      working

# Peek at agent output
sr conductor peek 1

# Stop an agent
sr conductor stop 1
```

**Domain Model:**

```go
type ConductorState struct {
    ID        string
    Agents    []Agent
    StartedAt time.Time
}

type Agent struct {
    ID          int
    Task        string
    Backend     string
    SessionID   string
    WorkspaceID string
    Status      AgentStatus  // "spawning", "working", "waiting", "done", "error"
    StartedAt   time.Time
    LastOutput  string
}
```

**Success Criteria:**
- [ ] `sr conductor start` initializes conductor state
- [ ] `sr conductor spawn` creates workspace + session + binds task
- [ ] `sr conductor status` shows all agents at a glance
- [ ] `sr conductor peek` streams recent output
- [ ] `sr conductor stop` gracefully terminates agent

---

### Wave 13: Live TUI Dashboard

**Goal:** Terminal UI showing all agents at a glance.

**Technology:** bubbletea/lipgloss (Go TUI framework)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  SKILLRUNNER CONDUCTOR                          3 agents running    │
├─────────────────────────────────────────────────────────────────────┤
│ [1] Implement authentication        claude-code │ ● working         │
│ ─────────────────────────────────────────────── │                   │
│ Creating auth middleware...                     │ feature-auth      │
│ Adding JWT token validation...                  │ +12 commits       │
├─────────────────────────────────────────────────┼───────────────────┤
│ [2] Fix issue #123                       aider │ ○ waiting          │
│ ─────────────────────────────────────────────── │                   │
│ Waiting for input...                            │ fix-123           │
│                                                 │ +3 commits        │
├─────────────────────────────────────────────────┼───────────────────┤
│ [3] Add unit tests                  claude-code │ ● working         │
│ ─────────────────────────────────────────────── │                   │
│ Writing test for login handler...               │ auth-tests        │
│ Running pytest...                               │ +5 commits        │
└─────────────────────────────────────────────────────────────────────┘
 [1-3] Focus  [a] Attach  [k] Kill  [r] Refresh  [p] PR  [q] Quit
```

**Key Features:**
- Real-time output streaming from all agents
- Keyboard shortcuts for common actions
- Status indicators (working/waiting/done/error)
- Git status per worktree

**Success Criteria:**
- [ ] TUI renders all active agents
- [ ] Real-time output updates
- [ ] Keyboard navigation and actions
- [ ] Focus mode (expand single agent)
- [ ] Attach jumps to tmux session

---

### Wave 14: PR/Merge Workflow

**Goal:** Review, PR, and merge agent work from CLI.

```bash
# Review agent's changes
sr conductor review 1
# Shows diff of all changes in worktree

# Create PR for agent's work
sr conductor pr 1 --title "Add user authentication"
# Creates PR on GitHub/GitLab

# Merge and cleanup
sr conductor merge 1
# Merges PR, deletes worktree and branch
```

**Integration Points:**
- GitHub API (gh CLI or direct API)
- GitLab API
- Bitbucket API (optional)

**Success Criteria:**
- [ ] `sr conductor review` shows worktree diff
- [ ] `sr conductor pr` creates PR via GitHub/GitLab API
- [ ] `sr conductor merge` merges and cleans up
- [ ] Auto-detect remote (GitHub vs GitLab)

---

### Wave 15: Issue Tracker Integration

**Goal:** Pull issues and assign to agents.

```bash
# Pull issues from Linear
sr conductor pull-issues --linear
# ID       Title                           Labels
# LIN-123  Fix login timeout               bug, auth
# LIN-124  Add password reset              feature, auth

# Pull from GitHub
sr conductor pull-issues --github
# #45      API rate limiting               enhancement
# #46      Fix memory leak                 bug

# Spawn agent for specific issue
sr conductor spawn --issue LIN-123
# Creates agent with task from issue, auto-names worktree

# Auto-update issue status
sr conductor sync
# Updates LIN-123 status to "In Progress"
```

**Integrations:**
- Linear (via API)
- GitHub Issues
- Jira (optional)

**Success Criteria:**
- [ ] `sr conductor pull-issues` lists available issues
- [ ] `sr conductor spawn --issue` binds issue to agent
- [ ] Issue metadata populates task description
- [ ] Status sync updates issue state

---

## Waves 16-17: Metrics & Observability

### Wave 16: Local Metrics Collection (MVP)

**Goal:** Capture execution metrics locally for user insights.

**SQLite Schema:**

```sql
CREATE TABLE skill_executions (
    id INTEGER PRIMARY KEY,
    execution_id TEXT UNIQUE,
    skill_id TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    status TEXT,  -- 'success', 'failed', 'cancelled'
    total_phases INTEGER,
    completed_phases INTEGER,
    total_tokens INTEGER,
    total_cost_usd REAL,
    routing_profile TEXT,
    orchestrator_source TEXT
);

CREATE TABLE phase_executions (
    id INTEGER PRIMARY KEY,
    execution_id TEXT REFERENCES skill_executions(execution_id),
    phase_id TEXT,
    provider TEXT,
    model TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    latency_ms INTEGER,
    input_tokens INTEGER,
    output_tokens INTEGER,
    cost_usd REAL,
    status TEXT,
    error TEXT
);
```

**CLI Commands:**

```bash
# View execution history
sr metrics history
# Date       Skill         Phases  Tokens   Cost     Status
# 2025-01-15 code-review   3/3     4,521    $0.02    success
# 2025-01-15 test-gen      2/2     2,103    $0.01    success

# View cost summary
sr metrics cost --period week
# Provider    Requests  Tokens    Cost
# claude-code 45        125,000   $0.00 (subscription)
# groq        120       89,000    $0.00 (free tier)
# ollama      30        45,000    $0.00 (local)

# View provider performance
sr metrics performance
# Provider    Avg Latency  P95 Latency  Success Rate
# ollama      1.2s         2.5s         98.5%
# groq        0.8s         1.5s         99.2%
# claude-code 2.1s         4.2s         97.8%
```

**Success Criteria:**
- [ ] SQLite schema for executions
- [ ] Automatic capture on workflow completion
- [ ] `sr metrics history` command
- [ ] `sr metrics cost` command
- [ ] `sr metrics performance` command

---

### Wave 17: Optional Telemetry & OpenTelemetry

**Goal:** Opt-in telemetry export for product improvement and OTel integration.

**OpenTelemetry Integration:**

```go
// Add tracing spans to workflow execution
func (e *WorkflowExecutor) Execute(ctx context.Context, skill *Skill) (*Result, error) {
    ctx, span := tracer.Start(ctx, "workflow.execute",
        trace.WithAttributes(
            attribute.String("skill.id", skill.ID),
            attribute.String("routing.profile", skill.RoutingProfile),
        ),
    )
    defer span.End()
    
    // ... execution logic
}
```

**Configuration:**

```yaml
telemetry:
  enabled: true
  
  # Local metrics
  local:
    database: ~/.skillrunner/metrics.db
    retention_days: 90
    
  # Optional cloud sync (requires consent)
  cloud:
    enabled: false
    consent_given: false
    endpoint: https://telemetry.skillrunner.io
    
  # OpenTelemetry (users can configure their own collector)
  opentelemetry:
    enabled: false
    endpoint: ""  # e.g., localhost:4317 for local Jaeger
    service_name: skillrunner
```

**Success Criteria:**
- [ ] OpenTelemetry tracing spans added
- [ ] Configurable OTel exporter
- [ ] Opt-in consent flow for cloud telemetry
- [ ] Privacy-preserving data export (no prompts/outputs)

---

## Implementation Timeline

### Phase 1: Intelligent Routing (Q1 2026)
| Wave | Feature | Duration | Dependencies |
|------|---------|----------|--------------|
| 9A | Plano Local | 2 weeks | None |
| 9B | Cloud Fallback | 2 weeks | 9A, DO infrastructure |
| 9C | Analytics Pipeline | 2 weeks | 9A, 9B |

### Phase 2: CLI Providers (Q1 2026)
| Wave | Feature | Duration | Dependencies |
|------|---------|----------|--------------|
| 10 | CLI Provider Adapters | 3 weeks | None |

### Phase 3: Conductor Mode (Q2 2026)
| Wave | Feature | Duration | Dependencies |
|------|---------|----------|--------------|
| 11 | Git Worktree | 2 weeks | None |
| 12 | Parallel Sessions | 2 weeks | 11 |
| 13 | TUI Dashboard | 3 weeks | 12 |
| 14 | PR/Merge Workflow | 2 weeks | 11 |
| 15 | Issue Integration | 2 weeks | 12 |

### Phase 4: Observability (Q2-Q3 2026)
| Wave | Feature | Duration | Dependencies |
|------|---------|----------|--------------|
| 16 | Local Metrics | 1 week | None |
| 17 | OTel & Telemetry | 2 weeks | 16 |

---

## Research Required

Before implementation, verify:

1. **Plano-Orchestrator GGUF availability** - Check HuggingFace for GGUF format
2. **Claude Code CLI interface** - Research `claude` command flags and output formats
3. **Codex CLI interface** - Review https://github.com/openai/codex for exact API
4. **Gemini CLI interface** - Check Google's documentation
5. **Tier detection methods** - Research config file locations and API responses

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Plano GGUF not available | High | Use alternative small models, fine-tune own |
| CLI tools change interfaces | Medium | Abstract behind adapter layer, version pinning |
| Rate limit detection unreliable | Medium | Conservative defaults, manual override option |
| TUI complexity | Medium | Start with basic status, iterate |
| GitHub/Linear API changes | Low | Use official SDKs where available |

---

## Success Metrics

### Technical Metrics
- Test coverage > 80% on new code
- P95 latency < 100ms for routing decisions
- Zero data loss for local metrics

### Business Metrics
- 50% of users enable CLI provider adapters
- 30% of power users adopt Conductor mode
- Cloud tier conversion rate > 5%

---

**Document Status:** Complete  
**Last Review:** 2025-12-26  
**Maintainer:** JBC Tech Solutions
