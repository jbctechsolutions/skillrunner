# SkillRunner Release Roadmap: v1.0.0 through Q2 2026

**Document Version:** 1.1
**Created:** December 3, 2025
**Last Updated:** December 3, 2025
**Prepared for:** JBC Tech Solutions Product Planning

---

## Overview

This document outlines the release plan for SkillRunner from the initial v1.0.0 launch through Q2 2026. Each release is organized with clear versioning, feature sets, and dependencies.

**Timeline Note:** This roadmap is calibrated for a solo developer with AI-assisted development (Claude Code, SkillRunner dogfooding). Buffer time has been added for testing, documentation, and community feedback cycles.

### Release Cadence

| Release Type | Frequency | Example |
|--------------|-----------|---------|
| Major (x.0.0) | Quarterly | v1.0.0, v2.0.0 |
| Minor (1.x.0) | 4-6 weeks | v1.1.0, v1.2.0 |
| Patch (1.0.x) | As needed | v1.0.1, v1.0.2 |

---

## Release Timeline Summary

```
Dec 2025    Jan 2026         Mar 2026         Apr 2026         May 2026         Jun 2026         Jul 2026
    |           |                |                |                |                |                |
 v1.0.0      v1.1.0           v1.2.0           v1.3.0           v1.4.0           v1.5.0           v2.0.0
  Launch   Providers+       Skill Decomp     MCP + Chat +      Enterprise      Desktop GUI +    Web Platform +
           Streaming         Core             Security          Prep            Cloud Sync       Enterprise
```

### Product Evolution

```
v1.0-v1.2: CLI Foundation          v1.3-v1.4: Power Features       v1.5-v2.0: Platform
─────────────────────────          ─────────────────────────       ─────────────────────
• Core orchestration               • sr chat (workflow discovery)  • Desktop GUI (free)
• Cost tracking                    • MCP integration               • Cloud sync (Pro)
• Skill decomposition              • Security hardening            • Web platform (Team)
• Multi-provider                   • Enterprise prep               • On-prem (Enterprise)
```

**Timeline Adjustment:** +4 weeks buffer added for solo developer + AI tooling workflow. December launch stays on track.

---

## v1.0.0-beta (December 1, 2025) - SHIPPED

**Theme:** Foundation Release

### Features Included
- [x] Core orchestration engine with DAG execution
- [x] Ollama provider (production ready)
- [x] Anthropic provider (production ready)
- [x] Profile-based routing (cheap/balanced/premium)
- [x] Cost tracking with cloud-equivalent comparison
- [x] Skill marketplace with multi-source import
- [x] Bidirectional format conversion (Markdown ↔ YAML)
- [x] Homebrew distribution

### Known Limitations
- 2 production providers only
- No MCP support
- API keys stored in plaintext
- 37% test coverage

---

## v1.0.0 (December 7-13, 2025) - PUBLIC LAUNCH

**Theme:** Community Launch

### Launch Schedule
| Date | Event |
|------|-------|
| Dec 7 | Reddit soft launch (r/LocalLLaMA, r/ollama, r/selfhosted) |
| Dec 13 | Hacker News "Show HN" |

### Features (Same as Beta)
- All v1.0.0-beta features
- Bug fixes from beta feedback

### Success Metrics
| Metric | Target |
|--------|--------|
| GitHub Stars | 500+ |
| Discord Members | 150+ |
| Install Issues | <5 |

### Linear Issues
- JBC-690: Reddit launch preparation
- JBC-693: HN post preparation

---

## v1.1.0 (January 24, 2026)

**Theme:** Provider Expansion & Streaming

### Features

#### New Providers
| Provider | Issue | Priority |
|----------|-------|----------|
| OpenAI | JBC-680 | High |
| Groq | JBC-681 | High |

#### Streaming Improvements
| Feature | Issue | Priority |
|---------|-------|----------|
| Anthropic real-time streaming | JBC-692 | High |
| OpenAI streaming | JBC-680 | High |
| Groq streaming | JBC-681 | High |

#### Quality Improvements
| Feature | Issue | Priority |
|---------|-------|----------|
| Test coverage to 50% | JBC-659 | Medium |
| Error message improvements | - | Medium |

### Marketing Message
"Now supporting 4+ providers: Ollama, Anthropic, OpenAI, and Groq"

### Dependencies
- OpenAI SDK integration
- Groq API integration

---

## v1.2.0 (March 7, 2026)

**Theme:** Intelligent Skill Decomposition - Core

### Features

#### Skill Analysis Engine (JBC-702 Epic)
| Feature | Issue | Priority |
|---------|-------|----------|
| Skill Complexity Analyzer | JBC-703 | High |
| Task Type Classifier | JBC-704 | High |
| Dependency Graph Builder | JBC-705 | High |

#### Basic Decomposition
| Feature | Issue | Priority |
|---------|-------|----------|
| Sequential Decomposer | JBC-706 | High |
| Automatic Model Selector | JBC-708 | High |

#### CLI Commands
| Feature | Issue | Priority |
|---------|-------|----------|
| `sr skill analyze` | JBC-709 | High |
| `sr skill convert --optimize` (basic) | JBC-710 | High |

### Marketing Message
"Import any skill, we analyze and optimize it automatically"

### Example Output
```
$ sr skill analyze my-skill.yaml

Complexity Score: 0.78 (High - good decomposition target)
Identified Stages: 4
Parallelization Opportunity: Stages 1-2 can run in parallel
Estimated Cost Savings: 75% ($0.12 → $0.03)
```

### Dependencies
- v1.1.0 provider expansion (needed for multi-model routing)

---

## v1.3.0 (April 4, 2026)

**Theme:** MCP + Security + Interactive Chat

### Features

#### `sr chat` - Interactive Workflow Discovery
| Feature | Description | Priority |
|---------|-------------|----------|
| Chat mode | `sr chat` launches interactive session | High |
| Workflow suggestion | AI suggests optimized workflows from conversation | High |
| Cost preview | Show estimated cost before execution | High |
| Inline execution | Run suggested workflows from chat | Medium |

**Example Flow:**
```bash
$ sr chat

You: Analyze this codebase for security issues

SkillRunner: I'll create a multi-phase workflow:
  Phase 1: File discovery (cheap) - find relevant files
  Phase 2: Pattern scan (cheap) - identify potential issues
  Phase 3: Deep analysis (balanced) - analyze flagged code
  Phase 4: Report (cheap) - generate findings

  Estimated cost: $0.08 (vs $0.45 single-phase)

  [Run] [Save as skill] [Modify] [Cancel]
```

#### MCP Protocol Support (JBC-691)
| Feature | Description | Priority |
|---------|-------------|----------|
| MCP Client | Connect to MCP servers | High |
| Database MCP | Query data in workflows | High |
| GitHub MCP | PR automation | Medium |
| File System MCP | Local file operations | Medium |

#### Advanced Decomposition
| Feature | Issue | Priority |
|---------|-------|----------|
| DAG Decomposer | JBC-707 | High |
| Parallel execution support | JBC-707 | High |

#### Security Hardening
| Feature | Issue | Priority |
|---------|-------|----------|
| API Key Encryption | JBC-650 | Critical |
| Secrets filtering in logs | JBC-654 | High |

#### Quality
| Feature | Issue | Priority |
|---------|-------|----------|
| Test coverage to 60% | JBC-659 | Medium |
| Quality Validation Framework | JBC-711 | Medium |

### Marketing Message
"Chat to discover workflows. Connect to 1000+ MCP servers. Enterprise-ready security."

### Dependencies
- v1.2.0 skill decomposition core
- MCP SDK integration

---

## v1.4.0 (May 2, 2026)

**Theme:** Enterprise Preparation

### Features

#### Enterprise Security
| Feature | Description | Priority |
|---------|-------------|----------|
| Audit logging | Track all executions | High |
| RBAC foundation | Role-based access | Medium |
| SSO/SAML preparation | Enterprise auth | Medium |

#### Skill Decomposition Polish
| Feature | Issue | Priority |
|---------|-------|----------|
| Benchmark Comparison Tool | JBC-712 | Medium |
| Quality threshold enforcement | JBC-711 | Medium |
| Decomposition preview mode | - | Medium |

#### CI/CD Integration
| Feature | Description | Priority |
|---------|-------------|----------|
| GitHub Action v1 | Automated workflows in CI | High |
| GitLab CI template | GitLab integration | Medium |

#### Quality
| Feature | Issue | Priority |
|---------|-------|----------|
| Test coverage to 70% | JBC-659 | Medium |

### Marketing Message
"Enterprise-ready: audit logs, SSO, CI/CD integration"

### Dependencies
- v1.3.0 security foundation

---

## v1.5.0 (June 6, 2026)

**Theme:** Desktop GUI + Cloud Sync (Beta)

### Features

#### Desktop Application (FREE - Tauri/Wails)
| Feature | Description | Priority |
|---------|-------------|----------|
| Visual workflow builder | Drag-drop phase editor | High |
| Cost dashboard | Local execution history, charts | High |
| Skill browser | Browse/import from marketplace | High |
| Execution viewer | Real-time workflow monitoring | High |
| Settings UI | Provider config, profiles | Medium |

#### Cloud Sync (Pro Tier)
| Feature | Description | Priority |
|---------|-------------|----------|
| User accounts | Registration, OAuth | High |
| Cross-device sync | Workflows sync across machines | High |
| Skill hosting | Publish skills to cloud | High |
| Historical analytics | Long-term cost trends | Medium |

#### CLI Integration
| Feature | Description | Priority |
|---------|-------------|----------|
| `sr cloud login` | Authenticate to cloud | High |
| `sr cloud publish` | Publish skills to cloud | High |
| `sr cloud sync` | Sync local/cloud skills | Medium |

### Pricing Tiers (Beta)
| Tier | Price | Desktop | Cloud |
|------|-------|---------|-------|
| Community | Free | Full GUI | - |
| Pro | $19/mo | Full GUI | Sync + Analytics |

### Marketing Message
"SkillRunner Desktop: Visual workflow builder with cost tracking - free forever. Add cloud sync for $19/mo."

### Tech Stack Decision
| Option | Recommendation |
|--------|----------------|
| **Tauri** | Preferred - Rust + Web, ~5MB binary, fits "single binary" brand |
| Wails | Alternative - Go-native, good fit |
| Electron | Avoid - 150MB+, resource heavy |

### Dependencies
- v1.4.0 enterprise features
- Desktop app development (new skillset)

---

## v2.0.0 (July 11, 2026)

**Theme:** Web Platform + Enterprise + Marketplace

### Features

#### Web Platform (Team/Enterprise Tiers)
| Feature | Description | Priority |
|---------|-------------|----------|
| Web-based chat | Conversational workflow discovery | High |
| Web workflow builder | Browser-based visual editor | High |
| Real-time collaboration | Multiple users editing | Medium |
| Execution dashboard | Team-wide monitoring | High |

#### Team Tier ($49/user/mo)
| Feature | Description | Priority |
|---------|-------------|----------|
| Web platform access | Full browser-based experience | High |
| Team workspaces | Shared skill libraries | High |
| SSO/SAML | Enterprise auth | High |
| Audit logging | Compliance tracking | High |
| Cost allocation | Per-team/project spending | Medium |

#### Enterprise Tier (Custom Pricing)
| Feature | Description | Priority |
|---------|-------------|----------|
| **Dedicated cloud tenant** | Isolated infrastructure | Critical |
| **On-premise deployment** | Air-gapped, self-hosted | High |
| Private skill registry | Internal-only marketplace | High |
| Custom SLA | 99.9%+ uptime guarantee | High |
| Priority support | Dedicated account manager | High |
| Custom integrations | API + webhook customization | Medium |

#### Public Skill Marketplace
| Feature | Description | Priority |
|---------|-------------|----------|
| Browse skills | Public skill directory | High |
| Ratings & reviews | Community feedback | Medium |
| Verified publishers | Trust indicators | Medium |
| Revenue sharing | Monetize skills (70/30 split) | Medium |

#### MCP Server Mode
| Feature | Description | Priority |
|---------|-------------|----------|
| MCP Server | Expose skills as MCP tools | High |
| Claude Desktop integration | Trigger skills from Claude | High |

### Deployment Options

```
┌─────────────────────────────────────────────────────────────────┐
│  COMMUNITY (Free)        │  PRO ($19/mo)                        │
│  ─────────────────────   │  ─────────────────────               │
│  CLI + Desktop           │  CLI + Desktop + Cloud Sync          │
│  Local execution only    │  Cross-device, analytics             │
├─────────────────────────────────────────────────────────────────┤
│  TEAM ($49/user/mo)      │  ENTERPRISE (Custom)                 │
│  ─────────────────────   │  ─────────────────────               │
│  + Web Platform          │  + Dedicated Tenant                  │
│  + Team collaboration    │  + On-Premise Option                 │
│  + SSO/SAML              │  + Air-gapped deployment             │
│  + Audit logs            │  + Custom SLA (99.9%+)               │
│  Shared cloud infra      │  + Priority support                  │
└─────────────────────────────────────────────────────────────────┘
```

### Privacy Architecture

| Tier | Data Location | Isolation |
|------|---------------|-----------|
| Community | Local only | N/A |
| Pro | Local + shared cloud | Logical isolation |
| Team | Local + shared cloud | Logical isolation + encryption |
| Enterprise | **Dedicated tenant OR on-prem** | **Physical isolation** |

### Marketing Messages

| Tier | Message |
|------|---------|
| Community | "Free forever. Your data stays local." |
| Pro | "Sync across devices. See your cost trends." |
| Team | "Collaborate on workflows. Enterprise auth included." |
| Enterprise | "Your infrastructure. Your data. Your rules." |

### Dependencies
- v1.5.0 desktop + cloud sync
- Payment infrastructure (Stripe)
- Multi-tenant architecture
- On-prem deployment tooling (Helm charts, Docker Compose)
- Legal (ToS, privacy policy, DPA for enterprise)

---

## Feature Dependency Graph

```
v1.0.0 (Foundation)
    │
    ├── v1.1.0 (Providers)
    │       │
    │       └── v1.2.0 (Skill Decomp Core)
    │               │
    │               ├── v1.3.0 (MCP + Security)
    │               │       │
    │               │       └── v1.4.0 (Enterprise Prep)
    │               │               │
    │               │               └── v1.5.0 (Cloud MVP)
    │               │                       │
    │               │                       └── v2.0.0 (Cloud GA)
    │               │
    │               └── (Decomposition features continue through v1.3.0)
    │
    └── Security track (JBC-650) → Required before v1.4.0
```

---

## Linear Issue Mapping

### v1.0.0 Issues
- JBC-690: Reddit launch
- JBC-693: HN launch

### v1.1.0 Issues
- JBC-680: OpenAI provider
- JBC-681: Groq provider
- JBC-692: Streaming verification
- JBC-659: Test coverage (phase 1)

### v1.2.0 Issues (Skill Decomposition Epic: JBC-702)
- JBC-703: Skill Complexity Analyzer
- JBC-704: Task Type Classifier
- JBC-705: Dependency Graph Builder
- JBC-706: Sequential Decomposer
- JBC-708: Automatic Model Selector
- JBC-709: `sr skill analyze` command
- JBC-710: `sr skill convert --optimize` command

### v1.3.0 Issues
- JBC-691: MCP Protocol Support
- JBC-650: API Key Encryption
- JBC-654: Secrets in logs
- JBC-707: DAG Decomposer
- JBC-711: Quality Validation Framework
- JBC-659: Test coverage (phase 2)

### v1.4.0 Issues
- JBC-712: Benchmark Comparison Tool
- JBC-659: Test coverage (phase 3)
- New: Audit logging
- New: GitHub Action v1
- New: GitLab CI template

### v1.5.0 Issues (New - Cloud)
- New: Cloud platform MVP
- New: User accounts
- New: Skill hosting
- New: Cost dashboard
- New: Team workspaces

### v2.0.0 Issues (New - Cloud GA)
- New: Payment integration
- New: Public marketplace
- New: MCP Server mode
- New: Enterprise SSO
- New: On-premise deployment

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Skill decomposition complexity | High | Start with sequential, add DAG later |
| MCP ecosystem changes | Medium | Track MCP spec closely |
| Cloud infrastructure delays | High | Launch CLI features independently |
| Security vulnerabilities | Critical | Complete JBC-650 before v1.4.0 |
| Provider API changes | Medium | Abstract provider layer |

---

## Success Metrics by Release

| Release | Primary Metric | Target |
|---------|---------------|--------|
| v1.0.0 | GitHub Stars | 500+ |
| v1.1.0 | Weekly Active Users | 500+ |
| v1.2.0 | Skills optimized | 1,000+ |
| v1.3.0 | MCP integrations used | 500+ |
| v1.4.0 | CI/CD integrations | 100+ |
| v1.5.0 | Cloud beta users | 200+ |
| v2.0.0 | Paid customers | 200+ |

---

## Resource Requirements

### Solo Developer + AI Tooling
| Phase | Focus | Estimated Effort | AI-Assisted Approach |
|-------|-------|------------------|---------------------|
| v1.0.0-v1.1.0 | Provider expansion | 6-8 weeks | Claude Code for boilerplate, SkillRunner for code review |
| v1.2.0-v1.3.0 | Skill decomposition + MCP | 8-10 weeks | Design with AI, iterate on heuristics |
| v1.4.0 | Enterprise features | 4-5 weeks | Security review with AI assistance |
| v1.5.0-v2.0.0 | Cloud platform | 10-14 weeks | Infrastructure automation, AI-driven testing |

### Marketing (Self-Service)
| Phase | Focus | Effort | Channel |
|-------|-------|--------|---------|
| v1.0.0 | Community launch | 2 weeks | Reddit, HN, Discord |
| v1.2.0 | Skill decomposition launch | 1 week | Blog post, demo video |
| v1.5.0 | Cloud beta announcement | 2 weeks | Email list, social |
| v2.0.0 | Cloud GA + marketplace launch | 4 weeks | Product Hunt, press |

### Dogfooding Strategy
SkillRunner will be used to build itself:
- Code review skills for PR automation
- Documentation generation workflows
- Test generation and validation
- Cost tracking for development API usage

---

## Decision Points

### Before v1.2.0
- [ ] Confirm skill decomposition is core differentiator
- [ ] Finalize analysis heuristics

### Before v1.3.0
- [ ] MCP spec stability assessment
- [ ] Security audit completion

### Before v1.5.0
- [ ] Cloud infrastructure selection (AWS/GCP/DO)
- [ ] Pricing validation with early users

### Before v2.0.0
- [ ] Payment provider selection (Stripe recommended)
- [ ] Marketplace economics (revenue share %)
- [ ] Legal review (ToS, privacy, GDPR)

---

*Document prepared by Product Team | December 3, 2025*
*v1.1: Adjusted timeline for solo developer + AI tooling workflow*
