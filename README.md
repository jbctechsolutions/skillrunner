# Skillrunner

A local-first AI workflow orchestration tool that enables multi-phase AI workflows with intelligent provider routing.

## Overview

Skillrunner executes complex AI workflows defined as **skills** - YAML-based configurations that define multiple execution phases with dependencies. It intelligently routes work to local LLM providers (like Ollama) while seamlessly falling back to cloud providers (Anthropic Claude, OpenAI, Groq) when needed.

### Key Features

- **Local-First Architecture** - Prioritizes local model execution for privacy and cost savings
- **Multi-Phase Workflows** - Define complex AI tasks as directed acyclic graphs (DAGs) of dependent phases
- **Intelligent Routing** - Automatically selects the best provider/model based on routing profiles
- **Multiple Providers** - Supports Ollama, Anthropic Claude, OpenAI, and Groq
- **Cost-Aware** - Built-in cost tracking and optimization through routing profiles
- **Extensible Skills** - Create and share reusable workflow definitions

## Quick Start

```bash
# Build from source
make build

# Initialize configuration
sr init

# Check system status
sr status

# List available skills
sr list

# Run a multi-phase workflow
sr run code-review "Review this Python code for security issues: $(cat mycode.py)"

# Quick single-phase query
sr ask code-review "What are common security vulnerabilities in Python?"
```

## Routing Profiles

Skillrunner uses routing profiles to balance cost and quality:

| Profile | Description | Best For |
|---------|-------------|----------|
| `cheap` | Local-first, cost optimization | Drafts, exploration, high-volume tasks |
| `balanced` | Quality-to-cost ratio (default) | General use, production workloads |
| `premium` | Best available models | Critical tasks, final outputs |

```bash
# Use a specific routing profile
sr run code-review --profile premium "Review this security-critical code"
```

## Built-in Skills

| Skill | Description |
|-------|-------------|
| `code-review` | 3-phase comprehensive code review (patterns → security → report) |
| `test-gen` | Generate unit tests with coverage analysis |
| `doc-gen` | Generate documentation from code |

## Configuration

Configuration is stored at `~/.skillrunner/config.yaml`:

```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
  anthropic:
    api_key_encrypted: ""
    enabled: false
  openai:
    api_key_encrypted: ""
    enabled: false
  groq:
    api_key_encrypted: ""
    enabled: false

routing:
  default_profile: balanced

logging:
  level: info
  format: text

skills:
  directory: ~/.skillrunner/skills
```

## Documentation

| Document | Description |
|----------|-------------|
| [Getting Started](docs/getting-started.md) | Installation, setup, and first workflow |
| [CLI Reference](docs/cli-reference.md) | Complete command-line documentation |
| [Skills Guide](docs/skills-guide.md) | Creating and using workflow skills |
| [Configuration](docs/configuration.md) | Configuration options and best practices |
| [Architecture](docs/architecture.md) | System design for developers/contributors |

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Full check (format, vet, lint, test)
make check

# Generate coverage report
make coverage
```

### Project Structure

```
skillrunner/
├── cmd/skillrunner/      # CLI entry point
├── internal/
│   ├── domain/           # Core business logic (skill, workflow, provider)
│   ├── application/      # Ports and application services
│   ├── adapters/         # Provider implementations
│   ├── infrastructure/   # Config, skills loading, utilities
│   └── presentation/     # CLI commands and output formatting
├── skills/               # Built-in skill definitions
├── configs/              # Configuration examples
└── docs/                 # User documentation
```

## Requirements

- Go 1.23+
- Ollama (recommended for local execution)
- API keys for cloud providers (optional)

## License

MIT License - see [LICENSE](LICENSE) for details.
