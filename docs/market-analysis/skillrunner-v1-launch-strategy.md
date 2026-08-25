# Skillrunner v1.0 Launch Strategy & Implementation Guide

**Generated:** December 3, 2025  
**Target Launch:** December 13, 2025 (HN) / December 7, 2025 (Soft Launch)  
**Repository:** https://github.com/jbctechsolutions/skillrunner

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Product Overview](#product-overview)
3. [Competitive Landscape](#competitive-landscape)
4. [Launch Timeline](#launch-timeline)
5. [Linear Issues Summary](#linear-issues-summary)
6. [Reddit Launch Strategy](#reddit-launch-strategy)
7. [Platform-Specific Post Drafts](#platform-specific-post-drafts)
8. [Pre-Launch Checklist](#pre-launch-checklist)
9. [Post-Launch Monitoring](#post-launch-monitoring)
10. [January Priorities](#january-priorities)

---

## Executive Summary

Skillrunner is a local-first AI workflow orchestration CLI tool built in Go. The core value proposition is **70-90% cost reduction** on AI API calls by intelligently routing tasks to local Ollama models first, with cloud fallback only when needed.

### Key Differentiators

| Feature | Skillrunner | Competitors |
|---------|-------------|-------------|
| Built-in Cost Tracking | ✅ Unique | ❌ None have this |
| Profile-Based Routing | ✅ Simple (cheap/balanced/premium) | ❌ Complex configs |
| Local-First Philosophy | ✅ Ollama primary | 🟡 Cloud primary |
| Single Go Binary | ✅ Zero infrastructure | ❌ Often Java/Docker |
| Skill Marketplace | ✅ Shareable workflows | ❌ None |

### Launch Strategy

- **Soft Launch (Dec 7):** Reddit communities (r/LocalLLaMA, r/ollama, r/selfhosted, r/SideProject)
- **Full Launch (Dec 13):** Hacker News "Show HN" with polished demo video

---

## Product Overview

### What Skillrunner Does

```
User Request → Profile Selection → Model Routing → Execution → Cost Tracking
                    ↓                    ↓              ↓            ↓
              cheap/balanced/      Local Ollama    Multi-phase    Shows savings
               premium             or Cloud API      YAML          vs cloud
```

### Installation

```bash
brew install jbctechsolutions/skillrunner/skillrunner
sr init
sr ask "explain this code" --input file.go
```

### Example Output

```
✅ Phase 1: Extract context    [ollama/qwen2.5:14b]   $0.00
✅ Phase 2: Generate review    [ollama/qwen2.5:14b]   $0.00
✅ Phase 3: Format output      [ollama/qwen2.5:14b]   $0.00

────────────────────────────────────────────────────────
Total Cost: $0.00
Cloud Equivalent: $0.08 (Claude Sonnet for all phases)
💰 Savings: $0.08 (100%)
────────────────────────────────────────────────────────
```

### Supported Providers

| Provider | Status | Use Case | Cost |
|----------|--------|----------|------|
| Ollama | ✅ Primary | Local, free | $0.00 |
| Anthropic | ✅ Supported | Claude models | $3-15/1M tokens |
| OpenAI | 🔄 Planned | GPT-4o | $2.50-10/1M tokens |
| Groq | 🔄 Planned | Fast inference | $0.05-0.79/1M tokens |

---

## Competitive Landscape

### Direct Competitors (Go-based CLI Tools)

| Tool | Stars | Key Differentiator | Gap vs Skillrunner |
|------|-------|-------------------|-------------------|
| **ADK-Go** | 5,133 | Google backing, MCP Toolbox | No cost tracking |
| **OpenCode** | ~30,000 | Interactive coding agent | Single-session only |
| **aichat (sigoden)** | 4,200 | Multi-provider chat | No workflow orchestration |

### Broader AI Orchestration Landscape

| Category | Tools | Skillrunner Advantage |
|----------|-------|----------------------|
| Enterprise Workflow | Conductor, Temporal | Single binary vs Java infrastructure |
| Agent Frameworks | CrewAI, AutoGen, LangChain | Go simplicity, cost tracking |
| Local LLM Tools | Ollama, LM Studio | Workflow orchestration layer |

### Competitive Gaps to Address

1. **MCP Protocol Support** (JBC-691) - ADK-Go, CrewAI, Dify all have MCP
2. **Loop Agents** (JBC-669) - ADK-Go has iteration support
3. **TUI Interface** (JBC-671) - OpenCode has interactive TUI

### Competitive Advantages to Emphasize

1. **Cost Tracking** - No competitor has built-in cost metrics
2. **Profile-Based Routing** - Simpler than config-heavy alternatives
3. **Skill Marketplace** - Shareable, version-controlled workflows
4. **Local-First Design** - Ollama as primary, not afterthought

---

## Launch Timeline

### Pre-Launch (Dec 3-6)

| Date | Task | Issue | Priority |
|------|------|-------|----------|
| Dec 5 | Homebrew tap verification | ✅ JBC-678 Done | - |
| Dec 5-6 | Conflux → Skillrunner audit | JBC-690 | Urgent |
| Dec 5-6 | Verify streaming output | JBC-692 | High |
| Dec 6 | OpenAI provider (optional) | JBC-680 | High |
| Dec 6 | Groq provider (optional) | JBC-681 | High |

### Soft Launch Weekend (Dec 7-8)

| Date | Platform | Issue | Audience |
|------|----------|-------|----------|
| Dec 7 | r/LocalLLaMA | JBC-696 | 573k - Core audience |
| Dec 7 | r/ollama | JBC-697 | 92k - Ollama users |
| Dec 7 | r/SideProject | JBC-700 | 453k - Indie hackers |
| Dec 7 | r/coolgithubprojects | JBC-701 | 60k - GitHub projects |
| Dec 8 | r/selfhosted | JBC-698 | 400k - Privacy-focused |

### Polish Week (Dec 9-12)

| Date | Task | Issue |
|------|------|-------|
| Dec 8 | Demo skills with cost output | JBC-682 |
| Dec 10 | r/golang post | JBC-699 |
| Dec 10 | Conflux audit complete | JBC-690 |
| Dec 11 | README rewrite | JBC-683 |
| Dec 11 | Demo video | JBC-684 |
| Dec 12 | Launch posts finalized | JBC-685 |

### Full Launch (Dec 13)

| Platform | Format | Key Message |
|----------|--------|-------------|
| Hacker News | Show HN | Cost savings + local-first |
| Twitter/X | Thread | 8-tweet narrative |
| LinkedIn | Professional post | Team workflow angle |
| Reddit | Cross-posts | Link to HN discussion |

---

## Linear Issues Summary

### Issues Created This Session

| Issue | Title | Priority | Due |
|-------|-------|----------|-----|
| JBC-690 | Pre-Launch Audit: Conflux → Skillrunner | Urgent | Dec 10 |
| JBC-691 | Implement MCP Protocol Support | High | Jan 31 |
| JBC-692 | Verify streaming output | High | Dec 8 |
| JBC-693 | Add `sr history` command | Low | Feb 28 |
| JBC-694 | Competitive Tracking: ADK-Go | Medium | Mar 15 |
| JBC-695 | Competitive Positioning: OpenCode | Low | Mar 31 |
| JBC-696 | Soft Launch: r/LocalLLaMA | Urgent | Dec 7 |
| JBC-697 | Soft Launch: r/ollama | High | Dec 7 |
| JBC-698 | Soft Launch: r/selfhosted | High | Dec 8 |
| JBC-699 | Launch: r/golang | Medium | Dec 10 |
| JBC-700 | Soft Launch: r/SideProject | High | Dec 7 |
| JBC-701 | Soft Launch: r/coolgithubprojects | Medium | Dec 7 |

### Issues Updated This Session

| Issue | Title | Change | New Priority |
|-------|-------|--------|--------------|
| JBC-678 | Homebrew tap | ✅ Marked Done | - |
| JBC-680 | OpenAI provider | Added launch context | High |
| JBC-681 | Groq provider | Added speed differentiator | High |
| JBC-682 | Demo skills | Added cost output requirement | High |
| JBC-683 | README rewrite | Due date set | High |
| JBC-684 | Demo video | Streaming requirement added | High |
| JBC-685 | Launch posts | Complete rewrite with drafts | High |
| JBC-650 | API key encryption | Elevated to CRITICAL | Urgent |
| JBC-655 | Circuit breakers | Elevated priority | High |
| JBC-657 | Structured logging | Added migration plan | Medium |
| JBC-659 | Test coverage 37%→80% | Added phase breakdown | High |
| JBC-667 | SQLite metrics | Extended for history | Medium |

### Launch Blocker Issues (Must Complete by Dec 13)

| Issue | Title | Status | Effort |
|-------|-------|--------|--------|
| ~~JBC-678~~ | Homebrew tap | ✅ Done | - |
| JBC-690 | Conflux audit | Backlog | 1-2 hrs |
| JBC-692 | Verify streaming | Backlog | 30 min |
| JBC-682 | Demo skills | Backlog | 4 hrs |
| JBC-683 | README rewrite | Backlog | 1-2 hrs |
| JBC-684 | Demo video | Backlog | 3 hrs |
| JBC-685 | Launch posts | Backlog | 2 hrs |

---

## Reddit Launch Strategy

### Subreddit Rules Summary

| Subreddit | Members | Self-Promo Policy | Best Approach |
|-----------|---------|-------------------|---------------|
| **r/LocalLLaMA** | 573k | ✅ Allowed (≤10% of activity) | "I built this to solve X" |
| **r/ollama** | 92k | ✅ Self-promo category exists | Direct project post |
| **r/selfhosted** | 400k+ | ✅ Self-hosted projects welcome | Privacy-first angle |
| **r/SideProject** | 453k | ✅ Explicitly promo-friendly | "I built..." format |
| **r/golang** | 250k+ | 🟡 Projects OK, disclose AI usage | Technical depth |
| **r/coolgithubprojects** | 60k | ✅ GitHub projects welcome | Link post |

### Posting Order Strategy

1. **Wave 1 (Dec 7 Morning):** r/LocalLLaMA → Most forgiving, validates messaging
2. **Wave 1 (Dec 7 Afternoon):** r/ollama, r/SideProject, r/coolgithubprojects
3. **Wave 2 (Dec 8):** r/selfhosted → After initial bugs fixed
4. **Wave 3 (Dec 10):** r/golang → After polish, stricter community

### Key Rules to Follow

1. **Be transparent** - "I built this" not hidden promotion
2. **Add value first** - Show the problem you solved
3. **Engage actively** - Respond to every comment for first 2 hours
4. **No cross-posting spam** - Tailor each post to the community
5. **r/golang specific** - Must disclose AI assistance in development

---

## Platform-Specific Post Drafts

### r/LocalLLaMA (Primary Target)

**Title:**
```
I built a CLI tool that routes AI tasks to Ollama first, cloud only when needed - cut my API costs 70-90%
```

**Body:**
```markdown
I was spending $30-50/day on Claude/GPT API calls and realized most of them were simple tasks that didn't need expensive models.

So I built **Skillrunner** - a local-first AI workflow orchestrator in Go. The core idea:

- Simple tasks → Ollama (free)  
- Complex tasks → Cloud APIs (paid, when needed)
- Built-in cost tracking shows exactly what you spend

**Example output:**
```
✅ Phase 1: Extract context    [ollama/qwen2.5:14b]   $0.00
✅ Phase 2: Generate review    [ollama/qwen2.5:14b]   $0.00
────────────────────────────────────────────────────────
Total Cost: $0.00
Cloud Equivalent: $0.08 (Claude Sonnet)
💰 Savings: 100%
```

**Key features:**
- Single Go binary, no infrastructure
- Profile-based routing (cheap/balanced/premium)
- Multi-phase YAML workflows
- Ollama as primary provider

Install: `brew install jbctechsolutions/skillrunner/skillrunner`

GitHub: https://github.com/jbctechsolutions/skillrunner

What workflows would you want pre-built? Looking for feedback on the routing profiles.
```

### r/ollama

**Title:**
```
Skillrunner: Orchestrate multi-phase AI workflows with Ollama as the primary provider
```

**Body:**
```markdown
Built a CLI tool specifically designed around local-first AI. Ollama is the default provider, cloud is only used when you explicitly need it.

**The problem I solved:** Running complex workflows that chain multiple prompts together, while keeping everything local.

**How it works:**
- Define workflows in YAML with multiple phases
- Each phase can use a different model
- Profile-based routing: `--profile cheap` = all Ollama

```yaml
phases:
  - name: extract
    model: ollama/qwen2.5:14b
    prompt: "Extract key points from {{input}}"
  - name: summarize  
    model: ollama/qwen2.5:14b
    prompt: "Summarize: {{phases.extract.output}}"
```

**Unique feature:** Built-in cost tracking. Even though Ollama is free, it shows you what the equivalent cloud cost would be - useful for proving ROI to teams.

GitHub: https://github.com/jbctechsolutions/skillrunner
Install: `brew install jbctechsolutions/skillrunner/skillrunner`

Happy to answer questions about the architecture.
```

### r/selfhosted

**Title:**
```
Skillrunner: Self-hosted AI workflow orchestration - route tasks to local Ollama, cloud only when needed
```

**Body:**
```markdown
For those running Ollama locally, I built a workflow orchestrator that keeps AI tasks self-hosted by default.

**Why I built it:** Cloud AI APIs are convenient but expensive and privacy-concerning. I wanted something that:
- Uses my local Ollama instance first
- Only hits cloud APIs when I explicitly allow it
- Shows me exactly what I'm spending

**Self-hosting angle:**
- Single Go binary - no Docker required (though works great with it)
- All config in `~/.skillrunner/` 
- No external dependencies except your Ollama instance
- Metrics stored locally

**Cost tracking example:**
```
Total Cost: $0.00 (local Ollama)
Cloud Equivalent: $0.08 (if using Claude)
```

Supports: Ollama (primary), Anthropic, OpenAI, Groq

GitHub: https://github.com/jbctechsolutions/skillrunner

Anyone else running Ollama for more than just chat? Curious what workflows people are automating.
```

### r/golang

**Title:**
```
Skillrunner: A Go CLI for local-first AI workflow orchestration
```

**Body:**
```markdown
Sharing a Go project I've been working on - a CLI tool for orchestrating multi-phase AI workflows with intelligent model routing.

**Tech highlights:**
- Single binary distribution via GoReleaser
- YAML-based workflow definitions
- Profile-based routing (cheap → local Ollama, premium → cloud)
- Built-in cost tracking and metrics

**Architecture:**
- Provider adapter pattern for Ollama/Anthropic/OpenAI/Groq
- DAG-based phase execution
- Streaming output support

The routing logic is straightforward - profiles map to model preferences, and the router picks the first available provider that matches.

**Disclosure:** This is my project, built primarily by me with some AI assistance for boilerplate. Happy to discuss the Go-specific design decisions.

GitHub: https://github.com/jbctechsolutions/skillrunner

Feedback welcome, especially on the provider abstraction and CLI structure.
```

### r/SideProject

**Title:**
```
I built Skillrunner - a local-first AI workflow orchestrator to cut my API costs 70-90%
```

**Body:**
```markdown
**The Problem:** I was spending $30-50/day on AI API calls. Half of them were simple extraction or formatting tasks.

**The Solution:** Built Skillrunner in Go - routes simple tasks to local Ollama (free), only uses cloud when needed.

**The Result:** 70-90% cost reduction on most workflows.

**Key features:**
- Single Go binary, zero infrastructure
- Profile-based routing (cheap/balanced/premium)
- Built-in cost tracking
- Multi-phase YAML workflows

GitHub: https://github.com/jbctechsolutions/skillrunner

Looking for feedback from other devs running local LLMs. What workflows would you automate?
```

### r/coolgithubprojects

**Link Post to:** `https://github.com/jbctechsolutions/skillrunner`

**Title:**
```
Skillrunner - Local-first AI workflow orchestration in Go. Route tasks to Ollama first, cloud only when needed.
```

**First Comment:**
```markdown
Key features:
- Single Go binary
- Ollama as primary provider
- Profile-based routing (cheap/balanced/premium)
- Built-in cost tracking
- Multi-phase YAML workflows

Cut API costs 70-90% by running simple tasks locally.
```

### Hacker News (Dec 13)

**Title:**
```
Show HN: Skillrunner – Local-first AI workflow orchestration (cut API costs 70-90%)
```

**Body:**
```markdown
Hi HN, I built Skillrunner because I was spending $30-50/day on Claude/GPT API calls, and realized half those calls were simple tasks that could run locally.

Skillrunner is a CLI tool that orchestrates AI workflows with intelligent model routing:
- Simple tasks → Ollama (free, local)
- Complex tasks → Cloud APIs (paid, when needed)

Key features:
• Single Go binary, zero infrastructure
• Profile-based routing: cheap/balanced/premium
• Built-in cost tracking (see exactly what you spend)
• Multi-phase workflows with YAML definitions
• Skill marketplace for shareable workflows

Example: A code review workflow that costs $0.08 with Claude runs for $0.00 with Skillrunner's local-first routing.

Unlike coding agents like OpenCode (single-session, interactive), Skillrunner is for reproducible multi-phase workflows you can version control and share.

GitHub: https://github.com/jbctechsolutions/skillrunner
Install: brew install jbctechsolutions/skillrunner/skillrunner

Would love feedback on the routing profiles and skill format!
```

### Twitter/X Thread

```
🧵 I built Skillrunner because I was spending $30-50/day on AI API calls.

Half those calls were simple tasks that could run locally for FREE.

Here's what I learned building local-first AI orchestration...

1/8
---
The insight: Not every AI task needs Claude or GPT-4.

Extraction? Local model.
Formatting? Local model.  
Complex reasoning? Okay, use the cloud.

Skillrunner routes tasks intelligently based on profiles: cheap / balanced / premium

2/8
---
The results:

Before: $0.08 per code review (Claude Sonnet)
After: $0.00 per code review (Ollama locally)

70-90% cost reduction on most workflows.

3/8
---
How it works:

Define workflows in YAML with multiple phases.
Each phase can specify its own model or use the profile default.

Local models handle extraction and formatting.
Cloud models handle complex reasoning when needed.

4/8
---
What makes Skillrunner different:

✅ Single Go binary (no Java infrastructure)
✅ Built-in cost tracking (unique - no competitor has this)
✅ Profile-based routing (simpler than config files)
✅ Skill marketplace (shareable workflows)

5/8
---
Providers supported:

• Ollama (local, free)
• Anthropic (Claude)
• OpenAI (GPT-4o)
• Groq (10-50x faster inference)

Local-first, cloud when needed.

6/8
---
Install in 30 seconds:

brew install jbctechsolutions/skillrunner/skillrunner
sr init
sr ask "explain this code" --input file.go

7/8
---
Try it out:

🔗 GitHub: github.com/jbctechsolutions/skillrunner
📺 Demo: [link to demo video]

Would love your feedback - what workflows would you want pre-built?

8/8
```

### LinkedIn

```
🚀 Launching Skillrunner: Local-first AI workflow orchestration

After spending $30-50/day on AI API calls, I realized something: half those calls were simple tasks that could run locally for free.

Skillrunner routes AI tasks intelligently:
→ Simple tasks: Ollama (free, local)
→ Complex tasks: Cloud APIs (paid, when needed)

The result? 70-90% cost reduction on most workflows.

Key features:
• Single Go binary - no infrastructure needed
• Built-in cost tracking - see exactly what you spend
• Profile-based routing - cheap/balanced/premium
• Skill marketplace - shareable, version-controlled workflows

Unlike single-session coding agents, Skillrunner orchestrates reproducible multi-phase workflows you can share with your team.

Check it out: github.com/jbctechsolutions/skillrunner

#AI #DevTools #OpenSource #Golang #LocalLLM
```

---

## Pre-Launch Checklist

### Technical Verification (Dec 5-6)

- [ ] `brew install jbctechsolutions/skillrunner/skillrunner` works
- [ ] `sr --version` shows correct version
- [ ] `sr init` creates `~/.skillrunner/` directory
- [ ] `sr ask "hello"` with Ollama produces streaming output
- [ ] Cost tracking appears in output
- [ ] No "Conflux" references in CLI output

### Code Audit (JBC-690)

```bash
# Run these commands and fix any hits
grep -rn "conflux" --include="*.go" .
grep -rn "Conflux" --include="*.go" .
grep -rn "conflux" --include="*.md" .
grep -rn "Conflux" --include="*.md" .
grep -rn "conflux" --include="*.yaml" .
```

### Documentation

- [ ] README has correct install command
- [ ] README shows cost comparison example
- [ ] Quick start guide works end-to-end
- [ ] All examples use "Skillrunner" not "Conflux"

### Demo Preparation

- [ ] Have sample code file ready for demo
- [ ] Test demo skill produces cost output
- [ ] Record terminal session for GIF (optional for soft launch)

---

## Post-Launch Monitoring

### First 2 Hours After Each Post

- [ ] Monitor for comments
- [ ] Respond to every comment
- [ ] Note any install issues
- [ ] Track upvote velocity

### Metrics to Track

| Metric | Good | Great | Excellent |
|--------|------|-------|-----------|
| r/LocalLLaMA upvotes | 50+ | 100+ | 200+ |
| GitHub stars (day 1) | 10+ | 25+ | 50+ |
| Install issues reported | <3 | 0-1 | 0 |
| Comments (engagement) | 10+ | 20+ | 40+ |

### Issue Response Templates

**Install Issue:**
```
Thanks for reporting! Can you share:
1. OS and version?
2. Error message?
3. Output of `brew --version`?

Will get this fixed ASAP.
```

**Feature Request:**
```
Great idea! Created an issue for this: [link]

Would you be interested in testing it when ready?
```

**Bug Report:**
```
Thanks for the detailed report! This is helpful.

Can you also share:
1. Skillrunner version (`sr --version`)
2. Ollama version (`ollama --version`)
3. Config file (redact API keys)

Will investigate today.
```

---

## January Priorities

### Security (Week 1-2)

| Issue | Title | Priority | Effort |
|-------|-------|----------|--------|
| JBC-650 | API key encryption (AES-GCM) | Urgent | 8 hrs |
| JBC-654 | Remove secrets from logs | High | 4 hrs |

### Reliability (Week 2-3)

| Issue | Title | Priority | Effort |
|-------|-------|----------|--------|
| JBC-655 | Circuit breakers | High | 8 hrs |
| JBC-658 | Health check endpoints | Medium | 4 hrs |

### Quality (Week 3-4)

| Issue | Title | Priority | Effort |
|-------|-------|----------|--------|
| JBC-659 | Test coverage 37%→80% | High | 6-8 days |
| JBC-657 | Structured logging (slog) | Medium | 9-11 hrs |

### Features (End of January)

| Issue | Title | Priority | Effort |
|-------|-------|----------|--------|
| JBC-691 | MCP Protocol Support | High | 5-7 days |
| JBC-667 | SQLite metrics | Medium | 7-8 hrs |

---

## Appendix: Key Messages

### One-Liner
> "Local-first AI workflow orchestration - cut API costs 70-90%"

### Elevator Pitch
> "Skillrunner routes AI tasks to local Ollama first, cloud only when needed. Built-in cost tracking shows exactly what you spend. Single Go binary, zero infrastructure."

### Problem Statement
> "I was spending $30-50/day on Claude/GPT API calls. Half were simple tasks that didn't need expensive models."

### Solution Statement
> "Simple tasks → Ollama (free). Complex tasks → Cloud (paid). Profile-based routing handles the decision automatically."

### Differentiator
> "No competitor has built-in cost tracking. Skillrunner shows exactly what you spend and what you save."

### Technical Credibility
> "Single Go binary. YAML workflows. Provider adapter pattern. Streaming output. Cost metrics stored in SQLite."

---

## Appendix: Competitive Quick Reference

### When Asked "How is this different from X?"

**vs ADK-Go:**
> "ADK-Go is great but has no cost tracking. Skillrunner shows exactly what you spend and saves. Also, simpler profile-based routing vs complex configuration."

**vs OpenCode:**
> "OpenCode is for interactive coding sessions. Skillrunner is for reproducible multi-phase workflows you can version control and share."

**vs CrewAI/AutoGen:**
> "Those are Python agent frameworks. Skillrunner is a single Go binary with zero infrastructure - just install and run."

**vs Conductor/Temporal:**
> "Those require Java infrastructure and orchestration servers. Skillrunner is a single binary that runs anywhere."

---

*Document generated from Claude conversation on December 3, 2025*
