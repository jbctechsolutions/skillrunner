# Research Gate: Go / No-Go Assessment

**Date:** 2026-03-09
**Phase:** 3.5 — Research Gate Checkpoint
**Decision Required:** Continue to Phase 4 (Strategy), pivot, or stop

---

## What the Research Found

### Market Size
- **TAM:** ~$3.6B (AI productivity tools, US)
- **SAM:** ~$249M (all consultants who might use workflow tools)
- **CLI-Adjusted SAM:** ~$61M (only consultants willing to use CLI tools)
- **Realistic Year 1:** $50K-$90K ARR
- **Year 3 with PMF:** $350K-$2.4M ARR
- **Confidence:** Low-Medium. The CLI-adjusted SAM rests on an unvalidated 20-30% assumption.

### Competition
- **Direct competitors at the exact intersection (CLI + YAML + AI + HITL + consultant):** Zero
- **Closest philosophical match:** Lobster/OpenClaw (small community project)
- **Dominant horizontal players:** n8n ($253M raised, $40M ARR), Make ($232M), Zapier (~$400M ARR)
- **Critical substitute threat:** ChatGPT and Claude as "good enough" ad-hoc workflow tools
- **Platform risk:** OpenAI/Anthropic could add saved workflow templates in 18-24 months

### Customer Demand Signals
- 92% of nonprofits adopted AI, but only 7% say it expanded capability → structured workflows needed
- Fractional CTO market doubled to 120K (2022-2024), now mainstream
- 42% higher margins reported when consultants productize engagements
- 30% of churches prioritize multilingual tools (validates translation pipeline)
- Top pain: "Every engagement starts from scratch" — IP leaks into client folders

### Timing
- **Score: 8/10** — Strong, but window is 18-24 months
- Gartner warns 40% of agentic AI projects will be cancelled by 2027
- EU AI Act transparency rules effective August 2026
- Goldman Sachs expects measurable AI GDP impact starting 2027

### Unique Assets
- 181-tool Office 365 MCP server (largest single-server found in ecosystem)
- Existing church/nonprofit client relationships
- Founder IS the target persona (fractional CTO)
- Working product with multi-provider LLM, YAML workflows, cost tracking

---

## The Honest Assessment

### Strengths
1. **Founder-market fit is exceptional.** You are the persona. You have the clients. You know the workflows. This alone puts you ahead of 90% of startup ideas.
2. **The competitive whitespace is real.** Nobody is building a CLI + YAML + AI + HITL workflow engine for consultants. Not even close.
3. **The timing is right.** The 92% adoption / 7% effectiveness gap is a real market failure that structured workflows can address.
4. **The MCP server is a genuine asset.** 181 tools, dual backend, unpublished — this is distribution waiting to happen.

### Weaknesses
1. **The product isn't usable by the founder yet.** HITL is missing. Open WebUI integration doesn't exist. You've said you can't successfully use your own product. This is the single biggest concern.
2. **Zero primary customer validation.** All demand signals are from secondary research. No fractional CTO has been asked "would you pay for this?" No church IT director has seen a demo.
3. **CLI limits the market by 70-80%.** The CLI-adjusted SAM of $61M is small, and the 20-30% assumption could be wrong.
4. **Open-source monetization is hard.** 70% of micro-SaaS businesses earn under $1K/month. The revenue path requires 3,000-10,000 free users to sustain 150 paying customers.

### The Core Question
Can you build HITL, validate demand with 5-10 real conversations, and ship a working product that you use yourself — all within the 18-24 month market window?

---

## Recommendation: YELLOW LIGHT — Conditional Go

**Not a green light** because:
- The product can't be used by its own founder
- Zero customers have been talked to
- The market size is uncertain (the CLI filter could make it unviably small)

**Not a red light** because:
- The founder-market fit is too strong to ignore
- The competitive gap is genuinely empty
- The timing signals are real and compelling
- The existing assets (MCP server, client relationships, working product core) provide a foundation

### Conditions for Proceeding to Phase 4

Before investing in full strategy, brand, and product planning, these three conditions should be met:

| # | Condition | Effort | Why It Matters |
|---|-----------|--------|---------------|
| 1 | **Talk to 5 fractional CTOs** about their workflow pain and whether they'd use a CLI tool | 2 weeks, $0 | Validates or invalidates the entire buyer thesis |
| 2 | **Build a working HITL prototype** — pause workflow, human reviews in terminal, workflow continues | 2-4 weeks engineering | Until the founder can use the product, everything is theoretical |
| 3 | **Publish the MCP server** to 3+ registries and measure inbound interest | 1 week, $0 | Tests whether the distribution asset actually drives awareness |

If all three come back positive (fractional CTOs show interest, HITL works, MCP server gets traction), proceed to Phase 4 with confidence.

If the conversations reveal that fractional CTOs build their own tools rather than buy them, or that CLI is a non-starter for the target audience, the product thesis needs revision before strategy work begins.

---

## Alternative Paths (If Yellow Turns Red)

If the conditions above fail, these alternatives emerged from the research:

1. **MCP Server as standalone product.** The 181-tool Office 365 MCP server may be more valuable than Skillrunner itself. List it, build community, and explore monetization through the emerging MCP marketplace ecosystem.

2. **Services-first, product-second.** Deploy Skillrunner workflows for your own consulting clients (you already have them). Charge consulting rates, not SaaS prices. Let the product emerge from actual client work rather than market speculation.

3. **Open WebUI plugin focus.** Build Skillrunner workflows as Open WebUI tools/functions. Leverage the 347K user community instead of building from scratch. Less ambitious, but faster to market.

4. **Pivot to non-CLI delivery.** If CLI adoption is truly 10-15% (not 20-30%), consider whether a web-based YAML editor with Skillrunner as the backend could open the other 70-80% of the market.

---

*Assessment based on 11 research agents, ~85 web searches, 10 raw research documents, 4 synthesis reports, and 3 prior session documents. All findings tagged with [Data], [Estimate], or [Assumption] confidence levels.*
