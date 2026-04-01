package tool_test

import (
	"context"
	"sync"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTool is a minimal Tool implementation for testing.
type stubTool struct {
	name        string
	description string
	schema      map[string]interface{}
}

func (s *stubTool) GetName() string                             { return s.name }
func (s *stubTool) GetDescription() string                      { return s.description }
func (s *stubTool) GetParametersSchema() map[string]interface{} { return s.schema }
func (s *stubTool) Execute(_ context.Context, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func newStub(name string) *stubTool {
	return &stubTool{name: name, description: name + " description"}
}

func TestNewRegistry(t *testing.T) {
	r := tool.NewRegistry()
	require.NotNil(t, r)
	assert.Empty(t, r.All())
	assert.Empty(t, r.Names())
}

func TestRegister_Success(t *testing.T) {
	r := tool.NewRegistry()
	err := r.Register(newStub("alpha"))
	require.NoError(t, err)

	got, ok := r.Get("alpha")
	require.True(t, ok)
	assert.Equal(t, "alpha", got.GetName())
}

func TestRegister_DuplicateError(t *testing.T) {
	r := tool.NewRegistry()
	require.NoError(t, r.Register(newStub("alpha")))

	err := r.Register(newStub("alpha"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegisterGroup_Success(t *testing.T) {
	r := tool.NewRegistry()
	err := r.RegisterGroup("grp", newStub("a"), newStub("b"), newStub("c"))
	require.NoError(t, err)

	group := r.GetGroup("grp")
	assert.Len(t, group, 3)

	// All tools should also be individually accessible
	for _, name := range []string{"a", "b", "c"} {
		_, ok := r.Get(name)
		assert.True(t, ok, "tool %q should be individually registered", name)
	}
}

func TestRegisterGroup_PartialDuplicate(t *testing.T) {
	r := tool.NewRegistry()
	require.NoError(t, r.Register(newStub("a")))

	err := r.RegisterGroup("grp", newStub("a"), newStub("b"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// "b" should NOT have been registered because the group registration failed
	_, ok := r.Get("b")
	assert.False(t, ok, "tool b should not be registered after group failure")
}

func TestRegisterGroup_IntraBatchDuplicate(t *testing.T) {
	r := tool.NewRegistry()

	err := r.RegisterGroup("grp", newStub("dup"), newStub("dup"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "appears more than once")

	// Neither tool should be registered after the failed group registration
	_, ok := r.Get("dup")
	assert.False(t, ok, "tool should not be registered after group failure")
}

func TestGet_Found(t *testing.T) {
	r := tool.NewRegistry()
	require.NoError(t, r.Register(newStub("x")))

	got, ok := r.Get("x")
	assert.True(t, ok)
	assert.Equal(t, "x", got.GetName())
}

func TestGet_NotFound(t *testing.T) {
	r := tool.NewRegistry()

	got, ok := r.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestGetGroup_Found(t *testing.T) {
	r := tool.NewRegistry()
	require.NoError(t, r.RegisterGroup("mygrp", newStub("t1"), newStub("t2")))

	group := r.GetGroup("mygrp")
	assert.Len(t, group, 2)
}

func TestGetGroup_Empty(t *testing.T) {
	r := tool.NewRegistry()

	group := r.GetGroup("no_such_group")
	assert.Nil(t, group)
}

func TestAll(t *testing.T) {
	r := tool.NewRegistry()
	require.NoError(t, r.Register(newStub("x")))
	require.NoError(t, r.Register(newStub("y")))
	require.NoError(t, r.Register(newStub("z")))

	all := r.All()
	assert.Len(t, all, 3)
}

func TestNames_Sorted(t *testing.T) {
	r := tool.NewRegistry()
	require.NoError(t, r.Register(newStub("charlie")))
	require.NoError(t, r.Register(newStub("alpha")))
	require.NoError(t, r.Register(newStub("bravo")))

	names := r.Names()
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, names)
}

func TestRemove(t *testing.T) {
	r := tool.NewRegistry()
	require.NoError(t, r.Register(newStub("del")))

	r.Remove("del")
	_, ok := r.Get("del")
	assert.False(t, ok)

	// Removing a non-existent tool should not panic
	r.Remove("does_not_exist")
}

func TestConcurrentAccess(t *testing.T) {
	r := tool.NewRegistry()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent registers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			name := "tool_" + string(rune('A'+id%26)) + string(rune('0'+id/26))
			_ = r.Register(newStub(name))
		}(i)
	}

	// Concurrent gets
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			name := "tool_" + string(rune('A'+id%26)) + string(rune('0'+id/26))
			r.Get(name)
		}(i)
	}

	// Concurrent removes
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			name := "tool_" + string(rune('A'+id%26)) + string(rune('0'+id/26))
			r.Remove(name)
		}(i)
	}

	wg.Wait()
	// If we get here without a race detector complaint, the test passes.
}
