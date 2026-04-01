package mcp_test

import (
	"context"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry_IsEmpty(t *testing.T) {
	r := mcp.NewRegistry()
	assert.Empty(t, r.All())
}

func TestRegistry_Register_Success(t *testing.T) {
	r := mcp.NewRegistry()
	err := r.Register(mcp.ServerConfig{Handle: "github", URL: "https://github.example.com"})
	require.NoError(t, err)

	cfg, ok := r.Get("github")
	assert.True(t, ok)
	assert.Equal(t, "github", cfg.Handle)
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := mcp.NewRegistry()
	err := r.Register(mcp.ServerConfig{Handle: "github"})
	require.NoError(t, err)

	err = r.Register(mcp.ServerConfig{Handle: "github"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := mcp.NewRegistry()
	_, ok := r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_All_ReturnsAllServers(t *testing.T) {
	r := mcp.NewRegistry()
	_ = r.Register(mcp.ServerConfig{Handle: "a"})
	_ = r.Register(mcp.ServerConfig{Handle: "b"})

	all := r.All()
	assert.Len(t, all, 2)

	handles := map[string]bool{}
	for _, cfg := range all {
		handles[cfg.Handle] = true
	}
	assert.True(t, handles["a"])
	assert.True(t, handles["b"])
}

func TestRegistry_ToolsFor_Success(t *testing.T) {
	mc := &mockClient{
		listToolsFn: func(_ context.Context, _ mcp.ServerConfig) ([]mcp.ToolInfo, error) {
			return []mcp.ToolInfo{
				{Name: "search", Description: "Search tool"},
			}, nil
		},
	}

	r := mcp.NewRegistry()
	_ = r.Register(mcp.ServerConfig{Handle: "github", Client: mc})

	tools, err := r.ToolsFor(context.Background(), "github")
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "search", tools[0].GetName())
}

func TestRegistry_ToolsFor_NotFound(t *testing.T) {
	r := mcp.NewRegistry()
	_, err := r.ToolsFor(context.Background(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_AllTools_CombinesAllServers(t *testing.T) {
	mc1 := &mockClient{
		listToolsFn: func(_ context.Context, _ mcp.ServerConfig) ([]mcp.ToolInfo, error) {
			return []mcp.ToolInfo{{Name: "tool-1"}}, nil
		},
	}
	mc2 := &mockClient{
		listToolsFn: func(_ context.Context, _ mcp.ServerConfig) ([]mcp.ToolInfo, error) {
			return []mcp.ToolInfo{{Name: "tool-2"}, {Name: "tool-3"}}, nil
		},
	}

	r := mcp.NewRegistry()
	_ = r.Register(mcp.ServerConfig{Handle: "s1", Client: mc1})
	_ = r.Register(mcp.ServerConfig{Handle: "s2", Client: mc2})

	tools, err := r.AllTools(context.Background())
	require.NoError(t, err)
	assert.Len(t, tools, 3)
}

func TestRegistry_Remove(t *testing.T) {
	r := mcp.NewRegistry()
	_ = r.Register(mcp.ServerConfig{Handle: "github"})

	r.Remove("github")

	_, ok := r.Get("github")
	assert.False(t, ok)
	assert.Empty(t, r.All())
}

func TestRegistry_Remove_NonExistent_IsNoop(t *testing.T) {
	r := mcp.NewRegistry()
	r.Remove("nonexistent") // should not panic
}
