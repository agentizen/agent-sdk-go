package agentsdk

import (
	"context"
	"testing"
)

type testTracer struct {
	events []Event
}

func (t *testTracer) RecordEvent(_ context.Context, event Event) {
	t.events = append(t.events, event)
}

func (t *testTracer) Flush() error { return nil }

func (t *testTracer) Close() error { return nil }

func TestPublicAPIExposesServerFacingHelpers(t *testing.T) {
	if _, ok := GetModelMetadata("openai", "gpt-5.4-mini-2026-03-17"); !ok {
		t.Fatal("expected OpenAI model metadata from public API")
	}
	if !ProviderSupports("openai", "gpt-5.4-mini-2026-03-17", CapabilityVision) {
		t.Fatal("expected vision capability from public API")
	}

	ctx := WithUserID(context.Background(), "user-42")
	if got := UserIDFromContext(ctx); got != "user-42" {
		t.Fatalf("expected user-42, got %q", got)
	}

	cfg := NewNetworkConfig().WithStrategy(StrategyParallel)
	if cfg.Strategy != StrategyParallel {
		t.Fatalf("expected strategy %q, got %q", StrategyParallel, cfg.Strategy)
	}

	if NewOpenAIProvider("test") == nil {
		t.Fatal("expected OpenAI provider constructor from public API")
	}
	if NewAnthropicProvider("test") == nil {
		t.Fatal("expected Anthropic provider constructor from public API")
	}
	if NewGeminiProvider("test") == nil {
		t.Fatal("expected Gemini provider constructor from public API")
	}
	if NewMistralProvider("test") == nil {
		t.Fatal("expected Mistral provider constructor from public API")
	}

	tracer := &testTracer{}
	SetGlobalTracer(tracer)
	if GetGlobalTracer() != tracer {
		t.Fatal("expected global tracer to be replaceable from public API")
	}
}
