package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/agentizen/agent-sdk-go/pkg/model"
)

type registrySource struct {
	Version   int              `json:"version"`
	Providers []providerSource `json:"providers"`
}

type providerSource struct {
	ID          string        `json:"id"`
	DisplayName string        `json:"display_name"`
	BaseURL     string        `json:"base_url"`
	DocsURL     string        `json:"docs_url"`
	PricingURL  string        `json:"pricing_url"`
	Models      []modelSource `json:"models"`
}

type modelSource struct {
	ID              string                 `json:"id"`
	DisplayName     string                 `json:"display_name"`
	Description     string                 `json:"description"`
	ReleaseDate     string                 `json:"release_date"`
	ContextWindow   int                    `json:"context_window"`
	MaxOutputTokens int                    `json:"max_output_tokens"`
	Capabilities    capabilitySource       `json:"capabilities"`
	Pricing         model.ModelPricingSpec `json:"pricing"`
}

type capabilitySource struct {
	AudioGeneration  bool `json:"audioGeneration"`
	BatchAPI         bool `json:"batchAPI"`
	Caching          bool `json:"caching"`
	CodeExecution    bool `json:"codeExecution"`
	Documents        bool `json:"documents"`
	FileSearch       bool `json:"fileSearch"`
	FunctionCalling  bool `json:"functionCalling"`
	ImageGeneration  bool `json:"imageGeneration"`
	LiveAPI          bool `json:"liveAPI"`
	StructuredOutput bool `json:"structuredOutput"`
	Thinking         bool `json:"thinking"`
	Vision           bool `json:"vision"`
}

var canonicalIDRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)

type registrySchemaMeta struct {
	Schema  string `json:"$schema"`
	Type    string `json:"type"`
	Version int    `json:"x-source-version"`
}

func loadRegistrySchemaVersion(path string) (int, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read registry schema: %w", err)
	}

	var schema registrySchemaMeta
	if err := json.Unmarshal(payload, &schema); err != nil {
		return 0, fmt.Errorf("parse registry schema JSON: %w", err)
	}
	if strings.TrimSpace(schema.Schema) == "" {
		return 0, fmt.Errorf("registry schema missing $schema")
	}
	if schema.Type != "object" {
		return 0, fmt.Errorf("registry schema root type must be object")
	}
	if schema.Version <= 0 {
		return 0, fmt.Errorf("registry schema x-source-version must be > 0")
	}

	return schema.Version, nil
}

func loadRegistrySource(path string) (registrySource, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return registrySource{}, fmt.Errorf("read registry source: %w", err)
	}

	var source registrySource
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&source); err != nil {
		return registrySource{}, fmt.Errorf("parse registry source JSON: %w", err)
	}
	if dec.More() {
		return registrySource{}, fmt.Errorf("registry source JSON contains trailing values")
	}
	if err := validateRegistrySource(source); err != nil {
		return registrySource{}, err
	}
	return source, nil
}

func validateRegistrySource(source registrySource) error {
	if source.Version <= 0 {
		return fmt.Errorf("registry source version must be > 0")
	}
	if len(source.Providers) == 0 {
		return fmt.Errorf("registry source must contain at least one provider")
	}

	providerIDs := map[string]struct{}{}
	for _, provider := range source.Providers {
		providerID := strings.TrimSpace(strings.ToLower(provider.ID))
		if providerID == "" {
			return fmt.Errorf("provider id is required")
		}
		if providerID != provider.ID {
			return fmt.Errorf("provider id must be lowercase canonical form: %s", provider.ID)
		}
		if !canonicalIDRegexp.MatchString(providerID) {
			return fmt.Errorf("provider id has invalid format: %s", providerID)
		}
		if _, exists := providerIDs[providerID]; exists {
			return fmt.Errorf("duplicate provider id: %s", providerID)
		}
		providerIDs[providerID] = struct{}{}
		if strings.TrimSpace(provider.DisplayName) == "" {
			return fmt.Errorf("provider %s display_name is required", providerID)
		}
		if strings.TrimSpace(provider.BaseURL) == "" {
			return fmt.Errorf("provider %s base_url is required", providerID)
		}

		modelIDs := map[string]struct{}{}
		for _, modelEntry := range provider.Models {
			modelID := strings.TrimSpace(strings.ToLower(modelEntry.ID))
			if modelID == "" {
				return fmt.Errorf("provider %s contains model with empty id", providerID)
			}
			if modelID != modelEntry.ID {
				return fmt.Errorf("provider %s model id must be lowercase canonical form: %s", providerID, modelEntry.ID)
			}
			if !canonicalIDRegexp.MatchString(modelID) {
				return fmt.Errorf("provider %s model id has invalid format: %s", providerID, modelID)
			}
			if _, exists := modelIDs[modelID]; exists {
				return fmt.Errorf("provider %s has duplicate model id: %s", providerID, modelID)
			}
			modelIDs[modelID] = struct{}{}
			if strings.TrimSpace(modelEntry.DisplayName) == "" {
				return fmt.Errorf("provider %s model %s display_name is required", providerID, modelID)
			}
			if strings.TrimSpace(modelEntry.Description) == "" {
				return fmt.Errorf("provider %s model %s description is required", providerID, modelID)
			}
			if modelEntry.ContextWindow <= 0 {
				return fmt.Errorf("provider %s model %s context_window must be > 0", providerID, modelID)
			}
			if modelEntry.Pricing.InputCostPerMillion < 0 || modelEntry.Pricing.OutputCostPerMillion < 0 {
				return fmt.Errorf("provider %s model %s pricing fields must be >= 0", providerID, modelID)
			}
		}
	}

	return nil
}

func sortedProviders(source registrySource) []providerSource {
	providers := make([]providerSource, 0, len(source.Providers))
	providers = append(providers, source.Providers...)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
	for i := range providers {
		sort.Slice(providers[i].Models, func(a, b int) bool {
			return providers[i].Models[a].ID < providers[i].Models[b].ID
		})
	}
	return providers
}
