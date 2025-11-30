# Skillrunner Quick Start Guide

## Overview

Skillrunner is a multi-LLM orchestration system that enables:
- **70-90% cost savings** by using free local Ollama models
- **Multi-phase workflows** with dependency management and parallel execution
- **Intelligent context management** for large inputs
- **Any LLM support** - Ollama (local/free), Anthropic Claude, OpenAI

> **Important**: All commands in this guide should be run from the **project root directory** (where `go.mod` is located), not from the `docs/` folder.

## Prerequisites

### 1. Install Ollama

```bash
# macOS
brew install ollama

# Linux
curl https://ollama.ai/install.sh | sh

# Start Ollama service
ollama serve
```

### 2. Pull Required Models

```bash
# Core model (required for most skills)
ollama pull qwen2.5:14b

# Code generation model (optional, for bug-fix and feature-implementation)
ollama pull qwen2.5-coder:32b

# Additional model (optional, for converted marketplace skills)
ollama pull deepseek-coder:33b
```

### 3. Build Skillrunner

```bash
# Clone the repository
git clone https://github.com/jbctechsolutions/skillrunner
cd skillrunner

# Build the binary
go build -o sr ./cmd/skillrunner/
```

## Quick Test

### 1. Run Hello Orchestration (4 phases)

```bash
sr run hello-orchestration "a developer learning Go"
```

**Expected Output:**
- Executes 4 phases in 3 batches
- Batch 2 runs 2 phases in parallel (tip || quote)
- Total duration: ~8-10s
- Cost: $0.00 (100% local)

### 2. Ask a Marketplace Skill

```bash
sr ask avl-expert "What microphones work best for a 100-person venue?"
```

**Expected Output:**
- Uses ollama/qwen2.5:14b (free)
- Shows token usage
- Displays cost savings vs Claude

### 3. List Available Skills

```bash
# List all available skills (built-in + orchestrated + imported)
sr list

# List only imported marketplace skills (detailed view)
sr imported
```

**Understanding the difference:**
- `sr list` shows **all skills** you can run with `sr run`:
  - Built-in skills (test, backend-architect)
  - Orchestrated skills (multi-phase workflows from `~/.skillrunner/skills/`)
  - Imported marketplace skills (from `~/.skillrunner/marketplace/`)
- `sr imported` shows **only imported marketplace skills** with detailed metadata (source, import date, etc.)

## Testing Phase 7 Context Management

### Test 1: Small Input (No Optimization)

```bash
sr run hello-orchestration "a developer"
```

Should NOT show "Context optimized" message.

### Test 2: Large Input (Triggers Optimization)

```bash
# Create a large test file (run from project root)
cat internal/context/manager.go internal/context/chunker.go > /tmp/large-input.txt

# Run code review with large input
sr run code-review "$(cat /tmp/large-input.txt)"
```

**Expected behavior:**
- Shows "Context optimized (exceeded X tokens)" message
- Skill still executes successfully
- Output quality maintained

### Test 3: Verify Token Counting

```bash
# Run tests to verify token counter accuracy (run from project root)
go test ./internal/context/... -v -run TestTokenCounter
```

## Available Commands

### Running Skills

All skills can be run with `sr run`, regardless of their type:

```bash
# Run any skill (orchestrated, built-in, or imported)
sr run <skill-name> <request>

# Examples
sr run hello-orchestration "your request"     # Orchestrated skill
sr run bug-fix "Login returns 500 error"      # Orchestrated skill
sr run code-review "func Divide(a, b int) { return a/b }"  # Orchestrated skill
sr run test "hello world"                     # Built-in skill
sr run backend-architect "add auth"           # Built-in skill
```

**Note:** Behavior differs by skill type:
- **Orchestrated skills**: Execute phases and show:
  - 🚀 Starting execution message
  - Progress for each phase/batch
  - 📋 Final Output section
  - 📊 Execution Summary with metrics
  - ✅ Execution Complete! confirmation
- **Built-in/Legacy skills**: Generate JSON envelope (no execution):
  - Outputs structured JSON with workflow steps
  - Use `--output <file>` to save envelope
  - Envelope can be passed to Claude Code CLI or other tools for execution

### Marketplace Skills

```bash
# Ask single question with intelligent model routing
sr ask <skill-id> <question>

# Examples
sr ask avl-expert "lighting design for theater"
sr ask business-analyst "revenue growth analysis"
```

### Skill Management

```bash
# List all available skills (built-in + orchestrated + imported)
sr list

# Import skill from source (web, local path, git)
sr import <source>
sr import /path/to/skill
sr import https://example.com/skill.yaml

# List only imported marketplace skills (with detailed metadata)
sr imported

# Update imported skill from its original source
sr update <skill-id>

# Remove imported skill
sr remove <skill-id>
```

**Skill Types Explained:**
- **Built-in skills** (Legacy): Core skills included with Skillrunner (e.g., `test`, `backend-architect`)
  - Generate JSON envelopes with structured workflow steps (plan/edit/run)
  - Do NOT execute LLM calls - they only create the workflow structure
  - Use `--output` flag to save envelope to file
  - `--stream` flag does NOT work (no execution happens)
  - Envelopes can be consumed by tools like Claude Code CLI or Continue IDE
- **Orchestrated skills**: Multi-phase workflows stored in `~/.skillrunner/skills/` (e.g., `hello-orchestration`, `bug-fix`)
  - Execute LLM calls and run phases with dependencies
  - Support `--stream` for real-time output
  - Return final results, not envelopes
- **Imported marketplace skills**: Skills imported from external sources, stored in `~/.skillrunner/marketplace/`

All skill types can be executed with `sr run <skill-name> <request>`.

### Using Legacy Skill Envelopes

Built-in/legacy skills generate JSON envelopes that can be executed by external tools:

```bash
# Generate envelope and save to file
sr run backend-architect "add user authentication" --output plan.json

# The envelope contains structured workflow steps:
# - plan: Analysis and planning step
# - edit: Implementation step
# - run: Testing/verification step
# Each step includes: prompts, context, model info, metadata
```

**Using with Claude Code CLI:**

The envelope JSON can be processed with Claude Code CLI in non-interactive mode:

```bash
# 1. Generate envelope
sr run backend-architect "build API endpoint" --output envelope.json

# 2. Process envelope with Claude Code CLI (non-interactive mode)
# Option A: Pipe envelope content
cat envelope.json | claude -p --output-format json

# Option B: Process envelope steps individually
# Extract prompts from envelope and execute
jq -r '.steps[0].prompt' envelope.json | claude -p

# Option C: Convert envelope to Claude Code format (if needed)
# You may need to transform the envelope structure to match Claude Code's expected format
```

**Claude Code CLI Flags:**
- `-p, --print`: Run in non-interactive mode and print results
- `--output-format <format>`: Specify output format (`text`, `json`, or `stream-json`)

**Envelope Structure:**
```json
{
  "version": "1.0",
  "skill": "backend-architect",
  "request": "your request",
  "steps": [
    {
      "intent": "plan",
      "model": "model-id",
      "prompt": "Analysis prompt...",
      "context": [
        {
          "type": "folder",
          "source": "/context"
        },
        {
          "type": "pattern",
          "source": "imports"
        }
      ],
      "file_ops": [],
      "metadata": {
        "phase": "planning",
        "model_provider": "ollama",
        "model_provider_tier": "fast"
      }
    },
    {
      "intent": "edit",
      "model": "model-id",
      "prompt": "Implementation prompt...",
      "context": [...],
      "file_ops": [...],
      "metadata": {...}
    },
    {
      "intent": "run",
      "model": "model-id",
      "prompt": "Testing prompt...",
      "context": [...],
      "file_ops": [...],
      "metadata": {...}
    }
  ],
  "metadata": {
    "created_at": "2025-11-28T21:18:29Z",
    "workspace": "."
  }
}
```

**Step Intent Types:**
- `plan`: Analysis and planning step
- `edit`: Implementation/code generation step
- `run`: Testing/verification step

**Processing Envelope Steps:**

You can iterate through envelope steps programmatically:

```bash
# Extract all prompts from envelope
jq -r '.steps[] | "\(.intent): \(.prompt)"' envelope.json

# Execute each step sequentially
for step in $(jq -c '.steps[]' envelope.json); do
  intent=$(echo $step | jq -r '.intent')
  prompt=$(echo $step | jq -r '.prompt')
  echo "Executing $intent step..."
  echo "$prompt" | claude -p --output-format json
done
```

**Important Notes:**
- Legacy skills do NOT execute LLM calls - they only generate the envelope structure
- The `--stream` flag does NOT work with legacy skills (no execution happens)
- To get actual execution with streaming, use orchestrated skills instead
- Envelopes are designed for tools that can execute structured workflows
- Claude Code CLI may require format conversion - check their documentation for exact envelope format requirements
- Each step in the envelope can be executed individually or as a complete workflow
- See [Envelope Integration Guide](../guides/envelope-integration.md) for detailed integration examples

### Format Conversion

```bash
# Convert between formats (auto-detects)
sr convert <input-file> <output-file>

# Claude → Skillrunner
sr convert marketplace-skill.yaml orchestrated-skill.yaml

# Skillrunner → Claude
sr convert orchestrated-skill.yaml marketplace-skill.yaml
```

### Configuration

```bash
# Show current config
sr config show

# Set config values
sr config set <key> <value>
sr config set default_model ollama/qwen2.5:14b
```

### System Status

```bash
# Check system status
sr status
```

### Marketplace Discovery

```bash
# Search for skills in HuggingFace marketplace
sr search "audio engineering"

# Inspect a skill before importing
sr inspect avl-expert
```

### Model Management

```bash
# List all available models
sr models list

# List models from specific provider
sr models list --provider=ollama

# Check if a model is available
sr models check ollama/qwen2.5:14b

# Validate models for a skill
sr models validate hello-orchestration

# Refresh provider caches
sr models refresh
```

### Metrics & Monitoring

```bash
# View routing metrics and costs
sr metrics

# View metrics for last 7 days
sr metrics --since 7d

# Export metrics to CSV
sr metrics --export costs.csv
```

### Cache Management

```bash
# Clear all cached execution results
sr cache clear

# Show cache statistics
sr cache stats
```

### Initialization

```bash
# Initialize Skillrunner configuration
sr init

# This creates ~/.skillrunner/ directory structure
```

## Example Workflows

### 1. Systematic Bug Fix

```bash
sr run bug-fix "Authentication fails when password contains @ symbol"
```

**Workflow (7 phases):**
1. Reproduce bug from report
2. Analyze root cause
3. Plan fix (parallel)
4. Identify tests (parallel)
5. Implement fix
6. Verify fix
7. Create summary

### 2. Full Feature Implementation

```bash
sr run feature-implementation "Add dark mode toggle to settings"
```

**Workflow (9 phases):**
1. Extract requirements
2. Design architecture
3. Plan implementation (parallel)
4. Design tests (parallel)
5. Generate code
6. Generate tests
7. Review implementation
8. Generate documentation
9. Create deployment checklist

### 3. Code Review

```bash
sr run code-review "$(cat my-code.go)"
```

**Workflow (5 phases):**
1. Extract requirements
2. Check style (parallel)
3. Check logic (parallel)
4. Check performance (parallel)
5. Generate report

## Performance Metrics

### hello-orchestration Benchmarks

```
Execution Time:
  Sequential: ~9.25s
  Parallel:   ~8.25s
  Speedup:    12%

Token Usage:
  Input:  361 tokens
  Output: 176 tokens

Cost:
  Ollama:  $0.00 (free)
  Claude:  $0.0037
  Savings: 100%
```

## Troubleshooting

### "Model not found"

```bash
# Check available models
ollama list

# Pull missing model
ollama pull qwen2.5:14b
```

### "Context deadline exceeded"

Model is downloading in background. Wait for download to complete:

```bash
# Check download status
ollama list
```

### "ANTHROPIC_API_KEY not set"

Only needed for cloud fallback. Set if you want to use Anthropic models:

```bash
export ANTHROPIC_API_KEY="your-api-key"
```

### Tests Failing

```bash
# Run all tests (run from project root)
go test ./...

# Run specific package tests
go test ./internal/context/... -v
```

## Next Steps

1. **Explore orchestrated skills** in `~/.skillrunner/skills/`
2. **Create custom skills** using YAML format
3. **Import marketplace skills** for specialized tasks
4. **Monitor cost savings** compared to cloud-only execution

## Understanding Skill Types

Skillrunner supports three types of skills:

1. **Built-in Skills** (`sr list` shows these)
   - Core skills included with Skillrunner
   - Examples: `test`, `backend-architect`
   - Located in the Skillrunner binary

2. **Orchestrated Skills** (`sr list` shows these)
   - Multi-phase workflows with dependencies
   - Stored in `~/.skillrunner/skills/<skill-name>/skill.yaml`
   - Examples: `hello-orchestration`, `bug-fix`, `code-review`
   - Can be created manually or converted from marketplace format

3. **Imported Marketplace Skills** (`sr list` and `sr imported` show these)
   - Skills imported from external sources
   - Stored in `~/.skillrunner/marketplace/<skill-id>/`
   - Use `sr ask <skill-id> <question>` for single questions
   - Use `sr run <skill-id> <request>` to run as workflow
   - `sr imported` shows detailed metadata (source, import date, etc.)

**Key Commands:**
- `sr list` - Shows ALL skills you can run (built-in + orchestrated + imported)
- `sr imported` - Shows ONLY imported marketplace skills with detailed info
- `sr run <skill>` - Runs any skill type
- `sr ask <skill-id>` - Asks a question using an imported marketplace skill

## Documentation

All documentation paths are relative to the project root:

- **Architecture**: `ARCHITECTURE.md` (in project root)
- **Configuration**: `docs/getting-started/CONFIGURATION.md`
- **User Guides**: `docs/guides/`

## Support

For issues or questions:
- Check `docs/` directory for detailed documentation
- Review test files for usage examples
- Examine skills in `~/.skillrunner/skills/` for YAML format
