package model

import "sort"

// ModelCapabilitySpec represents one provider/model-prefix capability definition
// with source metadata from the SDK registry.
type ModelCapabilitySpec struct {
	Provider        string
	Prefix          string
	Capabilities    map[Capability]bool
	OfficialDocsURL string
	Source          string
	Active          bool
}

// RegistrySpecs returns a deterministic flat list of all model capability specs
// known by the SDK registry.
func RegistrySpecs() []ModelCapabilitySpec {
	providers := KnownCapabilityProviders()
	out := make([]ModelCapabilitySpec, 0)
	for _, provider := range providers {
		docsURL, _ := OfficialProviderModelDocsURL(provider)
		entries := providerCapabilities[provider]
		for _, entry := range entries {
			out = append(out, ModelCapabilitySpec{
				Provider:        provider,
				Prefix:          entry.prefix,
				Capabilities:    cloneCapabilities(entry.caps),
				OfficialDocsURL: docsURL,
				Source:          "official-docs",
				Active:          true,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider == out[j].Provider {
			return out[i].Prefix < out[j].Prefix
		}
		return out[i].Provider < out[j].Provider
	})

	return out
}

func cloneCapabilities(in map[Capability]bool) map[Capability]bool {
	out := make(map[Capability]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
