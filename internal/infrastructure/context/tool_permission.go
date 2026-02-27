package context

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ToolPermissionPrompt prompts the user to approve MCP tool execution before a skill runs.
type ToolPermissionPrompt struct {
	autoApprove bool
}

// NewToolPermissionPrompt creates a new tool permission prompt.
// When autoApprove is true all prompts are skipped and tools are approved automatically.
func NewToolPermissionPrompt(autoApprove bool) *ToolPermissionPrompt {
	return &ToolPermissionPrompt{autoApprove: autoApprove}
}

// ToolInfo describes a single MCP tool shown in the permission prompt.
type ToolInfo struct {
	Name        string // full name: mcp__server__tool
	Description string
}

// PromptForTools asks the user to approve the given list of tools before execution.
// Returns nil if the user approves (or auto-approve is set), or an error if denied.
func (p *ToolPermissionPrompt) PromptForTools(tools []ToolInfo) error {
	if len(tools) == 0 {
		return nil
	}
	if p.autoApprove {
		return nil
	}

	fmt.Println("\n🔧 Tool Permission Request")
	fmt.Println("──────────────────────────")
	fmt.Printf("The skill wants to use %d tool(s):\n\n", len(tools))
	for i, t := range tools {
		fmt.Printf("  %d. %s", i+1, t.Name)
		if t.Description != "" {
			fmt.Printf(" — %s", t.Description)
		}
		fmt.Println()
	}
	fmt.Println()
	fmt.Print("Allow tool execution? [Y/n/individual/show] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "", "y", "yes":
		fmt.Printf("✓ Approved %d tool(s)\n\n", len(tools))
		return nil

	case "n", "no":
		fmt.Println("✗ Tool execution denied")
		return fmt.Errorf("user denied tool execution")

	case "i", "individual":
		return p.promptIndividual(tools)

	case "s", "show":
		return p.showAndPrompt(tools)

	default:
		fmt.Println("✗ Invalid response, denying tool execution")
		return fmt.Errorf("invalid response: %s", response)
	}
}

// promptIndividual prompts approval for each tool individually.
func (p *ToolPermissionPrompt) promptIndividual(tools []ToolInfo) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nApproving tools individually:")
	fmt.Println()

	var denied []string
	for i, t := range tools {
		fmt.Printf("  [%d/%d] %s", i+1, len(tools), t.Name)
		if t.Description != "" {
			fmt.Printf(" — %s", t.Description)
		}
		fmt.Println()
		fmt.Print("  Allow? [Y/n] ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "n" || response == "no" {
			denied = append(denied, t.Name)
			fmt.Println("  ✗ Denied")
		} else {
			fmt.Println("  ✓ Approved")
		}
		fmt.Println()
	}

	if len(denied) > 0 {
		return fmt.Errorf("tool execution denied for: %s", strings.Join(denied, ", "))
	}
	return nil
}

// showAndPrompt displays tool descriptions then prompts for bulk approval.
func (p *ToolPermissionPrompt) showAndPrompt(tools []ToolInfo) error {
	fmt.Println("\nTool Details:")
	fmt.Println()
	for i, t := range tools {
		fmt.Printf("─── [%d] %s ───\n", i+1, t.Name)
		if t.Description != "" {
			fmt.Printf("  %s\n", t.Description)
		} else {
			fmt.Println("  (no description available)")
		}
		fmt.Println()
	}

	fmt.Print("Allow all shown tools? [Y/n/individual] ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "", "y", "yes":
		fmt.Printf("✓ Approved %d tool(s)\n\n", len(tools))
		return nil
	case "i", "individual":
		return p.promptIndividual(tools)
	default:
		fmt.Println("✗ Tool execution denied")
		return fmt.Errorf("user denied tool execution")
	}
}
