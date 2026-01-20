// Package context provides context detection and injection for skills.
package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PermissionPrompt prompts the user to approve file access.
type PermissionPrompt struct {
	workingDir string
	autoApprove bool
}

// NewPermissionPrompt creates a new permission prompt.
func NewPermissionPrompt(autoApprove bool) *PermissionPrompt {
	wd, _ := os.Getwd()
	return &PermissionPrompt{
		workingDir: wd,
		autoApprove: autoApprove,
	}
}

// PromptForFiles asks the user to approve file access.
// Returns the approved files or an error if denied.
func (p *PermissionPrompt) PromptForFiles(files []FileReference) ([]FileReference, error) {
	if len(files) == 0 {
		return files, nil
	}

	// Auto-approve if flag is set
	if p.autoApprove {
		return files, nil
	}

	// Display files to be accessed
	fmt.Println("\n📄 File Context Request")
	fmt.Println("───────────────────────")
	fmt.Printf("The skill wants to access %d file(s):\n\n", len(files))

	for i, ref := range files {
		relPath, _ := filepath.Rel(p.workingDir, ref.Path)
		size := formatSize(ref.Size)
		fmt.Printf("  %d. %s (%s)\n", i+1, relPath, size)
	}

	fmt.Println()

	// Check for sensitive files
	hasSensitive := containsSensitiveFiles(files)
	if hasSensitive {
		fmt.Println("⚠️  Warning: Detected potentially sensitive files (.env, credentials, etc.)")
		fmt.Println()
	}

	// Prompt user
	fmt.Print("Allow access to these files? [Y/n/individual/show] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "", "y", "yes":
		// Approve all
		fmt.Printf("✓ Approved access to %d file(s)\n\n", len(files))
		return files, nil

	case "n", "no":
		// Deny all
		fmt.Println("✗ Access denied")
		return nil, fmt.Errorf("user denied file access")

	case "i", "individual":
		// Individual approval
		return p.promptIndividual(files)

	case "s", "show":
		// Show file contents preview
		return p.showAndPrompt(files)

	default:
		fmt.Println("✗ Invalid response, denying access")
		return nil, fmt.Errorf("invalid response: %s", response)
	}
}

// promptIndividual prompts for each file individually.
func (p *PermissionPrompt) promptIndividual(files []FileReference) ([]FileReference, error) {
	var approved []FileReference
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nApproving files individually:")
	fmt.Println()

	for i, ref := range files {
		relPath, _ := filepath.Rel(p.workingDir, ref.Path)
		size := formatSize(ref.Size)

		fmt.Printf("  [%d/%d] %s (%s)\n", i+1, len(files), relPath, size)
		fmt.Print("  Allow? [Y/n/show] ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return approved, fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "", "y", "yes":
			approved = append(approved, ref)
			fmt.Println("  ✓ Approved")

		case "s", "show":
			// Show preview and re-prompt
			if err := showFilePreview(ref); err != nil {
				fmt.Printf("  ✗ Error showing preview: %v\n", err)
			}
			// Re-prompt for this file
			fmt.Print("  Allow? [Y/n] ")
			response2, _ := reader.ReadString('\n')
			response2 = strings.TrimSpace(strings.ToLower(response2))
			if response2 == "" || response2 == "y" || response2 == "yes" {
				approved = append(approved, ref)
				fmt.Println("  ✓ Approved")
			} else {
				fmt.Println("  ✗ Denied")
			}

		default:
			fmt.Println("  ✗ Denied")
		}
		fmt.Println()
	}

	if len(approved) == 0 {
		return nil, fmt.Errorf("no files approved")
	}

	fmt.Printf("✓ Approved %d of %d file(s)\n\n", len(approved), len(files))
	return approved, nil
}

// showAndPrompt shows file previews then prompts for approval.
func (p *PermissionPrompt) showAndPrompt(files []FileReference) ([]FileReference, error) {
	fmt.Println("\nFile Previews:")
	fmt.Println()

	for i, ref := range files {
		relPath, _ := filepath.Rel(p.workingDir, ref.Path)
		fmt.Printf("─── [%d] %s ───\n", i+1, relPath)

		if err := showFilePreview(ref); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Println()
	}

	// Now prompt
	fmt.Print("Allow access to all shown files? [Y/n/individual] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "", "y", "yes":
		fmt.Printf("✓ Approved access to %d file(s)\n\n", len(files))
		return files, nil

	case "i", "individual":
		return p.promptIndividual(files)

	default:
		fmt.Println("✗ Access denied")
		return nil, fmt.Errorf("user denied file access")
	}
}

// showFilePreview shows the first 10 lines of a file.
func showFilePreview(ref FileReference) error {
	content, err := os.ReadFile(ref.Path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	previewLines := 10
	if len(lines) < previewLines {
		previewLines = len(lines)
	}

	for i := 0; i < previewLines; i++ {
		// Truncate long lines
		line := lines[i]
		if len(line) > 100 {
			line = line[:100] + "..."
		}
		fmt.Printf("  %3d │ %s\n", i+1, line)
	}

	if len(lines) > previewLines {
		fmt.Printf("  ... (%d more lines)\n", len(lines)-previewLines)
	}

	return nil
}

// formatSize formats a file size in human-readable format.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// containsSensitiveFiles checks if any files appear to be sensitive.
func containsSensitiveFiles(files []FileReference) bool {
	sensitivePatterns := []string{
		".env",
		"credentials",
		"secret",
		"password",
		"token",
		"api_key",
		"apikey",
		"private_key",
		"privatekey",
		".pem",
		".key",
		"id_rsa",
		"id_dsa",
		"id_ecdsa",
		"id_ed25519",
	}

	for _, ref := range files {
		basename := strings.ToLower(filepath.Base(ref.Path))
		for _, pattern := range sensitivePatterns {
			if strings.Contains(basename, pattern) {
				return true
			}
		}
	}

	return false
}
