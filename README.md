# Skillrunner

Local-first AI workflow orchestration. Run multi-phase AI tasks with intelligent model routing — use local models for most work, cloud only when needed.

**Cut your AI API costs by 70-90%.**

## Install

```bash
# macOS/Linux
brew install jbctechsolutions/tap/skillrunner

# Or download binary
curl -sSL https://github.com/jbctechsolutions/skillrunner/releases/latest/download/skillrunner_Linux_x86_64.tar.gz | tar xz
sudo mv skillrunner /usr/local/bin/
sudo ln -sf /usr/local/bin/skillrunner /usr/local/bin/sr
```

## Usage

The CLI is available as `skillrunner` or `sr` for short:

```bash
skillrunner run code-review --input "$(cat main.go)"
# or
sr run code-review --input "$(cat main.go)"
```

## Quick Start

```bash
# 1. Install Ollama (required for local models)
brew install ollama
ollama serve

# 2. Pull a model
ollama pull qwen2.5:14b

# 3. Initialize config
sr init

# 4. Run a skill
sr run code-review "Review this code for issues"

# 5. See usage metrics
sr metrics
```

## Features

- **Multi-phase workflows** — Break complex tasks into steps with dependencies
- **Intelligent routing** — Cheap models for simple work, premium for complex
- **Local-first** — Ollama support means most work never hits the cloud
- **Cost tracking** — See exactly what you spend per task
- **Marketplace** — Import skills from GitHub, npm, or HuggingFace

## Commands

```bash
skillrunner run <skill> <request>   # Run a skill
skillrunner ask <skill> <question>  # Quick single-phase query
skillrunner list                    # Show available skills
skillrunner status                  # System health check
skillrunner metrics                 # Usage and cost metrics
skillrunner init                    # Initialize configuration

# Or use the short alias:
sr run <skill> <request>
sr list
```

## Supported Providers

| Provider | Type | Status |
|----------|------|--------|
| Ollama | Local | ✅ Ready |
| Anthropic | Cloud | ✅ Ready |
| OpenAI | Cloud | ✅ Ready |
| OpenRouter | Cloud | ✅ Ready |

## Configuration

Config lives at `~/.skillrunner/config.yaml`:

```yaml
# Provider configuration with auto model discovery
providers:
  ollama:
    url: http://localhost:11434
    enabled: true

  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    enabled: true

  openai:
    api_key: ${OPENAI_API_KEY}
    enabled: false
```

See [config.example.yaml](config.example.yaml) for all options.

## Documentation

- [Getting Started](docs/getting-started/quick-start.md)
- [Configuration](docs/getting-started/CONFIGURATION.md)
- [API Keys Setup](docs/guides/api-keys-setup.md)
- [Architecture](ARCHITECTURE.md)

## License

MIT — see [LICENSE](LICENSE)
