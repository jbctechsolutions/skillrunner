#!/bin/bash
set -e
cd '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut'
PROMPT=$(cat '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut/.claude/orchestrator/workers/feature-6.prompt')
claude -p "$PROMPT" --allowedTools Bash,Read,Write,Edit,Glob,Grep 2>&1 | tee '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut/.claude/orchestrator/workers/feature-6.log'
echo 'WORKER_EXITED' >> '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/beirut/.claude/orchestrator/workers/feature-6.log'
