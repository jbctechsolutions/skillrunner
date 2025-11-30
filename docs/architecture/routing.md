# Skillrunner Multi-Provider Routing & Cost Simulation Spec
(Supporting Anthropic, OpenAI, Google Gemini with cheap/balanced/premium profiles)

This document defines the full technical specification for implementing multi-provider LLM routing, cost simulation, and cheap/balanced/premium routing profiles inside **Skillrunner**.

It is formatted for consumption by Cursor Agents and can be placed directly into the Skillrunner repository as `docs/routing-spec.md`.

---

## 1. Goals

1. Add first-class support for:
   - Anthropic (Claude 3.x)
   - OpenAI (GPT-4.x family)
   - Google Gemini (1.5 Flash/Pro via HTTP API)

2. Introduce routing profiles:
   - `cheap`
   - `balanced`
   - `premium`

3. For every workflow run:
   - Route each phase using the selected profile.
   - Track actual token usage and actual cost.
   - Compute counterfactual costs:
     - “What if this entire run used premium only?”
     - “What if this entire run used cheap only?”

4. Expose:
   - YAML config
   - Engine types (Go)
   - Provider implementations
   - Router
   - Cost computer
   - CLI integration
   - JSON run report generation

---

## 2. High-Level Architecture

New/updated directories:

```
internal/llm/
    provider.go
    anthropic.go
    openai.go
    google.go

internal/routing/
    config.go
    router.go

internal/metrics/
    cost.go

internal/run/
    report.go
```

The orchestration engine calls a generic `LLMClient` which uses:
- A `Router`
- Backed by provider implementations (`Anthropic`, `OpenAI`, `Google`)

---

## 3. Config Schema

File: `config/models.yaml`

### Top-Level YAML

```yaml
providers:
  - name: anthropic
    type: anthropic
    api_key_env: ANTHROPIC_API_KEY

  - name: openai
    type: openai
    api_key_env: OPENAI_API_KEY

  - name: google
    type: google
    api_key_env: GOOGLE_API_KEY

models:
  # CHEAP PROFILE MODELS
  - id: cheap-anthropic-haiku
    provider: anthropic
    model: claude-3-haiku-20240307
    profile_cost_per_1k_input: 0.25
    profile_cost_per_1k_output: 1.25

  - id: cheap-openai-mini
    provider: openai
    model: gpt-4.1-mini
    profile_cost_per_1k_input: 0.15
    profile_cost_per_1k_output: 0.60

  - id: cheap-google-flash
    provider: google
    model: gemini-1.5-flash
    profile_cost_per_1k_input: 0.10
    profile_cost_per_1k_output: 0.40

  # BALANCED PROFILE MODELS
  - id: balanced-anthropic-sonnet
    provider: anthropic
    model: claude-3.5-sonnet
    profile_cost_per_1k_input: 0.80
    profile_cost_per_1k_output: 4.00

  - id: balanced-openai-gpt4o
    provider: openai
    model: gpt-4.1
    profile_cost_per_1k_input: 0.50
    profile_cost_per_1k_output: 1.50

  - id: balanced-google-pro
    provider: google
    model: gemini-1.5-pro
    profile_cost_per_1k_input: 0.50
    profile_cost_per_1k_output: 1.50

  # PREMIUM PROFILE MODELS
  - id: premium-anthropic
    provider: anthropic
    model: claude-3.5-sonnet
    profile_cost_per_1k_input: 0.80
    profile_cost_per_1k_output: 4.00

  - id: premium-openai
    provider: openai
    model: gpt-4.1
    profile_cost_per_1k_input: 0.50
    profile_cost_per_1k_output: 1.50

  - id: premium-google
    provider: google
    model: gemini-1.5-pro
    profile_cost_per_1k_input: 0.50
    profile_cost_per_1k_output: 1.50

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

cost_simulation:
  premium_model_id: premium-anthropic
  cheap_model_id: cheap-google-flash
```

---

## 4. Go Provider Interfaces

File: `internal/llm/provider.go`

```go
package llm

import "context"

type MessageRole string

const (
    RoleUser      MessageRole = "user"
    RoleSystem    MessageRole = "system"
    RoleAssistant MessageRole = "assistant"
)

type ChatMessage struct {
    Role    MessageRole `json:"role"`
    Content string      `json:"content"`
}

type ChatRequest struct {
    Model       string
    Messages    []ChatMessage
    MaxTokens   int
    Temperature float32
}

type Usage struct {
    InputTokens  int
    OutputTokens int
}

type ChatResponse struct {
    Content string
    Usage   Usage
}

type Provider interface {
    Name() string
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
```

---

## 5. Provider Implementations

Three providers to be added:

- `internal/llm/anthropic.go`
- `internal/llm/openai.go`
- `internal/llm/google.go`

Each:
- Reads API key from env
- Formats request properly
- Maps token usage into `Usage`

Detailed API endpoints are included in the spec used earlier.

---

## 6. Router

File: `internal/routing/router.go`

```go
package routing

import (
    "fmt"
    "github.com/yourorg/skillrunner/internal/llm"
)

type Router struct {
    providers      map[string]llm.Provider
    models         map[string]ModelConfig
    profiles       map[string]RoutingProfileConfig
    costSimulation CostSimulationConfig
}

func (r *Router) Route(profile string) (llm.Provider, ModelConfig, error) {
    p, ok := r.profiles[profile]
    if !ok {
        return nil, ModelConfig{}, fmt.Errorf("unknown profile: %s", profile)
    }

    for _, modelID := range p.CandidateModels {
        mc, ok := r.models[modelID]
        if !ok {
            continue
        }
        prov, ok := r.providers[mc.Provider]
        if !ok {
            continue
        }
        return prov, mc, nil
    }

    return nil, ModelConfig{}, fmt.Errorf("no available providers for profile %s", profile)
}
```

---

## 7. Cost Simulation

File: `internal/metrics/cost.go`

Implements:
- `PhaseCost`
- `RunCostSummary`
- `CostComputer`

Provides:
- `ComputePhaseCost`
- `SummarizeRun`

This computes:
- Actual cost for used model
- Cost if premium-only
- Cost if cheap-only

---

## 8. Run Report

File: `internal/run/report.go`

```go
type RunReport struct {
    WorkflowName string                `json:"workflow_name"`
    ProfileUsed  string                `json:"profile_used"`
    CostSummary  metrics.RunCostSummary `json:"cost_summary"`
}
```

---

## 9. CLI Integration

Add flags:

```
--profile=cheap|balanced|premium
--show-savings
--report-json=path
```

Print:

```
Run complete.
Actual cost: $0.09
If premium-only: $0.63
If cheap-only: $0.04
```

Optional JSON file export.

---

## 10. Tasks for Cursor Agent

1. Add config structs, YAML loader.
2. Implement Anthropic, OpenAI, Google providers.
3. Implement router.
4. Add cost simulation engine.
5. Integrate with orchestrator.
6. Add CLI flags + printing logic.
7. Add JSON report writer.
8. Update documentation + provide example config.

---

End of spec.
