package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jbctechsolutions/skillrunner/internal/engine"
	"github.com/jbctechsolutions/skillrunner/internal/models"
	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// Router defines the interface for command routing
type Router interface {
	RouteRun(skillName, request, modelOverride, workspace, outputFile string, compact bool) error
	RouteList(format string) error
	RouteStatus(format string) error
}

// CommandRouter implements Router using the real engine
type CommandRouter struct {
	engineFactory func(string, models.ResolvePolicy) *engine.Skillrunner
	outputWriter  func(string, []byte) error
	stdoutWriter  func(string)
	stderrWriter  func(string, ...interface{})
}

// NewCommandRouter creates a new CommandRouter with default dependencies
func NewCommandRouter() *CommandRouter {
	return &CommandRouter{
		engineFactory: func(workspace string, policy models.ResolvePolicy) *engine.Skillrunner {
			return engine.NewSkillrunner(workspace, policy)
		},
		outputWriter: func(path string, data []byte) error {
			return os.WriteFile(path, data, 0644)
		},
		stdoutWriter: func(s string) {
			fmt.Print(s)
		},
		stderrWriter: func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, format, args...)
		},
	}
}

// RouteRun handles the run command
func (r *CommandRouter) RouteRun(skillName, request, modelOverride, workspace, outputFile string, compact bool) error {
	if skillName == "" {
		return fmt.Errorf("skill name is required")
	}
	if request == "" {
		return fmt.Errorf("request is required")
	}

	// Initialize engine with auto policy
	eng := r.engineFactory(workspace, models.ResolvePolicyAuto)

	// Execute the skill
	envelope, err := eng.Run(skillName, request, modelOverride)
	if err != nil {
		return fmt.Errorf("failed to run skill '%s': %w", skillName, err)
	}

	// Convert to JSON
	var jsonOutput []byte
	var marshalErr error
	if compact {
		jsonOutput, marshalErr = envelopeToJSON(envelope, true)
	} else {
		jsonOutput, marshalErr = envelopeToJSON(envelope, false)
	}
	if marshalErr != nil {
		return fmt.Errorf("failed to format output: %w", marshalErr)
	}

	// Output to file or stdout
	if outputFile != "" {
		if err := r.outputWriter(outputFile, jsonOutput); err != nil {
			return fmt.Errorf("failed to write output to '%s': %w", outputFile, err)
		}
		r.stderrWriter("Envelope written to %s\n", outputFile)
	} else {
		r.stdoutWriter(string(jsonOutput) + "\n")
	}

	return nil
}

// RouteList handles the list command
func (r *CommandRouter) RouteList(format string) error {
	eng := r.engineFactory("", models.ResolvePolicyAuto)
	skills, err := eng.ListSkills()
	if err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

	if format == "json" {
		output, err := skillsToJSON(skills)
		if err != nil {
			return fmt.Errorf("failed to format skills: %w", err)
		}
		r.stdoutWriter(output + "\n")
	} else {
		output := formatSkillsTable(skills)
		r.stdoutWriter(output)
	}

	return nil
}

// RouteStatus handles the status command
func (r *CommandRouter) RouteStatus(format string) error {
	eng := r.engineFactory("", models.ResolvePolicyAuto)
	status, err := eng.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if format == "json" {
		output, err := statusToJSON(status)
		if err != nil {
			return fmt.Errorf("failed to format status: %w", err)
		}
		r.stdoutWriter(output + "\n")
	} else {
		output := formatStatusTable(status)
		r.stdoutWriter(output)
	}

	return nil
}

// MockRouter implements Router for testing
type MockRouter struct {
	RunFunc    func(skillName, request, modelOverride, workspace, outputFile string, compact bool) error
	ListFunc   func(format string) error
	StatusFunc func(format string) error
}

// RouteRun calls the mock function or returns nil
func (m *MockRouter) RouteRun(skillName, request, modelOverride, workspace, outputFile string, compact bool) error {
	if m.RunFunc != nil {
		return m.RunFunc(skillName, request, modelOverride, workspace, outputFile, compact)
	}
	return nil
}

// RouteList calls the mock function or returns nil
func (m *MockRouter) RouteList(format string) error {
	if m.ListFunc != nil {
		return m.ListFunc(format)
	}
	return nil
}

// RouteStatus calls the mock function or returns nil
func (m *MockRouter) RouteStatus(format string) error {
	if m.StatusFunc != nil {
		return m.StatusFunc(format)
	}
	return nil
}

// Helper functions for JSON marshaling
func envelopeToJSON(envelope *types.Envelope, compact bool) ([]byte, error) {
	if compact {
		return json.Marshal(envelope)
	}
	return json.MarshalIndent(envelope, "", "  ")
}

func skillsToJSON(skills []types.SkillConfig) (string, error) {
	data, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func statusToJSON(status *types.SystemStatus) (string, error) {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Helper functions for table formatting
func formatSkillsTable(skills []types.SkillConfig) string {
	var output string
	output += "\nAvailable Skills:\n"
	output += "----------------------------------------------------------------------\n"
	for _, skill := range skills {
		output += fmt.Sprintf("  %-20s v%-10s\n", skill.Name, skill.Version)
		output += fmt.Sprintf("    %s\n\n", skill.Description)
	}
	return output
}

func formatStatusTable(status *types.SystemStatus) string {
	var output string
	output += "\nSkillrunner Status:\n"
	output += "----------------------------------------------------------------------\n"
	output += fmt.Sprintf("  Version:          %s\n", status.Version)
	output += fmt.Sprintf("  Available Skills: %d\n", status.SkillCount)
	output += fmt.Sprintf("  Workspace:        %s\n", status.Workspace)
	readyStr := "Ready"
	if !status.Ready {
		readyStr = "Not Ready"
	}
	output += fmt.Sprintf("  Status:           %s\n", readyStr)
	output += "\n  Configured Models:\n"
	for _, model := range status.ConfiguredModels {
		output += fmt.Sprintf("    - %s\n", model)
	}
	output += "\n"
	return output
}
