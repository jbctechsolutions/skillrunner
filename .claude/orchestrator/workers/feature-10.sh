#!/bin/bash
set -e
cd '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu'
PROMPT=$(cat '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu/.claude/orchestrator/workers/feature-10.prompt')
claude -p "$PROMPT" --allowedTools Bash,Read,Write,Edit,Glob,Grep 2>&1 | tee '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu/.claude/orchestrator/workers/feature-10.log'
echo 'WORKER_EXITED' >> '/Users/joel.castillo.cq/conductor/workspaces/skillrunner-v2/machu-picchu/.claude/orchestrator/workers/feature-10.log'
