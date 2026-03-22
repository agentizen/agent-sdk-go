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
		{"mistral", "mistral-small-2603", model.CapabilityVision, true},
		{"mistral", "magistral-medium-2509", model.CapabilityVision, true},
		{"mistral", "magistral-medium-2509", model.CapabilityThinking, true},
		{"mistral", "magistral-small-2509", model.CapabilityVision, true},
		{"mistral", "magistral-small-2509", model.CapabilityThinking, true},
		{"mistral", "ministral-8b-2512", model.CapabilityVision, true},
		{"mistral", "mistral-ocr-2512", model.CapabilityDocuments, true},
		{"mistral", "mistral-ocr-2512", model.CapabilityOCR, true},
		{"mistral", "mistral-ocr-2512", model.CapabilityVision, false},
		{"mistral", "codestral-25-08", model.CapabilityVision, false},

		// OpenAI
		{"openai", "gpt-5.4-2026-03-05", model.CapabilityVision, true},
		{"openai", "gpt-5.4-pro-2026-03-05", model.CapabilityVision, true},
		{"openai", "gpt-5.4-mini-2026-03-17", model.CapabilityVision, true},
		{"openai", "gpt-5.4-nano-2026-03-17", model.CapabilityVision, true},
		{"openai", "gpt-5.4-2026-03-05", model.CapabilityDocuments, false},
		{"openai", "gpt-3.5-turbo", model.CapabilityVision, false},

		// Anthropic
		{"anthropic", "claude-opus-4-6", model.CapabilityVision, true},
		{"anthropic", "claude-sonnet-4-6", model.CapabilityVision, true},
		{"anthropic", "claude-haiku-4-5-20251001", model.CapabilityVision, true},
		{"anthropic", "claude-opus-4-6", model.CapabilityDocuments, true},
		{"anthropic", "claude-2.1", model.CapabilityVision, false},

		// Gemini — vision/documents not in current registry; new capabilities apply
		{"gemini", "gemini-2.5-pro", model.CapabilityThinking, true},
		{"gemini", "gemini-2.5-flash", model.CapabilityCodeExecution, true},
		{"gemini", "gemini-2.5-flash-lite", model.CapabilityThinking, true},
		{"gemini", "gemini-2.5-pro", model.CapabilityVision, false},
		{"gemini", "gemini-2.5-flash-lite", model.CapabilityDocuments, false},
		{"gemini", "gemini-1.0-pro", model.CapabilityVision, false},

		// Unknown provider/model
		{"lmstudio", "mistral-7b", model.CapabilityVision, false},
		{"unknown", "gpt-5.4", model.CapabilityVision, false},

		// Case-insensitivity
		{"Mistral", "MISTRAL-LARGE-2512", model.CapabilityVision, true},
		{"OPENAI", "GPT-5.4-2026-03-05", model.CapabilityVision, true},
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
