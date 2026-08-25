# SkillRunner Release Roadmap v3.0: Workflow Intelligence

**Document Version:** 3.0
**Created:** December 4, 2025
**Supersedes:** v2.0 (Cost Intelligence Pivot)

---

## What Changed

**v2.0 Positioning:** "Cost tracking platform" - competed with Langfuse/Helicone on metrics
**v3.0 Positioning:** "Opinionated workflow runner that learns" - competes on developer experience

**Key Insight:** Cost tracking per phase is NOT unique (Langfuse can do it with tags). What's unique is:
1. Pre-built coding workflows (no setup)
2. Automatic phase tagging (no manual work)
3. Learning from outcomes (auto-optimization)
4. Zero-config value (works immediately)

---

## New Positioning

**FROM:** "The only AI tool that shows you exactly what you're spending"

**TO:** "AI coding workflows that optimize themselves"

**Tagline:** "Run. Learn. Save."

**Comparison:**
| Tool | What You Do |
|------|-------------|
| Langfuse | Define workflows, tag phases, build dashboards, analyze, decide |
| SkillRunner | Run `sr code-review .` - everything else is automatic |

---

## Revised Timeline

```
Dec 2025     Jan 2026     Feb 2026     Mar 2026     Apr 2026     May 2026
    |            |            |            |            |            |
 v1.0.0       v1.1.0       v1.2.0       v1.3.0       v1.4.0     DECISION
 Workflows    Providers    Learning     Insights     Teams       POINT
 + Metrics    + More WFs   From Use     Dashboard    + API
```

---

## v1.0.0 (December 2025) - WORKFLOW-FIRST LAUNCH

**Theme:** "Coding Workflows That Just Work"

### Core Deliverables

| Deliverable | Why It Matters |
|-------------|----------------|
| 3 pre-built workflows | Zero setup - immediate value |
| Per-phase metrics (auto-tagged) | See cost/time/retries per phase automatically |
| Multi-provider support | Ollama, Anthropic, OpenAI |
| Cost + time tracking | Every run shows what it cost and how long |

### Pre-Built Workflows (v1.0)

```yaml
# 1. code-review.yaml
phases:
  - id: scan
    task: "Identify files to review"
    default_model: cheap
  - id: analyze
    task: "Deep analysis of code quality"
    default_model: balanced
  - id: report
    task: "Generate review summary"
    default_model: cheap

# 2. test-gen.yaml
phases:
  - id: understand
    task: "Understand code structure"
    default_model: cheap
  - id: generate
    task: "Generate test cases"
    default_model: premium
  - id: validate
    task: "Validate tests compile"
    default_model: cheap

# 3. doc-gen.yaml
phases:
  - id: extract
    task: "Extract code signatures"
    default_model: cheap
  - id: document
    task: "Generate documentation"
    default_model: balanced
  - id: format
    task: "Format as markdown"
    default_model: cheap
```

### Automatic Metrics (No Tagging Required)

```bash
$ sr run code-review .

Running code-review workflow...

Phase Results:
┌─────────────────────────────────────────────────────────────────┐
│ Phase      │ Model        │ Time   │ Cost   │ Retries │ Status │
├─────────────────────────────────────────────────────────────────┤
│ scan       │ ollama/qwen  │ 2.3s   │ $0.00  │ 0       │ ✓      │
│ analyze    │ claude-sonnet│ 8.1s   │ $0.12  │ 0       │ ✓      │
│ report     │ claude-haiku │ 1.4s   │ $0.01  │ 1       │ ✓      │
├─────────────────────────────────────────────────────────────────┤
│ TOTAL      │              │ 11.8s  │ $0.13  │ 1       │        │
│ vs all-premium             │        │ $0.45  │         │ -71%   │
└─────────────────────────────────────────────────────────────────┘
```

### Launch Messaging

**Reddit:** "I built a CLI that runs AI coding tasks in phases and shows you exactly what each phase costs"

**HN:** "Show HN: SkillRunner - AI coding workflows with per-phase cost tracking"

### Success Metrics

| Metric | Target | Signal |
|--------|--------|--------|
| GitHub Stars | 300+ | Interest exists |
| Users running workflows | 50+ | Product works |
| "Workflow" mentions in feedback | >30% | Positioning resonates |

---

## v1.1.0 (January 2026) - More Providers + Workflows

**Theme:** "More Ways to Save"

### Features

| Feature | Priority | Notes |
|---------|----------|-------|
| OpenAI provider | HIGH | GPT-4o, GPT-4o-mini |
| Groq provider | HIGH | Fastest, cheapest cloud |
| 3 more workflows | HIGH | refactor, explain, fix-bug |
| Workflow history | MEDIUM | `sr history` - see past runs |
| Export metrics | MEDIUM | JSON/CSV for analysis |

### New Workflows

```yaml
# 4. refactor.yaml - Improve code quality
# 5. explain.yaml - Explain code to junior devs
# 6. fix-bug.yaml - Debug and fix issues
```

### Key Output

```bash
$ sr history --last 7d

Last 7 Days:
┌────────────────────────────────────────────────────────────────────┐
│ Workflow     │ Runs │ Avg Time │ Avg Cost │ Avg Retries │ Success │
├────────────────────────────────────────────────────────────────────┤
│ code-review  │ 23   │ 12.4s    │ $0.14    │ 0.3         │ 96%     │
│ test-gen     │ 15   │ 18.2s    │ $0.22    │ 1.2         │ 87%     │
│ doc-gen      │ 8    │ 6.1s     │ $0.08    │ 0.1         │ 100%    │
├────────────────────────────────────────────────────────────────────┤
│ TOTAL        │ 46   │          │ $6.42    │             │ 93%     │
│ Saved vs all-premium          │ $18.90   │             │         │
└────────────────────────────────────────────────────────────────────┘
```

---

## v1.2.0 (February 2026) - LEARNING FROM USE [CRITICAL]

**Theme:** "Gets Smarter Over Time"

**This is the moat.** Nobody else does this.

### Core Feature: Outcome-Based Learning

Track what ACTUALLY worked, not just what models claim to work:

```bash
$ sr stats --by-phase

Phase Performance (from your usage):
┌──────────────────────────────────────────────────────────────────────┐
│ Workflow    │ Phase    │ Model         │ Success │ Retries │ Cost   │
├──────────────────────────────────────────────────────────────────────┤
│ code-review │ scan     │ ollama/qwen   │ 98%     │ 0.1     │ $0.00  │
│ code-review │ analyze  │ claude-sonnet │ 94%     │ 0.2     │ $0.12  │
│ code-review │ report   │ claude-haiku  │ 89%     │ 0.8     │ $0.01  │ ← Issue
├──────────────────────────────────────────────────────────────────────┤
│ test-gen    │ generate │ claude-haiku  │ 72%     │ 2.1     │ $0.02  │ ← Problem
│ test-gen    │ generate │ claude-sonnet │ 95%     │ 0.3     │ $0.12  │ ← Better
└──────────────────────────────────────────────────────────────────────┘

Recommendations:
  • code-review.report: Upgrade haiku→sonnet (+$0.11, -0.6 retries)
  • test-gen.generate: Already using optimal model (sonnet)

Estimated monthly impact: +$4.20 cost, -18 retries, +12% success rate
```

### How It Works

1. **Track outcomes** - Did the phase succeed? How many retries?
2. **Aggregate by phase + model** - Which model works best for each phase?
3. **Recommend optimizations** - "Switch phase X to model Y"
4. **Auto-apply (opt-in)** - `sr config set auto-optimize true`

### Features

| Feature | Priority | Notes |
|---------|----------|-------|
| Outcome tracking | CRITICAL | Success/fail per phase |
| Retry counting | CRITICAL | How often does each phase retry? |
| Model comparison | HIGH | Same phase, different models |
| `sr optimize` | HIGH | Show recommendations |
| Auto-optimization | MEDIUM | Opt-in automatic model switching |

### Why This Is Unique

| Tool | What They Track | What They Optimize |
|------|-----------------|-------------------|
| Langfuse | Metrics you tag | Nothing (you analyze) |
| Martian/Not Diamond | Predicted quality | Single requests |
| **SkillRunner** | Actual outcomes per phase | Entire workflows |

---

## v1.3.0 (March 2026) - Insights Dashboard

**Theme:** "See Your Patterns"

### Features

| Feature | Priority | Notes |
|---------|----------|-------|
| Local web dashboard | HIGH | Single binary, no infra |
| Workflow comparison | HIGH | Which workflows save most? |
| Model leaderboard | MEDIUM | Best models for your usage |
| Export/share stats | MEDIUM | Anonymized benchmarks |

### Dashboard MVP

```
┌─────────────────────────────────────────────────────────────────────┐
│  SkillRunner Insights                              [Last 30 days]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Workflows Run: 312        Total Cost: $41.23                      │
│  Success Rate: 94%         Saved vs Premium: $127.45 (76%)         │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  Model Performance (Your Data)                                     │
│  ───────────────────────────────                                   │
│  ollama/qwen2.5    │ ████████████████████ 97% │ $0.00  │ scan     │
│  claude-haiku      │ ██████████████░░░░░░ 84% │ $0.01  │ report   │
│  claude-sonnet     │ ███████████████████░ 95% │ $0.12  │ analyze  │
│  gpt-4o-mini       │ █████████████████░░░ 91% │ $0.02  │ various  │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  Optimization Opportunities                                        │
│  ───────────────────────────                                       │
│  • test-gen.generate: Switch haiku→sonnet (saves 1.8 retries/run) │
│  • doc-gen.format: Switch sonnet→haiku (same quality, -$0.08)     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## v1.4.0 (April 2026) - Teams + API

**Theme:** "Share What Works"

### Features

| Feature | Priority | Notes |
|---------|----------|-------|
| Team sync | HIGH | Share optimized workflows |
| REST API | HIGH | Integrate with CI/CD |
| Aggregated benchmarks | MEDIUM | "Best models for code-review" |
| Budget alerts | MEDIUM | Team spending limits |

### Pricing

| Tier | Price | Features |
|------|-------|----------|
| Free | $0 | CLI, local learning, 3 workflows |
| Pro | $19/mo | All workflows, cloud sync, history |
| Team | $49/seat/mo | Dashboard, API, shared learnings |

---

## May 2026: DECISION POINT

### Metrics to Evaluate

| Metric | Stop | Continue | Scale |
|--------|------|----------|-------|
| GitHub Stars | <500 | 500-2K | >2K |
| Weekly Active | <100 | 100-500 | >500 |
| Paying Users | 0 | 10-50 | >50 |
| Success Stories | 0 | 3-10 | >10 |

---

## Key Differences from v2.0 Roadmap

| v2.0 (Cost Intelligence) | v3.0 (Workflow Intelligence) |
|--------------------------|------------------------------|
| "See what AI costs" | "Workflows that optimize themselves" |
| Compete with Langfuse on metrics | Compete on DX + automation |
| SDK for other tools | Opinionated product |
| Dashboard first | CLI first, dashboard later |
| Cost tracking as feature | Learning as feature |

### Deprioritized from v2.0

| Feature | v2.0 Version | v3.0 Status | Reason |
|---------|--------------|-------------|--------|
| Cost SDK library | v1.3.0 | DROPPED | Not differentiating |
| Factory/Claude Code integration | v1.3.0 | DROPPED | Focus on own product |
| Prometheus export | v1.3.0 | v1.4.0 | Later, if enterprise wants |
| Public benchmarks page | v1.1.0 | v1.3.0 | After we have data |

### Accelerated from v2.0

| Feature | v2.0 Version | v3.0 Status | Reason |
|---------|--------------|-------------|--------|
| Pre-built workflows | Implicit | v1.0.0 | Core value prop |
| Outcome tracking | Not planned | v1.2.0 | The actual moat |
| Auto-optimization | v1.2.0 | v1.2.0 | Confirmed priority |

---

## 2-Week Sprint to v1.0

### Week 1: Core Workflows

| Day | Deliverable |
|-----|-------------|
| Mon | code-review.yaml working end-to-end |
| Tue | test-gen.yaml working |
| Wed | doc-gen.yaml working |
| Thu | Per-phase metrics display |
| Fri | Multi-provider routing (Ollama + Anthropic) |

### Week 2: Polish + Launch

| Day | Deliverable |
|-----|-------------|
| Mon | OpenAI provider |
| Tue | README rewrite, examples |
| Wed | `sr history` basic implementation |
| Thu | Testing, bug fixes |
| Fri | Launch prep |
| Sat | Reddit soft launch |

---

## Success Criteria (Revised)

### 30-Day Check

| Metric | Target | Signal |
|--------|--------|--------|
| Stars | 300+ | Interest |
| Workflow runs | 500+ | Usage |
| "Easy to use" feedback | Any | DX resonates |

### 90-Day Check

| Metric | Target | Signal |
|--------|--------|--------|
| Stars | 1,000+ | Growth |
| Users with 10+ runs | 100+ | Retention |
| Optimization recommendations used | 50+ | Learning works |

### 180-Day Check

| Metric | Target | Signal |
|--------|--------|--------|
| Paying users | 50+ | Revenue |
| MRR | $2K+ | Path to $10K |
| "Saved $X" testimonials | 10+ | Value proven |

---

*Document prepared December 4, 2025*
*Strategic refinement: From cost platform to opinionated workflow product*
