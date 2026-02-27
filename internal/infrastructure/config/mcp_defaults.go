package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MCPServerEntry describes a single MCP server in the config file.
type MCPServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// npxEnv provides a writable npm cache so npx works even when the default
// ~/.npm directory is read-only (e.g., mounted ro in a container).
var npxEnv = map[string]string{
	"NPM_CONFIG_CACHE": "/tmp/npm-cache",
}

// DefaultMCPServers returns the bundled default MCP server configurations.
// Variables ${HOME} and ${PWD} are expanded at runtime.
func DefaultMCPServers() map[string]MCPServerEntry {
	return map[string]MCPServerEntry{
		"filesystem": {
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "${HOME}"},
			Env:     npxEnv,
		},
		"git": {
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-git"},
			Env:     npxEnv,
		},
	}
}

// WriteMCPServersConfig writes the default MCP servers config to the given path.
// If the file already exists it is NOT overwritten — returns nil and a hint instead.
func WriteMCPServersConfig(path string) (created bool, err error) {
	if _, err := os.Stat(path); err == nil {
		// Already exists
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}

	servers := expandMCPVars(DefaultMCPServers())
	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal MCP config: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return false, fmt.Errorf("write MCP config: %w", err)
	}
	return true, nil
}

// expandMCPVars replaces ${HOME} and ${PWD} in all string fields.
func expandMCPVars(servers map[string]MCPServerEntry) map[string]MCPServerEntry {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	replacer := func(s string) string {
		s = strings.ReplaceAll(s, "${HOME}", home)
		s = strings.ReplaceAll(s, "${PWD}", cwd)
		return s
	}

	result := make(map[string]MCPServerEntry, len(servers))
	for name, entry := range servers {
		expanded := MCPServerEntry{
			Command: replacer(entry.Command),
			Args:    make([]string, len(entry.Args)),
		}
		for i, a := range entry.Args {
			expanded.Args[i] = replacer(a)
		}
		if entry.Env != nil {
			expanded.Env = make(map[string]string, len(entry.Env))
			for k, v := range entry.Env {
				expanded.Env[k] = replacer(v)
			}
		}
		result[name] = expanded
	}
	return result
}

// ValidateMCPServersConfig validates a parsed MCP servers config map.
// Returns an error if any entry is missing a command.
func ValidateMCPServersConfig(servers map[string]MCPServerEntry) error {
	nameRE := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	for name, entry := range servers {
		if !nameRE.MatchString(name) {
			return fmt.Errorf("invalid server name %q: must match [a-zA-Z0-9_-]+", name)
		}
		if entry.Command == "" {
			return fmt.Errorf("server %q is missing a command", name)
		}
	}
	return nil
}
