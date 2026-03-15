# Model Registry Single Source of Truth

## Objective

Define a production-ready, easy-to-maintain SDK-owned registry for provider/model capabilities, pricing, metadata, and official provider sources.

## Current Source of Truth in SDK

- Capabilities map: `pkg/model/capabilities.go`
- Pricing map: `pkg/model/pricing.go`
- Model metadata map: `pkg/model/metadata.go`
- Provider metadata and official source URLs: `pkg/model/provider.go`
- Typed registry view: `pkg/model/registry.go`
- Unified sync automation from official sources: `scripts/sync-model-registry/`

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

- Deterministic output: generated capability entries are sorted alphabetically by prefix per provider.
- Safe default: sync script is report-only unless `-write` is explicitly set.
- Provider-specific hardening: strict family/shape validation rules per provider.
- HTML parsing strategy: `goquery`-based DOM extraction (body text, code/pre blocks, links), then provider-specific extraction rules.
- Explainability: optional `-verbose` output lists concrete missing model IDs.
- CI automation: scheduled workflow opens PRs with auditable diffs.

## Operational Flow

1. Scheduled/manual workflow runs the unified sync entrypoint.
2. Script fetches official docs and pricing URLs from SDK provider registry.
3. Script sanitizes HTML, extracts provider-specific model IDs, validates candidates.
4. Script computes missing capability and pricing entries.
5. Script rewrites provider sections in deterministic sorted order for both files.
6. Workflow runs model tests and opens a PR.

## Scope Boundaries

This pipeline governs capability and pricing synchronization today and is designed as the single extensible entrypoint for future model registry signals.
