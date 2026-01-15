// Package workflow provides workflow orchestration types for skill execution.
package workflow

import (
	"encoding/json"
	"time"
)

// PhasePlan represents the planned execution of a single phase.
type PhasePlan struct {
	PhaseID               string   `json:"phase_id"`
	PhaseName             string   `json:"phase_name"`
	RoutingProfile        string   `json:"routing_profile"`
	ResolvedModel         string   `json:"resolved_model"`
	ResolvedProvider      string   `json:"resolved_provider"`
	DependsOn             []string `json:"depends_on,omitempty"`
	EstimatedInputTokens  int      `json:"estimated_input_tokens"`
	EstimatedOutputTokens int      `json:"estimated_output_tokens"`
	EstimatedCost         float64  `json:"estimated_cost"`
	BatchIndex            int      `json:"batch_index"`
}

// TotalEstimatedTokens returns the sum of input and output tokens.
func (p *PhasePlan) TotalEstimatedTokens() int {
	return p.EstimatedInputTokens + p.EstimatedOutputTokens
}

// ExecutionPlan represents the complete execution plan for a skill.
type ExecutionPlan struct {
	SkillID                    string      `json:"skill_id"`
	SkillName                  string      `json:"skill_name"`
	SkillVersion               string      `json:"skill_version"`
	Input                      string      `json:"input"`
	Phases                     []PhasePlan `json:"phases"`
	Batches                    [][]string  `json:"batches"`
	TotalEstimatedInputTokens  int         `json:"total_estimated_input_tokens"`
	TotalEstimatedOutputTokens int         `json:"total_estimated_output_tokens"`
	TotalEstimatedCost         float64     `json:"total_estimated_cost"`
	CreatedAt                  time.Time   `json:"created_at"`
}

// NewExecutionPlan creates a new ExecutionPlan with the given skill information.
func NewExecutionPlan(skillID, skillName, skillVersion, input string) *ExecutionPlan {
	return &ExecutionPlan{
		SkillID:      skillID,
		SkillName:    skillName,
		SkillVersion: skillVersion,
		Input:        input,
		Phases:       make([]PhasePlan, 0),
		Batches:      make([][]string, 0),
		CreatedAt:    time.Now(),
	}
}

// AddPhasePlan adds a phase plan to the execution plan.
func (p *ExecutionPlan) AddPhasePlan(phase PhasePlan) {
	p.Phases = append(p.Phases, phase)
	p.TotalEstimatedInputTokens += phase.EstimatedInputTokens
	p.TotalEstimatedOutputTokens += phase.EstimatedOutputTokens
	p.TotalEstimatedCost += phase.EstimatedCost
}

// SetBatches sets the parallel execution batches for the plan.
func (p *ExecutionPlan) SetBatches(batches [][]string) {
	p.Batches = batches
}

// TotalEstimatedTokens returns the sum of input and output tokens for all phases.
func (p *ExecutionPlan) TotalEstimatedTokens() int {
	return p.TotalEstimatedInputTokens + p.TotalEstimatedOutputTokens
}

// PhaseCount returns the number of phases in the plan.
func (p *ExecutionPlan) PhaseCount() int {
	return len(p.Phases)
}

// BatchCount returns the number of parallel execution batches.
func (p *ExecutionPlan) BatchCount() int {
	return len(p.Batches)
}

// GetPhase returns the phase plan for the given phase ID, or nil if not found.
func (p *ExecutionPlan) GetPhase(phaseID string) *PhasePlan {
	for i := range p.Phases {
		if p.Phases[i].PhaseID == phaseID {
			return &p.Phases[i]
		}
	}
	return nil
}

// ToJSON serializes the execution plan to JSON bytes.
func (p *ExecutionPlan) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// ExecutionPlanFromJSON deserializes an execution plan from JSON bytes.
func ExecutionPlanFromJSON(data []byte) (*ExecutionPlan, error) {
	var plan ExecutionPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// HasLocalModels returns true if any phase uses a local model.
func (p *ExecutionPlan) HasLocalModels() bool {
	for _, phase := range p.Phases {
		if phase.EstimatedCost == 0 && phase.ResolvedProvider != "" {
			return true
		}
	}
	return false
}

// HasCloudModels returns true if any phase uses a cloud model (non-zero cost).
func (p *ExecutionPlan) HasCloudModels() bool {
	for _, phase := range p.Phases {
		if phase.EstimatedCost > 0 {
			return true
		}
	}
	return false
}
