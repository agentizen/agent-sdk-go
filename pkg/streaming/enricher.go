package streaming

import (
	"strings"
	"time"

	"github.com/agentizen/agent-sdk-go/pkg/model"
)

// Enrich transforms a raw model.StreamEvent channel into an enriched Event
// channel with thinking extraction, phase boundaries, tool call lifecycle
// tracking, and result aggregation.
//
// The returned StreamResult is populated incrementally and is fully valid only
// after the Event channel has been drained (closed).
//
// The caller MUST drain the returned channel to avoid goroutine leaks.
func Enrich(raw <-chan model.StreamEvent, bufferSize int) (<-chan Event, *StreamResult) {
	if bufferSize < 1 {
		bufferSize = 1
	}
	out := make(chan Event, bufferSize)
	sr := &StreamResult{}

	go func() {
		defer close(out)
		runEnricher(raw, out, sr)
	}()

	return out, sr
}

// runEnricher is the stateful event processing loop.
func runEnricher(raw <-chan model.StreamEvent, out chan<- Event, sr *StreamResult) {
	var (
		fullContent         strings.Builder
		fullThinking        strings.Builder
		currentContentBlock strings.Builder

		thinkingPhaseOpen       bool
		contentPhaseOpen        bool
		thinkingStartedAt       time.Time
		thinkingDurationSeconds int

		lastPendingToolCallID        string
		toolCallsList                []ToolCallRecord
		toolCallIndexByID            = make(map[string]int)
		streamEvents                 []StreamEventRecord
		streamEventIndexByToolCallID = make(map[string]int)
		usageInfo                    *model.Usage
	)

	closeContentPhase := func() {
		if !contentPhaseOpen {
			return
		}
		contentPhaseOpen = false
		evt := StreamEventRecord{Type: EventContentEnd, Content: currentContentBlock.String()}
		streamEvents = append(streamEvents, evt)
		out <- Event{Type: EventContentEnd, Content: evt.Content}
		currentContentBlock.Reset()
	}

	closeThinkingPhase := func() {
		if !thinkingPhaseOpen {
			return
		}
		thinkingPhaseOpen = false
		streamEvents = append(streamEvents, StreamEventRecord{Type: EventThinkingEnd})
		out <- Event{Type: EventThinkingEnd}
	}

	resolvePendingToolCall := func(success bool) {
		if lastPendingToolCallID == "" {
			return
		}
		status := "done"
		if !success {
			status = "error"
		}
		if index, ok := toolCallIndexByID[lastPendingToolCallID]; ok {
			toolCallsList[index].Success = success
			delete(toolCallIndexByID, lastPendingToolCallID)
		}
		if index, ok := streamEventIndexByToolCallID[lastPendingToolCallID]; ok {
			streamEvents[index].Status = status
			delete(streamEventIndexByToolCallID, lastPendingToolCallID)
		}
		out <- Event{
			Type:            EventToolCallResult,
			ToolCallID:      lastPendingToolCallID,
			ToolCallSuccess: success,
		}
		lastPendingToolCallID = ""
	}

	computeThinkingDurationSeconds := func() int {
		if thinkingStartedAt.IsZero() || fullThinking.Len() == 0 {
			return 0
		}
		durationSeconds := int(time.Since(thinkingStartedAt).Seconds())
		if durationSeconds < 1 {
			durationSeconds = 1
		}
		return durationSeconds
	}

	for event := range raw {
		switch event.Type {
		case model.StreamEventTypeContent:
			resolvePendingToolCall(true)

			if thinkingChunk, ok := ExtractThinkingText(event.Content); ok {
				closeContentPhase()
				if thinkingStartedAt.IsZero() {
					thinkingStartedAt = time.Now()
				}
				if !thinkingPhaseOpen {
					thinkingPhaseOpen = true
					streamEvents = append(streamEvents, StreamEventRecord{Type: EventThinkingStart})
					out <- Event{Type: EventThinkingStart}
				}
				fullThinking.WriteString(thinkingChunk)
				streamEvents = append(streamEvents, StreamEventRecord{Type: EventThinkingChunk, Content: thinkingChunk})
				out <- Event{Type: EventThinkingChunk, Content: thinkingChunk}
				continue
			}

			closeThinkingPhase()
			if thinkingDurationSeconds == 0 {
				thinkingDurationSeconds = computeThinkingDurationSeconds()
			}

			if !contentPhaseOpen {
				contentPhaseOpen = true
				streamEvents = append(streamEvents, StreamEventRecord{Type: EventContentStart})
				out <- Event{Type: EventContentStart}
			}
			currentContentBlock.WriteString(event.Content)
			streamEvents = append(streamEvents, StreamEventRecord{Type: EventContentChunk, Content: event.Content})
			out <- Event{Type: EventContentChunk, Content: event.Content}
			fullContent.WriteString(event.Content)

		case model.StreamEventTypeToolCall:
			if event.ToolCall == nil {
				continue
			}
			closeContentPhase()
			resolvePendingToolCall(true)

			lastPendingToolCallID = event.ToolCall.ID
			toolCallsList = append(toolCallsList, ToolCallRecord{
				ID:      event.ToolCall.ID,
				Name:    event.ToolCall.Name,
				Success: true,
			})
			toolCallIndexByID[event.ToolCall.ID] = len(toolCallsList) - 1
			streamEvents = append(streamEvents, StreamEventRecord{
				Type:   EventToolCall,
				ID:     event.ToolCall.ID,
				Name:   event.ToolCall.Name,
				Status: "running",
			})
			streamEventIndexByToolCallID[event.ToolCall.ID] = len(streamEvents) - 1
			out <- Event{Type: EventToolCall, ToolCall: event.ToolCall}

		case model.StreamEventTypeHandoff:
			closeContentPhase()
			out <- Event{Type: EventHandoff, HandoffCall: event.HandoffCall}

		case model.StreamEventTypeDone:
			resolvePendingToolCall(true)
			if event.Response != nil && event.Response.Usage != nil {
				usageInfo = event.Response.Usage
			}
			out <- Event{Type: EventDone, Response: event.Response}

		case model.StreamEventTypeError:
			resolvePendingToolCall(false)
			out <- Event{Type: EventError, Error: event.Error}
		}
	}

	// Close any open phases at stream end.
	closeThinkingPhase()
	closeContentPhase()

	// Compute thinking duration if thinking happened but no content followed.
	if thinkingDurationSeconds == 0 {
		thinkingDurationSeconds = computeThinkingDurationSeconds()
	}

	// Populate the aggregated result.
	sr.Content = fullContent.String()
	sr.Thinking = fullThinking.String()
	sr.Usage = usageInfo
	sr.ThinkingDurationSeconds = thinkingDurationSeconds
	sr.ToolCalls = toolCallsList
	sr.StreamEvents = CoalesceEvents(streamEvents)
}
