package model

import "sort"

// ProviderSpec holds all static metadata for an LLM provider registered in the SDK.
type ProviderSpec struct {
	// ID is the canonical lowercase identifier used throughout the SDK (e.g. "anthropic").
	ID string

	// DisplayName is the human-readable name shown in UIs (e.g. "Anthropic Claude").
	DisplayName string

	// BaseURL is the default API base URL for this provider.
	BaseURL string

	// DocsURL is the official models documentation page for this provider.
	DocsURL string

	// PricingURL is the official pricing page for this provider.
	// Empty for local / self-hosted providers (e.g. lmstudio).
	PricingURL string
}

var providerSpecs = map[string]ProviderSpec{
	"anthropic": {
		ID:          "anthropic",
		DisplayName: "Anthropic Claude",
		BaseURL:     "https://api.anthropic.com",
		DocsURL:     "https://platform.claude.com/docs/en/docs/about-claude/models",
		PricingURL:  "https://docs.anthropic.com/en/docs/about-claude/models",
	},
	"gemini": {
		ID:          "gemini",
		DisplayName: "Google Gemini",
		BaseURL:     "https://generativelanguage.googleapis.com/v1beta",
		DocsURL:     "https://ai.google.dev/gemini-api/docs/models",
		PricingURL:  "https://ai.google.dev/gemini-api/docs/pricing",
	},
	"mistral": {
		ID:          "mistral",
		DisplayName: "Mistral AI",
		BaseURL:     "https://api.mistral.ai/v1",
		DocsURL:     "https://docs.mistral.ai/getting-started/models/",
		PricingURL:  "https://mistral.ai/pricing",
	},
	"openai": {
		ID:          "openai",
		DisplayName: "OpenAI",
		BaseURL:     "https://api.openai.com/v1",
		DocsURL:     "https://developers.openai.com/api/docs/models/all",
		PricingURL:  "https://developers.openai.com/api/docs/pricing",
	},
	"lmstudio": {
		ID:          "lmstudio",
		DisplayName: "LM Studio",
		BaseURL:     "http://localhost:1234/v1",
		DocsURL:     "https://lmstudio.ai/docs",
		PricingURL:  "", // local provider — no pricing page
	},
}

// GetProvider returns the ProviderSpec for the given provider ID.
func GetProvider(id string) (ProviderSpec, bool) {
	spec, ok := providerSpecs[id]
	return spec, ok
}

// AllProviders returns all registered provider specs, sorted by ID.
func AllProviders() []ProviderSpec {
	ids := KnownProviders()
	out := make([]ProviderSpec, 0, len(ids))
	for _, id := range ids {
		out = append(out, providerSpecs[id])
	}
	return out
}

// KnownProviders returns all registered provider IDs, sorted alphabetically.
func KnownProviders() []string {
	ids := make([]string, 0, len(providerSpecs))
	for id := range providerSpecs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// OfficialProviderModelDocsURL returns the official models documentation URL for a
// provider. Kept for backward compatibility with sync scripts.
func OfficialProviderModelDocsURL(provider string) (string, bool) {
	spec, ok := providerSpecs[provider]
	if !ok || spec.DocsURL == "" {
		return "", false
	}
	return spec.DocsURL, true
}

// OfficialProviderPricingURL returns the official pricing page URL for a provider.
// Returns false for local/self-hosted providers with no pricing page.
func OfficialProviderPricingURL(provider string) (string, bool) {
	spec, ok := providerSpecs[provider]
	if !ok || spec.PricingURL == "" {
		return "", false
	}
	return spec.PricingURL, true
}
