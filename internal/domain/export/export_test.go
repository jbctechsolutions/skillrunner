package export_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/export"
)

func sampleExport() *export.WorkflowExport {
	return &export.WorkflowExport{
		SkillID:     "code-review",
		SkillName:   "code-review",
		Input:       "Review this function",
		Output:      "The function looks correct.",
		Profile:     "balanced",
		Status:      "completed",
		TotalCost:   0.001,
		TotalTokens: 120,
		Duration:    "500",
		ExportedAt:  time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC),
		Phases: []export.PhaseExport{
			{ID: "p1", Name: "Analysis", Output: "Phase 1 output", Status: "completed"},
		},
	}
}

func TestFormat_IsValid(t *testing.T) {
	valid := []export.Format{export.FormatJSON, export.FormatClaudeCode, export.FormatAider, export.FormatCursor}
	for _, f := range valid {
		if !f.IsValid() {
			t.Errorf("Format %q should be valid", f)
		}
	}
	if export.Format("unknown").IsValid() {
		t.Error("Format 'unknown' should not be valid")
	}
}

func TestWorkflowExport_MarshalJSON(t *testing.T) {
	e := sampleExport()
	b, err := e.Marshal(export.FormatJSON)
	if err != nil {
		t.Fatalf("Marshal(json) error: %v", err)
	}
	out := string(b)
	if !strings.Contains(out, "\"skill_name\":") {
		t.Errorf("JSON output missing skill_name: %s", out)
	}
	if !strings.Contains(out, "code-review") {
		t.Errorf("JSON output missing skill value: %s", out)
	}
}

func TestWorkflowExport_MarshalClaudeCode(t *testing.T) {
	e := sampleExport()
	b, err := e.Marshal(export.FormatClaudeCode)
	if err != nil {
		t.Fatalf("Marshal(claude-code) error: %v", err)
	}
	out := string(b)
	if !strings.Contains(out, "---") {
		t.Errorf("claude-code output missing frontmatter: %s", out)
	}
	if !strings.Contains(out, "## Final Output") {
		t.Errorf("claude-code output missing Final Output section: %s", out)
	}
}

func TestWorkflowExport_MarshalAider(t *testing.T) {
	e := sampleExport()
	b, err := e.Marshal(export.FormatAider)
	if err != nil {
		t.Fatalf("Marshal(aider) error: %v", err)
	}
	out := string(b)
	if !strings.Contains(out, "# skillrunner") {
		t.Errorf("aider output missing header: %s", out)
	}
}

func TestWorkflowExport_MarshalCursor(t *testing.T) {
	e := sampleExport()
	b, err := e.Marshal(export.FormatCursor)
	if err != nil {
		t.Fatalf("Marshal(cursor) error: %v", err)
	}
	out := string(b)
	if !strings.Contains(out, "cursor-context") {
		t.Errorf("cursor output missing context tag: %s", out)
	}
}

func TestWorkflowExport_MarshalUnknown(t *testing.T) {
	e := sampleExport()
	_, err := e.Marshal("unknown")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}
