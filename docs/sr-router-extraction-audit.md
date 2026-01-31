# SkillRunner Router Extraction Audit Report

## Executive Summary

SkillRunner is a **fully implemented, production-quality Go codebase** (~46,700 LOC across 195+ files) with a well-architected routing and cost optimization layer. The router component is **moderately extractable** but has meaningful coupling to SkillRunner's domain model that would require decoupling work.

**Bottom line**: The routing logic is solid and worth extracting, but it's ~2-3 weeks of focused work to create a truly standalone `sr-router` module.

---

## 1. Current State Analysis

### What's Built (Production-Ready)

| Component | Status | LOC | Test Coverage |
|-----------|--------|-----|---------------|
| **Router** | ✅ Complete | ~400 | Full |
| **Resolver** (Router + Cost) | ✅ Complete | ~220 | Full |
| **CostCalculator** | ✅ Complete | ~275 | Full |
| **Pricing Database** | ✅ Complete | ~190 | N/A |
| **Provider Interface** | ✅ Complete | ~70 | N/A |
| **Provider Registry** | ✅ Complete | ~200 | Full |
| **Provider Adapters** | ✅ Complete | ~5,100 | 85%+ |
| **Routing Config** | ✅ Complete | ~600 | Full |

### What's Scaffolded/Partial

- `internal/adapters/backend/opencode/` - experimental, has TODOs
- `internal/adapters/backend/aider/` - version detection TODO

### What's Planned Only

- Rate limiting enforcement (config exists, enforcement not implemented)
- Circuit breaker pattern
- Request queuing

---

## 2. Router Component Deep Dive

### Core Files

| File | Purpose | Lines |
|------|---------|-------|
| `internal/application/provider/router.go` | Profile-based model selection with fallback | 386 |
| `internal/application/provider/resolver.go` | Resolution + cost tracking facade | 223 |
| `internal/domain/provider/calculator.go` | Cost calculation engine | 275 |
| `internal/domain/provider/pricing.go` | 40+ model pricing database | 192 |
| `internal/domain/provider/cost.go` | Cost breakdown & summary types | ~150 |
| `internal/domain/provider/model.go` | Model domain type | ~120 |
| `internal/domain/provider/tier.go` | Tier definitions | ~50 |
| `internal/adapters/provider/registry.go` | Provider service discovery | 199 |
| `internal/application/ports/provider.go` | Provider interface | 72 |
| `internal/infrastructure/config/routing.go` | Routing configuration | 604 |

### Routing Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        Resolver.Resolve()                        │
│  (internal/application/provider/resolver.go:69)                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Router.SelectModel()                          │
│  (internal/application/provider/router.go:62)                    │
│                                                                  │
│  1. Validate profile (cheap/balanced/premium)                    │
│  2. Get profile config → generation model                        │
│  3. Check provider availability via Registry                     │
│  4. Fall back to fallback chain if unavailable                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Registry.FindByModel()                         │
│  (internal/adapters/provider/registry.go:132)                    │
│                                                                  │
│  1. Iterate providers in registration order                      │
│  2. Call provider.SupportsModel(modelID)                         │
│  3. Return first supporting provider                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                Provider.IsAvailable() / Complete()               │
│  (internal/adapters/provider/{anthropic,openai,groq,ollama}/)    │
└─────────────────────────────────────────────────────────────────┘
```

### Key Features Already Implemented

1. **3-tier profile system** (`router.go:320-327`)
   - `cheap` → local Ollama models
   - `balanced` → mid-tier local models
   - `premium` → cloud models (Claude, GPT-4o)

2. **Phase-aware routing** (`router.go:92-128`)
   - Automatic review phase detection by name patterns
   - Separate generation vs review model selection

3. **Capability-based selection** (`router.go:329-370`)
   - Filter models by capabilities (vision, function_calling)

4. **Fallback chain** (`router.go:194-245`)
   - Profile fallback → global fallback chain → any available model

5. **Cost tracking** (`resolver.go:137-161`, `calculator.go:119-156`)
   - Per-request cost breakdown
   - Aggregated cost summaries
   - Savings calculation vs premium baseline

6. **40+ model pricing** (`pricing.go:16-180`)
   - Anthropic Claude 3/3.5/4/4.5
   - OpenAI GPT-4o/4/3.5, O-series
   - Groq Llama/Mixtral/DeepSeek
   - Ollama local models (zero cost)

---

## 3. Dependency Analysis

### Direct Dependencies (Must Decouple)

| Import | Used By | Coupling Level |
|--------|---------|----------------|
| `internal/domain/skill` | Router, Config | **Medium** - profile constants, phase detection |
| `internal/adapters/provider` | Router, Resolver | **High** - Registry type |
| `internal/infrastructure/config` | Router, Resolver | **High** - RoutingConfiguration |
| `internal/application/ports` | Registry, Resolver | **Low** - just interface |

### Specific Coupling Points

**1. Skill Domain (`internal/domain/skill/routing.go`, `phase.go`)**
```go
// router.go:10-11
import "github.com/jbctechsolutions/skillrunner/internal/domain/skill"

// Usage: Profile constants
skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium

// Usage: Phase type for routing decisions
func SelectModelForPhase(ctx context.Context, phase *skill.Phase)
```

**2. Routing Configuration (`internal/infrastructure/config/routing.go`)**
```go
// router.go:39-40
config   *config.RoutingConfiguration
registry *adapterProvider.Registry
```

**3. Provider Registry (`internal/adapters/provider/registry.go`)**
```go
// Tight coupling to concrete Registry type
type Router struct {
    registry *adapterProvider.Registry  // Should be an interface
}
```

### What Would Need Decoupling

1. **Extract profile constants** to standalone package
2. **Define `ProviderRegistry` interface** instead of concrete type
3. **Define `Phase` interface** or use primitives for phase-aware routing
4. **Extract `RoutingConfiguration`** as standalone config package

---

## 4. Extractability Assessment

### Go Module (Native)

**Effort: Medium (2-3 weeks)**

```
sr-router/
├── router/           # Core routing logic
│   ├── router.go     # SelectModel, SelectModelForPhase
│   ├── resolver.go   # Resolution + cost tracking
│   └── config.go     # Routing configuration
├── cost/             # Cost calculation
│   ├── calculator.go
│   ├── pricing.go
│   └── types.go      # CostBreakdown, CostSummary
├── provider/         # Provider abstraction
│   ├── interface.go  # ProviderPort
│   ├── registry.go   # Registry (interface-based)
│   └── model.go      # Model metadata
└── adapters/         # Provider implementations
    ├── anthropic/
    ├── openai/
    ├── groq/
    └── ollama/
```

**Decoupling work needed:**
- [ ] Extract profile constants (trivial)
- [ ] Create `ProviderRegistry` interface (easy)
- [ ] Decouple phase-aware routing from skill.Phase (medium)
- [ ] Extract config types (medium)
- [ ] Create standalone go.mod (trivial)

### npm Package (for Moltbot)

**Effort: High (4-6 weeks)**

Options:
1. **WebAssembly (WASM)** - Compile Go to WASM, wrap with JS
2. **Native port** - Rewrite core logic in TypeScript
3. **HTTP microservice** - Deploy Go service, call from JS

**Recommended: Native TypeScript port** of core routing logic (~1,500 LOC to port) with the following structure:

```typescript
// sr-router/
interface ProviderPort {
  info(): ProviderInfo;
  listModels(): Promise<string[]>;
  supportsModel(modelID: string): Promise<boolean>;
  isAvailable(modelID: string): Promise<boolean>;
  complete(req: CompletionRequest): Promise<CompletionResponse>;
}

interface RouterOptions {
  profiles: Record<string, ProfileConfig>;
  fallbackChain: string[];
  defaultProvider: string;
}

class Router {
  selectModel(profile: string): Promise<ModelSelection>;
  selectModelForPhase(phase: PhaseInfo): Promise<ModelSelection>;
}

class CostCalculator {
  calculate(modelID: string, inputTokens: number, outputTokens: number): CostBreakdown;
  estimateCost(modelID: string, estimatedTokens: number): number;
}
```

### Python Package

**Effort: High (4-6 weeks)**

Similar to npm - native port recommended:

```python
# sr_router/
from dataclasses import dataclass
from typing import Protocol

class ProviderPort(Protocol):
    def info(self) -> ProviderInfo: ...
    def list_models(self) -> list[str]: ...
    def supports_model(self, model_id: str) -> bool: ...
    async def complete(self, req: CompletionRequest) -> CompletionResponse: ...

class Router:
    def select_model(self, profile: str) -> ModelSelection: ...
    def select_model_for_phase(self, phase: PhaseInfo) -> ModelSelection: ...

class CostCalculator:
    def calculate(self, model_id: str, input_tokens: int, output_tokens: int) -> CostBreakdown: ...
```

---

## 5. Minimal Public API for `sr-router`

```go
package srrouter

// Core routing
type Router interface {
    SelectModel(ctx context.Context, profile string) (*ModelSelection, error)
    SelectModelWithCapabilities(ctx context.Context, profile string, caps []string) (*ModelSelection, error)
    GetFallbackModel(ctx context.Context, profile string) (*ModelSelection, error)
}

// Cost calculation
type CostCalculator interface {
    Calculate(modelID string, inputTokens, outputTokens int) (*CostBreakdown, error)
    EstimateCost(modelID string, estimatedTokens int) (float64, error)
    RegisterModel(modelID, provider string, inputRate, outputRate float64)
}

// Provider abstraction
type Provider interface {
    Info() ProviderInfo
    ListModels(ctx context.Context) ([]string, error)
    SupportsModel(ctx context.Context, modelID string) (bool, error)
    IsAvailable(ctx context.Context, modelID string) (bool, error)
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest, cb StreamCallback) (*CompletionResponse, error)
}

// Registry
type ProviderRegistry interface {
    Register(provider Provider) error
    Get(name string) Provider
    FindByModel(ctx context.Context, modelID string) (Provider, error)
}

// Types
type ModelSelection struct {
    ModelID      string
    ProviderName string
    IsFallback   bool
}

type CostBreakdown struct {
    InputCost    float64
    OutputCost   float64
    TotalCost    float64
    InputTokens  int
    OutputTokens int
    Model        string
    Provider     string
}

type CompletionRequest struct {
    ModelID      string
    Messages     []Message
    MaxTokens    int
    Temperature  float32
    SystemPrompt string
    Tools        []Tool
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

### Simplified Usage Flow

```go
// 1. Initialize
registry := srrouter.NewRegistry()
registry.Register(anthropic.NewProvider(apiKey))
registry.Register(openai.NewProvider(apiKey))
registry.Register(ollama.NewProvider("http://localhost:11434"))

config := srrouter.DefaultConfig()
router := srrouter.NewRouter(config, registry)
calculator := srrouter.NewCostCalculatorWithDefaults()

// 2. Route
selection, _ := router.SelectModel(ctx, "balanced")

// 3. Execute
provider := registry.Get(selection.ProviderName)
resp, _ := provider.Complete(ctx, CompletionRequest{
    ModelID:  selection.ModelID,
    Messages: messages,
})

// 4. Track cost
cost := calculator.Calculate(selection.ModelID, resp.InputTokens, resp.OutputTokens)
fmt.Printf("Cost: $%.6f\n", cost.TotalCost)
```

---

## 6. Gaps for Production-Ready Standalone Library

### Missing Features

| Feature | Priority | Effort | Current State |
|---------|----------|--------|---------------|
| **Rate limiting enforcement** | High | Medium | Config exists, not enforced |
| **Retry logic with backoff** | High | Low | Provider-level only |
| **Circuit breaker** | Medium | Medium | Not implemented |
| **Request queuing** | Medium | Medium | Not implemented |
| **Streaming cost tracking** | Medium | Low | Partial - post-stream only |
| **Budget limits/alerts** | Medium | Low | Not implemented |
| **Model capability detection** | Low | Medium | Manual config only |
| **Token estimation** | Low | Low | Basic (tiktoken integrated) |

### Missing Provider Adapters

| Provider | Priority | Effort |
|----------|----------|--------|
| **Google Gemini** | High | Medium |
| **Azure OpenAI** | Medium | Low (extends OpenAI) |
| **AWS Bedrock** | Medium | Medium |
| **Cohere** | Low | Medium |
| **Mistral API** | Low | Medium |

### Architecture Improvements Needed

1. **Interface-based Registry** (`registry.go:13`)
   - Currently concrete type, should be interface for testability

2. **Config hot-reload**
   - Router supports `UpdateConfig()` but no file watcher

3. **Metrics/Observability hooks**
   - No hooks for external metrics systems (Prometheus, etc.)

4. **Provider health caching**
   - Currently checks on every request, should cache

5. **Structured errors**
   - Custom error types exist but not comprehensive

### Testing Gaps

| Area | Current | Target |
|------|---------|--------|
| Provider mocks | Basic | Need comprehensive mocks |
| Integration tests | Minimal | Need provider integration suite |
| Benchmark tests | None | Need performance baselines |
| Fuzzing | None | Consider for config parsing |

---

## 7. Recommendations

### For Moltbot Integration (Quick Win)

1. **Start with HTTP microservice** (1-2 weeks)
   - Deploy SkillRunner router as standalone service
   - Simple REST API: `POST /select`, `POST /complete`, `GET /cost`
   - Moltbot calls via HTTP

2. **Then native TypeScript port** (4-6 weeks)
   - Port core routing logic (~1,500 LOC)
   - Use existing provider SDKs (Anthropic SDK, OpenAI SDK)
   - Ship as `@sr-router/core` npm package

### For Go Module Extraction

1. **Phase 1: Interface extraction** (1 week)
   - Create `ProviderRegistry` interface
   - Extract profile constants
   - Minimal breaking changes to SkillRunner

2. **Phase 2: Package split** (1-2 weeks)
   - Move to `sr-router/` subdirectory
   - Create separate go.mod
   - Update SkillRunner to import

3. **Phase 3: Standalone features** (2+ weeks)
   - Rate limiting enforcement
   - Circuit breaker
   - Additional providers

### Files to Extract (Prioritized)

```
# Core (must have)
internal/application/ports/provider.go          → sr-router/provider/interface.go
internal/domain/provider/calculator.go          → sr-router/cost/calculator.go
internal/domain/provider/pricing.go             → sr-router/cost/pricing.go
internal/domain/provider/cost.go                → sr-router/cost/types.go
internal/domain/provider/model.go               → sr-router/provider/model.go
internal/domain/provider/tier.go                → sr-router/provider/tier.go
internal/application/provider/router.go         → sr-router/router/router.go
internal/application/provider/resolver.go       → sr-router/router/resolver.go
internal/adapters/provider/registry.go          → sr-router/provider/registry.go
internal/infrastructure/config/routing.go       → sr-router/config/routing.go

# Provider adapters (high value)
internal/adapters/provider/anthropic/           → sr-router/adapters/anthropic/
internal/adapters/provider/openai/              → sr-router/adapters/openai/
internal/adapters/provider/ollama/              → sr-router/adapters/ollama/
internal/adapters/provider/groq/                → sr-router/adapters/groq/
```

---

## 8. Code Quality Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| **Architecture** | ⭐⭐⭐⭐⭐ | Clean hexagonal architecture |
| **Documentation** | ⭐⭐⭐⭐ | Good docstrings, could use more examples |
| **Test coverage** | ⭐⭐⭐⭐ | 85%+ in core packages |
| **Error handling** | ⭐⭐⭐⭐ | Custom errors, good wrapping |
| **Concurrency** | ⭐⭐⭐⭐⭐ | Proper mutex usage throughout |
| **Extensibility** | ⭐⭐⭐⭐ | Interface-based design |
| **Production readiness** | ⭐⭐⭐⭐ | Solid, few gaps |

---

## Summary

The SkillRunner router is **well-designed and production-quality**. Extraction is feasible with 2-3 weeks of focused work for Go, or 4-6 weeks for TypeScript/Python. The main work is decoupling from the `skill` domain and creating interface-based abstractions for the registry.

For Moltbot integration, I recommend:
1. **Immediate**: HTTP microservice wrapper around existing code
2. **Medium-term**: Native TypeScript port of core logic
3. **Long-term**: Full `sr-router` multi-language SDK
