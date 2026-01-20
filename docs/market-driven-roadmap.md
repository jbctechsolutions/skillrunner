# Skillrunner Market-Driven Roadmap (v1.2 - v1.4)

**Created**: 2026-01-20
**Status**: Design Complete - Ready for Implementation
**Total Timeline**: 10-13 weeks (3 releases)

---

## Executive Summary

**Strategic Goal**: Build Skillrunner for personal use first to save money NOW on API costs. Monetization is 6-9 months out.

**Key Insight**: Skillrunner as cost-optimized automation layer alongside Claude Code:
- **Claude Code**: Interactive work with MCP servers (Linear, Notion, GitHub, Filesystem, Git, Slack, Supabase, Browser)
- **Skillrunner**: Batch/automated workflows with intelligent routing (Ollama for cheap, Claude for critical)

### Primary Workflows
1. **Linear → Plan → Build**: Pull specs from Linear, generate plans, build using specialized agents
2. **Data Sync**: Sync data between Notion ↔ Linear ↔ GitHub
3. **Deep Research**: Generate detailed implementation plans and research

### ROI by Phase

| Phase | Timeline | Investment | ROI |
|-------|----------|------------|-----|
| v1.2: Foundation | 3-4 weeks | 17 points | Enable automation workflows |
| v1.3: Make It Cheap | 3-4 weeks | 17 points | **$200-400/month savings** |
| v1.4: Make It Smart | 4-5 weeks | 15 points | Higher quality + safety |

---

## Phase 1: Foundation (v1.2 - 3-4 weeks, 17 points)

**Theme**: Get MCP working + basic cost controls
**Goal**: Enable Linear→Plan→Build workflow while preventing cost overruns

### P0: MCP Provider Integration (5 points, 3-4 days)

**What**: Pass MCP tools to Anthropic provider for function calling

**Critical Path**: This is the blocker for all MCP workflows

**Implementation**:
1. Fetch tool definitions from MCP registry
2. Convert to provider format (existing `tool_converter.go`)
3. Pass in `CompletionRequest.Tools`
4. Handle `tool_use` blocks in response
5. Execute via `MCPToolRegistryPort`
6. Feed results back to provider

**Files**:
- `internal/application/workflow/phase_executor.go` - Add tool fetching and execution loop
- `internal/application/workflow/streaming_phase_executor.go` - Same for streaming
- `internal/adapters/provider/anthropic/client.go` - Handle tool_use blocks
- `internal/adapters/provider/anthropic/streaming.go` - Stream tool calls

**Acceptance Criteria**:
- ✅ Executor fetches tools from MCP registry
- ✅ Tools passed in `CompletionRequest`
- ✅ Anthropic adapter handles tool calls
- ✅ Tool execution via `MCPToolRegistryPort`
- ✅ Results fed back for continuation
- ✅ Unit tests + integration test with real API
- ✅ Works with existing Linear, Filesystem, Git MCP servers

**Verification**:
```bash
# Test with Linear MCP server
sr run test-mcp-skill "List my Linear issues and count them"
# Should call linear__list_issues tool and return count
```

---

### P0: Skills Declare Tool Requirements (2 points, 1-2 days)

**What**: Extend skill YAML to declare tool dependencies

**Example**:
```yaml
id: linear-plan-generator
tools:
  - mcp__linear__get_issue
  - mcp__linear__list_issues
  - mcp__filesystem__write_file

phases:
  - id: generate
    allow_tools: true
    prompt_template: |
      Generate implementation plan for: {{.input}}
```

**Files**:
- `internal/domain/skill/skill.go` - Add `Tools []string` field
- `internal/domain/skill/phase.go` - Add `AllowTools bool` field
- `internal/domain/skill/skill_test.go` - Test new fields
- `internal/infrastructure/skills/loader.go` - Parse YAML fields
- `internal/infrastructure/skills/loader_test.go` - Test YAML parsing

**Acceptance Criteria**:
- ✅ `Skill` struct has `Tools []string` field
- ✅ `Phase` struct has `AllowTools bool` field
- ✅ YAML parser reads both fields
- ✅ Validation: tools match `mcp__server__tool` format
- ✅ Backward compatible (fields optional)
- ✅ Unit tests pass

**Verification**:
```bash
# Create test skill with tools field
echo '
id: test-skill
tools:
  - mcp__filesystem__read_file
phases:
  - id: test
    allow_tools: true
    prompt_template: "Test"
' > ~/.skillrunner/skills/test-skill.yaml

sr list  # Should load without errors
```

---

### P0: Tool Permission System (3 points, 2-3 days)

**What**: Permission prompts for tool execution (similar to file permissions)

**Example Prompt**:
```
🔧 Tool Permission Request
The skill wants to use 3 tool(s):
  1. linear__get_issue - Retrieve Linear issue details
  2. linear__list_issues - List Linear issues
  3. filesystem__write_file - Write files to disk

Allow tool execution? [Y/n/individual/show]
```

**Files**:
- `internal/infrastructure/context/tool_permission.go` (NEW) - Permission logic
- `internal/infrastructure/context/tool_permission_test.go` (NEW) - Tests
- `internal/application/workflow/executor.go` - Add permission check before tool usage
- `internal/presentation/cli/commands/run.go` - Add `-y` flag
- `internal/presentation/cli/commands/ask.go` - Add `-y` flag

**Pattern**: Copy from existing `permission.go` (file permissions)

**Acceptance Criteria**:
- ✅ Permission prompt before skill execution
- ✅ Four modes: Y, n, individual, show
- ✅ Auto-approve via `-y` flag
- ✅ Shows tool descriptions from MCP
- ✅ Denying aborts with clear error
- ✅ Unit tests with stdin mocking

**Verification**:
```bash
# Should prompt for permission
sr run linear-plan-generator "Generate plan for JBC-123"

# Should skip prompt
sr run linear-plan-generator "Generate plan for JBC-123" -y
```

---

### P1: MCP CLI Commands (3 points, 2-3 days)

**What**: Server management and tool discovery commands

**Commands**:
```bash
sr mcp list-servers
sr mcp start <name>
sr mcp stop <name>
sr mcp status <name>
sr mcp logs <name>
sr mcp list-tools
sr mcp describe-tool <name>
sr mcp call <name> --args='{...}'
```

**Files**:
- `internal/presentation/cli/commands/mcp.go` (NEW) - All MCP commands
- `internal/presentation/cli/commands/root.go` - Register mcp command

**Acceptance Criteria**:
- ✅ All 8 commands implemented
- ✅ Commands use existing `ServerManager` from container
- ✅ Proper error handling and user feedback
- ✅ Colors/formatting match existing CLI style
- ✅ Help text and examples
- ✅ Unit tests

**Verification**:
```bash
sr mcp list-servers  # Show configured servers
sr mcp start filesystem  # Start server
sr mcp list-tools  # Show available tools
sr mcp describe-tool mcp__filesystem__read_file  # Show tool schema
```

---

### P1: Budget Limits (2 points, 1-2 days)

**What**: Hard limits on spending per workflow

**Usage**:
```bash
sr config set budget.daily_limit 5.00
sr config set budget.per_workflow_limit 0.50

sr run complex-workflow "..." --budget=0.25
```

**Files**:
- `internal/domain/budget/` (NEW) - Budget domain model
- `internal/domain/budget/limits.go` - Limit checking logic
- `internal/domain/budget/repository.go` - Budget repository port
- `internal/infrastructure/storage/budget_repository.go` (NEW) - SQLite storage
- `internal/application/workflow/executor.go` - Check limits before/during execution
- `internal/infrastructure/config/config.go` - Add budget config fields

**Acceptance Criteria**:
- ✅ Global budget limits (daily, monthly)
- ✅ Per-workflow budget limits
- ✅ `--budget` flag for run/ask commands
- ✅ Abort workflow if limit exceeded
- ✅ Warning at 80% of limit
- ✅ Budget tracking persisted in SQLite

**Verification**:
```bash
sr config set budget.daily_limit 0.10
sr run expensive-task "..."  # Should abort if exceeds $0.10
```

---

### P2: Bundled MCP Servers (2 points, 1-2 days)

**What**: Default configurations for filesystem and git MCP servers

**Config**: `~/.skillrunner/mcp_servers.json`
```json
{
  "filesystem": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "${HOME}"]
  },
  "git": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-git"]
  }
}
```

**Files**:
- `internal/infrastructure/config/mcp_defaults.go` (NEW) - Default config templates
- `internal/presentation/cli/commands/init.go` - Create default config on init
- `configs/mcp_servers.example.json` (NEW) - Example config

**Acceptance Criteria**:
- ✅ Default config created by `sr init`
- ✅ Supports `${HOME}` and `${PWD}` expansion
- ✅ Validation on load
- ✅ Instructions if npx missing
- ✅ Unit tests

**Verification**:
```bash
rm -rf ~/.skillrunner
sr init  # Should create mcp_servers.json with defaults
cat ~/.skillrunner/mcp_servers.json  # Verify content
```

---

### Phase 1 Summary

**Total**: 17 points (~3-4 weeks)

**After v1.2, you can**:
- ✅ Pull Linear issues using MCP tools
- ✅ Generate plans that write to filesystem
- ✅ Sync data between Notion/Linear/GitHub
- ✅ Automated workflows with tool permissions
- ✅ Budget limits prevent runaway costs

**Success Criteria**:
- Skills can execute MCP tools with permissions
- `sr mcp` commands working end-to-end
- Budget limits enforced
- 2+ bundled MCP servers configured
- All tests passing (80%+ coverage)

---

## Phase 2: Make It Cheap (v1.3 - 3-4 weeks, 17 points)

**Theme**: Minimize API costs through intelligent routing and caching
**Goal**: Save $200-400/month through adaptive complexity, enhanced caching, context compression

### P0: Adaptive Complexity Analysis (5 points, 3-4 days)

**What**: Analyze task complexity before selecting model (Zeroshot-inspired)

**Example Flow**:
```yaml
# Low complexity
sr run code-review "Review this 50-line function"
→ complexity_score: 0.35 → llama3.2:3b

# High complexity
sr run code-review "Analyze distributed race condition"
→ complexity_score: 0.92 → claude-sonnet-4.5
```

**Complexity Signals**:
- Request length (tokens)
- Technical keywords ("distributed", "async", "security")
- Code volume
- Historical success rate for this skill+complexity

**Files**:
- `internal/domain/complexity/` (NEW) - Complexity analyzer domain
- `internal/domain/complexity/analyzer.go` - Scoring algorithm
- `internal/domain/complexity/signals.go` - Signal extraction
- `internal/infrastructure/complexity/analyzer.go` (NEW) - Implementation
- `internal/application/workflow/executor.go` - Integrate analyzer

**Acceptance Criteria**:
- ✅ Score: 0.0 (trivial) to 1.0 (expert-level)
- ✅ Maps to profiles (0-0.4=cheap, 0.4-0.7=balanced, 0.7+=premium)
- ✅ `--profile` flag overrides analysis
- ✅ Metrics track complexity scores vs. outcomes
- ✅ Unit tests with various request types

**Why P0**: Routes 60-70% of tasks to cheap models = biggest cost saver

---

### P0: Enhanced Response Caching (3 points, 2-3 days)

**What**: Semantic similarity caching - recognize "similar enough" requests

**Example**:
```bash
sr run doc-gen "Explain what auth.go does"  # Miss - full execution
sr run doc-gen "Describe the auth.go file"  # Hit - semantically similar
```

**Files**:
- `internal/adapters/cache/semantic_cache.go` (NEW) - Semantic cache adapter
- `internal/infrastructure/embeddings/` (NEW) - Embedding generation
- `internal/application/workflow/executor.go` - Check semantic cache first

**Acceptance Criteria**:
- ✅ Embedding-based similarity (cosine distance)
- ✅ Threshold: 0.90 similarity = cache hit
- ✅ Fallback to exact match if embeddings unavailable
- ✅ Config: `cache.semantic_similarity_enabled`
- ✅ Metrics: semantic hits vs. exact hits

**Why P0**: Reduces duplicate work when phrasing varies

---

### P0: Budget Alerts (2 points, 1-2 days)

**What**: Proactive alerts before hitting limits

**Example**:
```
⚠️  Budget Alert: Daily spending at 80% ($4.00 / $5.00)
💰 Cost Optimization: This workflow cost $0.45
    Estimated savings with 'cheap' profile: $0.38 (84%)
```

**Files**:
- `internal/domain/budget/alerts.go` (NEW) - Alert logic
- `internal/presentation/cli/output/budget.go` (NEW) - Alert formatting
- `internal/presentation/cli/commands/budget.go` (NEW) - Budget commands

**Acceptance Criteria**:
- ✅ Alert at 50%, 80%, 90%, 100% of budget
- ✅ Per-workflow cost estimates before execution
- ✅ Cost optimization suggestions
- ✅ `sr budget status` command
- ✅ Daily/weekly budget reports

---

### P1: Context Compression (3 points, 2-3 days)

**What**: Compress context to reduce token costs

**Strategies**:
- Remove redundant whitespace
- Deduplicate repeated content
- Summarize verbose logs
- Strip comments (when safe)

**Files**:
- `internal/domain/context/compressor.go` (NEW) - Compressor interface
- `internal/infrastructure/context/strategies/` (NEW) - Compression strategies
- `internal/application/workflow/executor.go` - Compress before provider call

**Acceptance Criteria**:
- ✅ Multiple compression strategies
- ✅ Config: `context.compression_enabled`
- ✅ Aggressive mode (>70%) for cheap profile
- ✅ Conservative mode (<30%) for premium
- ✅ Preserve semantic meaning

**Why P1**: 30-50% token reduction on verbose inputs

---

### P1: Ollama Model Optimization (2 points, 1-2 days)

**What**: Task-specific model recommendations

**Example**:
```yaml
code-review:
  cheap: qwen2.5:14b  # Better code understanding
  balanced: qwen2.5-coder:32b

doc-gen:
  cheap: llama3.2:3b  # Fast for docs
```

**Files**:
- `internal/infrastructure/config/model_hints.go` (NEW) - Model recommendations
- `internal/adapters/provider/ollama/selector.go` (NEW) - Model selection
- `configs/model_hints.example.yaml` (NEW) - Example hints

---

### P2: Cost Analytics (2 points, 1-2 days)

**What**: Detailed cost breakdowns and reports

**Commands**:
```bash
sr cost report --last-7-days
sr cost breakdown --by-skill
sr cost savings --potential
```

**Files**:
- `internal/presentation/cli/commands/cost.go` (NEW) - Cost commands
- `internal/domain/analytics/cost_analyzer.go` (NEW) - Analytics logic
- `internal/presentation/cli/output/cost_report.go` (NEW) - Report formatting

---

### Phase 2 Summary

**Total**: 17 points (~3-4 weeks)

**Estimated Savings**: $200-400/month
- Adaptive complexity: 60-70% → cheap models
- Enhanced caching: 20-30% hit rate improvement
- Context compression: 30-50% token reduction

---

## Phase 3: Make It Smart (v1.4 - 4-5 weeks, 15 points)

**Theme**: Learn from execution, operate safely, handle failures gracefully
**Goal**: Intelligent model selection, safe code generation, resilient workflows

### P0: Outcome Tracking & Learning (5 points, 3-4 days)

**What**: Track workflow outcomes, learn from repetitions

**Files**:
- `internal/domain/outcome/` (NEW) - Outcome domain model
- `internal/infrastructure/storage/outcome_repository.go` (NEW) - SQLite storage
- `internal/application/workflow/executor.go` - Record outcomes

**Acceptance Criteria**:
- ✅ SQLite-backed outcome history per skill+profile
- ✅ Success/failure tracking with optional quality scores
- ✅ Automatic routing profile suggestions
- ✅ `sr outcomes show code-review` command

---

### P0: Confidence Monitoring & Auto-Escalation (3 points, 2-3 days)

**What**: Detect low-confidence responses, retry with better models

**Files**:
- `internal/domain/confidence/` (NEW) - Confidence parser
- `internal/application/workflow/executor.go` - Confidence checks + retry
- `internal/infrastructure/confidence/parser.go` (NEW) - Extract confidence

**Acceptance Criteria**:
- ✅ Parse confidence markers from LLM responses
- ✅ Configurable thresholds per routing profile
- ✅ Automatic retry with next-tier model
- ✅ Metrics: escalation frequency and cost impact

---

### P1: Git Worktree Isolation (4 points, 3-4 days)

**What**: Code modification skills run in isolated git worktrees

**Usage**:
```bash
sr run refactor "Extract auth module" --isolate
# Creates: ~/.skillrunner/worktrees/refactor-2026-01-20-abc123/
```

**Files**:
- `internal/domain/isolation/` (NEW) - Isolation domain
- `internal/infrastructure/git/worktree.go` (NEW) - Git worktree management
- `internal/application/workflow/executor.go` - Isolation mode

**Acceptance Criteria**:
- ✅ `--isolate` flag
- ✅ Automatic worktree creation
- ✅ Post-execution: show diff, offer merge/discard
- ✅ Cleanup stale worktrees after 7 days

---

### P1: Session Continuity (3 points, 2 days)

**What**: Resume interrupted workflows

**Usage**:
```bash
sr run complex-pipeline "..."
# ^C (interrupted after Phase 2)
sr resume  # Picks up at Phase 3
```

**Files**:
- `internal/domain/session/` (NEW) - Session domain
- `internal/infrastructure/storage/session_repository.go` (NEW) - SQLite storage
- `internal/presentation/cli/commands/resume.go` (NEW) - Resume command

---

### P2: Post-Completion Review Phases (2 points, 1-2 days)

**What**: Auto-review code generation before presenting

**Files**:
- `internal/domain/skill/phase.go` - Add `post_completion_review` field
- `internal/application/workflow/executor.go` - Review logic

---

### P2: Universal Agent Format Export (2 points, 1-2 days)

**What**: Export workflow results for other AI tools

**Files**:
- `internal/domain/export/` (NEW) - Export domain
- `internal/infrastructure/export/formats/` (NEW) - Format implementations

---

### Phase 3 Summary

**Total**: 15 points (~4-5 weeks)

**Value**: Smart, safe, and resilient
- Learns from outcomes
- Escalates when uncertain
- Code changes isolated
- Handles interruptions

---

## Complete Roadmap Timeline

| Phase | Theme | Weeks | Points | Key Value |
|-------|-------|-------|--------|-----------|
| v1.2 | Foundation | 3-4 | 17 | MCP works + cost controls |
| v1.3 | Make It Cheap | 3-4 | 17 | $200-400/month savings |
| v1.4 | Make It Smart | 4-5 | 15 | Quality + safety + learning |
| **Total** | | **10-13** | **49** | **Complete platform** |

### Timeline Visualization

```
Jan 2026          Feb 2026          Mar 2026          Apr 2026
├─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│   v1.2          │      v1.3       │      v1.4       │
│ Foundation      │  Make It Cheap  │  Make It Smart  │
│                 │                 │                 │
│ Week 1-2: MCP   │ Week 5-6: Cost  │ Week 9-10:      │
│ Week 3-4: Tools │ Week 7-8: Cache │   Intelligence  │
│                 │                 │ Week 11-12:     │
│                 │                 │   Safety        │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┘
         ↓                 ↓                 ↓
    Automation        Cost Savings      Quality + Safety
```

---

## Next Steps

1. ✅ **Complete** - Market-driven roadmap designed
2. **Create `docs/market-driven-roadmap.md`** - Full roadmap documentation
3. **Create Linear Issues** - Import v1.2 (Foundation) issues
4. **Update `CHANGELOG.md`** - Add v1.2, v1.3, v1.4 sections
5. **Kick Off v1.2** - Start with MCP provider integration (P0, 5 points)

---

## Success Metrics

### v1.2 (Foundation)
- ✅ Skills execute MCP tools with permissions
- ✅ `sr mcp` commands work end-to-end
- ✅ Budget limits enforced
- ✅ 2+ example skills using MCP

### v1.3 (Make It Cheap)
- ✅ **$200-400/month cost savings**
- ✅ 60-70% tasks → cheap models
- ✅ 20-30% semantic cache hit improvement
- ✅ 30-50% token reduction

### v1.4 (Make It Smart)
- ✅ Outcome tracking improves model selection
- ✅ Confidence escalation catches 80%+ low-quality responses
- ✅ Zero code disasters (worktree isolation)
- ✅ 90%+ interrupted workflows resume successfully
