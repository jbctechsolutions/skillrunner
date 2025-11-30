# Architecture Documentation

Detailed architecture and design documentation for Skillrunner.

## Quick Navigation

| I want to... | Go to... |
|--------------|----------|
| Understand overall architecture | [Architecture Overview](../../ARCHITECTURE.md) |
| Learn about model routing | [Routing Architecture](routing.md) |

## Architecture Documents

### [Routing Architecture](routing.md)

Model routing design and specification:
- Priority order (CLI > skill > config > default)
- Profile-based routing (cheap, balanced, premium)
- Local-first vs cloud-first strategies
- Fallback chains

### [Architecture Overview](../../ARCHITECTURE.md)

High-level system architecture (in root directory):
- Component overview
- Data flow
- Provider system
- Orchestration engine

## Key Concepts

### Provider Adapters

The provider adapter system enables:
- Automatic model discovery from local and cloud providers
- Unified provider interface across Ollama, Anthropic, OpenAI, Google, OpenRouter, Groq
- Health checking with actionable errors
- Easy extension to new providers

### Model Routing

The routing system provides:
- Intelligent model selection based on task requirements
- Cost optimization through local-first strategies
- Fallback chains for reliability
- Profile-based routing (cheap/balanced/premium)

### Orchestration Engine

The orchestration engine supports:
- Multi-phase workflow execution
- DAG-based task dependencies
- Parallel execution where possible
- Context management across phases

## Related Documentation

- [Development Setup](../development/setup.md) - Development environment setup
- [Testing Guide](../development/testing.md) - Testing strategies
- [Model Selection Guide](../guides/model-selection.md) - User-facing routing guide
