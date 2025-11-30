package run

import (
	"encoding/json"
	"os"

	"github.com/jbctechsolutions/skillrunner/internal/metrics"
)

// RunReport represents a run report with cost summary
type RunReport struct {
	WorkflowName string                 `json:"workflow_name"`
	ProfileUsed  string                 `json:"profile_used"`
	CostSummary  metrics.RunCostSummary `json:"cost_summary"`
}

// WriteJSON writes the run report to a JSON file
func (r *RunReport) WriteJSON(filePath string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// ToJSON returns the JSON representation of the run report
func (r *RunReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
