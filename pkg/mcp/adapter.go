package mcp

import (
	"context"
	"fmt"

	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

// mcpTool adapts an MCP server endpoint into a standard tool.Tool.
type mcpTool struct {
	server ServerConfig
	info   ToolInfo
}

// ToolAdapter converts an MCP server endpoint into a standard tool.Tool.
func ToolAdapter(server ServerConfig, info ToolInfo) tool.Tool {
	return &mcpTool{server: server, info: info}
}

func (t *mcpTool) GetName() string {
	return t.info.Name
}

func (t *mcpTool) GetDescription() string {
	return t.info.Description
}

func (t *mcpTool) GetParametersSchema() map[string]interface{} {
	return t.info.Parameters
}

func (t *mcpTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if t.server.Client == nil {
		return nil, fmt.Errorf("mcp: server %q has no client configured", t.server.Handle)
	}
	return t.server.Client.Execute(ctx, t.server, t.info.Name, params)
}

// ToolsFromServer discovers all tools from an MCP server and returns them as tool.Tool.
func ToolsFromServer(ctx context.Context, server ServerConfig) ([]tool.Tool, error) {
	if server.Client == nil {
		return nil, fmt.Errorf("mcp: server %q has no client configured", server.Handle)
	}
	infos, err := server.Client.ListTools(ctx, server)
	if err != nil {
		return nil, err
	}
	tools := make([]tool.Tool, len(infos))
	for i, info := range infos {
		tools[i] = ToolAdapter(server, info)
	}
	return tools, nil
}
