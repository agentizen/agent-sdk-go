package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// defaultTimeout is the default execution timeout for an ExecutableTool.
const defaultTimeout = 30 * time.Second

// ExecutableTool is a tool that executes an external process. Parameters are
// serialized as JSON and written to the process's stdin. The process's stdout
// is read as the result, attempting JSON parsing first and falling back to a
// raw string.
type ExecutableTool struct {
	name        string
	description string
	schema      map[string]interface{}
	command     string
	args        []string
	timeout     time.Duration
	workDir     string
	env         []string
}

// NewExecutableTool creates a new ExecutableTool with the given name, description,
// command, and arguments. The default timeout is 30 seconds.
func NewExecutableTool(name, description, command string, args []string) *ExecutableTool {
	return &ExecutableTool{
		name:        name,
		description: description,
		schema:      map[string]interface{}{},
		command:     command,
		args:        args,
		timeout:     defaultTimeout,
	}
}

// WithSchema sets a custom JSON schema for the tool parameters.
func (t *ExecutableTool) WithSchema(schema map[string]interface{}) *ExecutableTool {
	t.schema = schema
	return t
}

// WithTimeout sets the execution timeout for the external process.
func (t *ExecutableTool) WithTimeout(d time.Duration) *ExecutableTool {
	t.timeout = d
	return t
}

// WithWorkDir sets the working directory for the external process.
func (t *ExecutableTool) WithWorkDir(dir string) *ExecutableTool {
	t.workDir = dir
	return t
}

// WithEnv sets the environment variables for the external process. Each entry
// should be in the form "KEY=VALUE".
func (t *ExecutableTool) WithEnv(env []string) *ExecutableTool {
	t.env = env
	return t
}

// GetName returns the name of the tool.
func (t *ExecutableTool) GetName() string {
	return t.name
}

// GetDescription returns the description of the tool.
func (t *ExecutableTool) GetDescription() string {
	return t.description
}

// GetParametersSchema returns the JSON schema for the tool parameters.
func (t *ExecutableTool) GetParametersSchema() map[string]interface{} {
	return t.schema
}

// Execute runs the external process with the given parameters. The parameters
// are serialized as JSON and written to stdin. If the process exits with a
// non-zero status, an error containing stderr output is returned.
func (t *ExecutableTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.command, t.args...)

	if t.workDir != "" {
		cmd.Dir = t.workDir
	}
	if len(t.env) > 0 {
		cmd.Env = t.env
	}

	// Serialize params as JSON to stdin
	inputData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize parameters: %w", err)
	}
	cmd.Stdin = bytes.NewReader(inputData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("command %q timed out after %s: %s", t.command, t.timeout, stderr.String())
		}
		return nil, fmt.Errorf("command %q failed: %s: %s", t.command, err, stderr.String())
	}

	// Try to parse stdout as JSON, fall back to raw string
	output := stdout.Bytes()
	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return stdout.String(), nil
	}
	return result, nil
}
