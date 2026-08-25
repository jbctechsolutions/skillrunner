# n8n Market Research Automation - Implementation Spec

**Project:** Automated Daily Competitive Intelligence for SkillRunner
**Target Domain:** automation.jbc.dev
**Output:** Daily digest to Slack + optional SMS alerts for critical items

---

## Infrastructure Requirements

### Digital Ocean Deployment

Deploy n8n on Digital Ocean with the following specs:

**Option A: Droplet (Recommended for simplicity)**
- Size: Basic $12/mo (2GB RAM, 1 vCPU, 50GB SSD)
- Region: NYC1 or nearest to you
- OS: Ubuntu 24.04 LTS
- Domain: automation.jbc.dev (A record → Droplet IP)

**Option B: App Platform (Managed, slightly more expensive)**
- n8n Docker container
- $12-24/mo depending on resources
- Auto-SSL, auto-scaling

**Required Setup:**
1. Nginx reverse proxy with Let's Encrypt SSL
2. PostgreSQL database (can run on same droplet or use DO managed DB)
3. n8n with queue mode for reliability
4. Firewall: 80, 443 open; 5678 (n8n) only via localhost

### Docker Compose Configuration

```yaml
version: '3.8'

services:
  n8n:
    image: n8nio/n8n:latest
    restart: always
    ports:
      - "127.0.0.1:5678:5678"
    environment:
      - N8N_HOST=automation.jbc.dev
      - N8N_PORT=5678
      - N8N_PROTOCOL=https
      - WEBHOOK_URL=https://automation.jbc.dev/
      - N8N_BASIC_AUTH_ACTIVE=true
      - N8N_BASIC_AUTH_USER=${N8N_USER}
      - N8N_BASIC_AUTH_PASSWORD=${N8N_PASSWORD}
      - DB_TYPE=postgresdb
      - DB_POSTGRESDB_HOST=postgres
      - DB_POSTGRESDB_PORT=5432
      - DB_POSTGRESDB_DATABASE=n8n
      - DB_POSTGRESDB_USER=${POSTGRES_USER}
      - DB_POSTGRESDB_PASSWORD=${POSTGRES_PASSWORD}
      - EXECUTIONS_DATA_PRUNE=true
      - EXECUTIONS_DATA_MAX_AGE=168  # 7 days
    volumes:
      - n8n_data:/home/node/.n8n
    depends_on:
      - postgres

  postgres:
    image: postgres:15
    restart: always
    environment:
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=n8n
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  n8n_data:
  postgres_data:
```

### Nginx Configuration

```nginx
server {
    listen 80;
    server_name automation.jbc.dev;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name automation.jbc.dev;

    ssl_certificate /etc/letsencrypt/live/automation.jbc.dev/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/automation.jbc.dev/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:5678;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }
}
```

---

## Workflow Architecture

### Master Workflow: Daily Market Research Digest

**Trigger:** Cron schedule - 6:00 AM ET daily

**Workflow Structure:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Daily Research Trigger                            │
│                         (Cron: 0 6 * * *)                               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌───────────┐   ┌───────────┐   ┌───────────┐
            │  GitHub   │   │  Reddit   │   │    HN     │
            │ Trending  │   │  Monitor  │   │  Monitor  │
            └───────────┘   └───────────┘   └───────────┘
                    │               │               │
                    ▼               ▼               ▼
            ┌───────────┐   ┌───────────┐   ┌───────────┐
            │  Twitter  │   │    YC     │   │  Product  │
            │  Monitor  │   │ Companies │   │   Hunt    │
            └───────────┘   └───────────┘   └───────────┘
                    │               │               │
                    └───────────────┼───────────────┘
                                    ▼
                        ┌─────────────────────┐
                        │   Merge All Data    │
                        └─────────────────────┘
                                    │
                                    ▼
                        ┌─────────────────────┐
                        │   Claude Analysis   │
                        │   (Anthropic API)   │
                        └─────────────────────┘
                                    │
                        ┌───────────┴───────────┐
                        ▼                       ▼
                ┌───────────────┐       ┌───────────────┐
                │ Slack Digest  │       │  SMS Alert    │
                │   (Daily)     │       │ (Critical)    │
                └───────────────┘       └───────────────┘
```

---

## Data Sources Configuration

### 1. GitHub Trending

**Endpoint:** https://github.com/trending (scrape) or use unofficial API

**Filter Criteria:**
- Languages: Go, Python, Rust, TypeScript
- Topics: llm, ai, agents, workflow, orchestration, cost-optimization
- Time range: Daily

**n8n Implementation:**
```
HTTP Request node → HTML Extract node → Filter node
```

**Alternative:** Use https://api.gitterapp.com/ or build custom scraper

### 2. Hacker News

**Endpoint:** https://hacker-news.firebaseio.com/v0/

**Queries to monitor:**
- Top stories mentioning: LLM, AI cost, workflow, Claude, GPT, Ollama
- Show HN posts in AI/dev tools category
- Comments mentioning: Langfuse, Helicone, Factory, cost tracking

**n8n Implementation:**
```javascript
// Fetch top 100 stories
const topStories = await fetch('https://hacker-news.firebaseio.com/v0/topstories.json');
// Fetch each story and filter by keywords
```

### 3. Reddit Monitoring

**Subreddits to monitor (from user subscriptions + research sources):**

**Tech/Dev (Primary - check daily):**
- r/golang
- r/devops
- r/sre
- r/docker
- r/Terraform
- r/hashicorp
- r/aws
- r/AZURE
- r/sysadmin
- r/linuxadmin
- r/coding

**AI/Productivity (Primary - check daily):**
- r/ChatGPTPromptGenius
- r/n8n
- r/ObsidianMD
- r/Notion
- r/PKMS
- r/productivity
- r/NoteTaking
- r/todoist

**Self-Hosting/Local (Secondary):**
- r/selfhosted (for SkillRunner positioning research)

**Church Tech (Personal interest):**
- r/churchtech
- r/ProPresenter
- r/QSYS
- r/livesound
- r/techtheatre

**General Tech News:**
- r/Futurology
- r/technology (via r/all filtering)

**Keywords to track:**
- cost tracking, cost optimization, AI costs
- workflow orchestration, multi-phase
- Langfuse, Helicone, Factory.ai
- Claude Code, Aider, OpenHands
- LLM routing, model selection

**n8n Implementation:**
```
HTTP Request (Reddit JSON API) → Filter → Aggregate
URL pattern: https://www.reddit.com/r/{subreddit}/new.json?limit=50
```

### 4. Twitter/X Monitoring

**High-value accounts extracted from user's bookmarks (ranked by bookmark frequency):**

**Tier 1 - Most Bookmarked (check every run):**
- @tom_doerr (146 bookmarks) - AI/dev tools curator
- @GithubProjects (50) - GitHub projects community
- @dani_avila7 (28) - Claude Code expert
- @iannuttall (21) - Factory.ai, dev tools
- @hayesdev_ (16) - AI app templates
- @DataChaz (15) - Data/AI content
- @aaditsh (15) - AI developer
- @Sumanth_077 (14) - AI content
- @steipete (14) - Developer tools
- @omarsar0 (13) - AI research

**Tier 2 - Frequently Bookmarked (check daily):**
- @svpino (12) - ML engineer
- @Saboo_Shubham_ (12) - AI agents
- @EXM7777 (11) - AI/Gemini coverage
- @akshay_pachaar (11) - AI dev
- @ryancarson (9) - Durable agents
- @PrajwalTomar_ (9) - AI content
- @kevinkern (9) - Dev tools
- @mattpocockuk (8) - TypeScript/Claude Code
- @ericzakariasson (8) - Cursor guides
- @badlogicgames (8) - Coding agents

**Tier 3 - Key Influencers (check daily):**
- @claudeai - Official Anthropic Claude
- @AnthropicAI - Anthropic official
- @simonw (7) - LLM tools expert
- @mitchellh (7) - HashiCorp founder
- @dhh (6) - Rails creator, dev opinions
- @pontusab (6) - AI SDK tools
- @boringmarketer (6) - Claude Code reviews
- @lennysan (5) - Product/startup
- @cursor_ai - Cursor official
- @cline - Cline AI official

**Tier 4 - Competitors & Ecosystem:**
- @LangfuseAI - Langfuse
- @hwchase17 - LangChain founder
- @BraceSproul - LangSmith
- @factoryai - Factory.ai (if exists)
- @OpenRouterAI - OpenRouter

**Total: ~50 accounts to track**

**API Options:**
1. **Twitter API Basic ($100/mo):** 10K tweet reads - covers all tiers
2. **Free Tier:** Limited to 1,500 tweets/mo - Tier 1 only
3. **No API Alternative:** Use RSS via Nitter or manual review

**Recommended:** Start with free tier for Tier 1 accounts, upgrade if needed

**Full Account List:** All 633 accounts exported to `twitter_accounts_to_monitor.txt` in this directory.

### 4b. Readwise Integration (Future Bookmarks Only)

**Important:** Readwise only syncs bookmarks AFTER you connect it. It won't backfill historical data. Use the CSV-extracted account list above for historical coverage.

**Readwise API:** https://readwise.io/api_defs

**Benefits:**
- Syncs NEW Twitter bookmarks automatically (going forward)
- Also captures highlights from articles, books, newsletters
- Single API for multiple content sources
- Signals what YOU found interesting (high-value filter)

**n8n Implementation:**
```javascript
// Fetch recent highlights from Readwise
const response = await fetch('https://readwise.io/api/v2/highlights/', {
  headers: {
    'Authorization': `Token ${READWISE_TOKEN}`
  }
});

// Filter for Twitter source and relevant tags
const twitterHighlights = data.results.filter(h =>
  h.source === 'twitter' &&
  h.highlighted_at > lastCheckTime
);
```

**Setup:**
1. Get API token from https://readwise.io/access_token
2. Configure n8n HTTP Request node with token
3. Poll daily for new highlights
4. Filter by source (twitter, web, kindle, etc.)

### 4c. Readwise Reader (Read-Later App)

**Readwise Reader** is a read-later app (like Pocket/Instapaper) that's separate from Readwise highlights. It captures articles, newsletters, and Twitter threads you save to read later.

**Why This Matters:**
Reader content is HIGH SIGNAL - you explicitly saved it. This makes it valuable for:
1. Surfacing articles you saved but haven't read yet
2. Extracting themes from your reading interests
3. Identifying content patterns (what topics keep appearing?)

**Readwise Reader API (v3):** https://readwise.io/reader_api

**n8n Implementation:**
```javascript
// Fetch saved documents from Reader
const response = await fetch('https://readwise.io/api/v3/list/', {
  headers: {
    'Authorization': `Token ${READWISE_TOKEN}`
  },
  params: {
    location: 'new',           // Unread items
    updatedAfter: lastCheckISO  // Since last run
  }
});

// Response includes:
// - url, title, author, summary
// - saved_at, reading_progress
// - notes (if you annotated)
// - source (twitter, web, email, etc.)
```

**Workflow Integration:**
```
n8n Schedule (daily)
    ↓
HTTP Request: GET /api/v3/list/ (saved in last 24h)
    ↓
Filter: source = 'web' OR source = 'twitter_thread'
    ↓
Claude Analysis: "What are the common themes in these saved articles?"
    ↓
Include in digest under "From Your Reading Queue"
```

**Digest Section Example:**
```markdown
## From Your Reading Queue (3 articles saved yesterday)

**Theme:** All 3 relate to AI agent orchestration

1. "Building Reliable AI Agents" - saved from @simonw
   - Key point: Retry logic is critical for multi-step workflows

2. "The Cost of AI in Production" - saved from HN
   - Key point: Most companies overspend 40% on LLM calls

3. "DAG vs State Machine for AI Workflows"
   - Relevance: Directly validates SkillRunner's DAG approach
```

**SaaS Value:** For the potential hosted version, Reader integration is a premium feature:
- Users connect their existing reading workflow
- No RSS configuration needed - their habits ARE the config
- Higher engagement (analyzing content they already care about)

### 5. YC Company Directory

**Endpoint:** https://www.ycombinator.com/companies

**Filter Criteria:**
- Batch: W24, S24, W25
- Industry: Developer Tools, AI, B2B
- Keywords: LLM, AI, workflow, cost

**n8n Implementation:**
- Scrape company listings weekly (not daily - low churn)
- Store seen companies to detect new ones

### 6. Product Hunt

**Endpoint:** https://api.producthunt.com/v2/api/graphql

**Filter Criteria:**
- Topics: Developer Tools, Artificial Intelligence
- Votes: > 50 (filter noise)

**n8n Implementation:**
```graphql
query {
  posts(first: 50, topic: "developer-tools") {
    edges {
      node {
        name
        tagline
        votesCount
        website
      }
    }
  }
}
```

### 7. Competitor Changelog Monitoring

**Direct competitors to track:**

| Competitor | Changelog URL | Check Frequency |
|------------|---------------|-----------------|
| Langfuse | https://langfuse.com/changelog | Daily |
| Helicone | https://www.helicone.ai/changelog | Daily |
| Factory.ai | https://www.factory.ai/blog | Daily |
| LiteLLM | https://github.com/BerriAI/litellm/releases | Daily |
| OpenRouter | https://openrouter.ai/docs/changelog | Daily |

**n8n Implementation:**
- RSS feeds where available
- HTML scraping with change detection for others
- Store last seen content hash to detect updates

---

## AI Analysis Configuration

### Claude API Integration

**Model:** claude-3-5-haiku-20241022 (fast, cheap for daily analysis)
**Fallback:** claude-3-5-sonnet-20241022 (for complex analysis)

**System Prompt:**

```
You are a competitive intelligence analyst for SkillRunner, an AI workflow orchestration tool.

SkillRunner's positioning: "AI coding workflows that optimize themselves"
Key differentiators: Per-phase cost tracking, automatic model selection, local-first with Ollama

Your job is to analyze daily market signals and identify:
1. THREATS: Competitor moves that could hurt SkillRunner
2. OPPORTUNITIES: Gaps or trends SkillRunner could capitalize on
3. SIGNALS: Early indicators of market shifts
4. INSPIRATION: Ideas worth borrowing or adapting

For each item, rate urgency: 🔴 Critical (act now) | 🟡 Important (this week) | 🟢 Monitor (track)

Format your output as a structured digest with clear sections.
```

**Analysis Prompt Template:**

```
Analyze these market signals for SkillRunner:

## GitHub Trending
{github_data}

## Hacker News
{hn_data}

## Reddit Discussions
{reddit_data}

## Twitter/X Activity
{twitter_data}

## Competitor Updates
{competitor_data}

## New YC Companies
{yc_data}

## Product Hunt Launches
{ph_data}

Provide:
1. Executive Summary (3 bullets max)
2. Critical Alerts (anything rated 🔴)
3. This Week's Focus (🟡 items)
4. Trends to Watch (🟢 items)
5. Actionable Recommendations (max 3)
```

---

## Output Configuration

### Slack Integration

**Webhook Setup:**
1. Create Slack App at https://api.slack.com/apps
2. Enable Incoming Webhooks
3. Create webhook for target channel (e.g., #market-intel)

**Message Format:**

```json
{
  "blocks": [
    {
      "type": "header",
      "text": {
        "type": "plain_text",
        "text": "📊 Daily Market Intelligence - {date}"
      }
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*Executive Summary*\n{summary}"
      }
    },
    {
      "type": "divider"
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*🔴 Critical Alerts*\n{critical_items}"
      }
    }
  ]
}
```

### SMS Alerts (Critical Items Only)

**Provider Options:**
- Twilio ($0.0075/SMS)
- AWS SNS ($0.00645/SMS)

**Trigger Criteria:**
- Any item rated 🔴 Critical
- Competitor major release
- Direct mention of SkillRunner anywhere

---

## Required API Keys & Credentials

| Service | Required | Cost | Setup URL |
|---------|----------|------|-----------|
| Anthropic API | Yes | ~$0.25-1/day | https://console.anthropic.com |
| Slack Webhook | Yes | Free | https://api.slack.com/apps |
| Readwise API | Recommended | Included with subscription | https://readwise.io/access_token |
| Twitter API | Optional | $100/mo or free tier | https://developer.twitter.com |
| Product Hunt API | Optional | Free | https://www.producthunt.com/v2/oauth/applications |
| Twilio (SMS) | Optional | Pay per use | https://www.twilio.com |
| Reddit API | Optional | Free | https://www.reddit.com/prefs/apps |

---

## Implementation Checklist

### Phase 1: Infrastructure (Day 1)
- [ ] Create Digital Ocean Droplet
- [ ] Configure DNS (automation.jbc.dev → Droplet IP)
- [ ] Install Docker & Docker Compose
- [ ] Deploy n8n with PostgreSQL
- [ ] Configure Nginx + SSL
- [ ] Test n8n access

### Phase 2: Core Workflows (Day 2)
- [ ] Create GitHub Trending workflow
- [ ] Create Hacker News workflow
- [ ] Create Reddit monitoring workflow
- [ ] Create competitor changelog workflow
- [ ] Test each workflow individually

### Phase 3: Analysis & Output (Day 3)
- [ ] Configure Anthropic API in n8n
- [ ] Create master workflow that aggregates data
- [ ] Create Claude analysis node
- [ ] Configure Slack webhook
- [ ] Test end-to-end flow

### Phase 4: Refinement (Day 4+)
- [ ] Add Twitter monitoring (if API access)
- [ ] Add Product Hunt monitoring
- [ ] Add YC company tracking
- [ ] Configure SMS alerts
- [ ] Tune relevance filters
- [ ] Set up error notifications

---

## Terraform Configuration (Optional)

For infrastructure-as-code deployment:

```hcl
# main.tf
terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

resource "digitalocean_droplet" "n8n" {
  image    = "ubuntu-24-04-x64"
  name     = "n8n-automation"
  region   = "nyc1"
  size     = "s-1vcpu-2gb"
  ssh_keys = [var.ssh_key_fingerprint]

  user_data = file("${path.module}/cloud-init.yaml")
}

resource "digitalocean_domain" "automation" {
  name = "automation.jbc.dev"
}

resource "digitalocean_record" "n8n" {
  domain = digitalocean_domain.automation.id
  type   = "A"
  name   = "@"
  value  = digitalocean_droplet.n8n.ipv4_address
}

resource "digitalocean_firewall" "n8n" {
  name = "n8n-firewall"

  droplet_ids = [digitalocean_droplet.n8n.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0"]
  }
}
```

---

## User Action Required

**Data Extracted ✅:**
1. ✅ **Reddit subscriptions:** Extracted from HTML export - 25+ relevant subreddits identified
2. ✅ **Twitter priority accounts:** Extracted from bookmark CSVs - 50 high-value accounts ranked by engagement

**Still Needed:**
3. **Slack workspace:** Confirm the Slack workspace and channel for digest delivery
4. **SMS phone number:** If you want SMS alerts for critical items, provide the number
5. **Readwise API token:** Get from https://readwise.io/access_token (recommended for cleaner Twitter coverage)
6. **Additional competitors:** Any specific companies or products to track beyond the list above

---

## Estimated Costs

| Item | Monthly Cost |
|------|-------------|
| DO Droplet (2GB) | $12 |
| Anthropic API (~$0.50/day) | $15 |
| Twitter API (optional) | $0-100 |
| Twilio SMS (optional) | $5-10 |
| **Total** | **$27-137/mo** |

---

*Spec created December 4, 2025*
*For use with Claude to build infrastructure PR*
