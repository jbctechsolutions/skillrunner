# Skillrunner CLI Reference

Skillrunner (sr) is a local-first AI workflow orchestration tool that enables multi-phase AI workflows with intelligent provider routing.

## Table of Contents

- [Global Flags](#global-flags)
- [Commands](#commands)
  - [version](#version)
  - [init](#init)
  - [list](#list)
  - [run](#run)
  - [ask](#ask)
  - [status](#status)
  - [import](#import)
  - [metrics](#metrics)
- [Exit Codes](#exit-codes)
- [Environment Variables](#environment-variables)

---

## Global Flags

These flags can be used with any command:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | `-c` | string | `~/.skillrunner/config.yaml` | Path to configuration file |
| `--output` | `-o` | string | `text` | Output format: `text`, `json` |
| `--verbose` | `-v` | bool | `false` | Enable verbose output |

### Global Flag Examples

```bash
# Use a custom config file
sr --config /path/to/config.yaml list

# Get JSON output for scripting
sr --output json status

# Enable verbose logging
sr --verbose run code-review "Check for security issues"
```

---

## Commands

### version

Display version, build information, and platform details.

#### Synopsis

```bash
sr version [flags]
```

#### Description

Shows detailed version information including version number, git commit hash, build date, Go version, and platform architecture.

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--short` | `-s` | bool | `false` | Print only the version number |

#### Examples

```bash
# Show full version information
sr version

# Show only version number
sr version --short

# Get version as JSON
sr version -o json
```

#### Output

**Text format (default):**
```
Skillrunner
───────────
  Version:     0.1.0-dev
  Git Commit:  unknown
  Build Date:  unknown
  Go Version:  go1.21.0
  Platform:    darwin/arm64
```

**JSON format:**
```json
{
  "version": "0.1.0-dev",
  "git_commit": "unknown",
  "build_date": "unknown",
  "go_version": "go1.21.0",
  "platform": "darwin/arm64"
}
```

---

### init

Initialize skillrunner configuration interactively.

#### Synopsis

```bash
sr init [flags]
```

#### Description

Creates the `~/.skillrunner/` directory structure and generates a `config.yaml` file with provider settings through an interactive wizard.

The initialization process:
- Creates `~/.skillrunner/` directory
- Creates `~/.skillrunner/skills/` directory for skill definitions
- Generates `~/.skillrunner/config.yaml` with provider configurations
- Prompts for Ollama endpoint and optional cloud provider API keys

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--force` | `-f` | bool | `false` | Overwrite existing configuration |

#### Examples

```bash
# Run interactive initialization
sr init

# Force reinitialize (overwrites existing config)
sr init --force

# Initialize with JSON output (uses defaults, no prompts)
sr init -o json
```

#### Interactive Prompts

The init wizard will prompt for:

1. **Ollama Configuration**
   - Ollama URL (default: `http://localhost:11434`)
   - Enable Ollama (default: yes)

2. **Cloud Providers** (optional)
   - Configure Anthropic (Claude)
   - Configure OpenAI
   - Configure Groq

#### Output

**Text format:**
```
Skillrunner Configuration

This wizard will help you set up skillrunner.

Local Provider (Ollama)
Ollama URL [http://localhost:11434]:
Enable Ollama [Y/n]: y

Cloud Providers (Optional)
API keys will be stored encrypted in config.yaml

Configure Anthropic (Claude) [y/N]: y
Anthropic API key: sk-ant-...

Configuration initialized successfully!

  Config directory:   /Users/user/.skillrunner
  Config file:        /Users/user/.skillrunner/config.yaml
  Skills directory:   /Users/user/.skillrunner/skills
```

**JSON format:**
```json
{
  "config_dir": "/Users/user/.skillrunner",
  "config_file": "/Users/user/.skillrunner/config.yaml",
  "skills_dir": "/Users/user/.skillrunner/skills",
  "initialized": true
}
```

#### Notes

- Configuration file is created with restricted permissions (0600) for security
- API keys are stored in the config file (TODO: encryption not yet implemented)
- If config already exists, use `--force` to overwrite
- JSON output mode skips interactive prompts and uses defaults

---

### list

Display available skills in the skillrunner.

#### Synopsis

```bash
sr list [flags]
```

#### Aliases

`ls`

#### Description

Shows all available multi-phase AI workflow skills, including name, description, number of phases, and routing profile for each skill.

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | `-f` | string | (uses global) | Output format: `text`, `json`, `table` |

#### Examples

```bash
# List all skills
sr list

# List as table
sr list --format table

# Get skill list as JSON
sr list -o json
```

#### Output

**Text/Table format:**
```
Available Skills

NAME           DESCRIPTION                               PHASES  ROUTING
code-review    Analyze code for quality, security...          3  quality-first
summarize      Generate concise summaries of docume...        2  local-first
translate      Translate text between languages wit...        2  cost-aware
extract-data   Extract structured data from unstruct...       4  performance
generate-tests Generate unit tests for code with cov...       3  quality-first

Total: 5 skill(s)
```

**JSON format:**
```json
{
  "skills": [
    {
      "name": "code-review",
      "description": "Analyze code for quality, security, and best practices",
      "phase_count": 3,
      "routing_profile": "quality-first"
    },
    ...
  ],
  "count": 5
}
```

#### Notes

- The `--format` flag takes precedence over the global `--output` flag
- If no skills are available, displays instructions on how to add skills

---

### run

Execute a multi-phase AI workflow skill.

#### Synopsis

```bash
sr run <skill> <request> [flags]
```

#### Description

Executes a skill definition, orchestrating the multi-phase workflow and managing provider selection based on the routing profile.

The run command handles:
- Multi-phase workflow execution
- Intelligent provider routing (local-first, cost-aware, performance-based)
- Phase dependency management
- Optional streaming output

#### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `skill` | Yes | Name of the skill to execute |
| `request` | Yes | The request/prompt for the skill |

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--profile` | `-p` | string | `balanced` | Routing profile: `cheap`, `balanced`, `premium` |
| `--stream` | `-s` | bool | `false` | Enable streaming output |

#### Routing Profiles

| Profile | Description |
|---------|-------------|
| `cheap` | Prioritize cost, use local/cheaper models |
| `balanced` | Balance between cost and quality (default) |
| `premium` | Prioritize quality, use best available models |

#### Examples

```bash
# Run a skill with default settings
sr run code-review "Review this pull request for security issues"

# Run with a specific profile
sr run code-review "Review this PR" --profile premium

# Run with streaming output
sr run summarize "Summarize this document" --stream

# Run with cheap profile for cost savings
sr run translate "Hello, world!" --profile cheap

# Get execution result as JSON
sr run code-review "Check for bugs" -o json
```

#### Output

**Text format:**
```
Skill Execution

  Skill:      code-review
  Profile:    balanced
  Streaming:  false

  Request:    Review this pull request for security issues

Executing skill: code-review with request: Review this pull request for security issues
```

**JSON format:**
```json
{
  "skill": "code-review",
  "request": "Review this pull request for security issues",
  "profile": "balanced",
  "stream": false,
  "status": "stub",
  "message": "Executing skill: code-review with request: Review this pull request for security issues"
}
```

#### Notes

- Profile must be one of: `cheap`, `balanced`, `premium`
- Invalid profile values will result in an error
- Streaming mode provides real-time output as the skill executes

---

### ask

Execute a quick single-phase query against a skill.

#### Synopsis

```bash
sr ask <skill> <question> [flags]
```

#### Description

Provides a simplified interface for single-phase queries, skipping the multi-phase workflow and returning a quick response using the first/default phase of the specified skill.

Ideal for:
- Quick questions
- Simple tasks that don't require full workflow orchestration
- Rapid prototyping and testing

#### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `skill` | Yes | Name of the skill to query |
| `question` | Yes | The question/prompt for the skill |

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--model` | `-m` | string | (auto) | Override model selection (e.g., `claude-3-opus`, `gpt-4`, `llama3`) |
| `--profile` | `-p` | string | `balanced` | Routing profile: `cheap`, `balanced`, `premium` |

#### Examples

```bash
# Ask a quick question using the default model
sr ask summarize "What are the key points of this document?"

# Ask with a specific model override
sr ask code-review "Is this function safe?" --model claude-3-opus

# Ask with a routing profile
sr ask translate "Hello, world!" --profile premium

# Quick question with cheap profile
sr ask summarize "Summarize this text" --profile cheap
```

#### Output

**Text format:**
```
Quick Ask

  Skill:    summarize
  Profile:  balanced

  Question: What are the key points of this document?

Executing single-phase query against skill: summarize
```

**JSON format:**
```json
{
  "skill": "summarize",
  "question": "What are the key points of this document?",
  "profile": "balanced",
  "model": "",
  "mode": "single-phase",
  "status": "stub",
  "message": "Quick query to skill: summarize"
}
```

#### Notes

- Only executes the first/default phase of the skill
- Model override bypasses normal provider routing logic
- Faster than `run` for simple queries

---

### status

Show system health status and provider connectivity.

#### Synopsis

```bash
sr status [flags]
```

#### Description

Displays the health status of the skillrunner system, including:
- Provider connectivity and health (Ollama, Anthropic, OpenAI, Groq)
- Available models per provider
- Configuration status
- Skill availability

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--detailed` | `-d` | bool | `false` | Show detailed status with latency and model info |

#### Examples

```bash
# Show basic status
sr status

# Show detailed status with latency info
sr status --detailed

# Get status as JSON for scripting
sr status -o json
```

#### Output

**Text format (basic):**
```
Skillrunner Status

  System:   ● healthy
  Version:  0.1.0-dev

Configuration
Config loaded from ~/.skillrunner/config.yaml
  Skills Dir:  ~/.skillrunner/skills (3 skills)

Providers

  ● healthy ollama [local]
  ● healthy anthropic [cloud]
  ● degraded openai [cloud]
      Error: high latency detected
  ● unavailable groq [cloud]
      Error: API key not configured

Summary: 2 healthy, 1 degraded, 1 unavailable
```

**Text format (detailed):**
```
Skillrunner Status

  System:   ● healthy
  Version:  0.1.0-dev

Configuration
Config loaded from ~/.skillrunner/config.yaml
  Skills Dir:  ~/.skillrunner/skills (3 skills)

Providers

  ● healthy ollama [local]
      Endpoint: http://localhost:11434
      Latency:  12ms
      Models:
        • llama3.2:latest
        • codellama:13b
        • mistral:latest
  ● healthy anthropic [cloud]
      Endpoint: https://api.anthropic.com
      Latency:  145ms
      Models:
        • claude-3-5-sonnet-20241022
        • claude-3-opus-20240229
  ...
```

**JSON format:**
```json
{
  "status": "healthy",
  "version": "0.1.0-dev",
  "providers": [
    {
      "name": "ollama",
      "type": "local",
      "status": "healthy",
      "endpoint": "http://localhost:11434",
      "models": ["llama3.2:latest", "codellama:13b", "mistral:latest"],
      "latency": "12ms"
    },
    ...
  ],
  "config_loaded": true,
  "config_path": "~/.skillrunner/config.yaml",
  "skills_dir": "~/.skillrunner/skills",
  "skill_count": 3
}
```

#### Provider Status Indicators

| Status | Color | Description |
|--------|-------|-------------|
| `healthy` | Green | Provider is operational |
| `degraded` | Yellow | Provider is operational but with issues (e.g., high latency) |
| `unavailable` | Red | Provider is not accessible |

#### Notes

- Status checks are performed in real-time when the command runs
- Detailed mode includes latency measurements and model listings
- Use JSON output for monitoring and automation scripts

---

### import

Import skill definitions from URLs, git repositories, or local paths.

#### Synopsis

```bash
sr import <source> [flags]
```

#### Description

Imports skill definitions from various sources:
- URL to a YAML file (e.g., `https://example.com/skill.yaml`)
- Git repository URL (clones and finds skills in `skills/` directory)
- Local file path (copies to skillrunner skills directory)

Imported skills are saved to `~/.skillrunner/skills/` by default.

#### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `source` | Yes | URL, git repository, or local path to skill definition(s) |

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--name` | `-n` | string | (auto) | Rename the skill on import |
| `--force` | `-f` | bool | `false` | Overwrite existing skill if it exists |

#### Source Types

The import command automatically detects the source type:

| Source Pattern | Detected Type | Description |
|----------------|---------------|-------------|
| `https://example.com/skill.yaml` | URL | Direct YAML file download |
| `https://github.com/user/repo.git` | Git | Clone repository |
| `https://github.com/user/repo` | Git | Clone repository |
| `./skill.yaml` or `/path/to/skill.yaml` | Local | Copy local file |
| `./skills/` or `/path/to/skills/` | Local | Copy from local directory |

#### Examples

```bash
# Import a skill from a URL
sr import https://example.com/skills/code-review.yaml

# Import from a git repository
sr import https://github.com/user/skillrunner-skills.git

# Import a local skill file
sr import ./my-skill.yaml

# Import with a custom name
sr import https://example.com/review.yaml --name my-review

# Force overwrite existing skill
sr import ./updated-skill.yaml --force

# Import all skills from a local directory
sr import ./my-skills/
```

#### Output

**Text format (single file):**
```
Skill imported successfully

  Source:       https://example.com/skills/code-review.yaml
  Type:         url
  Skill Name:   code-review
  Destination:  /Users/user/.skillrunner/skills/code-review.yaml
```

**Text format (git repository):**
```
Skill imported successfully

  Source:       https://github.com/user/skillrunner-skills.git
  Type:         git
  Skill Name:   code-review, summarize, translate
  Destination:  /Users/user/.skillrunner/skills

Imported 3 skill(s):
  • code-review
  • summarize
  • translate
```

**JSON format:**
```json
{
  "source": "https://example.com/skills/code-review.yaml",
  "source_type": "url",
  "destination": "/Users/user/.skillrunner/skills/code-review.yaml",
  "skill_name": "code-review",
  "success": true,
  "message": "Skill imported successfully"
}
```

#### Git Repository Import Behavior

When importing from a git repository:
1. Repository is cloned to a temporary directory
2. Searches for YAML files in `skills/` subdirectory (or root if not found)
3. All `.yaml` and `.yml` files are imported
4. Temporary directory is cleaned up after import

#### Notes

- Requires `git` to be installed and in PATH for git repository imports
- Existing skills are skipped unless `--force` flag is used
- When importing directories, all `.yaml` and `.yml` files are processed
- The `--name` flag only works for single file imports, not directories
- Skills directory (`~/.skillrunner/skills/`) is created automatically if it doesn't exist

---

### metrics

Display usage statistics and cost metrics.

#### Synopsis

```bash
sr metrics [flags]
```

#### Description

Shows usage statistics and cost metrics for skillrunner, including:
- Total requests and success/failure rates
- Tokens used per provider (input and output)
- Estimated costs per provider
- Top skills by usage
- Provider-specific latency and performance metrics

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--since` | string | `24h` | Time range for metrics (e.g., `24h`, `7d`, `30d`) |

#### Time Range Format

| Format | Description | Example |
|--------|-------------|---------|
| `Xh` | Hours | `24h`, `48h` |
| `Xd` | Days | `7d`, `30d` |
| Standard duration | Go duration format | `2h30m`, `168h` |

#### Examples

```bash
# Show metrics for the last 24 hours
sr metrics --since 24h

# Show metrics for the last 7 days
sr metrics --since 7d

# Show metrics for the last 30 days
sr metrics --since 30d

# Get metrics as JSON for scripting
sr metrics --since 30d -o json
```

#### Output

**Text format:**
```
Skillrunner Metrics

  Period:  Dec 19, 2025 14:30 to Dec 26, 2025 14:30

Summary

  Total Requests:  150
  Success Rate:    94.7% (142 successful, 8 failed)
  Total Tokens:    485.0K input, 127.0K output
  Estimated Cost:  $2.47

Provider Usage

Provider    Type    Requests  Success   Tokens In   Tokens Out  Cost      Avg Latency
ollama      local         89    98.9%      245.0K       67.0K    $0.00          45ms
anthropic   cloud         42    97.6%      156.0K       38.0K    $1.85         320ms
openai      cloud         15    80.0%       68.0K       18.0K    $0.52         445ms
groq        cloud          4    25.0%       16.0K        4.0K    $0.10          89ms

Top Skills

Skill           Executions  Success Rate  Avg Duration
code-review             45         97.8%          4.2s
test-gen                38         94.7%          6.8s
doc-gen                 32        100.0%          3.1s
```

**JSON format:**
```json
{
  "period": "24h",
  "start_date": "2025-12-25T14:30:00Z",
  "end_date": "2025-12-26T14:30:00Z",
  "total_requests": 150,
  "successful_count": 142,
  "failed_count": 8,
  "success_rate": 94.67,
  "total_tokens_input": 485000,
  "total_tokens_output": 127000,
  "total_estimated_cost": 2.47,
  "provider_metrics": [
    {
      "name": "ollama",
      "type": "local",
      "total_requests": 89,
      "successful_count": 88,
      "failed_count": 1,
      "tokens_input": 245000,
      "tokens_output": 67000,
      "estimated_cost": 0.00,
      "avg_latency_ms": 45
    },
    ...
  ],
  "top_skills": [
    {
      "name": "code-review",
      "executions": 45,
      "success_rate": 97.8,
      "avg_duration": "4.2s"
    },
    ...
  ]
}
```

#### Cost Estimation

Cost estimates are calculated based on:
- Provider-specific token pricing
- Input and output token counts
- Current pricing as of the build date

**Note:** Local providers (Ollama) have zero cost.

#### Notes

- Metrics are aggregated from execution logs and telemetry
- Success rate includes both successful completions and partial successes
- Latency is measured as time-to-first-token for streaming responses
- Costs are estimates and may not reflect exact billing

---

## Exit Codes

Skillrunner uses standard exit codes:

| Code | Description |
|------|-------------|
| `0` | Success |
| `1` | General error |

Error details are written to stderr in text format or included in JSON output.

---

## Environment Variables

Skillrunner respects the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SKILLRUNNER_CONFIG` | Path to config file | `~/.skillrunner/config.yaml` |
| `SKILLRUNNER_SKILLS_DIR` | Path to skills directory | `~/.skillrunner/skills` |
| `NO_COLOR` | Disable colored output | (not set) |

### Examples

```bash
# Use custom config location
export SKILLRUNNER_CONFIG=/etc/skillrunner/config.yaml
sr list

# Use custom skills directory
export SKILLRUNNER_SKILLS_DIR=/usr/local/share/skillrunner/skills
sr list

# Disable colors
export NO_COLOR=1
sr status
```

---

## Configuration File

The configuration file is located at `~/.skillrunner/config.yaml` by default and can be generated using `sr init`.

### Example Configuration

```yaml
# Skillrunner Configuration
# Generated by 'sr init'

providers:
  ollama:
    enabled: true
    url: http://localhost:11434
    timeout: 30s

  anthropic:
    enabled: true
    api_key_encrypted: <encrypted-key>
    timeout: 60s

  openai:
    enabled: true
    api_key_encrypted: <encrypted-key>
    timeout: 60s

  groq:
    enabled: false
    api_key_encrypted: <encrypted-key>
    timeout: 30s

routing:
  default_profile: balanced
  local_first: true
  max_retries: 3

telemetry:
  enabled: true
  anonymous: true
```

---

## Skills Directory

Skills are stored in `~/.skillrunner/skills/` as YAML files. Each skill defines a multi-phase workflow.

### Example Skill Structure

```yaml
name: code-review
description: Analyze code for quality, security, and best practices
version: 1.0.0

phases:
  - name: analysis
    model: llama3
    prompt: "Analyze the following code..."

  - name: recommendations
    model: claude-3-opus
    prompt: "Based on the analysis, provide recommendations..."
    depends_on:
      - analysis

routing:
  profile: quality-first
  fallback: balanced
```

---

## Common Workflows

### First-Time Setup

```bash
# 1. Initialize configuration
sr init

# 2. Check system status
sr status

# 3. List available skills
sr list

# 4. Run your first skill
sr run summarize "Hello, world!"
```

### Importing Skills

```bash
# Import from a git repository
sr import https://github.com/skillrunner/skills.git

# Import a single skill
sr import https://example.com/skills/my-skill.yaml

# Import local skills
sr import ./custom-skills/
```

### Running Skills

```bash
# Quick question (single-phase)
sr ask code-review "Is this function safe?"

# Full workflow execution
sr run code-review "Review this PR for security issues"

# Premium quality execution
sr run code-review "Detailed code review" --profile premium

# Cost-optimized execution
sr run summarize "Quick summary" --profile cheap
```

### Monitoring

```bash
# Check system health
sr status --detailed

# View usage metrics
sr metrics --since 7d

# Export metrics for analysis
sr metrics --since 30d -o json > metrics.json
```

---

## Troubleshooting

### Common Issues

**Config not found:**
```bash
# Solution: Initialize configuration
sr init
```

**Provider unavailable:**
```bash
# Check provider status
sr status --detailed

# Verify configuration
cat ~/.skillrunner/config.yaml
```

**Skill not found:**
```bash
# List available skills
sr list

# Import skills
sr import https://github.com/skillrunner/skills.git
```

**Permission denied:**
```bash
# Check directory permissions
ls -la ~/.skillrunner/

# Recreate with correct permissions
sr init --force
```

---

## Additional Resources

- **Documentation:** https://github.com/jbctechsolutions/skillrunner
- **Issues:** https://github.com/jbctechsolutions/skillrunner/issues
- **Skills Repository:** https://github.com/skillrunner/skills

---

*Generated for Skillrunner v0.1.0-dev*
