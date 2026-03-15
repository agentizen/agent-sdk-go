package model

import "sort"

// ModelMetadata holds non-pricing, non-capability descriptive information for a model.
// All fields use exact model ID matching (no prefix matching).
type ModelMetadata struct {
	// DisplayName is the human-readable model name (e.g. "Claude Haiku 4.5").
	DisplayName string

	// Description describes the model's primary use case and any notable characteristics.
	Description string

	// ReleaseDate is the month the model was publicly released, formatted YYYY-MM.
	ReleaseDate string

	// ContextWindow is the maximum number of tokens the model can process in its
	// sliding context (input prompt + conversation history).
	ContextWindow int

	// MaxOutputTokens is the maximum number of tokens the model can generate in a
	// single response.
	MaxOutputTokens int
}

// modelMetadata maps provider → exact modelID → ModelMetadata.
var modelMetadata = map[string]map[string]ModelMetadata{
	"anthropic": {
		"claude-haiku-4-5": {
			DisplayName:     "Claude Haiku 4.5",
			Description:     "Fastest model with near-frontier intelligence for quick responses.",
			ReleaseDate:     "2025-10",
			ContextWindow:   200000,
			MaxOutputTokens: 64000,
		},
		"claude-sonnet-4-6": {
			DisplayName:     "Claude Sonnet 4.6",
			Description:     "Smartest model for complex agents and heavy coding tasks.",
			ReleaseDate:     "2026-02",
			ContextWindow:   200000,
			MaxOutputTokens: 64000,
		},
		"claude-opus-4-6": {
			DisplayName:     "Claude Opus 4.6",
			Description:     "Most powerful model for highly complex tasks and frontier intelligence.",
			ReleaseDate:     "2026-01",
			ContextWindow:   200000,
			MaxOutputTokens: 128000,
		},
	},
	"gemini": {
		"gemini-2.5-flash-lite": {
			DisplayName:     "Gemini 2.5 Flash Lite",
			Description:     "Smallest and most cost effective model, built for high-volume usage.",
			ReleaseDate:     "2025-07",
			ContextWindow:   1048576,
			MaxOutputTokens: 65536,
		},
		"gemini-2.5-flash": {
			DisplayName:     "Gemini 2.5 Flash",
			Description:     "Hybrid reasoning model with 1M token context window and thinking budgets.",
			ReleaseDate:     "2025-06",
			ContextWindow:   1048576,
			MaxOutputTokens: 65536,
		},
		"gemini-2.5-pro": {
			DisplayName:     "Gemini 2.5 Pro",
			Description:     "State-of-the-art multipurpose model, excelling at coding and complex reasoning.",
			ReleaseDate:     "2025-06",
			ContextWindow:   1048576,
			MaxOutputTokens: 65536,
		},
	},
	"mistral": {
		"ministral-8b-2512": {
			DisplayName:     "Ministral 3 8B",
			Description:     "Powerful edge model with extremely high performance/price ratio.",
			ReleaseDate:     "2025-12",
			ContextWindow:   256000,
			MaxOutputTokens: 64000,
		},
		"mistral-small-2506": {
			DisplayName:     "Mistral Small 3.2",
			Description:     "Efficient model ideal for low-latency applications and simple tasks.",
			ReleaseDate:     "2025-06",
			ContextWindow:   128000,
			MaxOutputTokens: 64000,
		},
		"mistral-medium-2508": {
			DisplayName:     "Mistral Medium 3.1",
			Description:     "Frontier-class multimodal model for general-purpose tasks.",
			ReleaseDate:     "2025-08",
			ContextWindow:   128000,
			MaxOutputTokens: 64000,
		},
		"mistral-large-2512": {
			DisplayName:     "Mistral Large 3",
			Description:     "State-of-the-art multimodal MoE model for complex reasoning and general-purpose tasks.",
			ReleaseDate:     "2025-12",
			ContextWindow:   256000,
			MaxOutputTokens: 64000,
		},
		"magistral-small-2509": {
			DisplayName:     "Magistral Small 1.2",
			Description:     "Compact multimodal reasoning model for analyzing text and images.",
			ReleaseDate:     "2025-09",
			ContextWindow:   128000,
			MaxOutputTokens: 64000,
		},
		"magistral-medium-2509": {
			DisplayName:     "Magistral Medium 1.2",
			Description:     "High-performance reasoning model for complex multimodal challenges.",
			ReleaseDate:     "2025-09",
			ContextWindow:   128000,
			MaxOutputTokens: 64000,
		},
	},
	"openai": {
		"gpt-5-nano-2025-08-07": {
			DisplayName:     "GPT-5 Nano",
			Description:     "Fastest, most cost-efficient GPT-5 version. Great for summarization and classification.",
			ReleaseDate:     "2025-08",
			ContextWindow:   400000,
			MaxOutputTokens: 128000,
		},
		"gpt-5-mini-2025-08-07": {
			DisplayName:     "GPT-5 Mini",
			Description:     "Cost-efficient GPT-5 version optimized for well-defined tasks and precise prompts.",
			ReleaseDate:     "2025-08",
			ContextWindow:   400000,
			MaxOutputTokens: 128000,
		},
		"gpt-5.2-2025-12-11": {
			DisplayName:     "GPT-5.2",
			Description:     "Flagship model for coding and agentic tasks across industries.",
			ReleaseDate:     "2025-12",
			ContextWindow:   400000,
			MaxOutputTokens: 128000,
		},
	},
}

// GetModelMetadata returns the metadata for an exact provider/modelID combination.
func GetModelMetadata(provider, modelID string) (ModelMetadata, bool) {
	models, ok := modelMetadata[provider]
	if !ok {
		return ModelMetadata{}, false
	}
	meta, ok := models[modelID]
	return meta, ok
}

// KnownMetadataProviders returns providers that have metadata entries, sorted alphabetically.
func KnownMetadataProviders() []string {
	providers := make([]string, 0, len(modelMetadata))
	for p := range modelMetadata {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}
