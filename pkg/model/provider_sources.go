package model

import "sort"

var providerModelDocsURLs = map[string]string{
	"anthropic": "https://platform.claude.com/docs/en/docs/about-claude/models",
	"gemini":    "https://ai.google.dev/gemini-api/docs/models",
	"mistral":   "https://docs.mistral.ai/getting-started/models/",
	"openai":    "https://developers.openai.com/api/docs/models",
}

// OfficialProviderModelDocsURL returns the official models documentation page URL
// for a provider, when known by the SDK registry.
func OfficialProviderModelDocsURL(provider string) (string, bool) {
	url, ok := providerModelDocsURLs[provider]
	return url, ok
}

// KnownCapabilityProviders returns providers that have capability/source metadata
// registered in the SDK, sorted alphabetically for deterministic consumers.
func KnownCapabilityProviders() []string {
	providers := make([]string, 0, len(providerCapabilities))
	for provider := range providerCapabilities {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}
