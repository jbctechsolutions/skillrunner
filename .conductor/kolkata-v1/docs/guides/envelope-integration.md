# Envelope Integration Guide

This guide explains how to use Skillrunner envelope output with external tools like Claude Code CLI.

## What is an Envelope?

An envelope is a structured JSON format that contains a multi-step workflow definition. Legacy/built-in Skillrunner skills (like `backend-architect` and `test`) generate envelopes instead of executing LLM calls directly.

## Generating Envelopes

```bash
# Generate an envelope and save to file
sr run backend-architect "add user authentication" --output envelope.json

# Generate compact JSON (no pretty printing)
sr run backend-architect "build API" --output envelope.json --compact
```

## Envelope Structure

```json
{
  "version": "1.0",
  "skill": "backend-architect",
  "request": "add user authentication",
  "steps": [
    {
      "intent": "plan",
      "model": "deepseek-coder-v2:16b",
      "prompt": "Analyze the following backend architecture request...",
      "context": [
        {
          "type": "folder",
          "source": "/context"
        },
        {
          "type": "pattern",
          "source": "imports"
        }
      ],
      "file_ops": [],
      "metadata": {
        "phase": "planning",
        "model_provider": "ollama",
        "model_provider_tier": "fast"
      }
    },
    {
      "intent": "edit",
      "model": "deepseek-coder-v2:16b",
      "prompt": "Implement the planned backend changes...",
      "context": [
        {
          "type": "folder",
          "source": "/src"
        }
      ],
      "file_ops": [],
      "metadata": {
        "phase": "implementation"
      }
    },
    {
      "intent": "run",
      "model": "deepseek-coder-v2:16b",
      "prompt": "Run tests and verify the implementation",
      "context": [],
      "file_ops": [],
      "metadata": {
        "phase": "verification"
      }
    }
  ],
  "metadata": {
    "created_at": "2025-11-28T21:18:29Z",
    "workspace": "."
  }
}
```

### Step Intent Types

- **`plan`**: Analysis and planning step
- **`edit`**: Implementation/code generation step
- **`run`**: Testing/verification step

### Context Types

- **`folder`**: Include entire folder contents
- **`file`**: Include specific file
- **`pattern`**: Include files matching pattern (e.g., "imports")

## Integration with Claude Code CLI

Claude Code CLI supports non-interactive execution that can process envelope content.

### Basic Usage

```bash
# Generate envelope
sr run backend-architect "build API endpoint" --output envelope.json

# Process envelope with Claude Code CLI
cat envelope.json | claude -p --output-format json
```

### Processing Individual Steps

```bash
# Extract and execute first step (plan)
jq -r '.steps[0].prompt' envelope.json | claude -p --output-format json

# Extract and execute second step (edit)
jq -r '.steps[1].prompt' envelope.json | claude -p --output-format json

# Extract and execute third step (run)
jq -r '.steps[2].prompt' envelope.json | claude -p --output-format json
```

### Sequential Step Execution

```bash
# Execute all steps sequentially
for i in 0 1 2; do
  intent=$(jq -r ".steps[$i].intent" envelope.json)
  prompt=$(jq -r ".steps[$i].prompt" envelope.json)
  echo "=== Executing $intent step ==="
  echo "$prompt" | claude -p --output-format json
  echo ""
done
```

### Using jq for Advanced Processing

```bash
# List all step intents
jq -r '.steps[] | "\(.intent): \(.model)"' envelope.json

# Extract all prompts
jq -r '.steps[] | "=== \(.intent) ===\n\(.prompt)\n"' envelope.json

# Get step metadata
jq '.steps[] | {intent, model, metadata}' envelope.json
```

## Claude Code CLI Flags

- **`-p, --print`**: Run in non-interactive mode and print results
- **`--output-format <format>`**: Specify output format:
  - `text`: Plain text output
  - `json`: JSON format
  - `stream-json`: Streaming JSON format

## Integration with Continue IDE

Continue IDE can consume envelope formats. The envelope structure is compatible with Continue's workflow execution format.

### Using with Continue

1. Generate the envelope:
   ```bash
   sr run backend-architect "your request" --output envelope.json
   ```

2. Import into Continue IDE:
   - Open Continue IDE
   - Use the envelope file to create a new workflow
   - Continue will parse the steps and execute them

## Programmatic Integration

### Python Example

```python
import json
import subprocess

# Load envelope
with open('envelope.json', 'r') as f:
    envelope = json.load(f)

# Execute each step
for step in envelope['steps']:
    print(f"Executing {step['intent']} step...")

    # Extract prompt
    prompt = step['prompt']

    # Execute with Claude Code CLI
    result = subprocess.run(
        ['claude', '-p', '--output-format', 'json'],
        input=prompt,
        text=True,
        capture_output=True
    )

    print(result.stdout)
```

### Node.js Example

```javascript
const fs = require('fs');
const { execSync } = require('child_process');

// Load envelope
const envelope = JSON.parse(fs.readFileSync('envelope.json', 'utf8'));

// Execute each step
envelope.steps.forEach((step, index) => {
  console.log(`Executing ${step.intent} step...`);

  // Execute with Claude Code CLI
  const result = execSync(
    `echo "${step.prompt}" | claude -p --output-format json`,
    { encoding: 'utf8' }
  );

  console.log(result);
});
```

## Format Conversion

If you need to convert the envelope format for a specific tool:

```bash
# Extract just the prompts
jq '.steps[] | {intent, prompt}' envelope.json > prompts.json

# Convert to a different format
jq '{workflow: .skill, steps: [.steps[] | {type: .intent, instruction: .prompt}]}' envelope.json > converted.json
```

## Best Practices

1. **Save envelopes for reuse**: Use `--output` to save envelopes for later execution
2. **Validate envelopes**: Check envelope structure before processing:
   ```bash
   jq '.' envelope.json > /dev/null && echo "Valid JSON"
   ```
3. **Process steps individually**: Execute steps one at a time for better error handling
4. **Capture outputs**: Save step outputs for debugging:
   ```bash
   jq -r '.steps[0].prompt' envelope.json | claude -p > step1-output.json
   ```

## Troubleshooting

### Envelope is empty or invalid
- Ensure the skill ran successfully
- Check that `--output` flag was used correctly
- Verify JSON is valid: `jq '.' envelope.json`

### Claude Code CLI doesn't recognize format
- Check Claude Code CLI documentation for exact format requirements
- You may need to transform the envelope structure
- Consider extracting just the prompts if full envelope isn't supported

### Steps not executing in order
- Process steps sequentially using a loop
- Check step dependencies in the envelope metadata
- Ensure previous step completes before executing next

## Related Documentation

- [Quick Start Guide](../getting-started/quick-start.md)
- [Command Reference](../COMMAND_REFERENCE.md)
- [API Documentation](../API.md)
