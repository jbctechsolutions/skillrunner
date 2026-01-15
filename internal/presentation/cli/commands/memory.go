// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	infraMemory "github.com/jbctechsolutions/skillrunner/internal/infrastructure/memory"
)

// memoryFlags holds the flags for memory commands.
type memoryFlags struct {
	Global bool
}

var memoryOpts memoryFlags

// NewMemoryCmd creates the memory command group for managing memory files.
func NewMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage memory files for context persistence",
		Long: `Manage memory files (MEMORY.md/CLAUDE.md) that provide persistent context
across skillrunner sessions. Memory content is injected into LLM prompts.

Memory files are loaded in this order (most specific first):
  1. Project root: ./MEMORY.md or ./CLAUDE.md
  2. Global: ~/.skillrunner/MEMORY.md or ~/.skillrunner/CLAUDE.md

Use @include: ./path/to/file.md to include additional files.`,
	}

	cmd.AddCommand(NewMemoryEditCmd())
	cmd.AddCommand(NewMemoryViewCmd())

	return cmd
}

// NewMemoryEditCmd creates the memory edit command.
func NewMemoryEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open memory file in editor",
		Long: `Open the memory file in your default editor ($EDITOR).

By default, opens the project memory file (./MEMORY.md).
Use --global to edit the global memory file (~/.skillrunner/MEMORY.md).

If the file doesn't exist, it will be created.`,
		RunE: runMemoryEdit,
	}

	cmd.Flags().BoolVar(&memoryOpts.Global, "global", false, "edit global memory file (~/.skillrunner/MEMORY.md)")

	return cmd
}

// NewMemoryViewCmd creates the memory view command.
func NewMemoryViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Display memory content",
		Long: `Display the combined memory content from all sources.

Shows both project and global memory files merged together,
along with any @include directives resolved.

Use --global to view only the global memory content.`,
		RunE: runMemoryView,
	}

	cmd.Flags().BoolVar(&memoryOpts.Global, "global", false, "view only global memory")

	return cmd
}

// runMemoryEdit opens the memory file in the user's editor.
func runMemoryEdit(_ *cobra.Command, _ []string) error {
	formatter := GetFormatter()

	var memoryPath string
	if memoryOpts.Global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		skillrunnerDir := filepath.Join(homeDir, ".skillrunner")

		// Ensure directory exists
		if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		memoryPath = filepath.Join(skillrunnerDir, infraMemory.MemoryFileName)
	} else {
		// Use current working directory for project memory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		memoryPath = filepath.Join(cwd, infraMemory.MemoryFileName)
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(memoryPath); os.IsNotExist(err) {
		initialContent := `# Memory

Add instructions, context, and rules that should be included in all LLM prompts.

## Usage

Content in this file is automatically injected into every skill execution.
Use @include: ./path/to/file.md to include additional files.
`
		if err := os.WriteFile(memoryPath, []byte(initialContent), 0644); err != nil {
			return fmt.Errorf("failed to create memory file: %w", err)
		}
		formatter.Success("Created %s", memoryPath)
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		editors := []string{"vim", "vi", "nano", "code"}
		for _, e := range editors {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found. Set $EDITOR environment variable")
	}

	// Open editor
	formatter.Item("Opening", memoryPath)

	cmd := exec.Command(editor, memoryPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// runMemoryView displays the memory content.
func runMemoryView(_ *cobra.Command, _ []string) error {
	formatter := GetFormatter()
	appCtx := GetAppContext()

	maxTokens := 2000
	if appCtx != nil && appCtx.Config != nil {
		maxTokens = appCtx.Config.Memory.MaxTokens
	}

	// Get project directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load memory
	loader := infraMemory.NewLoader(maxTokens)
	mem, err := loader.Load(cwd)
	if err != nil {
		return fmt.Errorf("failed to load memory: %w", err)
	}

	if mem.IsEmpty() {
		formatter.Warning("No memory files found")
		formatter.Println("")
		formatter.Item("Create project memory", "./MEMORY.md")
		formatter.Item("Create global memory", "~/.skillrunner/MEMORY.md")
		formatter.Println("")
		formatter.Println("Run 'sr memory edit' to create a memory file.")
		return nil
	}

	if memoryOpts.Global {
		// Show only global memory
		if mem.GlobalContent() == "" {
			formatter.Warning("No global memory found")
			formatter.Println("Run 'sr memory edit --global' to create one.")
			return nil
		}

		formatter.Header("Global Memory")
		formatter.Println("")
		formatter.Println("%s", mem.GlobalContent())
	} else {
		// Show combined memory
		formatter.Header("Memory Content")
		formatter.Println("")

		sources := mem.Sources()
		if len(sources) > 0 {
			formatter.SubHeader("Sources")
			for _, src := range sources {
				formatter.BulletItem(src)
			}
			formatter.Println("")
		}

		formatter.SubHeader("Content")
		formatter.Println("")
		formatter.Println("%s", mem.Combined())

		formatter.Println("")
		formatter.Item("Estimated tokens", fmt.Sprintf("%d / %d", mem.EstimatedTokens(), maxTokens))
	}

	return nil
}
