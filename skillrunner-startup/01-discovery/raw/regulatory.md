# Regulatory Landscape: AI-Powered Workflow Tools for Consulting

**Research Date:** March 9, 2026
**Product:** Skillrunner -- CLI workflow engine for AI-assisted business workflows
**Target Market:** US-based fractional CTOs and agencies
**Key Use Cases:** Proposal generation (client business data), SOC2 audit workflows, translation pipelines (25,000 pages, church content), content generation for non-profits

---

## Current Regulations

### Federal (United States)

As of March 2026, the United States does **not** have a single comprehensive federal AI law. Regulation is a patchwork of state laws, federal agency guidance, and executive orders.

**Executive Order on AI (December 2025):** President Trump signed "Ensuring a National Policy Framework for Artificial Intelligence," proposing a uniform federal framework that could preempt state AI laws deemed inconsistent with federal policy. The scope of preemption remains legally untested and is expected to face court challenges. This creates significant regulatory uncertainty for 2026-2027.

**Existing Federal Laws That Apply:**
| Law | Relevance to Skillrunner | Requirements |
|-----|-------------------------|--------------|
| CAN-SPAM Act | If workflows generate marketing emails | Opt-out mechanisms, sender identification |
| COPPA | If any church/non-profit workflows involve minors' data | Parental consent for data collection from children under 13 |
| FTC Act (Section 5) | Deceptive AI practices | Truthful representations about AI capabilities |
| HIPAA | Only if consulting clients are healthcare entities | BAA with LLM providers, data encryption |

### State Laws (Effective 2026)

#### Colorado AI Act (Effective June 30, 2026)
- **Jurisdiction:** Colorado
- **Who it applies to:** Deployers and developers of "high-risk AI systems" (those making consequential decisions about consumers in employment, finance, housing, healthcare, education, insurance, legal services, government services)
- **Requirements:** Risk management policy, impact assessments, algorithmic discrimination prevention, consumer transparency/disclosure notices, documentation of AI decision-making
- **Relevance to Skillrunner:** Likely applies if SOC2 audit workflows or proposal generation involves decision-making that affects consumers. Skillrunner itself may be a "developer" if it provides high-risk AI system components.
- **Penalties:** Not yet finalized; enforcement by Colorado AG
- **Compliance cost estimate:** $15,000-$50,000 for impact assessments and policy development

#### California AI Transparency Act (SB 942, Effective January 1, 2026)
- **Jurisdiction:** California
- **Who it applies to:** "Covered Providers" -- AI systems publicly accessible in California with 1M+ monthly users
- **Requirements:** AI content detection tools, manifest and latent watermarks on AI-generated content
- **Relevance to Skillrunner:** Likely does NOT apply directly (CLI tool unlikely to reach 1M monthly users), but clients using Skillrunner outputs may need to comply
- **Penalties:** $5,000 per violation per day
- **Compliance cost estimate:** Low direct cost; documentation/disclosure recommended

#### California AB 2013 (Effective January 1, 2026)
- **Jurisdiction:** California
- **Who it applies to:** Developers of generative AI systems available for public use
- **Requirements:** Publish a "high-level summary" of training datasets
- **Relevance to Skillrunner:** Does NOT apply -- Skillrunner is a workflow tool, not a model developer. However, the LLM providers Skillrunner integrates with (Anthropic, OpenAI) must comply.

#### Texas RAIGA (Effective January 1, 2026)
- **Jurisdiction:** Texas
- **Who it applies to:** Government agencies, healthcare providers using consumer-facing AI
- **Requirements:** Disclosure when AI systems interact with consumers; bans on AI designed to incite self-harm, unlawfully discriminate, or produce deepfakes
- **Relevance to Skillrunner:** Low direct applicability unless consulting clients are Texas government agencies or healthcare providers
- **Compliance cost estimate:** Minimal for current use cases

#### CCPA/CPRA Amendments (Phased: January 1, 2026 - January 1, 2027)
- **Jurisdiction:** California
- **Who it applies to:** Businesses meeting CCPA thresholds ($25M+ revenue, 100K+ consumers' data, or 50%+ revenue from selling data)
- **2026 Requirements:** Risk assessments for six "significant risk" processing activities, cybersecurity audits for businesses processing data of 250K+ California consumers
- **2027 Requirements:** Pre-use notice, opt-out rights, and access rights for consumers regarding Automated Decision-Making Technology (ADMT)
- **Relevance to Skillrunner:** Applies if Skillrunner's clients are CCPA-threshold businesses using the tool for decisions affecting consumers. The ADMT provisions (2027) are particularly relevant for SOC2 audit workflows.
- **Compliance cost estimate:** $20,000-$75,000 (risk assessments + cybersecurity audit)

#### Multi-State Privacy Laws (Effective January 1, 2026)
Connecticut, Indiana, Kentucky, Oregon, Utah, and Virginia all implement privacy law amendments. By 2026, **15 states** have active comprehensive data privacy laws.

### European Union

#### EU AI Act
- **Status:** In phased implementation. Transparency rules effective August 2, 2026 (though the EC proposed extending high-risk AI rules to December 2027)
- **Who it applies to:** Any company placing AI systems on the EU market or using them within EU borders -- including US companies serving EU clients
- **Requirements (August 2026):** AI-generated content must be labeled (machine-readable marks + visible labels), transparency documentation, human oversight ("kill switch") for high-risk systems
- **Penalties:** Up to 7% of global revenue for prohibited AI violations; 3% for high-risk non-compliance
- **Relevance to Skillrunner:** Applies if any consulting client uses Skillrunner for EU-facing work or if the church translation content is distributed in the EU. The content labeling requirement directly affects generated proposals and translated content.
- **Compliance cost estimate:** $25,000-$100,000+ for full technical documentation, content labeling infrastructure, and human oversight mechanisms

#### GDPR (Ongoing)
- **Who it applies to:** Any entity processing personal data of EU residents
- **Key requirement for LLM usage:** Every API call sending personal data to an LLM is a "processing event" and potentially a cross-border data transfer subject to GDPR
- **Requirements:** Data Processing Agreements (DPAs) with LLM providers, legal basis for processing, data minimization, right to erasure
- **Penalties:** Up to 4% of global revenue or EUR 20M
- **Relevance to Skillrunner:** Applies if any client data processed through LLM APIs belongs to EU data subjects

---

## Data & Privacy

### Data Flow Risk Analysis for Skillrunner

```
Client Data (proposals, audits, church content)
    |
    v
Skillrunner CLI (local machine)
    |
    +---> Ollama (local) -- LOW RISK: data stays on-premises
    |
    +---> Anthropic API -- MEDIUM RISK: data transmitted to cloud
    |
    +---> OpenAI API -- MEDIUM RISK: data transmitted to cloud
```

### Requirements When Sending Client Data to LLM APIs

1. **Data Processing Agreements (DPAs):** Required if personal data is included. Both Anthropic and OpenAI offer enterprise DPAs.
2. **Encryption in Transit:** All major LLM API providers use HTTPS/TLS by default. Verify and document this.
3. **Data Retention Policies:**
   - Anthropic API: Does not use API inputs/outputs for training by default; offers zero-retention options on enterprise plans
   - OpenAI API: Does not use API data for training by default (since March 2023); enterprise plans offer zero-retention
   - Ollama (local): No data leaves the machine -- zero external retention risk
4. **Data Minimization:** Avoid sending more data than necessary. Implement PII stripping/masking before API calls where possible. Use techniques like token replacement for sensitive fields (SSNs, account numbers).
5. **Cross-Border Transfers:** Most LLM APIs are hosted in the US. For EU client data, ensure EU-US Data Privacy Framework adequacy or implement Standard Contractual Clauses (SCCs).
6. **Data Subject Rights:** Must be able to honor deletion requests. Ensure LLM provider contracts include data deletion provisions.

### SOC2 Compliance Considerations for AI Tools

SOC2 does not contain AI-specific requirements but applies to any technology provider handling customer data. Key Trust Service Criteria relevant to Skillrunner:

| Trust Service Criteria | AI-Specific Requirement | Skillrunner Implementation |
|----------------------|------------------------|---------------------------|
| **Security** | Access controls for AI systems, prompt injection protection | API key management, input validation |
| **Availability** | AI system uptime monitoring | Fallback to local models (Ollama) |
| **Processing Integrity** | Output validation, error logging, bias monitoring | Human review steps in workflows, audit logs |
| **Confidentiality** | Data classification, encryption, access restrictions | Data at rest encryption, role-based access |
| **Privacy** | PII handling, consent management, data minimization | PII masking before API calls, configurable data handling |

**For SOC2 audit workflows specifically:** Skillrunner must maintain traceable records of all data flows and AI system interactions, implement validation tools for AI output reliability, log errors in model outputs, establish corrective action protocols, and continuously assess model performance.

**Minimum SOC2 readiness cost:** $30,000-$50,000 for initial SOC2 Type I audit; $50,000-$100,000+ for ongoing SOC2 Type II compliance.

### Applicable Privacy Frameworks Summary

| Framework | Applies If | Key Requirement |
|-----------|-----------|-----------------|
| CCPA/CPRA | California consumers' data processed | Privacy notices, opt-out rights, risk assessments |
| GDPR | EU residents' data processed | DPAs, legal basis, data minimization, right to erasure |
| State privacy laws (15 states) | Data from those states processed | Varies; generally privacy notices, opt-out, data protection |
| COPPA | Minors' data (church youth programs) | Parental consent, data minimization |
| HIPAA | Healthcare client data | BAA, encryption, access controls |

---

## Industry-Specific Considerations

### Church/Non-Profit Data Handling

**Current State of AI Adoption in Churches:**
- Nearly 60% of church respondents use AI occasionally or regularly (as of 2025 surveys)
- 94% of churches say they either have no AI guidelines or are only discussing creating an AI policy
- This represents both a market opportunity and a compliance education need

**Data Types Requiring Protection:**
| Data Type | Sensitivity | Regulatory Concern |
|-----------|------------|-------------------|
| Member personal information (names, addresses, phone numbers) | High | State privacy laws, GDPR if international members |
| Donation/financial records | High | Financial privacy, state charity regulations |
| Counseling/pastoral care records | Very High | Potential HIPAA applicability, clergy-penitent privilege |
| Youth/children's data | Critical | COPPA, state child protection laws |
| Volunteer records | Medium | Employment-adjacent privacy laws |
| Sermon/teaching content | Low | Copyright, theological accuracy concerns |

**Key Compliance Requirements:**
1. **Privacy Policies:** Churches must clearly outline what data is collected, why, and who has access
2. **Data Anonymization:** Personal data (names, addresses, phone numbers) should be anonymized before being processed by AI systems
3. **Access Controls:** Only authorized personnel should handle sensitive data; implement MFA and role-based access
4. **Consent:** Explicit consent for processing member data through AI tools, especially for international congregations (GDPR)
5. **Ministerial Exception:** Some employment-related data may be exempt from certain regulations under the ministerial exception, but this does NOT extend to general member data privacy

**Recommendations for the 25,000-Page Translation Pipeline:**
- Ensure content does not contain embedded personal data (member names in testimonials, prayer requests, etc.)
- If translating content for international distribution (EU), comply with EU AI Act content labeling requirements by August 2026
- Implement human review process (aligns with ISO 18587 post-editing requirements)
- Maintain version control and audit trail of all translations

### Translation Quality Standards

#### ISO 18587:2017 -- Post-Editing of Machine Translation Output
- **Scope:** Requirements for human post-editing of machine-translated text
- **Key Requirements:**
  - Defines "light post-editing" (minimal corrections for gist) vs. "full post-editing" (human-quality output)
  - Post-editors must have: linguistic proficiency in source and target languages, subject matter expertise, familiarity with MT technology, knowledge of post-editing techniques
  - Mandatory quality control of raw MT output
  - Required competencies for review and QC personnel
- **Relevance:** Directly applicable to the 25,000-page translation pipeline. If using AI translation + human review, this is the standard to follow.
- **Certification cost:** $5,000-$15,000 for initial certification; ongoing audit costs

#### ISO 17100:2015 -- Translation Services
- **Scope:** General translation service requirements
- **Key Requirements:** Translator qualifications, revision process, project management
- **Relevance:** Baseline standard for any translation work; ISO 18587 builds on top of this for MT-specific workflows

#### Quality Claims and Liability
- Making claims about translation quality without following recognized standards creates liability risk
- For religious/theological content, translation errors can have reputational and pastoral consequences beyond legal liability
- Recommended: Implement a quality assurance framework based on ISO 18587 with domain-specific (theological) review steps

---

## Upcoming Changes

### 2026 (Remaining)

| Date | Change | Impact |
|------|--------|--------|
| June 30, 2026 | Colorado AI Act takes effect | Impact assessments required for high-risk AI deployers |
| August 2, 2026 | EU AI Act transparency rules | AI-generated content labeling mandatory |
| August 2, 2026 | EU Member States must establish AI regulatory sandboxes | Compliance guidance expected |
| Q2-Q3 2026 | EU guidelines on transparent AI systems published | Will clarify content labeling technical requirements |

### 2027

| Date | Change | Impact |
|------|--------|--------|
| January 1, 2027 | CCPA ADMT regulations take effect | Pre-use notice, opt-out rights for automated decision-making |
| December 2027 (proposed) | EU AI Act high-risk AI system rules | Full compliance for high-risk systems (extended from August 2026) |
| Throughout 2027 | Additional state AI laws expected | 5-10 more states expected to pass AI-specific legislation |

### Federal Preemption Uncertainty

The Trump executive order (December 2025) signals intent to preempt state AI laws, but:
- No federal AI law has been passed to implement preemption
- Legal challenges to preemption authority are expected
- Businesses should continue complying with state laws until federal preemption is legally established
- **Recommended posture:** Comply with the most restrictive applicable state law as a baseline

### Trend Analysis

Four themes run through nearly every AI regulation globally:
1. **Transparency** -- disclose AI use to affected parties
2. **Bias prevention** -- prevent algorithmic discrimination
3. **Data privacy** -- minimize and protect personal data
4. **Accountability** -- maintain human oversight and audit trails

These themes are unlikely to change regardless of the political or jurisdictional landscape. Building Skillrunner around these four pillars future-proofs against regulatory shifts.

---

## Compliance Cost Estimate

### Minimum Viable Compliance (MVP)

For a US-only, SMB-focused tool with no EU clients and no high-risk decision-making:

| Item | Cost | Timeline |
|------|------|----------|
| AI Acceptable Use Policy (internal) | $2,000-$5,000 (legal review) | 2-4 weeks |
| Privacy Policy & Terms of Service | $3,000-$8,000 | 2-4 weeks |
| DPAs with LLM providers (Anthropic, OpenAI) | $0 (use standard enterprise DPAs) | 1-2 weeks |
| Data handling documentation | $2,000-$5,000 (internal effort) | 2-4 weeks |
| PII detection/masking implementation | $5,000-$15,000 (engineering) | 4-8 weeks |
| Basic audit logging | $5,000-$10,000 (engineering) | 2-4 weeks |
| **Total MVP** | **$17,000-$43,000** | **3-6 months** |

### Full Compliance (Multi-State + EU Ready)

| Item | Cost | Timeline |
|------|------|----------|
| Everything in MVP | $17,000-$43,000 | 3-6 months |
| SOC2 Type I audit | $30,000-$50,000 | 6-12 months |
| SOC2 Type II (annual) | $50,000-$100,000/year | Ongoing |
| Colorado AI Act impact assessment | $15,000-$50,000 | 2-4 months |
| EU AI Act technical documentation | $25,000-$75,000 | 3-6 months |
| AI content labeling infrastructure | $10,000-$25,000 (engineering) | 4-8 weeks |
| GDPR compliance (DPAs, SCCs, DPIA) | $15,000-$30,000 | 2-4 months |
| ISO 18587 certification (translation) | $5,000-$15,000 | 2-3 months |
| Annual legal review and updates | $10,000-$25,000/year | Ongoing |
| Privacy risk assessments (CCPA) | $10,000-$25,000 | 2-3 months |
| Cybersecurity audit (if CCPA thresholds met) | $15,000-$40,000 | 3-6 months |
| **Total Full Compliance (Year 1)** | **$202,000-$478,000** | **12-18 months** |
| **Annual Ongoing** | **$85,000-$190,000/year** | Ongoing |

### Recommended Phased Approach

**Phase 1 (Months 1-3):** MVP compliance -- policies, DPAs, basic documentation. Cost: ~$20,000-$40,000
**Phase 2 (Months 4-9):** SOC2 Type I readiness, PII masking, audit logging. Cost: ~$50,000-$75,000
**Phase 3 (Months 10-18):** Full compliance expansion based on actual client requirements (EU, translation certification). Cost: Variable based on need.

---

## Risk Assessment

### Overall Regulatory Risk Level: **MEDIUM**

Skillrunner operates in a rapidly evolving regulatory environment, but its primary use case (CLI tool for consultants) places it at lower risk than consumer-facing AI applications.

### Key Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Client data leakage via LLM API** | Medium | High | PII masking, local model fallback (Ollama), enterprise API agreements, zero-retention options |
| **State law non-compliance (patchwork)** | Medium | Medium | Comply with most restrictive state; maintain regulatory tracking |
| **EU AI Act applicability (client's EU operations)** | Low-Medium | High | Content labeling infrastructure, documentation, GDPR compliance |
| **Colorado AI Act "high-risk" classification** | Low-Medium | Medium | Impact assessments, risk management documentation |
| **CCPA ADMT classification (2027)** | Medium | Medium | Consumer notice/opt-out infrastructure; plan now for 2027 deadline |
| **Translation quality liability** | Low | Medium-High | ISO 18587 compliance, human review process, quality disclaimers |
| **Federal preemption uncertainty** | Medium | Low-Medium | Continue state compliance; federal law would likely simplify, not complicate |
| **Church data sensitivity (pastoral/youth)** | Low | High | Strong data anonymization, COPPA compliance for youth data, clear data handling policies |
| **Copyright claims on AI-generated content** | Low-Medium | Medium | Disclose AI involvement, maintain human editorial control, document human contribution |
| **LLM provider policy changes** | Medium | Medium | Multi-provider support (already built), contractual protections, local model fallback |

### Risk Mitigation Priorities (Ranked)

1. **Implement PII masking/detection before API calls** -- addresses the highest-impact risk at lowest cost
2. **Maintain Ollama/local model support** -- provides a zero-data-exposure fallback for sensitive workflows
3. **Build audit logging into all workflows** -- satisfies SOC2, state laws, and EU AI Act simultaneously
4. **Publish clear data handling documentation** -- addresses transparency requirements across all frameworks
5. **Implement human review checkpoints in workflows** -- satisfies ISO 18587, EU AI Act human oversight, and reduces output quality risk

---

## Data Gaps

The following areas require additional research or monitoring:

1. **Federal AI legislation timeline:** No comprehensive federal AI law exists yet. The Trump executive order signals intent but lacks enforcement mechanism. Monitor congressional activity for federal AI bills in 2026-2027.

2. **State-by-state applicability mapping:** With 15+ state privacy laws active, a detailed analysis of which states' laws apply based on Skillrunner's specific client base and data flows is needed. This was not possible without knowing the exact client locations.

3. **LLM provider contractual terms (current):** Specific data retention, processing, and deletion terms for Anthropic and OpenAI API agreements should be reviewed by legal counsel. Enterprise agreements differ significantly from standard API terms.

4. **Religious organization exemptions:** Some state privacy laws include exemptions for religious organizations. A state-by-state analysis of these exemptions was not completed and could reduce compliance burden for church-focused workflows.

5. **Copyright status of AI-translated religious texts:** The copyright status of AI translations of copyrighted religious materials (e.g., specific Bible translations, denominational materials) requires legal analysis. Fair use arguments may apply differently to religious content.

6. **Insurance requirements:** Errors & Omissions (E&O) insurance and Cyber Liability insurance requirements for AI tool providers were not researched. These may be required by enterprise clients.

7. **Sector-specific AI guidance:** The FTC, SEC, and other federal agencies continue to issue AI-specific guidance. A systematic review of all agency guidance relevant to consulting and content generation was not completed.

8. **International translation regulations:** If the 25,000-page translation project involves languages/countries beyond the US and EU, additional regulatory research is needed (e.g., China's AI regulations, Brazil's LGPD).

9. **SOC2 AI-specific criteria evolution:** AICPA is developing AI-specific guidance for SOC2 audits. The current state of this guidance and expected publication timeline was not fully researched.

10. **Enforcement data:** No enforcement actions specifically targeting AI workflow tools or consulting AI usage were found. This is an emerging area; precedent-setting enforcement actions are expected in 2026-2027 as state laws take effect.

---

## Sources

- [White & Case -- AI Watch: Global Regulatory Tracker (US)](https://www.whitecase.com/insight-our-thinking/ai-watch-global-regulatory-tracker-united-states)
- [Wilson Sonsini -- 2026 Year in Preview: AI Regulatory Developments](https://www.wsgr.com/en/insights/2026-year-in-preview-ai-regulatory-developments-for-companies-to-watch-out-for.html)
- [CPO Magazine -- 2026 AI Legal Forecast](https://www.cpomagazine.com/data-protection/2026-ai-legal-forecast-from-innovation-to-compliance/)
- [Gunderson Dettmer -- 2026 AI Laws Update](https://www.gunder.com/en/news-insights/insights/2026-ai-laws-update-key-regulations-and-practical-guidance)
- [Drata -- Artificial Intelligence Regulations: State and Federal AI Laws 2026](https://drata.com/blog/artificial-intelligence-regulations-state-and-federal-ai-laws-2026)
- [Wiley -- Five Privacy Checkpoints to Start 2026](https://www.wiley.law/alert-Five-Privacy-Checkpoints-to-Start-2026)
- [Privacy World -- Primer on 2026 Consumer Privacy, AI, and Cybersecurity Laws](https://www.privacyworld.blog/2026/01/primer-on-2026-consumer-privacy-ai-and-cybersecurity-laws/)
- [EU AI Act Official](https://artificialintelligenceact.eu/)
- [Sembly AI -- EU AI Act Overview](https://www.sembly.ai/blog/eu-ai-act-overview-and-impact-on-ai-tools/)
- [LegalNodes -- EU AI Act 2026 Updates](https://www.legalnodes.com/article/eu-ai-act-2026-updates-compliance-requirements-and-business-risks)
- [DataGuard -- EU AI Act Timeline](https://www.dataguard.com/eu-ai-act/timeline)
- [ECIJA -- Companies Required to Label AI-Generated Content](https://www.ecija.com/en/news-and-insights/las-empresas-deberan-etiquetar-los-contenidos-generados-por-ia-a-partir-de-agosto-de-2026/)
- [Kirkland & Ellis -- EU Code of Practice on AI-Generated Content Transparency](https://www.kirkland.com/publications/kirkland-alert/2026/02/illuminating-ai-the-eus-first-draft-code-of-practice-on-transparency-for-ai)
- [Lasso Security -- LLM Data Privacy](https://www.lasso.security/blog/llm-data-privacy)
- [Rohan Paul -- Data Security for Third-Party LLM APIs](https://www.rohan-paul.com/p/data-security-and-privacy-precautions)
- [Skyflow -- Private LLMs](https://www.skyflow.com/post/private-llms-data-protection-potential-and-limitations)
- [Userfront -- SOC 2 Compliance in the Age of AI](https://userfront.com/blog/soc-2-ai-compliance)
- [CompassITC -- SOC 2 for AI Platforms](https://www.compassitc.com/blog/achieving-soc-2-compliance-for-artificial-intelligence-ai-platforms)
- [Moss Adams -- AI Controls in SOC 2 Reports](https://www.mossadams.com/articles/2025/12/ai-controls-for-soc-2-reports)
- [Linford & Co -- Auditing AI Platforms: SOC 2 Considerations](https://linfordco.com/blog/soc-2-audit-considerations-ai-ml-platforms/)
- [NTEN -- AI For Nonprofits Resource Hub](https://www.nten.org/learn/resource-hubs/artificial-intelligence)
- [National Council of Nonprofits -- Data Privacy for Nonprofits](https://www.councilofnonprofits.org/articles/earning-trust-imperative-data-privacy-nonprofits)
- [Independent Sector -- Data Privacy and AI Resources for Nonprofits](https://independentsector.org/resource/data-privacy-and-artificial-intelligence-resources-for-nonprofits/)
- [TUV SUD -- ISO 17100 & ISO 18587 Certifications](https://www.tuvsud.com/en-us/services/auditing-and-system-certification/iso-17100)
- [Adverbum -- ISO Standards for Localization 2026](https://www.adverbum.com/post/iso-compliant-localization-providers-guide-2026)
- [King & Spalding -- New State AI Laws 2026](https://www.kslaw.com/news-and-insights/new-state-ai-laws-are-effective-on-january-1-2026-but-a-new-executive-order-signals-disruption)
- [BDO -- CCPA Updates & New State Privacy Laws for 2026](https://www.bdo.com/insights/advisory/2026-is-a-pivotal-year-for-privacy)
- [Axiom Law -- State Privacy Laws: 2026 Changes & Compliance](https://www.axiomlaw.com/blog/state-privacy-laws)
- [CPPA -- CCPA Updates, ADMT, and Cybersecurity Regulations](https://cppa.ca.gov/regulations/ccpa_updates.html)
- [Wiley -- California Finalizes Pivotal CCPA Regulations on AI](https://www.wiley.law/alert-California-Finalizes-Pivotal-CCPA-Regulations-on-AI-Cyber-Audits-and-Risk-Governance)
