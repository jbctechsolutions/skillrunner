# Skillrunner API Documentation

## Overview

Skillrunner provides a CLI interface for orchestrating AI development workflows. This document describes the available commands and their usage.

## Envelope Format

Skillrunner legacy skills generate JSON envelopes that can be consumed by external tools like Claude Code CLI or Continue IDE. The envelope format is:

```json
{
  "version": "1.0",
  "skill": "skill-name",
  "request": "user request",
  "steps": [
    {
      "intent": "plan|edit|run",
      "model": "model-identifier",
      "prompt": "step prompt",
      "context": [{"type": "folder|file|pattern", "source": "..."}],
      "file_ops": [],
      "metadata": {}
    }
  ],
  "metadata": {"created_at": "...", "workspace": "..."}
}
```

For integration with Claude Code CLI, see the [Quick Start Guide](../getting-started/quick-start.md#using-legacy-skill-envelopes).

## Commands

### `sr run <skill> <request>`

Execute a skill (orchestrated, built-in/legacy, or imported).

**Options:**
- `--model, -m`: Override the default model
- `--workspace, -w`: Path to workspace directory
- `--output, -o`: Output file path (JSON format)
- `--compact`: Output compact JSON
- `--prefer-local`: Prefer local models (default: true)
- `--force-cloud`: Force cloud models
- `--task-type, -t`: Task type for model routing
- `--params`: JSON parameters for marketplace skills
- `--stream`: Stream responses in real-time (Ollama only, orchestrated skills only)
- `--no-cache`: Skip cache and force fresh execution (orchestrated skills only)

**Skill Types:**
- **Orchestrated skills**: Execute phases, support `--stream`, return results
- **Built-in/Legacy skills**: Generate JSON envelopes, `--stream` not supported, use `--output` to save envelope

**Examples:**
```bash
# Orchestrated skill (executes)
sr run hello-orchestration "a developer learning Go"
sr run code-review "$(cat myfile.go)" --stream

# Legacy skill (generates envelope)
sr run backend-architect "add auth" --output plan.json
```

### `sr ask <skill-id> <question>`

Ask a question using a marketplace skill with intelligent model routing.

**Options:**
- `--prefer-local`: Prefer local models (default: true)
- `--force-cloud`: Force cloud models
- `--task-type, -t`: Task type for routing (default: analysis)
- `--model, -m`: Override model selection
- `--stream`: Stream responses in real-time

**Examples:**
```bash
sr ask avl-expert "Best microphones for a 100-person venue?"
sr ask business-analyst "What KPIs should I track for SaaS?"
```

### `sr list`

List all available skills.

**Options:**
- `--format, -f`: Output format (table|json, default: table)

### `sr status`

Show system status including available skills, configured models, and system health.

**Options:**
- `--format, -f`: Output format (table|json, default: table)

### `sr import <source>`

Import a skill from various sources (web, local path, git repository).

**Examples:**
```bash
sr import https://example.com/skill.yaml
sr import ~/my-skills/custom-skill
sr import https://github.com/user/skill-repo.git
```

### `sr update <skill-id>`

Update an imported skill from its original source.

### `sr imported`

List all imported skills with metadata.

### `sr remove <skill-id>`

Remove an imported skill from the local cache.

### `sr convert <input-file> <output-file>`

Convert skill format between Claude Code marketplace format and Skillrunner orchestrated format.

### `sr config set <key> <value>`

Set a configuration value.

### `sr config show`

Show current configuration.

### `sr metrics`

View routing metrics and costs.

**Options:**
- `--format, -f`: Output format (table|json, default: table)
- `--since`: Time range (e.g., 24h, 7d, 1w)
- `--export`: Export to file (CSV or JSON)

**Examples:**
```bash
sr metrics
sr metrics --since 7d
sr metrics --export costs.csv
```

### `sr cache clear`

Clear all cached execution results.

### `sr cache stats`

Show cache statistics (total entries, valid entries, expired entries).

## Error Codes

Skillrunner uses structured error codes for better error handling:

- `SKILL_NOT_FOUND`: Skill not found
- `SKILL_INVALID`: Invalid skill definition
- `MODEL_UNAVAILABLE`: Model is not available
- `MODEL_TIMEOUT`: Model request timed out
- `EXECUTION_FAILED`: Execution failed
- `CONTEXT_TOO_LARGE`: Context exceeds model limits
- `CONFIG_INVALID`: Invalid configuration
- `NETWORK_ERROR`: Network error
- `API_ERROR`: API error

## Configuration

Configuration is stored at `~/.skillrunner/config.yaml`. Use `sr init` to set up initial configuration.

### Environment Variables

- `ANTHROPIC_API_KEY`: Anthropic API key
- `OPENAI_API_KEY`: OpenAI API key
- `OLLAMA_HOST`: Ollama host (default: http://localhost:11434)

## Data Storage

- **Skills**: `~/.skillrunner/skills/`
- **Imported Skills**: `~/.skillrunner/marketplace/`
- **Metrics**: `~/.skillrunner/metrics/executions.json`
- **Cache**: `~/.skillrunner/cache/results.json`
- **Configuration**: `~/.skillrunner/config.yaml`
