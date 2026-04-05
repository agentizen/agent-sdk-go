package streaming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractThinkingText(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantText  string
		wantFound bool
	}{
		{
			name:      "empty string",
			input:     "",
			wantText:  "",
			wantFound: false,
		},
		{
			name:      "whitespace only",
			input:     "   \t\n  ",
			wantText:  "",
			wantFound: false,
		},
		{
			name:      "no thinking keyword",
			input:     `[{"type":"content","data":"hello"}]`,
			wantText:  "",
			wantFound: false,
		},
		{
			name:      "valid thinking JSON",
			input:     `[{"type":"thinking","thinking":[{"type":"text","text":"I need to analyze this."}]}]`,
			wantText:  "I need to analyze this.",
			wantFound: true,
		},
		{
			name:      "escaped thinking JSON",
			input:     `[{\"type\":\"thinking\",\"thinking\":[{\"type\":\"text\",\"text\":\"Let me think.\"}]}]`,
			wantText:  "Let me think.",
			wantFound: true,
		},
		{
			name:      "empty thinking array",
			input:     `[{"type":"thinking","thinking":[]}]`,
			wantText:  "",
			wantFound: true,
		},
		{
			name:      "multiple thinking parts",
			input:     `[{"type":"thinking","thinking":[{"type":"text","text":"Part one. "},{"type":"text","text":"Part two."}]}]`,
			wantText:  "Part one. Part two.",
			wantFound: true,
		},
		{
			name:      "non-thinking type",
			input:     `[{"type":"content","thinking":[{"type":"text","text":"Not thinking."}]}]`,
			wantText:  "",
			wantFound: false,
		},
		{
			name:      "invalid JSON with thinking keyword",
			input:     `{thinking: broken json}`,
			wantText:  "",
			wantFound: false,
		},
		{
			name:      "multiple payloads with mixed types",
			input:     `[{"type":"thinking","thinking":[{"type":"text","text":"A"}]},{"type":"other","thinking":[]}]`,
			wantText:  "A",
			wantFound: true,
		},
		{
			name:      "thinking part with non-text type",
			input:     `[{"type":"thinking","thinking":[{"type":"image","text":"ignored"}]}]`,
			wantText:  "",
			wantFound: true,
		},
		{
			name:      "thinking part with empty text",
			input:     `[{"type":"thinking","thinking":[{"type":"text","text":""}]}]`,
			wantText:  "",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotFound := ExtractThinkingText(tt.input)
			assert.Equal(t, tt.wantFound, gotFound, "found mismatch")
			assert.Equal(t, tt.wantText, gotText, "text mismatch")
		})
	}
}
