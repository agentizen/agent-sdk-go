package plugin_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPlugin is a minimal Plugin implementation for testing.
type testPlugin struct {
	plugin.BasePlugin
	initFn func(ctx context.Context) error
}

func (p *testPlugin) Init(ctx context.Context) error {
	if p.initFn != nil {
		return p.initFn(ctx)
	}
	return nil
}

func newTestPlugin(name string) *testPlugin {
	return &testPlugin{
		BasePlugin: plugin.BasePlugin{PluginName: name, PluginVersion: "0.1.0"},
	}
}

func TestNewRegistry_IsEmpty(t *testing.T) {
	r := plugin.NewRegistry()
	assert.Empty(t, r.All())
}

func TestRegistry_Register_CallsInit(t *testing.T) {
	initCalled := false
	p := newTestPlugin("test")
	p.initFn = func(_ context.Context) error {
		initCalled = true
		return nil
	}

	r := plugin.NewRegistry()
	err := r.Register(context.Background(), p)
	require.NoError(t, err)
	assert.True(t, initCalled)
}

func TestRegistry_Register_DuplicateError(t *testing.T) {
	r := plugin.NewRegistry()
	err := r.Register(context.Background(), newTestPlugin("dup"))
	require.NoError(t, err)

	err = r.Register(context.Background(), newTestPlugin("dup"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_Register_FailingInit(t *testing.T) {
	p := newTestPlugin("failing")
	p.initFn = func(_ context.Context) error {
		return fmt.Errorf("missing API key")
	}

	r := plugin.NewRegistry()
	err := r.Register(context.Background(), p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "init failed")
	assert.Contains(t, err.Error(), "missing API key")

	// The plugin should not be stored after a failed init.
	_, ok := r.Get("failing")
	assert.False(t, ok)
}

func TestRegistry_Get_Success(t *testing.T) {
	r := plugin.NewRegistry()
	_ = r.Register(context.Background(), newTestPlugin("myplug"))

	p, ok := r.Get("myplug")
	require.True(t, ok)
	assert.Equal(t, "myplug", p.Name())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := plugin.NewRegistry()
	_, ok := r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_All_SortedByName(t *testing.T) {
	r := plugin.NewRegistry()
	_ = r.Register(context.Background(), newTestPlugin("charlie"))
	_ = r.Register(context.Background(), newTestPlugin("alpha"))
	_ = r.Register(context.Background(), newTestPlugin("bravo"))

	all := r.All()
	require.Len(t, all, 3)
	assert.Equal(t, "alpha", all[0].Name())
	assert.Equal(t, "bravo", all[1].Name())
	assert.Equal(t, "charlie", all[2].Name())
}

func TestRegistry_Unregister(t *testing.T) {
	r := plugin.NewRegistry()
	_ = r.Register(context.Background(), newTestPlugin("removable"))

	r.Unregister("removable")

	_, ok := r.Get("removable")
	assert.False(t, ok)
	assert.Empty(t, r.All())
}

func TestRegistry_Unregister_NonExistent_IsNoop(t *testing.T) {
	r := plugin.NewRegistry()
	r.Unregister("ghost") // should not panic
}
