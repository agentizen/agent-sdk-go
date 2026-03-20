package main

import (
	"fmt"
	"sort"
	"strings"
)

func renderCapabilitiesFromSource(source registrySource) string {
	var b strings.Builder
	b.WriteString(generatedFileHeader)
	b.WriteString("package model\n\n")
	b.WriteString("import \"strings\"\n\n")
	b.WriteString("type Capability string\n\n")
	b.WriteString("const (\n")
	b.WriteString("\tCapabilityAudioGeneration  Capability = \"audioGeneration\"\n")
	b.WriteString("\tCapabilityBatchAPI          Capability = \"batchAPI\"\n")
	b.WriteString("\tCapabilityCaching           Capability = \"caching\"\n")
	b.WriteString("\tCapabilityCodeExecution     Capability = \"codeExecution\"\n")
	b.WriteString("\tCapabilityDocuments         Capability = \"documents\"\n")
	b.WriteString("\tCapabilityFileSearch        Capability = \"fileSearch\"\n")
	b.WriteString("\tCapabilityFunctionCalling   Capability = \"functionCalling\"\n")
	b.WriteString("\tCapabilityImageGeneration   Capability = \"imageGeneration\"\n")
	b.WriteString("\tCapabilityLiveAPI           Capability = \"liveAPI\"\n")
	b.WriteString("\tCapabilityStructuredOutput  Capability = \"structuredOutput\"\n")
	b.WriteString("\tCapabilityThinking          Capability = \"thinking\"\n")
	b.WriteString("\tCapabilityVision            Capability = \"vision\"\n")
	b.WriteString(")\n\n")
	b.WriteString("type ModelCapabilitySet struct {\n")
	b.WriteString("\tAudioGeneration  bool\n")
	b.WriteString("\tBatchAPI         bool\n")
	b.WriteString("\tCaching          bool\n")
	b.WriteString("\tCodeExecution    bool\n")
	b.WriteString("\tDocuments        bool\n")
	b.WriteString("\tFileSearch       bool\n")
	b.WriteString("\tFunctionCalling  bool\n")
	b.WriteString("\tImageGeneration  bool\n")
	b.WriteString("\tLiveAPI          bool\n")
	b.WriteString("\tStructuredOutput bool\n")
	b.WriteString("\tThinking         bool\n")
	b.WriteString("\tVision           bool\n")
	b.WriteString("}\n\n")
	b.WriteString("func CapabilitiesFor(provider, modelID string) ModelCapabilitySet {\n")
	b.WriteString("\treturn ModelCapabilitySet{\n")
	b.WriteString("\t\tAudioGeneration:  ProviderSupports(provider, modelID, CapabilityAudioGeneration),\n")
	b.WriteString("\t\tBatchAPI:         ProviderSupports(provider, modelID, CapabilityBatchAPI),\n")
	b.WriteString("\t\tCaching:          ProviderSupports(provider, modelID, CapabilityCaching),\n")
	b.WriteString("\t\tCodeExecution:    ProviderSupports(provider, modelID, CapabilityCodeExecution),\n")
	b.WriteString("\t\tDocuments:        ProviderSupports(provider, modelID, CapabilityDocuments),\n")
	b.WriteString("\t\tFileSearch:       ProviderSupports(provider, modelID, CapabilityFileSearch),\n")
	b.WriteString("\t\tFunctionCalling:  ProviderSupports(provider, modelID, CapabilityFunctionCalling),\n")
	b.WriteString("\t\tImageGeneration:  ProviderSupports(provider, modelID, CapabilityImageGeneration),\n")
	b.WriteString("\t\tLiveAPI:          ProviderSupports(provider, modelID, CapabilityLiveAPI),\n")
	b.WriteString("\t\tStructuredOutput: ProviderSupports(provider, modelID, CapabilityStructuredOutput),\n")
	b.WriteString("\t\tThinking:         ProviderSupports(provider, modelID, CapabilityThinking),\n")
	b.WriteString("\t\tVision:           ProviderSupports(provider, modelID, CapabilityVision),\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
	b.WriteString("type capabilityEntry struct {\n")
	b.WriteString("\tprefix string\n")
	b.WriteString("\tcaps map[Capability]bool\n")
	b.WriteString("}\n\n")
	b.WriteString("var providerCapabilities = map[string][]capabilityEntry{\n")

	for _, provider := range sortedProviders(source) {
		fmt.Fprintf(&b, "\t%q: {\n", provider.ID)
		for _, modelEntry := range provider.Models {
			caps := modelCapabilityClauses(modelEntry.Capabilities)
			if len(caps) == 0 {
				continue
			}
			fmt.Fprintf(&b, "\t\t{prefix: %q, caps: map[Capability]bool{%s}},\n",
				modelEntry.ID, strings.Join(caps, ", "))
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("func ProviderSupports(provider, modelID string, cap Capability) bool {\n")
	b.WriteString("\tentries, ok := providerCapabilities[strings.ToLower(provider)]\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn false\n")
	b.WriteString("\t}\n")
	b.WriteString("\tlowerModelID := strings.ToLower(modelID)\n")
	b.WriteString("\tfor _, entry := range entries {\n")
	b.WriteString("\t\tif strings.HasPrefix(lowerModelID, entry.prefix) {\n")
	b.WriteString("\t\t\treturn entry.caps[cap]\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn false\n")
	b.WriteString("}\n")
	return b.String()
}

func modelCapabilityClauses(c capabilitySource) []string {
	var caps []string
	if c.AudioGeneration {
		caps = append(caps, "CapabilityAudioGeneration: true")
	}
	if c.BatchAPI {
		caps = append(caps, "CapabilityBatchAPI: true")
	}
	if c.Caching {
		caps = append(caps, "CapabilityCaching: true")
	}
	if c.CodeExecution {
		caps = append(caps, "CapabilityCodeExecution: true")
	}
	if c.Documents {
		caps = append(caps, "CapabilityDocuments: true")
	}
	if c.FileSearch {
		caps = append(caps, "CapabilityFileSearch: true")
	}
	if c.FunctionCalling {
		caps = append(caps, "CapabilityFunctionCalling: true")
	}
	if c.ImageGeneration {
		caps = append(caps, "CapabilityImageGeneration: true")
	}
	if c.LiveAPI {
		caps = append(caps, "CapabilityLiveAPI: true")
	}
	if c.StructuredOutput {
		caps = append(caps, "CapabilityStructuredOutput: true")
	}
	if c.Thinking {
		caps = append(caps, "CapabilityThinking: true")
	}
	if c.Vision {
		caps = append(caps, "CapabilityVision: true")
	}
	return caps
}

func renderPricingFromSource(source registrySource) string {
	var b strings.Builder
	b.WriteString(generatedFileHeader)
	b.WriteString("package model\n\n")
	b.WriteString("import \"sort\"\n\n")
	b.WriteString("type ModelPricingSpec struct {\n")
	b.WriteString("\tInputCostPerMillion float64\n")
	b.WriteString("\tCachedInputCostPerMillion float64\n")
	b.WriteString("\tOutputCostPerMillion float64\n")
	b.WriteString("\tBatchInputCostPerMillion float64\n")
	b.WriteString("\tBatchCachedInputCostPerMillion float64\n")
	b.WriteString("\tBatchOutputCostPerMillion float64\n")
	b.WriteString("\tPriorityInputCostPerMillion float64\n")
	b.WriteString("\tPriorityCachedInputCostPerMillion float64\n")
	b.WriteString("\tPriorityOutputCostPerMillion float64\n")
	b.WriteString("\tLongContextTriggerAtTokens int\n")
	b.WriteString("\tLongContextInputCostPerMillion float64\n")
	b.WriteString("\tLongContextCachedInputCostPerMillion float64\n")
	b.WriteString("\tLongContextOutputCostPerMillion float64\n")
	b.WriteString("\tTrainingCostPerHour float64\n")
	b.WriteString("\tEstimatedCostPerMinute float64\n")
	b.WriteString("\tEstimatedCostPerSecond float64\n")
	b.WriteString("\tOcrInputCostPerThousandPages float64\n")
	b.WriteString("\tOcrOutputCostPerThousandPages float64\n")
	b.WriteString("}\n\n")
	b.WriteString("var modelPricing = map[string]map[string]ModelPricingSpec{\n")
	for _, provider := range sortedProviders(source) {
		fmt.Fprintf(&b, "\t%q: {\n", provider.ID)
		for _, modelEntry := range provider.Models {
			fmt.Fprintf(&b, "\t\t%q: {InputCostPerMillion: %s", modelEntry.ID, formatFloat(modelEntry.Pricing.InputCostPerMillion))
			if modelEntry.Pricing.CachedInputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", CachedInputCostPerMillion: %s", formatFloat(modelEntry.Pricing.CachedInputCostPerMillion))
			}
			fmt.Fprintf(&b, ", OutputCostPerMillion: %s", formatFloat(modelEntry.Pricing.OutputCostPerMillion))
			if modelEntry.Pricing.BatchInputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", BatchInputCostPerMillion: %s", formatFloat(modelEntry.Pricing.BatchInputCostPerMillion))
			}
			if modelEntry.Pricing.BatchCachedInputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", BatchCachedInputCostPerMillion: %s", formatFloat(modelEntry.Pricing.BatchCachedInputCostPerMillion))
			}
			if modelEntry.Pricing.BatchOutputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", BatchOutputCostPerMillion: %s", formatFloat(modelEntry.Pricing.BatchOutputCostPerMillion))
			}
			if modelEntry.Pricing.PriorityInputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", PriorityInputCostPerMillion: %s", formatFloat(modelEntry.Pricing.PriorityInputCostPerMillion))
			}
			if modelEntry.Pricing.PriorityCachedInputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", PriorityCachedInputCostPerMillion: %s", formatFloat(modelEntry.Pricing.PriorityCachedInputCostPerMillion))
			}
			if modelEntry.Pricing.PriorityOutputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", PriorityOutputCostPerMillion: %s", formatFloat(modelEntry.Pricing.PriorityOutputCostPerMillion))
			}
			if modelEntry.Pricing.LongContextTriggerAtTokens > 0 {
				fmt.Fprintf(&b, ", LongContextTriggerAtTokens: %d", modelEntry.Pricing.LongContextTriggerAtTokens)
			}
			if modelEntry.Pricing.LongContextInputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", LongContextInputCostPerMillion: %s", formatFloat(modelEntry.Pricing.LongContextInputCostPerMillion))
			}
			if modelEntry.Pricing.LongContextCachedInputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", LongContextCachedInputCostPerMillion: %s", formatFloat(modelEntry.Pricing.LongContextCachedInputCostPerMillion))
			}
			if modelEntry.Pricing.LongContextOutputCostPerMillion > 0 {
				fmt.Fprintf(&b, ", LongContextOutputCostPerMillion: %s", formatFloat(modelEntry.Pricing.LongContextOutputCostPerMillion))
			}
			if modelEntry.Pricing.TrainingCostPerHour > 0 {
				fmt.Fprintf(&b, ", TrainingCostPerHour: %s", formatFloat(modelEntry.Pricing.TrainingCostPerHour))
			}
			if modelEntry.Pricing.EstimatedCostPerMinute > 0 {
				fmt.Fprintf(&b, ", EstimatedCostPerMinute: %s", formatFloat(modelEntry.Pricing.EstimatedCostPerMinute))
			}
			if modelEntry.Pricing.EstimatedCostPerSecond > 0 {
				fmt.Fprintf(&b, ", EstimatedCostPerSecond: %s", formatFloat(modelEntry.Pricing.EstimatedCostPerSecond))
			}
			if modelEntry.Pricing.OcrInputCostPerThousandPages > 0 {
				fmt.Fprintf(&b, ", OcrInputCostPerThousandPages: %s", formatFloat(modelEntry.Pricing.OcrInputCostPerThousandPages))
			}
			if modelEntry.Pricing.OcrOutputCostPerThousandPages > 0 {
				fmt.Fprintf(&b, ", OcrOutputCostPerThousandPages: %s", formatFloat(modelEntry.Pricing.OcrOutputCostPerThousandPages))
			}
			b.WriteString("},\n")
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("func GetModelPricing(provider, modelID string) (ModelPricingSpec, bool) {\n")
	b.WriteString("\tmodels, ok := modelPricing[provider]\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn ModelPricingSpec{}, false\n")
	b.WriteString("\t}\n")
	b.WriteString("\tspec, ok := models[modelID]\n")
	b.WriteString("\treturn spec, ok\n")
	b.WriteString("}\n\n")
	b.WriteString("func KnownPricingProviders() []string {\n")
	b.WriteString("\tproviders := make([]string, 0, len(modelPricing))\n")
	b.WriteString("\tfor p := range modelPricing {\n")
	b.WriteString("\t\tproviders = append(providers, p)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tsort.Strings(providers)\n")
	b.WriteString("\treturn providers\n")
	b.WriteString("}\n")
	return b.String()
}

func renderMetadataFromSource(source registrySource) string {
	var b strings.Builder
	b.WriteString(generatedFileHeader)
	b.WriteString("package model\n\n")
	b.WriteString("import \"sort\"\n\n")
	b.WriteString("type ModelMetadata struct {\n")
	b.WriteString("\tDisplayName string\n")
	b.WriteString("\tDescription string\n")
	b.WriteString("\tReleaseDate string\n")
	b.WriteString("\tContextWindow int\n")
	b.WriteString("\tMaxOutputTokens int\n")
	b.WriteString("}\n\n")
	b.WriteString("func GetModelMetadata(provider, modelID string) (ModelMetadata, bool) {\n")
	b.WriteString("\tmodels, ok := modelMetadata[provider]\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn ModelMetadata{}, false\n")
	b.WriteString("\t}\n")
	b.WriteString("\tmeta, ok := models[modelID]\n")
	b.WriteString("\treturn meta, ok\n")
	b.WriteString("}\n\n")
	b.WriteString("func KnownMetadataProviders() []string {\n")
	b.WriteString("\tproviders := make([]string, 0, len(modelMetadata))\n")
	b.WriteString("\tfor p := range modelMetadata {\n")
	b.WriteString("\t\tproviders = append(providers, p)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tsort.Strings(providers)\n")
	b.WriteString("\treturn providers\n")
	b.WriteString("}\n\n")
	b.WriteString("var modelMetadata = map[string]map[string]ModelMetadata{\n")
	for _, provider := range sortedProviders(source) {
		fmt.Fprintf(&b, "\t%q: {\n", provider.ID)
		for _, modelEntry := range provider.Models {
			fmt.Fprintf(&b, "\t\t%q: {\n", modelEntry.ID)
			fmt.Fprintf(&b, "\t\t\tDisplayName: %q,\n", modelEntry.DisplayName)
			fmt.Fprintf(&b, "\t\t\tDescription: %q,\n", modelEntry.Description)
			fmt.Fprintf(&b, "\t\t\tReleaseDate: %q,\n", modelEntry.ReleaseDate)
			fmt.Fprintf(&b, "\t\t\tContextWindow: %d,\n", modelEntry.ContextWindow)
			fmt.Fprintf(&b, "\t\t\tMaxOutputTokens: %d,\n", modelEntry.MaxOutputTokens)
			b.WriteString("\t\t},\n")
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func renderProvidersFromSource(source registrySource) string {
	providers := sortedProviders(source)
	var b strings.Builder
	b.WriteString(generatedFileHeader)
	b.WriteString("package model\n\n")
	b.WriteString("import \"sort\"\n\n")
	b.WriteString("type ProviderSpec struct {\n")
	b.WriteString("\tID string\n")
	b.WriteString("\tDisplayName string\n")
	b.WriteString("\tBaseURL string\n")
	b.WriteString("\tDocsURL string\n")
	b.WriteString("\tPricingURL string\n")
	b.WriteString("}\n\n")
	b.WriteString("func GetProvider(id string) (ProviderSpec, bool) {\n")
	b.WriteString("\tspec, ok := providerSpecs[id]\n")
	b.WriteString("\treturn spec, ok\n")
	b.WriteString("}\n\n")
	b.WriteString("func AllProviders() []ProviderSpec {\n")
	b.WriteString("\tids := KnownProviders()\n")
	b.WriteString("\tout := make([]ProviderSpec, 0, len(ids))\n")
	b.WriteString("\tfor _, id := range ids {\n")
	b.WriteString("\t\tout = append(out, providerSpecs[id])\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn out\n")
	b.WriteString("}\n\n")
	b.WriteString("func KnownProviders() []string {\n")
	b.WriteString("\tids := make([]string, 0, len(providerSpecs))\n")
	b.WriteString("\tfor id := range providerSpecs {\n")
	b.WriteString("\t\tids = append(ids, id)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tsort.Strings(ids)\n")
	b.WriteString("\treturn ids\n")
	b.WriteString("}\n\n")
	b.WriteString("func OfficialProviderModelDocsURL(provider string) (string, bool) {\n")
	b.WriteString("\tspec, ok := providerSpecs[provider]\n")
	b.WriteString("\tif !ok || spec.DocsURL == \"\" {\n")
	b.WriteString("\t\treturn \"\", false\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn spec.DocsURL, true\n")
	b.WriteString("}\n\n")
	b.WriteString("func OfficialProviderPricingURL(provider string) (string, bool) {\n")
	b.WriteString("\tspec, ok := providerSpecs[provider]\n")
	b.WriteString("\tif !ok || spec.PricingURL == \"\" {\n")
	b.WriteString("\t\treturn \"\", false\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn spec.PricingURL, true\n")
	b.WriteString("}\n\n")
	b.WriteString("var providerSpecs = map[string]ProviderSpec{\n")
	for _, provider := range providers {
		fmt.Fprintf(&b, "\t%q: {\n", provider.ID)
		fmt.Fprintf(&b, "\t\tID: %q,\n", provider.ID)
		fmt.Fprintf(&b, "\t\tDisplayName: %q,\n", provider.DisplayName)
		fmt.Fprintf(&b, "\t\tBaseURL: %q,\n", provider.BaseURL)
		fmt.Fprintf(&b, "\t\tDocsURL: %q,\n", provider.DocsURL)
		fmt.Fprintf(&b, "\t\tPricingURL: %q,\n", provider.PricingURL)
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func reportSourceSummary(source registrySource, verbose bool) {
	providers := sortedProviders(source)
	for _, provider := range providers {
		fmt.Printf("source provider=%s models=%d\n", provider.ID, len(provider.Models))
		if !verbose {
			continue
		}
		for _, modelEntry := range provider.Models {
			fmt.Printf("  - %s\n", modelEntry.ID)
		}
	}
}

func sourceModelCount(source registrySource) int {
	total := 0
	for _, provider := range source.Providers {
		total += len(provider.Models)
	}
	return total
}

func sortedModelIDs(models []modelSource) []string {
	ids := make([]string, 0, len(models))
	for _, modelEntry := range models {
		ids = append(ids, modelEntry.ID)
	}
	sort.Strings(ids)
	return ids
}
