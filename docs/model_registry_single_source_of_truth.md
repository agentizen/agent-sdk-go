# Model Registry Single Source of Truth

## Objective

Define a production-ready, easy-to-maintain SDK-owned registry for provider/model capabilities and provider documentation sources.

## Current Source of Truth in SDK

- Capabilities map: `pkg/model/capabilities.go`
- Official provider docs URLs: `pkg/model/provider_sources.go`
- Typed registry view: `pkg/model/registry.go`
- Sync automation from official sources: `scripts/sync_capabilities_from_official_docs.go`

## Target Structure

Use the SDK as the only source for:

1. Providers
2. Model prefixes/identifiers
3. Capability matrix (vision, documents)
4. Official source URLs used by automation
5. Validation rules per provider used by sync pipeline
6. Exportable typed metadata view for downstream consumers

## Data Model (incremental)

The current implementation keeps backward compatibility via `ProviderSupports(...)` and adds source metadata with:

- `OfficialProviderModelDocsURL(provider string) (string, bool)`
- `KnownCapabilityProviders() []string`
- `RegistrySpecs() []ModelCapabilitySpec`

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

1. Scheduled/manual workflow runs sync script.
2. Script fetches official provider docs URLs from SDK registry.
3. Script sanitizes HTML, extracts provider-specific model IDs, validates candidates.
4. Script computes missing multimodal models and updates `capabilities.go` only in write mode.
5. Script rewrites provider sections in deterministic sorted order.
6. Workflow runs capability tests and opens a PR.

## Scope Boundaries

This pipeline currently governs capability synchronization.

Pricing/token economics remain external to this registry today and can be integrated later by adding explicit pricing fields to the future typed model registry.
