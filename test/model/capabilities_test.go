package model_test

import (
	"testing"

	"github.com/citizenofai/agent-sdk-go/pkg/model"
)

func TestProviderSupports(t *testing.T) {
	tests := []struct {
		provider string
		modelID  string
		cap      model.Capability
		want     bool
	}{
		// Mistral
		{"mistral", "mistral-large-2512", model.CapabilityVision, true},
		{"mistral", "mistral-medium-2508", model.CapabilityVision, true},
		{"mistral", "mistral-small-2506", model.CapabilityVision, true},
		{"mistral", "magistral-medium-2509", model.CapabilityVision, true},
		{"mistral", "ministral-8b-2512", model.CapabilityVision, true},
		{"mistral", "mistral-ocr-2512", model.CapabilityDocuments, true},
		{"mistral", "ocr-3-25-12", model.CapabilityDocuments, true},
		{"mistral", "mistral-ocr-2512", model.CapabilityVision, false},
		{"mistral", "codestral-25-08", model.CapabilityVision, false},

		// OpenAI
		{"openai", "gpt-5.4", model.CapabilityVision, true},
		{"openai", "gpt-5-mini-2025-08-07", model.CapabilityVision, true},
		{"openai", "gpt-5-nano-2025-08-07", model.CapabilityVision, true},
		{"openai", "gpt-5.2-2025-12-11", model.CapabilityVision, true},
		{"openai", "gpt-5.2-2025-12-11", model.CapabilityDocuments, true},
		{"openai", "gpt-4o", model.CapabilityVision, true},
		{"openai", "gpt-4.1", model.CapabilityVision, true},
		{"openai", "o3-mini", model.CapabilityVision, true},
		{"openai", "gpt-3.5-turbo", model.CapabilityVision, false},

		// Anthropic
		{"anthropic", "claude-opus-4-6", model.CapabilityVision, true},
		{"anthropic", "claude-sonnet-4-6", model.CapabilityVision, true},
		{"anthropic", "claude-haiku-4-5", model.CapabilityVision, true},
		{"anthropic", "claude-opus-4-6", model.CapabilityDocuments, true},
		{"anthropic", "claude-3-7-sonnet-latest", model.CapabilityVision, true},
		{"anthropic", "claude-2.1", model.CapabilityVision, false},

		// Gemini
		{"gemini", "gemini-3.1-pro-preview", model.CapabilityVision, true},
		{"gemini", "gemini-3-flash-preview", model.CapabilityVision, true},
		{"gemini", "gemini-2.5-pro", model.CapabilityVision, true},
		{"gemini", "gemini-2.5-flash-lite", model.CapabilityDocuments, true},
		{"gemini", "gemini-flash-latest", model.CapabilityVision, true},
		{"gemini", "gemini-pro-latest", model.CapabilityDocuments, true},
		{"gemini", "gemini-1.5-pro", model.CapabilityVision, true},
		{"gemini", "gemini-1.0-pro", model.CapabilityVision, false},

		// Unknown provider/model
		{"lmstudio", "mistral-7b", model.CapabilityVision, false},
		{"unknown", "gpt-5.4", model.CapabilityVision, false},

		// Case-insensitivity
		{"Mistral", "MISTRAL-LARGE-2512", model.CapabilityVision, true},
		{"OPENAI", "GPT-5.4", model.CapabilityVision, true},
	}

	for _, tc := range tests {
		got := model.ProviderSupports(tc.provider, tc.modelID, tc.cap)
		if got != tc.want {
			t.Errorf(
				"ProviderSupports(%q, %q, %q) = %v, want %v",
				tc.provider,
				tc.modelID,
				tc.cap,
				got,
				tc.want,
			)
		}
	}
}
