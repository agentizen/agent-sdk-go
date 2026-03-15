package main

import (
	"strings"
	"testing"
)

func TestNormalizeModelIDAliases(t *testing.T) {
	rules := normalizeRules{allowNoDigitAliases: knownNoDigitModelAliases, rejectNoiseTokens: true}
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "gemini latest alias", in: "gemini-flash-latest", want: "gemini-flash-latest"},
		{name: "gemini latest alias pro", in: "gemini-pro-latest", want: "gemini-pro-latest"},
		{name: "bad token with slash", in: "gemini/pro/latest", want: ""},
		{name: "noise token", in: "gemini-card-2.0", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeModelID(tt.in, rules)
			if got != tt.want {
				t.Fatalf("normalizeModelID(%q) = %q, want %q", tt.in, got, tt.want)
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

	rules := normalizeRules{allowNoDigitAliases: knownNoDigitModelAliases, rejectNoiseTokens: true}
	got := extractModelIDs(content, []string{"gemini-"}, allowGeminiCapabilityID, rules)
	joined := strings.Join(got, ",")

	mustContain := []string{"gemini-2.5-pro", "gemini-2.5-flash-lite", "gemini-flash-latest", "gemini-pro-latest"}
	for _, item := range mustContain {
		if !strings.Contains(joined, item) {
			t.Fatalf("expected %q in extracted model IDs, got %v", item, got)
		}
	}
}

func TestApplyCapabilityUpdatesSortsAlphabetically(t *testing.T) {
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

	updates := capabilityUpdates{providers: []capabilityProviderUpdate{{
		providerID: "gemini",
		models: []capabilityModelUpdate{
			{modelID: "gemini-b", caps: capabilitySet{vision: true, documents: true}},
			{modelID: "gemini-a", caps: capabilitySet{vision: true, documents: true}},
		},
	}}}

	out, err := applyCapabilityUpdates(input, updates)
	if err != nil {
		t.Fatalf("applyCapabilityUpdates returned error: %v", err)
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

func TestApplyPricingUpdatesSortsAlphabetically(t *testing.T) {
	input := `var modelPricing = map[string]map[string]ModelPricingSpec{
	"anthropic": {
		"claude-z": {InputCostPerMillion: 1.0, OutputCostPerMillion: 2.0},
	},
	"gemini": {
		"gemini-z": {InputCostPerMillion: 1.0, OutputCostPerMillion: 2.0},
	},
	"mistral": {
		"mistral-z": {InputCostPerMillion: 1.0, OutputCostPerMillion: 2.0},
	},
	"openai": {
		"gpt-z": {InputCostPerMillion: 1.0, OutputCostPerMillion: 2.0},
	},
}
`

	updates := pricingUpdates{providers: []pricingProviderUpdate{{
		providerID: "gemini",
		models: []pricingModelUpdate{
			{modelID: "gemini-b", pricing: pricingSet{inputCost: 0.1, outputCost: 0.2, found: true}},
			{modelID: "gemini-a", pricing: pricingSet{inputCost: 0.1, outputCost: 0.2, found: true}},
		},
	}}}

	out, err := applyPricingUpdates(input, updates)
	if err != nil {
		t.Fatalf("applyPricingUpdates returned error: %v", err)
	}

	idxA := strings.Index(out, `"gemini-a"`)
	idxB := strings.Index(out, `"gemini-b"`)
	idxZ := strings.Index(out, `"gemini-z"`)
	if idxA < 0 || idxB < 0 {
		t.Fatalf("missing gemini pricing entries in output: %s", out)
	}
	if idxA >= idxB {
		t.Fatalf("gemini pricing entries are not sorted alphabetically: %s", out)
	}
	if idxZ >= 0 {
		t.Fatalf("stale gemini pricing entry should be replaced by scraped entries: %s", out)
	}
}
