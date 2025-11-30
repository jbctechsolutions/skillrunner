# Model Selection Guide

Skillrunner supports multiple ways to specify which model to use for skills, tasks, and phases. This guide explains all the options.

## Model Selection Priority

Models are selected in the following priority order:

1. **CLI `--model` flag** (highest priority - overrides everything)
2. **Phase-specific models** (from `routing.preferred_models` in skill YAML)
3. **Profile-based routing** (default - uses `--profile` flag)

## 1. CLI Model Override

Use the `--model` flag to override model selection for an entire skill execution:

```bash
# Use a specific model ID from routing_models
sr run golang-pro "refactor this code" --model balanced-anthropic-sonnet

# Use provider/model format
sr run golang-pro "refactor this code" --model anthropic/claude-3-5-sonnet-20241022
sr run golang-pro "refactor this code" --model openai/gpt-4o
sr run golang-pro "refactor this code" --model google/gemini-1.5-pro
```

### Supported Formats

- **Model ID**: Use IDs from `config/models.yaml` `routing_models` section
  - Examples: `balanced-anthropic-sonnet`, `cheap-openai-mini`, `premium-google`

- **Provider/Model format**: Use `provider/model-name` format
  - Examples: `anthropic/claude-3-5-sonnet-20241022`, `openai/gpt-4o`, `google/gemini-1.5-pro`

## 2. Phase-Specific Models

You can specify preferred models for individual phases in your skill YAML:

```yaml
phases:
  - id: analysis
    name: Code Analysis
    prompt_template: "Analyze this code: {{code}}"
    routing:
      preferred_models:
        - balanced-anthropic-sonnet  # Try this first
        - balanced-openai-gpt4o      # Fallback option
      fallback_models:
        - cheap-anthropic-haiku      # Last resort
    output_key: analysis_result

  - id: generation
    name: Code Generation
    prompt_template: "Generate code based on: {{analysis_result}}"
    routing:
      preferred_models:
        - premium-anthropic           # Use premium for generation
    output_key: generated_code
```

### Phase Routing Options

- **`preferred_models`**: List of model IDs to try in order (from `routing_models` in config)
- **`fallback_models`**: Models to try if all preferred models fail
- **`selection_strategy`**: (Future) Strategy for model selection (cheapest, fastest, best_quality)

## 3. Profile-Based Routing (Default)

If no model override or phase-specific models are specified, Skillrunner uses profile-based routing:

```bash
# Use cheap profile (tries cheapest models first)
sr run golang-pro "test" --profile cheap

# Use balanced profile (default)
sr run golang-pro "test" --profile balanced

# Use premium profile (tries premium models first)
sr run golang-pro "test" --profile premium
```

### Profile Configuration

Profiles are defined in `config/models.yaml`:

```yaml
routing_profiles:
  cheap:
    candidate_models:
      - cheap-anthropic-haiku
      - cheap-openai-mini
      - cheap-google-flash

  balanced:
    candidate_models:
      - balanced-anthropic-sonnet
      - balanced-openai-gpt4o
      - balanced-google-pro

  premium:
    candidate_models:
      - premium-anthropic
      - premium-openai
      - premium-google
```

The router tries models in order until one is available.

## Examples

### Example 1: Force a Specific Model

```bash
# Always use Claude Sonnet for this execution
sr run golang-pro "refactor code" --model anthropic/claude-3-5-sonnet-20241022
```

### Example 2: Phase-Specific Models

```yaml
# In your skill.yaml
phases:
  - id: quick_analysis
    routing:
      preferred_models:
        - cheap-anthropic-haiku  # Fast and cheap
    # ... rest of phase config

  - id: detailed_review
    routing:
      preferred_models:
        - premium-anthropic  # High quality for review
    # ... rest of phase config
```

### Example 3: Profile with Override

```bash
# Use premium profile, but override with specific model
sr run golang-pro "test" --profile premium --model balanced-anthropic-sonnet
# The --model flag takes precedence
```

## Model IDs Reference

Model IDs are defined in `config/models.yaml` under `routing_models`. Common IDs:

### Cheap Profile Models
- `cheap-anthropic-haiku`
- `cheap-openai-mini`
- `cheap-google-flash`

### Balanced Profile Models
- `balanced-anthropic-sonnet`
- `balanced-openai-gpt4o`
- `balanced-google-pro`

### Premium Profile Models
- `premium-anthropic`
- `premium-openai`
- `premium-google`

## Troubleshooting

### "model not found: X"

- Check that the model ID exists in `config/models.yaml` `routing_models`
- For provider/model format, ensure the provider name matches exactly (e.g., `anthropic`, not `claude`)

### "provider not available: X"

- Ensure the API key is set for that provider (see `docs/getting-started/CONFIGURATION.md`)
- Check that the provider is configured in `config/models.yaml` `providers` section

### Phase-specific models not working

- Ensure model IDs in `preferred_models` match IDs in `routing_models`
- Check that the provider for those models is configured and has API keys set

## Best Practices

1. **Use profiles for general execution**: Let the router choose the best model
2. **Use phase-specific models for optimization**: Specify models per phase based on task requirements
3. **Use CLI override for testing**: Override models when testing or debugging
4. **Prefer model IDs over provider/model format**: Model IDs are validated and easier to maintain
