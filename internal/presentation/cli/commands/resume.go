package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainWorkflow "github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

const defaultResumeTTL = 24 * time.Hour

// NewResumeCmd creates the resume command for continuing interrupted skill executions.
func NewResumeCmd() *cobra.Command {
	var listOnly bool

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume the most recent interrupted skill execution",
		Long: `Resume an interrupted skill execution from the last successful checkpoint.

  sr resume            # resume the most recent in-progress execution
  sr resume --list     # list all resumable sessions within the last 24 hours

Configure TTL via session.resume_ttl in config.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listOnly {
				return listResumableSessions()
			}
			return resumeLatestSession()
		},
	}

	cmd.Flags().BoolVar(&listOnly, "list", false, "list resumable sessions instead of resuming")
	return cmd
}

// listResumableSessions shows all in-progress checkpoints within the TTL window.
func listResumableSessions() error {
	formatter := GetFormatter()
	container := GetContainer()
	if container == nil {
		return fmt.Errorf("application not initialized")
	}

	repo := container.WorkflowCheckpointRepository()
	if repo == nil {
		return fmt.Errorf("checkpoint storage not available")
	}

	checkpoints, err := fetchResumable(context.Background(), repo)
	if err != nil {
		return fmt.Errorf("failed to query checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
		_ = formatter.Info("No resumable sessions found (within last 24 hours).")
		return nil
	}

	_ = formatter.Header(fmt.Sprintf("Resumable Sessions (%d)", len(checkpoints)))

	tableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: "#", Width: 3, Align: output.AlignRight},
			{Header: "Skill", Width: 20, Align: output.AlignLeft},
			{Header: "Progress", Width: 10, Align: output.AlignCenter},
			{Header: "Last Activity", Width: 14, Align: output.AlignLeft},
			{Header: "Input", Width: 40, Align: output.AlignLeft},
		},
		Rows: make([][]string, 0, len(checkpoints)),
	}

	for i, cp := range checkpoints {
		inputSnip := cp.Input()
		if len(inputSnip) > 38 {
			inputSnip = inputSnip[:35] + "..."
		}
		age := time.Since(cp.UpdatedAt())
		tableData.Rows = append(tableData.Rows, []string{
			fmt.Sprintf("%d", i+1),
			cp.SkillName(),
			cp.Progress(),
			formatAge(age) + " ago",
			inputSnip,
		})
	}

	_ = formatter.Table(tableData)
	_ = formatter.Println("")
	_ = formatter.Info("Run 'sr run <skill> <input> --resume' to continue a specific session.")
	_ = formatter.Info("Or run 'sr resume' (no flags) to auto-resume the most recent one.")
	return nil
}

// resumeLatestSession finds the most recent in-progress checkpoint and re-runs it.
func resumeLatestSession() error {
	formatter := GetFormatter()
	container := GetContainer()
	if container == nil {
		return fmt.Errorf("application not initialized")
	}

	repo := container.WorkflowCheckpointRepository()
	if repo == nil {
		return fmt.Errorf("checkpoint storage not available")
	}

	checkpoints, err := fetchResumable(context.Background(), repo)
	if err != nil {
		return fmt.Errorf("failed to query checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
		_ = formatter.Info("No resumable sessions found (within last 24 hours).")
		return nil
	}

	// Pick the most recently updated
	cp := checkpoints[0]
	_ = formatter.Info("Resuming: skill=%s  progress=%s  last activity=%s ago",
		cp.SkillName(), cp.Progress(), formatAge(time.Since(cp.UpdatedAt())))

	// Delegate to runSkill with Resume=true
	savedOpts := runOpts
	runOpts.Resume = true
	runOpts.Force = false
	defer func() { runOpts = savedOpts }()

	dummyCmd := &cobra.Command{}
	return runSkill(dummyCmd, []string{cp.SkillName(), cp.Input()})
}

// fetchResumable returns in-progress checkpoints updated within the TTL, newest first.
func fetchResumable(ctx context.Context, repo ports.WorkflowCheckpointPort) ([]*domainWorkflow.WorkflowCheckpoint, error) {
	ttl := defaultResumeTTL
	if appCtx := GetAppContext(); appCtx != nil && appCtx.Config != nil {
		if d := appCtx.Config.Session.ResumeTTL; d > 0 {
			ttl = d
		}
	}

	filter := &ports.WorkflowCheckpointFilter{
		Status:       []domainWorkflow.CheckpointStatus{domainWorkflow.CheckpointStatusInProgress},
		CreatedAfter: time.Now().Add(-ttl),
		Limit:        20,
	}

	return repo.List(ctx, filter)
}
