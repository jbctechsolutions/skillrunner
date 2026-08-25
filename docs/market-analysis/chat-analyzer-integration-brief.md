# Chat Analyzer × SkillRunner Integration Brief

**For:** Chat Analyzer Project
**From:** SkillRunner Market Analysis
**Date:** December 3, 2025

---

## Context

SkillRunner is a local-first AI workflow orchestrator with built-in cost tracking. It executes multi-phase AI workflows defined in YAML, routing tasks to optimal models (local Ollama or cloud providers) based on cost/quality profiles.

**Key SkillRunner capabilities:**
- Multi-phase DAG workflow execution
- Cost tracking per phase and workflow
- MCP (Model Context Protocol) support (v1.3.0+)
- Profile-based routing (cheap/balanced/premium)
- Skill marketplace for sharing workflows

---

## Integration Opportunity

Chat Analyzer can serve as a **personal context layer** for SkillRunner workflows by implementing an MCP server interface.

### Value Proposition

| For Chat Analyzer | For SkillRunner |
|-------------------|-----------------|
| Workflows that analyze stored context | Rich personal context for AI tasks |
| Automated insight generation | "Remember" capability across workflows |
| Structured output storage | Persistent knowledge graph integration |

---

## Recommended Implementation: MCP Server

### MCP Server Capabilities to Expose

```typescript
// Tools to expose via MCP
tools: [
  {
    name: "query_conversations",
    description: "Search conversations by person, topic, date range, or tags",
    parameters: {
      query: string,
      sources?: ["slack", "sms", "social", "cli"],
      dateRange?: { start: string, end: string },
      limit?: number
    }
  },
  {
    name: "get_connections",
    description: "Get AI-identified connections for a topic or conversation",
    parameters: {
      topic: string,
      depth?: number  // how many hops in the graph
    }
  },
  {
    name: "store_insight",
    description: "Store a new insight with connections to source material",
    parameters: {
      content: string,
      connections: string[],  // IDs of related items
      tags?: string[],
      backend?: "notion" | "obsidian" | "database"
    }
  },
  {
    name: "get_context_for_person",
    description: "Get all context about interactions with a specific person",
    parameters: {
      person: string,
      includeConnections?: boolean
    }
  },
  {
    name: "find_related",
    description: "Find items related to given text using AI similarity",
    parameters: {
      text: string,
      limit?: number
    }
  }
]
```

### Resources to Expose

```typescript
// Resources available via MCP
resources: [
  {
    uri: "chat-analyzer://recent",
    name: "Recent Conversations",
    description: "Last 7 days of indexed conversations"
  },
  {
    uri: "chat-analyzer://connections/{topic}",
    name: "Topic Connections",
    description: "Knowledge graph for a specific topic"
  },
  {
    uri: "chat-analyzer://people/{name}",
    name: "Person Context",
    description: "All context about a specific person"
  }
]
```

---

## Example SkillRunner Workflows Using Chat Analyzer

### 1. Weekly Synthesis

```yaml
name: weekly-synthesis
description: Analyze week's conversations and surface insights

phases:
  - id: gather
    prompt: |
      Query chat-analyzer for all conversations from the past week.
      Group by: projects, people, recurring themes.
    tools:
      - mcp://chat-analyzer/query_conversations
    profile: cheap

  - id: analyze
    prompt: |
      Analyze these conversations for:
      1. Key decisions made
      2. Open questions/blockers
      3. Emerging patterns
      4. Action items mentioned but not tracked

      Input: {{phases.gather.output}}
    profile: balanced
    depends_on: [gather]

  - id: connections
    prompt: |
      Identify non-obvious connections between topics discussed this week.
      Look for: related projects, people who should talk, ideas that complement.
    tools:
      - mcp://chat-analyzer/get_connections
    profile: balanced
    depends_on: [gather]

  - id: store
    prompt: |
      Store these insights in the knowledge graph:
      - Analysis: {{phases.analyze.output}}
      - Connections: {{phases.connections.output}}

      Tag as: weekly-synthesis, {{current_date}}
    tools:
      - mcp://chat-analyzer/store_insight
    profile: cheap
    depends_on: [analyze, connections]
```

### 2. Meeting Prep

```yaml
name: meeting-prep
description: Prepare context before meeting with someone

input:
  person: string
  meeting_topic: string

phases:
  - id: history
    prompt: |
      Get all interactions with {{input.person}} from the last 90 days.
    tools:
      - mcp://chat-analyzer/get_context_for_person
    profile: cheap

  - id: related
    prompt: |
      Find conversations related to: {{input.meeting_topic}}
    tools:
      - mcp://chat-analyzer/find_related
    profile: cheap

  - id: brief
    prompt: |
      Create a meeting prep brief:

      ## Relationship Context
      - Recent interactions: {{phases.history.output}}
      - Tone/sentiment trends
      - Open items from past conversations

      ## Topic Context
      - Related discussions: {{phases.related.output}}
      - Key points to remember
      - Potential questions they might ask

      ## Suggested Talking Points
      Based on the above, suggest 3-5 talking points.
    profile: balanced
    depends_on: [history, related]
```

### 3. Idea Development

```yaml
name: develop-idea
description: Expand an idea using personal knowledge context

input:
  idea: string

phases:
  - id: find_related
    prompt: |
      Search for all conversations, notes, and connections related to:
      "{{input.idea}}"
    tools:
      - mcp://chat-analyzer/find_related
      - mcp://chat-analyzer/get_connections
    profile: cheap

  - id: expand
    prompt: |
      Develop this idea further using the related context:

      IDEA: {{input.idea}}

      RELATED CONTEXT:
      {{phases.find_related.output}}

      Please:
      1. Identify how past conversations inform this idea
      2. Note any contradictions or challenges mentioned before
      3. Suggest next steps based on patterns in my thinking
      4. Identify people I should talk to about this
    profile: premium
    depends_on: [find_related]

  - id: store
    prompt: |
      Store this developed idea with connections to:
      - The original related items
      - Any people mentioned
      - Projects this relates to

      Content: {{phases.expand.output}}
    tools:
      - mcp://chat-analyzer/store_insight
    profile: cheap
    depends_on: [expand]
```

---

## Technical Integration Notes

### MCP Server Setup

Chat Analyzer should expose an MCP server that SkillRunner can connect to:

```json
// SkillRunner config (~/.skillrunner/config.yaml)
mcp_servers:
  chat-analyzer:
    command: "chat-analyzer"
    args: ["mcp-server"]
    # OR for remote
    url: "http://localhost:3100/mcp"
```

### Authentication

For local use, no auth needed. For cloud sync scenarios:
- Share auth tokens between products
- Or use separate auth with user linking

### Data Flow

```
User Request
    │
    ▼
SkillRunner Workflow
    │
    ├──▶ MCP Query ──▶ Chat Analyzer ──▶ Knowledge Graph
    │                        │
    │                        ▼
    │                   Query Results
    │                        │
    ◀────────────────────────┘
    │
    ▼
AI Processing (with context)
    │
    ▼
Store Results ──▶ Chat Analyzer ──▶ Knowledge Graph
```

---

## Future Considerations

### Unified Product Suite (v2.0.0+)

If both products mature, consider:
- Shared cloud backend
- Unified subscription ("JBC AI Suite")
- Combined cost tracking dashboard
- Single Team/Enterprise offering

### Shared Components

Potential shared infrastructure:
- Authentication system
- Cost tracking service
- Cloud sync backend
- MCP server framework

---

## Action Items for Chat Analyzer

1. **Implement MCP server interfacgit e** - Expose query/store capabilities
2. **Design tool schemas** - Define clear input/output contracts
3. **Add SkillRunner-friendly output formats** - Structured data for workflow phases
4. **Consider bidirectional sync** - Store SkillRunner outputs as knowledge items
5. **Document MCP endpoints** - For SkillRunner skill authors

---

*This brief is maintained in the SkillRunner project at `docs/market-analysis/chat-analyzer-integration-brief.md`*
