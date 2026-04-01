package plugin_test

import (
	"context"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/agentizen/agent-sdk-go/pkg/plugin"
	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
	"github.com/stretchr/testify/assert"
)

func TestBasePlugin_Name(t *testing.T) {
	p := &plugin.BasePlugin{PluginName: "analytics"}
	assert.Equal(t, "analytics", p.Name())
}

func TestBasePlugin_Description(t *testing.T) {
	p := &plugin.BasePlugin{PluginDescription: "Analytics tools"}
	assert.Equal(t, "Analytics tools", p.Description())
}

func TestBasePlugin_Version(t *testing.T) {
	p := &plugin.BasePlugin{PluginVersion: "1.2.3"}
	assert.Equal(t, "1.2.3", p.Version())
}

func TestBasePlugin_Tools(t *testing.T) {
	tools := []tool.Tool{}
	p := &plugin.BasePlugin{PluginTools: tools}
	assert.Equal(t, tools, p.Tools())
}

func TestBasePlugin_Skills(t *testing.T) {
	skills := []skill.Skill{}
	p := &plugin.BasePlugin{PluginSkills: skills}
	assert.Equal(t, skills, p.Skills())
}

func TestBasePlugin_MCPServers(t *testing.T) {
	servers := []mcp.ServerConfig{{Handle: "test"}}
	p := &plugin.BasePlugin{PluginMCPServers: servers}
	assert.Len(t, p.MCPServers(), 1)
	assert.Equal(t, "test", p.MCPServers()[0].Handle)
}

func TestBasePlugin_Init_IsNoop(t *testing.T) {
	p := &plugin.BasePlugin{}
	err := p.Init(context.Background())
	assert.NoError(t, err)
}

func TestBasePlugin_NilFields_ReturnDefaults(t *testing.T) {
	p := &plugin.BasePlugin{}
	assert.Equal(t, "", p.Name())
	assert.Equal(t, "", p.Description())
	assert.Equal(t, "", p.Version())
	assert.Nil(t, p.Tools())
	assert.Nil(t, p.Skills())
	assert.Nil(t, p.MCPServers())
}
