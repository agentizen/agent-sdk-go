// Package streaming provides enriched event streaming for agent-sdk-go.
//
// Raw model.StreamEvent values from the runner are transformed into higher-level
// Event values with thinking extraction, phase boundaries, tool call lifecycle
// tracking, and result aggregation.
package streaming

import "github.com/agentizen/agent-sdk-go/pkg/model"

// Enriched event type constants.
const (
	// Thinking phase markers and chunks.
	EventThinkingStart = "thinking_start"
	EventThinkingChunk = "thinking_chunk"
	EventThinkingEnd   = "thinking_end"

	// Content phase markers and chunks.
	EventContentStart = "content_start"
	EventContentChunk = "content_chunk"
	EventContentEnd   = "content_end"

	// Tool call lifecycle.
	EventToolCall       = "tool_call"
	EventToolCallResult = "tool_call_result"

	// Handoff, completion, and error (forwarded from SDK).
	EventHandoff = "handoff"
	EventDone    = "done"
	EventError   = "error"

	// Network-specific agent lifecycle.
	EventAgentStart = "agent_start"
	EventAgentEnd   = "agent_end"
)

// Event is an enriched streaming event emitted by the Enrich pipeline.
// Depending on the Type, different fields are populated.
type Event struct {
	// Type identifies the event kind (one of the Event* constants).
	Type string

	// Content carries text for thinking/content chunks and the accumulated block
	// text for content_end events.
	Content string

	// ToolCall is populated on EventToolCall.
	ToolCall *model.ToolCall

	// HandoffCall is populated on EventHandoff.
	HandoffCall *model.HandoffCall

	// Response is populated on EventDone (carries Usage, FinalOutput, etc.).
	Response *model.Response

	// Error is populated on EventError.
	Error error

	// ToolCallID and ToolCallSuccess are populated on EventToolCallResult.
	ToolCallID      string
	ToolCallSuccess bool

	// Network-specific fields, populated on EventAgentStart / EventAgentEnd.
	AgentName  string
	SubTaskID  string
	DurationMs int64
}

// StreamResult aggregates all outputs from a streaming run.
// It is populated incrementally by the enricher goroutine and is fully valid
// only after the enriched Event channel has been drained.
type StreamResult struct {
	// Content is the full accumulated non-thinking text output.
	Content string

	// Thinking is the full accumulated thinking text (extracted from content events).
	Thinking string

	// Usage holds token consumption info from the final done event.
	Usage *model.Usage

	// ThinkingDurationSeconds is the wall-clock seconds from the first thinking chunk
	// to the first content chunk.
	ThinkingDurationSeconds int

	// ToolCalls records every tool invocation with its resolution status.
	ToolCalls []ToolCallRecord

	// StreamEvents is the coalesced timeline of all enriched events.
	StreamEvents []StreamEventRecord
}

// ToolCallRecord tracks a single tool invocation and its outcome.
type ToolCallRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Success bool   `json:"success"`
}

// StreamEventRecord is a single entry in the ordered event timeline.
// Consecutive chunks of the same type are coalesced before being stored
// in StreamResult.StreamEvents.
type StreamEventRecord struct {
	Type       string `json:"type"`
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Content    string `json:"content,omitempty"`
	Status     string `json:"status,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}
