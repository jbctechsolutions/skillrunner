# Skillrunner Competitor Analysis — Deep Dive

**Date:** 2026-03-09
**Analyst:** Claude (AI-assisted research)
**Version:** 1.0
**Data Sources:** Web searches conducted March 2026; all figures sourced from public announcements, pricing pages, and industry reports

---

## Executive Summary

Skillrunner occupies a unique niche at the intersection of three crowded categories: (1) AI workflow automation platforms, (2) proposal/document generation tools, and (3) developer-oriented workflow engines. No single competitor combines CLI-first operation, YAML-defined workflows, human-in-the-loop approval gates, and a specific focus on consultants serving non-profits. However, several adjacent competitors overlap on individual dimensions. This analysis identifies **8 competitors** across direct, adjacent, and emerging categories.

---

## Category 1: AI Workflow Automation Platforms

### 1. n8n

| Attribute | Detail |
|-----------|--------|
| **Website** | [n8n.io](https://n8n.io) |
| **Founded** | 2019 (Berlin, Germany) |
| **Team Size** | ~787 employees (Feb 2026) |
| **Funding** | $253M total; $180M Series C (Oct 2025) led by Accel, Meritech, Redpoint |
| **Valuation** | $2.5B (Oct 2025) |
| **ARR** | $40M+ (2025) |
| **GitHub Stars** | ~173,769 |

**Core Offering:**
Open-source, fair-code workflow automation platform with 400+ integrations, a visual node-based editor, and AI workflow capabilities. Self-hostable or cloud-hosted.

**Key Features:**
- Visual drag-and-drop workflow builder with 400+ integration nodes
- AI agent capabilities with LLM integration
- Self-hosted Community Edition (free, unlimited executions)
- Code execution nodes (JavaScript, Python)
- Webhooks, scheduling, event-driven triggers
- Git integration and multi-environment support (Business plan)

**Pricing:**
- Community (self-hosted): Free (infra costs ~$20-40/mo)
- Cloud Starter: Free tier available
- Cloud Pro: ~EUR 20/mo
- Business (self-hosted): EUR 800/mo (EUR 400/mo with startup discount)

**Target Audience:**
Developers, DevOps, tech-savvy teams needing cross-app automation. Growing into business users with AI features.

**Strengths:**
- Massive community and ecosystem (173K+ GitHub stars)
- Self-hostable with full feature parity
- 400+ integrations out of the box
- Strong AI/LLM integration story
- Well-funded with proven traction ($40M ARR)

**Weaknesses:**
- GUI-first — no CLI-native workflow definition
- Not designed for consultant-specific workflows (proposals, audits)
- Complexity grows quickly for non-technical users
- No built-in human-in-the-loop approval gates for content review
- Overkill for single-consultant use cases

**How Skillrunner Differs:**
- CLI-first vs. GUI-first approach
- YAML workflow definitions live in version control by default
- Purpose-built for consultant deliverables (proposals, audits, content)
- Built-in human-in-the-loop review stages
- Lightweight single-binary vs. server infrastructure
- Fraction of the complexity for targeted use cases

---

### 2. Make (formerly Integromat)

| Attribute | Detail |
|-----------|--------|
| **Website** | [make.com](https://www.make.com) |
| **Founded** | 2012 (Prague, Czech Republic) |
| **Team Size** | ~800+ employees (estimated 2026) |
| **Funding** | $232M total (Series C, 2022) |
| **Partner Network** | 500+ certified solution partners |

**Core Offering:**
Visual automation platform connecting 3,000+ apps with advanced logic, branching, and AI-powered scenario suggestions. Enterprise-grade with SOC 2 compliance.

**Key Features:**
- Visual scenario builder with drag-and-drop modules
- Advanced routers, filters, and conditional logic
- AI-powered scenario suggestions (Pro+, added 2026)
- Rollover operations (unused credits carry forward 1 month)
- 1-minute execution intervals (Pro plan)
- Webhook and schedule triggers

**Pricing:**
- Free: 1,000 credits/month
- Core: $10.59/mo (10,000 credits)
- Pro: $99/mo (advanced features, 1-min intervals)
- Teams: $34.12/mo per user (collaboration features)
- Enterprise: Custom pricing, 24/7 support, 2-hour SLAs

**Target Audience:**
Business users, agencies, consultants needing visual multi-step automation across SaaS tools.

**Strengths:**
- Most intuitive visual builder in the market
- 3,000+ app integrations
- Strong consultant/agency ecosystem (500+ partners)
- Credit-based pricing is predictable
- AI scenario suggestions reduce setup time

**Weaknesses:**
- Entirely GUI-based — no CLI or YAML workflows
- Not designed for document generation or content pipelines
- Credit consumption can get expensive at scale
- No human-in-the-loop content review built in
- Vendor lock-in (no self-hosting option)

**How Skillrunner Differs:**
- CLI-native vs. browser-based UI
- Workflows are code (YAML) vs. visual scenarios
- Purpose-built for deliverable generation, not app integration
- Human review stages are first-class citizens
- Self-hosted by design, no vendor dependency
- No per-operation pricing

---

### 3. Dify

| Attribute | Detail |
|-----------|--------|
| **Website** | [dify.ai](https://dify.ai) |
| **Founded** | 2023 |
| **Team Size** | ~50-80 (estimated 2026) |
| **Funding** | $30M Series Pre-A (March 2026) led by HSG |
| **Valuation** | $180M (March 2026) |
| **GitHub Stars** | ~70,000+ |

**Core Offering:**
Open-source platform for building AI agents and RAG-powered applications. Combines LLM orchestration, knowledge bases, and workflow automation in a low-code/no-code interface.

**Key Features:**
- Visual AI workflow builder with LLM orchestration
- Built-in RAG (Retrieval-Augmented Generation) pipeline
- Knowledge base management with document ingestion
- Multi-model support (OpenAI, Anthropic, local models)
- Agent building with tool/function calling
- Self-hostable (open source)

**Pricing:**
- Sandbox: Free (limited usage)
- Professional: For independent developers and small teams
- Team: For medium teams with higher throughput
- Enterprise: Custom pricing

**Target Audience:**
Developers and startup teams building AI-native applications, particularly those needing RAG pipelines and multi-agent systems.

**Strengths:**
- AI-native design with deep LLM integration
- Strong RAG capabilities for knowledge-intensive tasks
- Open source with active community (~70K GitHub stars)
- Multi-model flexibility
- Growing rapidly ($30M fresh funding)

**Weaknesses:**
- Focused on AI app development, not consultant workflows
- GUI-first — no CLI workflow execution
- No built-in human approval/review stages
- Less mature than n8n/Make for traditional automation
- No consultant-specific templates or deliverable formats

**How Skillrunner Differs:**
- CLI-first execution model
- YAML-defined workflows for version control
- Purpose-built for consultant deliverables, not AI app building
- Human-in-the-loop review is core, not an afterthought
- Simpler mental model — workflow steps, not AI agent graphs
- Targets consultants, not AI developers

---

## Category 2: Proposal & Document Generation Tools

### 4. Proposify

| Attribute | Detail |
|-----------|--------|
| **Website** | [proposify.com](https://www.proposify.com) |
| **Founded** | 2013 (Halifax, Canada) |
| **Team Size** | ~150 employees |
| **Funding** | $27M total |

**Core Offering:**
Online proposal software for sales teams with interactive quotes, e-signatures, and analytics. Focus on visual proposal design and pipeline management.

**Key Features:**
- Drag-and-drop proposal editor with templates
- Interactive pricing tables (CPQ functionality)
- Legally binding e-signatures
- Proposal analytics (open/view tracking)
- CRM integrations (Salesforce, HubSpot)
- Team collaboration with roles and permissions

**Pricing:**
- Basic: $29/user/month
- Team: $49/user/month (quarterly minimum)
- Business: $65/user/month (10-user minimum, $650/mo)

**Target Audience:**
Sales teams, agencies, and freelancers sending client proposals frequently.

**Strengths:**
- Purpose-built for proposal creation and management
- Strong template library and design tools
- Built-in e-signatures and payment collection
- Proposal analytics for follow-up optimization
- CRM integration for sales pipeline visibility

**Weaknesses:**
- GUI-only — no CLI or programmatic workflow
- No AI content generation built in
- No multi-step workflow beyond proposal creation
- Cannot handle non-proposal deliverables (audits, translations)
- Per-user pricing gets expensive for small teams
- No human-in-the-loop review for AI-generated content

**How Skillrunner Differs:**
- CLI-driven vs. browser-based
- Handles any deliverable type, not just proposals
- AI-powered content generation with human review
- YAML templates are version-controlled and composable
- Multi-step workflows (research -> draft -> review -> finalize)
- Flat pricing, not per-user

---

### 5. Better Proposals

| Attribute | Detail |
|-----------|--------|
| **Website** | [betterproposals.io](https://betterproposals.io) |
| **Founded** | 2016 (Brighton, UK) |
| **Team Size** | ~30 employees (estimated) |
| **Funding** | Bootstrapped |

**Core Offering:**
Lightweight, web-based proposal tool built for speed and visual polish. Targets freelancers, agencies, and small sales teams who want modern proposals without complexity.

**Key Features:**
- Customizable proposal and quote templates
- Drag-and-drop editor with modern design
- E-signatures for instant client signing
- Analytics (open/read tracking)
- Payment integrations (Stripe, PayPal, GoCardless)
- Team collaboration (Enterprise: roles, approvals, commenting)

**Pricing:**
- Starter: ~$19/mo per user
- Premium: ~$29/mo per user (custom domains, 50-document limit)
- Enterprise: ~$49/mo per user (team features, CRM integrations)

**Target Audience:**
Freelancers, small agencies, solo consultants who want fast, visually appealing proposals.

**Strengths:**
- Very fast to get started — minimal learning curve
- Beautiful default templates
- Affordable for solo operators
- Flexible month-to-month billing
- Good payment collection integrations

**Weaknesses:**
- No AI content generation
- No multi-step workflow capabilities
- GUI-only — no CLI or API-first approach
- Cannot handle complex deliverables beyond proposals
- No human-in-the-loop review stages
- Limited integrations vs. Proposify

**How Skillrunner Differs:**
- CLI-first with YAML definitions
- AI-powered content generation, not just templates
- Multi-step workflows for any deliverable type
- Human-in-the-loop review is built in
- Version-controlled workflows
- Handles proposals, audits, translations, content pipelines

---

## Category 3: Developer-Oriented Workflow Engines

### 6. Kestra

| Attribute | Detail |
|-----------|--------|
| **Website** | [kestra.io](https://kestra.io) |
| **Founded** | 2021 (La Madeleine, France) |
| **Team Size** | ~31 employees |
| **Funding** | $8M (led by Alven, with Isai participation) |
| **GitHub Stars** | ~26,343 |

**Core Offering:**
Open-source, event-driven orchestration platform using declarative YAML workflow definitions. Brings Infrastructure-as-Code principles to workflow automation.

**Key Features:**
- Declarative YAML workflow definitions with built-in code editor
- 1,200+ plugins for integrations
- Namespaces, labels, subflows for organization
- Retries, timeouts, error handling, conditional branching
- Sequential and parallel task execution
- Event triggers, advanced scheduling, backfills
- Real-time syntax validation and auto-completion

**Pricing:**
- Open Source: Free (self-hosted)
- Enterprise: Custom pricing (SSO, RBAC, audit logs)

**Target Audience:**
Data engineers, backend developers, DevOps teams running scheduled jobs, data pipelines, and API orchestration.

**Strengths:**
- YAML-first workflow definition (closest to Skillrunner's approach)
- Strong plugin ecosystem (1,200+)
- Version control-friendly by design
- Robust error handling and retry logic
- Event-driven architecture

**Weaknesses:**
- Focused on data engineering, not consultant deliverables
- No AI/LLM integration as core feature
- No human-in-the-loop approval gates
- Requires server infrastructure (not a single binary CLI)
- No consultant-specific templates or workflow patterns
- Small team and limited funding vs. competitors

**How Skillrunner Differs:**
- CLI-native single binary vs. server deployment
- AI/LLM integration is core, not a plugin
- Human-in-the-loop review stages built in
- Consultant-specific workflow patterns (proposals, audits, content)
- Simpler deployment — no infrastructure required
- Targets consultants, not data engineers

---

### 7. Windmill

| Attribute | Detail |
|-----------|--------|
| **Website** | [windmill.dev](https://www.windmill.dev) |
| **Founded** | 2022 |
| **Team Size** | ~15-25 (estimated, YC-backed startup) |
| **Funding** | YC-backed (amount not disclosed publicly) |
| **GitHub Stars** | ~15,000+ |
| **License** | AGPLv3 (open source) |

**Core Offering:**
Open-source developer platform that turns scripts into workflows, webhooks, and UIs. Claims to be the fastest workflow engine (13x faster than Airflow).

**Key Features:**
- Script execution in 20+ languages (Python, TypeScript, Go, etc.)
- Full LSP support with auto-generated UIs
- Flow editor for drag-and-drop workflow construction
- Sub-20ms execution overhead
- Managed dependencies
- Multiple triggers: webhooks, schedules, CLI, Slack, email
- Auto-generated user interfaces from script parameters

**Pricing:**
- Free: Self-hosted, limited features
- Pro: Subscription-based (exact pricing varies)
- Enterprise: Custom pricing with commercial support
- Operators priced at 1/2 developer seats

**Target Audience:**
Developers building internal tools, data pipelines, and automated workflows. Alternative to Retool and Temporal.

**Strengths:**
- Extremely fast execution engine (13x Airflow)
- Multi-language script support (20+ languages)
- Auto-generated UIs from scripts
- CLI trigger support (partial CLI story)
- Open source with self-hosting

**Weaknesses:**
- Developer tool, not consultant tool
- No AI/LLM integration as core feature
- No human-in-the-loop approval workflows
- Requires server deployment
- No consultant-specific workflow patterns
- UI-generation focus doesn't match CLI-first philosophy

**How Skillrunner Differs:**
- Pure CLI experience vs. UI-generation platform
- YAML workflow definitions vs. script-first approach
- AI/LLM integration is core functionality
- Human-in-the-loop review stages built in
- Purpose-built for consultant deliverables
- Single-binary deployment vs. server infrastructure

---

### 8. Lobster (OpenClaw)

| Attribute | Detail |
|-----------|--------|
| **Website** | [docs.openclaw.ai/tools/lobster](https://docs.openclaw.ai/tools/lobster) |
| **Founded** | 2025 (part of OpenClaw ecosystem) |
| **Team Size** | Open-source community project |
| **Funding** | N/A (community-driven) |
| **GitHub** | [github.com/openclaw/lobster](https://github.com/openclaw/lobster) |

**Core Offering:**
A typed, local-first workflow shell and macro engine for OpenClaw that runs YAML/JSON workflows with approval checkpoints. Designed to make AI agent actions deterministic and auditable.

**Key Features:**
- YAML/JSON workflow files with steps, env, conditions, approvals
- Deterministic sequential execution with JSON data flow between steps
- Approval gates — side effects halt until explicitly approved
- Resumability — halted workflows resume without re-running
- Local-first execution (no cloud dependency)
- Reduces LLM token costs by moving orchestration to typed runtime

**Pricing:**
- Free / Open Source

**Target Audience:**
OpenClaw users who want deterministic, auditable AI workflows with human approval gates.

**Strengths:**
- Closest competitor to Skillrunner's philosophy (YAML + CLI + approvals)
- Human-in-the-loop approval gates are first-class
- Local-first, no cloud dependency
- Deterministic execution reduces AI unpredictability
- Resumable workflows

**Weaknesses:**
- Tightly coupled to OpenClaw ecosystem — not standalone
- No built-in LLM provider management
- No consultant-specific workflow patterns
- Community project with uncertain long-term support
- Limited documentation and ecosystem
- No multi-provider AI support

**How Skillrunner Differs:**
- Standalone tool vs. OpenClaw-dependent
- Built-in multi-provider LLM support (Ollama, Anthropic, etc.)
- Consultant-specific workflow templates and patterns
- Broader use case coverage (proposals, audits, translations, content)
- Hexagonal architecture for extensibility
- Active commercial development vs. community project

---

## Comparison Matrix

| Feature | Skillrunner | n8n | Make | Dify | Proposify | Better Proposals | Kestra | Windmill | Lobster |
|---------|-------------|-----|------|------|-----------|-------------------|--------|----------|---------|
| **CLI-First** | Yes | No | No | No | No | No | No | Partial | Yes |
| **YAML Workflows** | Yes | No | No | No | No | No | Yes | No | Yes |
| **AI/LLM Integration** | Core | Plugin | Plugin | Core | No | No | Plugin | No | Via OpenClaw |
| **Human-in-the-Loop** | Core | No | No | No | No | No | No | No | Yes |
| **Self-Hosted** | Yes | Yes | No | Yes | No | No | Yes | Yes | Yes |
| **Open Source** | Yes | Fair-code | No | Yes | No | No | Yes | AGPLv3 | Yes |
| **Consultant Focus** | Yes | No | Partial | No | Yes | Yes | No | No | No |
| **Proposal Generation** | Yes | No | No | No | Yes | Yes | No | No | No |
| **Audit Workflows** | Yes | No | No | No | No | No | No | No | No |
| **Content Pipelines** | Yes | Partial | Partial | Partial | No | No | Partial | Partial | Partial |
| **Version Control Native** | Yes | Business plan | No | No | No | No | Yes | Partial | Yes |
| **Single Binary** | Yes | No | N/A | No | N/A | N/A | No | No | Yes |
| **Multi-Provider LLM** | Yes | Yes | Yes | Yes | No | No | No | No | No |
| **Pricing** | TBD | Free-$800/mo | Free-$99/mo | Free-Custom | $29-65/user/mo | $19-49/user/mo | Free-Custom | Free-Custom | Free |
| **Funding** | Pre-seed | $253M | $232M | $30M | $27M | Bootstrapped | $8M | YC-backed | N/A |
| **Team Size** | Small | ~787 | ~800+ | ~50-80 | ~150 | ~30 | ~31 | ~15-25 | Community |

---

## Competitive Positioning Map

```
                    Consultant-Specific
                         ^
                         |
         Proposify       |      SKILLRUNNER
         Better Proposals|      (CLI + AI + HITL)
                         |
  GUI-First  <-----------+----------->  CLI-First
                         |
         Make            |      Lobster
         n8n             |      Kestra
         Dify            |      Windmill
                         |
                    Developer/General
```

---

## Key Findings

### 1. No Direct Competitor Exists
No tool combines CLI-first operation, YAML workflows, AI-powered consultant deliverables, and human-in-the-loop review. Skillrunner's positioning is genuinely unique.

### 2. Closest Competitors by Dimension

| Dimension | Closest Competitor | Gap |
|-----------|-------------------|-----|
| YAML + CLI workflows | Lobster (OpenClaw) | Tightly coupled to OpenClaw; no standalone use |
| YAML orchestration | Kestra | Data engineering focus; no AI or HITL |
| Consultant proposals | Proposify / Better Proposals | GUI-only; no AI generation; no multi-step workflows |
| AI workflow automation | n8n / Dify | GUI-first; not consultant-specific |
| Human-in-the-loop | Lobster (OpenClaw) | Only competitor with approval gates; but ecosystem-locked |

### 3. Market Gaps Skillrunner Exploits

1. **CLI-first consultant tools do not exist.** Proposal tools are all GUI. Workflow engines with CLI support target developers, not consultants.

2. **Human-in-the-loop is desired but rarely implemented.** Content approval workflows exist in enterprise tools, but not in lightweight CLI tools for solo consultants.

3. **YAML + AI is an emerging pattern** (GitHub Agentic Workflows, Lobster) but no one targets it at consultants.

4. **The "fractional CTO" archetype is underserved.** Technical enough for CLI, needs consultant deliverables. Too small for enterprise tools, too technical for GUI-only proposal software.

### 4. Competitive Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| n8n adds CLI mode + consultant templates | Medium | Move fast; build domain expertise moat |
| Lobster becomes standalone | Medium | Skillrunner's multi-provider LLM support is broader |
| Kestra adds AI/HITL features | Low | Different target audience (data engineers vs. consultants) |
| Proposify adds AI generation | Medium | They lack workflow engine; Skillrunner is more composable |
| New entrant copies the model | Medium | First-mover advantage; community + templates library |

### 5. Funding Landscape Context

The AI workflow space is heavily funded:
- n8n: $253M raised, $2.5B valuation
- Make: $232M raised
- Dify: $30M raised (just this month)
- Kestra: $8M raised

Skillrunner enters at pre-seed, competing on niche focus rather than capital. The well-funded competitors are all going horizontal (more integrations, more use cases). Skillrunner's vertical strategy (consultants + non-profits) avoids direct competition with these giants.

---

## Sources

- [n8n Pricing](https://n8n.io/pricing/)
- [n8n Series C Announcement](https://blog.n8n.io/series-c/)
- [n8n Revenue & Team Size](https://getlatka.com/companies/n8nio)
- [Make.com Pricing](https://www.make.com/en/pricing)
- [Make.com Review 2026](https://hackceleration.com/make-review/)
- [Dify Pricing](https://dify.ai/pricing)
- [Dify $30M Funding - VentureBeat](https://venturebeat.com/business/dify-raises-30-million-series-pre-a-to-power-enterprise-grade-agentic-workflows)
- [Proposify Pricing](https://www.proposify.com/blog/better-proposals-pricing)
- [Better Proposals Pricing](https://betterproposals.io/pricing)
- [Kestra.io](https://kestra.io/)
- [Kestra Funding - TradedVC](https://traded.co/vc/deal/kestra-secures-8-million-funding-round-led-by-alven-with-participation-from-isai-and-axeleo/)
- [Windmill.dev](https://www.windmill.dev/)
- [Windmill GitHub](https://github.com/windmill-labs/windmill)
- [Lobster - OpenClaw Docs](https://docs.openclaw.ai/tools/lobster)
- [Lobster GitHub](https://github.com/openclaw/lobster)
- [Kestra vs n8n Comparison](https://openalternative.co/compare/kestra/vs/n8n)
- [n8n vs Dify Comparison](https://www.gptbots.ai/blog/n8n-vs-dify)
- [Activepieces Open Source](https://www.activepieces.com/pricing)
- [AI Workflow Tools 2026 - Slack Blog](https://slack.com/blog/productivity/9-best-ai-automation-tools-to-automate-tasks-and-streamline-workflows)
- [GitHub Agentic Workflows](https://github.blog/ai-and-ml/automate-repository-tasks-with-github-agentic-workflows/)
- [Lindy AI Pricing](https://www.lindy.ai/pricing)
- [SOC 2 Automation 2026 - DSALTA](https://www.dsalta.com/resources/soc-2/soc-2-automation-ai-compliance)
- [Content Approval Workflows 2026 - InfluenceFlow](https://influenceflow.io/resources/content-approval-workflows-a-complete-guide-for-2026-1/)
