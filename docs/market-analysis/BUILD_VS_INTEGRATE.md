# SkillRunner: Build vs. Integrate Analysis

**Date:** December 4, 2025
**Purpose:** Determine which components to build in-house vs. leverage existing tools

---

## Current Architecture Summary

| Component | Location | Lines of Code | Complexity |
|-----------|----------|---------------|------------|
| DAG/Workflow Executor | `internal/orchestration/` | ~800 | Medium |
| Provider Routing | `internal/routing/` | ~400 | Low |
| LLM Providers | `internal/models/` | ~1200 | Medium |
| Cost Tracking | `internal/metrics/` | ~300 | Low |
| Metrics Storage | `internal/metrics/storage.go` | ~200 | Low |
| Cache | `internal/cache/` | ~200 | Low |
| Skill Loading | `internal/skills/` | ~400 | Low |

---

## Build vs. Integrate Decision Matrix

### 1. LLM Provider Layer (`internal/models/`)

**Current:** Custom implementations for Ollama, Anthropic, OpenAI, Groq

**Options:**
| Option | Pros | Cons |
|--------|------|------|
| **Keep Building** | Full control, Go-native, minimal deps | Maintenance burden, each provider ~300 LOC |
| **LiteLLM Proxy** | 100+ providers, cost tracking built-in | Adds Python dependency, network hop |
| **go-litellm client** | Go-native, community maintained | Incomplete, not official |
| **Bifrost (Go)** | Go-native, faster than LiteLLM | Newer, less proven |

**RECOMMENDATION: KEEP BUILDING**

Reasoning:
- You already have 4 providers working
- Go's HTTP client is simple - each provider is ~200-300 lines
- Adding providers is straightforward (copy pattern)
- Avoids Python dependency and extra network hop
- Your routing logic is tightly integrated with provider selection

**Action:** Continue current approach. Only consider LiteLLM proxy if you need 10+ providers.

---

### 2. Observability/Metrics (`internal/metrics/`)

**Current:** JSON file storage, basic cost tracking, per-execution records

**Options:**
| Option | Pros | Cons |
|--------|------|------|
| **Keep Building** | Simple, zero deps, works offline | Limited analytics, no dashboards |
| **Langfuse Go SDK** | Traces, spans, scores, cloud dashboard | Adds dependency, requires Langfuse account |
| **OpenTelemetry + go-openllmetry** | Standard, vendor-agnostic, flexible backends | Manual instrumentation, complex setup |
| **Just OpenTelemetry** | Standard spans/traces | No LLM-specific semantics |

**RECOMMENDATION: HYBRID APPROACH**

Reasoning:
- Your current metrics are fine for v1.0
- For v1.3+ (dashboard), you'll want proper storage anyway
- Langfuse Go SDK exists and works ([henomis/langfuse-go](https://github.com/henomis/langfuse-go))
- BUT: Keep local-first - don't require cloud account

**Action:**
1. **v1.0-v1.2:** Keep current JSON storage (works, simple)
2. **v1.3+:** Add optional Langfuse export via Go SDK
3. Keep local metrics as primary, Langfuse as optional sync

```go
// Future interface
type MetricsExporter interface {
    Export(record ExecutionRecord) error
}

type LocalExporter struct { ... }      // Current JSON file
type LangfuseExporter struct { ... }   // Optional cloud sync
type OpenTelemetryExporter struct { ... } // Optional OTEL
```

---

### 3. DAG/Workflow Execution (`internal/orchestration/`)

**Current:** Custom DAG with topological sort, batch execution, template variables

**Options:**
| Option | Pros | Cons |
|--------|------|------|
| **Keep Building** | Full control, Go-native, simple | No state persistence, basic error handling |
| **LangGraph** | State machines, checkpointing, loops | Python only, overkill for your use case |
| **Temporal** | Durable execution, retries, timeouts | Heavy infra, enterprise complexity |
| **Custom with improvements** | Best fit, incremental enhancement | Development time |

**RECOMMENDATION: KEEP BUILDING**

Reasoning:
- Your DAG implementation is solid and simple (~500 LOC)
- LangGraph is Python-only
- Temporal is massive overkill
- Your use case (linear/parallel phases) doesn't need complex state machines
- The "learning from outcomes" feature you want is YOUR code anyway

**Action:** Keep current implementation. Enhance with:
- Retry logic per phase (you'll need this for outcome tracking)
- Phase timeout handling
- Better error propagation

---

### 4. Cost Tracking (`internal/metrics/cost.go`)

**Current:** Simple formula: `(tokens / 1000) * cost_per_1k`

**Options:**
| Option | Pros | Cons |
|--------|------|------|
| **Keep Building** | Simple, accurate enough | Manual price updates |
| **LiteLLM cost tracking** | Auto-updated prices | Python dependency |
| **OpenRouter prices API** | Real-time pricing | External API call |

**RECOMMENDATION: KEEP BUILDING + PRICE FILE**

Reasoning:
- Your formula is correct
- Prices don't change that often
- You can maintain a `prices.yaml` that users can update
- This IS your differentiator - you want to own it

**Action:**
1. Keep current implementation
2. Add `~/.skillrunner/prices.yaml` for user overrides
3. Consider monthly price update check from a simple JSON endpoint

---

### 5. Caching (`internal/cache/`)

**Current:** JSON file with TTL, SHA256 keys

**Options:**
| Option | Pros | Cons |
|--------|------|------|
| **Keep Building** | Works, simple, no deps | Single machine only |
| **SQLite** | Better querying, still single-file | Adds CGO dependency |
| **Redis** | Distributed, fast | Heavy for CLI tool |
| **BadgerDB** | Go-native, embedded, fast | Adds dependency |

**RECOMMENDATION: KEEP BUILDING (for now)**

Reasoning:
- Current cache works fine for single-user CLI
- JSON file is portable and debuggable
- Only upgrade if you need team/shared caching

**Action:** Keep current. Consider BadgerDB for v1.4 (teams) if needed.

---

### 6. Skill Definition Format

**Current:** Custom YAML format with phases, dependencies, routing

**Options:**
| Option | Pros | Cons |
|--------|------|------|
| **Keep Current** | Works, you control it | Not standard |
| **OpenAI Function format** | Industry standard | Doesn't fit your workflow model |
| **LangGraph format** | Has state, edges | Python ecosystem |
| **MCP format** | Growing standard | Different purpose (tools, not workflows) |

**RECOMMENDATION: KEEP CURRENT**

Reasoning:
- Your format is designed for multi-phase workflows
- No standard exists for "cost-optimized AI workflow definitions"
- Converting from other formats is doable (you have `internal/converter/`)

**Action:** Keep format. Add importers for common formats if users request.

---

## Summary: Build vs. Integrate

| Component | Decision | Reasoning |
|-----------|----------|-----------|
| **LLM Providers** | BUILD | Already working, Go-native, no Python dep needed |
| **DAG Executor** | BUILD | Simple, fits your needs, LangGraph is Python-only |
| **Cost Tracking** | BUILD | This is your core value - own it |
| **Metrics Storage** | BUILD + OPTIONAL EXPORT | Keep local-first, add Langfuse export later |
| **Cache** | BUILD | Works fine, upgrade only if distributed needed |
| **Skill Format** | BUILD | No standard exists for your use case |

---

## What To Integrate (Optional, Later)

| Tool | When | Why |
|------|------|-----|
| **Langfuse Go SDK** | v1.3+ | Optional cloud sync for teams wanting dashboards |
| **OpenTelemetry** | v1.4+ | For enterprises with existing observability |
| **Prometheus exporter** | v1.4+ | For enterprises with existing monitoring |

---

## Code Changes Needed

### Immediate (v1.0-v1.2): None

Your current architecture is fine. Keep building.

### Future (v1.3+): Add Exporter Interface

```go
// internal/metrics/exporter.go

type Exporter interface {
    Export(ctx context.Context, record ExecutionRecord) error
    Flush(ctx context.Context) error
}

type CompositeExporter struct {
    exporters []Exporter
}

// Implementations:
// - LocalJSONExporter (current behavior)
// - LangfuseExporter (optional, uses henomis/langfuse-go)
// - OTELExporter (optional, uses go-openllmetry)
```

### Configuration

```yaml
# ~/.skillrunner/config.yaml
metrics:
  local: true  # Always enabled
  langfuse:
    enabled: false
    public_key: ${LANGFUSE_PUBLIC_KEY}
    secret_key: ${LANGFUSE_SECRET_KEY}
  otel:
    enabled: false
    endpoint: http://localhost:4317
```

---

## Architecture Diagram: Current vs. Future

### Current (v1.0)
```
┌─────────────────────────────────────────────────────┐
│                    SkillRunner CLI                  │
├─────────────────────────────────────────────────────┤
│  Skill Loader → DAG Executor → Phase Runner        │
│                      ↓                              │
│              Profile Router                         │
│         ↙        ↓         ↘                       │
│   Ollama    Anthropic    OpenAI                    │
│                      ↓                              │
│           Cost Computer → JSON Storage             │
└─────────────────────────────────────────────────────┘
```

### Future (v1.4+)
```
┌─────────────────────────────────────────────────────┐
│                    SkillRunner CLI                  │
├─────────────────────────────────────────────────────┤
│  Skill Loader → DAG Executor → Phase Runner        │
│                      ↓                              │
│              Profile Router (+ Learning)           │
│         ↙        ↓         ↘                       │
│   Ollama    Anthropic    OpenAI    ...more         │
│                      ↓                              │
│           Cost Computer + Outcome Tracker          │
│                      ↓                              │
│              Metrics Exporter                       │
│         ↙        ↓         ↘                       │
│   Local      Langfuse     OTEL                     │
│   JSON       (optional)   (optional)               │
└─────────────────────────────────────────────────────┘
```

---

## Conclusion

**Build everything for v1.0-v1.2.** Your current architecture is solid and appropriate for a Go CLI tool.

The main integration points to prepare for (but not implement yet):
1. **Metrics Exporter interface** - makes Langfuse/OTEL integration clean later
2. **Price data abstraction** - allows external price updates

**Do NOT add:**
- Python dependencies (LiteLLM, LangGraph)
- Heavy infrastructure (Redis, Temporal)
- Cloud requirements (must work offline)

**Your competitive advantage is being a simple, single-binary Go tool.** Don't compromise that for integrations you don't need yet.

---

*Document prepared December 4, 2025*
