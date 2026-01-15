// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"strings"
	"text/template"

	"github.com/jbctechsolutions/skillrunner/internal/application/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
	domainProvider "github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// PlannerConfig contains configuration options for the planner.
type PlannerConfig struct {
	// OutputTokenFraction is the fraction of MaxTokens to use for output token estimation.
	// Defaults to 0.5 (50% of MaxTokens).
	OutputTokenFraction float64
	// DefaultOutputTokens is the default output token estimate when MaxTokens is not set.
	// Defaults to 500.
	DefaultOutputTokens int
}

// DefaultPlannerConfig returns the default planner configuration.
func DefaultPlannerConfig() PlannerConfig {
	return PlannerConfig{
		OutputTokenFraction: 0.5,
		DefaultOutputTokens: 500,
	}
}

// Planner generates execution plans for skills.
// It uses the router to resolve models for each phase and the cost calculator
// to estimate costs based on token counts.
type Planner struct {
	router         *provider.Router
	costCalculator *domainProvider.CostCalculator
	tokenEstimator domainProvider.TokenEstimator
	config         PlannerConfig
}

// NewPlanner creates a new Planner with the given dependencies.
// The router and costCalculator can be nil for testing or simple scenarios,
// in which case the planner will use placeholder values.
func NewPlanner(
	router *provider.Router,
	costCalculator *domainProvider.CostCalculator,
	tokenEstimator domainProvider.TokenEstimator,
	config PlannerConfig,
) *Planner {
	if config.OutputTokenFraction <= 0 || config.OutputTokenFraction > 1 {
		config.OutputTokenFraction = 0.5
	}
	if config.DefaultOutputTokens <= 0 {
		config.DefaultOutputTokens = 500
	}

	return &Planner{
		router:         router,
		costCalculator: costCalculator,
		tokenEstimator: tokenEstimator,
		config:         config,
	}
}

// GeneratePlan creates an execution plan for the given skill and input.
// It builds the DAG, resolves models for each phase, estimates tokens and costs,
// and returns a complete execution plan.
func (p *Planner) GeneratePlan(
	ctx context.Context,
	sk *skill.Skill,
	input string,
	memoryContent string,
) (*workflow.ExecutionPlan, error) {
	if sk == nil {
		return nil, errors.NewError(errors.CodeValidation, "skill is nil", nil)
	}

	phases := sk.Phases()
	if len(phases) == 0 {
		return nil, errors.NewError(errors.CodeValidation, "skill has no phases", nil)
	}

	// Build DAG to get execution order
	dag, err := workflow.NewDAG(phases)
	if err != nil {
		return nil, err
	}

	// Get parallel batches for execution planning
	batches, err := dag.GetParallelBatches()
	if err != nil {
		return nil, err
	}

	// Create execution plan
	plan := workflow.NewExecutionPlan(sk.ID(), sk.Name(), sk.Version(), input)
	plan.SetBatches(batches)

	// Create a batch index map for quick lookup
	batchIndexMap := make(map[string]int)
	for batchIdx, batch := range batches {
		for _, phaseID := range batch {
			batchIndexMap[phaseID] = batchIdx
		}
	}

	// Generate plan for each phase
	for i := range phases {
		phase := &phases[i]
		phasePlan, err := p.planPhase(ctx, phase, input, memoryContent, batchIndexMap)
		if err != nil {
			// Log the error but continue with placeholder values
			phasePlan = p.createPlaceholderPhasePlan(phase, batchIndexMap)
		}
		plan.AddPhasePlan(*phasePlan)
	}

	return plan, nil
}

// planPhase generates a plan for a single phase.
func (p *Planner) planPhase(
	ctx context.Context,
	phase *skill.Phase,
	input string,
	memoryContent string,
	batchIndexMap map[string]int,
) (*workflow.PhasePlan, error) {
	// Resolve model using router
	modelID, providerName := p.resolveModel(ctx, phase)

	// Build estimated prompt to count tokens
	estimatedPrompt := p.buildEstimatedPrompt(phase, input, memoryContent)

	// Count input tokens
	inputTokens := 0
	if p.tokenEstimator != nil {
		inputTokens = p.tokenEstimator.CountTokens(estimatedPrompt)
	}

	// Estimate output tokens
	outputTokens := p.estimateOutputTokens(phase)

	// Calculate cost
	cost := p.estimateCost(modelID, inputTokens, outputTokens)

	return &workflow.PhasePlan{
		PhaseID:               phase.ID,
		PhaseName:             phase.Name,
		RoutingProfile:        phase.RoutingProfile,
		ResolvedModel:         modelID,
		ResolvedProvider:      providerName,
		DependsOn:             phase.DependsOn,
		EstimatedInputTokens:  inputTokens,
		EstimatedOutputTokens: outputTokens,
		EstimatedCost:         cost,
		BatchIndex:            batchIndexMap[phase.ID],
	}, nil
}

// createPlaceholderPhasePlan creates a phase plan with placeholder values.
func (p *Planner) createPlaceholderPhasePlan(
	phase *skill.Phase,
	batchIndexMap map[string]int,
) *workflow.PhasePlan {
	return &workflow.PhasePlan{
		PhaseID:               phase.ID,
		PhaseName:             phase.Name,
		RoutingProfile:        phase.RoutingProfile,
		ResolvedModel:         "unknown",
		ResolvedProvider:      "unknown",
		DependsOn:             phase.DependsOn,
		EstimatedInputTokens:  0,
		EstimatedOutputTokens: p.config.DefaultOutputTokens,
		EstimatedCost:         0,
		BatchIndex:            batchIndexMap[phase.ID],
	}
}

// resolveModel uses the router to select a model for the phase.
// Returns placeholder values if router is not available.
func (p *Planner) resolveModel(ctx context.Context, phase *skill.Phase) (modelID, providerName string) {
	if p.router == nil {
		// Return profile-based placeholder
		switch phase.RoutingProfile {
		case skill.ProfileCheap:
			return "local-model", "ollama"
		case skill.ProfilePremium:
			return "premium-model", "anthropic"
		default:
			return "balanced-model", "anthropic"
		}
	}

	selection, err := p.router.SelectModelForPhase(ctx, phase)
	if err != nil {
		// Fall back to placeholder
		return "unknown", "unknown"
	}

	return selection.ModelID, selection.ProviderName
}

// buildEstimatedPrompt creates an estimated prompt for token counting.
// This includes memory content, the rendered prompt template, and context overhead.
func (p *Planner) buildEstimatedPrompt(phase *skill.Phase, input string, memoryContent string) string {
	var builder strings.Builder

	// Add memory content overhead
	if memoryContent != "" {
		builder.WriteString("Project Memory:\n\n")
		builder.WriteString(memoryContent)
		builder.WriteString("\n\n")
	}

	// Render the prompt template with the input
	renderedPrompt := p.renderTemplate(phase.PromptTemplate, input)
	builder.WriteString(renderedPrompt)

	return builder.String()
}

// renderTemplate renders the prompt template with the given input.
// Returns the original template if rendering fails.
func (p *Planner) renderTemplate(templateStr string, input string) string {
	// Create template data with input
	data := map[string]any{
		"_input": input,
		"input":  input,
	}

	// Parse and execute the template
	tmpl, err := template.New("prompt").Parse(templateStr)
	if err != nil {
		return templateStr
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return templateStr
	}

	return buf.String()
}

// estimateOutputTokens estimates the number of output tokens for a phase.
func (p *Planner) estimateOutputTokens(phase *skill.Phase) int {
	if phase.MaxTokens <= 0 {
		return p.config.DefaultOutputTokens
	}
	return int(float64(phase.MaxTokens) * p.config.OutputTokenFraction)
}

// estimateCost calculates the estimated cost for the given token counts.
func (p *Planner) estimateCost(modelID string, inputTokens, outputTokens int) float64 {
	if p.costCalculator == nil {
		return 0
	}

	cost, err := p.costCalculator.EstimateInputOutputCost(modelID, inputTokens, outputTokens)
	if err != nil {
		// Model not registered, return 0
		return 0
	}

	return cost
}
