#!/bin/bash
set -e
cd '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut'
PROMPT=$(cat '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut/.claude/orchestrator/workers/code-review.prompt')
# Review mode: read-only tools plus Write for the findings file
# Note: Bash intentionally excluded for security - reviewers don't need shell access
claude -p "$PROMPT" --allowedTools Read,Glob,Grep,Write 2>&1 | tee '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut/.claude/orchestrator/workers/code-review.log'
echo 'REVIEWER_EXITED' >> '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut/.claude/orchestrator/workers/code-review.log'
