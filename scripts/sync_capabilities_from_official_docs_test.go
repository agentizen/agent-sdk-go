package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestNormalizeModelIDAliases(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		allow bool
	}{
		{name: "gemini latest alias", in: "gemini-flash-latest", want: "gemini-flash-latest", allow: true},
		{name: "gemini latest alias pro", in: "gemini-pro-latest", want: "gemini-pro-latest", allow: true},
		{name: "bad token with slash", in: "gemini/pro/latest", want: "", allow: false},
		{name: "noise token", in: "gemini-card-2.0", want: "", allow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeModelID(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeModelID(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if tt.allow && got == "" {
				t.Fatalf("expected %q to be allowed", tt.in)
			}
		})
	}
}

func TestExtractModelIDsGeminiIncludesLatestAliases(t *testing.T) {
	content := strings.ToLower(`
		Gemini references:
		gemini-2.5-pro gemini-2.5-flash-lite gemini-flash-latest gemini-pro-latest
		garbage gemini/card/latest gemini-foo
	`)

	got := extractModelIDs(content, []*regexp.Regexp{regexp.MustCompile(`(?i)(gemini-[a-z0-9.-]+)`)}, geminiAllowRegexp.MatchString)
	joined := strings.Join(got, ",")

	mustContain := []string{"gemini-2.5-pro", "gemini-2.5-flash-lite", "gemini-flash-latest", "gemini-pro-latest"}
	for _, item := range mustContain {
		if !strings.Contains(joined, item) {
			t.Fatalf("expected %q in extracted model IDs, got %v", item, got)
		}
	}
}

func TestApplyMissingAndSortByProviderAlphabetical(t *testing.T) {
	input := `var providerCapabilities = map[string][]capabilityEntry{
	"mistral": {
		{prefix: "mistral-z", caps: map[Capability]bool{CapabilityVision: true}},
		{prefix: "mistral-a", caps: map[Capability]bool{CapabilityVision: true}},
	},
	"openai": {
		{prefix: "gpt-z", caps: map[Capability]bool{CapabilityVision: true}},
	},
	"anthropic": {
		{prefix: "claude-z", caps: map[Capability]bool{CapabilityVision: true}},
	},
	"gemini": {
		{prefix: "gemini-z", caps: map[Capability]bool{CapabilityVision: true}},
	},
}
`

	missing := map[string]map[string]capSet{
		"gemini": {
			"gemini-b": {vision: true, documents: true},
			"gemini-a": {vision: true, documents: true},
		},
	}

	out, err := applyMissingAndSortByProvider(input, missing)
	if err != nil {
		t.Fatalf("applyMissingAndSortByProvider returned error: %v", err)
	}

	geminiIdxA := strings.Index(out, `{prefix: "gemini-a"`)
	geminiIdxB := strings.Index(out, `{prefix: "gemini-b"`)
	geminiIdxZ := strings.Index(out, `{prefix: "gemini-z"`)
	if geminiIdxA < 0 || geminiIdxB < 0 || geminiIdxZ < 0 {
		t.Fatalf("missing gemini entries in output: %s", out)
	}
	if geminiIdxA >= geminiIdxB || geminiIdxB >= geminiIdxZ {
		t.Fatalf("gemini entries are not sorted alphabetically: %s", out)
	}
}
