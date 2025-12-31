#!/bin/bash
set -e
cd '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu'
PROMPT=$(cat '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu/.claude/orchestrator/workers/architecture-review.prompt')
# Review mode: read-only tools plus Write for the findings file
# Note: Bash intentionally excluded for security - reviewers don't need shell access
claude -p "$PROMPT" --allowedTools Read,Glob,Grep,Write 2>&1 | tee '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu/.claude/orchestrator/workers/architecture-review.log'
echo 'REVIEWER_EXITED' >> '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu/.claude/orchestrator/workers/architecture-review.log'
