# SkillRunner: Viable Business Model Analysis

**Date:** December 4, 2025
**Question:** Can multi-phase DAG + automatic model selection become a $10K MRR business in 1 year?

---

## Executive Summary

**YES, but the automatic routing alone is not unique.** Multiple well-funded startups are building this:

| Competitor | Funding | Customers | What They Do |
|------------|---------|-----------|--------------|
| **Martian** | $9M+ Accenture | 300+ enterprise | Patent-pending "model mapping", auto-routes to optimal LLM |
| **Not Diamond** | $2.3M pre-seed | Enterprise | Custom ML router, learns from your data |
| **Unify AI** | $8M seed (YC) | 3,000+ signups | Neural network router, cost/quality/speed optimization |

**However, there's a gap:** None of these do **workflow-level optimization**. They route individual requests, not multi-phase DAGs.

---

## What's Actually Unique About SkillRunner

| Feature | Martian/Not Diamond/Unify | SkillRunner |
|---------|---------------------------|-------------|
| Single request routing | Yes | Yes |
| Multi-phase DAG execution | **No** | **Yes** |
| Per-phase model selection | **No** | **Yes** |
| Workflow cost optimization | **No** | **Yes** |
| Built-in workflow templates | **No** | **Yes** (code-review, feature-dev, etc.) |
| Learning from outcomes | No (predict only) | **Possible differentiator** |

**The Gap:** Routers optimize API calls. SkillRunner optimizes entire development workflows.

---

## Path to $10K MRR

### Validated Model: Infracost Pattern

Infracost (YC W21) did this for cloud infrastructure costs:
- Open source CLI → SaaS dashboard
- Free CLI, $50/seat/month for teams
- Fortune 500 customers
- **15-24 months to $10K MRR** (estimated)

**SkillRunner can follow this:**

| Timeline | Milestone | Revenue |
|----------|-----------|---------|
| Months 1-6 | Open source CLI, community building | $0 |
| Months 6-12 | Launch, content marketing, early adopters | $0-2K MRR |
| Months 12-18 | Team features, dashboard, first paying customers | $2K-10K MRR |
| Months 18-24 | Enterprise features, scale | $10K+ MRR |

### Pricing That Works

Based on successful dev tools:

| Tier | Price | Features |
|------|-------|----------|
| Community | Free | CLI, local execution, basic cost tracking |
| Pro | $19/mo | Cloud sync, cost alerts, optimization recommendations |
| Team | $49/seat/mo | Dashboard, team budgets, aggregated insights |
| Enterprise | Custom | SSO, VPC, SLAs, custom model training |

**Math:** 200 paying seats at $50/mo = $10K MRR

---

## Revenue Streams That Create a Moat

### 1. Anonymous Statistics → Insights-as-a-Service (10-20% of revenue)

**How it works:**
- Users opt-in to share anonymized execution data
- Aggregate: "Ollama qwen2.5 scores 87% on code review tasks, 45% cheaper than GPT-4"
- Publish benchmarks (builds trust, marketing)
- Sell detailed insights to enterprises

**Validation:**
- LMArena: $100M funding, $600M valuation from preference data
- Scale AI: $2B revenue from RLHF data
- Key: Sell insights, NOT raw data (trust killer)

### 2. Custom Model Training (Premium Feature)

**How it works:**
- Collect user execution outcomes (did the code compile? tests pass?)
- Train routing model on YOUR specific workflows
- "SkillRunner learns that for your codebase, Haiku beats Sonnet on test generation"

**Why it's unique:**
- Martian/Not Diamond predict quality, nobody learns from actual outcomes
- Reinforcement learning from real results = genuine moat

**Pricing:** $99/mo or Enterprise tier

### 3. Workflow Marketplace (Platform Revenue)

**How it works:**
- Users share optimized workflow YAML files
- "This code-review workflow saved me 80% vs Claude Code"
- Revenue split on premium workflows

**Validation:**
- Factory has Skills system but no marketplace
- npm/Raycast extensions prove developer willingness to share

### 4. Enterprise AI FinOps (Big Money)

**Market context:**
- 21% of enterprises have NO AI cost tracking
- 84% see 6%+ margin erosion from AI costs
- Average enterprise spends $400K on AI apps
- Budgets growing 75% YoY

**What enterprises pay:**
- Entry: $1K-5K/mo
- Mid-market: $5K-20K/mo
- Enterprise: $20K-100K+/year

**SkillRunner position:** "FinOps for AI development workflows"

---

## Competitive Positioning

### Current Landscape

```
                    WORKFLOW FOCUS
                         ↑
                         |
    Factory.ai           |        SkillRunner
    (Skills, Droids)     |        (DAG + Cost Opt)
                         |
   ←─────────────────────┼─────────────────────→
    ENTERPRISE           |              DEVELOPER
    FOCUS                |              FOCUS
                         |
    Martian, Not Diamond |        OpenRouter
    (Request routing)    |        (API gateway)
                         |
                         ↓
                    REQUEST FOCUS
```

**SkillRunner's quadrant:** Developer-focused + Workflow-focused
**Nobody else is there.**

### Differentiation Statement

> "Martian routes your API calls. Factory runs your coding tasks.
> SkillRunner optimizes entire development workflows—planning through deployment—
> automatically selecting the cheapest model for each phase."

---

## What Makes This a $10K+ MRR Business

### Must-Haves (Table Stakes)

- [ ] Multi-phase DAG execution ✓ (you have this)
- [ ] Per-phase cost tracking ✓ (you have this)
- [ ] Multiple provider support (OpenAI, Anthropic, Ollama)
- [ ] Simple YAML workflow definition
- [ ] CLI that just works

### Differentiators (Why Pay)

- [ ] **Automatic model selection** per phase
- [ ] **Pre-built coding workflows** (code-review, feature-dev, refactor)
- [ ] **Cost prediction before execution** ("This will cost $0.08")
- [ ] **Savings dashboard** ("You saved $156 this month")

### Moat Builders (Why Stay)

- [ ] **Learning from outcomes** - improves over time
- [ ] **Aggregated benchmarks** - "Best model for X task"
- [ ] **Workflow marketplace** - network effects
- [ ] **Custom training** - your routing, your data

---

## Recommended Strategy

### Phase 1: Launch (Months 1-3)
**Focus:** Prove the workflow optimization value

- Ship v1.0 with 3 pre-built workflows
- Automatic model selection (rule-based first)
- Show cost savings on every execution
- Reddit/HN launch with "I saved $X" angle

**Success metric:** 500 GitHub stars, 50 active users

### Phase 2: Learn (Months 4-6)
**Focus:** Collect data, understand users

- Add opt-in anonymous telemetry
- Track which models work for which tasks
- Publish first benchmarks ("Best models for code review")
- Build community (Discord)

**Success metric:** 1,000 stars, 10 testimonials with $ savings

### Phase 3: Monetize (Months 7-12)
**Focus:** First paying customers

- Launch Pro tier ($19/mo) - cloud sync, alerts
- Launch Team tier ($49/seat) - dashboard, budgets
- ML-based routing (use collected data)
- First enterprise pilot

**Success metric:** $5K MRR, 3 team accounts

### Phase 4: Scale (Months 12-18)
**Focus:** $10K MRR and beyond

- Custom training feature (learn from your outcomes)
- Workflow marketplace
- Enterprise features (SSO, VPC)
- Insights-as-a-service

**Success metric:** $10K+ MRR, 1 enterprise contract

---

## Risk Assessment

### High Risk
| Risk | Mitigation |
|------|------------|
| Martian copies workflow optimization | Move fast, build community moat |
| Factory adds cost tracking | Different market (enterprise vs developer) |
| LangGraph adds cost optimization | They're a framework, not a product |

### Medium Risk
| Risk | Mitigation |
|------|------------|
| Market too small | AI FinOps is $14B+ market, growing 40%/yr |
| No one pays for dev tools | Infracost, Plausible, Ghost all proved otherwise |
| Takes too long | 18-24 months is realistic, not 12 |

### Low Risk
| Risk | Mitigation |
|------|------------|
| Technical difficulty | You've already built the core |
| Open source competition | First mover + community = defensible |

---

## Final Assessment

### Is this viable for $10K MRR in 1 year?

**Optimistic:** Yes, if launch goes viral and enterprises bite early
**Realistic:** 18-24 months is more likely
**Pessimistic:** Never, if Martian/Factory copy the workflow angle

### Probability Update

| Outcome | Probability | Reasoning |
|---------|-------------|-----------|
| $10K MRR in 12 months | 20% | Aggressive but possible |
| $10K MRR in 18-24 months | 45% | Follows Infracost pattern |
| $5K MRR lifestyle business | 25% | Covers costs, not growth |
| Shutdown / acqui-hire | 10% | Market validated, worst case = talent acquisition |

### Go/No-Go Recommendation

**GO** - with these conditions:
1. Ship v1.0 in 2 weeks with cost-optimized workflows
2. Launch on Reddit/HN with "saved $X" messaging
3. If <200 stars in 30 days, reconsider
4. If no paying customers in 6 months, pivot or stop

**The unique angle exists. The market exists. The question is execution speed.**

---

## Sources

### Automatic LLM Routing Competition
- [Martian - Model Routing](https://withmartian.com/)
- [Not Diamond - AI Model Router](https://www.notdiamond.ai/)
- [Unify AI - YC-backed LLM Selection](https://techcrunch.com/2024/05/22/unify-helps-developers-find-the-best-llm-for-the-job/)
- [RouteLLM - Open Source Framework](https://routellm.dev/)

### LLM Benchmarking Business Models
- [LMArena - $100M at $600M valuation](https://research.contrary.com/company/lmarena)
- [Scale AI - $2B revenue](https://sacra.com/c/scale-ai/)
- [Hugging Face - $70M ARR](https://productmint.com/hugging-face-business-model/)

### Developer Tools $10K MRR Examples
- [Plausible - 15 months to $10K MRR](https://plausible.io/blog/growing-saas-mrr)
- [Ghost - $7.5M ARR](https://getlatka.com/companies/ghost)
- [Infracost - Cloud Cost CLI](https://www.infracost.io/)
- [Tailwind CSS - $4M+ annually](https://adamwathan.me/tailwindcss-from-side-project-byproduct-to-multi-mullion-dollar-business/)

### AI FinOps Market
- [Cloud FinOps Market - $14B-38B](https://www.precedenceresearch.com/cloud-finops-market)
- [Gartner - $1.5T AI spending 2025](https://www.gartner.com/en/newsroom/press-releases/2025-09-17-gartner-says-worldwide-ai-spending-will-total-1-point-5-trillion-in-2025)
- [State of AI Costs 2025](https://www.cloudzero.com/state-of-ai-costs/)
- [Helicone Pricing](https://www.helicone.ai/pricing)
- [Langfuse Enterprise](https://langfuse.com/enterprise)

---

*Document created December 4, 2025*
*Based on comprehensive market research across 4 parallel research tracks*
