# Skillrunner Documentation

Welcome to the Skillrunner documentation. This guide covers installation, configuration, and usage of the Skillrunner CLI for AI workflow orchestration.

## Quick Navigation

| I want to... | Go to... |
|--------------|----------|
| Get started quickly | [Quick Start Guide](getting-started/quick-start.md) |
| Configure providers | [API Keys Setup](guides/api-keys-setup.md) |
| Learn CLI commands | [Command Reference](COMMAND_REFERENCE.md) |
| Use in Cursor IDE | [Cursor Integration](guides/cursor-integration.md) |
| Contribute code | [Development Setup](development/setup.md) |

---

## Getting Started

New to Skillrunner? Start here:

1. **[Quick Start Guide](getting-started/quick-start.md)** - Install and run your first skill in 5 minutes
2. **[Configuration Guide](getting-started/CONFIGURATION.md)** - Configure providers and routing

### Prerequisites

- Go 1.21+ (for building from source)
- Ollama (for local models)
- API keys for cloud providers (optional)

---

## User Guides

Detailed guides for using Skillrunner effectively:

| Guide | Description |
|-------|-------------|
| [API Keys Setup](guides/api-keys-setup.md) | Configure API keys for Anthropic, OpenAI, Google, and more |
| [Model Selection](guides/model-selection.md) | Understand model routing and selection |
| [Cursor Integration](guides/cursor-integration.md) | Use Skillrunner in Cursor IDE with tasks and snippets |
| [Envelope Integration](guides/envelope-integration.md) | Integrate skill output with external tools |

---

## Reference

### CLI Commands

See **[Command Reference](COMMAND_REFERENCE.md)** for complete documentation of all commands:

```bash
sr run <skill> <request>   # Run a skill
sr ask <skill> <question>  # Quick single-phase query
sr list                    # List available skills
sr status                  # System health check
sr metrics                 # Usage and cost metrics
sr init                    # Initialize configuration
```

### API Documentation

See **[API Reference](API.md)** for envelope format, error codes, and data storage locations.

---

## Architecture

Technical documentation for understanding Skillrunner internals:

- **[Architecture Overview](../ARCHITECTURE.md)** - High-level system design
- **[Routing System](architecture/routing.md)** - Model routing and provider selection

---

## Development

For contributors and developers:

| Document | Description |
|----------|-------------|
| [Development Setup](development/setup.md) | Set up your development environment |
| [Testing Guide](development/testing.md) | Testing strategy and best practices |
| [Release Process](development/release-process.md) | How to create releases |
| [Contributing Guide](../CONTRIBUTING.md) | Contribution guidelines |

### Building from Source

```bash
git clone https://github.com/jbctechsolutions/skillrunner
cd skillrunner
make build
./bin/sr --version
```

---

## Support

- **Issues**: [GitHub Issues](https://github.com/jbctechsolutions/skillrunner/issues)
- **Security**: See [SECURITY.md](../SECURITY.md) for reporting vulnerabilities
- **License**: [MIT License](../LICENSE)
