package model

import "testing"

func TestAllModelSpecsNonEmpty(t *testing.T) {
	specs := AllModelSpecs()
	if len(specs) == 0 {
		t.Fatal("AllModelSpecs returned no specs")
	}
}

func TestAllModelSpecsHaveRequiredFields(t *testing.T) {
	for _, spec := range AllModelSpecs() {
		if spec.Provider == "" {
			t.Errorf("spec missing Provider: %+v", spec)
		}
		if spec.ModelID == "" {
			t.Errorf("spec missing ModelID: %+v", spec)
		}
		if spec.Metadata.DisplayName == "" {
			t.Errorf("spec %s/%s missing Metadata.DisplayName", spec.Provider, spec.ModelID)
		}
		if spec.Metadata.ContextWindow <= 0 {
			t.Errorf("spec %s/%s has invalid ContextWindow: %d", spec.Provider, spec.ModelID, spec.Metadata.ContextWindow)
		}
		if spec.Metadata.MaxOutputTokens <= 0 {
			t.Errorf("spec %s/%s has invalid MaxOutputTokens: %d", spec.Provider, spec.ModelID, spec.Metadata.MaxOutputTokens)
		}
		if spec.Pricing.InputCostPerMillion < 0 {
			t.Errorf("spec %s/%s has negative InputCostPerMillion", spec.Provider, spec.ModelID)
		}
		if spec.Pricing.OutputCostPerMillion < 0 {
			t.Errorf("spec %s/%s has negative OutputCostPerMillion", spec.Provider, spec.ModelID)
		}
	}
}

func TestAllModelSpecsAreSorted(t *testing.T) {
	specs := AllModelSpecs()
	for i := 1; i < len(specs); i++ {
		prev, cur := specs[i-1], specs[i]
		if prev.Provider > cur.Provider {
			t.Errorf("specs not sorted by provider: %s > %s", prev.Provider, cur.Provider)
		}
		if prev.Provider == cur.Provider && prev.ModelID >= cur.ModelID {
			t.Errorf("specs not sorted by model ID within provider %s: %s >= %s", prev.Provider, prev.ModelID, cur.ModelID)
		}
	}
}

func TestGetModelSpecKnownModels(t *testing.T) {
	cases := []struct {
		provider   string
		modelID    string
		wantVision bool
	}{
		{"anthropic", "claude-haiku-4-5", true},
		{"anthropic", "claude-sonnet-4-6", true},
		{"anthropic", "claude-opus-4-6", true},
		{"gemini", "gemini-2.5-flash", true},
		{"gemini", "gemini-2.5-pro", true},
		{"mistral", "mistral-large-2512", true},
		{"mistral", "magistral-medium-2509", true},
		{"openai", "gpt-5.4", true},
		{"openai", "gpt-5.4-pro", true},
	}
	for _, tc := range cases {
		spec, ok := GetModelSpec(tc.provider, tc.modelID)
		if !ok {
			t.Errorf("GetModelSpec(%q, %q) returned false", tc.provider, tc.modelID)
			continue
		}
		if spec.Provider != tc.provider {
			t.Errorf("spec.Provider = %q, want %q", spec.Provider, tc.provider)
		}
		if spec.ModelID != tc.modelID {
			t.Errorf("spec.ModelID = %q, want %q", spec.ModelID, tc.modelID)
		}
		if spec.Capabilities.Vision != tc.wantVision {
			t.Errorf("spec %s/%s: Capabilities.Vision = %v, want %v",
				tc.provider, tc.modelID, spec.Capabilities.Vision, tc.wantVision)
		}
	}
}

func TestGetModelSpecUnknownReturnsFalse(t *testing.T) {
	if _, ok := GetModelSpec("unknown", "no-such-model"); ok {
		t.Error("GetModelSpec(unknown, no-such-model) should return false")
	}
}

func TestProviderSpecsHaveDocsURL(t *testing.T) {
	for _, id := range KnownProviders() {
		spec, ok := GetProvider(id)
		if !ok {
			t.Errorf("GetProvider(%q) returned false", id)
			continue
		}
		if spec.DocsURL == "" && id != "lmstudio" {
			t.Errorf("provider %q missing DocsURL", id)
		}
	}
}

func TestModelSpecsForProvider(t *testing.T) {
	specs := ModelSpecsForProvider("anthropic")
	if len(specs) == 0 {
		t.Fatal("ModelSpecsForProvider(anthropic) returned no specs")
	}
	for _, spec := range specs {
		if spec.Provider != "anthropic" {
			t.Errorf("expected provider anthropic, got %q", spec.Provider)
		}
	}
}

func TestGeminiProviderHasDocsAndPricingURLs(t *testing.T) {
	spec, ok := GetProvider("gemini")
	if !ok {
		t.Fatal("GetProvider(gemini) returned false")
	}
	if spec.DocsURL == "" {
		t.Error("gemini missing DocsURL")
	}
	if spec.PricingURL == "" {
		t.Error("gemini missing PricingURL")
	}
}
