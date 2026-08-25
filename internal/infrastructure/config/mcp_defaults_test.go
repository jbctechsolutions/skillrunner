package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

func TestDefaultMCPServers_ContainsExpectedServers(t *testing.T) {
	servers := config.DefaultMCPServers()

	if _, ok := servers["filesystem"]; !ok {
		t.Error("expected 'filesystem' server in defaults")
	}
	if _, ok := servers["git"]; !ok {
		t.Error("expected 'git' server in defaults")
	}
}

func TestDefaultMCPServers_FilesystemEntry(t *testing.T) {
	servers := config.DefaultMCPServers()
	fs := servers["filesystem"]

	if fs.Command != "npx" {
		t.Errorf("filesystem command: got %q, want %q", fs.Command, "npx")
	}
	if len(fs.Args) == 0 {
		t.Fatal("filesystem args should not be empty")
	}
	found := false
	for _, a := range fs.Args {
		if strings.Contains(a, "server-filesystem") {
			found = true
			break
		}
	}
	if !found {
		t.Error("filesystem args should reference @modelcontextprotocol/server-filesystem")
	}
}

func TestDefaultMCPServers_GitEntry(t *testing.T) {
	servers := config.DefaultMCPServers()
	git := servers["git"]

	if git.Command != "npx" {
		t.Errorf("git command: got %q, want %q", git.Command, "npx")
	}
	found := false
	for _, a := range git.Args {
		if strings.Contains(a, "server-git") {
			found = true
			break
		}
	}
	if !found {
		t.Error("git args should reference @modelcontextprotocol/server-git")
	}
}

func TestWriteMCPServersConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_servers.json")

	created, err := config.WriteMCPServersConfig(path)
	if err != nil {
		t.Fatalf("WriteMCPServersConfig: %v", err)
	}
	if !created {
		t.Error("expected created=true for a new file")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var parsed map[string]config.MCPServerEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := parsed["filesystem"]; !ok {
		t.Error("written config should contain 'filesystem'")
	}
	if _, ok := parsed["git"]; !ok {
		t.Error("written config should contain 'git'")
	}
}

func TestWriteMCPServersConfig_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_servers.json")

	sentinel := []byte(`{"custom": {"command": "my-server", "args": []}}`)
	if err := os.WriteFile(path, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := config.WriteMCPServersConfig(path)
	if err != nil {
		t.Fatalf("WriteMCPServersConfig: %v", err)
	}
	if created {
		t.Error("expected created=false when file already exists")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(sentinel) {
		t.Error("existing file should not be overwritten")
	}
}

func TestWriteMCPServersConfig_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "mcp_servers.json")

	created, err := config.WriteMCPServersConfig(path)
	if err != nil {
		t.Fatalf("WriteMCPServersConfig: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

func TestWriteMCPServersConfig_ExpandsHomeVar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_servers.json")

	if _, err := config.WriteMCPServersConfig(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// ${HOME} should have been replaced with the actual home directory
	if strings.Contains(string(data), "${HOME}") {
		t.Error("${HOME} should be expanded in written config")
	}
}

func TestValidateMCPServersConfig_ValidConfig(t *testing.T) {
	servers := map[string]config.MCPServerEntry{
		"my-server": {Command: "npx", Args: []string{"-y", "some-package"}},
		"other_srv": {Command: "python", Args: []string{"server.py"}},
	}
	if err := config.ValidateMCPServersConfig(servers); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateMCPServersConfig_MissingCommand(t *testing.T) {
	servers := map[string]config.MCPServerEntry{
		"no-cmd": {Command: "", Args: []string{}},
	}
	if err := config.ValidateMCPServersConfig(servers); err == nil {
		t.Error("expected error for missing command")
	}
}

func TestValidateMCPServersConfig_InvalidName(t *testing.T) {
	servers := map[string]config.MCPServerEntry{
		"bad name!": {Command: "npx"},
	}
	if err := config.ValidateMCPServersConfig(servers); err == nil {
		t.Error("expected error for invalid server name")
	}
}

func TestValidateMCPServersConfig_EmptyIsValid(t *testing.T) {
	if err := config.ValidateMCPServersConfig(map[string]config.MCPServerEntry{}); err != nil {
		t.Errorf("empty config should be valid, got: %v", err)
	}
}
