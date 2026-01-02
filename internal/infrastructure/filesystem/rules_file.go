// Package filesystem provides filesystem operations for workspace management.
package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RulesFileHandler manages reading and writing the rules.md file.
type RulesFileHandler struct {
	workspaceFS *WorkspaceFS
}

// NewRulesFileHandler creates a new rules file handler.
func NewRulesFileHandler() *RulesFileHandler {
	return &RulesFileHandler{
		workspaceFS: NewWorkspaceFS(),
	}
}

// Read reads all rules from the rules.md file.
// Returns a map of rule names to their content.
func (h *RulesFileHandler) Read(repoPath string) (map[string]string, error) {
	rulesPath := h.workspaceFS.GetRulesPath(repoPath)

	// Check if file exists
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	file, err := os.Open(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rules file: %w", err)
	}
	defer file.Close()

	rules := make(map[string]string)
	scanner := bufio.NewScanner(file)

	var currentRule string
	var currentContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check for markdown heading (## Rule Name)
		if strings.HasPrefix(line, "## ") {
			// Save previous rule if any
			if currentRule != "" {
				rules[currentRule] = strings.TrimSpace(currentContent.String())
			}

			// Start new rule
			currentRule = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentContent.Reset()
			continue
		}

		// Skip main title (# Workspace Rules)
		if strings.HasPrefix(line, "# ") {
			continue
		}

		// Add line to current rule content
		if currentRule != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	// Save last rule
	if currentRule != "" {
		rules[currentRule] = strings.TrimSpace(currentContent.String())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	return rules, nil
}

// Write writes rules to the rules.md file.
func (h *RulesFileHandler) Write(repoPath string, rules map[string]string) error {
	rulesPath := h.workspaceFS.GetRulesPath(repoPath)

	// Ensure directory exists
	dir := filepath.Dir(rulesPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create rules directory: %w", err)
	}

	file, err := os.Create(rulesPath)
	if err != nil {
		return fmt.Errorf("failed to create rules file: %w", err)
	}
	defer file.Close()

	// Write header
	if _, err := file.WriteString("# Workspace Rules\n\n"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write each rule
	for name, content := range rules {
		if _, err := file.WriteString(fmt.Sprintf("## %s\n\n", name)); err != nil {
			return fmt.Errorf("failed to write rule name: %w", err)
		}

		if content != "" {
			if _, err := file.WriteString(content); err != nil {
				return fmt.Errorf("failed to write rule content: %w", err)
			}
			if _, err := file.WriteString("\n\n"); err != nil {
				return fmt.Errorf("failed to write newline: %w", err)
			}
		}
	}

	return nil
}

// AddRule adds or updates a rule in the rules.md file.
func (h *RulesFileHandler) AddRule(repoPath, name, content string) error {
	rules, err := h.Read(repoPath)
	if err != nil {
		return err
	}

	rules[name] = content

	return h.Write(repoPath, rules)
}

// RemoveRule removes a rule from the rules.md file.
func (h *RulesFileHandler) RemoveRule(repoPath, name string) error {
	rules, err := h.Read(repoPath)
	if err != nil {
		return err
	}

	delete(rules, name)

	return h.Write(repoPath, rules)
}

// Exists checks if the rules.md file exists.
func (h *RulesFileHandler) Exists(repoPath string) bool {
	rulesPath := h.workspaceFS.GetRulesPath(repoPath)
	_, err := os.Stat(rulesPath)
	return !os.IsNotExist(err)
}
