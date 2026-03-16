# Model Registry Single Source of Truth

## Objective

Define a production-ready, easy-to-maintain SDK-owned registry for provider/model capabilities, pricing, metadata, and official provider sources.

## Current Source of Truth in SDK

- Source JSON: `scripts/sources/model_registry.json`
- Source schema: `scripts/sources/model_registry.schema.json`
- Capabilities map: `pkg/model/capabilities.go`
- Pricing map: `pkg/model/pricing.go`
- Model metadata map: `pkg/model/metadata.go`
- Provider metadata and official source URLs: `pkg/model/provider.go`
- Typed registry view: `pkg/model/registry.go`
- Unified generation entrypoint: `scripts/sync-model-registry/`

## Target Structure

Use the SDK as the only source for:

1. Providers
2. Model prefixes/identifiers
3. Capability matrix (vision, documents)
4. Pricing matrix (USD per million tokens)
5. Official source URLs used by automation
6. Validation rules per provider used by sync pipeline
7. Exportable typed metadata view for downstream consumers

## Data Model (incremental)

The current implementation keeps runtime compatibility via `ProviderSupports(...)` and provides source metadata with:

- `OfficialProviderModelDocsURL(provider string) (string, bool)`
- `OfficialProviderPricingURL(provider string) (string, bool)`
- `KnownProviders() []string`
- `AllModelSpecs() []ModelSpec`

Next incremental extension can introduce an exported typed model registry, for example:

- `ProviderRegistry`
- `ModelCapabilitySpec`
- `SourceMetadata` (URL, last-checked timestamp, sync strategy)

without changing existing runtime behavior.

## Production Readiness Principles Applied

- Deterministic output: generated files are sorted by provider and model ID.
- Safe default: generation is report-only unless `-write` is explicitly set.
- Strict input validation: JSON source rejects unknown fields and invalid IDs.
- Schema alignment gate: source `version` must match schema `x-source-version`.
- Explainability: optional `-verbose` output lists concrete source entries by provider.
- CI automation: scheduled workflow opens PRs with auditable diffs.

## Operational Flow

1. Scheduled/manual workflow runs the unified sync entrypoint.
2. Script validates source JSON against code-level constraints and schema version.
3. Script renders generated model registry files in deterministic order.
4. Workflow runs model tests and opens a PR.

## Source Maintenance Process

The generation workflow does not discover new models by itself. It only renders Go files from the source JSON.

### Manual Update (baseline)

1. Update `scripts/sources/model_registry.json` with provider/model/pricing metadata.
2. Run `go run ./scripts/sync-model-registry/ -write`.
3. Run `gofmt -w pkg/model/capabilities.go pkg/model/pricing.go pkg/model/metadata.go pkg/model/provider.go`.
4. Run tests and open a PR.

### AI-Assisted Update (recommended)

1. An AI coding agent prepares a proposed update to `scripts/sources/model_registry.json` based on official provider pages.
2. The proposal must keep unknown values explicit (`0` pricing or empty `release_date`) instead of speculative data.
3. The proposal is validated through generation + tests.
4. A human reviewer approves or requests corrections before merge.

This human-in-the-loop model keeps automation speed while preserving source quality and auditability.

### Dedicated Workflow

Use a single unified workflow: `.github/workflows/model-capabilities-sync.yml`.

It can run in two modes:

- AI refresh enabled: update `scripts/sources/model_registry.json` through an external LLM API, then validate schema, generate files, test, and open one PR.
- AI refresh disabled: skip AI update and only validate schema, generate files, test, and open one PR.

This keeps source and generated changes together in the same review.

Required configuration:

- Secret: `MODEL_REGISTRY_AI_API_KEY`
- Optional variables: `MODEL_REGISTRY_AI_BASE_URL`, `MODEL_REGISTRY_AI_MODEL`

If the API key is missing while AI mode is enabled, the workflow fails explicitly to avoid silent no-op runs.

## Scope Boundaries

This pipeline governs source-driven model registry generation and remains the single extensible entrypoint for future model registry signals.
