# Readwise Reader Cleanup - Context for New Session

**Date:** December 4, 2025
**Purpose:** Clean up Readwise Reader feeds to reduce noise for n8n market research automation

---

## Background

You're building an n8n-based market research automation that runs daily and sends digests to Slack. Readwise Reader is one of the input sources - it captures articles/threads you save to read later.

The problem: Your Reader account has accumulated feeds that create noise. You need to curate it so the automation surfaces high-signal content.

---

## What You're Building

See full spec: `docs/specs/N8N_MARKET_RESEARCH_AUTOMATION.md`

**Key points:**
- n8n deployed at automation.jbc.dev
- Daily digest to Slack analyzing: GitHub Trending, HN, Reddit, Twitter accounts, Readwise Reader
- Claude API for analysis (~$15/mo)
- Reader integration via Readwise API v3: `GET /api/v3/list/`

---

## How Reader Fits In

Reader content is HIGH SIGNAL because you explicitly saved it. The n8n workflow will:

1. Pull articles saved in the last 24 hours
2. Have Claude identify themes and extract key points
3. Include a "From Your Reading Queue" section in the daily digest

Example output:
```markdown
## From Your Reading Queue (3 articles saved yesterday)

**Theme:** All 3 relate to AI agent orchestration

1. "Building Reliable AI Agents" - saved from @simonw
   - Key point: Retry logic is critical for multi-step workflows

2. "The Cost of AI in Production" - saved from HN
   - Key point: Most companies overspend 40% on LLM calls
```

---

## Cleanup Criteria

When reviewing your Reader feeds/sources, keep things relevant to:

**SkillRunner (your product):**
- AI/LLM cost optimization
- Workflow orchestration, DAG execution
- Multi-model routing
- Developer tools, CLI tools
- Go programming

**Competitors to monitor:**
- Langfuse, Helicone, Factory.ai
- LiteLLM, OpenRouter
- Claude Code, Cursor, Cline, Aider

**General professional:**
- Developer productivity
- Startup/product strategy
- DevOps, infrastructure

**Personal (if you want to keep):**
- Church tech, ProPresenter, AVL
- Productivity systems

---

## Twitter Accounts Already Extracted

You have 633 Twitter accounts from your bookmarks in:
`docs/specs/twitter_accounts_to_monitor.txt`

Top accounts by bookmark frequency:
- @tom_doerr (146 bookmarks)
- @GithubProjects (50)
- @dani_avila7 (28) - Claude Code expert
- @iannuttall (21) - Factory.ai
- @hayesdev_ (16)
- @claudeai, @AnthropicAI, @cursor_ai, @cline

If any of these are in your Reader feeds, they're high-value keeps.

---

## Reddit Subreddits Already Identified

From your subscriptions (keep feeds related to):
- r/LocalLLaMA, r/ollama, r/ClaudeAI, r/ChatGPTCoding
- r/golang, r/devops, r/selfhosted
- r/n8n, r/ObsidianMD, r/Notion
- r/churchtech, r/ProPresenter

---

## Questions to Ask Yourself

For each feed/source in Reader:
1. Does this relate to SkillRunner or my professional work?
2. Have I actually read/engaged with content from this source?
3. Is this duplicated by another source in my n8n workflow?
4. Is the signal-to-noise ratio good?

If no to all, unsubscribe/remove.

---

## After Cleanup

Once Reader is cleaned up, the n8n automation will pull from a curated set of sources, making the daily digest more actionable.

---

*Context saved December 4, 2025 for continuation in new Claude session*
