# Skillrunner + Cursor Integration Guide

## Overview

This guide shows how to use Skillrunner within Cursor IDE, enabling you to run AI-powered workflows directly from your code editor with the power of multi-phase orchestration and local model execution.

## Setup

### 1. Build Skillrunner Binary

```bash
# Clone the repository
git clone https://github.com/jbctechsolutions/skillrunner
cd skillrunner

# Build the binary
go build -o sr ./cmd/skillrunner/
```

### 2. Make Binary Accessible

**Option A: Add to PATH**
```bash
# Add project directory to PATH in ~/.zshrc or ~/.bashrc
# Replace /path/to/sr with your actual path
export PATH="/path/to/sr:$PATH"

# Reload shell
source ~/.zshrc  # or source ~/.bashrc
```

**Option B: Symbolic Link**
```bash
# Create a symlink to make sr available globally
ln -s /path/to/skillrunner/sr /usr/local/bin/sr
```

### 3. Verify Installation

```bash
# From any directory
sr status
```

## Using in Cursor Terminal

### Open Integrated Terminal

1. Press `` Ctrl+` `` or `Cmd+J` to open terminal
2. You're now in the integrated terminal with full access to `sr` commands

### Quick Commands

```bash
# Test with hello-orchestration
sr run hello-orchestration "a developer learning Go"

# Ask marketplace skill
sr ask avl-expert "microphone recommendation for 100-person venue"

# Code review current file
sr run code-review "$(cat $(pwd)/yourfile.go)"

# List available skills
sr list
```

## Cursor Composer Integration

### Using @ Commands with Skillrunner Output

You can pipe Skillrunner output into Cursor's AI:

```bash
# Generate architecture, then use in Cursor
sr run feature-implementation "dark mode toggle" > /tmp/plan.md

# In Cursor Composer, reference the output
# @/tmp/plan.md
```

### Create Custom Cursor Commands

Add to your `.cursor/commands/` directory:

**.cursor/commands/skillrunner-review.md**
```markdown
---
name: Review Code with Skillrunner
description: Run Skillrunner code review on selected code
---

Run the Skillrunner code-review skill on the selected code:

1. Save the selected code to a temp file
2. Run: sr run code-review "$(cat /tmp/selected-code.txt)"
3. Show the review results
4. Suggest improvements based on the output
```

**.cursor/commands/skillrunner-implement.md**
```markdown
---
name: Implement Feature with Skillrunner
description: Use Skillrunner to plan and implement a feature
---

Implement the requested feature using Skillrunner orchestration:

1. Run: sr run feature-implementation "<user's request>"
2. Review the 9-phase workflow output:
   - Requirements extraction
   - Architecture design
   - Implementation plan
   - Test design
   - Code generation
   - Test generation
   - Implementation review
   - Documentation
   - Deployment checklist
3. Create the necessary files based on the output
4. Run tests
```

**.cursor/commands/skillrunner-debug.md**
```markdown
---
name: Debug with Skillrunner
description: Systematic debugging using Skillrunner bug-fix workflow
---

Debug the issue using Skillrunner's systematic approach:

1. Run: sr run bug-fix "<user's bug description>"
2. Follow the 7-phase workflow:
   - Reproduce the bug
   - Root cause analysis
   - Plan fix
   - Identify tests
   - Implement fix
   - Verify fix
   - Create summary
3. Apply the suggested fixes
4. Verify the solution
```

## Cursor Snippets

Add to your global snippets (Cmd+Shift+P → "Preferences: Configure User Snippets"):

```json
{
  "Skillrunner Review": {
    "prefix": "sr-review",
    "body": [
      "sr run code-review \"$(cat ${TM_FILEPATH})\""
    ],
    "description": "Review current file with Skillrunner"
  },
  "Skillrunner Ask": {
    "prefix": "sr-ask",
    "body": [
      "sr ask ${1:skill-id} \"${2:question}\""
    ],
    "description": "Ask marketplace skill"
  },
  "Skillrunner Implement": {
    "prefix": "sr-implement",
    "body": [
      "sr run feature-implementation \"${1:feature description}\""
    ],
    "description": "Implement feature with Skillrunner"
  }
}
```

## Cursor Tasks Integration

Create `.vscode/tasks.json` for common Skillrunner operations:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Skillrunner: Review Current File",
      "type": "shell",
      "command": "sr",
      "args": [
        "run",
        "code-review",
        "$(cat ${file})"
      ],
      "problemMatcher": [],
      "presentation": {
        "reveal": "always",
        "panel": "new"
      }
    },
    {
      "label": "Skillrunner: Test Hello Orchestration",
      "type": "shell",
      "command": "sr",
      "args": [
        "run",
        "hello-orchestration",
        "a developer learning Go"
      ],
      "problemMatcher": []
    },
    {
      "label": "Skillrunner: List Skills",
      "type": "shell",
      "command": "sr",
      "args": ["list"],
      "problemMatcher": []
    },
    {
      "label": "Skillrunner: System Status",
      "type": "shell",
      "command": "sr",
      "args": ["status"],
      "problemMatcher": []
    }
  ]
}
```

**Run tasks with**: `Cmd+Shift+P` → "Tasks: Run Task" → Select task

## Keyboard Shortcuts

Add to your `keybindings.json` (Cmd+K Cmd+S):

```json
[
  {
    "key": "cmd+shift+r",
    "command": "workbench.action.tasks.runTask",
    "args": "Skillrunner: Review Current File"
  },
  {
    "key": "cmd+shift+s",
    "command": "workbench.action.tasks.runTask",
    "args": "Skillrunner: List Skills"
  }
]
```

## Example Workflows in Cursor

### 1. Review Code Before Commit

```bash
# In Cursor terminal
git status                                    # See changed files
sr run code-review "$(cat src/api.go)"   # Review changes
# Fix issues based on review
git add .
git commit -m "feat: ..."
```

### 2. Implement New Feature

```bash
# In Cursor terminal
sr run feature-implementation "Add user profile endpoint"

# Output shows:
# - Architecture design
# - Implementation steps
# - Test plan
# - Documentation needs

# Use Cursor Composer with the output:
# @/path/to/output Ask Cursor to implement based on Skillrunner's plan
```

### 3. Debug Production Issue

```bash
# In Cursor terminal
sr run bug-fix "API returns 500 when username contains special chars"

# Follow the systematic approach:
# 1. Reproduction steps
# 2. Root cause analysis
# 3. Fix implementation
# 4. Test verification
```

### 4. Ask Expert During Development

```bash
# Quick questions without context switching
sr ask avl-expert "Best practice for audio mixing in small venue?"
sr ask business-analyst "How to calculate customer LTV?"
sr ask data-analyst "Statistical significance for A/B test?"
```

## Advanced: Cursor + Skillrunner Automation

### Auto-Review on Save

Create a Git pre-commit hook that uses Skillrunner:

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Get staged Go files
FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')

if [ -n "$FILES" ]; then
  echo "Running Skillrunner code review..."
  for file in $FILES; do
    sr run code-review "$(cat $file)" > /tmp/review.txt
    if grep -q "critical" /tmp/review.txt; then
      echo "Critical issues found in $file"
      cat /tmp/review.txt
      exit 1
    fi
  done
fi
```

### Cursor Settings for Skillrunner

Add to `.cursor/settings.json`:

```json
{
  "terminal.integrated.env.osx": {
    "SKILLRUNNER_MODEL_POLICY": "cost_optimized",
    "OLLAMA_HOST": "http://localhost:11434"
  },
  "terminal.integrated.shellArgs.osx": [
    "-l"
  ],
  "files.watcherExclude": {
    "**/.skillrunner": true
  }
}
```

## Testing in Cursor

### Run Skillrunner Tests

```bash
# In Cursor terminal
go test ./...                              # All tests
go test ./internal/context/... -v          # Context management
go test -run TestChunker -v               # Specific test
```

### Watch Mode

```bash
# Auto-run tests on file changes
go test ./internal/context/... -v | grep -E '(PASS|FAIL)'
```

## Cost Tracking

Monitor your cost savings directly in Cursor:

```bash
# Check status
sr status

# Run workflow and see savings
sr run hello-orchestration "test" | grep "Cost:"
# Cost: $0.00 (saved $0.0037 vs Claude)
```

## Troubleshooting in Cursor

### Terminal Can't Find `sr`

```bash
# Check PATH in terminal
echo $PATH

# If missing, add to PATH (replace with your actual path):
export PATH="/path/to/sr:$PATH"

# Make permanent by adding to ~/.zshrc
```

### Ollama Not Running

```bash
# Start Ollama in background
ollama serve &

# Verify it's running
curl http://localhost:11434/api/tags
```

### Model Not Found

```bash
# List available models
ollama list

# Pull missing model
ollama pull qwen2.5:14b
```

## Best Practices

1. **Use Terminal for Quick Tasks**: Review, ask questions, quick implementations
2. **Use Tasks for Repeated Operations**: Code reviews, testing workflows
3. **Use Snippets for Common Commands**: Faster access to Skillrunner
4. **Combine with Cursor Composer**: Use Skillrunner output as context for further development
5. **Monitor Costs**: Keep track of free vs paid model usage

## Next Steps

- Explore all available skills: `sr list`
- Create custom orchestrated skills in `~/.skillrunner/skills/`
- Set up pre-commit hooks for automated reviews
- Integrate Skillrunner output into your Cursor workflows

## Related Documentation

- [Quick Start Guide](../getting-started/quick-start.md)
- [Model Selection Guide](model-selection.md)
- [API Keys Setup](api-keys-setup.md)
