package tool_test

import (
	"context"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTool is a controllable Tool for middleware testing.
type fakeTool struct {
	name        string
	description string
	schema      map[string]interface{}
	executeFn   func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

func (f *fakeTool) GetName() string                             { return f.name }
func (f *fakeTool) GetDescription() string                      { return f.description }
func (f *fakeTool) GetParametersSchema() map[string]interface{} { return f.schema }
func (f *fakeTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return f.executeFn(ctx, params)
}

func newFake() *fakeTool {
	return &fakeTool{
		name:        "inner",
		description: "inner description",
		schema:      map[string]interface{}{"type": "object"},
		executeFn: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return "inner-result", nil
		},
	}
}

func TestWrapExecute_DelegatesMetadata(t *testing.T) {
	inner := &fakeTool{
		name:        "wrapped-tool",
		description: "wrapped description",
		schema:      map[string]interface{}{"key": "val"},
		executeFn: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return "inner", nil
		},
	}

	wrapped := tool.WrapExecute(inner, func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
		return "custom", nil
	})

	assert.Equal(t, "wrapped-tool", wrapped.GetName())
	assert.Equal(t, "wrapped description", wrapped.GetDescription())
	assert.Equal(t, map[string]interface{}{"key": "val"}, wrapped.GetParametersSchema())
}

func TestWrapExecute_UsesCustomExecute(t *testing.T) {
	innerCalled := false
	inner := &fakeTool{
		name: "inner",
		executeFn: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			innerCalled = true
			return "inner-result", nil
		},
	}

	customCalled := false
	wrapped := tool.WrapExecute(inner, func(_ context.Context, params map[string]interface{}) (interface{}, error) {
		customCalled = true
		return "custom-result", nil
	})

	result, err := wrapped.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, customCalled, "custom execute should have been called")
	assert.False(t, innerCalled, "inner execute should NOT have been called")
	assert.Equal(t, "custom-result", result)
}

func TestWithMiddleware_SingleMiddleware(t *testing.T) {
	inner := newFake()
	called := false

	mw := func(next tool.Tool) tool.Tool {
		return &fakeTool{
			name:        next.GetName(),
			description: next.GetDescription(),
			schema:      next.GetParametersSchema(),
			executeFn: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				called = true
				return next.Execute(ctx, params)
			},
		}
	}

	wrapped := tool.WithMiddleware(inner, mw)
	result, err := wrapped.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, called, "middleware should have been called")
	assert.Equal(t, "inner-result", result)
}

func TestWithMiddleware_MultipleMiddleware_Order(t *testing.T) {
	inner := newFake()
	inner.executeFn = func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
		return "base", nil
	}

	var order []string

	makeMW := func(label string) tool.Middleware {
		return func(next tool.Tool) tool.Tool {
			return &fakeTool{
				name:        next.GetName(),
				description: next.GetDescription(),
				schema:      next.GetParametersSchema(),
				executeFn: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					order = append(order, label+"-before")
					res, err := next.Execute(ctx, params)
					order = append(order, label+"-after")
					return res, err
				},
			}
		}
	}

	wrapped := tool.WithMiddleware(inner, makeMW("first"), makeMW("second"))
	result, err := wrapped.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "base", result)

	// First middleware is outermost, so it runs before/after the second
	assert.Equal(t, []string{"first-before", "second-before", "second-after", "first-after"}, order)
}

func TestWithMiddleware_ModifiesParams(t *testing.T) {
	inner := newFake()
	inner.executeFn = func(_ context.Context, params map[string]interface{}) (interface{}, error) {
		return params["injected"], nil
	}

	mw := func(next tool.Tool) tool.Tool {
		return &fakeTool{
			name:        next.GetName(),
			description: next.GetDescription(),
			schema:      next.GetParametersSchema(),
			executeFn: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				if params == nil {
					params = make(map[string]interface{})
				}
				params["injected"] = "value-from-middleware"
				return next.Execute(ctx, params)
			},
		}
	}

	wrapped := tool.WithMiddleware(inner, mw)
	result, err := wrapped.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "value-from-middleware", result)
}

func TestWithMiddleware_ModifiesResult(t *testing.T) {
	inner := newFake()
	inner.executeFn = func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
		return "original", nil
	}

	mw := func(next tool.Tool) tool.Tool {
		return &fakeTool{
			name:        next.GetName(),
			description: next.GetDescription(),
			schema:      next.GetParametersSchema(),
			executeFn: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				res, err := next.Execute(ctx, params)
				if err != nil {
					return nil, err
				}
				return res.(string) + "-modified", nil
			},
		}
	}

	wrapped := tool.WithMiddleware(inner, mw)
	result, err := wrapped.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "original-modified", result)
}

func TestWithMiddleware_DelegatesToInnerTool(t *testing.T) {
	inner := &fakeTool{
		name:        "my-tool",
		description: "my-desc",
		schema:      map[string]interface{}{"key": "val"},
		executeFn: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return nil, nil
		},
	}

	mw := func(next tool.Tool) tool.Tool {
		return &fakeTool{
			name:        next.GetName(),
			description: next.GetDescription(),
			schema:      next.GetParametersSchema(),
			executeFn: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return next.Execute(ctx, params)
			},
		}
	}

	wrapped := tool.WithMiddleware(inner, mw)
	assert.Equal(t, "my-tool", wrapped.GetName())
	assert.Equal(t, "my-desc", wrapped.GetDescription())
	assert.Equal(t, map[string]interface{}{"key": "val"}, wrapped.GetParametersSchema())
}
