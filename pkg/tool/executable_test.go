package tool_test

import (
	"context"
	"testing"
	"time"

	"github.com/agentizen/agent-sdk-go/pkg/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutableTool(t *testing.T) {
	et := tool.NewExecutableTool("mytool", "does stuff", "echo", []string{"hello"})
	require.NotNil(t, et)
	assert.Equal(t, "mytool", et.GetName())
	assert.Equal(t, "does stuff", et.GetDescription())
	assert.NotNil(t, et.GetParametersSchema())
}

func TestWithSchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{"type": "string"},
		},
	}
	et := tool.NewExecutableTool("s", "d", "echo", nil).WithSchema(schema)
	assert.Equal(t, schema, et.GetParametersSchema())
}

func TestWithTimeout(t *testing.T) {
	et := tool.NewExecutableTool("s", "d", "echo", nil).WithTimeout(5 * time.Second)
	require.NotNil(t, et)
}

func TestWithWorkDir(t *testing.T) {
	et := tool.NewExecutableTool("s", "d", "echo", nil).WithWorkDir("/tmp")
	require.NotNil(t, et)
}

func TestWithEnv(t *testing.T) {
	et := tool.NewExecutableTool("s", "d", "echo", nil).WithEnv([]string{"FOO=bar"})
	require.NotNil(t, et)
}

func TestExecute_StdinToStdout(t *testing.T) {
	// "cat" reads stdin and writes it to stdout
	et := tool.NewExecutableTool("cat-tool", "echoes stdin", "cat", nil).
		WithTimeout(5 * time.Second)

	params := map[string]interface{}{"greeting": "hello"}
	result, err := et.Execute(context.Background(), params)
	require.NoError(t, err)

	// cat outputs the JSON written to stdin; the tool parses it back as JSON
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok, "expected parsed JSON map, got %T", result)
	assert.Equal(t, "hello", resultMap["greeting"])
}

func TestExecute_Timeout(t *testing.T) {
	et := tool.NewExecutableTool("slow", "sleeps forever", "sleep", []string{"60"}).
		WithTimeout(100 * time.Millisecond)

	_, err := et.Execute(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestExecute_NonZeroExitCode(t *testing.T) {
	et := tool.NewExecutableTool("fail", "exits 1", "bash", []string{"-c", "echo oops >&2; exit 1"}).
		WithTimeout(5 * time.Second)

	_, err := et.Execute(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oops")
}

func TestGetName(t *testing.T) {
	et := tool.NewExecutableTool("nm", "desc", "echo", nil)
	assert.Equal(t, "nm", et.GetName())
}

func TestGetDescription(t *testing.T) {
	et := tool.NewExecutableTool("nm", "my description", "echo", nil)
	assert.Equal(t, "my description", et.GetDescription())
}

func TestGetParametersSchema(t *testing.T) {
	et := tool.NewExecutableTool("nm", "d", "echo", nil)
	// Default schema is an empty map
	assert.NotNil(t, et.GetParametersSchema())
	assert.Empty(t, et.GetParametersSchema())
}
