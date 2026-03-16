# Script Utilities

This directory contains utility scripts for the Agent SDK Go project.

## Quick Reference

| Script | Purpose |
|--------|---------|
| `./scripts/check_all.sh` | Run all checks (version, lint, security, tests) |
| `./scripts/lint.sh` | Formatting + imports + vet + build |
| `./scripts/security_check.sh` | Security scan (gosec) |
| `./scripts/build.sh` | Build with consistent flags |
| `./scripts/version.sh bump` | Interactive version tag bump |
| `./scripts/ci_setup.sh` | Install required tools (CI / first-time setup) |
| `./scripts/install-hooks.sh` | Install native Git pre-commit hook |
| `./scripts/check_go_version.sh` | Verify Go version meets requirements |
| `go run ./scripts/sync-model-registry/` | Validate source registry JSON and generate model registry code |

---

## Scripts

### `check_all.sh`

Runs the full quality pipeline in sequence: Go version check → lint → security → tests.

```bash
./scripts/check_all.sh
```

### `lint.sh`

Checks code formatting (`gofmt`), import organization (`goimports`), runs `go vet`, and verifies the project builds.

```bash
./scripts/lint.sh
```

### `security_check.sh`

Runs a security scan using `gosec`, excluding the `examples/` directory.

```bash
./scripts/security_check.sh
```

Requires `gosec` to be installed (`go install github.com/securego/gosec/v2/cmd/gosec@latest`).

### `build.sh`

Builds all packages with consistent flags. Supports `--race` and `--verbose` options.

```bash
./scripts/build.sh
./scripts/build.sh --race
./scripts/build.sh --verbose ./pkg/...
```

### `version.sh`

Interactive script to bump the semantic version and create a signed git tag. Prompts for major/minor/patch bump and pushes the tag.

```bash
./scripts/version.sh bump
```

### `ci_setup.sh`

Installs all required tooling for CI or first-time local setup: `golangci-lint`, `gosec`, `goimports`.

```bash
./scripts/ci_setup.sh
```

### `check_go_version.sh`

Verifies the installed Go version meets the minimum requirement (1.25+). Called automatically by `check_all.sh` and `ci_setup.sh`.

```bash
./scripts/check_go_version.sh
```

### `sync-model-registry/`

A unified Go generation package with a one-way flow:

1. Read source JSON: `scripts/sources/model_registry.json`
2. Validate schema metadata from: `scripts/sources/model_registry.schema.json`
3. Enforce strict JSON decoding (unknown fields fail)
4. Generate code files under `pkg/model/`

There is no runtime scraping in this command.

The command generates:

- `pkg/model/capabilities.go`
- `pkg/model/pricing.go`
- `pkg/model/metadata.go`
- `pkg/model/provider.go`

Schema is defined in `scripts/sources/model_registry.schema.json`.

```bash
# Dry-run (report mode)
go run ./scripts/sync-model-registry/

# Write changes
go run ./scripts/sync-model-registry/ -write
gofmt -w pkg/model/capabilities.go pkg/model/pricing.go pkg/model/metadata.go pkg/model/provider.go
```

---

### `install-hooks.sh`

Installs the native Git pre-commit hook from `scripts/hooks/pre-commit`. No external dependencies required.

```bash
./scripts/install-hooks.sh
```

The hook runs automatically on every `git commit` and checks:
- `gofmt` — code formatting
- `go vet` — suspicious constructs
- `go build` — compilation
- `golangci-lint` — lint rules (skipped silently if not installed)

Run manually at any time:
```bash
.git/hooks/pre-commit
```

Uninstall:
```bash
rm .git/hooks/pre-commit
```

> The hook is stored at `scripts/hooks/pre-commit` (versioned) and symlinked into `.git/hooks/` by the install script.
