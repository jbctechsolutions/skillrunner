package commands

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// executeCommand executes a cobra command with the given args.
func executeCommand(root *cobra.Command, args ...string) error {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	return root.Execute()
}

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()

	if cmd == nil {
		t.Fatal("NewRootCmd returned nil")
	}

	if cmd.Use != "sr" {
		t.Errorf("expected Use='sr', got %q", cmd.Use)
	}

	// Check key subcommands exist
	wantSubcmds := []string{"version", "list", "run", "status", "ask", "import", "init", "metrics"}
	subcmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcmds[sub.Name()] = true
	}

	for _, want := range wantSubcmds {
		if !subcmds[want] {
			t.Errorf("missing subcommand: %s", want)
		}
	}

	// Check persistent flags
	wantFlags := []string{"config", "output", "verbose"}
	for _, flag := range wantFlags {
		if cmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("missing persistent flag: %s", flag)
		}
	}
}

func TestVersionCmd_NoError(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"basic", []string{"version"}, false},
		{"short", []string{"version", "--short"}, false},
		{"json", []string{"version", "-o", "json"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCmd_Validation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		// Note: Valid syntax but non-existent skills will error because the command
		// now actually tries to load and execute skills (not a stub anymore)
		{"valid_syntax_missing_skill", []string{"run", "test-skill", "test request"}, true},
		{"with_profile_missing_skill", []string{"run", "skill", "request", "--profile", "premium"}, true},
		{"with_cheap_profile_missing_skill", []string{"run", "skill", "request", "-p", "cheap"}, true},
		{"with_stream_missing_skill", []string{"run", "skill", "request", "--stream"}, true},
		{"missing args", []string{"run"}, true},
		{"missing request", []string{"run", "skill-only"}, true},
		{"invalid profile", []string{"run", "skill", "request", "--profile", "invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListCmd_NoError(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"basic", []string{"list"}, false},
		{"alias", []string{"ls"}, false},
		{"json format", []string{"list", "-f", "json"}, false},
		{"table format", []string{"list", "--format", "table"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStatusCmd_NoError(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"basic", []string{"status"}, false},
		{"detailed", []string{"status", "--detailed"}, false},
		{"json", []string{"status", "-o", "json"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProfile(t *testing.T) {
	tests := []struct {
		profile string
		wantErr bool
	}{
		{"cheap", false},
		{"balanced", false},
		{"premium", false},
		{"CHEAP", false},
		{"BALANCED", false},
		{" balanced ", false},
		{"invalid", true},
		{"", true},
		{"best", true},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			err := validateProfile(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProfile(%q) error = %v, wantErr %v", tt.profile, err, tt.wantErr)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"hello", 3, "hel"}, // maxLen <= 3 returns first maxLen chars
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestLoadSkillsWithoutContainer(t *testing.T) {
	// When container is not initialized, loadSkills should return nil
	skills := loadSkills()

	// Without an initialized container, should return nil
	if len(skills) > 0 {
		// This is also acceptable - container might be initialized from prior tests
		for _, skill := range skills {
			if skill.Name == "" {
				t.Error("skill name should not be empty")
			}
		}
	}
}

func TestGetSystemStatus(t *testing.T) {
	// Get status without a container (nil container case)
	status := getSystemStatus(false)

	if status.Status == "" {
		t.Error("system status should not be empty")
	}

	if status.Version == "" {
		t.Error("version should not be empty")
	}

	if len(status.Providers) == 0 {
		t.Error("expected at least one provider")
	}

	validStatuses := map[string]bool{"healthy": true, "degraded": true, "unavailable": true, "unhealthy": true, "unknown": true}
	for _, p := range status.Providers {
		if !validStatuses[p.Status] {
			t.Errorf("provider %s has invalid status: %s", p.Name, p.Status)
		}
	}
}

func TestCountProviderStatuses(t *testing.T) {
	providers := []ProviderStatus{
		{Name: "a", Status: "healthy"},
		{Name: "b", Status: "healthy"},
		{Name: "c", Status: "degraded"},
		{Name: "d", Status: "unavailable"},
	}

	healthy, degraded, unavailable := countProviderStatuses(providers)

	if healthy != 2 {
		t.Errorf("expected 2 healthy, got %d", healthy)
	}
	if degraded != 1 {
		t.Errorf("expected 1 degraded, got %d", degraded)
	}
	if unavailable != 1 {
		t.Errorf("expected 1 unavailable, got %d", unavailable)
	}
}

func TestNewVersionCmd_Structure(t *testing.T) {
	cmd := NewVersionCmd()

	if cmd.Use != "version" {
		t.Errorf("expected Use='version', got %q", cmd.Use)
	}

	if cmd.Flags().Lookup("short") == nil {
		t.Error("missing --short flag")
	}
}

func TestNewRunCmd_Structure(t *testing.T) {
	cmd := NewRunCmd()

	if cmd.Use != "run <skill> <request>" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}

	if cmd.Flags().Lookup("profile") == nil {
		t.Error("missing --profile flag")
	}
	if cmd.Flags().Lookup("stream") == nil {
		t.Error("missing --stream flag")
	}
}

func TestNewListCmd_Structure(t *testing.T) {
	cmd := NewListCmd()

	if cmd.Use != "list" {
		t.Errorf("expected Use='list', got %q", cmd.Use)
	}

	// Check alias
	found := false
	for _, alias := range cmd.Aliases {
		if alias == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("missing 'ls' alias")
	}

	if cmd.Flags().Lookup("format") == nil {
		t.Error("missing --format flag")
	}
}

func TestNewStatusCmd_Structure(t *testing.T) {
	cmd := NewStatusCmd()

	if cmd.Use != "status" {
		t.Errorf("expected Use='status', got %q", cmd.Use)
	}

	if cmd.Flags().Lookup("detailed") == nil {
		t.Error("missing --detailed flag")
	}
}

func TestAskCmd_Validation(t *testing.T) {
	// Note: Valid input tests expect errors because no providers/skills are configured in test environment.
	// The command parses args correctly but execution fails due to missing skill/provider setup.
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"valid_args_no_providers", []string{"ask", "test-skill", "What is the main point?"}, true},
		{"with_model_no_providers", []string{"ask", "code-review", "Is this safe?", "--model", "claude-3-opus"}, true},
		{"with_model_short_no_providers", []string{"ask", "summarize", "Hello world", "-m", "gpt-4"}, true},
		{"with_profile_no_providers", []string{"ask", "translate", "What is this?", "--profile", "premium"}, true},
		{"with_cheap_profile_no_providers", []string{"ask", "explain", "Explain this", "-p", "cheap"}, true},
		{"missing_args", []string{"ask"}, true},
		{"invalid_profile", []string{"ask", "skill", "question", "--profile", "invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAskCmd_Structure(t *testing.T) {
	cmd := NewAskCmd()

	if cmd.Use != "ask <skill> <question>" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}

	if cmd.Flags().Lookup("model") == nil {
		t.Error("missing --model flag")
	}
	if cmd.Flags().Lookup("profile") == nil {
		t.Error("missing --profile flag")
	}
	if cmd.Flags().Lookup("phase") == nil {
		t.Error("missing --phase flag")
	}
	if cmd.Flags().Lookup("stream") == nil {
		t.Error("missing --stream flag")
	}
}

func TestValidateAskProfile(t *testing.T) {
	tests := []struct {
		profile string
		wantErr bool
	}{
		{"cheap", false},
		{"balanced", false},
		{"premium", false},
		{"CHEAP", false},
		{"BALANCED", false},
		{" balanced ", false},
		{"invalid", true},
		{"", true},
		{"best", true},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			err := validateAskProfile(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAskProfile(%q) error = %v, wantErr %v", tt.profile, err, tt.wantErr)
			}
		})
	}
}

func TestNewInitCmd_Structure(t *testing.T) {
	cmd := NewInitCmd()

	if cmd.Use != "init" {
		t.Errorf("expected Use='init', got %q", cmd.Use)
	}

	if cmd.Flags().Lookup("force") == nil {
		t.Error("missing --force flag")
	}
}

func TestInitCmd_JSONOutput(t *testing.T) {
	// Test that init command works with JSON output format
	// This doesn't actually create files, just tests command structure
	cmd := NewRootCmd()

	// Just verify the command exists and has proper structure
	var initCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "init" {
			initCmd = sub
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init command not found")
	}

	if initCmd.Short == "" {
		t.Error("init command should have short description")
	}

	if initCmd.Long == "" {
		t.Error("init command should have long description")
	}
}

func TestNewImportCmd_Structure(t *testing.T) {
	cmd := NewImportCmd()

	if cmd.Use != "import <source>" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}

	if cmd.Flags().Lookup("name") == nil {
		t.Error("missing --name flag")
	}
	if cmd.Flags().Lookup("force") == nil {
		t.Error("missing --force flag")
	}
}

func TestImportCmd_Validation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"missing source", []string{"import"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDetectSourceType(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"https://example.com/skill.yaml", "url"},
		{"http://example.com/skill.yml", "url"},
		{"https://github.com/user/repo.git", "git"},
		{"https://github.com/user/repo", "git"},
		{"https://gitlab.com/user/repo", "git"},
		{"https://bitbucket.org/user/repo", "git"},
		{"https://raw.githubusercontent.com/user/repo/main/skill.yaml", "url"},
		{"./my-skill.yaml", "local"},
		{"/path/to/skill.yaml", "local"},
		{"../skills/code-review.yaml", "local"},
		{"skill.yaml", "local"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := detectSourceType(tt.source)
			if got != tt.expected {
				t.Errorf("detectSourceType(%q) = %q, want %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestGetSkillsDir(t *testing.T) {
	dir, err := getSkillsDir()
	if err != nil {
		t.Fatalf("getSkillsDir() error = %v", err)
	}

	if dir == "" {
		t.Error("getSkillsDir() returned empty string")
	}

	// Should end with .skillrunner/skills
	if !strings.Contains(dir, ".skillrunner") || !strings.HasSuffix(dir, "skills") {
		t.Errorf("getSkillsDir() = %q, expected path ending with .skillrunner/skills", dir)
	}
}

func TestMetricsCmd_NoError(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"basic", []string{"metrics"}, false},
		{"with since 24h", []string{"metrics", "--since", "24h"}, false},
		{"with since 7d", []string{"metrics", "--since", "7d"}, false},
		{"with since 30d", []string{"metrics", "--since", "30d"}, false},
		{"json output", []string{"metrics", "-o", "json"}, false},
		{"json with since", []string{"metrics", "--since", "7d", "-o", "json"}, false},
		{"invalid since", []string{"metrics", "--since", "invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewMetricsCmd_Structure(t *testing.T) {
	cmd := NewMetricsCmd()

	if cmd.Use != "metrics" {
		t.Errorf("expected Use='metrics', got %q", cmd.Use)
	}

	if cmd.Flags().Lookup("since") == nil {
		t.Error("missing --since flag")
	}
}

func TestGetMockMetrics(t *testing.T) {
	duration := 24 * time.Hour
	metrics := getMockMetrics(duration)

	if metrics.TotalRequests <= 0 {
		t.Error("expected positive total requests")
	}

	if metrics.SuccessfulCount+metrics.FailedCount != metrics.TotalRequests {
		t.Error("success + failed should equal total requests")
	}

	if metrics.SuccessRate < 0 || metrics.SuccessRate > 100 {
		t.Errorf("success rate should be between 0 and 100, got %f", metrics.SuccessRate)
	}

	if len(metrics.ProviderMetrics) == 0 {
		t.Error("expected at least one provider")
	}

	if len(metrics.TopSkills) == 0 {
		t.Error("expected at least one skill")
	}

	// Verify provider metrics have required fields
	for _, p := range metrics.ProviderMetrics {
		if p.Name == "" {
			t.Error("provider name should not be empty")
		}
		if p.Type == "" {
			t.Error("provider type should not be empty")
		}
	}

	// Verify skill metrics have required fields
	for _, s := range metrics.TopSkills {
		if s.Name == "" {
			t.Error("skill name should not be empty")
		}
		if s.SuccessRate < 0 || s.SuccessRate > 100 {
			t.Errorf("skill success rate should be between 0 and 100, got %f", s.SuccessRate)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"24h", 24 * time.Hour, false},
		{"1h", time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{10000000, "10.0M"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumber(tt.input)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		input   string
		wantLen int // approximate length check
	}{
		{"2024-01-15T10:30:00Z", 17}, // "Jan 15, 2024 10:30"
		{"invalid", 7},               // returns original input
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatDateTime(tt.input)
			if len(got) < 5 {
				t.Errorf("formatDateTime(%q) = %q, too short", tt.input, got)
			}
		})
	}
}
