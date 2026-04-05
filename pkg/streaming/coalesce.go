package streaming

// CoalesceEvents merges consecutive chunk entries of the same type
// (thinking_chunk, content_chunk) into a single entry to reduce payload size.
// Non-chunk events are passed through as-is.
func CoalesceEvents(events []StreamEventRecord) []StreamEventRecord {
	if len(events) == 0 {
		return events
	}
	result := make([]StreamEventRecord, 0, len(events))
	for _, e := range events {
		if e.Type == EventThinkingChunk || e.Type == EventContentChunk {
			last := len(result) - 1
			if last >= 0 && result[last].Type == e.Type {
				result[last].Content += e.Content
				continue
			}
		}
		result = append(result, e)
	}
	return result
}
