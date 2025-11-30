# Privacy Policy

**Last Updated: November 2025**

## Overview

Skillrunner is an open-source tool that operates entirely on your local machine. We are committed to protecting your privacy and ensuring transparency about how the software handles data.

## Data Collection and Storage

### Local Data Storage

Skillrunner stores the following data locally on your machine in the `~/.skillrunner/` directory:

1. **Configuration Data** (`~/.skillrunner/config.yaml`)
   - API keys (stored in plain text - use environment variables for better security)
   - Model preferences and routing profiles
   - Default settings

2. **Skills and Agents** (`~/.skillrunner/skills/`)
   - Imported skill definitions
   - Custom orchestration workflows
   - Agent configurations

3. **Usage Metrics** (`~/.skillrunner/metrics/`)
   - Execution timestamps
   - Model usage statistics
   - Token counts and cost estimates
   - Performance metrics
   - Error logs

4. **Cache Data** (`~/.skillrunner/cache/`)
   - Cached execution results (with TTL)
   - Temporary processing files

### No Remote Data Collection

- **Skillrunner does not transmit any data to JBC Tech Solutions servers**
- No telemetry or analytics are sent to us
- No user behavior tracking
- No automatic error reporting

## Third-Party Services

When configured to use cloud LLM providers, data is transmitted directly to:

### Anthropic Claude API
- Prompts and responses are sent to Anthropic servers
- Subject to [Anthropic's Privacy Policy](https://www.anthropic.com/legal/privacy)
- API keys required (stored locally)

### OpenAI API
- Prompts and responses are sent to OpenAI servers
- Subject to [OpenAI's Privacy Policy](https://openai.com/policies/privacy-policy)
- API keys required (stored locally)

### Google AI (Gemini) API
- Prompts and responses are sent to Google servers
- Subject to [Google's Privacy Policy](https://policies.google.com/privacy)
- API keys required (stored locally)

### Ollama (Local)
- Runs entirely on your local machine
- No data leaves your system
- Open source under MIT License

## Data Security Recommendations

### API Key Management
- **Use environment variables** instead of storing API keys in config files:
  ```bash
  export ANTHROPIC_API_KEY="your-key"
  export OPENAI_API_KEY="your-key"
  export GOOGLE_API_KEY="your-key"
  ```
- Never commit API keys to version control
- Rotate API keys regularly

### File Permissions
- The `~/.skillrunner/` directory is created with user-only permissions (755)
- Sensitive files should be protected with appropriate filesystem permissions

### Data Retention
- Cached results expire based on configured TTL (default: 1 hour)
- Metrics are stored indefinitely unless manually deleted
- You can clear all data by removing the `~/.skillrunner/` directory

## User Rights

As an open-source tool running locally, you have complete control over your data:

1. **Access**: All data is stored in plaintext in `~/.skillrunner/`
2. **Modification**: You can edit any stored data directly
3. **Deletion**: Remove `~/.skillrunner/` to delete all data
4. **Portability**: All data is in standard formats (YAML, JSON)
5. **Transparency**: Source code is available for review

## Compliance Considerations

### GDPR Compliance
- No personal data is collected by JBC Tech Solutions
- All data remains under user control
- No data processors involved (except configured APIs)

### CCPA Compliance
- No sale of personal information
- No data sharing with third parties
- Full user control over data

### HIPAA Considerations
- Not HIPAA certified
- Should not be used for processing PHI without appropriate safeguards
- Users are responsible for compliance when processing sensitive data

## Children's Privacy

Skillrunner is not directed at children under 13. The tool should be used in compliance with applicable laws regarding children's data.

## Changes to This Policy

We may update this privacy policy as the software evolves. Check the "Last Updated" date for the latest version.

## Contact Information

For privacy-related questions about Skillrunner:

- **GitHub Issues**: https://github.com/jbctechsolutions/skillrunner/issues
- **Email**: privacy@jbctechsolutions.com

## Open Source Commitment

As an MIT-licensed open-source project, you can:
- Review the source code to verify privacy claims
- Fork and modify the software for your needs
- Contribute privacy enhancements

## Third-Party Forks

This privacy policy applies only to the official Skillrunner distribution from JBC Tech Solutions. Forks and modifications may have different privacy practices.
