# Skillrunner v2.0: Complete Rebuild Specification

## Executive Summary

This specification defines a ground-up rebuild of Skillrunner - a CLI tool for local-first AI workflow orchestration that routes AI tasks to local models (Ollama) first, falling back to cloud providers (Anthropic, OpenAI, Groq) when needed, achieving 70-90% cost savings.

**Module:** `github.com/jbctechsolutions/skillrunner`
**Go Version:** 1.23+
**Approach:** Test-Driven Development (TDD) with Clean/Hexagonal Architecture

---

## Table of Contents

1. [Product Vision](#1-product-vision)
2. [Architecture](#2-architecture)
3. [Package Structure](#3-package-structure)
4. [Core Interfaces](#4-core-interfaces)
5. [Domain Entities](#5-domain-entities)
6. [Testing Strategy](#6-testing-strategy)
7. [Implementation Roadmap](#7-implementation-roadmap)
8. [External Dependencies](#8-external-dependencies)
9. [Configuration](#9-configuration)
10. [Security Requirements](#10-security-requirements)

---

## 1. Product Vision

### Core Value Proposition
- **Cost Optimization:** 70-90% savings through intelligent local-first routing
- **Multi-Phase Workflows:** DAG-based execution with dependency management
- **Universal Skill Import:** Import skills from Claude, MCP servers, web, git
- **Observability:** Per-phase cost tracking, outcome analysis, auto-optimization

### Target Users
- Developers running AI-powered workflows locally
- Teams sharing optimized workflows
- CI/CD pipelines needing AI capabilities

### Feature Versions

| Version | Theme | Key Features |
|---------|-------|--------------|
| v1.0 | Core Launch | CLI, 4 providers, DAG execution, cost tracking, 3 demo skills |
| v1.1 | Security & Reliability | AES-GCM encryption, circuit breakers, slog, 80% coverage |
| v1.2 | Learning | Outcome tracking, auto-optimization, SQLite metrics |
| v1.3 | Intelligence | Web dashboard, skill decomposition, MCP protocol |
| v1.4 | Teams & API | REST API, team sync, budget alerts, paid tiers |

---

## 2. Architecture

### Hexagonal Architecture (Ports & Adapters)

```
┌─────────────────────────────────────────────────────────────────┐
│                      PRESENTATION LAYER                         │
│   CLI Commands (cobra)  │  Web Dashboard  │  REST API (v1.4)   │
└────────────────────────────────┬────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                      APPLICATION LAYER                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    USE CASES                            │   │
│  │  RunSkill │ ResolveModel │ TrackCost │ ImportSkill     │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                      PORTS                              │   │
│  │  ProviderPort │ StoragePort │ CachePort │ SecretPort   │   │
│  └─────────────────────────────────────────────────────────┘   │
└────────────────────────────────┬────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                        DOMAIN LAYER                             │
│  Skill │ Phase │ DAG │ Execution │ Model │ Cost │ Team         │
│                    (NO EXTERNAL DEPENDENCIES)                   │
└─────────────────────────────────────────────────────────────────┘
                                 ▲
                                 │ implements
┌────────────────────────────────┴────────────────────────────────┐
│                       ADAPTERS LAYER                            │
│  Providers: Ollama │ Anthropic │ OpenAI │ Groq                 │
│  Storage: SQLite │ Filesystem │ Memory                         │
│  Security: Keyring │ AES-GCM │ CircuitBreaker                  │
│  Import: Git │ Web │ Claude │ MCP                              │
└─────────────────────────────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                    INFRASTRUCTURE LAYER                         │
│  Config (viper) │ Logging (slog) │ DI Container │ Telemetry   │
└─────────────────────────────────────────────────────────────────┘
```

### Dependency Rule
Dependencies flow INWARD only. Domain has zero external dependencies.

---

## 3. Package Structure

```
skillrunner/
├── cmd/
│   ├── skillrunner/              # CLI entry point
│   │   └── main.go
│   └── api/                      # REST API server (v1.4)
│       └── main.go
│
├── internal/
│   ├── domain/                   # PURE BUSINESS LOGIC - NO I/O
│   │   ├── skill/
│   │   │   ├── skill.go          # Skill aggregate root
│   │   │   ├── phase.go          # Phase value object
│   │   │   └── routing.go        # Routing configuration
│   │   ├── workflow/
│   │   │   ├── dag.go            # DAG construction & validation
│   │   │   ├── execution.go      # Execution state machine
│   │   │   └── result.go         # Result value objects
│   │   ├── provider/
│   │   │   ├── model.go          # Model metadata
│   │   │   ├── tier.go           # AgentTier enum
│   │   │   └── cost.go           # Cost calculations
│   │   ├── metrics/
│   │   │   ├── cost.go           # Cost tracking entities
│   │   │   └── outcome.go        # Outcome tracking (v1.2)
│   │   ├── team/                 # v1.4
│   │   │   └── team.go
│   │   └── errors/
│   │       └── errors.go         # Domain errors
│   │
│   ├── application/              # USE CASES
│   │   ├── skill/
│   │   │   ├── run.go            # Execute skill use case
│   │   │   ├── list.go           # List skills use case
│   │   │   └── import.go         # Import skill use case
│   │   ├── workflow/
│   │   │   ├── execute.go        # Phase execution
│   │   │   └── optimize.go       # Auto-optimization (v1.2)
│   │   ├── provider/
│   │   │   ├── resolve.go        # Model resolution
│   │   │   └── health.go         # Health checking
│   │   ├── metrics/
│   │   │   └── track.go          # Cost tracking
│   │   └── ports/                # INTERFACE DEFINITIONS
│   │       ├── provider.go       # LLM provider interface
│   │       ├── storage.go        # Persistence interface
│   │       ├── cache.go          # Caching interface
│   │       ├── secrets.go        # Secrets interface (v1.1)
│   │       └── events.go         # Event publishing
│   │
│   ├── adapters/                 # EXTERNAL INTEGRATIONS
│   │   ├── provider/
│   │   │   ├── ollama/
│   │   │   │   ├── client.go
│   │   │   │   └── provider.go
│   │   │   ├── anthropic/
│   │   │   │   ├── client.go
│   │   │   │   └── provider.go
│   │   │   ├── openai/
│   │   │   │   ├── client.go
│   │   │   │   └── provider.go
│   │   │   ├── groq/
│   │   │   │   ├── client.go
│   │   │   │   └── provider.go
│   │   │   └── mock/
│   │   │       └── provider.go   # Test doubles
│   │   ├── storage/
│   │   │   ├── sqlite/           # v1.2
│   │   │   │   ├── migrations/
│   │   │   │   └── repository.go
│   │   │   ├── filesystem/
│   │   │   │   └── loader.go
│   │   │   └── memory/
│   │   │       └── cache.go
│   │   ├── security/             # v1.1
│   │   │   ├── keyring/
│   │   │   ├── encryption/
│   │   │   └── circuitbreaker/
│   │   ├── importer/
│   │   │   ├── git.go
│   │   │   ├── web.go
│   │   │   ├── claude.go         # Claude skill format
│   │   │   └── mcp.go            # v1.3
│   │   └── http/                 # v1.4
│   │       ├── router.go
│   │       ├── handlers/
│   │       └── middleware/
│   │
│   ├── infrastructure/           # CROSS-CUTTING
│   │   ├── config/
│   │   │   ├── config.go
│   │   │   └── loader.go
│   │   ├── logging/
│   │   │   └── slog.go
│   │   ├── di/
│   │   │   └── container.go
│   │   └── testutil/
│   │       ├── fixtures/
│   │       ├── mocks/
│   │       └── helpers.go
│   │
│   └── presentation/             # UI LAYER
│       ├── cli/
│       │   ├── commands/
│       │   │   ├── run.go
│       │   │   ├── ask.go
│       │   │   ├── list.go
│       │   │   ├── import.go
│       │   │   ├── status.go
│       │   │   ├── metrics.go
│       │   │   └── config.go
│       │   └── formatter/
│       │       ├── table.go
│       │       └── progress.go
│       └── web/                  # v1.3
│           ├── static/
│           └── server.go
│
├── pkg/                          # PUBLIC SDK
│   ├── sdk/
│   │   └── client.go
│   └── protocol/
│       └── mcp.go
│
├── configs/
│   ├── config.example.yaml
│   └── models.example.yaml
│
├── docs/
│   ├── architecture.md
│   └── adr/                      # Architecture Decision Records
│
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml
└── .goreleaser.yml
```

---

## 4. Core Interfaces

### ProviderPort (Primary Interface)

```go
// internal/application/ports/provider.go

type ProviderPort interface {
    // Info returns provider metadata
    Info() ProviderInfo

    // ListModels returns available models
    ListModels(ctx context.Context) ([]Model, error)

    // SupportsModel checks if model is available
    SupportsModel(ctx context.Context, modelID string) (bool, error)

    // IsAvailable checks provider/model reachability
    IsAvailable(ctx context.Context, modelID string) (bool, error)

    // Complete sends a completion request
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

    // Stream sends a streaming completion request
    Stream(ctx context.Context, req CompletionRequest, cb StreamCallback) (*CompletionResponse, error)

    // HealthCheck returns detailed health with remediation
    HealthCheck(ctx context.Context, modelID string) (*HealthStatus, error)
}

type CompletionRequest struct {
    ModelID      string
    Messages     []Message
    MaxTokens    int
    Temperature  float32
    SystemPrompt string
}

type CompletionResponse struct {
    Content      string
    InputTokens  int
    OutputTokens int
    FinishReason string
    ModelUsed    string
    Duration     time.Duration
}
```

### StoragePort

```go
// internal/application/ports/storage.go

type MetricsStoragePort interface {
    SaveExecution(ctx context.Context, exec *Execution) error
    GetExecutionsBySkill(ctx context.Context, skillID string, limit int) ([]Execution, error)
    GetCostSummary(ctx context.Context, filter CostFilter) (*CostSummary, error)
    GetOutcomeStats(ctx context.Context, skillID string) (*OutcomeStats, error)
}

type SkillLoaderPort interface {
    Load(ctx context.Context, skillID string) (*Skill, error)
    List(ctx context.Context) ([]SkillSummary, error)
    Exists(ctx context.Context, skillID string) (bool, error)
    Refresh(ctx context.Context) error
}

type CachePort interface {
    Get(ctx context.Context, key string) (interface{}, bool)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

### SecretStorePort (v1.1)

```go
// internal/application/ports/secrets.go

type SecretStorePort interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string) error
    Delete(ctx context.Context, key string) error
    List(ctx context.Context) ([]string, error)
}
```

---

## 5. Domain Entities

### Skill Aggregate

```go
// internal/domain/skill/skill.go

type Skill struct {
    id          string
    name        string
    version     string
    description string
    phases      []Phase
    routing     RoutingConfig
    metadata    map[string]interface{}
}

func NewSkill(id, name, version string, phases []Phase) (*Skill, error) {
    // Validation: id required, name required, phases non-empty
}

type Phase struct {
    ID             string
    Name           string
    PromptTemplate string
    RoutingProfile string
    DependsOn      []string
    MaxTokens      int
    Temperature    float32
}

type RoutingConfig struct {
    DefaultProfile   string
    GenerationModel  string
    ReviewModel      string
    FallbackModel    string
    MaxContextTokens int
}
```

### DAG (Value Object)

```go
// internal/domain/workflow/dag.go

type DAG struct {
    nodes map[string]*Node
    edges map[string][]string
}

func NewDAG(phases []Phase) (*DAG, error) {
    // Build nodes, validate dependencies exist, detect cycles
}

func (d *DAG) TopologicalSort() ([]string, error)
func (d *DAG) GetParallelBatches() ([][]string, error)
```

### Domain Errors

```go
// internal/domain/errors/errors.go

var (
    ErrSkillNotFound       = errors.New("skill not found")
    ErrSkillIDRequired     = errors.New("skill ID required")
    ErrNoPhasesDefinied    = errors.New("at least one phase required")
    ErrCycleDetected       = errors.New("cycle in phase dependencies")
    ErrModelUnavailable    = errors.New("model unavailable")
    ErrProviderUnreachable = errors.New("provider unreachable")
    ErrContextTooLarge     = errors.New("context exceeds max tokens")
)

type SkillrunnerError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Context map[string]interface{}
}
```

---

## 6. Testing Strategy

### Testing Pyramid

```
        /\
       /  \
      / E2E\           5% - Full CLI workflows
     /------\
    /        \
   /Integration\       20% - Provider + storage integration
  /--------------\
 /                \
/   Unit Tests     \   75% - Domain logic, use cases with mocks
/--------------------\
```

### Coverage Targets

| Layer | Target | Priority |
|-------|--------|----------|
| Domain | 100% | Critical |
| Application | 95% | Critical |
| Adapters | 85% | High |
| Presentation | 75% | Medium |
| **Overall** | **80%+** | Required |

### Test Organization

```
internal/
├── domain/
│   └── workflow/
│       ├── dag.go
│       └── dag_test.go           # Unit tests (pure logic)
├── adapters/
│   └── provider/
│       └── ollama/
│           ├── provider.go
│           ├── provider_test.go      # Unit tests with httptest
│           └── provider_live_test.go # Integration (build tag: integration)
└── testutil/
    ├── fixtures.go               # Test factories
    ├── mocks.go                  # Hand-written mocks
    └── helpers.go                # Test utilities

tests/
├── e2e/                          # Build tag: e2e
│   └── cli_test.go
└── contract/                     # Build tag: contract
    └── provider_contract_test.go
```

### TDD Workflow

1. **Write failing test** for the smallest unit of behavior
2. **Implement minimum code** to make test pass
3. **Refactor** while keeping tests green
4. **Repeat** for next behavior

### Mock Strategy

| Component | Mock Approach |
|-----------|---------------|
| HTTP APIs | `httptest.Server` |
| Providers | Interface mocks |
| SQLite | In-memory SQLite |
| Filesystem | `t.TempDir()` |
| Time | Injectable clock |

---

## 7. Implementation Roadmap

### Version 1.0: Core Launch (8 Sprints)

#### Sprint 1: Foundation Types
- Core types (Skill, Phase, Envelope)
- Provider interface definition
- Error types package
- **Tests:** Type serialization, interface compliance

#### Sprint 2: Ollama Provider
- HTTP client with httptest mocks
- Chat completion + streaming
- Model listing
- **Tests:** All API interactions mocked

#### Sprint 3: Anthropic Provider
- Messages API implementation
- Token usage tracking
- Rate limit handling
- **Tests:** Response parsing, error scenarios

#### Sprint 4: OpenAI & Groq Providers
- OpenAI provider (same pattern as Anthropic)
- Groq provider (same pattern)
- Provider factory + registry
- **Tests:** Factory creation, provider lookup

#### Sprint 5: DAG & Workflow Engine
- DAG construction from phases
- Cycle detection (DFS)
- Topological sort (Kahn's algorithm)
- Parallel batch generation
- Phase executor
- **Tests:** DAG validation, execution order

#### Sprint 6: Profile-Based Routing
- Routing config YAML loader
- Profile router (cheap/balanced/premium)
- Cost simulation
- **Tests:** Model selection, cost calculation

#### Sprint 7: CLI Commands Part 1
- Cobra CLI setup
- `run` command
- `list` command
- `status` command
- **Tests:** CLI output, error handling

#### Sprint 8: CLI Commands Part 2 + Demo Skills
- `ask`, `import`, `init`, `metrics` commands
- Web/Git/Marketplace import
- Demo skills: code-review, test-gen, doc-gen
- **Tests:** Import sources, skill execution

### Version 1.1: Security & Reliability (4 Sprints)

#### Sprint 9: AES-GCM Encryption
- Machine-specific key derivation
- Encrypt/decrypt API keys
- Secure storage integration
- Migration from plaintext
- **Tests:** Roundtrip encryption, key derivation

#### Sprint 10: Circuit Breakers
- State machine (closed → open → half-open)
- Failure counting + thresholds
- Per-provider configuration
- **Tests:** State transitions, recovery

#### Sprint 11: Structured Logging
- slog logger setup
- Contextual logging
- Request/response logging (sanitized)
- **Tests:** Log output, context propagation

#### Sprint 12: Health Checks + Coverage
- Provider health endpoints
- System health aggregation
- Gap analysis + test completion
- **Target:** 80% coverage achieved

### Version 1.2: Learning & Optimization (4 Sprints)

#### Sprint 13: SQLite Storage
- Connection management
- Schema migrations
- Execution/phase/model tables
- **Tests:** CRUD operations, migrations

#### Sprint 14: Outcome Tracking
- Phase success/failure recording
- Retry counting
- Aggregation queries
- **Tests:** Classification logic

#### Sprint 15: Model Performance Stats
- Per-model success rates
- Latency + cost analysis
- Historical trends
- **Tests:** Aggregation accuracy

#### Sprint 16: Auto-Optimization
- Optimization algorithm
- Phase-model pairing suggestions
- `sr optimize` command
- **Tests:** Recommendation quality

### Version 1.3: Insights & Intelligence (4 Sprints)

#### Sprint 17: Local Web Dashboard
- Embedded HTTP server
- API endpoints
- Dashboard UI (HTML/JS)
- **Tests:** API responses, E2E browser

#### Sprint 18: Intelligent Skill Decomposition
- Task complexity analysis (LLM-powered)
- Phase recommendation
- Integration with run command
- **Tests:** Structured output parsing

#### Sprint 19: MCP Protocol Support
- MCP client implementation
- MCP server implementation
- Tool + resource registration
- **Tests:** Protocol compliance

#### Sprint 20: Workflow Comparison
- Comparison queries
- Model leaderboard
- Export functionality
- **Tests:** Ranking algorithms

### Version 1.4: Teams & API (4 Sprints)

#### Sprint 21: REST API
- Authentication (JWT)
- Workflow execution endpoint
- Status + metrics endpoints
- **Tests:** Auth flows, request handling

#### Sprint 22: Team Sync
- Workflow serialization
- Sync protocol
- Conflict resolution
- **Tests:** Merge scenarios

#### Sprint 23: Budget Alerts
- Benchmark aggregation
- Budget configuration
- Alert notifications
- **Tests:** Threshold triggering

#### Sprint 24: Paid Tier Support
- License validation
- Feature gating
- Usage metering
- **Tests:** Gate enforcement

---

## 8. External Dependencies

### Core (Minimal)

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Configuration |
| `gopkg.in/yaml.v3` | YAML parsing |
| `log/slog` | Structured logging (stdlib) |

### Testing

| Package | Purpose |
|---------|---------|
| `github.com/stretchr/testify` | Assertions + mocks |
| `github.com/jarcoal/httpmock` | HTTP mocking |

### v1.1 (Security)

| Package | Purpose |
|---------|---------|
| `golang.org/x/crypto` | AES-GCM |
| `github.com/99designs/keyring` | OS keyring |
| `github.com/sony/gobreaker` | Circuit breaker |

### v1.2 (Storage)

| Package | Purpose |
|---------|---------|
| `modernc.org/sqlite` | Pure Go SQLite |

### v1.3+ (Web/API)

| Package | Purpose |
|---------|---------|
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/golang-jwt/jwt/v5` | JWT auth |

---

## 9. Configuration

### User Config (`~/.skillrunner/config.yaml`)

```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
  anthropic:
    api_key_encrypted: <encrypted>
    enabled: true
  openai:
    api_key_encrypted: <encrypted>
    enabled: true
  groq:
    api_key_encrypted: <encrypted>
    enabled: true

routing:
  default_profile: balanced

logging:
  level: info
  format: json
```

### Models Config (`config/models.yaml`)

```yaml
routing_profiles:
  cheap:
    candidates:
      - provider: ollama
        model: qwen2.5:7b
        priority: 1
      - provider: anthropic
        model: claude-3-haiku
        priority: 2
  balanced:
    candidates:
      - provider: ollama
        model: qwen2.5:14b
        priority: 1
      - provider: anthropic
        model: claude-3-5-sonnet
        priority: 2
  premium:
    candidates:
      - provider: anthropic
        model: claude-3-opus
        priority: 1

cost_simulation:
  premium_model: anthropic/claude-3-opus
  cheap_model: ollama/qwen2.5:7b
```

---

## 10. Security Requirements

### v1.1 Security Features

1. **API Key Encryption**
   - AES-256-GCM encryption
   - Machine-specific key derivation (hostname + user)
   - Never log or display keys

2. **Circuit Breakers**
   - Prevent cascade failures
   - Configurable thresholds
   - Automatic recovery

3. **Input Validation**
   - Sanitize all user input
   - Validate skill YAML before execution
   - Escape template variables

4. **Secure Defaults**
   - HTTPS for cloud providers
   - Minimal permissions
   - No credential caching in memory

---

## Implementation Notes

### Critical Design Decisions

1. **Domain-First**: Build domain layer first with 100% test coverage before any adapters
2. **Interface Segregation**: Small, focused interfaces (ProviderPort, StoragePort)
3. **Dependency Injection**: All dependencies injected, never created internally
4. **Error Wrapping**: Use `fmt.Errorf("%w", err)` for error chains
5. **Context Propagation**: Pass `context.Context` through all layers

### CLI Commands (v1.0)

```bash
sr run <skill> <request>     # Execute multi-phase workflow
sr ask <skill> <question>    # Quick single-phase query
sr list                      # Show available skills
sr import <source>           # Import from web/git/marketplace
sr status                    # System health check
sr metrics                   # Usage and cost metrics
sr init                      # Initialize configuration
sr config get/set            # Manage configuration
```

### Success Metrics

- **v1.0**: Working CLI with all 4 providers, 75% test coverage
- **v1.1**: 80% test coverage, encrypted secrets, circuit breakers
- **v1.2**: Outcome tracking, optimization recommendations
- **v1.3**: Web dashboard, MCP support
- **v1.4**: REST API, team features, monetization

---

## References

- **Current Codebase**: ~29,575 LOC across 22 packages
- **Linear Issues**: JBC-650 (encryption), JBC-655 (circuit breakers), JBC-667 (SQLite), JBC-702 (decomposition), JBC-691 (MCP)
- **Notion**: Skillrunner product page, Product Portfolio Overview
- **Launch Date**: December 14, 2025 (v1.0)
