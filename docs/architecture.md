# Skillrunner Architecture Overview

**Version:** 2.0
**Last Updated:** 2025-12-26

## Table of Contents

1. [Introduction](#introduction)
2. [Hexagonal Architecture](#hexagonal-architecture)
3. [Directory Structure](#directory-structure)
4. [Domain Model](#domain-model)
5. [Provider System](#provider-system)
6. [Workflow Execution](#workflow-execution)
7. [Key Design Decisions](#key-design-decisions)
8. [Extension Points](#extension-points)

---

## Introduction

Skillrunner is a **local-first, multi-provider LLM orchestration system** that executes complex AI workflows through a structured skill framework. It uses **Hexagonal Architecture** (Ports & Adapters) to maintain clean boundaries between business logic and external dependencies.

### Core Concepts

- **Skill**: A multi-phase workflow definition with routing configuration
- **Phase**: A discrete execution step with dependencies and LLM configuration
- **Provider**: An LLM service adapter (Anthropic, OpenAI, Groq, Ollama)
- **DAG**: Directed Acyclic Graph for dependency-aware phase execution
- **Router**: Intelligent model selection based on profiles (cheap/balanced/premium)

---

## Hexagonal Architecture

Skillrunner implements the **Ports & Adapters pattern** to achieve testability, maintainability, and independence from external systems.

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                       │
│                         (CLI)                               │
└─────────────────┬───────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────────┐
│                   APPLICATION LAYER                         │
│                                                             │
│  ┌──────────────────────────────────────────────────┐      │
│  │              Service Orchestration                │      │
│  │  • Workflow Executor                              │      │
│  │  • Phase Executor                                 │      │
│  │  • Provider Router                                │      │
│  │  • Resolver (fallback logic)                      │      │
│  └──────────────────────────────────────────────────┘      │
│                                                             │
│  ┌──────────────────────────────────────────────────┐      │
│  │                    PORTS                          │      │
│  │  • ProviderPort (LLM interface)                   │      │
│  │  • CachePort (future)                             │      │
│  │  • StoragePort (future)                           │      │
│  └──────────────────────────────────────────────────┘      │
└─────────────────┬───────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────────┐
│                    DOMAIN LAYER                             │
│                 (Pure Business Logic)                       │
│                                                             │
│  • Skill aggregate (validation, invariants)                │
│  • Phase value object (immutable configuration)            │
│  • Workflow DAG (graph algorithms)                         │
│  • RoutingConfig (cost-aware routing)                      │
│  • Domain Errors (typed error hierarchy)                   │
└─────────────────────────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────────┐
│                    ADAPTER LAYER                            │
│                                                             │
│  ┌────────────┬────────────┬────────────┬────────────┐     │
│  │ Anthropic  │  OpenAI    │   Groq     │  Ollama    │     │
│  │  Provider  │  Provider  │  Provider  │  Provider  │     │
│  └────────────┴────────────┴────────────┴────────────┘     │
│                                                             │
│  • Provider Registry (thread-safe)                         │
│  • HTTP clients (provider-specific)                        │
│  • Response mapping (port compliance)                      │
└─────────────────────────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────────┐
│                 INFRASTRUCTURE LAYER                        │
│                                                             │
│  • Configuration (YAML/env loading)                         │
│  • Skill loading (filesystem)                              │
│  • Test utilities                                          │
└─────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

#### 1. Domain Layer (`internal/domain/`)
- **Pure business logic** - no external dependencies
- Contains aggregates, entities, and value objects
- Enforces invariants and business rules
- Independent of frameworks, databases, or external APIs

#### 2. Application Layer (`internal/application/`)
- **Orchestration services** - coordinates domain objects
- Defines **ports** (interfaces) for external dependencies
- Implements use cases (workflow execution, phase execution)
- Router and resolver for intelligent model selection

#### 3. Adapter Layer (`internal/adapters/`)
- **External integrations** - implements ports
- Provider adapters for Anthropic, OpenAI, Groq, Ollama
- Registry for provider discovery and management
- Converts between external APIs and internal ports

#### 4. Infrastructure Layer (`internal/infrastructure/`)
- **Configuration management** - YAML, env vars, defaults
- Skill loading from filesystem
- Utility functions and test helpers

#### 5. Presentation Layer (`internal/presentation/`)
- **CLI interface** - user-facing commands
- Output formatters (JSON, text, tables)
- Command handlers and flags

---

## Directory Structure

```
internal/
├── domain/                    # Business logic (no external dependencies)
│   ├── skill/                # Skill aggregate and value objects
│   │   ├── skill.go          # Skill aggregate root
│   │   ├── phase.go          # Phase value object
│   │   └── routing.go        # RoutingConfig
│   ├── workflow/             # Workflow orchestration types
│   │   └── dag.go            # DAG for phase dependency resolution
│   ├── errors/               # Domain-specific errors
│   │   └── errors.go         # Typed error hierarchy
│   └── provider/             # Provider domain types
│
├── application/              # Use cases and orchestration
│   ├── ports/                # Interfaces for external systems
│   │   ├── provider.go       # ProviderPort interface
│   │   ├── cache.go          # CachePort (future)
│   │   └── storage.go        # StoragePort (future)
│   ├── workflow/             # Workflow execution services
│   │   ├── executor.go       # DAG-based skill executor
│   │   └── phase_executor.go # Single phase executor
│   └── provider/             # Provider routing and resolution
│       ├── router.go         # Profile-based model selection
│       └── resolver.go       # Fallback and health checks
│
├── adapters/                 # External integrations
│   └── provider/             # LLM provider implementations
│       ├── registry.go       # Thread-safe provider registry
│       ├── anthropic/        # Anthropic adapter
│       │   ├── provider.go   # ProviderPort implementation
│       │   ├── client.go     # HTTP client
│       │   └── types.go      # Request/response types
│       ├── openai/           # OpenAI adapter
│       ├── groq/             # Groq adapter
│       └── ollama/           # Ollama adapter (local)
│
├── infrastructure/           # Cross-cutting concerns
│   ├── config/               # Configuration management
│   │   └── routing.go        # Routing config (YAML support)
│   ├── skills/               # Skill loading from filesystem
│   └── testutil/             # Test helpers and mocks
│
└── presentation/             # User interfaces
    └── cli/                  # Command-line interface
        ├── commands/         # CLI command implementations
        └── output/           # Output formatters
```

---

## Domain Model

The domain layer contains the core business logic with zero external dependencies.

### Skill Aggregate

The **Skill** is the aggregate root representing a complete workflow.

```go
type Skill struct {
    id          string
    name        string
    version     string
    description string
    phases      []Phase         // Ordered execution phases
    routing     RoutingConfig   // Model selection config
    metadata    map[string]any  // Extensible metadata
}
```

**Key responsibilities:**
- **Validation** - ensures all phases are valid and dependencies exist
- **Cycle detection** - prevents circular phase dependencies
- **Invariant enforcement** - maintains aggregate consistency
- **Encapsulation** - defensive copying to prevent external mutation

**Domain rules:**
- ID and name are required
- Must have at least one phase
- All phase dependencies must reference existing phases
- No cycles allowed in dependency graph

### Phase Value Object

The **Phase** represents a single execution step with its configuration.

```go
type Phase struct {
    ID             string
    Name           string
    PromptTemplate string   // Go template with dependency interpolation
    RoutingProfile string   // cheap | balanced | premium
    DependsOn      []string // Phase IDs this depends on
    MaxTokens      int
    Temperature    float32
}
```

**Routing profiles:**
- `cheap` - Cost-optimized models (local preferred)
- `balanced` - Balance of cost and capability (default)
- `premium` - Highest capability models (cloud)

**Template system:**
```go
// Access dependency outputs:
{{._input}}              // Original skill input
{{.phase_id}}            // Output from dependency phase
{{index . "phase-id"}}   // For IDs with special characters
```

### Workflow DAG

The **DAG** (Directed Acyclic Graph) manages phase dependency resolution.

```go
type DAG struct {
    nodes map[string]*Node     // phase ID -> node
    edges map[string][]string  // phase ID -> dependent phase IDs
}

type Node struct {
    Phase    *skill.Phase
    InDegree int      // Number of dependencies
    OutEdges []string // Dependents
}
```

**Key algorithms:**
1. **Topological Sort** - Sequential execution order (Kahn's algorithm)
2. **Parallel Batches** - Groups phases for concurrent execution
3. **Cycle Detection** - DFS-based cycle detection during construction

**Parallel execution example:**
```
Phase dependencies:
  A -> (no deps)
  B -> (no deps)
  C -> depends on A
  D -> depends on A, B
  E -> depends on C

Parallel batches:
  Batch 1: [A, B]       # Run in parallel
  Batch 2: [C]          # Wait for A
  Batch 3: [D]          # Wait for A, B
  Batch 4: [E]          # Wait for C
```

### RoutingConfig

Model routing configuration for cost-aware execution.

```go
type RoutingConfig struct {
    DefaultProfile   string // cheap | balanced | premium
    GenerationModel  string // Model for generation phases
    ReviewModel      string // Model for review phases
    FallbackModel    string // Fallback when primary unavailable
    MaxContextTokens int
}
```

---

## Provider System

The provider system enables **multi-provider LLM access** with intelligent routing and failover.

### ProviderPort Interface

All providers implement this port (interface):

```go
type ProviderPort interface {
    Info() ProviderInfo
    ListModels(ctx context.Context) ([]string, error)
    SupportsModel(ctx context.Context, modelID string) (bool, error)
    IsAvailable(ctx context.Context, modelID string) (bool, error)
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest, cb StreamCallback) (*CompletionResponse, error)
    HealthCheck(ctx context.Context, modelID string) (*HealthStatus, error)
}
```

**Design benefits:**
- **Testability** - Easy to mock for unit tests
- **Swappability** - Add/remove providers without changing core logic
- **Consistency** - Uniform interface regardless of provider API

### Provider Implementations

Each provider is an adapter implementing `ProviderPort`:

```
┌──────────────────────────────────────────────────────┐
│              ProviderPort (Interface)                │
└──────────────────┬───────────────────────────────────┘
                   │
        ┌──────────┴──────────┬──────────┬──────────┐
        │                     │          │          │
   ┌────▼────┐         ┌──────▼───┐ ┌───▼────┐ ┌──▼─────┐
   │Anthropic│         │  OpenAI  │ │  Groq  │ │ Ollama │
   │Provider │         │ Provider │ │Provider│ │Provider│
   └────┬────┘         └──────┬───┘ └───┬────┘ └──┬─────┘
        │                     │          │          │
   ┌────▼────┐         ┌──────▼───┐ ┌───▼────┐ ┌──▼─────┐
   │Anthropic│         │  OpenAI  │ │  Groq  │ │ Ollama │
   │  Client │         │  Client  │ │ Client │ │ Client │
   └─────────┘         └──────────┘ └────────┘ └────────┘
```

**Provider characteristics:**

| Provider   | Type   | Use Case                    | IsLocal |
|------------|--------|-----------------------------|---------|
| Ollama     | Local  | Free, private, fast         | Yes     |
| Groq       | Cloud  | High-speed inference        | No      |
| OpenAI     | Cloud  | GPT models                  | No      |
| Anthropic  | Cloud  | Claude models               | No      |

### Registry Pattern

The **Registry** provides thread-safe provider management:

```go
type Registry struct {
    mu        sync.RWMutex
    providers map[string]ports.ProviderPort
    order     []string // Registration order
}
```

**Key methods:**
- `Register(provider)` - Add a provider
- `Get(name)` - Retrieve by name
- `FindByModel(ctx, modelID)` - Find provider supporting a model
- `FindAvailable(ctx)` - Get all healthy providers
- `GetLocalProviders()` - Filter for local providers
- `GetCloudProviders()` - Filter for cloud providers

**Thread-safety:** All methods use read/write locks for concurrent access.

### Router and Model Selection

The **Router** implements intelligent model selection:

```go
type Router struct {
    config   *config.RoutingConfiguration
    registry *adapterProvider.Registry
}
```

**Selection algorithm:**
1. Check profile configuration (cheap/balanced/premium)
2. Determine phase type (generation vs review)
3. Select appropriate model from profile
4. Verify model availability via registry
5. Fall back to next option if unavailable
6. Walk fallback chain if all primary models fail

**Fallback chain:**
```
Primary model (from profile)
  ↓ (unavailable)
Profile fallback model
  ↓ (unavailable)
Fallback chain providers (in order):
  - Ollama (local, free)
  - Groq (fast cloud)
  - OpenAI (reliable cloud)
  - Anthropic (premium cloud)
```

**Cost-aware routing:**
- `cheap` profile → Local models preferred (Ollama)
- `balanced` profile → Mid-tier models (Groq, small cloud models)
- `premium` profile → Best models (Claude 3.5, GPT-4)

---

## Workflow Execution

The workflow executor orchestrates DAG-based skill execution with parallel optimization.

### Execution Flow

```
┌─────────────────────────────────────────────────────────────┐
│                   Skill Execution                           │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
         ┌────────────────┐
         │ Validate Skill │
         └────────┬───────┘
                  │
                  ▼
         ┌────────────────┐
         │   Build DAG    │ (Check cycles, dependencies)
         └────────┬───────┘
                  │
                  ▼
      ┌──────────────────────┐
      │ Get Parallel Batches │ (Group independent phases)
      └──────────┬───────────┘
                 │
                 ▼
      ┌──────────────────────┐
      │ For each batch:      │
      │  - Execute phases    │ (Parallel, semaphore-limited)
      │    in parallel       │
      │  - Gather outputs    │
      │  - Check errors      │
      └──────────┬───────────┘
                 │
                 ▼
      ┌──────────────────────┐
      │ Determine final      │ (From terminal phases)
      │ output               │
      └──────────┬───────────┘
                 │
                 ▼
      ┌──────────────────────┐
      │ Return execution     │
      │ result               │
      └──────────────────────┘
```

### Executor Configuration

```go
type ExecutorConfig struct {
    MaxParallel int           // Max concurrent phases (default: 4)
    Timeout     time.Duration // Overall timeout (default: 5m)
}
```

### Phase Execution

Each phase is executed by the `phaseExecutor`:

1. **Build prompt** - Render template with dependency outputs
2. **Select model** - Use router to choose model based on profile
3. **Build messages** - Add context from dependencies
4. **Call provider** - Execute LLM request
5. **Return result** - Capture output, tokens, errors

**Context passing:**
```go
dependencyOutputs := map[string]string{
    "_input":    "Original skill input",
    "phase_a":   "Output from phase A",
    "phase_b":   "Output from phase B",
}

// Template rendering:
prompt := renderTemplate(phase.PromptTemplate, dependencyOutputs)

// Context message building:
messages := []Message{
    {Role: "system", Content: "Context from previous phases..."},
    {Role: "user", Content: prompt},
}
```

### Parallel Execution

Phases in the same batch execute concurrently with semaphore-based limiting:

```go
sem := make(chan struct{}, config.MaxParallel)

for _, phaseID := range batch {
    go func(p *skill.Phase) {
        // Acquire semaphore
        sem <- struct{}{}
        defer func() { <-sem }()

        // Execute phase
        result := phaseExecutor.Execute(ctx, p, dependencyOutputs)

        // Store result
        results[p.ID] = result
    }(phase)
}
```

**Benefits:**
- **Performance** - Independent phases run in parallel
- **Control** - Semaphore prevents resource exhaustion
- **Safety** - Mutex protects shared result map

### Result Aggregation

```go
type ExecutionResult struct {
    SkillID      string
    SkillName    string
    Status       PhaseStatus
    PhaseResults map[string]*PhaseResult
    FinalOutput  string                    // From terminal phases
    TotalTokens  int                       // Sum of all phase tokens
    Duration     time.Duration
}
```

**Terminal phase logic:**
- Find phases with no dependents
- If single terminal → return its output
- If multiple terminals → concatenate outputs

---

## Key Design Decisions

### 1. Local-First Philosophy

**Decision:** Prefer local models (Ollama) over cloud APIs when possible.

**Rationale:**
- Privacy - sensitive data never leaves the machine
- Cost - no API fees for local execution
- Availability - works offline
- Performance - low latency for local models

**Implementation:**
- Ollama is the default provider
- `PreferLocal` flag in profile configuration
- Fallback chain starts with local providers

### 2. Minimal Dependencies

**Decision:** Use only Go standard library where possible.

**Rationale:**
- Reliability - fewer third-party dependencies
- Security - smaller attack surface
- Simplicity - easier to audit and understand
- Performance - no unnecessary abstractions

**Examples:**
- HTTP clients - `net/http` from stdlib
- JSON parsing - `encoding/json` from stdlib
- Templates - `text/template` from stdlib

### 3. Thread-Safe Components

**Decision:** All shared components use proper synchronization.

**Rationale:**
- Correctness - prevent race conditions
- Concurrency - support parallel execution
- Safety - predictable behavior under load

**Thread-safe components:**
- Provider Registry (`sync.RWMutex`)
- Router configuration updates
- Shared result maps in executor

### 4. Cost-Aware Routing

**Decision:** Profile-based routing with cost tiers (cheap/balanced/premium).

**Rationale:**
- Flexibility - users control cost vs quality tradeoff
- Transparency - explicit about model selection
- Optimization - use expensive models only when needed

**Cost configuration:**
```yaml
models:
  claude-3-5-sonnet-20241022:
    tier: premium
    cost_per_input_token: 0.000003
    cost_per_output_token: 0.000015
```

### 5. Dependency Injection

**Decision:** Use ports (interfaces) and inject dependencies.

**Rationale:**
- Testability - easy to mock external systems
- Flexibility - swap implementations without changing code
- Decoupling - domain independent of adapters

**Example:**
```go
// Application defines the port
type ProviderPort interface { ... }

// Adapter implements the port
type OllamaProvider struct { ... }

// Inject via constructor
executor := NewExecutor(provider, config)
```

### 6. Immutable Value Objects

**Decision:** Phase and routing config are immutable after creation.

**Rationale:**
- Safety - prevent accidental mutations
- Clarity - configuration set once at creation
- Thread-safety - safe to share across goroutines

**Pattern:**
```go
phase := NewPhase(id, name, prompt).
    WithRoutingProfile("premium").
    WithDependencies([]string{"phase_a"}).
    WithMaxTokens(2048)
```

### 7. Error Wrapping and Context

**Decision:** Use typed errors with context for debugging.

**Rationale:**
- Debugging - context helps trace issues
- Handling - typed errors enable conditional logic
- User experience - clear error messages

**Implementation:**
```go
type SkillrunnerError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Context map[string]interface{}
}

err := NewError(CodeValidation, "invalid phase", ErrPhaseNotFound)
err = WithContext(err, "phase_id", "review")
```

---

## Extension Points

The architecture is designed for extensibility. Here's how to add new functionality:

### Adding a New Provider

1. **Create provider package:**
   ```
   internal/adapters/provider/newprovider/
   ├── provider.go  # Implements ProviderPort
   ├── client.go    # HTTP client
   └── types.go     # Request/response types
   ```

2. **Implement ProviderPort:**
   ```go
   type Provider struct {
       client *Client
   }

   func (p *Provider) Info() ports.ProviderInfo { ... }
   func (p *Provider) Complete(ctx, req) (*ports.CompletionResponse, error) { ... }
   // ... implement all methods
   ```

3. **Register in main:**
   ```go
   provider := newprovider.NewProvider()
   registry.Register(provider)
   ```

4. **Add to configuration:**
   ```yaml
   providers:
     newprovider:
       enabled: true
       priority: 10
       models:
         model-id:
           tier: balanced
           enabled: true
   ```

### Adding a New Routing Strategy

1. **Extend Router:**
   ```go
   func (r *Router) SelectModelWithCustomStrategy(
       ctx context.Context,
       strategy string,
       criteria ...interface{},
   ) (*ModelSelection, error) {
       // Implement custom selection logic
   }
   ```

2. **Add configuration:**
   ```yaml
   profiles:
     custom:
       strategy: latency-optimized
       max_latency_ms: 100
   ```

### Adding a New Port

1. **Define interface in `application/ports/`:**
   ```go
   type CachePort interface {
       Get(ctx context.Context, key string) ([]byte, error)
       Set(ctx context.Context, key string, value []byte) error
   }
   ```

2. **Create adapter in `adapters/`:**
   ```go
   type RedisCache struct { ... }
   func (c *RedisCache) Get(ctx, key) ([]byte, error) { ... }
   ```

3. **Inject in application layer:**
   ```go
   type CachedExecutor struct {
       cache    ports.CachePort
       executor Executor
   }
   ```

### Adding Phase Capabilities

1. **Extend Phase configuration:**
   ```go
   type Phase struct {
       // ... existing fields
       Capabilities []string  // e.g., ["vision", "function_calling"]
   }
   ```

2. **Update router selection:**
   ```go
   func (r *Router) SelectModelWithCapabilities(
       ctx context.Context,
       profile string,
       capabilities []string,
   ) (*ModelSelection, error) {
       // Filter models by required capabilities
   }
   ```

3. **Configure in YAML:**
   ```yaml
   phases:
     - id: image_analysis
       capabilities: [vision]
   ```

### Adding Middleware/Interceptors

1. **Define middleware pattern:**
   ```go
   type ProviderMiddleware func(ports.ProviderPort) ports.ProviderPort

   type LoggingProvider struct {
       next ports.ProviderPort
   }

   func (p *LoggingProvider) Complete(ctx, req) (*ports.CompletionResponse, error) {
       log.Printf("Request: %+v", req)
       resp, err := p.next.Complete(ctx, req)
       log.Printf("Response: %+v", resp)
       return resp, err
   }
   ```

2. **Wrap providers:**
   ```go
   provider = WithLogging(provider)
   provider = WithMetrics(provider)
   provider = WithRateLimiting(provider)
   ```

### Adding Skill Loading Sources

1. **Implement loader interface:**
   ```go
   type SkillLoader interface {
       Load(ctx context.Context, id string) (*skill.Skill, error)
       List(ctx context.Context) ([]string, error)
   }
   ```

2. **Create implementation:**
   ```go
   type HTTPSkillLoader struct { ... }
   type S3SkillLoader struct { ... }
   type GitSkillLoader struct { ... }
   ```

---

## Testing Strategy

### Unit Tests
- **Domain layer** - Pure logic, no mocks needed
- **Application layer** - Mock ports with interfaces
- **Adapters** - Mock HTTP clients

### Integration Tests
- **E2E directory** - Full workflow tests with real providers
- **Provider tests** - Real API calls (optional, gated by env vars)

### Test Utilities
- **Mock providers** - In `infrastructure/testutil/`
- **Test fixtures** - Sample skills and configurations
- **Assertions** - Helpers for common validations

---

## Future Enhancements

### Planned Features
1. **Caching** - Cache LLM responses for repeated inputs
2. **Streaming** - Support for streaming execution results
3. **Observability** - Metrics, tracing, and structured logging
4. **Rate limiting** - Per-provider rate limit enforcement
5. **Retries** - Automatic retry with exponential backoff
6. **Cost tracking** - Real-time cost calculation and budgets
7. **Skill marketplace** - Share and discover skills

### Architectural Evolution
- **Plugin system** - Dynamic provider loading
- **WASM support** - Run skills in browser
- **Distributed execution** - Multi-node skill execution
- **Stateful workflows** - Long-running, resumable skills

---

## Glossary

- **Aggregate** - Domain-driven design pattern, cluster of objects treated as a unit
- **DAG** - Directed Acyclic Graph, graph with no cycles
- **Hexagonal Architecture** - Ports & Adapters pattern for clean architecture
- **Phase** - Single execution step in a skill workflow
- **Port** - Interface defining contract with external system
- **Provider** - LLM service adapter (Anthropic, OpenAI, Groq, Ollama)
- **Registry** - Central repository for provider instances
- **Router** - Component for intelligent model selection
- **Skill** - Complete workflow definition with multiple phases
- **Value Object** - Immutable domain object with no identity

---

## References

### Architecture Patterns
- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/) - Alistair Cockburn
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) - Robert C. Martin
- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html) - Eric Evans

### Go Best Practices
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)

### Project Resources
- [Skillrunner Repository](https://github.com/jbctechsolutions/skillrunner)
- [API Documentation](./api.md) (if exists)
- [Configuration Guide](./configuration.md) (if exists)

---

**Document Status:** Draft v1.0
**Contributors:** Architecture Team
**Last Review:** 2025-12-26
