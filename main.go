// Package agentsdk provides a Go SDK for building multi-agent AI applications.
//
// It supports tool use, skills, MCP integrations, plugins, agent handoffs,
// streaming responses, multi-turn conversations, and integrations with multiple
// LLM providers (OpenAI, Anthropic, Gemini, Mistral, LM Studio, Azure OpenAI).
//
// # Quick Start
//
//	import (
//	    agentsdk "github.com/agentizen/agent-sdk-go"
//	    "github.com/agentizen/agent-sdk-go/pkg/model/providers/openai"
//	)
//
//	provider := openai.NewProvider(os.Getenv("OPENAI_API_KEY"))
//
//	a := agentsdk.NewAgent("Assistant")
//	a.SetModelProvider(provider)
//	a.WithModel("gpt-4o-mini")
//	a.SetSystemInstructions("You are a helpful assistant.")
//
//	r := agentsdk.NewRunner()
//	r.WithDefaultProvider(provider)
//
//	result, err := r.Run(context.Background(), a, &agentsdk.RunOptions{
//	    Input: "Hello!",
//	})
//
// See the examples/ directory for complete working examples with each provider.
package agentsdk

import (
	"io"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/plugin"
	"github.com/agentizen/agent-sdk-go/pkg/result"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

// Version is the SDK version. It is overridden at release time via:
//
//	-ldflags "-X github.com/agentizen/agent-sdk-go.Version=<tag>"
var Version = "dev"

// Core type aliases — re-exported for convenience so callers only need to
// import this package for the most common use cases.
type (
	// Agent is an AI agent with tools, skills, MCP servers, plugins,
	// handoffs, and lifecycle hooks.
	Agent = agent.Agent

	// Runner executes agents, managing the turn loop, tool execution, and
	// agent-to-agent handoffs.
	Runner = runner.Runner

	// RunOptions configures a single invocation of Runner.Run or
	// Runner.RunStreaming.
	RunOptions = runner.RunOptions

	// RunConfig holds global, run-level configuration overrides such as model,
	// provider, guardrails, and tracing settings.
	RunConfig = runner.RunConfig

	// WorkflowConfig controls retry, state management, validation, and recovery
	// behavior for multi-agent workflows.
	WorkflowConfig = runner.WorkflowConfig

	// RetryConfig configures automatic retries for tool calls and handoffs.
	RetryConfig = runner.RetryConfig

	// TracingConfig carries metadata forwarded to the active tracer.
	TracingConfig = runner.TracingConfig

	// Tool is the interface every agent tool must satisfy.
	Tool = tool.Tool

	// FunctionTool wraps a Go function as an agent tool.
	FunctionTool = tool.FunctionTool

	// StreamEvent is a single event emitted during a streaming run.
	StreamEvent = model.StreamEvent

	// ContentPart is one segment of a multi-modal message (text, image, or
	// document).
	ContentPart = model.ContentPart

	// ModelSettings configures model-level parameters (temperature, top-p, …).
	ModelSettings = model.Settings

	// ModelProvider resolves model names to Model instances.
	ModelProvider = model.Provider

	// RunResult is the final result of a completed non-streaming run.
	RunResult = result.RunResult

	// StreamedRunResult is the handle returned by Runner.RunStreaming. Consume
	// events from its Stream channel, then read FinalOutput when done.
	StreamedRunResult = result.StreamedRunResult
)

// Streaming event type constants — forwarded from the model package.
const (
	// StreamEventTypeContent is emitted for each streaming text chunk.
	StreamEventTypeContent = model.StreamEventTypeContent

	// StreamEventTypeToolCall is emitted when the model invokes a tool.
	StreamEventTypeToolCall = model.StreamEventTypeToolCall

	// StreamEventTypeHandoff is emitted on agent-to-agent handoffs.
	StreamEventTypeHandoff = model.StreamEventTypeHandoff

	// StreamEventTypeDone is emitted when the model finishes generating.
	StreamEventTypeDone = model.StreamEventTypeDone

	// StreamEventTypeError is emitted when an unrecoverable error occurs
	// during streaming.
	StreamEventTypeError = model.StreamEventTypeError
)

// NewAgent creates a new Agent. An optional name and instructions can be
// provided as positional arguments: NewAgent("name", "instructions").
func NewAgent(name ...string) *Agent {
	return agent.NewAgent(name...)
}

// NewRunner creates a new Runner with default configuration (max 10 turns).
func NewRunner() *Runner {
	return runner.NewRunner()
}

// NewFunctionTool creates a tool backed by an arbitrary Go function.
//
// The function signature may be:
//   - func(ctx context.Context, params map[string]interface{}) (interface{}, error)
//   - func(params map[string]interface{}) (interface{}, error)
//   - func(ctx context.Context, input MyStruct) (MyResult, error)
//
// A JSON Schema is inferred automatically from the function signature.
// Use (*FunctionTool).WithSchema to override it.
func NewFunctionTool(name, description string, fn interface{}) *FunctionTool {
	return tool.NewFunctionTool(name, description, fn)
}

// --- Tool factories ---

// NewToolRegistry creates a new thread-safe tool registry with group support.
func NewToolRegistry() *ToolRegistry {
	return tool.NewRegistry()
}

// NewExecutableTool creates a tool that runs an external process (shell, python, node).
// Parameters are JSON-serialized to stdin; stdout is parsed as the result.
func NewExecutableTool(name, description, command string, args []string) *ExecutableTool {
	return tool.NewExecutableTool(name, description, command, args)
}

// WithToolMiddleware wraps a tool with one or more middleware layers.
func WithToolMiddleware(t Tool, mw ...ToolMiddleware) Tool {
	return tool.WithMiddleware(t, mw...)
}

// --- Skill factories ---

// LoadSkillFromFile loads a skill from a markdown file with YAML frontmatter.
func LoadSkillFromFile(path string) (Skill, error) {
	return skill.LoadFromFile(path)
}

// LoadSkillFromReader loads a skill from an io.Reader.
func LoadSkillFromReader(r io.Reader) (Skill, error) {
	return skill.LoadFromReader(r)
}

// LoadSkillFromString loads a skill from a raw markdown string.
func LoadSkillFromString(content string) (Skill, error) {
	return skill.LoadFromString(content)
}

// NewSkillRegistry creates a new skill registry.
func NewSkillRegistry() *SkillRegistry {
	return skill.NewRegistry()
}

// --- MCP factories ---

// NewMCPHTTPClient creates an HTTP client for communicating with MCP servers.
func NewMCPHTTPClient(opts MCPClientOptions) *MCPHTTPClient {
	return mcp.NewHTTPClient(opts)
}

// NewMCPRegistry creates a new MCP server registry.
func NewMCPRegistry() *MCPRegistry {
	return mcp.NewRegistry()
}

// --- Plugin factories ---

// NewPluginRegistry creates a new plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return plugin.NewRegistry()
}
