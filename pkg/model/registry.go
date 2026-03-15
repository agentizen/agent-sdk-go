package model

import "sort"

// ModelSpec is the complete, unified specification for a model registered in the SDK.
// It aggregates metadata, pricing, and capabilities from their respective sources.
type ModelSpec struct {
	// Provider is the canonical provider ID (e.g. "anthropic", "openai").
	Provider string

	// ModelID is the exact model identifier as used in API calls (e.g. "claude-sonnet-4-6").
	ModelID string

	// Metadata holds descriptive, non-pricing information about the model.
	Metadata ModelMetadata

	// Pricing holds the cost rates for this model.
	Pricing ModelPricingSpec

	// Capabilities holds the resolved feature flags for this model.
	Capabilities ModelCapabilitySet
}

// GetModelSpec returns the complete spec for an exact provider/modelID combination.
// Metadata and Pricing require an exact ID match; Capabilities use prefix matching.
// Returns false only when no metadata is registered for the given provider/modelID.
func GetModelSpec(provider, modelID string) (ModelSpec, bool) {
	meta, ok := GetModelMetadata(provider, modelID)
	if !ok {
		return ModelSpec{}, false
	}
	pricing, _ := GetModelPricing(provider, modelID)
	return ModelSpec{
		Provider:     provider,
		ModelID:      modelID,
		Metadata:     meta,
		Pricing:      pricing,
		Capabilities: CapabilitiesFor(provider, modelID),
	}, true
}

// AllModelSpecs returns a deterministic flat list of all registered model specs,
// sorted by provider then model ID.
func AllModelSpecs() []ModelSpec {
	providers := KnownMetadataProviders()
	out := make([]ModelSpec, 0)
	for _, provider := range providers {
		models := modelMetadata[provider]
		modelIDs := make([]string, 0, len(models))
		for id := range models {
			modelIDs = append(modelIDs, id)
		}
		sort.Strings(modelIDs)
		for _, modelID := range modelIDs {
			spec, _ := GetModelSpec(provider, modelID)
			out = append(out, spec)
		}
	}
	return out
}

// ModelSpecsForProvider returns all registered model specs for a given provider,
// sorted by model ID.
func ModelSpecsForProvider(provider string) []ModelSpec {
	models := modelMetadata[provider]
	if len(models) == 0 {
		return nil
	}
	modelIDs := make([]string, 0, len(models))
	for id := range models {
		modelIDs = append(modelIDs, id)
	}
	sort.Strings(modelIDs)
	out := make([]ModelSpec, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		spec, _ := GetModelSpec(provider, modelID)
		out = append(out, spec)
	}
	return out
}
