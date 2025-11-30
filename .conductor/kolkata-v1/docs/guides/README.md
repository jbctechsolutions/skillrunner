# User Guides

Detailed guides for using Skillrunner effectively.

## Available Guides

### Provider Setup

| Guide | Description |
|-------|-------------|
| [API Keys Setup](api-keys-setup.md) | Configure API keys for Anthropic, OpenAI, Google, OpenRouter, and Groq |

### Model Routing

| Guide | Description |
|-------|-------------|
| [Model Selection](model-selection.md) | Understand how Skillrunner selects models and routes requests |

### IDE Integration

| Guide | Description |
|-------|-------------|
| [Cursor Integration](cursor-integration.md) | Use Skillrunner in Cursor IDE with tasks, snippets, and keyboard shortcuts |

### Tool Integration

| Guide | Description |
|-------|-------------|
| [Envelope Integration](envelope-integration.md) | Integrate Skillrunner output with Claude Code CLI and other external tools |

## Guide Summaries

### [API Keys Setup](api-keys-setup.md)

Set up API keys for cloud LLM providers:
- Anthropic (Claude models)
- OpenAI (GPT models)
- Google (Gemini models)
- OpenRouter (multiple providers)
- Groq (fast inference)

Includes pricing information, security best practices, and troubleshooting.

### [Model Selection](model-selection.md)

Understand model routing:
- Priority order (CLI > skill > config > default)
- Profile-based routing (cheap, balanced, premium)
- Local-first vs cloud-first strategies
- Cost optimization tips

### [Cursor Integration](cursor-integration.md)

Comprehensive Cursor IDE integration:
- VS Code tasks for common operations
- Keyboard shortcuts
- Custom Cursor commands
- Snippets for fast access
- Pre-commit hooks

### [Envelope Integration](envelope-integration.md)

Use Skillrunner output with external tools:
- Envelope JSON format
- Claude Code CLI integration
- Python and Node.js examples
- Workflow automation

## Related Documentation

- [Quick Start Guide](../getting-started/quick-start.md) - Get started with Skillrunner
- [Configuration Guide](../getting-started/CONFIGURATION.md) - Full configuration options
- [Command Reference](../COMMAND_REFERENCE.md) - All CLI commands
