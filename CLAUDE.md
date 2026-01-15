# Skillrunner Development Guide

## Architecture

Skillrunner follows **Hexagonal Architecture** (Ports & Adapters):
- Dependencies flow INWARD only
- Domain layer has ZERO external dependencies

```
cmd/skillrunner/      # CLI entry point
internal/
├── domain/           # Pure business logic (no I/O, no external deps)
├── application/      # Use cases, ports (interfaces), orchestration
├── adapters/         # Provider implementations (Ollama, Anthropic, etc.)
├── infrastructure/   # Config, logging, storage, cross-cutting concerns
└── presentation/     # CLI commands and output formatting
```

## Test-Driven Development (TDD)

### Workflow
1. Write failing test for smallest unit of behavior
2. Implement minimum code to make test pass
3. Refactor while keeping tests green
4. Repeat

### Coverage Targets
| Layer | Target | Priority |
|-------|--------|----------|
| Domain | 100% | Critical |
| Application | 95% | Critical |
| Adapters/Infrastructure | 85% | High |
| Presentation | 75% | Medium |
| **Overall** | **80%+** | Required |

### Testing Pyramid
- 75% Unit Tests (domain logic, use cases with mocks)
- 20% Integration Tests (provider + storage integration)
- 5% E2E Tests (full CLI workflows)

### Mock Strategy
| Component | Mock Approach |
|-----------|---------------|
| HTTP APIs | `httptest.Server` |
| Providers | Interface mocks |
| SQLite | In-memory SQLite |
| Filesystem | `t.TempDir()` |
| Time | Injectable clock |

## Domain-First Development

**Always build domain layer FIRST** with 100% coverage before implementing adapters.

### Domain Model Requirements
- Immutable where possible (private fields with getters)
- Validation in constructors (`NewXxx` functions)
- Serializable (JSON)
- Never have external dependencies
- Include comprehensive error types

### Domain Errors Pattern
```go
// internal/domain/{feature}/errors.go
var (
    ErrNotFound    = errors.New("not found")
    ErrInvalid     = errors.New("invalid")
    // ...
)
```

## Error Handling

- Use `fmt.Errorf("%w", err)` for error chains
- All errors wrapped with context
- Define domain-specific errors per package

## Code Style

- Format: `go fmt`
- Lint: `golangci-lint`
- Imports: `goimports`
- Context: Pass `context.Context` through all layers
- Logging: Use `log/slog` (structured)

## Commands

```bash
make check     # Run fmt, vet, lint, test
make test      # Run tests with coverage
make lint      # Run golangci-lint
make build     # Build binary
```

## Spec Documentation

Full specification available at:
`~/.repos/github.com/jbctechsolutions/skillrunner-spec-docs/`
