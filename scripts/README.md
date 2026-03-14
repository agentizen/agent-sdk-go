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
| `go run ./scripts/sync_capabilities_from_official_docs.go` | Sync model capabilities from provider docs |

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

### `sync_capabilities_from_official_docs.go`

A Go script that fetches model capability data from official provider documentation (OpenAI, Anthropic, Gemini, Mistral) and updates `pkg/model/capabilities.go`. Run automatically by the `model-capabilities-sync` GitHub Actions workflow.

```bash
# Dry-run (print diff only)
go run ./scripts/sync_capabilities_from_official_docs.go \
  -capabilities pkg/model/capabilities.go

# Write changes
go run ./scripts/sync_capabilities_from_official_docs.go \
  -write \
  -capabilities pkg/model/capabilities.go
gofmt -w pkg/model/capabilities.go
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
