# Skillrunner Skills Guide

## Table of Contents

1. [What are Skills?](#what-are-skills)
2. [Skill Structure](#skill-structure)
3. [Phase Configuration](#phase-configuration)
4. [Dependencies & DAG Execution](#dependencies--dag-execution)
5. [Routing Configuration](#routing-configuration)
6. [Built-in Skills](#built-in-skills)
7. [Creating Custom Skills](#creating-custom-skills)
8. [Importing Skills](#importing-skills)
9. [Best Practices](#best-practices)

---

## What are Skills?

**Skills** are multi-phase AI workflow definitions written in YAML. They allow you to define complex, structured AI tasks that execute in a coordinated sequence based on dependencies. Each skill consists of one or more phases that can run independently or depend on the output of other phases, forming a Directed Acyclic Graph (DAG) for execution.

### Key Features

- **Multi-phase execution**: Break complex tasks into discrete, manageable steps
- **Dependency management**: Phases can depend on outputs from previous phases
- **Parallel execution**: Independent phases execute concurrently for efficiency
- **Variable substitution**: Access input and phase outputs using template variables
- **Routing profiles**: Control model selection (cheap, balanced, premium) per phase
- **Metadata support**: Tag, categorize, and document your skills

### Use Cases

- Code review workflows
- Test generation pipelines
- Documentation generation
- Multi-step analysis tasks
- Quality assurance processes
- Any structured AI workflow requiring multiple coordinated steps

---

## Skill Structure

A skill definition is a YAML file with the following top-level structure:

```yaml
id: string              # Required: Unique identifier (e.g., "code-review")
name: string            # Required: Human-readable name
version: string         # Optional: Semantic version (e.g., "1.0.0")
description: string     # Optional: Multi-line description
phases: []              # Required: Array of phase definitions (minimum 1)
routing: {}             # Optional: Routing configuration
metadata: {}            # Optional: Additional metadata (tags, author, etc.)
```

### Complete Example

```yaml
id: my-skill
name: My Custom Skill
version: "1.0.0"
description: |
  A comprehensive skill that performs multi-phase analysis.
  This is a longer description that can span multiple lines.

routing:
  default_profile: balanced
  generation_model: claude
  review_model: claude
  fallback_model: "llama3.2:3b"
  max_context_tokens: 16384

phases:
  - id: phase-1
    name: First Phase
    prompt_template: "Analyze: {{.input}}"
    routing_profile: cheap
    max_tokens: 2048
    temperature: 0.3

  - id: phase-2
    name: Second Phase
    prompt_template: |
      Previous result: {{.phases.phase-1.output}}
      Input: {{.input}}
    depends_on:
      - phase-1
    routing_profile: balanced
    max_tokens: 4096
    temperature: 0.5

metadata:
  author: your-name
  category: analysis
  license: MIT
  tags:
    - analysis
    - review
```

### Field Descriptions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier for the skill. Use lowercase with hyphens (e.g., "code-review") |
| `name` | string | Yes | Human-readable display name |
| `version` | string | No | Semantic version string (e.g., "1.0.0") |
| `description` | string | No | Detailed description of what the skill does. Can use YAML multi-line format |
| `phases` | array | Yes | List of phase definitions (minimum 1 required) |
| `routing` | object | No | Routing configuration for model selection |
| `metadata` | map | No | Arbitrary key-value metadata for categorization and documentation |

---

## Phase Configuration

Phases are the building blocks of skills. Each phase represents a discrete step in the workflow with its own prompt template, model preferences, and execution parameters.

### Phase Structure

```yaml
phases:
  - id: string              # Required: Unique phase identifier
    name: string            # Required: Human-readable phase name
    prompt_template: string # Required: Prompt with variable substitution
    routing_profile: string # Optional: cheap|balanced|premium (default: balanced)
    depends_on: []          # Optional: List of phase IDs this phase depends on
    max_tokens: int         # Optional: Maximum output tokens (default: 4096)
    temperature: float      # Optional: LLM temperature 0.0-2.0 (default: 0.7)
```

### Phase Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | - | Unique identifier within the skill. Use lowercase with hyphens |
| `name` | string | Yes | - | Display name for the phase |
| `prompt_template` | string | Yes | - | Template with variable substitution (see below) |
| `routing_profile` | string | No | `balanced` | Model quality tier: `cheap`, `balanced`, or `premium` |
| `depends_on` | array | No | `[]` | List of phase IDs that must complete before this phase |
| `max_tokens` | int | No | `4096` | Maximum tokens for the phase output (must be positive) |
| `temperature` | float | No | `0.7` | LLM temperature between 0.0 (deterministic) and 2.0 (creative) |

### Prompt Template Variables

Prompt templates support variable substitution using Go template syntax:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.input}}` | The original input provided to the skill | User's code or text |
| `{{.phases.<phase-id>.output}}` | Output from a specific phase | `{{.phases.analyze.output}}` |
| `{{.phases.<phase-id>}}` | Shorthand for phase output | `{{.phases.analyze}}` |

### Phase Examples

**Simple Phase (No Dependencies)**

```yaml
- id: analyze
  name: Code Analysis
  prompt_template: |
    Analyze the following code for potential issues:

    {{.input}}

    Focus on:
    1. Logic errors
    2. Performance issues
    3. Code style
  routing_profile: cheap
  max_tokens: 2048
  temperature: 0.3
```

**Dependent Phase (Uses Previous Output)**

```yaml
- id: security
  name: Security Review
  prompt_template: |
    Based on the code analysis:
    {{.phases.analyze}}

    Original code:
    {{.input}}

    Identify security vulnerabilities including:
    - Injection attacks
    - Authentication issues
    - Data exposure risks
  depends_on:
    - analyze
  routing_profile: premium
  max_tokens: 4096
  temperature: 0.2
```

**Multi-Dependency Phase**

```yaml
- id: report
  name: Final Report
  prompt_template: |
    Generate a comprehensive report combining:

    ## Code Analysis
    {{.phases.analyze.output}}

    ## Security Findings
    {{.phases.security.output}}

    ## Test Results
    {{.phases.testing.output}}

    Provide an executive summary with prioritized recommendations.
  depends_on:
    - analyze
    - security
    - testing
  routing_profile: balanced
  max_tokens: 8192
  temperature: 0.5
```

### Routing Profiles

Each phase can specify a routing profile to control model selection:

| Profile | Use Case | Cost | Quality |
|---------|----------|------|---------|
| `cheap` | Simple analysis, structure extraction | Low | Good |
| `balanced` | General-purpose tasks | Medium | Very Good |
| `premium` | Complex reasoning, high-quality output | High | Excellent |

**Guidelines:**
- Use `cheap` for structural analysis, parsing, or simple transformations
- Use `balanced` for most generation and analysis tasks
- Use `premium` for security reviews, complex reasoning, or final outputs

---

## Dependencies & DAG Execution

Skills execute phases as a Directed Acyclic Graph (DAG) based on the `depends_on` relationships.

### How Dependencies Work

1. **Independent phases** (no dependencies) execute in parallel
2. **Dependent phases** wait for all their dependencies to complete
3. **Outputs are accessible** via template variables once a phase completes
4. **Cycle detection** ensures no circular dependencies exist

### Execution Example

```yaml
phases:
  # Phase 1: Runs immediately (no dependencies)
  - id: analyze
    name: Analysis
    prompt_template: "Analyze: {{.input}}"
    # No depends_on - executes first

  # Phase 2: Runs immediately (no dependencies)
  - id: lint
    name: Linting
    prompt_template: "Check style: {{.input}}"
    # No depends_on - executes in parallel with 'analyze'

  # Phase 3: Waits for 'analyze' to complete
  - id: security
    name: Security Check
    prompt_template: "Review security based on {{.phases.analyze}}"
    depends_on:
      - analyze

  # Phase 4: Waits for ALL dependencies
  - id: report
    name: Final Report
    prompt_template: |
      Combine results:
      Analysis: {{.phases.analyze}}
      Linting: {{.phases.lint}}
      Security: {{.phases.security}}
    depends_on:
      - analyze
      - lint
      - security
```

### Execution Timeline

```
Time →

T0:  [analyze] [lint]        # Both start immediately (no deps)
T1:  [security]              # Starts after 'analyze' completes
T2:  [report]                # Starts after all three complete
```

### Validation Rules

The skill loader validates dependencies to ensure:

1. **All dependencies exist**: Referenced phase IDs must be defined
2. **No cycles**: Dependencies must form a DAG (no circular references)
3. **Phase IDs are unique**: Each phase has a distinct identifier

**Invalid Example (Cycle):**

```yaml
phases:
  - id: a
    depends_on: [b]  # Depends on b
  - id: b
    depends_on: [a]  # Depends on a - CYCLE!
```

---

## Routing Configuration

Routing configuration controls how Skillrunner selects AI models for execution. It can be specified at both the skill level (default for all phases) and per-phase level.

### Routing Structure

```yaml
routing:
  default_profile: string     # cheap|balanced|premium (default: balanced)
  generation_model: string    # Preferred model for generation
  review_model: string        # Preferred model for review phases
  fallback_model: string      # Fallback when primary unavailable
  max_context_tokens: int     # Maximum context window (default: 4096)
```

### Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `default_profile` | string | No | `balanced` | Default routing profile for phases |
| `generation_model` | string | No | - | Model preference for generation tasks |
| `review_model` | string | No | - | Model preference for review/analysis tasks |
| `fallback_model` | string | No | - | Model to use when primary is unavailable |
| `max_context_tokens` | int | No | `4096` | Maximum tokens in context window |

### Routing Examples

**Cost-Optimized (Cheap Profile)**

```yaml
routing:
  default_profile: cheap
  max_context_tokens: 8192
  fallback_model: "llama3.2:3b"
```

**Quality-First (Premium Profile)**

```yaml
routing:
  default_profile: premium
  generation_model: claude
  review_model: claude
  max_context_tokens: 32000
```

**Balanced Approach**

```yaml
routing:
  default_profile: balanced
  generation_model: claude
  fallback_model: gpt-4
  max_context_tokens: 16384
```

### Profile Override

Phases can override the skill-level default:

```yaml
routing:
  default_profile: balanced  # Skill default

phases:
  - id: quick-scan
    routing_profile: cheap   # Override to cheap
    # ...

  - id: deep-analysis
    routing_profile: premium # Override to premium
    # ...

  - id: standard-task
    # Uses 'balanced' from skill default
    # ...
```

---

## Built-in Skills

Skillrunner includes 10 pre-built skills ready to use out of the box, covering common development workflows.

### 1. Code Review (`code-review`)

**Purpose:** Comprehensive multi-phase code review with security analysis

**Phases:**
1. `analyze` - Code pattern analysis and quality assessment
2. `security` - Security vulnerability scanning
3. `report` - Consolidated review report generation

**Usage:**
```bash
sr run code-review --input mycode.go
```

**Configuration:**
```yaml
id: code-review
name: Code Review
version: "1.0.0"
routing:
  default_profile: premium  # Quality-first for thorough reviews
  max_context_tokens: 32000

phases:
  - id: analyze
    name: Code Pattern Analysis
    routing_profile: premium
    temperature: 0.3

  - id: security
    name: Security Analysis
    depends_on: [analyze]
    routing_profile: premium
    temperature: 0.2

  - id: report
    name: Review Summary Report
    depends_on: [analyze, security]
    routing_profile: premium
    temperature: 0.4
```

**Output Includes:**
- Executive summary
- Code quality score (1-10)
- Prioritized issue list with severity levels
- Security findings with OWASP Top 10 coverage
- Actionable recommendations
- Approval recommendation

---

### 2. Test Generation (`test-gen`)

**Purpose:** Generate comprehensive unit tests from source code

**Phases:**
1. `analyze` - Code structure analysis for test planning
2. `generate` - Unit test generation with mocks
3. `validate` - Test quality and coverage validation

**Usage:**
```bash
sr run test-gen --input mycode.go
```

**Configuration:**
```yaml
id: test-gen
name: Test Generation
version: "1.0.0"
routing:
  default_profile: balanced
  max_context_tokens: 16384

phases:
  - id: analyze
    name: Code Analysis
    routing_profile: cheap
    temperature: 0.3

  - id: generate
    name: Test Generation
    depends_on: [analyze]
    routing_profile: balanced
    temperature: 0.5
    max_tokens: 8192

  - id: validate
    name: Test Validation
    depends_on: [analyze, generate]
    routing_profile: balanced
    temperature: 0.3
```

**Features:**
- Identifies all testable units
- Creates mocks for dependencies
- Tests happy paths, edge cases, and errors
- Table-driven tests where appropriate
- Coverage analysis and quality scoring

**Supported Languages:**
- Go
- Python
- JavaScript/TypeScript
- Java
- Rust

---

### 3. Documentation Generator (`doc-gen`)

**Purpose:** Generate comprehensive documentation from source code

**Phases:**
1. `analyze` - Extract code structure for documentation
2. `generate` - Generate formatted documentation

**Usage:**
```bash
sr run doc-gen --input mycode.go
```

**Configuration:**
```yaml
id: doc-gen
name: Documentation Generator
version: "1.0.0"
routing:
  default_profile: cheap      # Cost-optimized
  max_context_tokens: 8192
  fallback_model: "llama3.2:3b"

phases:
  - id: analyze
    name: Code Structure Analysis
    routing_profile: cheap
    temperature: 0.3
    max_tokens: 2048

  - id: generate
    name: Documentation Generation
    depends_on: [analyze]
    routing_profile: cheap
    temperature: 0.5
    max_tokens: 4096
```

**Output Includes:**
- Module/file overview
- API reference for all public functions
- Type definitions and descriptions
- Usage examples
- Performance considerations
- Markdown-formatted output

---

### 4. Changelog Generator (`changelog`)

**Purpose:** Generate changelog entries from git history

**Phases:** 2
1. `analyze` - Analyze git commits and categorize changes
2. `format` - Format into Keep a Changelog format

**Usage:**
```bash
sr run changelog "Generate changelog for v1.2.0"
```

**Output Includes:**
- Categorized changes (Added, Changed, Fixed, etc.)
- Commit attribution
- Breaking change warnings

---

### 5. Commit Message Generator (`commit-msg`)

**Purpose:** Generate conventional commit messages from staged changes

**Phases:** 2
1. `analyze` - Analyze diff and understand changes
2. `generate` - Generate conventional commit message

**Usage:**
```bash
sr run commit-msg "$(git diff --staged)"
```

**Output Includes:**
- Type prefix (feat, fix, refactor, etc.)
- Scope identification
- Breaking change indicators
- Detailed body when needed

---

### 6. PR Description Generator (`pr-description`)

**Purpose:** Generate pull request descriptions from branch changes

**Phases:** 3
1. `analyze` - Analyze commit history and file changes
2. `summarize` - Create summary of changes
3. `format` - Format as PR description

**Usage:**
```bash
sr run pr-description "Describe changes for auth feature"
```

**Output Includes:**
- Summary of changes
- List of modified files
- Testing checklist
- Breaking change notes

---

### 7. Lint Fix (`lint-fix`)

**Purpose:** Identify and fix linting errors

**Phases:** 3
1. `identify` - Identify linting issues
2. `fix` - Apply fixes
3. `verify` - Verify fixes don't introduce new issues

**Usage:**
```bash
sr run lint-fix "Fix linting errors in auth.go"
```

**Output Includes:**
- List of identified issues
- Applied fixes
- Verification results

---

### 8. Test Fix (`test-fix`)

**Purpose:** Debug and fix failing tests

**Phases:** 3
1. `analyze` - Analyze test failures
2. `diagnose` - Identify root cause
3. `fix` - Suggest or apply fixes

**Usage:**
```bash
sr run test-fix "Fix TestUserAuth failure"
```

**Output Includes:**
- Failure analysis
- Root cause explanation
- Suggested fix with code

---

### 9. Refactor (`refactor`)

**Purpose:** Apply refactoring patterns to code

**Phases:** 3
1. `analyze` - Analyze code structure
2. `transform` - Apply refactoring
3. `validate` - Validate no behavior changes

**Usage:**
```bash
sr run refactor "Extract method from handleAuth"
```

**Supports patterns:**
- Extract function/method
- Inline function
- Rename variable/function
- Simplify conditionals

---

### 10. Issue Breakdown (`issue-breakdown`)

**Purpose:** Break down large issues into smaller subtasks

**Phases:** 2
1. `analyze` - Analyze issue complexity
2. `decompose` - Create subtask breakdown

**Usage:**
```bash
sr run issue-breakdown "Break down: Add user authentication"
```

**Output Includes:**
- Subtask list with estimates
- Dependencies between tasks
- Suggested implementation order

---

## Creating Custom Skills

Follow these steps to create your own skill:

### Step 1: Create the YAML File

Create a new file in the `skills/` directory:

```bash
touch skills/my-skill.yaml
```

### Step 2: Define the Skill Metadata

Start with the required fields:

```yaml
id: my-custom-skill
name: My Custom Skill
version: "1.0.0"
description: |
  This skill performs a custom workflow for my specific use case.
  It demonstrates how to create multi-phase skills.
```

### Step 3: Add Phases

Define your workflow phases with dependencies:

```yaml
phases:
  # Phase 1: Initial analysis (runs first)
  - id: initial-analysis
    name: Initial Analysis
    prompt_template: |
      Analyze the following input and extract key information:

      {{.input}}

      Provide a structured summary.
    routing_profile: cheap
    max_tokens: 2048
    temperature: 0.3

  # Phase 2: Deep dive (depends on phase 1)
  - id: deep-analysis
    name: Deep Analysis
    prompt_template: |
      Based on the initial analysis:
      {{.phases.initial-analysis}}

      Perform a deeper analysis focusing on:
      1. Complex patterns
      2. Hidden relationships
      3. Potential issues
    depends_on:
      - initial-analysis
    routing_profile: balanced
    max_tokens: 4096
    temperature: 0.5

  # Phase 3: Final output (depends on both)
  - id: synthesis
    name: Synthesis
    prompt_template: |
      Synthesize the findings into a final report:

      ## Initial Findings
      {{.phases.initial-analysis}}

      ## Detailed Analysis
      {{.phases.deep-analysis}}

      Create a comprehensive final report.
    depends_on:
      - initial-analysis
      - deep-analysis
    routing_profile: balanced
    max_tokens: 8192
    temperature: 0.4
```

### Step 4: Configure Routing

Add routing configuration for model selection:

```yaml
routing:
  default_profile: balanced
  max_context_tokens: 16384
  fallback_model: "gpt-4"
```

### Step 5: Add Metadata

Include metadata for organization:

```yaml
metadata:
  author: your-name
  category: analysis
  license: MIT
  tags:
    - analysis
    - custom
    - workflow
  version_history:
    - "1.0.0: Initial release"
```

### Step 6: Validate Your Skill

Test your skill with a sample input:

```bash
sr run my-custom-skill --input "Sample input text"
```

### Complete Custom Skill Example

```yaml
id: sentiment-analyzer
name: Multi-Phase Sentiment Analyzer
version: "1.0.0"
description: |
  Analyzes sentiment in text using a multi-phase approach:
  1. Extract entities and topics
  2. Analyze sentiment for each entity
  3. Generate sentiment summary report

routing:
  default_profile: balanced
  max_context_tokens: 16384

phases:
  - id: extract
    name: Entity Extraction
    prompt_template: |
      Extract all entities, people, organizations, and key topics from:

      {{.input}}

      List each entity with context where it appears.
    routing_profile: cheap
    max_tokens: 2048
    temperature: 0.3

  - id: analyze
    name: Sentiment Analysis
    prompt_template: |
      For each entity identified:
      {{.phases.extract}}

      Original text:
      {{.input}}

      Analyze the sentiment (positive/negative/neutral) and provide:
      - Overall sentiment score (-1 to +1)
      - Key positive aspects
      - Key negative aspects
      - Supporting quotes from text
    depends_on:
      - extract
    routing_profile: balanced
    max_tokens: 4096
    temperature: 0.4

  - id: report
    name: Sentiment Report
    prompt_template: |
      Generate a comprehensive sentiment analysis report:

      ## Entities Identified
      {{.phases.extract}}

      ## Sentiment Analysis
      {{.phases.analyze}}

      Create a report with:
      - Executive summary
      - Sentiment breakdown by entity
      - Overall sentiment trend
      - Key insights and recommendations
    depends_on:
      - extract
      - analyze
    routing_profile: balanced
    max_tokens: 6144
    temperature: 0.5

metadata:
  author: skillrunner
  category: nlp
  license: MIT
  tags:
    - sentiment-analysis
    - nlp
    - text-analysis
```

---

## Importing Skills

Skillrunner supports importing skills from remote sources using the `sr import` command.

### Import from URL

Import a skill directly from a URL:

```bash
sr import https://example.com/skills/my-skill.yaml
```

### Import from Git Repository

Import skills from a Git repository:

```bash
# Import specific skill file
sr import https://github.com/user/repo/blob/main/skills/skill.yaml

# Import entire skills directory
sr import https://github.com/user/repo/tree/main/skills
```

### Import with Custom Name

Rename the skill during import:

```bash
sr import https://example.com/skill.yaml --name custom-skill
```

### Verify Imported Skills

List all available skills:

```bash
sr list
```

View skill details (with verbose output):

```bash
sr list --verbose
```

### Skill Repository Structure

When publishing skills for others to import, use this structure:

```
my-skills-repo/
├── README.md
├── skills/
│   ├── skill-1.yaml
│   ├── skill-2.yaml
│   └── skill-3.yaml
└── examples/
    ├── skill-1-example.md
    └── skill-2-example.md
```

---

## Best Practices

### 1. Skill Design

**Keep Phases Focused**
- Each phase should have a single, clear purpose
- Avoid cramming multiple unrelated tasks into one phase

**Use Appropriate Dependencies**
- Only add dependencies when one phase truly needs another's output
- Over-specifying dependencies reduces parallelism

**Example:**
```yaml
# Good: Focused phases
- id: extract-functions
  name: Extract Functions
  # ... extracts function signatures

- id: generate-tests
  name: Generate Tests
  depends_on: [extract-functions]
  # ... generates tests for extracted functions

# Bad: Unfocused phase
- id: do-everything
  name: Extract and Test
  # ... tries to do both extraction and test generation
```

### 2. Prompt Template Design

**Provide Clear Instructions**
- Be explicit about what you want the AI to do
- Include format requirements
- Specify what to include and exclude

**Use Examples**
```yaml
prompt_template: |
  Analyze the code for security issues.

  Format your response as:
  ## Issue: [Brief Title]
  - **Severity**: Critical|High|Medium|Low
  - **Description**: [Detailed explanation]
  - **Location**: [File:Line]
  - **Recommendation**: [How to fix]

  Example:
  ## Issue: SQL Injection Vulnerability
  - **Severity**: Critical
  - **Description**: User input directly concatenated into SQL query
  - **Location**: database.go:45
  - **Recommendation**: Use parameterized queries
```

**Reference Phase Outputs Correctly**
```yaml
# Good: Clear references
prompt_template: |
  Based on the analysis:
  {{.phases.analyze.output}}

  And the original input:
  {{.input}}

# Also valid (shorthand)
prompt_template: |
  Analysis results:
  {{.phases.analyze}}
```

### 3. Routing Strategy

**Match Profile to Task Complexity**

```yaml
phases:
  # Simple extraction - use cheap
  - id: parse
    routing_profile: cheap

  # Complex reasoning - use balanced
  - id: analyze
    routing_profile: balanced

  # Critical final output - use premium
  - id: final-report
    routing_profile: premium
```

**Set Appropriate Context Limits**

```yaml
# Small context for simple tasks
routing:
  default_profile: cheap
  max_context_tokens: 4096

# Large context for complex analysis
routing:
  default_profile: premium
  max_context_tokens: 32000
```

### 4. Temperature Settings

**Lower temperature (0.0 - 0.4)** for:
- Security analysis
- Code extraction
- Structured data output
- Factual analysis

**Medium temperature (0.5 - 0.7)** for:
- Code generation
- Documentation writing
- General-purpose tasks

**Higher temperature (0.8 - 1.0)** for:
- Creative writing
- Brainstorming
- Varied outputs

```yaml
phases:
  - id: security-scan
    temperature: 0.2  # Deterministic, factual

  - id: generate-docs
    temperature: 0.6  # Balanced

  - id: suggest-names
    temperature: 0.9  # Creative
```

### 5. Token Limits

**Set realistic max_tokens**

```yaml
phases:
  # Small output - simple extraction
  - id: extract
    max_tokens: 1024

  # Medium output - analysis
  - id: analyze
    max_tokens: 4096

  # Large output - comprehensive report
  - id: report
    max_tokens: 8192
```

### 6. Error Handling

**Provide fallback instructions in prompts**

```yaml
prompt_template: |
  Analyze the code for issues.

  If the code is incomplete or invalid, respond with:
  "ERROR: Unable to analyze - [reason]"

  Otherwise, provide structured analysis.
```

### 7. Metadata for Organization

**Use comprehensive metadata**

```yaml
metadata:
  author: team-name
  category: code-quality
  version_history:
    - "1.0.0: Initial release"
    - "1.1.0: Added security phase"
  license: MIT
  tags:
    - code-review
    - security
    - quality
  supported_languages:
    - go
    - python
  estimated_tokens: 15000
  estimated_cost: low
```

### 8. Testing and Iteration

**Test with diverse inputs**
- Test with minimal valid input
- Test with large/complex input
- Test with edge cases
- Test with invalid input

**Iterate on prompts**
- Run skill multiple times
- Refine prompts based on output quality
- Adjust temperature and token limits
- Add examples to prompts if outputs are inconsistent

### 9. Documentation

**Document your skill well**

```yaml
description: |
  ## Purpose
  This skill performs comprehensive code review including:
  - Pattern analysis
  - Security scanning
  - Best practice verification

  ## Input Format
  Accepts source code in any language. Optimal for files < 500 lines.

  ## Output Format
  Generates a markdown report with severity-rated findings.

  ## Cost Considerations
  Uses premium models for quality. Estimated cost: ~$0.50 per review.

  ## Example Usage
  ```bash
  sr run code-review --input mycode.go > review.md
  ```
```

### 10. Version Control

**Version your skills**

```yaml
version: "1.2.0"
metadata:
  version_history:
    - "1.0.0: Initial release with basic review"
    - "1.1.0: Added security scanning phase"
    - "1.2.0: Improved prompt templates for better accuracy"
  changelog_url: "https://github.com/user/skills/blob/main/CHANGELOG.md"
```

---

## Advanced Patterns

### Pattern 1: Fan-Out, Fan-In

Execute multiple independent analyses, then combine:

```yaml
phases:
  # Fan-out: Multiple parallel analyses
  - id: style-check
    # ... analyzes code style

  - id: complexity-check
    # ... analyzes complexity

  - id: performance-check
    # ... analyzes performance

  # Fan-in: Combine results
  - id: combine
    depends_on:
      - style-check
      - complexity-check
      - performance-check
    # ... synthesizes all analyses
```

### Pattern 2: Progressive Refinement

Iteratively refine output through multiple phases:

```yaml
phases:
  - id: draft
    # ... generates initial draft

  - id: refine
    depends_on: [draft]
    # ... refines the draft

  - id: polish
    depends_on: [refine]
    # ... final polish
```

### Pattern 3: Validation Pipeline

Separate generation from validation:

```yaml
phases:
  - id: generate
    # ... generates content

  - id: validate
    depends_on: [generate]
    # ... validates generated content

  - id: fix
    depends_on: [generate, validate]
    # ... fixes issues found in validation
```

---

## Troubleshooting

### Common Issues

**Issue: "Phase not found" error**
- Check that `depends_on` references use correct phase IDs
- Ensure phase IDs are unique within the skill

**Issue: "Cycle detected" error**
- Review dependencies to ensure no circular references
- Draw a dependency graph to visualize the flow

**Issue: Phase output is empty or truncated**
- Increase `max_tokens` for the phase
- Check that prompt template is correctly formatted
- Verify variable substitution syntax (`{{.input}}`, not `{.input}`)

**Issue: Skill runs slowly**
- Check for unnecessary sequential dependencies
- Consider using `cheap` profile for simple phases
- Reduce `max_tokens` where possible

**Issue: Inconsistent outputs**
- Lower `temperature` for more deterministic results
- Add examples to prompts
- Use `premium` profile for complex tasks

---

## Conclusion

Skills in Skillrunner provide a powerful way to orchestrate complex, multi-phase AI workflows. By understanding phase dependencies, routing profiles, and prompt template design, you can create sophisticated skills that combine the strengths of multiple AI models to tackle complex tasks efficiently.

### Next Steps

1. Explore the [built-in skills](#built-in-skills) to understand common patterns
2. Create your first [custom skill](#creating-custom-skills)
3. Share your skills with the community via Git repositories
4. Join the Skillrunner community to exchange skills and best practices

### Additional Resources

- **Skillrunner CLI Documentation**: See the main README for command-line usage
- **Example Skills Repository**: [github.com/skillrunner/skills](https://github.com/skillrunner/skills)
- **Community Skills**: Discover and share skills with other users

---

**Version**: 1.0.0
**Last Updated**: 2025-12-26
