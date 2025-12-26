# Getting Started with Skillrunner

Welcome to Skillrunner, a local-first AI workflow orchestration tool that brings intelligent multi-phase AI workflows to your command line.

## What is Skillrunner?

Skillrunner is a Go-based tool that enables you to execute sophisticated, multi-phase AI workflows with intelligent provider routing. It's designed around a "local-first" philosophy, prioritizing local LLM providers like Ollama while seamlessly falling back to cloud providers (Anthropic Claude, OpenAI, Groq) when needed.

### Key Benefits

- **Local-First Architecture**: Run workflows on your local machine using Ollama, reducing costs and maintaining privacy
- **Multi-Phase Workflows**: Break complex AI tasks into coordinated phases with dependency management
- **Intelligent Routing**: Automatically route phases to the best provider based on configurable profiles (cheap, balanced, premium)
- **Provider Flexibility**: Seamlessly fallback from local to cloud providers when needed
- **Skill-Based Workflows**: Pre-built and custom skill definitions for common AI tasks

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.23 or higher** - Required to build Skillrunner from source
- **Ollama** (optional but recommended) - For local LLM execution
  - Download from [ollama.ai](https://ollama.ai)
  - Start Ollama and pull models: `ollama pull llama3.2`, `ollama pull codellama`
- **API Keys** (optional) - For cloud provider access:
  - [Anthropic Claude API key](https://console.anthropic.com/)
  - [OpenAI API key](https://platform.openai.com/api-keys)
  - [Groq API key](https://console.groq.com/)

## Installation

### Building from Source

1. Clone the repository or navigate to the skillrunner directory:

```bash
cd /path/to/skillrunner
```

2. Build the binary using make:

```bash
make build
```

This will create a `skillrunner` binary in the current directory.

3. (Optional) Move the binary to your PATH:

```bash
# On macOS/Linux
sudo mv skillrunner /usr/local/bin/sr

# Or add to your PATH manually
export PATH=$PATH:/path/to/skillrunner
```

4. Verify the installation:

```bash
sr version
```

## Quick Start

### Step 1: Initialize Configuration

Initialize Skillrunner's configuration interactively:

```bash
sr init
```

This command will:
- Create `~/.skillrunner/` directory
- Create `~/.skillrunner/skills/` directory for skill definitions
- Generate `~/.skillrunner/config.yaml` with provider configurations
- Prompt you for Ollama endpoint and optional cloud provider API keys

Example interaction:

```
Skillrunner Configuration

This wizard will help you set up skillrunner.

Local Provider (Ollama)

Ollama URL [http://localhost:11434]:
Enable Ollama [Y/n]: y

Cloud Providers (Optional)

API keys will be stored encrypted in config.yaml

Configure Anthropic (Claude) [y/N]: y
Anthropic API key: sk-ant-***************

Configure OpenAI [y/N]: n
Configure Groq [y/N]: n

Configuration initialized successfully!
```

### Step 2: Check System Status

Verify that your providers are configured and healthy:

```bash
sr status
```

This displays:
- Overall system health
- Provider connectivity (Ollama, Anthropic, OpenAI, Groq)
- Available models per provider
- Configuration status
- Skill availability

Example output:

```
Skillrunner Status

  System:  ● healthy
  Version: v1.0.0

Configuration
✓ Config loaded from ~/.skillrunner/config.yaml
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

For detailed status with latency and model information:

```bash
sr status --detailed
```

### Step 3: List Available Skills

View all available skills:

```bash
sr list
```

This shows:
- Skill name
- Description
- Number of phases
- Routing profile

Example output:

```
Available Skills

NAME              DESCRIPTION                                  PHASES  ROUTING
code-review       Analyze code for quality, security, and...      3    quality-first
summarize         Generate concise summaries of documents...      2    local-first
translate         Translate text between languages with c...      2    cost-aware
extract-data      Extract structured data from unstructur...      4    performance
generate-tests    Generate unit tests for code with cover...      3    quality-first

Total: 5 skill(s)
```

Aliases are also supported:

```bash
sr ls
```

### Step 4: Run a Skill

Execute a skill with a request:

```bash
sr run code-review "Review this code for security issues"
```

Options:
- `--profile` or `-p`: Specify routing profile (cheap, balanced, premium)
- `--stream` or `-s`: Enable streaming output

Examples:

```bash
# Use default balanced profile
sr run code-review "Review this pull request"

# Use premium profile for highest quality
sr run code-review "Review this PR" --profile premium

# Use cheap profile to minimize costs
sr run summarize "Summarize this document" --profile cheap

# Enable streaming for real-time output
sr run translate "Hello, world!" --stream
```

### Step 5: Quick Ask

For simple, single-phase queries, use the `ask` command:

```bash
sr ask code-review "What security patterns should I look for?"
```

The ask command:
- Executes only the first/default phase of a skill
- Returns faster results for quick questions
- Supports model and profile overrides

Options:
- `--model` or `-m`: Override model selection (e.g., claude-3-opus, gpt-4, llama3)
- `--profile` or `-p`: Specify routing profile

Examples:

```bash
# Quick question with default settings
sr ask summarize "What are the key points of this document?"

# Override model selection
sr ask code-review "Is this function safe?" --model claude-3-opus

# Use specific profile
sr ask translate "Bonjour" --profile premium
```

## Your First Workflow: Code Review

Let's walk through a complete example using the `code-review` skill. This skill demonstrates Skillrunner's multi-phase workflow capabilities.

### The Code Review Skill

The code-review skill consists of three phases:

1. **Pattern Analysis** - Analyzes code patterns, identifies bugs, performance concerns, and best practices
2. **Security Analysis** - Performs security-focused review checking for vulnerabilities
3. **Report Generation** - Generates a comprehensive code review report

Each phase depends on the previous one, creating a coordinated workflow.

### Running a Code Review

1. Save some code to review in a file:

```bash
cat > example.go <<'EOF'
package main

import (
    "fmt"
    "os/exec"
)

func runCommand(userInput string) {
    cmd := exec.Command("sh", "-c", userInput)
    output, _ := cmd.Output()
    fmt.Println(string(output))
}
EOF
```

2. Run the code review:

```bash
sr run code-review "Review the code in example.go for security issues"
```

3. Skillrunner will:
   - Execute the **analyze** phase using Claude (premium routing)
   - Pass results to the **security** phase for vulnerability analysis
   - Generate a comprehensive **report** with findings and recommendations

### Understanding the Output

The code review will produce:

- **Executive Summary**: Brief overview of code quality and key findings
- **Code Quality Assessment**: Overall quality score, strengths, and areas for improvement
- **Issues Found**: Prioritized list of issues with severity levels
- **Security Findings**: Security concerns with remediation priorities (will identify command injection vulnerability)
- **Recommendations**: Actionable next steps
- **Conclusion**: Final assessment and approval recommendation

## Routing Profiles

Skillrunner supports three routing profiles that control how work is distributed across providers:

### Cheap Profile

```bash
sr run summarize "Summarize this text" --profile cheap
```

- **Priority**: Minimize costs
- **Behavior**: Uses local providers (Ollama) when possible
- **Use Cases**: Quick drafts, experimentation, high-volume tasks
- **Tradeoffs**: May produce lower quality results than cloud models

### Balanced Profile (Default)

```bash
sr run code-review "Review this code" --profile balanced
```

- **Priority**: Balance between cost and quality
- **Behavior**: Uses local for simple tasks, cloud for complex analysis
- **Use Cases**: General-purpose workflows, most production use
- **Tradeoffs**: Good quality-to-cost ratio

### Premium Profile

```bash
sr run code-review "Review critical security code" --profile premium
```

- **Priority**: Maximum quality
- **Behavior**: Routes to best available models (Claude, GPT-4)
- **Use Cases**: Critical code reviews, important documents, security analysis
- **Tradeoffs**: Higher costs but best quality

### Skill-Specific Routing

Skills can define their own default routing profiles. For example, the `code-review` skill defaults to `premium` to ensure thorough analysis:

```yaml
routing:
  default_profile: premium  # Quality-first approach for thorough reviews
  generation_model: claude  # Prefer Claude for high-quality analysis
```

You can override skill defaults with the `--profile` flag.

## Configuration

### Configuration File Location

Skillrunner stores configuration at:

```
~/.skillrunner/config.yaml
```

### Configuration Structure

```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
    timeout: 30s
  anthropic:
    api_key_encrypted: ""  # Set via 'sr config set anthropic.api_key'
    enabled: false
    timeout: 60s
  openai:
    api_key_encrypted: ""
    enabled: false
    timeout: 60s
  groq:
    api_key_encrypted: ""
    enabled: false
    timeout: 30s

routing:
  default_profile: balanced  # cheap, balanced, premium

logging:
  level: info   # debug, info, warn, error
  format: text  # json, text

skills:
  directory: ~/.skillrunner/skills
```

### Managing API Keys

API keys are stored encrypted in the configuration file. Future versions will support secure configuration management via:

```bash
sr config set anthropic.api_key
sr config set openai.api_key
sr config set groq.api_key
```

## Skills Directory

Skills are defined in YAML files located in:

```
~/.skillrunner/skills/
```

### Example Skill Structure

```yaml
id: code-review
name: Code Review
version: "1.0.0"
description: |
  Performs a comprehensive multi-phase code review including pattern analysis,
  security vulnerability checks, and generates a detailed review summary.

routing:
  default_profile: premium
  max_context_tokens: 32000

phases:
  - id: analyze
    name: Code Pattern Analysis
    prompt_template: |
      You are a senior software engineer performing a thorough code review.

      Analyze the following code and identify:
      1. Code patterns and architectural decisions
      2. Potential bugs or logic errors
      3. Performance concerns
      4. Code style and readability issues
      5. Adherence to best practices

      Code to review:
      {{.input}}
    routing_profile: premium
    max_tokens: 4096
    temperature: 0.3

  - id: security
    name: Security Analysis
    depends_on:
      - analyze
    # ... additional configuration
```

### Built-in Skills

Skillrunner includes several built-in skills:

- **code-review**: Comprehensive code quality and security analysis
- **doc-gen**: Generate documentation from code
- **test-gen**: Generate unit tests with coverage analysis

## Output Formats

Skillrunner supports multiple output formats for scripting and automation:

### Text Output (Default)

Human-readable output with colors and formatting:

```bash
sr status
```

### JSON Output

Machine-readable JSON for scripting:

```bash
sr status --output json
sr list -o json
```

Example JSON output:

```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "providers": [
    {
      "name": "ollama",
      "type": "local",
      "status": "healthy",
      "endpoint": "http://localhost:11434",
      "models": ["llama3.2:latest", "codellama:13b"]
    }
  ],
  "config_loaded": true,
  "skill_count": 3
}
```

### Table Output

Structured table format:

```bash
sr list --format table
```

## Next Steps

Now that you have Skillrunner set up, explore these topics:

1. **CLI Reference** - Learn all available commands and options
   - `sr --help`
   - `sr run --help`
   - `sr ask --help`

2. **Creating Custom Skills** - Build your own multi-phase workflows
   - Review skill YAML format
   - Understand phase dependencies
   - Configure routing strategies

3. **Provider Configuration** - Fine-tune provider settings
   - Configure model preferences
   - Set timeouts and retries
   - Manage API keys securely

4. **Advanced Workflows** - Leverage advanced features
   - Chain multiple skills together
   - Use template variables in prompts
   - Optimize for cost or quality

## Common Issues

### Ollama Not Running

```
Error: cannot connect to Ollama at http://localhost:11434
```

**Solution**: Start Ollama:
```bash
# macOS
ollama serve

# Linux/Docker
docker run -d -v ollama:/root/.ollama -p 11434:11434 --name ollama ollama/ollama
```

### No Models Available

```
Error: no models available for provider ollama
```

**Solution**: Pull models:
```bash
ollama pull llama3.2
ollama pull codellama
ollama pull mistral
```

### API Key Not Configured

```
Error: API key not configured for provider anthropic
```

**Solution**: Run initialization again or manually edit config:
```bash
sr init --force
```

## Getting Help

- **Command Help**: Run any command with `--help` flag
  ```bash
  sr --help
  sr run --help
  sr status --help
  ```

- **Version Information**: Check your version
  ```bash
  sr version
  ```

- **Status Diagnostics**: Check provider health
  ```bash
  sr status --detailed
  ```

- **GitHub Repository**: [github.com/jbctechsolutions/skillrunner](https://github.com/jbctechsolutions/skillrunner)

## Summary

You've learned how to:

- Install and configure Skillrunner
- Check system status and provider health
- List and run skills
- Use quick ask mode for simple queries
- Understand routing profiles
- Execute a complete code review workflow

Skillrunner brings the power of multi-phase AI workflows to your command line, with intelligent routing that balances quality, cost, and privacy. Start exploring the available skills and create your own custom workflows!
