// Package output provides CLI output formatting utilities.
package output

import (
	"fmt"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// DAGRenderer renders execution plans with DAG visualization.
type DAGRenderer struct {
	formatter *Formatter
}

// NewDAGRenderer creates a new DAG renderer with the given formatter.
func NewDAGRenderer(formatter *Formatter) *DAGRenderer {
	return &DAGRenderer{
		formatter: formatter,
	}
}

// RenderPlan renders an execution plan with DAG visualization.
func (r *DAGRenderer) RenderPlan(plan *workflow.ExecutionPlan) {
	r.renderHeader(plan)
	r.renderPhaseTable(plan)
	r.renderTotals(plan)
}

// renderHeader renders the skill and input information.
func (r *DAGRenderer) renderHeader(plan *workflow.ExecutionPlan) {
	_ = r.formatter.Header("Execution Plan")
	_ = r.formatter.Item("Skill", fmt.Sprintf("%s v%s", plan.SkillName, plan.SkillVersion))

	// Truncate input if too long
	input := plan.Input
	if len(input) > 80 {
		input = input[:77] + "..."
	}
	_ = r.formatter.Item("Input", fmt.Sprintf("%q", input))
	_ = r.formatter.Println("")
}

// renderPhaseTable renders the phase details as a box-drawing table.
func (r *DAGRenderer) renderPhaseTable(plan *workflow.ExecutionPlan) {
	if len(plan.Phases) == 0 {
		return
	}

	_ = r.formatter.SubHeader("Phase Details")
	_ = r.formatter.Println("")

	// Draw the phases in boxes
	for i, phase := range plan.Phases {
		isFirst := i == 0
		isLast := i == len(plan.Phases)-1

		r.renderPhaseBox(phase, isFirst, isLast)
	}
}

// renderPhaseBox renders a single phase as a box with connections.
func (r *DAGRenderer) renderPhaseBox(phase workflow.PhasePlan, isFirst, isLast bool) {
	// Box width for consistent rendering
	const boxWidth = 50

	// Build phase info
	var deps string
	if len(phase.DependsOn) > 0 {
		deps = fmt.Sprintf(" (depends: %s)", strings.Join(phase.DependsOn, ", "))
	}

	name := fmt.Sprintf("%s%s", phase.PhaseName, deps)
	if len(name) > boxWidth-4 {
		name = name[:boxWidth-7] + "..."
	}

	profile := fmt.Sprintf("%s → %s", phase.RoutingProfile, phase.ResolvedModel)
	if phase.ResolvedProvider != "" && phase.ResolvedProvider != "unknown" {
		profile = fmt.Sprintf("%s (%s)", profile, phase.ResolvedProvider)
	}
	if len(profile) > boxWidth-4 {
		profile = profile[:boxWidth-7] + "..."
	}

	tokens := fmt.Sprintf("Est. tokens: ~%d input, ~%d output",
		phase.EstimatedInputTokens, phase.EstimatedOutputTokens)

	var cost string
	if phase.EstimatedCost > 0 {
		cost = fmt.Sprintf("Est. cost: $%.4f", phase.EstimatedCost)
	} else {
		cost = "Est. cost: $0.00 (local)"
	}

	batch := fmt.Sprintf("[batch %d]", phase.BatchIndex+1)

	// Render top border
	if isFirst {
		_ = r.formatter.Println("┌%s┐", strings.Repeat("─", boxWidth-2))
	} else {
		_ = r.formatter.Println("├%s┤", strings.Repeat("─", boxWidth-2))
	}

	// Render phase content
	r.renderBoxLine(name, batch, boxWidth)
	r.renderBoxLine(profile, "", boxWidth)
	r.renderBoxLine(tokens, "", boxWidth)
	r.renderBoxLine(cost, "", boxWidth)

	// Render bottom border or separator
	if isLast {
		_ = r.formatter.Println("└%s┘", strings.Repeat("─", boxWidth-2))
	}
}

// renderBoxLine renders a line inside the box with proper padding.
func (r *DAGRenderer) renderBoxLine(left, right string, boxWidth int) {
	// Calculate available space
	availableWidth := boxWidth - 4 // 2 for borders, 2 for padding

	if right != "" {
		// With right content: left ... right
		rightPadded := " " + right
		leftWidth := availableWidth - len(rightPadded)
		if len(left) > leftWidth {
			left = left[:leftWidth-3] + "..."
		}
		padding := strings.Repeat(" ", leftWidth-len(left))
		_ = r.formatter.Println("│ %s%s%s │", left, padding, rightPadded)
	} else {
		// Just left content
		if len(left) > availableWidth {
			left = left[:availableWidth-3] + "..."
		}
		padding := strings.Repeat(" ", availableWidth-len(left))
		_ = r.formatter.Println("│ %s%s │", left, padding)
	}
}

// renderTotals renders the total cost and token estimates.
func (r *DAGRenderer) renderTotals(plan *workflow.ExecutionPlan) {
	_ = r.formatter.Println("")
	_ = r.formatter.SubHeader("Totals")

	_ = r.formatter.Item("Estimated Input Tokens", fmt.Sprintf("%d", plan.TotalEstimatedInputTokens))
	_ = r.formatter.Item("Estimated Output Tokens", fmt.Sprintf("~%d", plan.TotalEstimatedOutputTokens))
	_ = r.formatter.Item("Total Estimated Tokens", fmt.Sprintf("~%d", plan.TotalEstimatedTokens()))

	if plan.TotalEstimatedCost > 0 {
		_ = r.formatter.Item("Estimated Total Cost", fmt.Sprintf("$%.4f", plan.TotalEstimatedCost))
	} else {
		_ = r.formatter.Item("Estimated Total Cost", "$0.00 (all local models)")
	}

	_ = r.formatter.Item("Execution Batches", fmt.Sprintf("%d", plan.BatchCount()))
	_ = r.formatter.Println("")
}

// RenderApprovalPrompt renders the approval prompt.
func (r *DAGRenderer) RenderApprovalPrompt() {
	r.formatter.Bold("Proceed with execution? [Y/n] ")
}

// RenderPlanSaved renders a message indicating the plan was saved.
func (r *DAGRenderer) RenderPlanSaved(path string) {
	_ = r.formatter.Success("Plan saved to: %s", path)
}

// RenderPlanJSON outputs the plan as JSON.
func (r *DAGRenderer) RenderPlanJSON(plan *workflow.ExecutionPlan) error {
	return r.formatter.JSON(plan)
}
