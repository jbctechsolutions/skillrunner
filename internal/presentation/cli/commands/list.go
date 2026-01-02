// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// SkillInfo represents information about a skill for display.
type SkillInfo struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	PhaseCount     int    `json:"phase_count"`
	RoutingProfile string `json:"routing_profile"`
}

// SkillListOutput represents the output for the list command.
type SkillListOutput struct {
	Skills []SkillInfo `json:"skills"`
	Count  int         `json:"count"`
}

// NewListCmd creates the list command for displaying available skills.
func NewListCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Long: `Display a list of all available skills in the skillrunner.

Each skill represents a multi-phase AI workflow that can be executed.
The output shows the skill name, description, number of phases, and routing profile.`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(format)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "", "output format: text, json, table (default: uses global --output flag)")

	return cmd
}

func runList(formatFlag string) error {
	// Determine output format
	// Priority: --format flag > global --output flag > default (text)
	format := output.FormatText
	if formatFlag != "" {
		parsedFormat, err := output.ParseFormat(formatFlag)
		if err != nil {
			return fmt.Errorf("invalid format: %s (valid options: text, json, table)", formatFlag)
		}
		format = parsedFormat
	} else if globalFlags.Output == "json" {
		format = output.FormatJSON
	} else if globalFlags.Output == "table" {
		format = output.FormatTable
	}

	formatter := output.NewFormatter(
		output.WithFormat(format),
		output.WithColor(format != output.FormatJSON),
	)

	// Get skill data from registry
	skills := loadSkills()

	// Build output structure
	listOutput := SkillListOutput{
		Skills: skills,
		Count:  len(skills),
	}

	// Output based on format
	switch format {
	case output.FormatJSON:
		return formatter.JSON(listOutput)
	case output.FormatTable, output.FormatText:
		return renderSkillsTable(formatter, skills)
	default:
		return renderSkillsTable(formatter, skills)
	}
}

// renderSkillsTable renders skills as a formatted table.
func renderSkillsTable(formatter *output.Formatter, skills []SkillInfo) error {
	if len(skills) == 0 {
		formatter.Info("No skills available")
		formatter.Println("")
		formatter.Println("To add skills, place skill definitions in your skills directory.")
		formatter.Println("Run 'sr --help' for more information.")
		return nil
	}

	// Define table columns
	tableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: "NAME", Width: 20, Align: output.AlignLeft},
			{Header: "DESCRIPTION", Width: 40, Align: output.AlignLeft},
			{Header: "PHASES", Width: 8, Align: output.AlignRight},
			{Header: "ROUTING", Width: 15, Align: output.AlignLeft},
		},
		Rows: make([][]string, 0, len(skills)),
	}

	// Build rows
	for _, skill := range skills {
		row := []string{
			skill.Name,
			truncateString(skill.Description, 40),
			strconv.Itoa(skill.PhaseCount),
			skill.RoutingProfile,
		}
		tableData.Rows = append(tableData.Rows, row)
	}

	// Print header
	formatter.Println("")
	formatter.Println("%s", formatter.Bold("Available Skills"))
	formatter.Println("")

	// Render table
	if err := formatter.Table(tableData); err != nil {
		return err
	}

	// Print summary
	formatter.Println("")
	formatter.Println("%s", formatter.Dim(fmt.Sprintf("Total: %d skill(s)", len(skills))))

	return nil
}

// truncateString truncates a string to the specified length with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// loadSkills loads skills from the skill registry.
// Returns skill info for display in the list command.
func loadSkills() []SkillInfo {
	container := GetContainer()
	if container == nil {
		// Container not initialized, return empty list
		return nil
	}

	registry := container.SkillRegistry()
	if registry == nil {
		return nil
	}

	// Get all skills from the registry
	skills := registry.ListSkills()

	// Convert to SkillInfo for display
	result := make([]SkillInfo, 0, len(skills))
	for _, s := range skills {
		info := SkillInfo{
			Name:           s.Name(),
			Description:    s.Description(),
			PhaseCount:     len(s.Phases()),
			RoutingProfile: s.Routing().DefaultProfile,
		}
		result = append(result, info)
	}

	return result
}
