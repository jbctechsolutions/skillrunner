# API Keys Setup Guide

This guide explains how to set up API keys for all cloud providers supported by Skillrunner.

## Supported Providers

Skillrunner supports the following cloud LLM providers:

1. **Anthropic** - Claude models (Opus, Sonnet, Haiku)
2. **OpenAI** - GPT models (GPT-4, GPT-3.5)
3. **Google AI** - Gemini models
4. **OpenRouter** - Gateway to 100+ models from multiple providers
5. **Groq** - Ultra-fast inference for Llama, Mixtral, Gemma models

## Routing Configuration

Skillrunner uses two configuration files:

1. **`~/.skillrunner/config.yaml`** - Main provider configuration (for the engine)
2. **`config/models.yaml`** - Routing configuration (for orchestrated skills)

The routing config (`config/models.yaml`) defines:
- **Providers**: Which providers are available (Ollama, Anthropic, OpenAI, Google)
- **Models**: Model definitions with costs
- **Profiles**: Which models to try for each profile (cheap, balanced, premium)

**Important**: Ollama is automatically included in the routing config and is **prioritized first** in all profiles for cost savings (free, local inference). Cloud providers are used as fallbacks.

You only need to set API keys for cloud providers if you want to use them. Ollama works without any API keys.

## Configuration Methods

You can configure API keys in two ways:

### Method 1: Environment Variables (Recommended)

Set environment variables in your shell profile (`.bashrc`, `.zshrc`, etc.):

```bash
# Anthropic
export ANTHROPIC_API_KEY="sk-ant-api03-..."

# OpenAI
export OPENAI_API_KEY="sk-..."

# Google AI
export GOOGLE_API_KEY="AIza..."

# OpenRouter
export OPENROUTER_API_KEY="sk-or-v1-..."

# Groq
export GROQ_API_KEY="gsk_..."
```

Then configure providers in `~/.skillrunner/config.yaml`:

```yaml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    enabled: true

  openai:
    api_key: ${OPENAI_API_KEY}
    enabled: true

  google:
    api_key: ${GOOGLE_API_KEY}
    enabled: true

  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    enabled: true

  groq:
    api_key: ${GROQ_API_KEY}
    enabled: true

  ollama:
    url: http://localhost:11434
    enabled: true
```

### Method 2: Direct Configuration

Store API keys directly in `~/.skillrunner/config.yaml` (less secure):

```yaml
providers:
  anthropic:
    api_key: "sk-ant-api03-..."
    enabled: true

  openai:
    api_key: "sk-..."
    enabled: true

  google:
    api_key: "AIza..."
    enabled: true

  openrouter:
    api_key: "sk-or-v1-..."
    enabled: true

  groq:
    api_key: "gsk_..."
    enabled: true
```

⚠️ **Security Warning**: If you use this method, ensure your config file has restricted permissions:
```bash
chmod 600 ~/.skillrunner/config.yaml
```

---

## Provider-Specific Setup

### 1. Anthropic (Claude)

**Models:** Claude 3 Opus, Claude 3.5 Sonnet, Claude 3 Haiku

#### Getting an API Key

1. Go to [console.anthropic.com](https://console.anthropic.com/)
2. Sign up or log in
3. Navigate to **API Keys** section
4. Click **Create Key**
5. Copy the key (starts with `sk-ant-api03-`)

#### Configuration

```yaml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    enabled: true
```

#### Pricing (as of 2024)

- **Claude 3.5 Sonnet**: $3.00 / 1M input tokens, $15.00 / 1M output tokens
- **Claude 3 Opus**: $15.00 / 1M input tokens, $75.00 / 1M output tokens
- **Claude 3 Haiku**: $0.25 / 1M input tokens, $1.25 / 1M output tokens

#### Testing

```bash
sr models check anthropic/claude-3-5-sonnet-20241022
```

---

### 2. OpenAI (GPT)

**Models:** GPT-4, GPT-4 Turbo, GPT-3.5 Turbo

#### Getting an API Key

1. Go to [platform.openai.com](https://platform.openai.com/)
2. Sign up or log in
3. Navigate to **API Keys** section
4. Click **Create new secret key**
5. Copy the key (starts with `sk-`)
6. Add payment method under **Billing**

#### Configuration

```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    enabled: true
```

#### Pricing (as of 2024)

- **GPT-4 Turbo**: $10.00 / 1M input tokens, $30.00 / 1M output tokens
- **GPT-4**: $30.00 / 1M input tokens, $60.00 / 1M output tokens
- **GPT-3.5 Turbo**: $0.50 / 1M input tokens, $1.50 / 1M output tokens

#### Testing

```bash
sr models check openai/gpt-4-turbo
```

---

### 3. Google AI (Gemini)

**Models:** Gemini 1.5 Pro, Gemini 1.5 Flash

#### Getting an API Key

1. Go to [aistudio.google.com](https://aistudio.google.com/)
2. Sign in with Google account
3. Click **Get API Key**
4. Create a new API key
5. Copy the key (starts with `AIza`)

#### Configuration

```yaml
providers:
  google:
    api_key: ${GOOGLE_API_KEY}
    enabled: true
```

#### Pricing (as of 2024)

- **Gemini 1.5 Pro**: $3.50 / 1M input tokens, $10.50 / 1M output tokens
- **Gemini 1.5 Flash**: $0.075 / 1M input tokens, $0.30 / 1M output tokens

#### Testing

```bash
sr models check google/gemini-1.5-pro
```

---

### 4. OpenRouter (Gateway)

**Models:** 100+ models including Claude, GPT, Gemini, Llama, Mistral, and more

#### Getting an API Key

1. Go to [openrouter.ai](https://openrouter.ai/)
2. Click **Sign In** (supports Google, GitHub, or email)
3. Navigate to **Keys** section
4. Click **Create Key**
5. Copy the key (starts with `sk-or-v1-`)
6. Add credits under **Credits** (pay-as-you-go)

#### Configuration

```yaml
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1  # optional, this is the default
    enabled: true
```

#### Features

- **No markup**: Same pricing as underlying providers
- **Auto-fallback**: If one model is down, automatically tries alternatives
- **Model discovery**: Access 100+ models without separate API keys
- **Free models**: Several free models available (gemma-7b-it, phi-2, etc.)

#### Pricing

Varies by model. OpenRouter charges the same as the underlying provider:
- Claude 3.5 Sonnet: $3.00 / 1M input, $15.00 / 1M output
- GPT-4 Turbo: $10.00 / 1M input, $30.00 / 1M output
- Llama 3.1 70B: Free or $0.52 / 1M tokens (varies by provider)

#### Testing

```bash
# List all available models
sr models list --provider=openrouter

# Check a specific model
sr models check openrouter/anthropic/claude-3.5-sonnet
```

---

### 5. Groq (Ultra-Fast Inference)

**Models:** Llama 3.3 70B, Llama 3.1 70B/8B, Mixtral 8x7B, Gemma 2 9B

#### Getting an API Key

1. Go to [console.groq.com](https://console.groq.com/)
2. Sign up (free tier available)
3. Navigate to **API Keys** section
4. Click **Create API Key**
5. Copy the key (starts with `gsk_`)

#### Configuration

```yaml
providers:
  groq:
    api_key: ${GROQ_API_KEY}
    enabled: true
```

#### Features

- **Ultra-fast**: 500-800 tokens/second (10-50x faster than standard APIs)
- **Free tier**: Generous free tier with rate limits
- **Open source models**: Llama, Mixtral, Gemma

#### Pricing (as of 2024)

Free tier with rate limits:
- 30 requests per minute
- 6,000 tokens per minute
- 14,400 requests per day

Paid tiers available with higher limits.

#### Testing

```bash
# List all Groq models
sr models list --provider=groq

# Check a specific model
sr models check groq/llama-3.3-70b-versatile
```

---

## Verification

After configuring your API keys, verify they work:

### 1. List All Models

```bash
sr models list
```

Expected output:
```
PROVIDER    MODEL                        CONTEXT  MEMORY   STATUS
anthropic   claude-3-5-sonnet-20241022  200K     -        ✓ Ready
openai      gpt-4-turbo                 128K     -        ✓ Ready
google      gemini-1.5-pro              2M       -        ✓ Ready
openrouter  anthropic/claude-3.5-sonnet 200K     -        ✓ Ready
groq        llama-3.3-70b-versatile     128K     -        ✓ Ready
ollama      qwen2.5-coder:32b           32K      19GB     ✓ Ready
```

### 2. Check Specific Provider

```bash
sr models list --provider=anthropic
sr models list --provider=openai
sr models list --provider=google
sr models list --provider=openrouter
sr models list --provider=groq
```

### 3. Health Check a Model

```bash
sr models check anthropic/claude-3-5-sonnet-20241022
```

Expected output:
```
✓ Model available
  Provider: Anthropic
  Context window: 200,000 tokens
  Status: Ready to use
```

If you see errors, check:
1. API key is correct and not expired
2. Environment variable is set correctly
3. Provider is enabled in config
4. Sufficient credits/quota available

---

## Troubleshooting

### "Model not found" Error

```bash
sr models check anthropic/claude-3-5-sonnet-20241022
```

If you get:
```
Error: Model 'anthropic/claude-3-5-sonnet-20241022' not found

Provider: Anthropic (https://api.anthropic.com)
Status:   API key invalid

Suggestions:
  1. Check your API key in ~/.skillrunner/config.yaml or $ANTHROPIC_API_KEY
  2. Verify the key at console.anthropic.com
  3. Ensure the provider is enabled in config
```

**Solutions:**
1. Check API key format (should start with `sk-ant-api03-`)
2. Verify key hasn't expired
3. Check environment variable is exported: `echo $ANTHROPIC_API_KEY`
4. Ensure config file has correct syntax

### "Rate limit exceeded" Error

**Solutions:**
1. Check your usage quota on provider's dashboard
2. Add payment method if on free tier
3. Use local models (Ollama) to reduce cloud API calls
4. Implement retry logic with exponential backoff

### Environment Variables Not Loading

**Solutions:**
1. Reload shell: `source ~/.bashrc` or `source ~/.zshrc`
2. Check variable is set: `echo $ANTHROPIC_API_KEY`
3. Ensure no extra spaces in export statement
4. Use single quotes for complex values: `export KEY='value'`

### Config File Not Found

**Solutions:**
1. Create config directory: `mkdir -p ~/.skillrunner`
2. Create config file: `touch ~/.skillrunner/config.yaml`
3. Set correct permissions: `chmod 600 ~/.skillrunner/config.yaml`

---

## Security Best Practices

### 1. Never Commit API Keys

Add to `.gitignore`:
```gitignore
# Config files with API keys
config.yaml
.env
*.key
```

### 2. Use Environment Variables

Store keys in environment, not in code:
```bash
# ~/.bashrc or ~/.zshrc
export ANTHROPIC_API_KEY="sk-ant-api03-..."
```

### 3. Restrict File Permissions

```bash
chmod 600 ~/.skillrunner/config.yaml
chmod 700 ~/.skillrunner
```

### 4. Rotate Keys Regularly

Rotate API keys every 90 days or if compromised:
1. Generate new key on provider's dashboard
2. Update environment variable or config
3. Revoke old key
4. Test new key works

### 5. Use Separate Keys for Dev/Prod

```bash
# Development
export ANTHROPIC_API_KEY="sk-ant-api03-dev-..."

# Production
export ANTHROPIC_API_KEY="sk-ant-api03-prod-..."
```

### 6. Monitor Usage

Check usage dashboards regularly:
- **Anthropic**: console.anthropic.com
- **OpenAI**: platform.openai.com/usage
- **Google AI**: aistudio.google.com
- **OpenRouter**: openrouter.ai/activity
- **Groq**: console.groq.com/usage

---

## Cost Optimization

### 1. Use Local Models First

Configure Ollama for free, local inference:
```yaml
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
```

### 2. Set Model Preferences

Prefer cheaper models in skills:
```yaml
skills:
  code-review:
    preferred_models:
      - ollama/qwen2.5-coder:32b  # Free (local)
      - groq/llama-3.1-70b         # Free (cloud, rate-limited)
      - anthropic/claude-3-haiku   # Cheap (cloud)
      - anthropic/claude-3-5-sonnet # Expensive (cloud, fallback)
```

### 3. Use Groq for Speed

Groq offers free ultra-fast inference:
```yaml
providers:
  groq:
    api_key: ${GROQ_API_KEY}
    enabled: true
```

Models: Llama 3.3 70B, Mixtral 8x7B (free tier)

### 4. Use OpenRouter for Best Pricing

OpenRouter finds cheapest providers:
```yaml
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    enabled: true
```

Benefits:
- Same pricing as direct providers (no markup)
- Automatic fallback to cheaper alternatives
- Access to free models

---

## Example Complete Configuration

```yaml
# ~/.skillrunner/config.yaml

providers:
  # Local (free)
  ollama:
    url: http://localhost:11434
    enabled: true

  # Cloud providers (paid)
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    enabled: true

  openai:
    api_key: ${OPENAI_API_KEY}
    enabled: true

  google:
    api_key: ${GOOGLE_API_KEY}
    enabled: true

  # Gateways (paid with free options)
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    enabled: true

  groq:
    api_key: ${GROQ_API_KEY}
    enabled: true

# Model resolution policy
model_policy: local_first  # Try local (Ollama) first, then cloud
```

Environment variables (`~/.bashrc` or `~/.zshrc`):
```bash
export ANTHROPIC_API_KEY="sk-ant-api03-..."
export OPENAI_API_KEY="sk-..."
export GOOGLE_API_KEY="AIza..."
export OPENROUTER_API_KEY="sk-or-v1-..."
export GROQ_API_KEY="gsk_..."
```

---

## Next Steps

1. **Test configuration**: Run `sr models list` to verify all providers
2. **Run a skill**: Try a simple skill to test end-to-end
3. **Monitor usage**: Check provider dashboards for usage/costs
4. **Optimize**: Adjust model preferences based on performance/cost

---

## Related Documentation

- [Provider Configuration Migration](PROVIDER_CONFIG_MIGRATION.md) - Migrating from old config format
- [Provider Adapter Design](PROVIDER_ADAPTER_DESIGN.md) - Architecture details
- [Provider Adapter Implementation Summary](PROVIDER_ADAPTER_IMPLEMENTATION_SUMMARY.md) - Implementation details

---

**Last Updated**: 2025-01-19
**Maintained By**: Joel Castillo + Claude Code
