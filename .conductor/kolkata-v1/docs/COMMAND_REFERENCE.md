# Skillrunner Command Reference

Complete reference for all Skillrunner CLI commands and options.

## Table of Contents

- [Core Commands](#core-commands)
- [Skill Management](#skill-management)
- [Marketplace & Discovery](#marketplace--discovery)
- [Model Management](#model-management)
- [Configuration](#configuration)
- [Metrics & Monitoring](#metrics--monitoring)
- [Cache Management](#cache-management)
- [Utility Commands](#utility-commands)

---

## Core Commands

### `sr run <skill> <request>`

Execute a skill (orchestrated, built-in/legacy, or imported).

**Arguments:**
- `<skill>` - Name of the skill to run
- `<request>` - Natural language request describing what to do

**Options:**
- `--model, -m <model-id>` - Override the default model selection
- `--workspace, -w <path>` - Path to workspace directory
- `--output, -o <file>` - Output file path (JSON format)
- `--compact` - Output compact JSON (no pretty printing)
- `--prefer-local` - Prefer local models (Ollama) over cloud models (default: true)
- `--force-cloud` - Force cloud models (Anthropic/OpenAI)
- `--task-type, -t <type>` - Task type for model routing (extraction|generation|analysis|summarization)
- `--profile <profile>` - Routing profile (cheap|balanced|premium)
- `--stream` - Stream responses in real-time (Ollama only, orchestrated skills only)
- `--no-cache` - Skip cache and force fresh execution (orchestrated skills only)

**Skill Type Behavior:**

**Orchestrated Skills** (e.g., `hello-orchestration`, `bug-fix`):
- Execute LLM calls and run phases with dependencies
- Support `--stream` for real-time output
- Return final results
- Support `--no-cache` to bypass caching

**Built-in/Legacy Skills** (e.g., `test`, `backend-architect`):
- Generate JSON envelopes with structured workflow steps
- Do NOT execute LLM calls - only create workflow structure
- `--stream` flag does NOT work (no execution happens)
- Use `--output` to save envelope for external tools
- Envelopes can be consumed by Claude Code CLI or Continue IDE

**Examples:**
```bash
# Run an orchestrated skill (executes phases)
sr run hello-orchestration "a developer learning Go"
sr run bug-fix "Login fails" --stream
sr run feature-implementation "Add dark mode" --profile cheap --no-cache

# Run a legacy skill (generates envelope)
sr run backend-architect "add auth" --output plan.json
sr run test "hello world" --output test-envelope.json

# Run with specific model
sr run code-review "$(cat myfile.go)" --model ollama/qwen2.5-coder:32b
```

---

### `sr ask <skill-id> <question>`

Ask a question using a marketplace skill with intelligent model routing.

**Arguments:**
- `<skill-id>` - ID of the marketplace skill to use
- `<question>` - Your question or request

**Options:**
- `--prefer-local` - Prefer local models (default: true)
- `--force-cloud` - Force cloud models
- `--task-type, -t <type>` - Task type for routing (default: analysis)
- `--model, -m <model-id>` - Override model selection
- `--stream` - Stream responses in real-time

**Examples:**
```bash
# Ask a marketplace skill
sr ask avl-expert "Best microphones for a 100-person venue?"

# Force cloud model
sr ask business-analyst "Revenue analysis" --force-cloud

# Use specific model
sr ask data-analyst "A/B test significance" --model anthropic/claude-sonnet-4-20250514
```

---

### `sr list`

List all available orchestrated skills.

**Options:**
- `--format, -f <format>` - Output format (table|json, default: table)

**Examples:**
```bash
# List all skills in table format
sr list

# List in JSON format
sr list --format json
```

---

### `sr status`

Show system status including available skills, configured models, and system health.

**Options:**
- `--format, -f <format>` - Output format (table|json, default: table)

**Examples:**
```bash
# Show system status
sr status

# Get status as JSON
sr status --format json
```

---

## Skill Management

### `sr import <source>`

Import a skill from various sources (web, local path, git repository).

**Arguments:**
- `<source>` - Source location (URL, file path, or git repository)

**Examples:**
```bash
# Import from web URL
sr import https://example.com/skill.yaml

# Import from local path
sr import ~/my-skills/custom-skill
sr import /path/to/skill/SKILL.md

# Import from git repository
sr import https://github.com/user/skill-repo.git
sr import git@github.com:user/skill-repo.git
```

**Notes:**
- Automatically detects source type (web, local, git)
- Supports both SKILL.md (marketplace) and AGENT.md (agent) formats
- Caches imported skills at `~/.skillrunner/marketplace/`

---

### `sr update <skill-id>`

Update an imported skill from its original source.

**Arguments:**
- `<skill-id>` - ID of the skill to update

**Examples:**
```bash
# Update a skill
sr update avl-expert
sr update my-custom-skill
```

---

### `sr imported`

List all imported skills with metadata.

**Examples:**
```bash
# List all imported skills
sr imported
```

**Output includes:**
- Skill ID
- Source location
- Last updated timestamp
- Cache location

---

### `sr remove <skill-id>`

Remove an imported skill from the local cache.

**Arguments:**
- `<skill-id>` - ID of the skill to remove

**Examples:**
```bash
# Remove a skill
sr remove my-skill
```

---

### `sr convert <input-file> <output-file>`

Convert skill format between Claude Code marketplace format and Skillrunner orchestrated format.

**Arguments:**
- `<input-file>` - Input skill file
- `<output-file>` - Output skill file

**Examples:**
```bash
# Convert marketplace skill to orchestrated format
sr convert marketplace-skill.yaml orchestrated-skill.yaml

# Convert orchestrated skill to marketplace format
sr convert orchestrated-skill.yaml marketplace-skill.yaml
```

**Notes:**
- Auto-detects input format
- Preserves all skill metadata
- Validates output format

---

## Marketplace & Discovery

### `sr search <query>`

Search for skills in the HuggingFace marketplace.

**Arguments:**
- `<query>` - Search query

**Examples:**
```bash
# Search for skills
sr search "audio engineering"
sr search "data analysis"
```

---

### `sr inspect <skill-id>`

Inspect a marketplace skill to see details before importing.

**Arguments:**
- `<skill-id>` - ID of the skill to inspect

**Examples:**
```bash
# Inspect a skill
sr inspect avl-expert
sr inspect business-analyst
```

**Output includes:**
- Skill description
- Capabilities
- Model requirements
- Example usage

---

## Model Management

### `sr models list [--provider=<name>]`

List all available models from all providers (or filtered by provider).

**Options:**
- `--provider <name>` - Filter by provider (ollama|anthropic|openai|google|groq|openrouter)

**Examples:**
```bash
# List all models
sr models list

# List only Ollama models
sr models list --provider=ollama

# List only Anthropic models
sr models list --provider=anthropic
```

---

### `sr models check <model-id>`

Check the health and availability of a specific model.

**Arguments:**
- `<model-id>` - Model identifier (format: `[provider/]model-name`)

**Examples:**
```bash
# Check specific model
sr models check ollama/qwen2.5:14b
sr models check anthropic/claude-3-5-sonnet-20241022

# Check without provider (searches all providers)
sr models check qwen2.5:14b
```

---

### `sr models validate <skill-id>`

Load a skill and check all its preferred models to recommend the best available one.

**Arguments:**
- `<skill-id>` - ID of the skill to validate

**Examples:**
```bash
# Validate models for a skill
sr models validate hello-orchestration
sr models validate backend-architect
```

---

### `sr models refresh`

Clear provider caches and force re-fetch of model lists.

**Examples:**
```bash
# Refresh all provider caches
sr models refresh
```

**Use cases:**
- After adding new models to Ollama
- To get latest model information from providers
- After provider configuration changes

---

## Configuration

### `sr config set <key> <value>`

Set a configuration value.

**Supported keys:**
- `workspace` - Path to the workspace directory
- `default_model` - Default model to use (e.g., `ollama/qwen2.5:14b`)
- `output_format` - Output format (`table|json`)
- `compact_output` - Compact output mode (`true|false`)

**Examples:**
```bash
# Set workspace path
sr config set workspace /path/to/workspace

# Set default model
sr config set default_model ollama/qwen2.5:14b

# Set output format
sr config set output_format json

# Set compact output
sr config set compact_output true
```

---

### `sr config show [--format=<format>]`

Show current configuration.

**Options:**
- `--format, -f <format>` - Output format (table|json, default: table)

**Examples:**
```bash
# Show config in table format
sr config show

# Show config as JSON
sr config show --format json
```

---

### `sr init`

Initialize Skillrunner configuration and create directory structure.

**Examples:**
```bash
# Initialize Skillrunner
sr init
```

**Creates:**
- `~/.skillrunner/config.yaml` - Configuration file
- `~/.skillrunner/skills/` - Directory for orchestrated skills
- `~/.skillrunner/marketplace/` - Directory for imported skills
- `~/.skillrunner/metrics/` - Directory for metrics data
- `~/.skillrunner/cache/` - Directory for result cache

---

## Metrics & Monitoring

### `sr metrics [options]`

View routing metrics and costs.

**Options:**
- `--format, -f <format>` - Output format (table|json, default: table)
- `--since <duration>` - Time range (e.g., `24h`, `7d`, `1w`, `1m`)
- `--export <file>` - Export to file (CSV or JSON based on extension)

**Examples:**
```bash
# View all metrics
sr metrics

# View metrics for last 7 days
sr metrics --since 7d

# View metrics for last 24 hours
sr metrics --since 24h

# Export metrics to CSV
sr metrics --export costs.csv

# Export metrics to JSON
sr metrics --export metrics.json --format json
```

**Metrics include:**
- Total API calls
- Token usage (input/output)
- Cost breakdown by provider
- Model usage statistics
- Skill usage statistics
- P95/P99 latency percentiles

---

## Cache Management

### `sr cache clear`

Clear all cached execution results.

**Examples:**
```bash
# Clear all cache
sr cache clear
```

---

### `sr cache stats`

Show cache statistics.

**Examples:**
```bash
# Show cache stats
sr cache stats
```

**Output includes:**
- Total cache entries
- Valid entries
- Expired entries
- Cache size

---

## Utility Commands

### `sr completion <shell>`

Generate autocompletion script for the specified shell.

**Arguments:**
- `<shell>` - Shell name (bash|zsh|fish|powershell)

**Examples:**
```bash
# Generate bash completion
sr completion bash > ~/.bash_completion

# Generate zsh completion
sr completion zsh > ~/.zsh_completion
```

---

## Global Flags

These flags are available for all commands:

- `--mcp-endpoint <url>` - MCP server endpoint (default: http://localhost:3000)
- `--model-policy <policy>` - Model selection policy (`auto|local_first|performance_first|cost_optimized`)
- `-h, --help` - Show help for command
- `-v, --version` - Show version information

**Examples:**
```bash
# Use custom MCP endpoint
sr ask avl-expert "question" --mcp-endpoint http://localhost:8080

# Set model policy globally
sr run skill "request" --model-policy cost_optimized
```

---

## Error Codes

Skillrunner uses structured error codes for better error handling:

- `SKILL_NOT_FOUND` - Skill not found
- `SKILL_INVALID` - Invalid skill definition
- `MODEL_UNAVAILABLE` - Model is not available
- `MODEL_TIMEOUT` - Model request timed out
- `EXECUTION_FAILED` - Execution failed
- `CONTEXT_TOO_LARGE` - Context exceeds model limits
- `CONFIG_INVALID` - Invalid configuration
- `NETWORK_ERROR` - Network error
- `API_ERROR` - API error

---

## Environment Variables

- `ANTHROPIC_API_KEY` - Anthropic API key
- `OPENAI_API_KEY` - OpenAI API key
- `GOOGLE_API_KEY` - Google API key
- `OLLAMA_HOST` - Ollama host (default: http://localhost:11434)

---

## Data Storage Locations

- **Skills**: `~/.skillrunner/skills/`
- **Imported Skills**: `~/.skillrunner/marketplace/`
- **Metrics**: `~/.skillrunner/metrics/executions.json`
- **Cache**: `~/.skillrunner/cache/results.json`
- **Configuration**: `~/.skillrunner/config.yaml`

---

## See Also

- [Quick Start Guide](getting-started/quick-start.md)
- [API Documentation](API.md)
- [Architecture Overview](../ARCHITECTURE.md)
- [Configuration Guide](getting-started/CONFIGURATION.md)
