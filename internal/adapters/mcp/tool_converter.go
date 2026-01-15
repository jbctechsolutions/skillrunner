package mcp

import (
	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// ToProviderTools converts MCP tools to provider-agnostic tools.
// If deferLoading is true, all tools will have DeferLoading set to true
// for use with the Tool Search Tool feature.
func ToProviderTools(tools []*domainMCP.Tool, deferLoading bool) []ports.Tool {
	if len(tools) == 0 {
		return nil
	}

	result := make([]ports.Tool, len(tools))
	for i, tool := range tools {
		result[i] = ports.Tool{
			Name:         tool.FullName(), // Use mcp__server__tool format
			Description:  tool.Description(),
			InputSchema:  tool.InputSchema(),
			DeferLoading: deferLoading,
		}
	}
	return result
}

// ToProviderTool converts a single MCP tool to a provider-agnostic tool.
func ToProviderTool(tool *domainMCP.Tool, deferLoading bool) ports.Tool {
	return ports.Tool{
		Name:         tool.FullName(),
		Description:  tool.Description(),
		InputSchema:  tool.InputSchema(),
		DeferLoading: deferLoading,
	}
}
