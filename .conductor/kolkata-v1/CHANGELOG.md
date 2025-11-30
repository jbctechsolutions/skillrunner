# Changelog

All notable changes to Skillrunner will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-18

### Added

#### Multi-Provider Routing & Cost Simulation
- **Profile-based routing** with cheap/balanced/premium profiles
- **Multi-provider support** for Anthropic, OpenAI, and Google Gemini
- **Cost simulation** with counterfactual cost calculations (premium-only vs cheap-only)
- **Run reports** with JSON export for cost analysis
- **CLI flags**: `--profile`, `--show-savings`, `--report-json`
- **Provider interface** for unified LLM provider abstraction
- **Cost tracking** per phase with automatic cost summary generation

#### Core Features
- **Multi-phase orchestration engine** with DAG-based execution
- **Parallel phase execution** for improved performance
- **Intelligent model routing** with health checking and fallback
- **Context management** with chunking and summarization
- **Result caching** with TTL-based expiration
- **Streaming responses** for real-time output (Ollama)

#### Context Management
- Simple chunking with overlap
- Hierarchical chunking (preserves markdown structure)
- **Semantic chunking** using keyword-based similarity
- **Smart chunk selection** with scoring and merging
- Automatic context optimization when limits are exceeded

#### Model Routing
- Health checking for Ollama and cloud models
- Automatic fallback chain (preferred → fallback → router selection)
- Cost-optimized routing (prefers local models)
- Per-phase model selection
- **Profile-based routing** (cheap/balanced/premium) for cost optimization
- **Multi-provider support** (Anthropic, OpenAI, Google Gemini)

#### Metrics & Analytics
- **Metrics storage** with file-based persistence
- **Metrics dashboard** with model usage statistics
- **CSV/JSON export** for metrics data
- Performance metrics (P95, P99 latency)
- Cost tracking and savings calculation

#### Skill Management
- Skill import from web, local paths, and git repositories
- Skill update and removal
- Format conversion (Claude Code ↔ Skillrunner)
- Skill registry with metadata tracking

#### Error Handling
- Structured error types with error codes
- User-friendly error messages
- Retryable error detection
- Error context and recovery suggestions

#### CLI Commands
- `sr run <skill> <request>` - Execute orchestrated skills
- `sr ask <skill-id> <question>` - Ask marketplace skills
- `sr list` - List available skills
- `sr status` - Show system status
- `sr import <source>` - Import skills
- `sr update <skill-id>` - Update imported skills
- `sr imported` - List imported skills
- `sr remove <skill-id>` - Remove imported skills
- `sr convert <input> <output>` - Convert skill formats
- `sr config set/show` - Configuration management
- `sr metrics` - View metrics and costs
- `sr cache clear/stats` - Cache management
- `--stream` flag for streaming responses
- `--profile` flag for routing profile selection (cheap|balanced|premium)
- `--show-savings` flag to display cost comparisons
- `--report-json` flag to export cost reports

#### Condition Parser
- Support for template variables (`{{phase1.status}}`)
- Boolean expressions (`&&`, `||`, `!`)
- Comparisons (`==`, `!=`, `>`, `<`, `>=`, `<=`)
- Evaluation against phase results and user context

### Changed

- Improved error messages with actionable suggestions
- Enhanced metrics display with model and skill statistics
- Better chunk selection algorithm

### Fixed

- Fixed broken test files (init_test.go, docker_test.go, route_test.go, metrics_test.go)
- Fixed compilation errors (unused variables, incorrect function signatures)
- Fixed division by zero in metrics display

### Technical Details

- Language: Go 1.21+
- CLI Framework: Cobra
- Configuration: YAML
- Storage: File-based JSON (metrics, cache, registry)
- Models Supported: Ollama (local), Anthropic Claude, OpenAI, Google Gemini

## [Unreleased]

### Added
- Initial release with core orchestration engine
- Basic model routing
- Context management (Phase 7)
- Skill import system (Phase 5)
- Marketplace integration (Phase 4)
