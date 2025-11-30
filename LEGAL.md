# Legal Documentation for Skillrunner v1.0

## Executive Summary

This document provides comprehensive legal guidance for the Skillrunner v1.0 release, addressing licensing, compliance, privacy, and regulatory considerations.

## License Compliance

### Primary License
- **License**: MIT License
- **Copyright Holder**: JBC Tech Solutions (2025)
- **File**: LICENSE (created)

### Dependencies License Compatibility

All dependencies are compatible with MIT license:

| Dependency | License | Compatibility |
|------------|---------|---------------|
| spf13/cobra | Apache 2.0 | Compatible |
| spf13/pflag | BSD 3-Clause | Compatible |
| yaml.v3 | MIT/Apache 2.0 | Compatible |
| inconshreveable/mousetrap | Apache 2.0 | Compatible |

**No GPL or copyleft licenses detected** - Safe for proprietary use.

## Third-Party API Compliance

### Service Terms Requirements

1. **Anthropic Claude API**
   - Requires API key authentication
   - Subject to Anthropic's Terms of Service
   - No data retention for API calls by default
   - Rate limits apply

2. **OpenAI API**
   - Requires API key authentication
   - Subject to OpenAI's Terms of Use
   - 30-day data retention policy
   - Usage policies prohibit certain content

3. **Google AI (Gemini) API**
   - Requires API key authentication
   - Subject to Google's Terms of Service
   - Data may be used for service improvement
   - Geographic restrictions may apply

4. **Ollama (Local)**
   - MIT Licensed - fully compatible
   - No external API calls
   - Complete data sovereignty

### Compliance Recommendations

1. **User Notification**: Inform users about third-party API usage
2. **API Key Security**: Mandate environment variables for API keys
3. **Data Processing**: Document that API calls go directly to providers
4. **Terms Pass-Through**: Users must comply with provider terms

## Data Privacy Compliance

### GDPR Compliance Assessment

**Status: Generally Compliant** (with recommendations)

Strengths:
- No data collection by JBC Tech Solutions
- Local data storage only
- User has full control over data
- Transparent about data handling
- Open source for verification

Recommendations:
1. Add explicit consent mechanism for API usage
2. Implement data retention controls
3. Add data export functionality
4. Document data flows clearly

### CCPA Compliance Assessment

**Status: Compliant**

- No sale of personal information
- No data sharing with third parties
- Full user control over data
- Transparent privacy practices

### HIPAA Considerations

**Status: Not HIPAA Certified**

Warning: Should not be used for PHI without:
- Business Associate Agreements with API providers
- Encryption at rest and in transit
- Access controls and audit logging
- Security risk assessment

## Export Control Compliance

### Classification
- **ECCN**: Likely 5D002 (publicly available encryption)
- **License Exception**: TSU (Technology and Software Unrestricted)

### Requirements
1. No export to embargoed countries
2. No use by denied persons/entities
3. Self-classification recommended for commercial distribution

### Recommendations
- Add export control notice to README
- Document encryption capabilities
- Consider legal review for international distribution

## Intellectual Property

### Copyright
- All original code copyright JBC Tech Solutions
- Proper attribution for dependencies maintained
- No patent claims or restrictions

### Trademark
- "Skillrunner" name should be trademark searched
- No use of third-party trademarks except as required

## Software Bill of Materials (SBOM)

### Recommendations
1. Generate SBOM using `go mod vendor` and tools like:
   ```bash
   go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
   cyclonedx-gomod mod -json -output sbom.json
   ```

2. Include in releases for enterprise users
3. Update with each release

## Required Documentation

### Files Created
1. **LICENSE** - MIT License with proper copyright
2. **NOTICE** - Third-party attributions and notices
3. **PRIVACY.md** - Privacy policy and data handling
4. **SECURITY.md** - Security policy and vulnerability reporting
5. **LEGAL.md** - This comprehensive legal guide

### Files Needed (Existing/Updated)
1. **CONTRIBUTING.md** - Already exists, consider adding CLA section
2. **CODE_OF_CONDUCT.md** - Recommended for community management

## Risk Assessment

### Low Risk Areas
- License compatibility (all compatible)
- Open source compliance (proper attributions)
- Local data processing (user controlled)

### Medium Risk Areas
- API terms compliance (user responsibility)
- Export control (standard encryption)
- Data privacy (depends on usage)

### Mitigation Strategies
1. Clear documentation of user responsibilities
2. Security best practices in documentation
3. Regular dependency updates
4. Vulnerability disclosure process

## Recommendations for v1.0 Release

### Required Actions
1. Add LICENSE file (MIT)
2. Add NOTICE file with attributions
3. Add PRIVACY.md policy
4. Add SECURITY.md policy
5. Review and merge CONTRIBUTING.md updates
6. Generate and include SBOM

### Recommended Actions
1. Add CODE_OF_CONDUCT.md
2. Implement API usage consent mechanism
3. Add data retention configuration
4. Include enterprise compliance guide
5. Consider trademark search for "Skillrunner"

### Legal Notices to Include

Add to README.md:
```markdown
## Legal Notice

This software is provided under the MIT License. See LICENSE for details.

Use of third-party APIs is subject to their respective terms of service.
Users are responsible for compliance with applicable laws and regulations.

This software includes cryptographic functionality that may be subject to
export controls. Users are responsible for compliance with export laws.

Not intended for use with protected health information (PHI) or in
regulated environments without appropriate safeguards.
```

## Compliance Checklist for Release

- [x] MIT License file added
- [x] NOTICE file with attributions
- [x] Privacy policy documented
- [x] Security policy established
- [x] All dependencies license-compatible
- [ ] SBOM generated
- [ ] Export control notice added
- [ ] API compliance documented
- [ ] Contributing guidelines updated
- [ ] Legal notice in README

## Support and Questions

For legal questions about Skillrunner:
- Open source licensing: Review LICENSE and NOTICE
- Privacy concerns: See PRIVACY.md
- Security issues: Follow SECURITY.md process
- Commercial use: MIT license permits commercial use

## Disclaimer

This legal documentation is provided for informational purposes only and does not constitute legal advice. Organizations should consult with their legal counsel for specific compliance requirements.

---
Document Version: 1.0
Last Updated: November 2025
Prepared for: Skillrunner v1.0 Release
