#!/bin/bash
# Linear API Query Script for v1.2.0 MCP Issues
# This script checks if MCP-related issues already exist in Linear

set -e

# Check for API key
if [ -z "$LINEAR_API_KEY" ]; then
    echo "Error: LINEAR_API_KEY environment variable not set"
    echo ""
    echo "Get your API key from: Linear Settings → API → Create Personal API Key"
    echo "Then run: export LINEAR_API_KEY='your-key-here'"
    exit 1
fi

# Linear GraphQL API endpoint
API_URL="https://api.linear.app/graphql"

# Function to query Linear
query_linear() {
    local query="$1"
    curl -s -X POST "$API_URL" \
        -H "Authorization: $LINEAR_API_KEY" \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"$query\"}"
}

echo "Querying Linear for MCP-related issues..."
echo ""

# Query for issues with MCP or tool-related keywords
QUERY='
{
  issues(
    filter: {
      or: [
        { title: { containsIgnoreCase: "mcp" } }
        { title: { containsIgnoreCase: "tool" } }
        { title: { containsIgnoreCase: "v1.2" } }
        { description: { containsIgnoreCase: "model context protocol" } }
      ]
    }
    first: 50
  ) {
    nodes {
      id
      identifier
      title
      state { name }
      description
      createdAt
      labels { nodes { name } }
    }
  }
}
'

# Execute query
RESULT=$(query_linear "$QUERY")

# Check if we got results
if echo "$RESULT" | grep -q '"errors"'; then
    echo "Error querying Linear:"
    echo "$RESULT" | jq '.errors'
    exit 1
fi

# Parse and display results
ISSUE_COUNT=$(echo "$RESULT" | jq '.data.issues.nodes | length')

if [ "$ISSUE_COUNT" -eq 0 ]; then
    echo "✓ No existing MCP-related issues found in Linear"
    echo ""
    echo "Safe to import all 12 issues from docs/v1.2.0-linear-issues.md"
    echo ""
    echo "Next steps:"
    echo "1. Create Epic: v1.2.0 - MCP Tool Execution"
    echo "2. Import issues MCP-1 through MCP-12"
    echo "3. Set dependencies and estimates"
else
    echo "Found $ISSUE_COUNT potentially related issues:"
    echo ""
    echo "$RESULT" | jq -r '.data.issues.nodes[] | "[\(.identifier)] \(.title) - \(.state.name)"'
    echo ""
    echo "Review these issues and update/merge as needed before importing new ones."
fi

echo ""
echo "To create new issues programmatically, use the Linear API:"
echo "  curl -X POST https://api.linear.app/graphql \\"
echo "    -H \"Authorization: \$LINEAR_API_KEY\" \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -d '{\"query\": \"mutation { issueCreate(input: {...}) { issue { id } } }\"}'"
