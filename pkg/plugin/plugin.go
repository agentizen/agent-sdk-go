// Package plugin provides a bundle abstraction that groups tools, skills,
// and MCP servers into a single pluggable unit for agents.
package plugin

import (
	"context"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

// Plugin is a bundle of tools, skills, and MCP servers that can be
// registered to an agent as a single unit.
type Plugin interface {
	// Name returns the plugin's unique identifier.
	Name() string
	// Description returns a human-readable description.
	Description() string
	// Version returns the plugin version.
	Version() string
	// Tools returns the tools provided by this plugin.
	Tools() []tool.Tool
	// Skills returns the skills provided by this plugin.
	Skills() []skill.Skill
	// MCPServers returns the MCP server configs provided by this plugin.
	MCPServers() []mcp.ServerConfig
	// Init is called once when the plugin is registered. It receives a context
	// and can perform setup such as validating API keys.
	Init(ctx context.Context) error
}

// BasePlugin provides a default implementation of Plugin that consumers
// can embed and override selectively.
type BasePlugin struct {
	PluginName        string
	PluginDescription string
	PluginVersion     string
	PluginTools       []tool.Tool
	PluginSkills      []skill.Skill
	PluginMCPServers  []mcp.ServerConfig
}

// Name returns the plugin's unique identifier.
func (p *BasePlugin) Name() string { return p.PluginName }

// Description returns a human-readable description.
func (p *BasePlugin) Description() string { return p.PluginDescription }

// Version returns the plugin version.
func (p *BasePlugin) Version() string { return p.PluginVersion }

// Tools returns the tools provided by this plugin.
func (p *BasePlugin) Tools() []tool.Tool { return p.PluginTools }

// Skills returns the skills provided by this plugin.
func (p *BasePlugin) Skills() []skill.Skill { return p.PluginSkills }

// MCPServers returns the MCP server configs provided by this plugin.
func (p *BasePlugin) MCPServers() []mcp.ServerConfig { return p.PluginMCPServers }

// Init is a no-op default. Override in embedding structs to add setup logic.
func (p *BasePlugin) Init(_ context.Context) error { return nil }
