// Package export provides types and formatters for exporting workflow results
// in formats consumable by external AI coding tools.
package export

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Format represents an export target format.
type Format string

const (
	FormatJSON       Format = "json"
	FormatClaudeCode Format = "claude-code"
	FormatAider      Format = "aider"
	FormatCursor     Format = "cursor"
)

// ValidFormats lists all supported export formats.
var ValidFormats = []Format{FormatJSON, FormatClaudeCode, FormatAider, FormatCursor}

// IsValid reports whether the format is supported.
func (f Format) IsValid() bool {
	for _, v := range ValidFormats {
		if f == v {
			return true
		}
	}
	return false
}

// PhaseExport contains the exported result of a single phase.
type PhaseExport struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Model  string  `json:"model,omitempty"`
	Output string  `json:"output"`
	Tokens int     `json:"tokens,omitempty"`
	Cost   float64 `json:"cost_usd,omitempty"`
	Status string  `json:"status"`
}

// WorkflowExport is the canonical representation of a completed skill execution.
type WorkflowExport struct {
	SkillID     string        `json:"skill_id"`
	SkillName   string        `json:"skill_name"`
	Input       string        `json:"input"`
	Output      string        `json:"output"`
	Profile     string        `json:"profile"`
	Status      string        `json:"status"`
	Phases      []PhaseExport `json:"phases"`
	Duration    string        `json:"duration_ms"`
	TotalCost   float64       `json:"total_cost_usd,omitempty"`
	TotalTokens int           `json:"total_tokens,omitempty"`
	ExportedAt  time.Time     `json:"exported_at"`
}

// Marshal serialises the export in the requested format.
func (w *WorkflowExport) Marshal(f Format) ([]byte, error) {
	switch f {
	case FormatJSON:
		return json.MarshalIndent(w, "", "  ")
	case FormatClaudeCode:
		return []byte(w.toClaudeCode()), nil
	case FormatAider:
		return []byte(w.toAider()), nil
	case FormatCursor:
		return []byte(w.toCursor()), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", f)
	}
}

// toClaudeCode produces a markdown document suitable for pasting into Claude Code.
func (w *WorkflowExport) toClaudeCode() string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("skill: %s\n", w.SkillName))
	sb.WriteString(fmt.Sprintf("profile: %s\n", w.Profile))
	sb.WriteString(fmt.Sprintf("status: %s\n", w.Status))
	if w.TotalCost > 0 {
		sb.WriteString(fmt.Sprintf("cost: $%.4f\n", w.TotalCost))
	}
	sb.WriteString(fmt.Sprintf("exported_at: %s\n", w.ExportedAt.Format(time.RFC3339)))
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# %s — Skill Output\n\n", w.SkillName))
	sb.WriteString("## Input\n\n")
	sb.WriteString(w.Input + "\n\n")

	if len(w.Phases) > 1 {
		sb.WriteString("## Phase Results\n\n")
		for _, p := range w.Phases {
			sb.WriteString(fmt.Sprintf("### %s\n\n", p.Name))
			sb.WriteString(p.Output + "\n\n")
		}
	}

	sb.WriteString("## Final Output\n\n")
	sb.WriteString(w.Output + "\n")
	return sb.String()
}

// toAider produces a context block in Aider's convention.
func (w *WorkflowExport) toAider() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# skillrunner: %s\n\n", w.SkillName))
	sb.WriteString(fmt.Sprintf("> **Input:** %s\n\n", w.Input))
	sb.WriteString("## Output\n\n")
	sb.WriteString(w.Output + "\n\n")

	if len(w.Phases) > 1 {
		sb.WriteString("## Context (per phase)\n\n")
		for _, p := range w.Phases {
			sb.WriteString(fmt.Sprintf("<details><summary>%s</summary>\n\n%s\n\n</details>\n\n", p.Name, p.Output))
		}
	}
	return sb.String()
}

// toCursor produces a cursor-compatible markdown context block.
func (w *WorkflowExport) toCursor() string {
	var sb strings.Builder
	sb.WriteString("<!-- cursor-context: skillrunner -->\n")
	sb.WriteString(fmt.Sprintf("## %s\n\n", w.SkillName))
	sb.WriteString(fmt.Sprintf("**Input:** %s\n\n", w.Input))
	sb.WriteString("**Output:**\n\n")
	sb.WriteString(w.Output + "\n")
	return sb.String()
}
