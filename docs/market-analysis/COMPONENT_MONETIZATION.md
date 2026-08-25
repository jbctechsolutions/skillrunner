# SkillRunner Components: Standalone Library Potential

**Date:** December 4, 2025
**Purpose:** Evaluate which components could be licensed as standalone libraries

---

## Component Assessment

### 1. Cost Computer (`internal/metrics/cost.go`)

**Current State:** ~90 lines, simple token-to-cost calculation

**Standalone Value:** LOW

| Factor | Assessment |
|--------|------------|
| Uniqueness | Formula is trivial: `(tokens/1000) * price` |
| Competition | Helicone, LiteLLM, tiktoken all do this |
| Complexity | Too simple to charge for |
| Market Need | Everyone needs this, but it's 10 lines of code |

**Verdict:** NOT WORTH EXTRACTING
- The calculation is trivial
- The value is in price data maintenance, not the formula
- Could offer price data as a service instead

---

### 2. Profile Router (`internal/routing/router.go`)

**Current State:** ~110 lines, profile-to-model routing with fallback

**Standalone Value:** MEDIUM

| Factor | Assessment |
|--------|------------|
| Uniqueness | Profile abstraction (cheap/balanced/premium) is useful |
| Competition | LiteLLM, Martian, OpenRouter all route |
| Complexity | Simple but useful pattern |
| Market Need | Yes - people want "just give me a cheap model" |

**Potential Library:**
```go
import "github.com/jbctechsolutions/llm-router"

router := llmrouter.New(llmrouter.Config{
    Profiles: map[string][]string{
        "cheap":    {"ollama/qwen2.5", "groq/llama-3.1-8b"},
        "balanced": {"anthropic/claude-3-haiku", "openai/gpt-4o-mini"},
        "premium":  {"anthropic/claude-3-5-sonnet"},
    },
})

provider, model, err := router.Route("cheap")
```

**Revenue Model:**
- Open source core (MIT)
- Paid: Managed routing service with availability monitoring

**Verdict:** MAYBE - but LiteLLM proxy does this already

---

### 3. DAG Executor (`internal/orchestration/dag.go`)

**Current State:** ~220 lines, topological sort, batch execution

**Standalone Value:** MEDIUM-HIGH

| Factor | Assessment |
|--------|------------|
| Uniqueness | Clean Go implementation, no deps |
| Competition | LangGraph (Python), generic DAG libs exist |
| Complexity | Non-trivial, solves real problem |
| Market Need | Yes - anyone building multi-step LLM workflows |

**Potential Library:**
```go
import "github.com/jbctechsolutions/llm-dag"

dag := llmdag.New()
dag.AddPhase("analyze", llmdag.Phase{
    Prompt: "Analyze {{input}}",
    Model:  "cheap",
})
dag.AddPhase("summarize", llmdag.Phase{
    Prompt:    "Summarize {{analyze.output}}",
    Model:     "balanced",
    DependsOn: []string{"analyze"},
})

results, err := dag.Execute(ctx, "some input")
```

**Revenue Model:**
- Open source core (MIT)
- Paid: Cloud execution with persistence, retries, observability

**Verdict:** POSSIBLE - fills gap for Go developers

---

### 4. Outcome Tracker (NOT YET BUILT)

**Proposed Functionality:** Track success/failure per phase + model combination

**Standalone Value:** HIGH

| Factor | Assessment |
|--------|------------|
| Uniqueness | NOBODY does this - routers predict, they don't learn |
| Competition | None for outcome-based learning |
| Complexity | Medium - aggregation + recommendation engine |
| Market Need | High - "which model actually works for my use case?" |

**Potential Library:**
```go
import "github.com/jbctechsolutions/llm-outcomes"

tracker := outcomes.New(outcomes.Config{
    Storage: outcomes.SQLite("metrics.db"),
})

// After each LLM call
tracker.Record(outcomes.Result{
    TaskType:  "code-review",
    Phase:     "analyze",
    Model:     "anthropic/claude-3-haiku",
    Success:   true,
    Retries:   0,
    Duration:  2500 * time.Millisecond,
    Cost:      0.002,
})

// Get recommendations
stats := tracker.Stats("code-review", "analyze")
// stats.BestModel = "anthropic/claude-3-5-sonnet" (95% success)
// stats.CheapestWorking = "ollama/qwen2.5" (89% success, $0.00)
```

**Revenue Model:**
- Open source local tracking (MIT)
- Paid: Aggregated insights across users ("What model works best for code review?")

**Verdict:** HIGH POTENTIAL - this is genuinely unique

---

### 5. LLM Pricing Database (NOT YET EXTRACTED)

**Proposed Functionality:** Maintained database of LLM prices across providers

**Standalone Value:** HIGH

| Factor | Assessment |
|--------|------------|
| Uniqueness | Helicone has 300+ models, but it's not a standalone library |
| Competition | LiteLLM has pricing, but requires Python |
| Complexity | Low code, high maintenance |
| Market Need | Every cost calculator needs this |

**Potential Library/Service:**
```go
import "github.com/jbctechsolutions/llm-prices"

prices := llmprices.Latest()
cost := prices.Calculate("anthropic/claude-3-5-sonnet", 1000, 500)
// cost = $0.0065

// Or as API
// GET https://prices.skillrunner.dev/v1/models
// GET https://prices.skillrunner.dev/v1/calculate?model=...&input=1000&output=500
```

**Revenue Model:**
- Free API with rate limits
- Paid: Higher rate limits, webhooks for price changes, historical data

**Verdict:** HIGH POTENTIAL - low effort, recurring value

---

## Revenue Models Comparison

### A. Static Monthly Fee

| Component | Price | Justification |
|-----------|-------|---------------|
| Router Library | $0 (OSS) | Too simple, use for marketing |
| DAG Library | $0 (OSS) | Build community, upsell cloud |
| Pricing API | $9/mo | 10K requests, price alerts |
| Outcome Tracker Cloud | $29/mo | Aggregated insights, dashboards |

**Pros:** Predictable revenue, simple billing
**Cons:** Doesn't scale with customer value

### B. Usage-Based Pricing

| Component | Price | Metric |
|-----------|-------|--------|
| Pricing API | $0.001/request | Per price lookup |
| Outcome Tracker | $0.01/record | Per outcome logged |
| Cloud DAG Execution | $0.05/run | Per workflow run |

**Pros:** Scales with usage, low entry barrier
**Cons:** Unpredictable revenue, complex billing

### C. Percentage of Savings (Value-Based)

| Model | Price |
|-------|-------|
| "You saved $X, we take 10%" | 10% of documented savings |

**Example:**
- User runs 1000 workflows/month
- Without optimization: $500/month
- With SkillRunner optimization: $100/month
- Savings: $400/month
- SkillRunner fee: $40/month (10%)

**Pros:** Aligned incentives, easy to justify ROI
**Cons:** Hard to verify, requires trust, complex calculation

### D. Hybrid Model (RECOMMENDED)

| Tier | Price | What You Get |
|------|-------|--------------|
| Free | $0 | Libraries (MIT), local tracking, 1K price API calls/mo |
| Pro | $19/mo | 50K price API calls, cloud outcome sync, basic insights |
| Team | $49/seat/mo | Aggregated insights, team dashboards, priority support |
| Enterprise | 5% of savings | Custom SLAs, dedicated support, on-prem |

---

## Extractable Components: Priority Order

### Phase 1: Build and Keep Internal (v1.0-v1.2)
These stay in SkillRunner, prove value first.

1. **Outcome Tracker** - Build this first, it's your moat
2. **DAG Executor** - Keep integrated for now

### Phase 2: Extract as Microservices (v1.3+)

1. **LLM Pricing API** (`prices.skillrunner.dev`)
   - Effort: 1-2 weeks
   - Revenue: $500-2K/mo at scale
   - Value: Marketing, ecosystem building

2. **Outcome Insights API** (`insights.skillrunner.dev`)
   - Effort: 2-3 weeks (after outcome tracker is built)
   - Revenue: $2K-10K/mo at scale
   - Value: This is the real product

### Phase 3: Extract as Libraries (v1.4+)

1. **llm-dag** (Go library)
   - Effort: 1 week to extract and document
   - Revenue: $0 (OSS for marketing)
   - Value: Ecosystem, credibility, contributor pipeline

2. **llm-outcomes** (Go library)
   - Effort: 1 week to extract
   - Revenue: $0 (OSS) + cloud upsell
   - Value: Lock-in, data collection

---

## Revenue Projections by Model

### Scenario: 1,000 Active Users

| Revenue Model | Monthly Revenue | Notes |
|---------------|-----------------|-------|
| Static Pro ($19/mo) | $19,000 | Assumes 100% conversion (unrealistic) |
| Static Pro ($19/mo) | $1,900 | Assumes 10% conversion (realistic) |
| Usage ($0.01/record) | $1,000 | 100K records/mo |
| % Savings (10%) | $4,000 | $100 avg savings × 10% × 400 users |
| Hybrid | $3,500 | Mix of free, Pro, usage |

### Scenario: 10,000 Active Users

| Revenue Model | Monthly Revenue | Notes |
|---------------|-----------------|-------|
| Static Pro ($19/mo) | $19,000 | 10% conversion |
| Usage ($0.01/record) | $10,000 | 1M records/mo |
| % Savings (10%) | $40,000 | Scales with savings |
| Hybrid | $35,000 | Best of all models |

---

## Recommendation

### What to Extract (Priority Order)

1. **LLM Pricing API** - Low effort, quick win, marketing value
2. **Outcome Insights API** - Your actual moat, build after v1.2
3. **llm-dag library** - OSS for ecosystem, after product proven

### Revenue Model

**Start with Hybrid:**
- Free tier: Local everything, 1K API calls
- Pro ($19/mo): Cloud sync, more API calls, basic insights
- Team ($49/seat): Dashboards, aggregated insights
- Enterprise: Custom, potentially % of savings

**Why not pure % of savings:**
- Hard to verify savings
- Requires deep integration to measure
- Enterprises distrust "we'll see your costs"
- Save this for enterprise tier only

### What NOT to Extract

- **Cost Calculator** - Too trivial, just include in pricing API
- **Router** - LiteLLM already does this, no differentiation
- **Cache** - Generic problem, use Redis/BadgerDB

---

## Summary Table

| Component | Extract? | Revenue Model | Priority |
|-----------|----------|---------------|----------|
| Cost Calculator | NO | - | - |
| Profile Router | NO | - | - |
| DAG Executor | LATER | OSS + Cloud | P3 |
| Outcome Tracker | YES | Cloud + Insights | P1 |
| Pricing Database | YES | API + Subscription | P2 |

**The money is in:**
1. **Outcome insights** - "What model works best for your task?"
2. **Aggregated benchmarks** - "What model works best across all users?"
3. **Pricing data** - Maintained, accurate, API-accessible

**Not in:**
- Libraries (keep OSS for marketing)
- Simple utilities (too easy to replicate)

---

*Document prepared December 4, 2025*
