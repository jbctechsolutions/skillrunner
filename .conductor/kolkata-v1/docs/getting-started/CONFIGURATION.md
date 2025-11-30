# Skillrunner Configuration Guide

This guide explains how to configure Skillrunner to work with Ollama, Claude (Anthropic), ChatGPT (OpenAI), and Gemini (Google).

## Quick Setup

### 1. Ollama (Local - Free)

Ollama doesn't require any API keys. Just make sure it's installed and running:

```bash
# Install Ollama (if not already installed)
brew install ollama  # macOS
# or
curl https://ollama.ai/install.sh | sh  # Linux

# Start Ollama service
ollama serve

# Pull some models (optional, but recommended)
ollama pull qwen2.5:14b
ollama pull qwen2.5-coder:32b
```

**No configuration needed** - Ollama is automatically detected when running on `http://localhost:11434`.

### 2. Set Environment Variables

Set these environment variables in your shell profile (`~/.zshrc`, `~/.bashrc`, etc.):

```bash
# Anthropic (Claude)
export ANTHROPIC_API_KEY="sk-ant-api03-..."

# OpenAI (ChatGPT)
export OPENAI_API_KEY="sk-..."

# Google (Gemini)
export GOOGLE_API_KEY="AIza..."
```

Then reload your shell:
```bash
source ~/.zshrc  # or source ~/.bashrc
```

### 3. Verify Configuration

The `config/models.yaml` file is already configured to use these environment variables. You can verify your setup:

```bash
# Check if providers are available
sr models list

# Test a simple execution
sr run golang-pro "test"
```

## Getting API Keys

### Anthropic (Claude)

1. Go to [console.anthropic.com](https://console.anthropic.com/)
2. Sign up or log in
3. Navigate to **API Keys**
4. Click **Create Key**
5. Copy the key (starts with `sk-ant-api03-`)
6. Set: `export ANTHROPIC_API_KEY="your-key-here"`

### OpenAI (ChatGPT)

1. Go to [platform.openai.com](https://platform.openai.com/)
2. Sign up or log in
3. Navigate to **API Keys**
4. Click **Create new secret key**
5. Copy the key (starts with `sk-`)
6. **Important**: Add a payment method under **Billing** (required for API access)
7. Set: `export OPENAI_API_KEY="your-key-here"`

### Google (Gemini)

1. Go to [aistudio.google.com](https://aistudio.google.com/)
2. Sign in with your Google account
3. Click **Get API Key**
4. Create a new API key or use existing project
5. Copy the key (starts with `AIza`)
6. Set: `export GOOGLE_API_KEY="your-key-here"`

## Configuration File

The routing configuration is in `config/models.yaml`. It's already set up with:

- **Cheap profile**: Uses free Ollama models first, then cheapest cloud models (Haiku, GPT-4o-mini, Gemini Flash)
- **Balanced profile** (default): Uses free Ollama models first, then balanced cloud models (Sonnet, GPT-4o, Gemini Pro)
- **Premium profile**: Uses premium models (same as balanced, but you can customize)

**Ollama is prioritized** in all profiles for cost savings (free, local inference). Cloud providers are used as fallbacks when:
- Ollama models are not available
- You need higher quality/responses
- You explicitly force cloud models with `--force-cloud`

The config automatically reads API keys from environment variables:
- `ANTHROPIC_API_KEY` → Anthropic provider
- `OPENAI_API_KEY` → OpenAI provider
- `GOOGLE_API_KEY` → Google provider

**Note**: Ollama doesn't require API keys - it's automatically configured when Ollama is running on `http://localhost:11434`.

## Testing Your Setup

### Test Ollama (Free)

```bash
# Ollama should work without any API keys
sr run golang-pro "hello" --profile cheap
```

### Test Cloud Providers

```bash
# Test with Anthropic
sr run golang-pro "test" --profile balanced

# Test with OpenAI
sr run golang-pro "test" --profile premium

# Test with Google
# (Will be selected automatically based on profile and availability)
```

### Check Available Models

```bash
# List all available models
sr models list

# Check specific provider
sr models list --provider=ollama
sr models list --provider=anthropic
```

## Profile-Based Routing

Skillrunner uses **profile-based routing** to select models:

- **`--profile cheap`**: Tries free Ollama models first, then cheapest cloud models (Gemini Flash, GPT-4o-mini, Claude Haiku)
- **`--profile balanced`** (default): Tries free Ollama models first, then balanced cloud models (Gemini Pro, GPT-4o, Claude Sonnet)
- **`--profile premium`**: Uses premium models (same as balanced currently)

The router automatically:
1. **Tries Ollama models first** (free, local) - prioritized in all profiles
2. Falls back to cloud providers if Ollama is unavailable or you need higher quality
3. Tries models in the profile's candidate list in order
4. Falls back to next available provider if one fails

**Cost Savings**: By default, Skillrunner uses Ollama (free) for 70-90% of requests, only falling back to cloud providers when needed.

## Troubleshooting

### "no available providers for profile X"

This means the API keys aren't set or the providers aren't configured. Check:

```bash
# Verify environment variables are set
echo $ANTHROPIC_API_KEY
echo $OPENAI_API_KEY
echo $GOOGLE_API_KEY

# If empty, add them to your shell profile and reload
```

### Ollama not working

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# If not running, start it
ollama serve
```

### Provider-specific errors

- **Anthropic**: Make sure your API key is valid and has credits
- **OpenAI**: Ensure payment method is added in OpenAI dashboard
- **Google**: Verify API key is enabled in Google AI Studio

## Minimal Setup (Ollama Only)

If you only want to use Ollama (free, local):

1. Install and start Ollama (see above)
2. **No API keys needed**
3. Run: `sr run golang-pro "test" --profile cheap`

The system will automatically prefer Ollama models when available.

## Full Setup (All Providers)

For maximum flexibility:

1. Set all three environment variables (ANTHROPIC_API_KEY, OPENAI_API_KEY, GOOGLE_API_KEY)
2. Start Ollama service
3. The router will automatically select the best model based on:
   - Profile preference (cheap/balanced/premium)
   - Provider availability
   - Cost optimization

## Next Steps

- See `docs/getting-started/quick-start.md` for usage examples
- See `docs/guides/api-keys-setup.md` for detailed API key setup
- Run `sr list` to see available skills
- Run `sr run <skill> <request>` to execute workflows
