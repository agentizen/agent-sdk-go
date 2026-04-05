package streaming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoalesceEvents(t *testing.T) {
	tests := []struct {
		name   string
		input  []StreamEventRecord
		expect []StreamEventRecord
	}{
		{
			name:   "empty slice",
			input:  []StreamEventRecord{},
			expect: []StreamEventRecord{},
		},
		{
			name:   "nil slice",
			input:  nil,
			expect: nil,
		},
		{
			name: "single event unchanged",
			input: []StreamEventRecord{
				{Type: EventContentChunk, Content: "hello"},
			},
			expect: []StreamEventRecord{
				{Type: EventContentChunk, Content: "hello"},
			},
		},
		{
			name: "consecutive thinking chunks merged",
			input: []StreamEventRecord{
				{Type: EventThinkingChunk, Content: "part1"},
				{Type: EventThinkingChunk, Content: "part2"},
				{Type: EventThinkingChunk, Content: "part3"},
			},
			expect: []StreamEventRecord{
				{Type: EventThinkingChunk, Content: "part1part2part3"},
			},
		},
		{
			name: "consecutive content chunks merged",
			input: []StreamEventRecord{
				{Type: EventContentChunk, Content: "Hello"},
				{Type: EventContentChunk, Content: " World"},
			},
			expect: []StreamEventRecord{
				{Type: EventContentChunk, Content: "Hello World"},
			},
		},
		{
			name: "mixed thinking and content NOT merged",
			input: []StreamEventRecord{
				{Type: EventThinkingChunk, Content: "think"},
				{Type: EventContentChunk, Content: "content"},
			},
			expect: []StreamEventRecord{
				{Type: EventThinkingChunk, Content: "think"},
				{Type: EventContentChunk, Content: "content"},
			},
		},
		{
			name: "non-chunk events pass through",
			input: []StreamEventRecord{
				{Type: EventThinkingStart},
				{Type: EventContentStart},
				{Type: EventToolCall, ID: "tc1", Name: "search"},
				{Type: EventContentEnd, Content: "final"},
			},
			expect: []StreamEventRecord{
				{Type: EventThinkingStart},
				{Type: EventContentStart},
				{Type: EventToolCall, ID: "tc1", Name: "search"},
				{Type: EventContentEnd, Content: "final"},
			},
		},
		{
			name: "complex sequence with interleaved types",
			input: []StreamEventRecord{
				{Type: EventThinkingStart},
				{Type: EventThinkingChunk, Content: "t1"},
				{Type: EventThinkingChunk, Content: "t2"},
				{Type: EventThinkingEnd},
				{Type: EventToolCall, ID: "tc1", Name: "search"},
				{Type: EventContentStart},
				{Type: EventContentChunk, Content: "c1"},
				{Type: EventContentChunk, Content: "c2"},
				{Type: EventContentEnd, Content: "c1c2"},
			},
			expect: []StreamEventRecord{
				{Type: EventThinkingStart},
				{Type: EventThinkingChunk, Content: "t1t2"},
				{Type: EventThinkingEnd},
				{Type: EventToolCall, ID: "tc1", Name: "search"},
				{Type: EventContentStart},
				{Type: EventContentChunk, Content: "c1c2"},
				{Type: EventContentEnd, Content: "c1c2"},
			},
		},
		{
			name: "non-consecutive same-type chunks not merged",
			input: []StreamEventRecord{
				{Type: EventContentChunk, Content: "a"},
				{Type: EventToolCall, ID: "tc1"},
				{Type: EventContentChunk, Content: "b"},
			},
			expect: []StreamEventRecord{
				{Type: EventContentChunk, Content: "a"},
				{Type: EventToolCall, ID: "tc1"},
				{Type: EventContentChunk, Content: "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoalesceEvents(tt.input)
			assert.Equal(t, tt.expect, got)
		})
	}
}
