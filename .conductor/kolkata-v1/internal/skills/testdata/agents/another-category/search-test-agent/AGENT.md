---
name: search-test-agent
description: An agent for testing search functionality with unique keywords
model: sonnet
primary_skill: search-skill
tools: Read, Write, Edit
routing:
  defer_to_skill: false
  fallback_model: anthropic/claude-3-haiku-20240229
---

# Search Test Agent

This agent helps test the search functionality with unique keywords like "database" and "backend".
