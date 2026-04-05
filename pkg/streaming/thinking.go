package streaming

import (
	"encoding/json"
	"strings"
)

// thinkingPayload is the JSON structure used by providers (e.g. Anthropic) to
// embed extended thinking inside content stream chunks.
type thinkingPayload struct {
	Type     string `json:"type"`
	Thinking []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"thinking"`
}

// ExtractThinkingText checks whether a content chunk contains provider-embedded
// thinking text (e.g. Anthropic extended thinking JSON) and extracts it.
//
// Returns the concatenated thinking text and true if thinking was found,
// or ("", false) otherwise.
func ExtractThinkingText(chunk string) (string, bool) {
	trimmed := strings.TrimSpace(chunk)
	if trimmed == "" {
		return "", false
	}

	if !strings.Contains(trimmed, "thinking") {
		return "", false
	}

	// Try both the raw chunk and an un-escaped variant (some providers double-escape).
	candidates := []string{trimmed, strings.ReplaceAll(trimmed, `\"`, `"`)}
	for _, candidate := range candidates {
		var payloads []thinkingPayload
		if err := json.Unmarshal([]byte(candidate), &payloads); err != nil {
			continue
		}

		hasThinkingPayload := false
		var out strings.Builder
		for _, p := range payloads {
			if p.Type != "thinking" {
				continue
			}
			hasThinkingPayload = true
			for _, part := range p.Thinking {
				if part.Type != "text" || part.Text == "" {
					continue
				}
				out.WriteString(part.Text)
			}
		}

		if hasThinkingPayload {
			return out.String(), true
		}
	}

	return "", false
}
