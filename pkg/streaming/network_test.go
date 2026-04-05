package streaming

import (
	"errors"
	"testing"
	"time"

	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/network"
	"github.com/stretchr/testify/assert"
)

// drainNetworkEvents reads all events from the channel into a slice.
func drainNetworkEvents(ch <-chan Event) []Event {
	var events []Event
	for evt := range ch {
		events = append(events, evt)
	}
	return events
}

func TestEnrichNetworkStream_AgentLifecycle(t *testing.T) {
	raw := make(chan network.NetworkStreamEvent, 10)
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentStart,
		AgentName: "finance-agent",
		SubTaskID: "task-1",
	}
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentEnd,
		AgentName: "finance-agent",
		SubTaskID: "task-1",
		Duration:  500 * time.Millisecond,
		Usage:     &model.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}
	raw <- network.NetworkStreamEvent{
		Type:  network.EventOrchestratorDone,
		Usage: &model.Usage{PromptTokens: 20, CompletionTokens: 10, TotalTokens: 30},
	}
	close(raw)

	ch, result := EnrichNetworkStream(raw, 16)
	events := drainNetworkEvents(ch)

	// Find agent_start and agent_end events.
	var agentStartFound, agentEndFound bool
	for _, evt := range events {
		switch evt.Type {
		case EventAgentStart:
			agentStartFound = true
			assert.Equal(t, "finance-agent", evt.AgentName)
			assert.Equal(t, "task-1", evt.SubTaskID)
		case EventAgentEnd:
			agentEndFound = true
			assert.Equal(t, "finance-agent", evt.AgentName)
			assert.Equal(t, "task-1", evt.SubTaskID)
			assert.Equal(t, int64(500), evt.DurationMs)
		}
	}
	assert.True(t, agentStartFound, "expected agent_start event")
	assert.True(t, agentEndFound, "expected agent_end event")

	// Usage should be aggregated.
	assert.NotNil(t, result.Usage)
	assert.Equal(t, 30, result.Usage.PromptTokens)
	assert.Equal(t, 15, result.Usage.CompletionTokens)
	assert.Equal(t, 45, result.Usage.TotalTokens)
}

func TestEnrichNetworkStream_OrchestratorContent(t *testing.T) {
	raw := make(chan network.NetworkStreamEvent, 10)
	raw <- network.NetworkStreamEvent{
		Type:    network.EventOrchestratorContent,
		Content: "Hello ",
	}
	raw <- network.NetworkStreamEvent{
		Type:    network.EventOrchestratorContent,
		Content: "World",
	}
	raw <- network.NetworkStreamEvent{
		Type: network.EventOrchestratorDone,
	}
	close(raw)

	ch, result := EnrichNetworkStream(raw, 16)
	events := drainNetworkEvents(ch)

	types := make([]string, 0)
	for _, evt := range events {
		types = append(types, evt.Type)
	}

	assert.Contains(t, types, EventContentStart)
	assert.Contains(t, types, EventContentChunk)
	assert.Contains(t, types, EventContentEnd)
	assert.Contains(t, types, EventDone)

	assert.Equal(t, "Hello World", result.Content)
}

func TestEnrichNetworkStream_OrchestratorToolCall(t *testing.T) {
	tc := &model.ToolCall{ID: "ntc-1", Name: "network_search"}

	raw := make(chan network.NetworkStreamEvent, 10)
	raw <- network.NetworkStreamEvent{
		Type:     network.EventOrchestratorToolCall,
		ToolCall: tc,
	}
	raw <- network.NetworkStreamEvent{
		Type:    network.EventOrchestratorContent,
		Content: "Result",
	}
	raw <- network.NetworkStreamEvent{
		Type: network.EventOrchestratorDone,
	}
	close(raw)

	ch, result := EnrichNetworkStream(raw, 16)
	events := drainNetworkEvents(ch)

	types := make([]string, 0)
	for _, evt := range events {
		types = append(types, evt.Type)
	}

	assert.Contains(t, types, EventToolCall)
	assert.Contains(t, types, EventToolCallResult)

	assert.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "ntc-1", result.ToolCalls[0].ID)
	assert.True(t, result.ToolCalls[0].Success)
}

func TestEnrichNetworkStream_UsageAggregation(t *testing.T) {
	raw := make(chan network.NetworkStreamEvent, 10)
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentStart,
		AgentName: "agent-a",
		SubTaskID: "t1",
	}
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentEnd,
		AgentName: "agent-a",
		SubTaskID: "t1",
		Duration:  100 * time.Millisecond,
		Usage:     &model.Usage{PromptTokens: 50, CompletionTokens: 25, TotalTokens: 75},
	}
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentStart,
		AgentName: "agent-b",
		SubTaskID: "t2",
	}
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentEnd,
		AgentName: "agent-b",
		SubTaskID: "t2",
		Duration:  200 * time.Millisecond,
		Usage:     &model.Usage{PromptTokens: 30, CompletionTokens: 15, TotalTokens: 45},
	}
	raw <- network.NetworkStreamEvent{
		Type:  network.EventOrchestratorDone,
		Usage: &model.Usage{PromptTokens: 20, CompletionTokens: 10, TotalTokens: 30},
	}
	close(raw)

	ch, result := EnrichNetworkStream(raw, 16)
	drainNetworkEvents(ch)

	assert.NotNil(t, result.Usage)
	assert.Equal(t, 100, result.Usage.PromptTokens)    // 50+30+20
	assert.Equal(t, 50, result.Usage.CompletionTokens) // 25+15+10
	assert.Equal(t, 150, result.Usage.TotalTokens)     // 75+45+30
}

func TestEnrichNetworkStream_Error(t *testing.T) {
	raw := make(chan network.NetworkStreamEvent, 10)
	raw <- network.NetworkStreamEvent{
		Type:    network.EventOrchestratorContent,
		Content: "Partial",
	}
	raw <- network.NetworkStreamEvent{
		Type:  network.EventNetworkError,
		Error: errors.New("network failure"),
	}
	close(raw)

	ch, _ := EnrichNetworkStream(raw, 16)
	events := drainNetworkEvents(ch)

	types := make([]string, 0)
	for _, evt := range events {
		types = append(types, evt.Type)
	}

	assert.Contains(t, types, EventError)

	// Find the error event and verify it carries the error.
	for _, evt := range events {
		if evt.Type == EventError {
			assert.Error(t, evt.Error)
			assert.Equal(t, "network failure", evt.Error.Error())
		}
	}
}

func TestEnrichNetworkStream_FullSequence(t *testing.T) {
	raw := make(chan network.NetworkStreamEvent, 20)

	// Sub-agent lifecycle
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentStart,
		AgentName: "researcher",
		SubTaskID: "sub-1",
	}
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentEnd,
		AgentName: "researcher",
		SubTaskID: "sub-1",
		Duration:  300 * time.Millisecond,
		Usage:     &model.Usage{PromptTokens: 40, CompletionTokens: 20, TotalTokens: 60},
	}

	// Orchestrator content
	raw <- network.NetworkStreamEvent{
		Type:    network.EventOrchestratorContent,
		Content: "Based on research: ",
	}
	raw <- network.NetworkStreamEvent{
		Type:    network.EventOrchestratorContent,
		Content: "the answer is 42.",
	}

	// Done
	raw <- network.NetworkStreamEvent{
		Type:  network.EventOrchestratorDone,
		Usage: &model.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}
	close(raw)

	ch, result := EnrichNetworkStream(raw, 16)
	events := drainNetworkEvents(ch)

	// Should contain agent lifecycle + content enrichment + done.
	types := make([]string, 0)
	for _, evt := range events {
		types = append(types, evt.Type)
	}

	assert.Contains(t, types, EventAgentStart)
	assert.Contains(t, types, EventAgentEnd)
	assert.Contains(t, types, EventContentStart)
	assert.Contains(t, types, EventContentChunk)
	assert.Contains(t, types, EventContentEnd)
	assert.Contains(t, types, EventDone)

	// Verify result.
	assert.Equal(t, "Based on research: the answer is 42.", result.Content)
	assert.NotNil(t, result.Usage)
	assert.Equal(t, 50, result.Usage.PromptTokens)
	assert.Equal(t, 25, result.Usage.CompletionTokens)
	assert.Equal(t, 75, result.Usage.TotalTokens)

	// StreamEvents should include agent events.
	hasAgentStart := false
	for _, se := range result.StreamEvents {
		if se.Type == EventAgentStart {
			hasAgentStart = true
			assert.Equal(t, "researcher", se.Name)
			assert.Equal(t, "sub-1", se.ID)
			assert.Equal(t, "running", se.Status)
		}
	}
	assert.True(t, hasAgentStart, "expected agent_start in StreamEvents")
}

func TestEnrichNetworkStream_EmptyStream(t *testing.T) {
	raw := make(chan network.NetworkStreamEvent, 10)
	close(raw)

	ch, result := EnrichNetworkStream(raw, 16)
	events := drainNetworkEvents(ch)

	assert.Empty(t, events)
	assert.Empty(t, result.Content)
	assert.Nil(t, result.Usage)
}

func TestEnrichNetworkStream_SubAgentEndWithNilUsage(t *testing.T) {
	raw := make(chan network.NetworkStreamEvent, 10)
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentStart,
		AgentName: "agent-x",
		SubTaskID: "t-nil",
	}
	raw <- network.NetworkStreamEvent{
		Type:      network.EventSubAgentEnd,
		AgentName: "agent-x",
		SubTaskID: "t-nil",
		Duration:  100 * time.Millisecond,
		Usage:     nil, // no usage
	}
	raw <- network.NetworkStreamEvent{
		Type:  network.EventOrchestratorDone,
		Usage: &model.Usage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
	}
	close(raw)

	ch, result := EnrichNetworkStream(raw, 16)
	drainNetworkEvents(ch)

	// Only orchestrator usage counted.
	assert.NotNil(t, result.Usage)
	assert.Equal(t, 5, result.Usage.PromptTokens)
	assert.Equal(t, 3, result.Usage.CompletionTokens)
	assert.Equal(t, 8, result.Usage.TotalTokens)
}

func TestAgentStatus(t *testing.T) {
	assert.Equal(t, "done", agentStatus(EventAgentEnd))
	assert.Equal(t, "running", agentStatus(EventAgentStart))
	assert.Equal(t, "running", agentStatus("anything_else"))
}
