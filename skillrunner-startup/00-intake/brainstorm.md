# Skillrunner Brainstorm (Revised)

**Date:** 2026-03-09
**Phase:** 2 - Brainstorm (Revision 2)
**Confidence:** Medium-High
**Flags:** Major pivot — from developer tool to consultant/agency workflow engine

---

## The Journey of Pivots

| # | Vision | Why It Failed |
|---|--------|---------------|
| 1 | Route Claude Code to local/cheap models | claude-code-router (29k stars) already does this |
| 2 | Standalone CLI workflow orchestrator | Founder can't use it — work is interactive, not batch |
| 3 | LLM proxy with learning | Bifrost, LiteLLM, Helicone already dominate |
| 4 | Enterprise AI platform with web UI | Dify (130k stars), needs a team, not a side project |
| 5 | **Workflow engine for consulting deliverables** | **Untested — but matches founder's actual work** |

---

## The Refined Idea (v5)

**Skillrunner as a workflow engine for consultants and agencies who produce repeatable AI-assisted deliverables.**

### Founder's Actual Workflows

| Workflow | Volume | Current Tool | Pain |
|----------|--------|-------------|------|
| Proposals from client notes | Regular | GitHub Actions + markdown templates | Poor formatting, manual steps |
| SOC2 audits | Per engagement | Manual | Time-intensive, repetitive |
| Translation (25k pages, 3 languages) | Massive, ongoing | Manual + DeepL | Human-AI coordination, terminology management |
| Content pipelines (newsletters, blog, marketing, prayer cards) | Regular | Manual | Staff approval bottleneck, grunt work |
| Invoices | Regular | GitHub Actions | Works OK |

### Why This Fits Skillrunner

1. **These are YAML-definable workflows** — structured, repeatable, multi-phase
2. **They need human-in-the-loop** — not fully automated, staff reviews and approves
3. **They span multiple tools** — DeepL, LLMs, document conversion, file management
4. **Cost tracking matters** — client-billable AI usage needs to be tracked
5. **The founder would actually USE this** — solves his own daily problems

### The Niche

**Fractional CTOs, consultants, and small agencies serving churches/non-profits**

- Small niche but reachable (founder already has clients and referrals)
- Price-sensitive market → cost tracking is genuinely valuable
- Repeatable deliverables across clients → workflow templates
- Non-technical staff need to interact → future web UI justified by real need

---

## What Needs to Change in Skillrunner

### Must Add
- **Human-in-the-loop phases** — pause workflow, wait for human review/approval, continue
- **Better document output** — branded DOCX/PDF generation (founder already has this in proposal-management)
- **External tool integration** — DeepL API, Google Drive, Notion
- **Template variables** — same workflow, different client context

### Already Built (Reusable)
- Multi-provider LLM support (Anthropic, OpenAI, Groq, Ollama)
- YAML workflow definition
- Cost tracking per phase
- Session resume / checkpoint
- MCP tool execution

### Can Drop / Deprioritize
- Competing with Claude Code / Cursor
- LLM proxy / gateway functionality
- Enterprise AI platform ambitions
- Developer-focused positioning

---

## Competitive Landscape (Revised)

| Tool | What It Does | Overlap with New Vision |
|------|-------------|------------------------|
| n8n | General automation | Partial — can do AI workflows but not AI-native |
| Dify | LLM app platform | Partial — visual workflows but cloud-focused, not consultant-oriented |
| GitHub Actions | CI/CD automation | What founder uses now — rigid, poor formatting |
| Zapier/Make | Business automation | No AI-native workflows, expensive at scale |
| **Nothing** | Consultant workflow engine with human-in-the-loop AI | **Gap** |

---

## Anti-Patterns Detected

- **[RESOLVED]** "Solution looking for a problem" → now targeting founder's actual problems
- **[WATCH]** "Boiling the ocean" → must resist adding proxy/enterprise features
- **[WATCH]** "Building in stealth" → need to ship something usable in weeks, not months
