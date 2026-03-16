# Agent SDK Go

Build and orchestrate AI agents in Go with a maintained multi-provider runtime and a typed model registry.

<p align="center">
  <a href="https://github.com/citizenofai/agent-sdk-go/actions/workflows/code-quality.yml"><img src="https://github.com/citizenofai/agent-sdk-go/actions/workflows/code-quality.yml/badge.svg" alt="Code Quality"></a>
  <a href="https://github.com/citizenofai/agent-sdk-go/actions/workflows/model-capabilities-sync.yml"><img src="https://github.com/citizenofai/agent-sdk-go/actions/workflows/model-capabilities-sync.yml/badge.svg" alt="Model Registry Refresh"></a>
  <a href="https://github.com/citizenofai/agent-sdk-go/commits/main/scripts/sources/model_registry.json"><img src="https://img.shields.io/github/last-commit/citizenofai/agent-sdk-go/main?path=scripts/sources/model_registry.json&label=registry.json%20last%20update" alt="registry.json last update"></a>
  <a href="https://pkg.go.dev/github.com/citizenofai/agent-sdk-go"><img src="https://pkg.go.dev/badge/github.com/citizenofai/agent-sdk-go.svg" alt="PkgGoDev"></a>
</p>

## Why Use This Repository

This project is useful in two common scenarios:

1. You need a Go SDK to run AI agents with tools, handoffs, streaming, and tracing.
2. You need a reliable model catalog (provider, capabilities, pricing, metadata) that your app or AI agent can query directly.

If you only need catalog data, you can use `pkg/model` without adopting the full agent runtime.

## Start in 2 Minutes

```bash
go get github.com/citizenofai/agent-sdk-go
```

```go
package main

import (
	"fmt"

	"github.com/citizenofai/agent-sdk-go/pkg/model"
)

func main() {
	spec, ok := model.GetModelSpec("openai", "gpt-5.4")
	if !ok {
		panic("model not found")
	}

	fmt.Println(spec.Provider, spec.ModelID)
	fmt.Println(spec.Metadata.DisplayName)
	fmt.Println(spec.Pricing.InputCostPerMillion, spec.Pricing.OutputCostPerMillion)
	fmt.Println(spec.Capabilities.Vision, spec.Capabilities.Documents)
}
```

## Model Registry API (Core Types)

Typed structures:

- [ProviderSpec](./pkg/model/provider.go)
- [ModelPricingSpec](./pkg/model/pricing.go)
- [ModelMetadata](./pkg/model/metadata.go)
- [ModelCapabilitySet](./pkg/model/capabilities.go)
- [ModelSpec](./pkg/model/registry.go)

Core queries:

- [GetProvider](./pkg/model/provider.go)
- [GetModelPricing](./pkg/model/pricing.go)
- [GetModelMetadata](./pkg/model/metadata.go)
- [CapabilitiesFor](./pkg/model/capabilities.go)
- [GetModelSpec](./pkg/model/registry.go)
- [AllModelSpecs](./pkg/model/registry.go)

## Agent Runtime (Core Packages)

- `pkg/agent`: agent definition
- `pkg/runner`: execution loop and streaming
- `pkg/tool`: function tools and schema
- `pkg/model/providers/*`: provider clients (OpenAI, Anthropic, Gemini, Mistral, LM Studio, Azure OpenAI)

## Model Registry Refresh Workflow

Single workflow: [ .github/workflows/model-capabilities-sync.yml ](./.github/workflows/model-capabilities-sync.yml)

It can:

1. Optionally refresh `scripts/sources/model_registry.json` using an LLM.
2. Validate JSON against `scripts/sources/model_registry.schema.json`.
3. Regenerate:
   1. `pkg/model/capabilities.go`
   2. `pkg/model/pricing.go`
   3. `pkg/model/metadata.go`
   4. `pkg/model/provider.go`
4. Run tests and open one PR containing both source and generated updates.

Required for AI refresh mode:

- Secret: `MODEL_REGISTRY_AI_API_KEY`
- Optional variables: `MODEL_REGISTRY_AI_BASE_URL`, `MODEL_REGISTRY_AI_MODEL`

Default OpenAI-compatible values:

- `MODEL_REGISTRY_AI_BASE_URL`: `https://api.openai.com/v1`
- `MODEL_REGISTRY_AI_MODEL`: `gpt-5` or `gpt-5-mini`

Copilot note:

- Copilot Chat is not a direct API target for this workflow.
- Use an OpenAI-compatible endpoint/token (OpenAI or equivalent gateway).

## Examples and Detailed Guides

High-signal examples:

- [examples/openai_example](./examples/openai_example)
- [examples/openai_multi_agent_example](./examples/openai_multi_agent_example)
- [examples/gemini_example](./examples/gemini_example)
- [examples/mistral_example](./examples/mistral_example)
- [examples/anthropic_example](./examples/anthropic_example)
- [examples/bidirectional_flow_example](./examples/bidirectional_flow_example)
- [examples/azure_openai_example](./examples/azure_openai_example)

Project docs:

- [docs/model_registry_single_source_of_truth.md](./docs/model_registry_single_source_of_truth.md)
- [docs/GO_QUALITY_GUIDELINES.md](./docs/GO_QUALITY_GUIDELINES.md)

## Development and Contribution

Quick setup:

```bash
git clone https://github.com/citizenofai/agent-sdk-go.git
cd agent-sdk-go
./scripts/ci_setup.sh
./scripts/check_all.sh
```

Contributing guide:

- [CONTRIBUTING.md](./CONTRIBUTING.md)

License:

- [LICENSE](./LICENSE)
