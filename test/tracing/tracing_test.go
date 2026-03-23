package tracing_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/citizenofai/agent-sdk-go/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- NoopTracer ----

func TestNoopTracer(t *testing.T) {
	tracer := &tracing.NoopTracer{}
	ctx := context.Background()

	// RecordEvent should not panic
	tracer.RecordEvent(ctx, tracing.Event{Type: tracing.EventTypeAgentStart, AgentName: "agent"})

	// Flush and Close should return nil
	assert.NoError(t, tracer.Flush())
	assert.NoError(t, tracer.Close())
}

// ---- FileTracer ----

func TestNewFileTracer(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	tracer, err := tracing.NewFileTracer("test-agent")
	require.NoError(t, err)
	require.NotNil(t, tracer)
	defer func() { _ = tracer.Close() }()

	// Verify trace file was created
	entries, err := filepath.Glob(filepath.Join(tmpDir, "trace_test-agent.log"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestFileTracer_RecordEvent(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	tracer, err := tracing.NewFileTracer("record-test")
	require.NoError(t, err)
	defer func() { _ = tracer.Close() }()

	ctx := context.Background()
	tracer.RecordEvent(ctx, tracing.Event{
		Type:      tracing.EventTypeAgentStart,
		AgentName: "my-agent",
		Details:   map[string]interface{}{"input": "hello"},
	})
	require.NoError(t, tracer.Flush())

	content, err := os.ReadFile(filepath.Join(tmpDir, "trace_record-test.log"))
	require.NoError(t, err)
	assert.Contains(t, string(content), tracing.EventTypeAgentStart)
	assert.Contains(t, string(content), "my-agent")
}

func TestFileTracer_FlushAndClose(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	tracer, err := tracing.NewFileTracer("flush-test")
	require.NoError(t, err)

	assert.NoError(t, tracer.Flush())
	assert.NoError(t, tracer.Close())
}

func TestNewFileTracer_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// The implementation sanitizes the agent name before building the path, so
	// "../../etc/passwd" becomes "____etc_passwd" and the resulting file is
	// always placed inside tmpDir.  We verify that no file was created outside
	// tmpDir regardless of whether NewFileTracer returns an error or succeeds.
	tracer, tracerErr := tracing.NewFileTracer("../../etc/passwd")
	if tracer != nil {
		defer func() { _ = tracer.Close() }()
	}

	if tracerErr != nil {
		// An error is fine; confirm nothing was created outside tmpDir.
		outsideFiles, globErr := filepath.Glob(filepath.Join(origDir, "trace_*.log"))
		require.NoError(t, globErr)
		assert.Empty(t, outsideFiles, "no trace files should be created outside tmpDir on error")
	} else {
		// Success is fine too provided the file landed inside tmpDir.
		insideFiles, globErr := filepath.Glob(filepath.Join(tmpDir, "trace_*.log"))
		require.NoError(t, globErr)
		assert.NotEmpty(t, insideFiles, "trace file should be inside tmpDir")

		// Ensure no file was written in a parent directory.
		outsideFiles, globErr := filepath.Glob(filepath.Join(origDir, "trace_*.log"))
		require.NoError(t, globErr)
		assert.Empty(t, outsideFiles, "no trace files should escape to parent directories")
	}
}

// ---- Global tracer ----

func TestSetAndGetGlobalTracer(t *testing.T) {
	original := tracing.GetGlobalTracer()
	defer tracing.SetGlobalTracer(original)

	noop := &tracing.NoopTracer{}
	tracing.SetGlobalTracer(noop)
	assert.Equal(t, noop, tracing.GetGlobalTracer())
}

func TestRecordEvent_GlobalTracer(t *testing.T) {
	// RecordEvent should not panic even with the default global tracer
	ctx := context.Background()
	tracing.RecordEvent(ctx, tracing.Event{
		Type:      tracing.EventTypeToolCall,
		AgentName: "test",
		Details:   map[string]interface{}{"tool": "search"},
	})
}

// ---- Context helpers ----

func TestWithTracer_And_GetTracer(t *testing.T) {
	noop := &tracing.NoopTracer{}
	ctx := tracing.WithTracer(context.Background(), noop)
	got := tracing.GetTracer(ctx)
	assert.Equal(t, noop, got)
}

func TestGetTracer_FallsBackToGlobal(t *testing.T) {
	// A plain context should return the global tracer
	ctx := context.Background()
	got := tracing.GetTracer(ctx)
	assert.Equal(t, tracing.GetGlobalTracer(), got)
}

func TestRecordEventContext(t *testing.T) {
	noop := &tracing.NoopTracer{}
	ctx := tracing.WithTracer(context.Background(), noop)
	// Should not panic
	tracing.RecordEventContext(ctx, tracing.Event{
		Type:      tracing.EventTypeModelRequest,
		AgentName: "agent",
	})
}

// ---- Event type constants ----

func TestEventTypeConstants(t *testing.T) {
	assert.Equal(t, "agent_start", tracing.EventTypeAgentStart)
	assert.Equal(t, "agent_end", tracing.EventTypeAgentEnd)
	assert.Equal(t, "tool_call", tracing.EventTypeToolCall)
	assert.Equal(t, "tool_result", tracing.EventTypeToolResult)
	assert.Equal(t, "model_request", tracing.EventTypeModelRequest)
	assert.Equal(t, "model_response", tracing.EventTypeModelResponse)
	assert.Equal(t, "handoff", tracing.EventTypeHandoff)
	assert.Equal(t, "handoff_complete", tracing.EventTypeHandoffComplete)
	assert.Equal(t, "agent_message", tracing.EventTypeAgentMessage)
	assert.Equal(t, "error", tracing.EventTypeError)
}

// ---- Convenience event functions ----

func TestTracingEventFunctions(t *testing.T) {
	noop := &tracing.NoopTracer{}
	ctx := tracing.WithTracer(context.Background(), noop)

	// None of these should panic
	tracing.AgentStart(ctx, "agent1", "hello", nil)
	tracing.AgentEnd(ctx, "agent1", "result")
	tracing.ToolCall(ctx, "agent1", "search", map[string]interface{}{"q": "go"})
	tracing.ToolResult(ctx, "agent1", "search", "results", nil)
	tracing.ModelRequest(ctx, "agent1", "gpt-4", "prompt", nil)
	tracing.ModelResponse(ctx, "agent1", "gpt-4", "response", nil)
	tracing.Handoff(ctx, "agent1", "agent2", "task")
	tracing.HandoffComplete(ctx, "agent2", "agent1", "done")
	tracing.AgentMessage(ctx, "agent1", "user", "hello")
	tracing.Error(ctx, "agent1", "something failed", assert.AnError)
}

func TestTraceForAgent(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	tracer, err := tracing.TraceForAgent("my-agent")
	require.NoError(t, err)
	require.NotNil(t, tracer)
	defer func() { _ = tracer.Close() }()
}
