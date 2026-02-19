# SkillRunner → Claude Code Plugin: Architecture Evaluation & Implementation Plan

## Executive Summary

**Recommendation: PROCEED WITH PLUGIN PIVOT + DUAL-MODE SUPPORT**

SkillRunner's hexagonal architecture is perfectly positioned for this pivot. **65-70% of existing code is directly reusable**, with the core domain and application layers requiring minimal changes. The pivot is technically sound and can be completed in **2-3 weeks** (13-20 days).

**Key Finding:** The Provider Router, Cost Calculator, and Session Management systems were architected with exactly the kind of abstraction needed for hook-based interception.

---

## 1. Current Architecture Analysis

### Core Components (Reusable)

**Provider Routing System** (`internal/application/provider/`)
- **Router** (386 LOC): Profile-based model selection (cheap/balanced/premium)
- **Resolver** (223 LOC): Cost-aware resolution with tracking
- **Registry** (150 LOC): Provider discovery and availability checks
- **Reusability: 95%** - Core routing logic works perfectly for hooks

**Cost Optimization** (`internal/domain/provider/`)
- **Calculator** (275 LOC): Token-based pricing with per-1K rates
- **CostSummary** (137 LOC): Aggregation across providers/models
- **Pricing Database**: 40+ models with Jan 2026 rates
- **Reusability: 100%** - Pure functions, zero external dependencies

**Provider Adapters** (`internal/adapters/provider/`)
- **Anthropic**: Messages API native (230 LOC)
- **OpenAI**: Format translation via buildRequest/buildResponse (220 LOC)
- **Groq**: OpenAI-compatible (180 LOC)
- **Ollama**: Local model support (200 LOC)
- **Reusability: 100%** - Already implements ProviderPort interface

**Session Management** (`internal/application/context/`)
- Session tracking with token usage
- Workspace context management
- Checkpoint persistence (SQLite)
- **Reusability: 50%** - Core models reusable, lifecycle needs adaptation

### Components to Discard

**Workflow Execution** (~950 LOC)
- Multi-phase skill orchestration (executor.go, planner.go, dag.go)
- Not needed for single-request routing in plugin mode

**CLI Presentation** (~8,000 LOC)
- 24 Cobra commands, output formatting
- Replaced by HTTP service + hook script

---

## 2. Target Architecture: HTTP Service + PreToolUse Hooks

### Architecture Overview

```
Claude Code
    ↓ (PreToolUse hook: JSON on stdin)
Hook Script (Python/Bash)
    ↓ (HTTP POST to localhost:8787)
SkillRunner HTTP Service (long-running daemon)
    ├─ Routing Service: Analyze request → Select provider
    ├─ Session Manager: Track conversation history + costs
    ├─ Existing Router: Reuse profile-based selection
    └─ Provider Adapters: Translate to target API format
    ↓
Selected Provider (Ollama/Anthropic/OpenAI/Groq)
```

### API Endpoints

**POST /api/v1/route** - Core routing endpoint
```json
Request:
{
  "conversation_id": "conv_123",
  "request": {
    "model": "claude-sonnet-4.5",
    "messages": [...],
    "max_tokens": 4096
  }
}

Response:
{
  "action": "route",
  "provider": {
    "name": "ollama",
    "endpoint": "http://localhost:11434/v1/messages",
    "model": "llama3.2"
  },
  "session": {
    "session_id": "sr_session_456",
    "estimated_cost": 0.0
  }
}
```

**GET /api/v1/session/:id** - Session state and cost tracking

**GET /api/v1/health** - Service health check

### Session Persistence Strategy

**Hybrid Approach:**
- Use Claude Code's `conversation_id` as primary key
- Store session metadata in SQLite (existing infrastructure)
- Track: total cost, models used, routing history
- TTL: 24h of inactivity (configurable)

**Database Schema:**
```sql
CREATE TABLE plugin_sessions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL UNIQUE,
    workspace_path TEXT,
    created_at TIMESTAMP,
    last_activity TIMESTAMP,
    total_cost REAL,
    total_input_tokens INTEGER,
    total_output_tokens INTEGER,
    provider_history JSON
);
```

### Routing Logic

**Complexity Analysis Signals:**
- Token count (via existing tokenizer)
- Tool usage (function calling)
- Message count (conversation length)
- Image presence (vision models)
- System prompt length

**Profile Selection:**
```go
if tokens > 50000 → "premium"    // Large context
if tokens < 2000 && !tools → "cheap"  // Simple queries → local
if tools && toolCount > 5 → "premium"  // Complex tool use
default → "balanced"
```

### Performance Targets

| Operation | Target | Strategy |
|-----------|--------|----------|
| HTTP handling | <10ms | Minimal middleware |
| Token estimation | <20ms | Cached tokenizer |
| Provider selection | <30ms | In-memory lookup |
| Session load/save | <20ms | SQLite prepared statements |
| Request translation | <10ms | Pre-compiled transforms |
| **Total** | **<100ms** | Well under 500ms hook timeout |

---

## 3. Component Reusability Matrix

| Component | Category | Reuse % | Effort | Notes |
|-----------|----------|---------|--------|-------|
| **Domain Layer** | | | | |
| Cost Calculator | ✅ As-Is | 95% | None | Pure functions |
| Provider Models | ✅ As-Is | 100% | None | Pricing data |
| Session Domain | 🔧 Modify | 60% | Low | Adapt for stateless hooks |
| Workflow DAG | ❌ Discard | 0% | N/A | Not needed for single requests |
| **Application Layer** | | | | |
| Provider Router | ✅ As-Is | 95% | None | Core routing perfect for hooks |
| Provider Resolver | ✅ As-Is | 90% | Low | Cost tracking integration |
| Context Manager | 🔧 Modify | 40% | Medium | Extract context building only |
| Workflow Executor | ❌ Discard | 0% | N/A | Multi-phase not needed |
| **Adapter Layer** | | | | |
| All Provider Adapters | ✅ As-Is | 100% | None | Already Messages API compatible |
| Provider Registry | ✅ As-Is | 95% | None | Discovery unchanged |
| MCP Integration | 🔧 Modify | 70% | Low | Persist across hook calls |
| **Infrastructure** | | | | |
| Config System | 🔧 Modify | 85% | Low | Same schema, different path |
| SQLite Storage | 🔧 Modify | 80% | Medium | Session persistence |
| Tokenizer | ✅ As-Is | 100% | None | Token estimation |
| **Presentation** | | | | |
| CLI Commands | 🆕 Replace | 0% | High | HTTP handlers instead |
| **New Components** | | | | |
| HTTP Server | 🆕 Build | 0% | High | ~400 LOC |
| Hook Script | 🆕 Build | 0% | Medium | ~200 LOC |
| Plugin Manifest | 🆕 Build | 0% | Low | ~50 LOC |

**Overall Code Reuse: 65-70%**

---

## 4. Implementation Plan: 4 Phases, 13-20 Days

### Phase 1: Core Concept Validation (Days 1-2)

**Goal:** Prove hook interception → routing → response works <500ms

**Tasks:**
1. Create HTTP service skeleton (`cmd/sr-plugin/main.go`)
2. Implement hook parser (`internal/infrastructure/hooks/parser.go`)
3. Wire up existing Router for provider selection
4. Test single request end-to-end

**Success Criteria:**
- ✅ HTTP service responds <500ms
- ✅ Single request routes to correct provider
- ✅ Response format validated

**Deliverables:**
- Working HTTP server on localhost:8787
- POST /hook endpoint accepting Messages API format
- Basic routing using existing Router

---

### Phase 2: Service Layer (Days 3-7)

**Goal:** Full session management with context continuity

**Tasks:**
1. Session state management (`internal/domain/plugin/session.go`)
   - Track conversation history per Claude conversation_id
   - SQLite persistence with existing storage layer
   - 24h TTL and cleanup

2. Request/response translation (`internal/adapters/plugin/translator.go`)
   - Claude API format → internal CompletionRequest
   - Preserve metadata: stop_reason, usage, model_used

3. Cost tracking integration
   - Reuse existing metrics infrastructure
   - Per-session budget tracking
   - Savings calculation vs. premium baseline

4. HTTP middleware
   - Logging with correlation IDs
   - Timeout enforcement (<450ms)
   - Error handling and recovery

**Success Criteria:**
- ✅ 10-turn conversation maintains context
- ✅ Cost tracking matches provider usage
- ✅ Session survives service restart
- ✅ Response time <450ms p99

**Deliverables:**
- Session persistence layer
- Cost aggregation across requests
- Full HTTP API (route, session, health endpoints)

---

### Phase 3: Claude Code Integration (Days 8-10)

**Goal:** Working plugin installed in Claude Code

**Tasks:**
1. Plugin configuration
   - Create `.claude/plugins/skillrunner/PLUGIN.md`
   - Hook matcher for `anthropic__*` tools
   - Installation instructions

2. Service management
   - Auto-start on Claude Code launch
   - Health check endpoint
   - Graceful shutdown

3. Complexity-based routing (`internal/application/plugin/selector.go`)
   - Token count analysis
   - Tool usage detection
   - Map to profiles (cheap/balanced/premium)

4. End-to-end testing
   - Install in Claude Code
   - Multi-turn conversations
   - Verify provider switching

**Success Criteria:**
- ✅ Plugin installs without errors
- ✅ Hooks intercept Anthropic API calls
- ✅ 5+ consecutive requests work
- ✅ Cost savings >70% on simple tasks

**Deliverables:**
- Claude Code plugin manifest
- Hook script (Python/Bash)
- E2E test suite

---

### Phase 4: Polish & Distribution (Days 11-13)

**Goal:** Production-ready plugin with docs and observability

**Tasks:**
1. Error handling
   - Service unavailable → passthrough
   - Routing failure → fallback to balanced
   - Timeout → return "ask" decision

2. Documentation
   - README-PLUGIN.md (installation guide)
   - Architecture documentation
   - Troubleshooting guide

3. Observability
   - Dashboard: `sr plugin status`
   - Metrics: requests, savings, models used
   - CSV/JSON export

4. Distribution
   - Binary releases (macOS/Linux)
   - Claude Code marketplace listing

**Success Criteria:**
- ✅ Error rate <1% across 100 requests
- ✅ Documentation enables self-service
- ✅ Metrics actionable

**Deliverables:**
- Complete documentation
- Distribution packages
- Observability dashboard

---

## 5. Critical Implementation Decisions

### Decision 1: Request Router vs Proxy

**Choice: Request Router** (not full proxy)

**Rationale:**
- Routing decision <100ms vs full proxy 2-5s
- Leverage Claude Code's streaming/error handling
- Fewer points of failure
- Compatible with all Claude Code features

### Decision 2: Dual-Mode Support

**Choice: Support both CLI and Plugin modes**

**Rationale:**
- 95% code overlap (shared core)
- Different entry points (main.go)
- Complementary use cases:
  - CLI: Direct skill execution, batch workflows
  - Plugin: Transparent cost optimization in Claude Code

### Decision 3: Session Tracking

**Choice: Hybrid (Claude conversation_id + SkillRunner metadata)**

**Rationale:**
- Claude provides stable conversation_id
- SkillRunner enriches with cost/provider history
- Enables per-conversation analytics

### Decision 4: Service Mode

**Choice: Long-running daemon** (not per-request spawn)

**Rationale:**
- Hook timeout 500ms requires fast response
- Amortize startup cost across requests
- In-memory caches (routing, provider health)
- SQLite connection pooling

---

## 6. Critical Files to Modify/Create

### New Files (Plugin Mode)

**cmd/sr-plugin/main.go** (~150 LOC)
- HTTP service entry point
- Initialize container (reuse existing)
- Wire up router and session manager

**internal/infrastructure/hooks/parser.go** (~200 LOC)
- Parse PreToolUse hook JSON from stdin
- Extract Messages API request
- Convert to internal CompletionRequest

**internal/domain/plugin/session.go** (~150 LOC)
- PluginSession model
- Session lifecycle management
- Cost aggregation per conversation

**internal/application/plugin/selector.go** (~200 LOC)
- Complexity analysis (tokens, tools, images)
- Profile selection logic
- Routing decision engine

**internal/infrastructure/plugin/http_handler.go** (~300 LOC)
- HTTP server implementation
- POST /api/v1/route, GET /session, GET /health
- Middleware (logging, timeout, error handling)

**Hook Script** (~100 LOC Python/Bash)
- Read hook JSON from stdin
- POST to localhost:8787
- Return routing decision to stdout

### Modified Files (Existing Code)

**internal/application/provider/router.go**
- Minor: Add plugin-specific wrapper methods
- Core logic unchanged

**internal/infrastructure/config/loader.go**
- Support plugin config path: `~/.config/claude-code/routing.yaml`
- Same schema, different default location

**internal/infrastructure/storage/sqlite.go**
- Add plugin_sessions table schema
- Session CRUD operations

---

## 7. Risk Assessment & Mitigation

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Hook timeout violations | Medium | Critical | Aggressive caching, <450ms limit |
| Session state loss | Low | High | SQLite WAL mode, correlation ID recovery |
| Provider API changes | Low | Medium | Adapter pattern isolates changes |
| Response time degradation | Medium | High | Performance testing from Day 1 |
| Hook format changes | Medium | High | Version detection, graceful fallback |

---

## 8. Success Metrics

### Phase Success Criteria

**Phase 1 (Days 1-2):**
- HTTP service <500ms response time
- Single request routed correctly
- Response format validated

**Phase 2 (Days 3-7):**
- 10-turn conversation maintains context
- Cost tracking accurate
- Session persists across restarts

**Phase 3 (Days 8-10):**
- Plugin installs cleanly
- 5+ consecutive requests work
- 70%+ cost savings on simple tasks

**Phase 4 (Days 11-13):**
- <1% error rate
- Documentation complete
- Metrics dashboard functional

### Long-Term Success (3-6 months)

- 70-90% cost reduction vs direct Anthropic
- <500ms p99 response time
- 50+ active users
- Zero data loss incidents

---

## 9. First Steps (This Week)

### Day 1 Morning: HTTP Service Skeleton
```bash
mkdir -p cmd/sr-plugin
# Create basic HTTP server listening on :8787
# Single POST /hook endpoint
go run cmd/sr-plugin/main.go
curl http://localhost:8787/health  # Should return 200 OK
```

### Day 1 Afternoon: Hook Parser
```bash
mkdir -p internal/infrastructure/hooks
# TDD: Write tests for Messages API parsing
# Implement parser.go
go test ./internal/infrastructure/hooks/...
```

### Day 2 Morning: Wire Up Router
```go
// In cmd/sr-plugin/main.go:
// 1. Initialize existing Container
// 2. Get Router from container
// 3. Call SelectModel() in hook handler
```

### Day 2 Afternoon: Validation Test
```bash
# End-to-end test: stdin → routing → response
./test-hook.sh
# Expected: Routes to Ollama, <500ms
```

---

## 10. Verification Plan

### Unit Tests
- Hook parser (all format variations)
- Routing logic (complexity analysis)
- Session state management
- Cost calculation accuracy

### Integration Tests
- HTTP service with real providers
- Session persistence across restarts
- Error handling scenarios
- Timeout enforcement

### E2E Tests
- Install plugin in Claude Code (manual)
- Multi-turn conversations
- Provider switching mid-conversation
- Cost tracking validation

### Performance Tests
- Hook response time (p50, p95, p99)
- Concurrent request handling
- Memory usage under load
- Session cleanup efficiency

**Target:** 85%+ code coverage, <500ms p99 latency

---

## Recommendation Summary

**PROCEED WITH PLUGIN PIVOT**

The architecture is sound, reusability is high (65-70%), and the implementation is straightforward. The existing hexagonal architecture means the core business logic (routing, cost tracking, provider adapters) requires zero changes.

**Estimated Timeline:** 2-3 weeks (13-20 days)

**Confidence Level:** High (80%)

**Next Action:** Start Phase 1 validation this week to prove hook integration works as expected.
