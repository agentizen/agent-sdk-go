# Contributing to Agent SDK Go

Thank you for your interest in contributing to Agent SDK Go! This document provides guidelines and instructions for contributing to this project.

## Code of Conduct

Please be respectful and considerate of others when contributing to this project. We aim to foster an inclusive and welcoming community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR-USERNAME/agent-sdk-go.git`
3. Set up the development environment:
   ```bash
   ./scripts/ci_setup.sh       # installs golangci-lint, gosec, goimports
   ./scripts/install-hooks.sh  # installs Git pre-commit hook
   ```
4. Create a new branch for your changes: `git checkout -b feature/your-feature-name`

## Development Workflow

1. Make your changes
2. Run the full quality pipeline before submitting:
   ```bash
   ./scripts/check_all.sh
   ```
3. Commit your changes — the pre-commit hook will run `gofmt`, `go vet`, `go build`, and `golangci-lint` automatically
4. Push your changes to your fork
5. Submit a pull request

## Git Hooks

The repository provides a native Git pre-commit hook (no Python required).

**Install once per clone:**
```bash
./scripts/install-hooks.sh
```

**Run manually:**
```bash
.git/hooks/pre-commit
```

**Uninstall:**
```bash
rm .git/hooks/pre-commit
```

The hook runs fast local checks only: `gofmt`, `go vet`, `go build`, and `golangci-lint` (if installed). Heavier checks (security scan, full test suite) run in CI.

## Pull Request Process

1. Ensure your code passes all checks (linting, security, tests)
2. Update documentation if necessary
3. Include a clear description of the changes in your pull request
4. Link any related issues in your pull request description

## Coding Standards

- Follow Go best practices and idiomatic Go
- Use meaningful variable and function names
- Write clear comments and documentation
- Include tests for new functionality
- Keep functions and methods focused and small

## Testing

All new features and bug fixes should include tests. Run the tests with:

```bash
cd test && make test
```

## Documentation

Update documentation to reflect any changes you make. This includes:

- Code comments
- README.md updates
- Examples if applicable

## Versioning

We use semantic versioning. The version is managed in the `version.txt` file and can be bumped using:

```bash
./scripts/version.sh bump
```

## License

By contributing to this project, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).

## Questions?

If you have any questions or need help, please open an issue or reach out to the maintainers. 