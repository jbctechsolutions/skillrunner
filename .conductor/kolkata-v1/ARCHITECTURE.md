# Skillrunner Architecture

## System Overview

Skillrunner is a CLI tool that orchestrates AI development workflows with intelligent model routing. It combines skill-based task execution with smart routing between local and cloud AI models.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│                    (cmd/skillrunner/main.go)                        │
│  Commands: run, route, list, status, worktree                   │
└────────────────────────────┬────────────────────────────────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
        ┌───────▼────────┐      ┌────────▼──────────┐
        │  Engine        │      │  Router           │
        │  (existing)    │      │  (new)            │
        └───────┬────────┘      └────────┬──────────┘
                │                         │
                │                         │
        ┌───────▼────────┐      ┌────────▼──────────┐
        │  Envelope      │      │  Providers        │
        │  Builder       │      │  - Ollama         │
        │  (existing)    │      │  - Anthropic      │
        └────────────────┘      │  - OpenAI         │
                                └───────────────────┘
                                         │
                ┌────────────────────────┼────────────────────────┐
                │                        │                        │
        ┌───────▼────────┐      ┌────────▼──────────┐  ┌────────▼──────────┐
        │  Skills        │      │  Context          │  │  Worktree         │
        │  Loader        │      │  Manager          │  │  Manager          │
        │  (YAML)        │      │  (Chunking)       │  │  (Git)            │
        └────────────────┘      └───────────────────┘  └───────────────────┘
```

## Component Responsibilities

### CLI Layer (`cmd/skillrunner/`)
- Command parsing and validation
- User interaction
- Output formatting
- Error presentation

### Engine (`internal/engine/`)
- Skill execution orchestration
- Workflow step generation
- Envelope creation
- **Existing functionality**

### Router (`internal/router/`)
- Model selection and routing
- Generation → Review → Fallback logic
- Metrics collection
- Cost tracking
- **New core component**

### Providers (`internal/router/providers/`)
- Abstract model provider interface
- Ollama implementation (local)
- Anthropic implementation (cloud)
- OpenAI implementation (cloud)
- Cost estimation

### Skills (`internal/skills/`)
- YAML skillrunner loading
- Skill validation
- Skill caching
- **New component**

### Context Manager (`internal/router/context/`)
- Context chunking strategies
- Hierarchical summarization
- Token estimation
- **New component**

### Worktree Manager (`internal/worktree/`)
- Git worktree creation
- Naming convention enforcement
- Collision detection
- Cleanup utilities
- **New component**

## Data Flow

### Routing Flow

```
User Input
    │
    ├─> Load Skill YAML
    │       │
    │       └─> Parse Routing Config
    │
    ├─> Load Context (if provided)
    │       │
    │       ├─> Check Size
    │       │
    │       └─> Chunk if Needed
    │
    ├─> Route Generation
    │       │
    │       ├─> Try Local Model (Ollama)
    │       │       │
    │       │       ├─> Success → Continue
    │       │       └─> Failure → Fallback to Cloud
    │       │
    │       └─> Generate Response
    │
    ├─> Route Review (if enabled)
    │       │
    │       └─> Cloud Model (Anthropic/OpenAI)
    │
    └─> Return Result + Metrics
```

### Worktree Flow

```
Agent Task Request
    │
    ├─> Generate Worktree Name
    │       │
    │       └─> Check Collisions
    │
    ├─> Create Git Worktree
    │
    ├─> Execute Router in Worktree
    │
    └─> Return Diff
```

## Package Dependencies

```
cmd/skillrunner
    ├─> internal/engine
    ├─> internal/router
    ├─> internal/skills
    └─> internal/worktree

internal/router
    ├─> internal/router/providers
    ├─> internal/router/context
    └─> internal/types

internal/engine
    ├─> internal/envelope
    └─> internal/types

internal/skills
    └─> internal/types

internal/worktree
    └─> (git operations, no internal deps)
```

## Key Design Patterns

### 1. Provider Pattern
Abstract model providers behind a common interface for easy testing and extension.

### 2. Strategy Pattern
Context chunking strategies (simple, hierarchical, semantic) are pluggable.

### 3. Builder Pattern
Envelope builder (existing) for constructing workflow envelopes.

### 4. Factory Pattern
Provider factory for creating providers based on model string.

## Configuration

### Skill YAML Structure

```yaml
skill:
  name: architecture
  version: 1.0.0
  description: Generate architecture documentation

  routing:
    generation_model: ollama/deepseek-coder-v2:16b
    review_model: anthropic/claude-3-5-sonnet-20241022
    fallback_model: anthropic/claude-3-5-sonnet-20241022
    max_context_tokens: 50000
    chunk_strategy: hierarchical_summarization

  context_strategy:
    type: hierarchical_summarization
    chunk_size: 10000
    overlap: 500
    summarization_model: ollama/deepseek-coder-v2:16b
```

### Environment Variables

```bash
# Required for cloud providers
ANTHROPIC_API_KEY=xxx
OPENAI_API_KEY=xxx

# Optional
OLLAMA_HOST=http://localhost:11434
SKILLS_DIR=./skills
```

## Error Handling Strategy

1. **Transient Errors**: Retry once, then fallback
2. **Fatal Errors**: Immediate fallback to next model
3. **Provider Errors**: Log and continue with next provider
4. **Validation Errors**: Fail fast with clear error messages

## Performance Considerations

1. **Parallel Chunking**: Process multiple chunks concurrently when possible
2. **Connection Pooling**: Reuse HTTP connections for API calls
3. **Caching**: Cache skillrunner configurations and context summaries
4. **Lazy Loading**: Load providers only when needed

## Security Considerations

1. **API Keys**: Never log or expose API keys
2. **Worktree Isolation**: Each worktree is isolated from main branch
3. **Input Validation**: Validate all user inputs and skillrunner configs
4. **Path Traversal**: Prevent path traversal in context loading

## Testing Strategy

- **Unit Tests**: Each package has comprehensive unit tests
- **Integration Tests**: Test router with mock providers
- **E2E Tests**: Test full flow with real Ollama (when available)
- **Mock Providers**: Mock HTTP responses for testing

## Future Extensions

1. **Streaming**: Add streaming response support
2. **Caching**: Persistent cache for generations
3. **Metrics DB**: Store metrics in database
4. **Plugin System**: Allow custom providers
5. **Web UI**: Dashboard for metrics and skillrunner management
