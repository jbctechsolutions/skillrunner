// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

// NewMCPCmd creates the top-level `sr mcp` command group.
func NewMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP (Model Context Protocol) servers and tools",
		Long: `Manage MCP servers and inspect/invoke the tools they expose.

MCP servers provide tools that skills can call during execution.
Use these commands to start/stop servers, discover tools, and test them.`,
	}

	cmd.AddCommand(newMCPListServersCmd())
	cmd.AddCommand(newMCPStartCmd())
	cmd.AddCommand(newMCPStopCmd())
	cmd.AddCommand(newMCPStatusCmd())
	cmd.AddCommand(newMCPListToolsCmd())
	cmd.AddCommand(newMCPDescribeToolCmd())
	cmd.AddCommand(newMCPCallCmd())

	return cmd
}

// newMCPListServersCmd lists all configured MCP servers.
func newMCPListServersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-servers",
		Short: "List configured MCP servers",
		Long:  `Show all configured MCP servers and their current state.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			reg := container.MCPRegistry()
			if reg == nil {
				return fmt.Errorf("MCP registry not available")
			}
			formatter := GetFormatter()

			names := reg.ListConfiguredServers()
			if len(names) == 0 {
				formatter.Warning("No MCP servers configured.")
				formatter.Println("Add servers to ~/.skillrunner/mcp_servers.json or .claude/mcp.json")
				return nil
			}

			formatter.Header("MCP Servers")
			for _, name := range names {
				// Get runtime info if the server has been started, otherwise show stopped
				info, err := reg.Manager().GetInfo(name)
				state := domainMCP.ServerStateStopped
				var toolCount int
				if err == nil {
					state = info.State
					toolCount = info.ToolCount
				}
				cfg, _ := reg.GetServerConfig(name)
				cmdStr := cfg.Command
				if len(cfg.Args) > 0 {
					cmdStr += " " + strings.Join(cfg.Args, " ")
				}
				formatter.Println("  %s %-20s  state: %-10s  tools: %d  cmd: %s",
					serverStateIcon(state), name, state, toolCount, cmdStr)
			}
			return nil
		},
	}
}

// newMCPStartCmd starts a named MCP server.
func newMCPStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			reg := container.MCPRegistry()
			if reg == nil {
				return fmt.Errorf("MCP registry not available")
			}
			formatter := GetFormatter()

			formatter.Println("Starting MCP server %q...", serverName)
			if err := reg.EnsureServerRunning(context.Background(), serverName); err != nil {
				return fmt.Errorf("failed to start server %q: %w", serverName, err)
			}
			formatter.Success("Server %q started.", serverName)
			return nil
		},
	}
}

// newMCPStopCmd stops a named MCP server.
func newMCPStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			reg := container.MCPRegistry()
			if reg == nil {
				return fmt.Errorf("MCP registry not available")
			}
			formatter := GetFormatter()

			if err := reg.Manager().Stop(context.Background(), serverName); err != nil {
				return fmt.Errorf("failed to stop server %q: %w", serverName, err)
			}
			formatter.Success("Server %q stopped.", serverName)
			return nil
		},
	}
}

// newMCPStatusCmd shows detailed status for a named MCP server.
func newMCPStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Show status of an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			reg := container.MCPRegistry()
			if reg == nil {
				return fmt.Errorf("MCP registry not available")
			}
			formatter := GetFormatter()

			info, err := reg.Manager().GetInfo(serverName)
			if err != nil {
				return fmt.Errorf("server %q not found: %w", serverName, err)
			}

			formatter.Header(fmt.Sprintf("MCP Server: %s", info.Name))
			formatter.Item("State", string(info.State))
			formatter.Item("Tools", fmt.Sprintf("%d", info.ToolCount))
			if !info.StartedAt.IsZero() {
				formatter.Item("Started", info.StartedAt.Format("2006-01-02 15:04:05"))
			}
			if info.ErrorMessage != "" {
				formatter.Item("Error", info.ErrorMessage)
			}
			cfg, ok := reg.GetServerConfig(serverName)
			if ok {
				formatter.Item("Command", cfg.Command+" "+strings.Join(cfg.Args, " "))
			}
			return nil
		},
	}
}

// newMCPListToolsCmd lists tools from one or all MCP servers.
func newMCPListToolsCmd() *cobra.Command {
	var serverFilter string
	cmd := &cobra.Command{
		Use:   "list-tools",
		Short: "List available MCP tools",
		Long:  `List tools from all running MCP servers, or from a specific server with --server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			reg := container.MCPRegistry()
			if reg == nil {
				return fmt.Errorf("MCP registry not available")
			}
			formatter := GetFormatter()
			ctx := context.Background()

			var tools []*domainMCP.Tool
			var err error

			if serverFilter != "" {
				if err = reg.EnsureServerRunning(ctx, serverFilter); err != nil {
					return fmt.Errorf("could not start server %q: %w", serverFilter, err)
				}
				tools, err = reg.Manager().ListTools(ctx, serverFilter)
			} else {
				tools, err = reg.GetAllTools(ctx)
			}
			if err != nil {
				return fmt.Errorf("failed to list tools: %w", err)
			}

			if len(tools) == 0 {
				formatter.Warning("No tools available. Start an MCP server first.")
				return nil
			}

			formatter.Header(fmt.Sprintf("MCP Tools (%d)", len(tools)))
			for _, t := range tools {
				desc := t.Description()
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				formatter.Println("  %-40s  %s", t.FullName(), desc)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&serverFilter, "server", "", "filter tools by server name")
	return cmd
}

// newMCPDescribeToolCmd shows the full schema for a named tool.
func newMCPDescribeToolCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe-tool <name>",
		Short: "Show full details and JSON schema for an MCP tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fullName := args[0]
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			reg := container.MCPRegistry()
			if reg == nil {
				return fmt.Errorf("MCP registry not available")
			}
			formatter := GetFormatter()
			ctx := context.Background()

			serverName, toolName, err := domainMCP.ParseToolName(fullName)
			if err != nil {
				return fmt.Errorf("invalid tool name %q: %w", fullName, err)
			}

			if err := reg.EnsureServerRunning(ctx, serverName); err != nil {
				return fmt.Errorf("could not start server %q: %w", serverName, err)
			}

			tool, err := reg.Manager().GetTool(ctx, serverName, toolName)
			if err != nil {
				return fmt.Errorf("tool %q not found: %w", fullName, err)
			}

			formatter.Header(fmt.Sprintf("Tool: %s", tool.FullName()))
			formatter.Item("Server", serverName)
			formatter.Item("Name", toolName)
			if tool.Description() != "" {
				formatter.Item("Description", tool.Description())
			}

			if schema := tool.InputSchema(); len(schema) > 0 {
				var pretty map[string]any
				if err := json.Unmarshal(schema, &pretty); err == nil {
					prettyJSON, _ := json.MarshalIndent(pretty, "  ", "  ")
					formatter.Println("\nInput Schema:")
					formatter.Println("  %s", string(prettyJSON))
				} else {
					formatter.Println("\nInput Schema: %s", string(schema))
				}
			}
			return nil
		},
	}
}

// newMCPCallCmd invokes an MCP tool directly from the CLI.
func newMCPCallCmd() *cobra.Command {
	var argsJSON string
	cmd := &cobra.Command{
		Use:   "call <name>",
		Short: "Call an MCP tool directly",
		Long: `Execute an MCP tool directly from the CLI.

Arguments are passed as a JSON object via --args.

Example:
  sr mcp call mcp__filesystem__read_file --args '{"path": "/tmp/example.txt"}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fullName := args[0]
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			reg := container.MCPRegistry()
			if reg == nil {
				return fmt.Errorf("MCP registry not available")
			}
			formatter := GetFormatter()
			ctx := context.Background()

			// Parse arguments
			var callArgs map[string]any
			if argsJSON != "" {
				if err := json.Unmarshal([]byte(argsJSON), &callArgs); err != nil {
					return fmt.Errorf("invalid --args JSON: %w", err)
				}
			}

			result, err := reg.CallToolByFullName(ctx, fullName, callArgs)
			if err != nil {
				return fmt.Errorf("tool call failed: %w", err)
			}

			if result.IsError {
				formatter.Error("Tool returned an error:")
				formatter.Println(result.TextContent())
				return fmt.Errorf("tool call returned error")
			}

			formatter.Success("Tool call succeeded:")
			formatter.Println(result.TextContent())
			return nil
		},
	}
	cmd.Flags().StringVar(&argsJSON, "args", "", "tool arguments as a JSON object")
	return cmd
}

// serverStateIcon returns a visual indicator for a server state.
func serverStateIcon(state domainMCP.ServerState) string {
	switch state {
	case domainMCP.ServerStateReady:
		return "✓"
	case domainMCP.ServerStateStarting:
		return "◐"
	case domainMCP.ServerStateStopped:
		return "○"
	case domainMCP.ServerStateError:
		return "✗"
	default:
		return "?"
	}
}
