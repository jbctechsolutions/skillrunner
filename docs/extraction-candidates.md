# Extractable Standalone Library Candidates

Analysis of Skillrunner components that could be extracted and released as independent Go libraries.

## Summary

The Sacramento codebase is well-suited for component extraction due to:
- **Strict hexagonal architecture** — domain layer has zero external dependencies
- **Interface-based design** — all external dependencies pass through ports
- **100% domain test coverage** — components come with tests ready to ship
- **Immutable, defensively-copied models** — no hidden state coupling

---

## Tier 1: Ready to Extract (Zero External Dependencies)

### 1. Workflow DAG Engine

**Location:** `internal/domain/workflow/` (`dag.go`, `plan.go`, `checkpoint.go`)

Pure Go DAG implementation with topological sorting (Kahn's algorithm), cycle detection, and crash recovery via checkpointing.

- **Lines:** ~1,500 | **Coverage:** 100%
- **Dependencies:** stdlib only
- **Potential package:** `github.com/jbctechsolutions/go-workflow-dag`
- **Use cases:** Task orchestration, CI/CD pipelines, build systems, any DAG-based workflow engine

**Why it's great:** Complete, battle-tested implementation. Cycle detection and topological sort are things every workflow system needs to reimplement. Checkpointing gives crash recovery out of the box.

---

### 2. MCP Protocol Implementation

**Location:** `internal/domain/mcp/` (`jsonrpc.go`, `tool.go`, `server.go`, `errors.go`)

Complete JSON-RPC 2.0 and Model Context Protocol typed domain layer — messages, tools, server lifecycle, and error codes.

- **Lines:** ~1,200 | **Coverage:** 100%
- **Dependencies:** stdlib only
- **Potential package:** `github.com/jbctechsolutions/go-mcp`
- **Use cases:** Any MCP client or server implementation in Go

**Why it's great:** MCP is an emerging standard with growing adoption. No complete Go implementation exists yet. The domain layer handles all the protocol types; adapters handle transport.

---

### 3. LLM Cost Calculator

**Location:** `internal/domain/provider/` (`cost.go`, `pricing.go`, `calculator.go`, `tier.go`)

Per-token cost calculation, savings tracking (local vs. cloud), and tier categorization for multi-provider LLM applications.

- **Lines:** ~800 | **Coverage:** 100%
- **Dependencies:** stdlib only
- **Potential package:** `github.com/jbctechsolutions/go-llm-cost`
- **Key types:** `CostBreakdown`, `CostSummary`, `AgentTier` (cheap/balanced/premium)
- **Use cases:** Any Go application orchestrating multiple LLM providers

**Why it's great:** Every multi-model LLM app needs cost tracking. This has clean immutable models, savings calculations, and is completely generic (not Skillrunner-specific).

---

### 4. Machine-Specific Encryption

**Location:** `internal/infrastructure/crypto/encryption.go`

AES-256-GCM encryption with keys automatically derived from machine identity (hostname + salt). Useful for locking credentials to a specific machine without a key management service.

- **Lines:** ~200 | **Coverage:** 85%
- **Dependencies:** stdlib only (`crypto/aes`, `crypto/cipher`, `crypto/rand`)
- **Potential package:** `github.com/jbctechsolutions/go-machine-crypto`
- **Use cases:** CLI tools with stored API keys, Electron-style desktop apps, any app needing machine-bound secrets

**Why it's great:** Solves a specific problem elegantly — "encrypt a secret so only this machine can decrypt it" — without needing a key management service or external dependency.

---

## Tier 2: Minor Refactoring Needed

### 5. Composite Cache System

**Location:** `internal/adapters/cache/` (`memory.go`, `sqlite.go`, `composite.go`)

Two-tier cache (in-memory + SQLite) with TTL, hit/miss statistics, size-based eviction, and a composite that uses memory as L1 and SQLite as L2.

- **Lines:** ~1,500 | **Coverage:** 85%
- **Dependencies:** `mattn/go-sqlite3`
- **Potential package:** `github.com/jbctechsolutions/go-composite-cache`
- **Refactoring needed:** Remove LLM-specific metadata fields, generalize key/value types
- **Use cases:** Any application needing persistent + in-memory two-tier caching

---

### 6. Structured Error Handling

**Location:** `internal/domain/errors/errors.go`

Error wrapping with typed codes, structured context maps, and cause chains — richer than `fmt.Errorf` but lighter than full observability libraries.

- **Lines:** ~300 | **Coverage:** 100%
- **Dependencies:** stdlib only
- **Potential package:** `github.com/jbctechsolutions/go-structured-errors`
- **Refactoring needed:** Replace Skillrunner-specific error codes with generic ones
- **Use cases:** Any Go application that wants structured error context without pulling in a full observability stack

---

## Recommended Extraction Order

1. **`go-mcp`** — Riding the MCP wave; smallest surface area; most differentiated
2. **`go-workflow-dag`** — Broader appeal; self-contained; complete implementation
3. **`go-llm-cost`** — Niche but valuable; growing LLM ecosystem needs this
4. **`go-machine-crypto`** — Small scope; easy to ship; useful for CLI tool authors
5. **`go-composite-cache`** — Crowded space but clean; do this last after refactoring

---

## Why Now

The domain layer's zero-dependency constraint (enforced by the hexagonal architecture) means extraction is mostly a copy-and-publish operation for Tier 1 candidates. Tests come along for free. The main work is:

1. Creating new repos with proper Go module names
2. Removing any remaining Skillrunner-specific types/names
3. Writing a README with usage examples
4. Publishing to pkg.go.dev
