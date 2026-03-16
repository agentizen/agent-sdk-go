package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestLoadRegistrySchemaVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	schemaPath := writeTempFile(t, dir, "schema.json", `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "x-source-version": 3
}`)

	version, err := loadRegistrySchemaVersion(schemaPath)
	if err != nil {
		t.Fatalf("loadRegistrySchemaVersion returned error: %v", err)
	}
	if version != 3 {
		t.Fatalf("schema version = %d, want 3", version)
	}
}

func TestLoadRegistrySourceDisallowUnknownFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sourcePath := writeTempFile(t, dir, "source.json", `{
  "version": 1,
  "providers": [
    {
      "id": "openai",
      "display_name": "OpenAI",
      "base_url": "https://api.openai.com",
      "docs_url": "https://developers.openai.com/api/docs/models",
      "pricing_url": "https://developers.openai.com/api/docs/pricing",
      "models": [
        {
          "id": "gpt-4o",
          "display_name": "GPT-4o",
          "description": "General-purpose model",
          "release_date": "",
          "context_window": 128000,
          "max_output_tokens": 16384,
          "capabilities": {
            "vision": true,
            "documents": true
          },
          "pricing": {
            "InputCostPerMillion": 2.5,
            "CachedInputCostPerMillion": 0,
            "OutputCostPerMillion": 10,
            "BatchInputCostPerMillion": 0,
            "BatchCachedInputCostPerMillion": 0,
            "BatchOutputCostPerMillion": 0,
            "PriorityInputCostPerMillion": 0,
            "PriorityCachedInputCostPerMillion": 0,
            "PriorityOutputCostPerMillion": 0,
            "LongContextTriggerAtTokens": 0,
            "LongContextInputCostPerMillion": 0,
            "LongContextCachedInputCostPerMillion": 0,
            "LongContextOutputCostPerMillion": 0,
            "TrainingCostPerHour": 0,
            "EstimatedCostPerMinute": 0,
            "EstimatedCostPerSecond": 0
          },
          "unexpected": true
        }
      ]
    }
  ]
}`)

	_, err := loadRegistrySource(sourcePath)
	if err == nil {
		t.Fatal("expected error for unknown JSON field, got nil")
	}
}

func TestValidateRegistrySourceRejectsUppercaseProviderID(t *testing.T) {
	t.Parallel()
	source := registrySource{
		Version: 1,
		Providers: []providerSource{
			{
				ID:          "OpenAI",
				DisplayName: "OpenAI",
				BaseURL:     "https://api.openai.com",
				Models:      []modelSource{},
			},
		},
	}

	err := validateRegistrySource(source)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestValidateRegistrySourceRejectsZeroContextWindow(t *testing.T) {
	t.Parallel()
	source := registrySource{
		Version: 1,
		Providers: []providerSource{
			{
				ID:          "openai",
				DisplayName: "OpenAI",
				BaseURL:     "https://api.openai.com",
				Models: []modelSource{
					{
						ID:              "gpt-4o",
						DisplayName:     "GPT-4o",
						Description:     "General-purpose model",
						ContextWindow:   0,
						MaxOutputTokens: 1,
					},
				},
			},
		},
	}

	err := validateRegistrySource(source)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
