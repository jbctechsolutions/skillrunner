# Target Audience: Skillrunner

**Synthesized:** 2026-03-09
**Sources:** Customer voice interviews, demand signals, distribution research, market sizing
**Position:** "Ansible for AI workflows" -- CLI workflow engine deployed by fractional CTOs via Open WebUI

---

## 1. Primary Persona: Marcus, the Fractional CTO

**Name:** Marcus Chen
**Age:** 38
**Title:** Fractional CTO / Independent Technology Consultant
**Clients:** 3-5 simultaneous engagements, mostly small organizations (churches, non-profits, bootstrapped SaaS)
**Income:** $180K-$350K/year ($200-$400/hr, 20-30 billable hrs/week)
**Location:** Mid-size US metro, works remote
**Technical depth:** Lives in the terminal. Comfortable with YAML, Git, Docker. Runs Ollama locally.

**Day in the life:** Marcus starts his morning triaging Slack messages from three clients. One church needs their volunteer scheduling workflow fixed -- again. A non-profit wants a donor thank-you email sequence, and Marcus opens ChatGPT, pastes in the org's context (for the fourth time this month), tweaks the output, copies it into a Google Doc, and emails it. By 10am he's done 90 minutes of work he's done before, for a different client, last month. His intellectual property -- the frameworks, the templates, the checklists that make him effective -- is scattered across client Google Drives, Notion workspaces, and email drafts. Half of it contradicts the other half.

**Core frustration:**
> "Every consulting project generates intellectual property -- templates, frameworks, checklists, and documentation structures. Before systematizing this, that IP disappears into client folders."

**What Marcus wants:**
- Repeatable engagement toolkits he can deploy across clients without rebuilding from scratch
- A way to hand off automated workflows to non-technical client staff (church admins, volunteer coordinators)
- Version-controlled workflows he owns, not locked inside a SaaS vendor
- Self-hosted deployments so sensitive client data stays under his control

**Budget psychology:** Marcus bills $200-$500/hr. A $49/month tool that saves him one hour per month delivers 4-10x ROI. Tool purchases under $100/month don't require client approval -- they're a rounding error against his $3K-$15K/month engagement retainers.

**Decision style:** Technical, skeptical of marketing. Evaluates tools by reading the source code, checking the GitHub repo, and running it locally before committing. Prefers open-source with optional paid tiers. Will not tolerate opaque pricing.

**Where he hangs out:** CTO Craft Slack (18,000+ members), fractionalctos.org, Hacker News, r/selfhosted, r/LocalLLaMA, LinkedIn (the "fractional CTO" title grew from 2,000 to 110,000 profiles between 2022-2026).

**Why he matters:** One fractional CTO who adopts Skillrunner deploys it across 3-5 clients. Marcus is not just one customer -- he is a distribution channel. The fractional CTO market doubled from 60,000 to 120,000 between 2022 and 2024, and 25-35% of US businesses now use fractional hiring.

**Representative quote:**
> "Fractional CTOs who package their expertise into standardized service tiers report 42 percent higher gross margins and shave an average of 11 billable hours off each new engagement by productizing knowledge into clearly priced service tiers and reusable assets." -- Umbrex Fractional CTO Playbook

---

## 2. Secondary Persona: Diane, the Church Operations Director

**Name:** Diane Ramirez
**Age:** 52
**Title:** Director of Operations / Church Administrator
**Org:** Mid-size church (300-800 weekly attendance), budget ~$300K-$800K/year
**Technical depth:** Proficient with Office 365, Planning Center, and basic spreadsheets. Not comfortable with code, APIs, or CLIs.
**Staff:** 3-5 paid staff, 40-80 active volunteers

**Day in the life:** Diane arrives at 7:30am and immediately starts putting out fires. The email newsletter needs to go out but the volunteer who usually formats it is unavailable. The pastor wants a summary of giving trends for the board meeting -- she opens three different tools to piece together the data. She spends 23 hours/week on tasks that could be automated, but the "automation" tools she's tried (Zapier, n8n) require technical knowledge she doesn't have.

**Core frustration:**
> "Most churches rely on a small number of volunteers or one overworked IT staff member, and when something breaks midweek or during events, there's no one available to help."

**What Diane wants:**
- "One push button" simplicity -- workflows she can trigger without understanding what's behind them
- Tools that volunteers can use after a 5-minute explanation
- Something that costs less than the $299/month Zapier plan her church tried and abandoned
- Technology that supports the mission instead of distracting from it

**Budget psychology:** Extremely price-sensitive. Every dollar spent on technology competes with mission spending. The church board must approve purchases, adding 2-4 weeks to any buying decision. But Diane is value-responsive: she will fight for budget if a tool demonstrably saves staff time or prevents volunteer burnout.

**Why she matters as a secondary persona:** Diane is NOT the buyer of Skillrunner. She is the end user. Marcus (the fractional CTO) buys and deploys Skillrunner; Diane interacts with it through Open WebUI. Understanding Diane's constraints -- her technical ceiling, her budget sensitivity, her need for simplicity -- directly shapes what Marcus needs from Skillrunner's deployment capabilities. If Diane can't use it, Marcus can't sell it.

**Representative quote:**
> "Automated functions are ideal, allowing volunteers to run complicated operations with one push button rather than doing complicated keystrokes." -- FOR-A Forum, church tech training webinar

---

## 3. Anti-Persona: Who NOT to Target

### Enterprise DevOps Engineers
Engineers at companies with 500+ employees who already have Airflow, Prefect, Temporal, or internal workflow orchestration. They have dedicated platform teams, established toolchains, and procurement processes that take 6-12 months. Skillrunner's value proposition (repeatable consulting workflows, Open WebUI handoff) is irrelevant to their world. Chasing enterprise deals would consume all go-to-market energy with near-zero conversion.

### Non-Technical Solo Consultants (Marketing, HR, Finance)
Business consultants who work entirely in Google Docs and Zoom. They will never open a terminal. Even if Skillrunner added a GUI, these users are better served by Zapier or Make. Targeting them dilutes messaging and creates support burden with users who can't self-serve.

### AI Researchers and ML Engineers
People building models, training datasets, or running experiments. They need MLflow, Weights & Biases, or Jupyter -- not a workflow engine for business process automation. They would evaluate Skillrunner against tools it's not competing with and be disappointed.

### Churches Without a Technology Consultant
Churches that buy their own software directly (no consultant in the loop) need turnkey SaaS products like Planning Center, Pushpay, or ChurchTrac. They lack the technical capacity to deploy or maintain Skillrunner, even through Open WebUI. Selling directly to churches bypasses the fractional CTO channel and creates unsustainable support obligations.

---

## 4. Customer Pain Hierarchy (Ranked by Frequency x Intensity)

| Rank | Pain | Frequency | Intensity | Segment | Current Workaround |
|------|------|-----------|-----------|---------|--------------------|
| 1 | **IP leaks into client folders; every engagement starts from scratch** | Every engagement | Critical (existential for scaling) | Fractional CTOs | Scattered Google Docs, Notion, email drafts -- often contradictory versions |
| 2 | **AI copy-paste loop: re-uploading context, re-explaining preferences every session** | Daily | Critical (hours wasted) | All consultants | Manual ChatGPT/Claude sessions; "good enough" but not scalable |
| 3 | **No-code tools too complex for non-technical end users** | Every client handoff | Severe (creates permanent dependency) | Fractional CTOs + Church staff | Consultant builds workflows; clients can't maintain them; everyone goes back to manual |
| 4 | **Automation pricing kills small org budgets** | Every budget cycle | Severe (hard budget constraint) | Churches, non-profits | Free tiers with crippling limits, or abandon automation |
| 5 | **Tool sprawl: 6+ disconnected tools, 2 hrs/day lost** | Daily | Severe (burnout driver) | Church operations staff | One overworked person holds it all together |
| 6 | **AI hallucination -- no structured review process for generated content** | Every AI-generated deliverable | Moderate-Severe (trust deficit blocks adoption) | All consultants | Manual review of everything, or avoid AI entirely |
| 7 | **Data privacy concerns with cloud AI** | Per engagement (especially regulated clients) | Moderate (compliance risk) | Fractional CTOs with sensitive clients | Use cloud AI anyway and hope, or avoid AI |

---

## 5. Jobs-to-be-Done Framework

### Functional Jobs
- **When** I onboard a new client, **I want to** deploy a proven set of workflows in under an hour, **so I can** start delivering value immediately instead of spending the first two weeks setting up tooling.
- **When** I build an AI-assisted workflow, **I want to** define it in version-controlled YAML, **so I can** reuse it across clients, track changes, and avoid losing my intellectual property.
- **When** I hand off a system to a non-technical client team, **I want to** give them a simple web interface, **so they can** trigger workflows without calling me for help.
- **When** AI generates a proposal or communication, **I want** a human review checkpoint built into the workflow, **so I can** catch hallucinations before they reach the client's audience.

### Social Jobs
- **I want to** be seen by my clients as someone who brings modern, sophisticated solutions, **not** someone who just does what they could do themselves with ChatGPT.
- **I want to** demonstrate to peer fractional CTOs that I've productized my practice, **so they** see me as someone who has figured out the scaling problem.
- **I want to** show church leadership that AI can serve the mission without being scary or unethical, **so they** trust my technology recommendations.

### Emotional Jobs
- **I want to** stop feeling like I'm reinventing the wheel with every engagement. The phrase they use: "the goal is to systematize excellence, not reinvent the wheel every time."
- **I want to** stop worrying that my client's sensitive data is being fed into training sets. "Many public AI models use your input as future training data."
- **I want to** feel confident that when I hand something off, it will keep working. The current reality: "when something breaks midweek or during events, there's no one available to help."

---

## 6. Language Map: Exact Words Customers Use

### Describing the Problem (Use These in Marketing Copy)
| Their Words | What They Mean | Where They Say It |
|-------------|----------------|-------------------|
| "reinventing the wheel" | Doing the same setup work for every client | LinkedIn, consulting blogs |
| "copy-paste" | Manual transfer of context between AI sessions | Lenny's Newsletter, HN |
| "start from scratch" | No reusable templates or workflows | Consulting forums |
| "disappears into client folders" | Lost intellectual property | Consulting workflow analyses |
| "steep learning curve" | Non-technical users can't use the tool | G2 reviews, n8n discussions |
| "one overworked person" | Single point of failure in church/nonprofit IT | Church IT blogs |
| "yelling into the void" | Frustration with complex tool setup | Lindy.ai n8n review |
| "every single time" | Repetitive manual AI interactions | Lenny's Newsletter |
| "tool sprawl" | Too many disconnected applications | Zapier surveys |
| "costs can skyrocket" | Anxiety about per-task/per-execution pricing | G2 reviews, comparison sites |
| "can't hand off" | Workflows require the consultant to operate | Church IT forums |
| "salvaging AI-generated content" | Fixing hallucinated/incorrect AI output | Loopio analysis |

### Describing What They Want (Use These in Product Positioning)
| Their Words | What They Mean | Positioning Implication |
|-------------|----------------|------------------------|
| "repeatable" | Consistent, reusable processes | Lead with repeatability in all messaging |
| "systemize" / "productize" | Turn ad hoc consulting into packaged services | Frame Skillrunner as a productization engine |
| "one push button" | End-user simplicity | Emphasize Open WebUI as the "one button" layer |
| "human in the loop" | AI + human judgment at checkpoints | Highlight review phases as a core feature |
| "self-hosted" / "local" / "private" | Data sovereignty, no cloud dependency | Lead with self-hosted in security-conscious channels |
| "version control" | Track changes, roll back, audit | YAML-in-git is the killer feature for this audience |
| "hand off" | Transfer operation to non-technical staff | The Open WebUI deployment story |
| "standardized service tiers" | Packaged consulting offerings | Skillrunner enables tiered service delivery |

### Competitor Language (Negative -- Use for Differentiation)
| Their Words About Competitors | Competitor | Skillrunner Counter |
|-------------------------------|------------|---------------------|
| "IKEA couch on five hours of sleep" | n8n | "YAML you can read in 30 seconds" |
| "task-based pricing" | Zapier | "Self-hosted, no per-task cost" |
| "no version control" | Zapier | "Every workflow is a Git commit" |
| "bottleneck with IT teams" | n8n | "Deploy once, run from Open WebUI" |
| "basic" (AI features) | Zapier | "Built for AI workflows from day one" |

---

## 7. Buying Behavior

### Decision Process
1. **Trigger:** Marcus hits the wall -- a new client engagement that requires rebuilding workflows he's already built twice before. Or: a client's Zapier bill crosses $200/month and the board pushes back.
2. **Research:** GitHub repo first. README, architecture, license. Then: "Is it Go? Is it YAML? Does it self-host?" Marcus evaluates tools by running them locally, not reading landing pages.
3. **Trial:** Downloads the free/community tier. Builds one real workflow for one real client. If it works and the client team can use the Open WebUI interface, he's hooked.
4. **Adoption:** Deploys across 2-3 more clients. Converts to paid when he hits limits on client deployments or wants team features.
5. **Expansion:** Shares with CTO Craft or fractionalctos.org peers. "Here's what I'm using" posts drive the next wave.

### Decision Criteria (Ranked)
1. **Works locally / self-hosted** -- non-negotiable for data-sensitive clients
2. **YAML-defined, version-controllable** -- must fit into Git-based workflow
3. **CLI-first** -- matches how Marcus already works
4. **Open-source core** -- Marcus will not adopt a tool he can't inspect, fork, or escape from
5. **Client-facing interface** -- Open WebUI integration must be seamless for Diane
6. **Price** -- must be under $100/month for solo; under $200/month for team

### Buying Cycle
- **Solo fractional CTO:** 1-2 weeks from discovery to free tier adoption; 1-3 months to paid conversion
- **Church/non-profit (through the consultant):** 2-6 weeks for the consultant to evaluate; 2-4 additional weeks for church board approval if cost is passed through
- **Fractional CTO firm (2-5 consultants):** 1-2 months evaluation; needs one internal champion

### Common Objections
| Objection | Response |
|-----------|----------|
| "I can just use ChatGPT/Claude directly" | You can. And you'll re-upload context and re-explain preferences every single time. Skillrunner saves the workflow so you never start from scratch. |
| "My clients can't use a CLI tool" | They won't. They use Open WebUI -- a chat interface. You build the workflows; they push one button. |
| "I don't want vendor lock-in" | It's open-source YAML. Your workflows are text files in a Git repo. Walk away anytime. |
| "n8n/Zapier already does this" | n8n requires JSON, API knowledge, and ongoing technical maintenance your client can't provide. Zapier costs $299/month to be useful and has no version control. |
| "AI-generated content can't be trusted" | That's why Skillrunner has human-in-the-loop review phases built into every workflow. AI drafts; humans approve. |

---

## 8. Where to Reach Them (Ranked Channels)

| Rank | Channel | Density of Target Buyers | Cost | Signal Quality | Action |
|------|---------|--------------------------|------|----------------|--------|
| 1 | **Existing client referrals** | Very high (warm intros) | $0 | Highest | Ask for intros to peer churches/orgs and their consultants |
| 2 | **Fractional CTO communities** (CTO Craft, fractionalctos.org) | High (18,000+ members in CTO Craft alone) | $0 | High | Share deployment playbooks, not pitches |
| 3 | **Open WebUI Community Marketplace** | High (347,000 users) | $0 (dev time only) | Medium-High | Publish Skillrunner integration tools |
| 4 | **Church IT Network Conference** (Oct, Louisville, KY) | Very high (500 church IT decision-makers) | $85 registration | Very high | Present case study, demo on laptop |
| 5 | **MCP Server Marketplaces** (mcp.so, Cline, Claude Code) | Medium (18,320 servers on mcp.so) | $0 | Medium | List Office 365 MCP server with Skillrunner references |
| 6 | **LinkedIn organic** | Medium (110,000 "fractional" profiles) | $0 | Medium | Post deployment case studies |
| 7 | **Hacker News (Show HN)** | Medium (developer audience) | $0 | Medium (weak activation) | Frame as "Ansible for AI Workflows (Go, local-first)" |
| 8 | **Reddit** (r/selfhosted, r/LocalLLaMA, r/msp, r/churchtechnology) | Low-Medium | $0 | Medium | Answer questions, demonstrate expertise |
| 9 | **SEO / Blog content** | Low (compounds over 12-14 months) | $0 (time) | Low initially, high long-term | Target "church AI automation," "fractional CTO AI deployment" |

**Channels to avoid:**
- Paid ads (zero budget, audience too niche for broad targeting)
- Product Hunt (good for awareness, poor for this specific niche)
- Large church conferences like CFX (expensive to exhibit, production-focused audience)

---

## 9. Demand Validation

### Market Growth Signals
| Signal | Data Point | Source |
|--------|-----------|--------|
| Fractional CTO market doubling | 60,000 -> 120,000 fractional leaders (2022-2024) | CTO.Clinic |
| US businesses using fractional hiring | 25% (2023) -> 35% projected (2025) | CTO.Clinic |
| AI investment acceleration | $250B (2024) -> $375B (2025) -> $500B (2026) | Aiken House |
| Church tech adoption cleared | 95% of church leaders affirm technology's value; 45% now use AI (80% YoY increase) | Pushpay/Barna 2026 |
| Church software market | $6.2B (2024) -> $15.8B projected (2033), 9.7% CAGR | Data Horizon Research |
| Workflow automation market | $26B (2026), 9.4-10.1% CAGR | Mordor Intelligence |
| AI consulting market | $14B (2026) -> $117B projected (2035), 26.5% CAGR | Business Research Insights |

### Willingness-to-Pay Evidence
- Fractional CTOs billing $200-500/hr. A $49/month tool paying for itself in 12 minutes of saved time is trivial ROI.
- n8n Pro at $60/month and Dify Pro at $59/month are actively purchased by this audience -- $49/month is below both.
- Churches spend 2-13% of budget on IT. Median church budget of $300K means $6K-$39K/year on technology. A $49/month tool ($588/year) fits even tight budgets.
- 81.64% of non-profit software revenue comes from SaaS subscriptions -- the model is accepted.
- Developer tools principle: "Every paid feature is evaluated against: 'Can I get this from an open-source alternative?'" The free community tier is essential to overcome this.

### Competitive White Space
No tool currently combines all of: CLI-first, YAML-defined, self-hosted, human-in-the-loop, with a non-technical frontend (Open WebUI). Recent launches (Dvina, Aident AI, Agentfield, Relay.app) target generic enterprise/SMB. None targets fractional CTOs serving churches and non-profits.

---

## 10. Data Gaps

### Critical (Would Change Strategy if Filled)
1. **No primary interview data with fractional CTOs.** All persona insights are inferred from published content, not direct conversations. Five interviews would validate or invalidate the pain hierarchy.
2. **No data on how many consultants use CLI-based workflow tools.** The assumption that 20-30% of IT consultants are CLI-comfortable could be significantly off in either direction.
3. **No reliable count of consultants serving churches/non-profits.** The 10,000-20,000 estimate is an educated guess. The actual number could be 5,000 or 50,000.
4. **No direct willingness-to-pay survey.** Pricing is benchmarked from adjacent categories, not from asking target buyers "would you pay $49/month for this?"

### Moderate (Would Sharpen Tactics)
5. **No churn rate data for developer CLI tools specifically.** Using SMB SaaS benchmarks (3-7% monthly) as a proxy.
6. **No data on fractional CTO tool adoption patterns.** How many tools does the average fractional CTO pay for? What's the replacement cycle?
7. **Church/non-profit IT consultant spending on tools** -- no direct measurement exists.

### Minor (Nice to Have)
8. **Geographic distribution** of fractional CTOs and church tech consultants within the US.
9. **Competitive traction data** -- downloads, revenue, or user counts for Dify, n8n community edition, or similar open-source tools.

---

## 11. Red Flags and Yellow Flags

### Red Flags (Could Kill the Business)

**CLI limits the addressable market by 70-80%.**
Only 20-30% of even technical consultants will adopt a CLI tool. This is not a problem to solve later -- it is a fundamental ceiling on the market. The CLI-adjusted SAM is ~$61M, down from $249M. If the actual CLI-willing percentage is closer to 10%, the SAM drops to ~$25M and the Year 1 SOM becomes dangerously small.

**Open-source monetization conversion rates are brutal.**
Industry benchmarks show 0.5-3% free-to-paid conversion for open-source developer tools. 70% of micro-SaaS businesses earn under $1K/month. The free tier is essential for adoption, but it creates a large base of users who may never pay. The entire revenue model depends on hitting the 2-5% conversion target.

**Selling through the consultant to the church adds friction.**
The buying chain is: Skillrunner -> Fractional CTO -> Church board. Each link introduces delay, dilution, and potential failure. If the fractional CTO doesn't see enough value, the chain breaks. If the church board sees "AI" and gets nervous, the chain breaks. Two-step distribution is harder than direct sales.

### Yellow Flags (Risks to Monitor)

**AI agent builders are commoditizing fast.**
In 2026, AI agents are available to businesses with 5+ employees at $20/month per agent. If general-purpose AI agent platforms become "good enough" for church workflows, the differentiation gap narrows. Monitor Aident AI, Relay.app, and Zapier's AI features quarterly.

**Church market may adopt slower than signals suggest.**
95% of church leaders affirm technology's value, but affirming value and purchasing new tools are different things. "Every dollar spent on technology is one you can't spend to reach your neighbors and feed your community." Budget approval cycles at churches can take months, and volunteer turnover means re-training costs are ongoing.

**Fractional CTO as a distribution channel is unproven.**
The theory is sound: one fractional CTO deploys across 3-5 clients. But no data exists on whether fractional CTOs actually adopt and re-deploy tools this way, versus building bespoke solutions per client. If the deployment playbook doesn't transfer cleanly between engagements, the 3-5x multiplier doesn't materialize.

**Market research firm estimates vary by 2-3x.**
Workflow automation market estimates range from $9.1B to $27.1B depending on methodology. The $26B figure used here may be the high end. All market sizing in this analysis should be treated as directional, not precise.

**No first-party validation yet.**
All research is secondary -- published articles, reviews, surveys, market reports. Zero customers have been interviewed. Zero have used a prototype. The pain hierarchy, the persona details, and the willingness-to-pay estimates are hypotheses until tested with real conversations and real deployments.
