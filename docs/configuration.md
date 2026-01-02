# Configuration Reference

Skillrunner uses YAML-based configuration to manage provider settings, routing profiles, logging, and skills. This document provides a comprehensive guide to all configuration options.

## Table of Contents

1. [Configuration File Location](#configuration-file-location)
2. [Quick Start](#quick-start)
3. [Provider Configuration](#provider-configuration)
4. [Routing Configuration](#routing-configuration)
5. [Logging Configuration](#logging-configuration)
6. [Skills Configuration](#skills-configuration)
7. [Cache Configuration](#cache-configuration)
8. [Observability Configuration](#observability-configuration)
9. [Environment Variables](#environment-variables)
10. [Complete Example](#complete-example)
11. [Security Best Practices](#security-best-practices)
12. [Advanced Topics](#advanced-topics)

---

## Configuration File Location

### Default Location

By default, Skillrunner looks for its configuration file at:

```
~/.skillrunner/config.yaml
```

### Custom Configuration Path

You can override the default location using the `--config` flag:

```bash
sr --config /path/to/custom/config.yaml run my-skill
# or using short form
sr -c /path/to/custom/config.yaml run my-skill
```

### Directory Structure

When initialized, Skillrunner creates the following directory structure:

```
~/.skillrunner/
├── config.yaml          # Main configuration file
└── skills/             # Directory for skill definitions
```

---

## Quick Start

### Initialize Configuration

The easiest way to get started is to use the interactive initialization command:

```bash
sr init
```

This will:
- Create the `~/.skillrunner/` directory structure
- Generate a `config.yaml` file with your provider settings
- Prompt for Ollama endpoint and optional cloud provider API keys
- Create the skills directory

### Force Overwrite

To reinitialize an existing configuration:

```bash
sr init --force
```

---

## Provider Configuration

Skillrunner supports multiple LLM providers: Ollama (local), Anthropic, OpenAI, and Groq (cloud-based).

### Provider Configuration Structure

```yaml
providers:
  ollama:      # Local provider configuration
  anthropic:   # Cloud provider configuration
  openai:      # Cloud provider configuration
  groq:        # Cloud provider configuration
```

### Ollama (Local Provider)

Ollama is the default local LLM provider and is enabled by default.

```yaml
providers:
  ollama:
    url: http://localhost:11434    # Ollama server endpoint
    enabled: true                  # Enable/disable provider
    timeout: 30s                   # Request timeout duration
```

**Configuration Options:**

| Option | Type | Default | Required | Description |
|--------|------|---------|----------|-------------|
| `url` | string | `http://localhost:11434` | Yes (when enabled) | Ollama server endpoint URL |
| `enabled` | boolean | `true` | Yes | Whether this provider is active |
| `timeout` | duration | `30s` | No | Maximum time to wait for requests |

**Example:**

```yaml
providers:
  ollama:
    url: http://192.168.1.100:11434  # Remote Ollama server
    enabled: true
    timeout: 45s
```

### Cloud Providers (Anthropic, OpenAI, Groq)

Cloud providers share a common configuration structure but are disabled by default.

```yaml
providers:
  anthropic:
    api_key_encrypted: ""          # Encrypted API key
    enabled: false                 # Enable/disable provider
    timeout: 60s                   # Request timeout duration
```

**Configuration Options:**

| Option | Type | Default | Required | Description |
|--------|------|---------|----------|-------------|
| `api_key_encrypted` | string | `""` | Yes (when enabled) | Encrypted API key for authentication |
| `enabled` | boolean | `false` | Yes | Whether this provider is active |
| `timeout` | duration | `60s` | No | Maximum time to wait for requests |

**Example - Multiple Cloud Providers:**

```yaml
providers:
  anthropic:
    api_key_encrypted: "encrypted_key_here"
    enabled: true
    timeout: 60s

  openai:
    api_key_encrypted: "encrypted_key_here"
    enabled: true
    timeout: 60s

  groq:
    api_key_encrypted: "encrypted_key_here"
    enabled: false
    timeout: 30s
```

### Provider Timeout Values

Timeouts are specified as duration strings:

- `30s` - 30 seconds
- `1m` - 1 minute
- `1m30s` - 1 minute 30 seconds
- `2m` - 2 minutes

**Recommended Timeout Values:**

- **Ollama (local):** `30s` - Local models respond quickly
- **Cloud providers:** `60s` - Account for network latency and API rate limits
- **Large context requests:** `2m` - When sending very large prompts

### Provider Validation Rules

When enabled, each provider must satisfy certain requirements:

**Ollama:**
- `url` must be specified and non-empty
- `timeout` must be non-negative

**Cloud Providers:**
- `api_key_encrypted` must be specified and non-empty
- `timeout` must be non-negative

---

## Routing Configuration

Routing configuration determines which models are used for different phases of skill execution. Skillrunner supports three built-in profiles: `cheap`, `balanced`, and `premium`.

### Basic Routing Configuration

```yaml
routing:
  default_profile: balanced  # cheap, balanced, or premium
```

**Configuration Options:**

| Option | Type | Default | Required | Description |
|--------|------|---------|----------|-------------|
| `default_profile` | string | `balanced` | Yes | Default routing profile to use |

### Routing Profiles

Skillrunner provides three pre-configured routing profiles optimized for different use cases:

#### 1. Cheap Profile

Optimized for cost-effectiveness using smaller local models.

**Default Configuration:**
- **Generation Model:** `llama3.2:3b`
- **Review Model:** `llama3.2:3b`
- **Fallback Model:** `llama3.2:1b`
- **Max Context Tokens:** `4096`
- **Prefer Local:** `true`

**Best For:**
- Development and testing
- Simple tasks with limited context
- Cost-sensitive workloads
- Quick iterations

```yaml
routing:
  default_profile: cheap
```

#### 2. Balanced Profile (Default)

Optimized for a balance between performance and cost.

**Default Configuration:**
- **Generation Model:** `llama3.2:8b`
- **Review Model:** `llama3.2:8b`
- **Fallback Model:** `llama3.2:3b`
- **Max Context Tokens:** `8192`
- **Prefer Local:** `true`

**Best For:**
- General-purpose workflows
- Most production workloads
- Good quality with reasonable cost
- Local-first execution

```yaml
routing:
  default_profile: balanced
```

#### 3. Premium Profile

Optimized for maximum quality using state-of-the-art cloud models.

**Default Configuration:**
- **Generation Model:** `claude-3-5-sonnet-20241022`
- **Review Model:** `gpt-4o`
- **Fallback Model:** `llama3.2:70b`
- **Max Context Tokens:** `128000`
- **Prefer Local:** `false`

**Best For:**
- Complex reasoning tasks
- Large context requirements
- Critical production workloads
- Maximum quality over cost

```yaml
routing:
  default_profile: premium
```

### How Routing Works

1. **Profile Selection:** The `default_profile` determines which pre-configured profile to use
2. **Phase-Based Selection:** Different models can be used for generation vs. review phases
3. **Fallback Handling:** If the primary model is unavailable, the fallback model is used
4. **Local Preference:** When `prefer_local` is true, local models are prioritized over cloud models
5. **Context Management:** `max_context_tokens` limits the size of context sent to models

### Advanced Routing Configuration

For advanced use cases, you can define custom routing configurations with specific providers, models, and rate limits. See the [Advanced Routing Configuration](#advanced-routing-configuration) section for details.

---

## Logging Configuration

Configure how Skillrunner outputs logs and diagnostic information.

### Configuration Options

```yaml
logging:
  level: info     # debug, info, warn, error
  format: text    # text, json
```

| Option | Type | Default | Valid Values | Description |
|--------|------|---------|--------------|-------------|
| `level` | string | `info` | `debug`, `info`, `warn`, `error` | Minimum log level to display |
| `format` | string | `text` | `text`, `json` | Output format for logs |

### Log Levels

#### debug
Shows all log messages including detailed debugging information.

**Use When:**
- Troubleshooting issues
- Developing skills
- Understanding execution flow

```yaml
logging:
  level: debug
```

**Example Output:**
```
[DEBUG] Loading skill from /path/to/skill.yaml
[DEBUG] Validating skill configuration
[DEBUG] Initializing provider: ollama
[INFO] Executing skill: my-skill
```

#### info (Default)
Shows informational messages and above (info, warn, error).

**Use When:**
- Normal operation
- Monitoring execution
- Production environments

```yaml
logging:
  level: info
```

**Example Output:**
```
[INFO] Executing skill: my-skill
[INFO] Phase 1: generation completed
[WARN] High token usage detected
```

#### warn
Shows warnings and errors only.

**Use When:**
- Production with minimal logging
- Focusing on issues
- Reducing log noise

```yaml
logging:
  level: warn
```

#### error
Shows only error messages.

**Use When:**
- Critical production systems
- Minimal logging requirements
- Error-only monitoring

```yaml
logging:
  level: error
```

### Log Formats

#### text (Default)
Human-readable text format with colors (when supported).

```yaml
logging:
  format: text
```

**Example Output:**
```
[INFO] Executing skill: my-skill
[SUCCESS] Skill completed successfully
```

#### json
Structured JSON format for machine parsing and log aggregation.

```yaml
logging:
  format: json
```

**Example Output:**
```json
{"level":"info","msg":"Executing skill: my-skill","time":"2024-01-15T10:30:00Z"}
{"level":"info","msg":"Skill completed successfully","time":"2024-01-15T10:30:15Z"}
```

**Use When:**
- Integrating with log aggregation systems (ELK, Splunk, etc.)
- Automated log parsing
- Production monitoring

### Logging Best Practices

1. **Development:** Use `debug` level with `text` format
2. **Production:** Use `info` or `warn` level with `json` format
3. **Troubleshooting:** Temporarily switch to `debug` level
4. **CI/CD:** Use `json` format for easier parsing

---

## Skills Configuration

Configure where Skillrunner looks for skill definition files.

### Configuration Options

```yaml
skills:
  directory: ~/.skillrunner/skills
```

| Option | Type | Default | Required | Description |
|--------|------|---------|----------|-------------|
| `directory` | string | `~/.skillrunner/skills` | Yes | Path to directory containing skill YAML files |

### Directory Structure

Skills are stored as YAML files in the configured directory:

```
~/.skillrunner/skills/
├── code-review.yaml
├── documentation.yaml
└── testing.yaml
```

### Custom Skills Directory

You can specify a custom location for your skills:

```yaml
skills:
  directory: /path/to/my/skills
```

Or use relative paths:

```yaml
skills:
  directory: ./project-skills
```

### Loading Behavior

- **Automatic Discovery:** Skillrunner automatically scans the directory for `.yaml` files
- **Recursive Search:** Subdirectories are not searched by default
- **Validation:** Each skill file is validated on load
- **Error Handling:** Invalid skills are logged but don't prevent other skills from loading

### Multiple Skill Directories

Currently, Skillrunner supports a single skill directory. To use skills from multiple locations:

1. Create symbolic links in your main skills directory
2. Copy skills to the main directory
3. Use a build script to aggregate skills

**Example using symbolic links:**

```bash
cd ~/.skillrunner/skills
ln -s /path/to/project/skills/custom-skill.yaml
```

---

## Cache Configuration

Skillrunner includes a two-tier caching system for LLM responses to improve performance and reduce costs.

### Configuration Options

```yaml
cache:
  enabled: true              # Enable/disable caching
  max_memory_size: 104857600 # L1 memory cache size in bytes (100MB)
  max_disk_size: 1073741824  # L2 SQLite cache size in bytes (1GB)
  default_ttl: 1h            # Default time-to-live for cache entries
  cleanup_period: 5m         # How often to clean expired entries
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable the caching system |
| `max_memory_size` | int | `104857600` (100MB) | Maximum size of in-memory L1 cache |
| `max_disk_size` | int | `1073741824` (1GB) | Maximum size of SQLite L2 cache |
| `default_ttl` | duration | `1h` | Default time-to-live for cached responses |
| `cleanup_period` | duration | `5m` | Interval for cache cleanup operations |

### Cache Architecture

Skillrunner uses a two-tier cache architecture:

1. **L1 Memory Cache** - Fast in-memory cache for frequently accessed responses
2. **L2 SQLite Cache** - Persistent disk-based cache for larger capacity

When a cache lookup occurs:
1. Check L1 (memory) first
2. If L1 miss, check L2 (SQLite)
3. On L2 hit, promote to L1
4. On cache miss, fetch from provider

### Cache Key Generation

Cache keys are generated from:
- Provider name
- Model ID
- Request messages (hashed)
- Temperature and other parameters

This ensures that identical requests return cached responses while different parameters trigger new requests.

### Example Configuration

```yaml
cache:
  enabled: true
  max_memory_size: 209715200   # 200MB for high-traffic scenarios
  max_disk_size: 5368709120    # 5GB for extensive caching
  default_ttl: 24h             # Cache for 24 hours
  cleanup_period: 10m
```

### Disabling Cache

To disable caching entirely:

```yaml
cache:
  enabled: false
```

---

## Observability Configuration

Skillrunner provides comprehensive observability features including structured logging, distributed tracing, and metrics collection.

### Configuration Structure

```yaml
observability:
  metrics:
    enabled: true
    aggregation_level: standard  # minimal, standard, or debug
  tracing:
    enabled: false
    exporter_type: stdout        # stdout, otlp, or none
    otlp_endpoint: ""
    service_name: skillrunner
    sample_rate: 1.0
```

### Metrics Configuration

Metrics track execution statistics, token usage, costs, and performance data.

```yaml
observability:
  metrics:
    enabled: true
    aggregation_level: standard
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable metrics collection |
| `aggregation_level` | string | `standard` | Level of detail: `minimal`, `standard`, `debug` |

**Aggregation Levels:**

| Level | Description |
|-------|-------------|
| `minimal` | Basic counts and totals only |
| `standard` | Includes per-provider and per-skill breakdowns |
| `debug` | Full detail including phase-level metrics |

### Tracing Configuration

Distributed tracing provides visibility into workflow execution using OpenTelemetry.

```yaml
observability:
  tracing:
    enabled: true
    exporter_type: otlp
    otlp_endpoint: http://localhost:4317
    service_name: skillrunner
    sample_rate: 1.0
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable/disable distributed tracing |
| `exporter_type` | string | `stdout` | Trace exporter: `stdout`, `otlp`, `none` |
| `otlp_endpoint` | string | `""` | OTLP collector endpoint (required for `otlp` exporter) |
| `service_name` | string | `skillrunner` | Service name in traces |
| `sample_rate` | float | `1.0` | Sampling rate (0.0-1.0, where 1.0 = 100%) |

**Exporter Types:**

| Type | Description | Use Case |
|------|-------------|----------|
| `stdout` | Prints traces to console | Development and debugging |
| `otlp` | Sends to OTLP collector | Production with Jaeger, Tempo, etc. |
| `none` | No-op exporter | Disable tracing without code changes |

### Trace Hierarchy

Skillrunner creates the following span hierarchy:

```
workflow:{skill_name}
  ├── phase:{phase_id}
  │   └── provider:{provider_name}
  └── phase:{phase_id}
      └── provider:{provider_name}
```

**Span Attributes:**

- **Workflow spans**: skill_id, skill_name, phase_count, total_tokens, cost
- **Phase spans**: phase_id, phase_name, tokens, cache_hit, duration
- **Provider spans**: provider, model, output_tokens, finish_reason

### Example Configurations

**Development (verbose logging):**

```yaml
observability:
  metrics:
    enabled: true
    aggregation_level: debug
  tracing:
    enabled: true
    exporter_type: stdout
    service_name: skillrunner-dev
    sample_rate: 1.0
```

**Production with Jaeger:**

```yaml
observability:
  metrics:
    enabled: true
    aggregation_level: standard
  tracing:
    enabled: true
    exporter_type: otlp
    otlp_endpoint: http://jaeger:4317
    service_name: skillrunner
    sample_rate: 0.1  # Sample 10% of traces
```

**Minimal (metrics only):**

```yaml
observability:
  metrics:
    enabled: true
    aggregation_level: minimal
  tracing:
    enabled: false
```

### Viewing Metrics

Use the `sr metrics` command to view collected metrics:

```bash
# View last 24 hours
sr metrics --since 24h

# View last 7 days as JSON
sr metrics --since 7d -o json
```

Metrics include:
- Total executions and success rates
- Token usage per provider
- Estimated costs
- Top skills by usage
- Provider latency

### Cost Tracking

Skillrunner automatically calculates costs based on provider pricing:

| Provider | Input Cost (per 1M tokens) | Output Cost (per 1M tokens) |
|----------|---------------------------|----------------------------|
| Ollama | $0.00 | $0.00 |
| Anthropic (Claude 3.5 Sonnet) | $3.00 | $15.00 |
| OpenAI (GPT-4o) | $2.50 | $10.00 |
| Groq | $0.05 | $0.10 |

Costs are calculated per-phase and aggregated for reporting.

---

## Environment Variables

Skillrunner currently has limited environment variable support. Configuration is primarily file-based.

### Supported Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HOME` | User home directory (used to locate `~/.skillrunner/`) | System default |

### Future Environment Variable Support

The following environment variables are planned for future releases:

- `SKILLRUNNER_CONFIG` - Override default config file location
- `SKILLRUNNER_LOG_LEVEL` - Override log level from config
- `SKILLRUNNER_LOG_FORMAT` - Override log format from config
- `SKILLRUNNER_SKILLS_DIR` - Override skills directory

---

## Complete Example

Here's a fully annotated configuration file demonstrating all available options:

```yaml
# Skillrunner Configuration
# Generated by 'sr init'
#
# Documentation: https://github.com/jbctechsolutions/skillrunner

# Provider Configuration
# Configure LLM providers for skill execution
providers:
  # Ollama - Local LLM provider
  # Enabled by default for privacy and cost-effectiveness
  ollama:
    url: http://localhost:11434    # Ollama server endpoint
    enabled: true                  # Enable Ollama provider
    timeout: 30s                   # Request timeout (local is fast)

  # Anthropic - Claude API
  # Premium cloud provider for high-quality reasoning
  anthropic:
    api_key_encrypted: ""          # Set via 'sr config set anthropic.api_key'
    enabled: false                 # Disabled by default
    timeout: 60s                   # Longer timeout for cloud API

  # OpenAI - GPT API
  # Popular cloud provider with various model options
  openai:
    api_key_encrypted: ""          # Set via 'sr config set openai.api_key'
    enabled: false                 # Disabled by default
    timeout: 60s                   # Longer timeout for cloud API

  # Groq - Fast inference API
  # Cloud provider optimized for speed
  groq:
    api_key_encrypted: ""          # Set via 'sr config set groq.api_key'
    enabled: false                 # Disabled by default
    timeout: 30s                   # Groq is typically fast

# Routing Configuration
# Determines which models to use for skill execution
routing:
  default_profile: balanced        # cheap | balanced | premium
  # cheap:    Small local models, low cost, fast
  # balanced: Medium local models, good quality/cost ratio (default)
  # premium:  Cloud models, highest quality, higher cost

# Logging Configuration
# Controls log output verbosity and format
logging:
  level: info                      # debug | info | warn | error
  format: text                     # text | json
  # Use 'debug' for troubleshooting
  # Use 'json' for log aggregation systems

# Skills Configuration
# Location of skill definition files
skills:
  directory: ~/.skillrunner/skills  # Path to skills directory
  # Skills are YAML files that define multi-phase AI workflows

# Cache Configuration
# Two-tier caching for LLM responses
cache:
  enabled: true                    # Enable response caching
  max_memory_size: 104857600       # L1 memory cache: 100MB
  max_disk_size: 1073741824        # L2 SQLite cache: 1GB
  default_ttl: 1h                  # Cache entry lifetime
  cleanup_period: 5m               # Cleanup interval

# Observability Configuration
# Metrics, tracing, and structured logging
observability:
  metrics:
    enabled: true                  # Enable metrics collection
    aggregation_level: standard    # minimal | standard | debug
  tracing:
    enabled: false                 # Enable distributed tracing
    exporter_type: stdout          # stdout | otlp | none
    otlp_endpoint: ""              # OTLP collector endpoint
    service_name: skillrunner      # Service name in traces
    sample_rate: 1.0               # Trace sampling rate (0.0-1.0)
```

### Minimal Configuration

The minimum required configuration:

```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
    timeout: 30s

routing:
  default_profile: balanced

logging:
  level: info
  format: text

skills:
  directory: ~/.skillrunner/skills
```

### Production Configuration

Recommended configuration for production use:

```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
    timeout: 30s

  anthropic:
    api_key_encrypted: "encrypted_key_here"
    enabled: true
    timeout: 60s

  openai:
    api_key_encrypted: "encrypted_key_here"
    enabled: true
    timeout: 60s

routing:
  default_profile: balanced

logging:
  level: warn                      # Reduce log noise
  format: json                     # Enable log aggregation

skills:
  directory: /opt/skillrunner/skills  # Absolute path

cache:
  enabled: true
  max_memory_size: 209715200       # 200MB for production
  max_disk_size: 5368709120        # 5GB persistent cache
  default_ttl: 24h
  cleanup_period: 10m

observability:
  metrics:
    enabled: true
    aggregation_level: standard
  tracing:
    enabled: true
    exporter_type: otlp
    otlp_endpoint: http://jaeger:4317
    service_name: skillrunner
    sample_rate: 0.1               # Sample 10% of traces
```

### Development Configuration

Recommended configuration for development:

```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
    timeout: 45s                   # Slightly longer for debugging

routing:
  default_profile: cheap           # Use small models for testing

logging:
  level: debug                     # Verbose logging
  format: text                     # Human-readable output

skills:
  directory: ./skills              # Relative to project

cache:
  enabled: true
  max_memory_size: 52428800        # 50MB for development
  max_disk_size: 104857600         # 100MB
  default_ttl: 30m
  cleanup_period: 1m

observability:
  metrics:
    enabled: true
    aggregation_level: debug       # Full detail for debugging
  tracing:
    enabled: true
    exporter_type: stdout          # Print traces to console
    service_name: skillrunner-dev
    sample_rate: 1.0               # Trace everything
```

---

## Security Best Practices

### API Key Management

1. **Never commit API keys to version control**
   - Add `config.yaml` to `.gitignore`
   - Use environment-specific configurations

2. **Use encrypted storage**
   - API keys are stored in `api_key_encrypted` fields
   - Encryption is automatically handled by Skillrunner

3. **Set API keys via CLI**
   ```bash
   sr config set anthropic.api_key
   # (Prompts for key securely)
   ```

4. **Limit API key permissions**
   - Use API keys with minimal required permissions
   - Rotate keys regularly
   - Monitor API usage

### File Permissions

The configuration file contains sensitive information and should have restricted permissions:

```bash
# Recommended permissions
chmod 600 ~/.skillrunner/config.yaml
```

When Skillrunner creates the configuration file via `sr init`, it automatically sets:
- **Config file:** `0600` (read/write for owner only)
- **Config directory:** `0750` (read/write/execute for owner, read/execute for group)
- **Skills directory:** `0750` (read/write/execute for owner, read/execute for group)

### Encryption Details

- **Current Implementation:** API keys are stored with the `api_key_encrypted` field name
- **Planned Enhancement:** Full encryption will be implemented in a future release
- **Temporary Workaround:** Store actual API keys in `api_key_encrypted` field until encryption is implemented

**Note:** The `TODO: encrypt` comments in the source code indicate that full encryption is a planned feature.

### Network Security

1. **HTTPS for Cloud Providers**
   - All cloud provider APIs use HTTPS by default
   - Never disable TLS verification

2. **Localhost for Ollama**
   - Default configuration uses `localhost` for security
   - Only expose Ollama on network if necessary

3. **Firewall Configuration**
   - Restrict access to Ollama port (11434) if exposing on network
   - Use VPN for remote Ollama access

### Configuration Backup

1. **Backup your configuration**
   ```bash
   cp ~/.skillrunner/config.yaml ~/.skillrunner/config.yaml.backup
   ```

2. **Encrypt backups**
   ```bash
   gpg -c ~/.skillrunner/config.yaml.backup
   ```

3. **Store backups securely**
   - Use encrypted storage
   - Keep separate from primary system

---

## Advanced Topics

### Advanced Routing Configuration

For complex deployments, you can create a separate routing configuration file with detailed provider and model settings.

#### Routing Configuration Structure

```yaml
# Advanced routing configuration
providers:
  ollama:
    enabled: true
    priority: 1                    # Lower number = higher priority
    timeout: 30                    # Seconds
    base_url: http://localhost:11434
    models:
      llama3.2:3b:
        tier: cheap
        cost_per_input_token: 0.0
        cost_per_output_token: 0.0
        max_tokens: 2048
        context_window: 4096
        enabled: true
        capabilities:
          - text_generation
      llama3.2:8b:
        tier: balanced
        cost_per_input_token: 0.0
        cost_per_output_token: 0.0
        max_tokens: 4096
        context_window: 8192
        enabled: true

  anthropic:
    enabled: true
    priority: 2
    timeout: 60
    models:
      claude-3-5-sonnet-20241022:
        tier: premium
        cost_per_input_token: 0.000003
        cost_per_output_token: 0.000015
        max_tokens: 8192
        context_window: 200000
        enabled: true
        capabilities:
          - text_generation
          - vision
          - function_calling

default_provider: ollama

profiles:
  cheap:
    generation_model: llama3.2:3b
    review_model: llama3.2:3b
    fallback_model: llama3.2:1b
    max_context_tokens: 4096
    prefer_local: true

  balanced:
    generation_model: llama3.2:8b
    review_model: llama3.2:8b
    fallback_model: llama3.2:3b
    max_context_tokens: 8192
    prefer_local: true

  premium:
    generation_model: claude-3-5-sonnet-20241022
    review_model: gpt-4o
    fallback_model: llama3.2:70b
    max_context_tokens: 128000
    prefer_local: false

fallback_chain:
  - ollama
  - groq
  - openai
  - anthropic
```

#### Model Configuration Options

| Option | Type | Description |
|--------|------|-------------|
| `tier` | string | Cost/capability tier: `cheap`, `balanced`, or `premium` |
| `cost_per_input_token` | float | Cost per input token in USD |
| `cost_per_output_token` | float | Cost per output token in USD |
| `max_tokens` | int | Maximum tokens the model can generate |
| `context_window` | int | Maximum context size in tokens |
| `enabled` | boolean | Whether this model is available for routing |
| `capabilities` | array | Model capabilities (e.g., `vision`, `function_calling`) |
| `aliases` | array | Alternative names for this model |

#### Rate Limiting

Configure rate limits to prevent API throttling:

```yaml
providers:
  anthropic:
    enabled: true
    rate_limits:
      requests_per_minute: 50
      tokens_per_minute: 100000
      concurrent_requests: 5
      burst_limit: 10
```

#### Provider Priority

Lower priority numbers indicate higher preference:

```yaml
providers:
  ollama:
    priority: 1    # First choice
  groq:
    priority: 2    # Second choice
  openai:
    priority: 3    # Third choice
```

#### Fallback Chain

Define the order of fallback providers:

```yaml
fallback_chain:
  - ollama       # Try local first
  - groq         # Fast cloud fallback
  - openai       # Reliable fallback
  - anthropic    # Premium fallback
```

### Configuration Validation

Skillrunner validates all configuration on startup. Common validation errors:

#### Invalid Log Level
```
Error: invalid configuration: logging: invalid log level "trace": must be one of debug, info, warn, error
```

**Fix:** Use a valid log level: `debug`, `info`, `warn`, or `error`

#### Missing Required Field
```
Error: invalid configuration: routing: default_profile is required
```

**Fix:** Ensure all required fields are present in your configuration

#### Invalid Provider Configuration
```
Error: invalid configuration: providers: anthropic: api_key_encrypted is required when enabled
```

**Fix:** Provide API key or disable the provider

### Configuration Merging

When using advanced routing configurations, Skillrunner merges multiple configuration sources:

1. **Default Configuration** (built-in defaults)
2. **Main Configuration File** (`~/.skillrunner/config.yaml`)
3. **Routing Configuration File** (if specified)

Later configurations override earlier ones.

### Debugging Configuration Issues

Enable debug logging to see configuration loading:

```yaml
logging:
  level: debug
```

Or use the `--verbose` flag:

```bash
sr --verbose run my-skill
```

---

## Configuration Schema Reference

### Root Configuration

```typescript
{
  providers: ProviderConfigs,
  routing: RoutingConfig,
  logging: LoggingConfig,
  skills: SkillsConfig
}
```

### Provider Configs

```typescript
{
  ollama: OllamaConfig,
  anthropic: CloudConfig,
  openai: CloudConfig,
  groq: CloudConfig
}
```

### Ollama Config

```typescript
{
  url: string,          // Required when enabled
  enabled: boolean,     // Required
  timeout: duration     // Optional, default: 30s
}
```

### Cloud Config

```typescript
{
  api_key_encrypted: string,  // Required when enabled
  enabled: boolean,           // Required
  timeout: duration           // Optional, default: 60s
}
```

### Routing Config

```typescript
{
  default_profile: "cheap" | "balanced" | "premium"  // Required
}
```

### Logging Config

```typescript
{
  level: "debug" | "info" | "warn" | "error",  // Required
  format: "text" | "json"                       // Required
}
```

### Skills Config

```typescript
{
  directory: string  // Required
}
```

---

## Troubleshooting

### Configuration Not Found

**Error:**
```
Error: config file not found: ~/.skillrunner/config.yaml
```

**Solution:**
Run `sr init` to create a new configuration file.

### Provider Connection Issues

**Error:**
```
Error: failed to connect to ollama at http://localhost:11434
```

**Solution:**
1. Verify Ollama is running: `ollama serve`
2. Check the URL in your configuration
3. Test connectivity: `curl http://localhost:11434/api/version`

### Invalid API Key

**Error:**
```
Error: authentication failed for provider: anthropic
```

**Solution:**
1. Verify your API key is correct
2. Check API key has not expired
3. Ensure provider is enabled in configuration
4. Re-run `sr init` to reconfigure

### Timeout Issues

**Error:**
```
Error: request timeout exceeded for provider: anthropic
```

**Solution:**
Increase the timeout value in your configuration:

```yaml
providers:
  anthropic:
    timeout: 120s  # Increase to 2 minutes
```

---

## See Also

- [Getting Started Guide](getting-started.md)
- [Skills Guide](skills.md)
- [Provider Guide](providers.md)
- [CLI Reference](cli-reference.md)

---

## Version History

- **v0.3.0** - Wave 11: Observability
  - Added observability configuration (metrics, tracing)
  - Structured logging with correlation IDs
  - OpenTelemetry distributed tracing support
  - Cost tracking per provider/model

- **v0.2.0** - Wave 10: Caching & Performance
  - Added cache configuration section
  - Two-tier caching (memory + SQLite)
  - Configurable TTL and cleanup periods

- **v0.1.0** - Initial configuration schema
  - Basic provider configuration
  - Three-tier routing profiles
  - Logging and skills configuration
