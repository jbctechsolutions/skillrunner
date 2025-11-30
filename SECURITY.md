# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Currently supported versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| 0.x.x   | :x:                |

## Reporting a Vulnerability

We take the security of Skillrunner seriously. If you discover a security vulnerability, please follow these steps:

### 1. Do NOT Create a Public Issue

Security vulnerabilities should not be reported via public GitHub issues.

### 2. Report Privately

Please report security vulnerabilities by emailing: security@jbctechsolutions.com

Include the following information:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Any suggested fixes (optional)

### 3. Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 5 business days
- **Resolution Target**: Within 30 days for critical issues

## Security Considerations

### API Key Management

**Best Practices:**
- Use environment variables for API keys
- Never hardcode API keys in configuration files
- Never commit API keys to version control
- Use separate API keys for development and production
- Rotate API keys regularly
- Monitor API usage for anomalies

**Secure Configuration Example:**
```bash
# Set via environment variables (recommended)
export ANTHROPIC_API_KEY="sk-..."
export OPENAI_API_KEY="sk-..."
export GOOGLE_API_KEY="..."

# Run Skillrunner
sr run <skill> <request>
```

### Local Data Security

**File Permissions:**
- Configuration directory (`~/.skillrunner/`) created with 755 permissions
- Sensitive files should be protected with 600 permissions:
  ```bash
  chmod 600 ~/.skillrunner/config.yaml
  ```

**Data Encryption:**
- API keys stored in plaintext (use environment variables instead)
- Consider full-disk encryption for sensitive deployments
- Cache files are not encrypted by default

### Third-Party Dependencies

**Dependency Security:**
- Regularly update dependencies: `go get -u ./...`
- Review dependency licenses for compatibility
- Monitor for security advisories

**Current Dependencies:**
- `spf13/cobra` - Apache 2.0 License
- `yaml.v3` - MIT/Apache 2.0 License
- `spf13/pflag` - BSD 3-Clause License
- `inconshreveable/mousetrap` - Apache 2.0 License

### Network Security

**API Communications:**
- All API calls use HTTPS
- TLS verification enabled by default
- No custom certificate validation bypass

**Local Ollama:**
- Communicates over localhost only (default: http://localhost:11434)
- No authentication by default (secure your Ollama instance separately)

### Input Validation

**Prompt Injection Protection:**
- Be cautious with untrusted input to skills
- Review generated code before execution
- Validate file paths and URLs

**Command Injection:**
- No direct shell command execution from user input
- Git operations use safe command construction

## Security Features

### Built-in Protections

1. **No Telemetry**: No data sent to JBC Tech Solutions
2. **Local First**: Prefer local models over cloud APIs
3. **Isolation**: Each skill runs independently
4. **Validation**: Input validation for critical operations

### Recommended Configurations

**For Sensitive Environments:**

```yaml
# ~/.skillrunner/config.yaml
settings:
  api_timeout: 30s
  max_retries: 1
  cache_ttl: 5m  # Short cache lifetime

profiles:
  secure:
    description: "Local-only execution"
    priority:
      - ollama/qwen2.5:14b
    fallback: false  # Don't fall back to cloud
```

## Known Security Considerations

### Current Limitations

1. **API Keys in Memory**: API keys are held in memory during execution
2. **Cache Files**: Execution results cached in plaintext
3. **Metrics Logging**: Usage data stored in plaintext JSON
4. **No Authentication**: Local API has no built-in authentication

### Mitigation Strategies

1. **Use environment variables** for all sensitive configuration
2. **Implement disk encryption** for `~/.skillrunner/` directory
3. **Regular cleanup** of cache and metrics data
4. **Network isolation** for sensitive deployments
5. **Code review** all generated outputs before execution

## Security Checklist

Before deploying Skillrunner in production:

- [ ] API keys stored in environment variables, not config files
- [ ] File permissions set appropriately (600 for sensitive files)
- [ ] Dependencies updated to latest versions
- [ ] Network access restricted if needed
- [ ] Disk encryption enabled for sensitive data
- [ ] Regular security updates applied
- [ ] Audit logs reviewed periodically
- [ ] Backup strategy in place
- [ ] Incident response plan documented

## Vulnerability Disclosure

We follow responsible disclosure practices:

1. **Private Disclosure**: 30-day window for fixes
2. **CVE Assignment**: For qualifying vulnerabilities
3. **Public Disclosure**: After patch release
4. **Credit**: Security researchers credited (with permission)

## Contact

**Security Team Email**: security@jbctechsolutions.com
**PGP Key**: Available at https://jbctechsolutions.com/security.asc

## Compliance Notes

### Export Control

This software includes cryptographic functionality and may be subject to export controls in various jurisdictions. Users are responsible for compliance with:

- U.S. Export Administration Regulations (EAR)
- Local export/import laws
- Restrictions on cryptographic software

### AI/ML Considerations

When using AI models, consider:
- Data privacy regulations (GDPR, CCPA)
- AI-specific regulations in your jurisdiction
- Model bias and fairness requirements
- Transparency and explainability requirements