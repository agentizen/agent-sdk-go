package streaming

import (
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/network"
)

// EnrichNetworkStream transforms a raw NetworkStreamEvent channel into an
// enriched Event channel with agent lifecycle tracking, tool call resolution,
// content/thinking extraction, and result aggregation.
//
// The returned StreamResult is populated incrementally and is fully valid only
// after the Event channel has been drained (closed).
//
// The caller MUST drain the returned channel to avoid goroutine leaks.
func EnrichNetworkStream(raw <-chan network.NetworkStreamEvent, bufferSize int) (<-chan Event, *StreamResult) {
	if bufferSize < 1 {
		bufferSize = 1
	}
	// Adapt NetworkStreamEvent to model.StreamEvent + emit agent lifecycle events.
	adapted := make(chan model.StreamEvent, bufferSize)
	agentEvents := make(chan Event, bufferSize)
	sr := &StreamResult{}

	// Phase 1: Adapt network events into model events + agent lifecycle.
	go func() {
		defer close(adapted)
		defer close(agentEvents)

		var totalUsage model.Usage

		for evt := range raw {
			switch evt.Type {
			case network.EventSubAgentStart:
				agentEvents <- Event{
					Type:      EventAgentStart,
					AgentName: evt.AgentName,
					SubTaskID: evt.SubTaskID,
				}

			case network.EventSubAgentEnd:
				if evt.Usage != nil {
					totalUsage.PromptTokens += evt.Usage.PromptTokens
					totalUsage.CompletionTokens += evt.Usage.CompletionTokens
					totalUsage.TotalTokens += evt.Usage.TotalTokens
				}
				agentEvents <- Event{
					Type:       EventAgentEnd,
					AgentName:  evt.AgentName,
					SubTaskID:  evt.SubTaskID,
					DurationMs: evt.Duration.Milliseconds(),
				}

			case network.EventOrchestratorContent:
				adapted <- model.StreamEvent{
					Type:    model.StreamEventTypeContent,
					Content: evt.Content,
				}

			case network.EventOrchestratorToolCall:
				adapted <- model.StreamEvent{
					Type:     model.StreamEventTypeToolCall,
					ToolCall: evt.ToolCall,
				}

			case network.EventOrchestratorDone:
				if evt.Usage != nil {
					totalUsage.PromptTokens += evt.Usage.PromptTokens
					totalUsage.CompletionTokens += evt.Usage.CompletionTokens
					totalUsage.TotalTokens += evt.Usage.TotalTokens
				}
				adapted <- model.StreamEvent{
					Type: model.StreamEventTypeDone,
					Response: &model.Response{
						Usage: &totalUsage,
					},
				}

			case network.EventNetworkError:
				adapted <- model.StreamEvent{
					Type:  model.StreamEventTypeError,
					Error: evt.Error,
				}
			}
		}
	}()

	// Phase 2: Enrich the adapted model events.
	enrichedCh, enrichedResult := Enrich(adapted, bufferSize)

	// Phase 3: Merge agent lifecycle events with enriched model events.
	merged := make(chan Event, bufferSize)
	go func() {
		defer close(merged)

		var agentStreamEvents []StreamEventRecord
		agentEventsCh := agentEvents
		enrichedEventsCh := enrichedCh

		// Drain agent events and enriched events concurrently.
		// When one channel closes, set it to nil so its select case is disabled.
		for agentEventsCh != nil || enrichedEventsCh != nil {
			select {
			case evt, ok := <-agentEventsCh:
				if !ok {
					agentEventsCh = nil
					continue
				}
				agentStreamEvents = append(agentStreamEvents, StreamEventRecord{
					Type:       evt.Type,
					ID:         evt.SubTaskID,
					Name:       evt.AgentName,
					Status:     agentStatus(evt.Type),
					DurationMs: evt.DurationMs,
				})
				merged <- evt

			case evt, ok := <-enrichedEventsCh:
				if !ok {
					enrichedEventsCh = nil
					continue
				}
				merged <- evt
			}
		}

		// Copy the enriched fields we need into sr, adding agent events to the timeline.
		sr.Content = enrichedResult.Content
		sr.Thinking = enrichedResult.Thinking
		sr.Usage = enrichedResult.Usage
		sr.ThinkingDurationSeconds = enrichedResult.ThinkingDurationSeconds
		sr.ToolCalls = enrichedResult.ToolCalls
		sr.StreamEvents = CoalesceEvents(append(agentStreamEvents, enrichedResult.StreamEvents...))
	}()

	return merged, sr
}

func agentStatus(eventType string) string {
	if eventType == EventAgentEnd {
		return "done"
	}
	return "running"
}
