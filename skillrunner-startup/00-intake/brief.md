# Skillrunner Startup Brief

**Date:** 2026-03-09
**Phase:** 1 - Intake
**Confidence:** Medium
**Flags:** Product-market fit uncertainty, form factor question unresolved

---

## The Idea

**Problem:** Developers using AI coding tools (Claude Code, Aider, Cursor, etc.) have no visibility into which models perform best for which tasks, and no way to automatically route requests to the optimal model for cost and quality.

**Current Solution (built):** A Go single-binary CLI workflow orchestrator with YAML-based skills, multi-provider support (Ollama, Anthropic, OpenAI, Groq), built-in cost tracking, MCP tool execution, session resume, and worktree isolation. Hexagonal architecture, v1.4.

**Actual Vision:** An intelligent routing layer that sits *between* existing developer tools and LLM APIs. It learns which models handle which task types best and routes accordingly — reducing cost and improving quality without replacing the tools developers already use.

**Core Tension:** The product as built is a standalone CLI that replaces existing tools. The founder's actual desire is to augment them. The interception/proxy mechanism hasn't been figured out.

---

## The Founder

- Solo founder, side project
- Background: Consulting (active revenue from separate company)
- Full-time consulting, Skillrunner is evenings/weekends
- No co-founders
- Technical (Go developer, hexagonal architecture, TDD practitioner)

---

## The Market

- **Target:** Individual developers using AI coding tools
- **Secondary:** Teams (future)
- **Geography:** Global (English-first)
- **Existing alternatives:** Langfuse (observability), Helicone (routing + observability), OpenRouter (model routing), Factory.ai (enterprise workflows), plus dozens of agent frameworks
- **Key insight from prior research:** No individual differentiator is unique. The combination (workflow + cost + local + single binary) might be, but the "combination moat" is weak.

---

## The Business

- **Revenue model:** Open core (free CLI + Pro tier, previously debated between $9-19/mo)
- **Current revenue from Skillrunner:** $0
- **Monthly burn (total company):** ~$1,500 (not Skillrunner-specific)
- **Funding:** Bootstrapped, no external funding
- **Prior market analysis:** Extensive (Dec 2025) — concluded "continue but redefine success as $30-50K MRR lifestyle business"

---

## What's Happened (Dec 2025 - Mar 2026)

- December launch (Reddit/HN) did NOT happen — blocked by fear of response
- GitHub stars: 0 (no public launch)
- Ollama partnership: not pursued
- MCP support: shipped (v1.2)
- OpenAI/Groq providers: shipped
- User conversations: mostly with business users (wrong audience), no meaningful learnings
- Product positioning: largely unchanged from December
- Founder hasn't been able to use Skillrunner himself day-to-day

---

## Constraints

- Side project time only
- No budget for marketing/growth
- Solo developer (no team to delegate to)
- Consulting income provides runway but limits time

---

## Hard Questions & Answers

| Question | Answer |
|----------|--------|
| Why haven't you launched? | Fear of the response |
| What would make you walk away? | "I want to use this myself" — no financial kill criteria |
| Talked to potential users? | Business users mostly (wrong audience), learned nothing |
| Strongest argument against? | No individual differentiator is unique |
| Unfair advantage? | Deep Go expertise, low burn, can outlast funded competitors |

---

## Critical Unknowns

1. **Form factor:** How to intercept LLM calls without replacing existing tools
2. **Adaptive routing:** Can a system learn model-task performance and route intelligently?
3. **Distribution:** How to reach individual developers who already use Claude Code/Aider
4. **Validation:** Zero real-world usage data — product hasn't been launched or used by founder

---

## Red Flags

- **[RED]** Product built for 15+ months without launch or real users
- **[RED]** Founder can't use own product — signals product-market fit issue
- **[RED]** Fear blocking launch — will compound if not addressed
- **[YELLOW]** "Combination moat" is weak and easily replicated
- **[YELLOW]** Talked to wrong audience (business users vs developers)
- **[YELLOW]** Vision (intelligent routing proxy) ≠ product (standalone CLI orchestrator)
