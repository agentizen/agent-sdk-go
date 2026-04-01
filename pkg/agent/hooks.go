package agent

import (
	"context"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/plugin"
	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

// Hooks defines lifecycle hooks for an agent
type Hooks interface {
	// OnAgentStart is called when the agent starts processing
	OnAgentStart(ctx context.Context, agent *Agent, input interface{}) error

	// OnBeforeModelCall is called before the model is called
	OnBeforeModelCall(ctx context.Context, agent *Agent, request *model.Request) error

	// OnAfterModelCall is called after the model is called
	OnAfterModelCall(ctx context.Context, agent *Agent, response *model.Response) error

	// OnBeforeToolCall is called before a tool is called
	OnBeforeToolCall(ctx context.Context, agent *Agent, tool tool.Tool, params map[string]interface{}) error

	// OnAfterToolCall is called after a tool is called
	OnAfterToolCall(ctx context.Context, agent *Agent, tool tool.Tool, result interface{}, err error) error

	// OnBeforeHandoff is called before a handoff to another agent
	OnBeforeHandoff(ctx context.Context, agent *Agent, handoffAgent *Agent) error

	// OnAfterHandoff is called after a handoff to another agent
	OnAfterHandoff(ctx context.Context, agent *Agent, handoffAgent *Agent, result interface{}) error

	// OnAgentEnd is called when the agent finishes processing
	OnAgentEnd(ctx context.Context, agent *Agent, result interface{}) error

	// OnSkillLoad is called when a skill's full content is loaded into context
	OnSkillLoad(ctx context.Context, agent *Agent, s skill.Skill, content string) error

	// OnBeforeMCPCall is called before an MCP tool is invoked
	OnBeforeMCPCall(ctx context.Context, agent *Agent, server mcp.ServerConfig, toolName string, params map[string]interface{}) error

	// OnAfterMCPCall is called after an MCP tool returns
	OnAfterMCPCall(ctx context.Context, agent *Agent, server mcp.ServerConfig, toolName string, result interface{}, err error) error

	// OnPluginInit is called when a plugin is initialized on the agent
	OnPluginInit(ctx context.Context, agent *Agent, p plugin.Plugin) error
}

// DefaultAgentHooks provides a default no-op implementation of Hooks
type DefaultAgentHooks struct{}

// OnAgentStart is called when the agent starts processing
func (h *DefaultAgentHooks) OnAgentStart(_ context.Context, _ *Agent, _ interface{}) error {
	return nil
}

// OnBeforeModelCall is called before the model is called
func (h *DefaultAgentHooks) OnBeforeModelCall(_ context.Context, _ *Agent, _ *model.Request) error {
	return nil
}

// OnAfterModelCall is called after the model is called
func (h *DefaultAgentHooks) OnAfterModelCall(_ context.Context, _ *Agent, _ *model.Response) error {
	return nil
}

// OnBeforeToolCall is called before a tool is called
func (h *DefaultAgentHooks) OnBeforeToolCall(_ context.Context, _ *Agent, _ tool.Tool, _ map[string]interface{}) error {
	return nil
}

// OnAfterToolCall is called after a tool is called
func (h *DefaultAgentHooks) OnAfterToolCall(_ context.Context, _ *Agent, _ tool.Tool, _ interface{}, _ error) error {
	return nil
}

// OnBeforeHandoff is called before a handoff to another agent
func (h *DefaultAgentHooks) OnBeforeHandoff(_ context.Context, _ *Agent, _ *Agent) error {
	return nil
}

// OnAfterHandoff is called after a handoff to another agent
func (h *DefaultAgentHooks) OnAfterHandoff(_ context.Context, _ *Agent, _ *Agent, _ interface{}) error {
	return nil
}

// OnAgentEnd is called when the agent finishes processing
func (h *DefaultAgentHooks) OnAgentEnd(_ context.Context, _ *Agent, _ interface{}) error {
	return nil
}

// OnSkillLoad is called when a skill's full content is loaded into context
func (h *DefaultAgentHooks) OnSkillLoad(_ context.Context, _ *Agent, _ skill.Skill, _ string) error {
	return nil
}

// OnBeforeMCPCall is called before an MCP tool is invoked
func (h *DefaultAgentHooks) OnBeforeMCPCall(_ context.Context, _ *Agent, _ mcp.ServerConfig, _ string, _ map[string]interface{}) error {
	return nil
}

// OnAfterMCPCall is called after an MCP tool returns
func (h *DefaultAgentHooks) OnAfterMCPCall(_ context.Context, _ *Agent, _ mcp.ServerConfig, _ string, _ interface{}, _ error) error {
	return nil
}

// OnPluginInit is called when a plugin is initialized on the agent
func (h *DefaultAgentHooks) OnPluginInit(_ context.Context, _ *Agent, _ plugin.Plugin) error {
	return nil
}
