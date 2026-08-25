# SkillRunner Release Roadmap v2.0: Cost Intelligence Pivot

**Document Version:** 2.0
**Created:** December 4, 2025
**Prepared for:** JBC Tech Solutions - Strategic Pivot

---

## Strategic Context

Following competitive analysis of Factory.ai ($50M Series B, $300M valuation), we are pivoting SkillRunner's positioning from "workflow orchestrator" to **"AI cost intelligence platform"**.

**Key Insight:** Factory's business model (token abstraction) structurally prevents them from offering cost transparency. This is our defensible moat.

---

## New Positioning

**FROM:** "Local-first AI workflow orchestrator with built-in cost tracking"

**TO:** "The only AI tool that shows you exactly what you're spending—and automatically optimizes it"

**Tagline:** "Know what your AI costs. Optimize automatically."

---

## Revised Timeline

```
Dec 2025    Jan 2026    Feb 2026    Mar 2026    Apr 2026    May 2026
    |           |           |           |           |           |
 v1.0.0      v1.1.0      v1.2.0      v1.3.0      v1.4.0    DECISION
 Cost        Providers   Skill       Cost SDK    Cost API   POINT
 Launch      +Benchmarks DECOMP      (Alpha)     +Dashboard
                        [ACCELERATED]
```

---

## v1.0.0 (December 7-13, 2025) - COST TRANSPARENCY LAUNCH

**Theme:** "See What AI Actually Costs"

### Messaging Pivot
| Old Message | New Message |
|-------------|-------------|
| "Local-first AI workflow orchestrator" | "See what your AI actually costs" |
| "Multi-phase DAG execution" | "Save 70-90% with intelligent routing" |
| "Profile-based routing" | "Automatic cost optimization" |

### Launch Schedule
| Date | Event | Angle |
|------|-------|-------|
| Dec 7 | Reddit soft launch | "I was spending $40/day on AI APIs..." |
| Dec 13 | Hacker News | "Show HN: See what AI actually costs" |

### Success Metrics (Updated)
| Metric | Target | Why Changed |
|--------|--------|-------------|
| GitHub Stars | 500+ | Unchanged |
| "Cost savings" mentions in feedback | 50%+ | NEW - validates positioning |
| Discord cost-tracking channel activity | Active | NEW - community signal |

### Required Changes Before Launch
- [ ] Update README to lead with cost tracking
- [ ] Add "Saved $X.XX vs cloud-only" to execution output
- [ ] Update CLI help text
- [ ] Prepare cost-focused launch posts

---

## v1.1.0 (January 24, 2026) - Providers + Cost Benchmarks

**Theme:** "More Providers, More Savings Data"

### Features
| Feature | Priority | Notes |
|---------|----------|-------|
| OpenAI provider (JBC-680) | HIGH | |
| Groq provider (JBC-681) | HIGH | Cheapest cloud option |
| **Public cost benchmarks page** | **HIGH** | NEW - differentiation |
| **Cost alerts/budgets** | **HIGH** | NEW - user request |
| Streaming improvements (JBC-692) | MEDIUM | |

### New: Cost Benchmarks
```bash
$ sr benchmarks

Provider Comparison (code review task):
┌──────────────────────────────────────────────────────┐
│ Provider          │ Cost/1K tokens │ Speed   │ Quality │
├──────────────────────────────────────────────────────┤
│ Ollama (local)    │ $0.00          │ 45 tok/s│ 85%     │
│ Groq              │ $0.05          │ 500 tok/s│ 88%    │
│ Anthropic Haiku   │ $0.25          │ 80 tok/s│ 90%     │
│ OpenAI GPT-4o     │ $2.50          │ 60 tok/s│ 92%     │
│ Anthropic Sonnet  │ $3.00          │ 70 tok/s│ 95%     │
└──────────────────────────────────────────────────────┘

SkillRunner recommendation: Use Ollama for phases 1-3, Sonnet for phase 4
Estimated savings: 73% ($0.08 vs $0.30)
```

### Marketing Message
"Now with 4+ providers. See exactly what each one costs."

---

## v1.2.0 (February 14, 2026) - Skill Decomposition [ACCELERATED]

**Theme:** "Automatic Cost Optimization"

**MOVED UP 3 WEEKS** - This is the core defensible moat.

### Why Accelerate
- Factory has NO automatic cost optimization
- This is genuinely unique in the market
- Combines with cost tracking for maximum differentiation

### Features
| Feature | Issue | Priority |
|---------|-------|----------|
| Skill Complexity Analyzer | JBC-703 | CRITICAL |
| Task Type Classifier | JBC-704 | CRITICAL |
| Automatic Model Selector | JBC-708 | CRITICAL |
| `sr skill analyze` | JBC-709 | HIGH |
| `sr skill convert --optimize` | JBC-710 | HIGH |
| Cost prediction before execution | NEW | HIGH |

### Key Output
```bash
$ sr skill analyze code-review.yaml

Analysis Results:
─────────────────────────────────────────────────────────
Complexity Score: 0.78 (High - excellent optimization target)

Current Cost Estimate: $0.45 per run
Optimized Cost Estimate: $0.08 per run
Potential Savings: 82% ($0.37 per run)

Optimization Recommendations:
  Phase 1 (file-scan): Move to Ollama → saves $0.12
  Phase 2 (pattern-match): Move to Ollama → saves $0.15
  Phase 3 (analysis): Keep on Sonnet (quality-critical)
  Phase 4 (report): Move to Haiku → saves $0.10

Run `sr skill convert --optimize code-review.yaml` to apply
─────────────────────────────────────────────────────────
```

### Marketing Message
"Import any workflow. We'll show you how to cut costs by 70-90%."

---

## v1.3.0 (March 14, 2026) - Cost SDK Alpha

**Theme:** "Cost Tracking for Any Tool"

### Strategic Shift
Begin positioning SkillRunner as infrastructure, not just a CLI tool.

### Features
| Feature | Priority | Notes |
|---------|----------|-------|
| Extract cost tracking as Go library | HIGH | `github.com/jbctechsolutions/sr-cost` |
| Cost tracking API (local REST) | HIGH | For integrations |
| Integration examples | MEDIUM | Factory, Claude Code, Cursor |
| Prometheus metrics export | MEDIUM | FinOps integration |

### New Package Structure
```
sr-cost/
├── calculator.go    # Cost computation
├── tracker.go       # Execution recording
├── providers/       # Provider pricing data
├── export/          # Prometheus, JSON, CSV
└── examples/
    ├── standalone/
    ├── factory-integration/
    └── claude-code-wrapper/
```

### Marketing Message
"Track AI costs across any tool. SkillRunner's cost engine, everywhere."

---

## v1.4.0 (April 11, 2026) - Cost API + Dashboard

**Theme:** "See Your AI Spend"

### Features
| Feature | Priority | Notes |
|---------|----------|-------|
| REST API for cost data | HIGH | |
| Simple web dashboard | HIGH | Local-first, single binary |
| Team/project allocation | MEDIUM | |
| Budget enforcement | MEDIUM | Hard limits |
| Historical trends | MEDIUM | Charts |

### Dashboard MVP
```
┌─────────────────────────────────────────────────────────────┐
│  SkillRunner Cost Dashboard                    [7 days ▼]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Total Spend: $23.45        Saved vs Cloud-Only: $156.78   │
│  ████████████░░░░░░░░░░░░   ████████████████████████████   │
│  Budget: $50/week           Savings Rate: 87%              │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  By Provider                │  By Skill                    │
│  ─────────────              │  ──────────                  │
│  Ollama: $0.00 (312 runs)   │  code-review: $12.34        │
│  Anthropic: $18.90          │  doc-gen: $6.78              │
│  OpenAI: $4.55              │  test-gen: $4.33             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## May 2026: DECISION POINT

### Evaluate Based on Traction

| Signal | Continue CLI Focus | Pivot to SaaS |
|--------|-------------------|---------------|
| GitHub Stars | <2K | >5K |
| Weekly Active Users | <500 | >2K |
| SDK adoption | Low | Growing |
| Enterprise inquiries | 0 | 3+ |

### Option A: Continue CLI + SDK
- Keep building cost intelligence tools
- Target $30-50K MRR lifestyle business
- Focus on developer community

### Option B: Pivot to Cost SaaS
- Build full web platform
- Raise seed funding
- Target enterprise FinOps teams

---

## DEPRIORITIZED (Moved to Backlog)

| Original Feature | Original Version | Status | Rationale |
|------------------|------------------|--------|-----------|
| MCP Client Support | v1.3.0 | BACKLOG | Factory already has MCP |
| `sr chat` | v1.3.0 | BACKLOG | Nice-to-have, not differentiating |
| Desktop GUI | v1.5.0 | BACKLOG | High effort, low differentiation |
| Web Platform | v2.0.0 | DECISION | Wait for traction signals |
| Enterprise SSO/SAML | v1.4.0 | BACKLOG | Premature, Factory owns enterprise |
| On-premise deployment | v2.0.0 | BACKLOG | Wait for enterprise demand |

---

## Revised Linear Issue Priorities

### CRITICAL (Do Now)
| Issue | Description | Version |
|-------|-------------|---------|
| NEW | README + messaging pivot | v1.0.0 |
| NEW | Cost output enhancement | v1.0.0 |
| JBC-680 | OpenAI provider | v1.1.0 |
| JBC-681 | Groq provider | v1.1.0 |

### HIGH (Accelerate)
| Issue | Description | Version |
|-------|-------------|---------|
| JBC-703 | Skill Complexity Analyzer | v1.2.0 |
| JBC-704 | Task Type Classifier | v1.2.0 |
| JBC-708 | Automatic Model Selector | v1.2.0 |
| NEW | Cost benchmarks page | v1.1.0 |
| NEW | Cost alerts/budgets | v1.1.0 |

### MEDIUM (Keep on Track)
| Issue | Description | Version |
|-------|-------------|---------|
| JBC-692 | Streaming verification | v1.1.0 |
| JBC-709 | `sr skill analyze` | v1.2.0 |
| JBC-710 | `sr skill convert --optimize` | v1.2.0 |
| JBC-650 | API key encryption | v1.3.0 |

### LOW (Deprioritize)
| Issue | Description | Original Version |
|-------|-------------|------------------|
| JBC-691 | MCP Protocol Support | v1.3.0 → BACKLOG |
| JBC-707 | DAG Decomposer | v1.3.0 → v1.2.0 (simplified) |
| JBC-711 | Quality Validation Framework | v1.3.0 → BACKLOG |

---

## Pricing Adjustment

### Current Plan
| Tier | Price |
|------|-------|
| Community | Free |
| Pro | $19/mo |
| Team | $49/user/mo |

### Revised Plan
| Tier | Price | Notes |
|------|-------|-------|
| Community | Free | CLI + local everything |
| Pro | **$9/mo** | Cloud sync + alerts (lower to compete with Factory free) |
| Team | $29/user/mo | API access + dashboards |

---

## 72-Hour Execution Checklist

### Day 1 (December 4)
- [ ] Update README.md with cost-first messaging
- [ ] Enhance cost output in CLI
- [ ] Draft Reddit post with cost angle
- [ ] Draft HN post with cost angle

### Day 2 (December 5)
- [ ] Update this roadmap document
- [ ] Reprioritize Linear issues
- [ ] Update CLI help text
- [ ] Create cost comparison screenshot for launch

### Day 3 (December 6)
- [ ] Final review of all messaging
- [ ] Test cost output enhancements
- [ ] Prepare Discord channels
- [ ] Buffer for fixes

### Launch (December 7)
- [ ] Reddit soft launch
- [ ] Monitor feedback for "cost" mentions
- [ ] Iterate based on response

---

## Success Criteria for Pivot

### 30-Day Check (January 7, 2026)
| Metric | Target | Signal |
|--------|--------|--------|
| GitHub Stars | 500+ | Market interest |
| % feedback mentioning costs | >50% | Positioning resonance |
| Cost savings reported by users | Any | Product-market fit |

### 90-Day Check (March 7, 2026)
| Metric | Target | Signal |
|--------|--------|--------|
| GitHub Stars | 1,500+ | Growing interest |
| Weekly Active Users | 500+ | Retention |
| Skill decomposition usage | 100+ skills analyzed | Feature adoption |

---

*Document prepared December 4, 2025*
*Strategic pivot in response to Factory.ai competitive analysis*
