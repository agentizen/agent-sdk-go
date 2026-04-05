package streaming

import (
	"errors"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/stretchr/testify/assert"
)

// drainEvents reads all events from the channel into a slice.
func drainEvents(ch <-chan Event) []Event {
	var events []Event
	for evt := range ch {
		events = append(events, evt)
	}
	return events
}

// eventTypes extracts just the Type fields from a slice of events.
func eventTypes(events []Event) []string {
	types := make([]string, len(events))
	for i, e := range events {
		types[i] = e.Type
	}
	return types
}

func TestEnrich_SimpleContent(t *testing.T) {
	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Hello"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: " World"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{
		EventContentStart,
		EventContentChunk,
		EventContentChunk,
		EventDone,
		EventContentEnd,
	}, types)

	assert.Equal(t, "Hello World", result.Content)
	assert.Empty(t, result.Thinking)
	assert.Empty(t, result.ToolCalls)
}

func TestEnrich_ThinkingThenContent(t *testing.T) {
	thinkingJSON := `[{"type":"thinking","thinking":[{"type":"text","text":"Let me think."}]}]`

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: thinkingJSON}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Answer"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{
		EventThinkingStart,
		EventThinkingChunk,
		EventThinkingEnd,
		EventContentStart,
		EventContentChunk,
		EventDone,
		EventContentEnd,
	}, types)

	assert.Equal(t, "Answer", result.Content)
	assert.Equal(t, "Let me think.", result.Thinking)
	assert.GreaterOrEqual(t, result.ThinkingDurationSeconds, 1)
}

func TestEnrich_InterleavedThinkingAndContent(t *testing.T) {
	think1 := `[{"type":"thinking","thinking":[{"type":"text","text":"Think1"}]}]`
	think2 := `[{"type":"thinking","thinking":[{"type":"text","text":"Think2"}]}]`

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: think1}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Content1"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: think2}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Content2"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{
		// First thinking block
		EventThinkingStart,
		EventThinkingChunk,
		// Transition to content
		EventThinkingEnd,
		EventContentStart,
		EventContentChunk,
		// Second thinking block
		EventContentEnd,
		EventThinkingStart,
		EventThinkingChunk,
		// Second content block
		EventThinkingEnd,
		EventContentStart,
		EventContentChunk,
		// Done and close
		EventDone,
		EventContentEnd,
	}, types)

	assert.Equal(t, "Content1Content2", result.Content)
	assert.Equal(t, "Think1Think2", result.Thinking)
}

func TestEnrich_ToolCallLifecycle(t *testing.T) {
	tc := &model.ToolCall{ID: "tc-1", Name: "search"}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Before"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: tc}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "After"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{
		EventContentStart,
		EventContentChunk,
		EventContentEnd, // closed before tool call
		EventToolCall,
		EventToolCallResult, // resolved by next content
		EventContentStart,
		EventContentChunk,
		EventDone,
		EventContentEnd,
	}, types)

	// Verify tool call event has correct data.
	toolCallEvt := events[3]
	assert.Equal(t, "tc-1", toolCallEvt.ToolCall.ID)
	assert.Equal(t, "search", toolCallEvt.ToolCall.Name)

	// Verify tool call result.
	toolResultEvt := events[4]
	assert.Equal(t, "tc-1", toolResultEvt.ToolCallID)
	assert.True(t, toolResultEvt.ToolCallSuccess)

	// Verify aggregated result.
	assert.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "tc-1", result.ToolCalls[0].ID)
	assert.Equal(t, "search", result.ToolCalls[0].Name)
	assert.True(t, result.ToolCalls[0].Success)
}

func TestEnrich_NilToolCallIgnored(t *testing.T) {
	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: nil}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{EventDone}, types)
	assert.Empty(t, result.ToolCalls)
}

func TestEnrich_DoneWithUsage(t *testing.T) {
	usage := &model.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	resp := &model.Response{Usage: usage, Content: "Final"}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Hello"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone, Response: resp}
	close(raw)

	ch, result := Enrich(raw, 16)
	drainEvents(ch)

	assert.NotNil(t, result.Usage)
	assert.Equal(t, 100, result.Usage.PromptTokens)
	assert.Equal(t, 50, result.Usage.CompletionTokens)
	assert.Equal(t, 150, result.Usage.TotalTokens)
}

func TestEnrich_DoneWithoutUsage(t *testing.T) {
	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone, Response: &model.Response{}}
	close(raw)

	ch, result := Enrich(raw, 16)
	drainEvents(ch)

	assert.Nil(t, result.Usage)
}

func TestEnrich_ErrorEvent(t *testing.T) {
	tc := &model.ToolCall{ID: "tc-err", Name: "execute"}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: tc}
	raw <- model.StreamEvent{Type: model.StreamEventTypeError, Error: errors.New("something broke")}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{
		EventToolCall,
		EventToolCallResult, // resolved as failure
		EventError,
	}, types)

	// Tool call resolved as failure.
	toolResult := events[1]
	assert.Equal(t, "tc-err", toolResult.ToolCallID)
	assert.False(t, toolResult.ToolCallSuccess)

	// Error event carries the error.
	assert.Error(t, events[2].Error)
	assert.Equal(t, "something broke", events[2].Error.Error())

	// Result records failure.
	assert.Len(t, result.ToolCalls, 1)
	assert.False(t, result.ToolCalls[0].Success)
}

func TestEnrich_ContentPhaseClosedAtEnd(t *testing.T) {
	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Hello"}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{
		EventContentStart,
		EventContentChunk,
		EventContentEnd, // emitted at channel close
	}, types)

	assert.Equal(t, "Hello", result.Content)
}

func TestEnrich_ThinkingPhaseClosedAtEnd(t *testing.T) {
	thinkingJSON := `[{"type":"thinking","thinking":[{"type":"text","text":"Thinking..."}]}]`

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: thinkingJSON}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Equal(t, []string{
		EventThinkingStart,
		EventThinkingChunk,
		EventThinkingEnd, // closed at channel close
	}, types)

	assert.Equal(t, "Thinking...", result.Thinking)
	assert.Empty(t, result.Content)
	assert.GreaterOrEqual(t, result.ThinkingDurationSeconds, 1)
}

func TestEnrich_EmptyStream(t *testing.T) {
	raw := make(chan model.StreamEvent, 10)
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	assert.Empty(t, events)
	assert.Empty(t, result.Content)
	assert.Empty(t, result.Thinking)
	assert.Nil(t, result.Usage)
	assert.Empty(t, result.ToolCalls)
	assert.Empty(t, result.StreamEvents)
	assert.Equal(t, 0, result.ThinkingDurationSeconds)
}

func TestEnrich_StreamResultAggregation(t *testing.T) {
	thinkingJSON := `[{"type":"thinking","thinking":[{"type":"text","text":"Hmm"}]}]`
	tc := &model.ToolCall{ID: "tc-agg", Name: "lookup"}
	usage := &model.Usage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: thinkingJSON}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Result: "}
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: tc}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "done."}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone, Response: &model.Response{Usage: usage}}
	close(raw)

	ch, result := Enrich(raw, 16)
	drainEvents(ch)

	assert.Equal(t, "Result: done.", result.Content)
	assert.Equal(t, "Hmm", result.Thinking)
	assert.NotNil(t, result.Usage)
	assert.Equal(t, 300, result.Usage.TotalTokens)
	assert.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "tc-agg", result.ToolCalls[0].ID)
	assert.True(t, result.ToolCalls[0].Success)
	assert.GreaterOrEqual(t, result.ThinkingDurationSeconds, 1)

	// StreamEvents should be coalesced and non-empty.
	assert.NotEmpty(t, result.StreamEvents)
}

func TestEnrich_HandoffEvent(t *testing.T) {
	hc := &model.HandoffCall{AgentName: "finance-agent"}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "Routing..."}
	raw <- model.StreamEvent{Type: model.StreamEventTypeHandoff, HandoffCall: hc}
	close(raw)

	ch, _ := Enrich(raw, 16)
	events := drainEvents(ch)

	types := eventTypes(events)
	assert.Contains(t, types, EventHandoff)

	// Find the handoff event.
	for _, evt := range events {
		if evt.Type == EventHandoff {
			assert.Equal(t, "finance-agent", evt.HandoffCall.AgentName)
		}
	}
}

func TestEnrich_MultipleToolCalls(t *testing.T) {
	tc1 := &model.ToolCall{ID: "tc-1", Name: "search"}
	tc2 := &model.ToolCall{ID: "tc-2", Name: "execute"}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: tc1}
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: tc2}
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone}
	close(raw)

	ch, result := Enrich(raw, 16)
	events := drainEvents(ch)

	// tc1 resolved by tc2, tc2 resolved by done.
	types := eventTypes(events)
	assert.Equal(t, []string{
		EventToolCall,
		EventToolCallResult, // tc1 resolved
		EventToolCall,
		EventToolCallResult, // tc2 resolved
		EventDone,
	}, types)

	assert.Len(t, result.ToolCalls, 2)
	assert.True(t, result.ToolCalls[0].Success)
	assert.True(t, result.ToolCalls[1].Success)
}

func TestEnrich_StreamEventsTrackToolCallStatuses(t *testing.T) {
	tcSuccess := &model.ToolCall{ID: "tc-ok", Name: "search"}
	tcError := &model.ToolCall{ID: "tc-bad", Name: "execute"}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: tcSuccess}
	raw <- model.StreamEvent{Type: model.StreamEventTypeContent, Content: "resolved"}
	raw <- model.StreamEvent{Type: model.StreamEventTypeToolCall, ToolCall: tcError}
	raw <- model.StreamEvent{Type: model.StreamEventTypeError, Error: errors.New("boom")}
	close(raw)

	ch, result := Enrich(raw, 16)
	drainEvents(ch)

	if assert.Len(t, result.ToolCalls, 2) {
		assert.True(t, result.ToolCalls[0].Success)
		assert.False(t, result.ToolCalls[1].Success)
	}

	var toolEvents []StreamEventRecord
	for _, evt := range result.StreamEvents {
		if evt.Type == EventToolCall {
			toolEvents = append(toolEvents, evt)
		}
	}

	if assert.Len(t, toolEvents, 2) {
		assert.Equal(t, "tc-ok", toolEvents[0].ID)
		assert.Equal(t, "done", toolEvents[0].Status)
		assert.Equal(t, "tc-bad", toolEvents[1].ID)
		assert.Equal(t, "error", toolEvents[1].Status)
	}
}

func TestEnrich_DoneResponseForwarded(t *testing.T) {
	resp := &model.Response{
		Content: "Final answer",
		Usage:   &model.Usage{TotalTokens: 42},
	}

	raw := make(chan model.StreamEvent, 10)
	raw <- model.StreamEvent{Type: model.StreamEventTypeDone, Response: resp}
	close(raw)

	ch, _ := Enrich(raw, 16)
	events := drainEvents(ch)

	assert.Len(t, events, 1)
	assert.Equal(t, EventDone, events[0].Type)
	assert.NotNil(t, events[0].Response)
	assert.Equal(t, "Final answer", events[0].Response.Content)
}
