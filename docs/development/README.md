# Development Documentation

Documentation for developers working on Skillrunner.

## Quick Navigation

| I want to... | Go to... |
|--------------|----------|
| Set up my environment | [Development Setup](setup.md) |
| Run tests | [Testing Guide](testing.md) |
| Create a release | [Release Process](release-process.md) |

## Development Guides

### [Development Setup](setup.md)

Set up your development environment:
- Prerequisites (Go 1.21+, Git)
- Building from source
- Running locally
- Code organization

### [Testing Guide](testing.md)

Testing strategies and guidelines:
- Unit tests
- Integration tests
- Running test suites
- Test coverage

### [Release Process](release-process.md)

Release workflow and procedures:
- Version numbering
- Release checklist
- GoReleaser configuration
- Publishing releases

## Code Structure

```
skillrunner/
├── cmd/skillrunner/      # CLI entry point
├── internal/             # Internal packages
│   ├── models/           # Provider adapters
│   ├── orchestration/    # Multi-phase execution
│   ├── router/           # Model routing
│   ├── context/          # Context management
│   ├── envelope/         # Output envelope format
│   ├── skills/           # Skill loading/validation
│   └── ...
├── docs/                 # Documentation
└── ...
```

## Key Packages

| Package | Description |
|---------|-------------|
| `internal/models/` | Provider adapter system (Ollama, Anthropic, OpenAI, etc.) |
| `internal/orchestration/` | Multi-phase workflow execution |
| `internal/router/` | Model routing and selection |
| `internal/context/` | Context management |
| `internal/skills/` | Skill loading and validation |
| `internal/envelope/` | Output envelope format |

## Related Documentation

- [Contributing Guidelines](../../CONTRIBUTING.md) - How to contribute
- [Architecture Overview](../../ARCHITECTURE.md) - System architecture
- [Architecture Documentation](../architecture/) - Detailed architecture docs
