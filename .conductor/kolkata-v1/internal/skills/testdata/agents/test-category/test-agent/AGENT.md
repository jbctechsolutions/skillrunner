---
name: test-agent
description: A test agent for unit testing
model: sonnet
primary_skill: test-skill
supporting_skills:
  - helper-skill
tools: Read, Write
routing:
  defer_to_skill: true
  fallback_model: anthropic/claude-3-sonnet-20240229
orchestration:
  enabled: true
  default_phases:
    - analysis
    - generation
  routing_strategy: local_first
  cost_optimization: true
---

# Test Agent

This is a test agent for unit testing purposes.

## Capabilities

- Testing agent loading
- Testing frontmatter parsing
