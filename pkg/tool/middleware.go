package tool

import "context"

// Middleware is a function that wraps a Tool, returning a new Tool with
// additional or modified behavior. This enables the decorator pattern for
// composing cross-cutting concerns such as logging, validation, or retries.
type Middleware func(next Tool) Tool

// middlewareTool wraps an inner Tool and delegates all interface methods to it.
// The Execute method can be overridden by the middleware that created it.
type middlewareTool struct {
	inner   Tool
	execute func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// GetName returns the name of the inner tool.
func (m *middlewareTool) GetName() string {
	return m.inner.GetName()
}

// GetDescription returns the description of the inner tool.
func (m *middlewareTool) GetDescription() string {
	return m.inner.GetDescription()
}

// GetParametersSchema returns the parameters schema of the inner tool.
func (m *middlewareTool) GetParametersSchema() map[string]interface{} {
	return m.inner.GetParametersSchema()
}

// Execute invokes the middleware's execute function.
func (m *middlewareTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return m.execute(ctx, params)
}

// WrapExecute creates a middleware-aware Tool that delegates GetName,
// GetDescription, and GetParametersSchema to the inner tool, while using
// the provided execute function. This is the recommended way for middleware
// authors to wrap a tool's execution while preserving its identity.
func WrapExecute(inner Tool, execute func(ctx context.Context, params map[string]interface{}) (interface{}, error)) Tool {
	return &middlewareTool{
		inner:   inner,
		execute: execute,
	}
}

// WithMiddleware applies the given middlewares to a tool in order. The first
// middleware in the list is the outermost wrapper (executed first), and the
// last middleware is closest to the original tool.
func WithMiddleware(t Tool, mw ...Middleware) Tool {
	// Apply in reverse so that the first middleware is outermost
	for i := len(mw) - 1; i >= 0; i-- {
		t = mw[i](t)
	}
	return t
}
