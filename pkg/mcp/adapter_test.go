package mcp_test

import (
	"context"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements mcp.Client for testing.
type mockClient struct {
	executeFn   func(ctx context.Context, server mcp.ServerConfig, toolName string, params map[string]interface{}) (interface{}, error)
	listToolsFn func(ctx context.Context, server mcp.ServerConfig) ([]mcp.ToolInfo, error)
}

func (m *mockClient) Execute(ctx context.Context, server mcp.ServerConfig, toolName string, params map[string]interface{}) (interface{}, error) {
	return m.executeFn(ctx, server, toolName, params)
}

func (m *mockClient) ListTools(ctx context.Context, server mcp.ServerConfig) ([]mcp.ToolInfo, error) {
	return m.listToolsFn(ctx, server)
}

func TestToolAdapter_ReturnsCorrectMetadata(t *testing.T) {
	info := mcp.ToolInfo{
		Name:        "translate",
		Description: "Translates text between languages",
		Parameters:  map[string]interface{}{"type": "object"},
	}
	cfg := mcp.ServerConfig{Handle: "test"}

	adapted := mcp.ToolAdapter(cfg, info)

	assert.Equal(t, "translate", adapted.GetName())
	assert.Equal(t, "Translates text between languages", adapted.GetDescription())
	assert.Equal(t, map[string]interface{}{"type": "object"}, adapted.GetParametersSchema())
}

func TestToolAdapter_Execute_DelegatesToClient(t *testing.T) {
	called := false
	mc := &mockClient{
		executeFn: func(_ context.Context, _ mcp.ServerConfig, toolName string, params map[string]interface{}) (interface{}, error) {
			called = true
			assert.Equal(t, "translate", toolName)
			assert.Equal(t, "hello", params["text"])
			return map[string]interface{}{"result": "bonjour"}, nil
		},
	}

	cfg := mcp.ServerConfig{
		Handle: "test",
		Client: mc,
	}
	info := mcp.ToolInfo{Name: "translate"}
	adapted := mcp.ToolAdapter(cfg, info)

	result, err := adapted.Execute(context.Background(), map[string]interface{}{"text": "hello"})
	require.NoError(t, err)
	assert.True(t, called)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "bonjour", resultMap["result"])
}

func TestToolAdapter_Execute_NilClient_ReturnsError(t *testing.T) {
	cfg := mcp.ServerConfig{Handle: "no-client"}
	info := mcp.ToolInfo{Name: "translate"}
	adapted := mcp.ToolAdapter(cfg, info)

	_, err := adapted.Execute(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no client configured")
}

func TestToolsFromServer_NilClient_ReturnsError(t *testing.T) {
	cfg := mcp.ServerConfig{Handle: "no-client"}

	_, err := mcp.ToolsFromServer(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no client configured")
}

func TestToolsFromServer_DiscoversAndAdaptsAllTools(t *testing.T) {
	mc := &mockClient{
		listToolsFn: func(_ context.Context, _ mcp.ServerConfig) ([]mcp.ToolInfo, error) {
			return []mcp.ToolInfo{
				{Name: "tool-a", Description: "First tool"},
				{Name: "tool-b", Description: "Second tool"},
			}, nil
		},
	}

	cfg := mcp.ServerConfig{
		Handle: "test",
		Client: mc,
	}

	tools, err := mcp.ToolsFromServer(context.Background(), cfg)
	require.NoError(t, err)
	require.Len(t, tools, 2)
	assert.Equal(t, "tool-a", tools[0].GetName())
	assert.Equal(t, "First tool", tools[0].GetDescription())
	assert.Equal(t, "tool-b", tools[1].GetName())
	assert.Equal(t, "Second tool", tools[1].GetDescription())
}
