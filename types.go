package agentsdk

import (
	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	anthropicprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/anthropic"
	geminiprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/gemini"
	mistralprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/mistral"
	openaiprovider "github.com/agentizen/agent-sdk-go/pkg/model/providers/openai"
	"github.com/agentizen/agent-sdk-go/pkg/network"
	"github.com/agentizen/agent-sdk-go/pkg/plugin"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/agentizen/agent-sdk-go/pkg/streaming"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
	"github.com/agentizen/agent-sdk-go/pkg/tracing"
)

// Extended type aliases for advanced use cases such as multi-agent workflows,
// guardrails, state management, and raw model interactions.
type (
	// WorkflowState tracks the current phase and artifacts of a running
	// multi-phase workflow.
	WorkflowState = runner.WorkflowState

	// Model is the interface implemented by all LLM backends.
	Model = model.Model

	// Provider resolves a model name to an executable model instance.
	Provider = model.Provider

	// Request is the structured input passed to a model.
	Request = model.Request

	// Response is the structured output returned by a model.
	Response = model.Response

	// ToolCall represents a function/tool invocation requested by the model.
	ToolCall = model.ToolCall

	// Settings configures model-level options such as temperature and max tokens.
	Settings = model.Settings

	// ContentPartType identifies the type of a multimodal content part.
	ContentPartType = model.ContentPartType

	// ValidationRule defines a predicate applied to data at handoff boundaries.
	ValidationRule = runner.ValidationRule

	// ValidationSeverity indicates whether a failed rule blocks progress or
	// only emits a warning.
	ValidationSeverity = runner.ValidationSeverity

	// StateManagementConfig configures workflow-state persistence and
	// checkpoint frequency.
	StateManagementConfig = runner.StateManagementConfig

	// ValidationConfig holds the set of rules applied before and after
	// handoffs.
	ValidationConfig = runner.ValidationConfig

	// RecoveryConfig controls automatic recovery from panics and transient
	// failures.
	RecoveryConfig = runner.RecoveryConfig

	// WorkflowStateStore is the interface for persisting and restoring
	// workflow state across checkpoints.
	WorkflowStateStore = runner.WorkflowStateStore

	// HandoffInputFilter transforms the input payload before it is forwarded
	// to the receiving agent during a handoff.
	HandoffInputFilter = runner.HandoffInputFilter

	// InputGuardrail validates agent input before each model call.
	InputGuardrail = runner.InputGuardrail

	// OutputGuardrail validates agent output after each model call.
	OutputGuardrail = runner.OutputGuardrail

	// ModelResponse is the raw, structured response returned by a model
	// provider after a non-streaming call.
	ModelResponse = model.Response

	// Capability identifies one model capability such as vision or thinking.
	Capability = model.Capability

	// ModelCapabilitySet is the resolved set of capabilities for a model.
	ModelCapabilitySet = model.ModelCapabilitySet

	// ModelMetadata contains descriptive metadata for a registered model.
	ModelMetadata = model.ModelMetadata

	// ModelPricingSpec contains pricing metadata for a registered model.
	ModelPricingSpec = model.ModelPricingSpec

	// ModelSpec is the complete, unified specification for a registered model.
	ModelSpec = model.ModelSpec

	// ProviderSpec describes a model provider.
	ProviderSpec = model.ProviderSpec

	// ModelRequest is the structured request sent to a model provider.
	ModelRequest = model.Request

	// Usage holds token-consumption data reported by a model provider.
	Usage = model.Usage

	// HandoffCall describes the parameters of an agent-to-agent handoff or
	// return-to-delegator event.
	HandoffCall = model.HandoffCall

	// Skill is a markdown document with a YAML header, loadable by an agent
	// via the load_skill tool.
	Skill = skill.Skill

	// SkillHeader contains skill metadata (name, description, version).
	SkillHeader = skill.Header

	// SkillRegistry manages skill discovery and storage.
	SkillRegistry = skill.Registry

	// MCPServerConfig describes an MCP server and its transport.
	MCPServerConfig = mcp.ServerConfig

	// MCPClient is the transport interface for communicating with MCP servers.
	MCPClient = mcp.Client

	// MCPHTTPClient is the default HTTP implementation of MCPClient.
	MCPHTTPClient = mcp.HTTPClient

	// MCPClientOptions configures the HTTP MCP client transport.
	MCPClientOptions = mcp.ClientOptions

	// MCPToolInfo describes a tool exposed by an MCP server.
	MCPToolInfo = mcp.ToolInfo

	// MCPRegistry manages MCP server configurations.
	MCPRegistry = mcp.Registry

	// DispatchStrategy controls how sub-tasks are distributed in a network.
	DispatchStrategy = network.DispatchStrategy

	// NetworkStreamEventType identifies a streamed network event.
	NetworkStreamEventType = network.NetworkStreamEventType

	// AgentSlot binds an agent to a role in a multi-agent network.
	AgentSlot = network.AgentSlot

	// NetworkConfig configures a multi-agent network execution.
	NetworkConfig = network.NetworkConfig

	// NetworkRunner executes a multi-agent network.
	NetworkRunner = network.NetworkRunner

	// NetworkResult is the aggregated result of a network execution.
	NetworkResult = network.NetworkResult

	// AgentRunResult is the result of one sub-agent run inside a network.
	AgentRunResult = network.AgentRunResult

	// Plugin is a bundle of tools, skills, and MCP servers pluggable to an agent.
	Plugin = plugin.Plugin

	// BasePlugin provides a default embeddable implementation of Plugin.
	BasePlugin = plugin.BasePlugin

	// PluginRegistry manages registered plugins.
	PluginRegistry = plugin.Registry

	// ToolRegistry is a thread-safe tool registry with group support.
	ToolRegistry = tool.Registry

	// ExecutableTool runs an external process as a tool.
	ExecutableTool = tool.ExecutableTool

	// ToolMiddleware wraps a Tool with additional behavior.
	ToolMiddleware = tool.Middleware

	// Event is a trace event emitted by the SDK tracer.
	Event = tracing.Event

	// Tracer is the interface implemented by tracing backends.
	Tracer = tracing.Tracer

	// FileTracer writes trace events to a local file.
	FileTracer = tracing.FileTracer

	// NoopTracer is a tracer implementation that discards all events.
	NoopTracer = tracing.NoopTracer

	// OpenAIProvider is the public OpenAI provider type.
	OpenAIProvider = openaiprovider.Provider

	// OpenAIAPIType identifies the OpenAI transport mode.
	OpenAIAPIType = openaiprovider.APIType

	// AnthropicProvider is the public Anthropic provider type.
	AnthropicProvider = anthropicprovider.Provider

	// GeminiProvider is the public Gemini provider type.
	GeminiProvider = geminiprovider.Provider

	// MistralProvider is the public Mistral provider type.
	MistralProvider = mistralprovider.Provider
)

// Validation severity constants.
const (
	// ValidationError is a blocking validation failure that halts the workflow.
	ValidationError = runner.ValidationError

	// StrategyParallel runs all agents concurrently.
	StrategyParallel = network.StrategyParallel

	// StrategySequential runs agents one after another.
	StrategySequential = network.StrategySequential

	// StrategyCompetitive sends the same prompt to all agents and keeps the first winner.
	StrategyCompetitive = network.StrategyCompetitive

	// EventSubAgentStart is emitted when a sub-agent starts work.
	EventSubAgentStart = network.EventSubAgentStart

	// EventSubAgentEnd is emitted when a sub-agent finishes work.
	EventSubAgentEnd = network.EventSubAgentEnd

	// EventOrchestratorContent is emitted for each orchestrator content chunk.
	EventOrchestratorContent = network.EventOrchestratorContent

	// EventOrchestratorDone is emitted when orchestrator synthesis completes.
	EventOrchestratorDone = network.EventOrchestratorDone

	// EventOrchestratorToolCall is emitted when the orchestrator invokes a tool.
	EventOrchestratorToolCall = network.EventOrchestratorToolCall

	// EventNetworkError is emitted on unrecoverable network execution errors.
	EventNetworkError = network.EventNetworkError

	// EventThinkingStart is emitted when enriched streaming enters the thinking phase.
	EventThinkingStart = streaming.EventThinkingStart

	// EventThinkingChunk is emitted for each thinking chunk.
	EventThinkingChunk = streaming.EventThinkingChunk

	// EventThinkingEnd is emitted when the thinking phase ends.
	EventThinkingEnd = streaming.EventThinkingEnd

	// EventContentStart is emitted when enriched streaming enters the content phase.
	EventContentStart = streaming.EventContentStart

	// EventContentChunk is emitted for each content chunk.
	EventContentChunk = streaming.EventContentChunk

	// EventContentEnd is emitted when the content phase ends.
	EventContentEnd = streaming.EventContentEnd

	// EventToolCall is emitted when an enriched stream records a tool call.
	EventToolCall = streaming.EventToolCall

	// EventToolCallResult is emitted when an enriched stream records a tool result.
	EventToolCallResult = streaming.EventToolCallResult

	// EventHandoff is emitted when an enriched stream forwards a handoff.
	EventHandoff = streaming.EventHandoff

	// EventDone is emitted when an enriched stream completes.
	EventDone = streaming.EventDone

	// EventError is emitted when an enriched stream encounters an error.
	EventError = streaming.EventError

	// EventAgentStart is emitted when a sub-agent starts in a network stream.
	EventAgentStart = streaming.EventAgentStart

	// EventAgentEnd is emitted when a sub-agent ends in a network stream.
	EventAgentEnd = streaming.EventAgentEnd

	// ValidationWarning is a non-blocking validation failure logged but not
	// halting.
	ValidationWarning = runner.ValidationWarning
)

// ContentPartType constants for multi-modal messages.
const (
	// ContentPartTypeText marks a plain-text segment.
	ContentPartTypeText = model.ContentPartTypeText

	// CapabilityAudioGeneration indicates audio generation support.
	CapabilityAudioGeneration = model.CapabilityAudioGeneration

	// CapabilityBatchAPI indicates batch API support.
	CapabilityBatchAPI = model.CapabilityBatchAPI

	// CapabilityCaching indicates input caching support.
	CapabilityCaching = model.CapabilityCaching

	// CapabilityCodeExecution indicates code execution support.
	CapabilityCodeExecution = model.CapabilityCodeExecution

	// CapabilityDocuments indicates native document support.
	CapabilityDocuments = model.CapabilityDocuments

	// CapabilityFileSearch indicates file search support.
	CapabilityFileSearch = model.CapabilityFileSearch

	// CapabilityFunctionCalling indicates function calling support.
	CapabilityFunctionCalling = model.CapabilityFunctionCalling

	// CapabilityImageGeneration indicates image generation support.
	CapabilityImageGeneration = model.CapabilityImageGeneration

	// CapabilityLiveAPI indicates live API support.
	CapabilityLiveAPI = model.CapabilityLiveAPI

	// CapabilityStructuredOutput indicates structured output support.
	CapabilityStructuredOutput = model.CapabilityStructuredOutput

	// CapabilityThinking indicates reasoning/thinking support.
	CapabilityThinking = model.CapabilityThinking

	// CapabilityVision indicates vision support.
	CapabilityVision = model.CapabilityVision

	// ContentPartTypeDocument marks a document segment (PDF, plain-text file).
	ContentPartTypeDocument = model.ContentPartTypeDocument

	// ContentPartTypeImage marks an image segment (PNG, JPEG, GIF, WEBP).
	ContentPartTypeImage = model.ContentPartTypeImage
)

// Handoff type constants.
const (
	// HandoffTypeDelegate indicates the current agent is delegating a task to
	// another agent.
	HandoffTypeDelegate = model.HandoffTypeDelegate

	// EventTypeAgentStart is emitted when an agent run begins.
	EventTypeAgentStart = tracing.EventTypeAgentStart

	// EventTypeAgentEnd is emitted when an agent run completes.
	EventTypeAgentEnd = tracing.EventTypeAgentEnd

	// EventTypeToolCall is emitted when a tool is invoked.
	EventTypeToolCall = tracing.EventTypeToolCall

	// EventTypeToolResult is emitted when a tool result is returned.
	EventTypeToolResult = tracing.EventTypeToolResult

	// EventTypeModelRequest is emitted before a model request.
	EventTypeModelRequest = tracing.EventTypeModelRequest

	// EventTypeModelResponse is emitted after a model response.
	EventTypeModelResponse = tracing.EventTypeModelResponse

	// EventTypeHandoff is emitted for handoff activity.
	EventTypeHandoff = tracing.EventTypeHandoff

	// EventTypeHandoffComplete is emitted when a handoff finishes.
	EventTypeHandoffComplete = tracing.EventTypeHandoffComplete

	// EventTypeAgentMessage is emitted for agent-generated messages.
	EventTypeAgentMessage = tracing.EventTypeAgentMessage

	// EventTypeError is emitted when tracing records an error.
	EventTypeError = tracing.EventTypeError

	// EventTypeSkillLoad is emitted when a skill is loaded.
	EventTypeSkillLoad = tracing.EventTypeSkillLoad

	// EventTypeMCPCall is emitted for MCP calls.
	EventTypeMCPCall = tracing.EventTypeMCPCall

	// EventTypeMCPResult is emitted for MCP results.
	EventTypeMCPResult = tracing.EventTypeMCPResult

	// EventTypePluginInit is emitted when a plugin is initialized.
	EventTypePluginInit = tracing.EventTypePluginInit

	// OpenAIAPITypeOpenAI identifies the default OpenAI API mode.
	OpenAIAPITypeOpenAI = openaiprovider.APITypeOpenAI

	// OpenAIAPITypeAzure identifies Azure OpenAI key-based mode.
	OpenAIAPITypeAzure = openaiprovider.APITypeAzure

	// OpenAIAPITypeAzureAD identifies Azure OpenAI AAD mode.
	OpenAIAPITypeAzureAD = openaiprovider.APITypeAzureAD

	// HandoffTypeReturn indicates the current agent is returning a completed
	// task result to its delegator.
	HandoffTypeReturn = model.HandoffTypeReturn
)
