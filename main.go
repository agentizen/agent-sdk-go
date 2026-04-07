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
	"context"
	"io"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	anthropicprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/anthropic"
	geminiprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/gemini"
	mistralprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/mistral"
	openaiprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/openai"
	"github.com/agentizen/agent-sdk-go/pkg/network"
	"github.com/agentizen/agent-sdk-go/pkg/plugin"
	"github.com/agentizen/agent-sdk-go/pkg/result"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/agentizen/agent-sdk-go/pkg/streaming"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
	"github.com/agentizen/agent-sdk-go/pkg/tooldata"
	"github.com/agentizen/agent-sdk-go/pkg/tracing"
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

	// Hooks defines the lifecycle hook interface for an agent.
	Hooks = agent.Hooks

	// DefaultAgentHooks provides a no-op embeddable implementation of Hooks.
	DefaultAgentHooks = agent.DefaultAgentHooks

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

	// StreamEvent is a single low-level event emitted during a streaming run.
	StreamEvent = model.StreamEvent

	// StreamingEvent is a higher-level event emitted by the enriched streaming
	// helpers.
	StreamingEvent = streaming.Event

	// NetworkStreamEvent is a single event emitted during a multi-agent network
	// streaming run.
	NetworkStreamEvent = network.NetworkStreamEvent

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

	// StreamResult aggregates content, thinking, usage, and tool lifecycle data
	// from an enriched streaming run.
	StreamResult = streaming.StreamResult

	// ToolCallRecord tracks a single tool invocation and its outcome during an
	// enriched streaming run.
	ToolCallRecord = streaming.ToolCallRecord

	// StreamEventRecord is a coalesced entry in the enriched streaming event
	// timeline.
	StreamEventRecord = streaming.StreamEventRecord

	// ToolDataBus is a request-scoped, thread-safe store for large binary
	// payloads that must reach tool handlers without transiting through the
	// LLM context. It lives for the duration of a single RunStreaming call.
	ToolDataBus = tooldata.ToolDataBus
)

// Streaming event type constants — forwarded from the model and streaming
// packages.
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

	// StreamingEventTypeThinkingStart is emitted when enriched streaming enters
	// the thinking phase.
	StreamingEventTypeThinkingStart = streaming.EventThinkingStart

	// StreamingEventTypeThinkingChunk is emitted for each enriched thinking chunk.
	StreamingEventTypeThinkingChunk = streaming.EventThinkingChunk

	// StreamingEventTypeThinkingEnd is emitted when the thinking phase ends.
	StreamingEventTypeThinkingEnd = streaming.EventThinkingEnd

	// StreamingEventTypeContentStart is emitted when enriched streaming enters
	// the content phase.
	StreamingEventTypeContentStart = streaming.EventContentStart

	// StreamingEventTypeContentChunk is emitted for each enriched content chunk.
	StreamingEventTypeContentChunk = streaming.EventContentChunk

	// StreamingEventTypeContentEnd is emitted when the content phase ends.
	StreamingEventTypeContentEnd = streaming.EventContentEnd

	// StreamingEventTypeToolCall is emitted when an enriched stream records a
	// tool invocation.
	StreamingEventTypeToolCall = streaming.EventToolCall

	// StreamingEventTypeToolCallResult is emitted when the last tool call is
	// resolved.
	StreamingEventTypeToolCallResult = streaming.EventToolCallResult

	// StreamingEventTypeHandoff is emitted when an enriched stream forwards a
	// handoff event.
	StreamingEventTypeHandoff = streaming.EventHandoff

	// StreamingEventTypeDone is emitted when the enriched stream completes.
	StreamingEventTypeDone = streaming.EventDone

	// StreamingEventTypeError is emitted when the enriched stream encounters an
	// unrecoverable error.
	StreamingEventTypeError = streaming.EventError

	// StreamingEventTypeAgentStart is emitted when a sub-agent begins work in a
	// network stream.
	StreamingEventTypeAgentStart = streaming.EventAgentStart

	// StreamingEventTypeAgentEnd is emitted when a sub-agent finishes work in a
	// network stream.
	StreamingEventTypeAgentEnd = streaming.EventAgentEnd
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

// --- Streaming helpers ---

// Enrich converts raw streaming events into higher-level streaming events with
// thinking extraction, tool lifecycle tracking, and aggregated results.
func Enrich(raw <-chan StreamEvent, bufferSize int) (<-chan StreamingEvent, *StreamResult) {
	return streaming.Enrich(raw, bufferSize)
}

// EnrichNetworkStream converts multi-agent network streaming events into
// higher-level streaming events with agent lifecycle tracking and aggregated
// results.
func EnrichNetworkStream(raw <-chan NetworkStreamEvent, bufferSize int) (<-chan StreamingEvent, *StreamResult) {
	return streaming.EnrichNetworkStream(raw, bufferSize)
}

// CoalesceEvents merges consecutive compatible streaming timeline records.
func CoalesceEvents(events []StreamEventRecord) []StreamEventRecord {
	return streaming.CoalesceEvents(events)
}

// ExtractThinkingText extracts `<think>...</think>` text from a stream chunk,
// if present.
func ExtractThinkingText(chunk string) (string, bool) {
	return streaming.ExtractThinkingText(chunk)
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

// NewLoadSkillTool creates a tool that allows an agent to load
// the full content of a skill by name at runtime.
func NewLoadSkillTool(skills []Skill) Tool {
	return skill.NewLoadSkillTool(skills)
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

// MCPToolsFromServer discovers all tools from an MCP server and returns them
// as standard tool.Tool values, ready to be attached to an agent.
func MCPToolsFromServer(ctx context.Context, server MCPServerConfig) ([]Tool, error) {
	return mcp.ToolsFromServer(ctx, server)
}

// --- Plugin factories ---

// NewPluginRegistry creates a new plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return plugin.NewRegistry()
}

// --- Model registry helpers ---

// GetProvider returns public metadata for a model provider.
func GetProvider(id string) (ProviderSpec, bool) {
	return model.GetProvider(id)
}

// GetModelMetadata returns descriptive metadata for a provider/model pair.
func GetModelMetadata(provider, modelID string) (ModelMetadata, bool) {
	return model.GetModelMetadata(provider, modelID)
}

// GetModelPricing returns pricing information for a provider/model pair.
func GetModelPricing(provider, modelID string) (ModelPricingSpec, bool) {
	return model.GetModelPricing(provider, modelID)
}

// GetModelSpec returns the complete registered specification for a model.
func GetModelSpec(provider, modelID string) (ModelSpec, bool) {
	return model.GetModelSpec(provider, modelID)
}

// ModelSpecsForProvider returns all registered models for the provider.
func ModelSpecsForProvider(provider string) []ModelSpec {
	return model.ModelSpecsForProvider(provider)
}

// AllProviders returns the registered provider list.
func AllProviders() []ProviderSpec {
	return model.AllProviders()
}

// ProviderSupports reports whether a provider/model exposes a capability.
func ProviderSupports(provider, modelID string, cap Capability) bool {
	return model.ProviderSupports(provider, modelID, cap)
}

// CapabilitiesFor returns the full capability set for a provider/model pair.
func CapabilitiesFor(provider, modelID string) ModelCapabilitySet {
	return model.CapabilitiesFor(provider, modelID)
}

// --- Network helpers ---

// NewNetworkConfig creates a default multi-agent network configuration.
func NewNetworkConfig() NetworkConfig {
	return network.NewNetworkConfig()
}

// NewNetworkRunner creates a multi-agent network runner from a base runner.
func NewNetworkRunner(base *Runner) *NetworkRunner {
	return network.NewNetworkRunner(base)
}

// --- MCP context helpers ---

// WithUserID attaches a user ID to the context for MCP transports.
func WithUserID(ctx context.Context, userID string) context.Context {
	return mcp.WithUserID(ctx, userID)
}

// UserIDFromContext extracts the MCP user ID from a context.
func UserIDFromContext(ctx context.Context) string {
	return mcp.UserIDFromContext(ctx)
}

// WithHeaders attaches additional MCP headers to the context.
func WithHeaders(ctx context.Context, headers map[string]string) context.Context {
	return mcp.WithHeaders(ctx, headers)
}

// HeadersFromContext extracts MCP headers from a context.
func HeadersFromContext(ctx context.Context) map[string]string {
	return mcp.HeadersFromContext(ctx)
}

// --- Tracing helpers ---

// SetGlobalTracer sets the process-wide tracer.
func SetGlobalTracer(tracer Tracer) {
	tracing.SetGlobalTracer(tracer)
}

// GetGlobalTracer returns the process-wide tracer.
func GetGlobalTracer() Tracer {
	return tracing.GetGlobalTracer()
}

// RecordEvent records an event using the process-wide tracer.
func RecordEvent(ctx context.Context, event Event) {
	tracing.RecordEvent(ctx, event)
}

// --- Provider constructors ---

// NewOpenAIProvider creates an OpenAI provider with default settings.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return openaiprovider.NewProvider(apiKey)
}

// NewAnthropicProvider creates an Anthropic provider with default settings.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return anthropicprovider.NewProvider(apiKey)
}

// NewGeminiProvider creates a Gemini provider with default settings.
func NewGeminiProvider(apiKey string) *GeminiProvider {
	return geminiprovider.NewProvider(apiKey)
}

// NewMistralProvider creates a Mistral provider with default settings.
func NewMistralProvider(apiKey string) *MistralProvider {
	return mistralprovider.NewProvider(apiKey)
}

// --- ToolDataBus helpers ---

// NewToolDataBus creates an empty ToolDataBus.
func NewToolDataBus() *ToolDataBus { return tooldata.NewToolDataBus() }

// WithToolDataBus returns a new context carrying the given bus.
func WithToolDataBus(ctx context.Context, bus *ToolDataBus) context.Context {
	return tooldata.WithBus(ctx, bus)
}

// ToolDataBusFromContext extracts the ToolDataBus from ctx.
// Returns nil if no bus is present — callers must handle nil.
func ToolDataBusFromContext(ctx context.Context) *ToolDataBus {
	return tooldata.BusFromContext(ctx)
}
