# Linear Import Guide for v1.2.0 MCP Issues

## Prerequisites

1. **Linear API Key**
   - Go to: Linear Settings → API → Create Personal API Key
   - Export it: `export LINEAR_API_KEY='your-key-here'`

2. **Check for Existing Issues**
   ```bash
   ./docs/linear-query-script.sh
   ```

---

## Quick Import: Empty Linear Workspace

If Linear shows no MCP-related issues, import all 12:

### Step 1: Create Epic

**Linear UI:**
- Projects → Create New Project/Epic
- Title: `v1.2.0 - MCP Tool Execution`
- Description: Copy from `docs/v1.2.0-linear-issues.md` (Epic section)

**Or via API:**
```bash
curl -X POST https://api.linear.app/graphql \
  -H "Authorization: $LINEAR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { projectCreate(input: { name: \"v1.2.0 - MCP Tool Execution\", description: \"Enable MCP tool execution for skills\" }) { project { id } } }"
  }'
```

### Step 2: Import Issues

For each issue (MCP-1 through MCP-12), create in Linear:

**Issue MCP-1: Server Management Commands**
```
Title: MCP Server Management Commands
Labels: mcp, cli, week-1
Priority: High
Estimate: 3 points
Description: [Copy from docs/v1.2.0-linear-issues.md]
```

**Issue MCP-2: Tool Discovery Commands**
```
Title: MCP Tool Discovery Commands
Labels: mcp, cli, week-1
Priority: High
Estimate: 2 points
Dependencies: MCP-1
Description: [Copy from docs/v1.2.0-linear-issues.md]
```

...and so on for all 12 issues.

### Step 3: Set Dependencies

In Linear, link issues with dependencies:
- MCP-2 → depends on MCP-1
- MCP-5 → depends on MCP-4
- MCP-6 → depends on MCP-4, MCP-5
- MCP-7 → depends on MCP-6
- MCP-9 → depends on MCP-4, MCP-6, MCP-8

---

## Automated Import Script

**Create all issues via API:**

```bash
#!/bin/bash
# bulk-import-linear.sh

LINEAR_API_KEY="$LINEAR_API_KEY"
API_URL="https://api.linear.app/graphql"

# Get team ID (replace with your team)
TEAM_ID="your-team-id"

# Function to create issue
create_issue() {
    local title="$1"
    local description="$2"
    local estimate="$3"
    local labels="$4"

    curl -X POST "$API_URL" \
      -H "Authorization: $LINEAR_API_KEY" \
      -H "Content-Type: application/json" \
      -d "{
        \"query\": \"mutation {
          issueCreate(input: {
            teamId: \\\"$TEAM_ID\\\",
            title: \\\"$title\\\",
            description: \\\"$description\\\",
            estimate: $estimate,
            labelIds: [\\\"$labels\\\"]
          }) {
            issue { id identifier }
          }
        }\"
      }"
}

# Create MCP-1
create_issue \
  "MCP Server Management Commands" \
  "Implement CLI commands for managing MCP servers..." \
  3 \
  "mcp,cli"

# Create MCP-2
# ... etc for all 12 issues
```

---

## Manual Import (Recommended for First Time)

**Why manual?**
- Review each issue description
- Ensure understanding of requirements
- Customize for your team's workflow
- Add team-specific context

**Steps:**
1. Open `docs/v1.2.0-linear-issues.md`
2. For each issue section:
   - Create new Linear issue
   - Copy title
   - Copy description
   - Set labels, priority, estimate
   - Add to v1.2.0 project/epic
3. Set dependencies after all issues created
4. Assign to agents or team members

**Time estimate:** 15-20 minutes for all 12 issues

---

## Mapping Linear to Implementation Plan

After importing, update:

**docs/v1.2.0-mcp-implementation-plan.md:**
```markdown
**Linear Epic:** SKILL-123  # Replace with actual epic ID

| Issue ID | Title | Estimate | Phase | Linear |
|----------|-------|----------|-------|--------|
| MCP-1 | Server Management | 3 | CLI | SKILL-124 |
| MCP-2 | Tool Discovery | 2 | CLI | SKILL-125 |
...
```

This creates bidirectional linking:
- Linear issue → Implementation plan (reference docs/v1.2.0-mcp-implementation-plan.md)
- Implementation plan → Linear issue ID

---

## Updating Existing Issues

If `linear-query-script.sh` found existing issues:

1. **Review each match:**
   - Is it the same scope as our MCP-X issue?
   - Is it more specific or more general?

2. **Decide:**
   - **Exact match:** Update description, add details from our spec
   - **Partial overlap:** Break into sub-tasks or merge
   - **Different scope:** Create new issue, reference old one

3. **Update references:**
   - Note in implementation plan which Linear issues map to which MCP-X

---

## Agent Assignment

**After import:**

1. **Label for parallel work:**
   - Week-1: MCP-1, MCP-2, MCP-3, MCP-8
   - Week-2: MCP-4, MCP-5, MCP-9, MCP-10
   - Week-3: MCP-6, MCP-7, MCP-11

2. **Assign to agents:**
   - Tag issues with agent names
   - Or use Linear's assignment feature
   - Set status to "Backlog" or "Ready"

3. **Track in Linear board:**
   - Columns: Backlog → In Progress → Review → Done
   - Use filters by label (mcp, week-1, etc.)

---

## Verification Checklist

After import:
- [ ] Epic created (v1.2.0 - MCP Tool Execution)
- [ ] 12 issues created (MCP-1 through MCP-12)
- [ ] All estimates set (total: 29 points)
- [ ] Labels applied (mcp, cli, integration, etc.)
- [ ] Dependencies linked
- [ ] All issues added to epic/project
- [ ] Implementation plan updated with Linear IDs

---

## Next Steps

1. Run `./docs/linear-query-script.sh` to check for existing issues
2. If clean, import all 12 issues (manual or automated)
3. Update implementation plan with Linear issue IDs
4. Assign to agents
5. Start parallel execution!

---

**Questions?**
- Linear API docs: https://developers.linear.app/docs/graphql/working-with-the-graphql-api
- Skillrunner implementation plan: `docs/v1.2.0-mcp-implementation-plan.md`
- Issue templates: `docs/v1.2.0-linear-issues.md`
