package commands

import (
	"runtime"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// VersionInfo holds version information for JSON output.
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version, build information, and platform details for skillrunner.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(short)
		},
	}

	cmd.Flags().BoolVarP(&short, "short", "s", false, "print only the version number")

	return cmd
}

func runVersion(short bool) error {
	// Determine output format from global flags
	format := output.FormatText
	if globalFlags.Output == "json" {
		format = output.FormatJSON
	}

	formatter := output.NewFormatter(
		output.WithFormat(format),
		output.WithColor(format != output.FormatJSON),
	)

	if short {
		if format == output.FormatJSON {
			return formatter.JSON(map[string]string{"version": Version})
		}
		formatter.Println("%s", Version)
		return nil
	}

	info := VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	if format == output.FormatJSON {
		return formatter.JSON(info)
	}

	// Print version info in text format
	formatter.Println("%s", formatter.Bold("Skillrunner"))
	formatter.Println("%s", "───────────")
	formatter.Println("  %s  %s", formatter.Dim("Version:"), info.Version)
	formatter.Println("  %s  %s", formatter.Dim("Git Commit:"), info.GitCommit)
	formatter.Println("  %s  %s", formatter.Dim("Build Date:"), info.BuildDate)
	formatter.Println("  %s  %s", formatter.Dim("Go Version:"), info.GoVersion)
	formatter.Println("  %s  %s", formatter.Dim("Platform:"), info.Platform)

	return nil
}
