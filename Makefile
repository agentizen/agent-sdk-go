# Agent SDK Go — Development & CI Makefile
#
# Usage:
#   make              Run full validation pipeline (tidy + fmt + vet + lint + build + test)
#   make ci-setup     Install CI tools (golangci-lint, gosec, goimports)
#   make ci           Full CI pipeline (ci-setup + all)
#   make quality      Lint + test with coverage
#   make examples     Run all runnable examples

COVERAGE_FILE  := coverage.out
COVERPKG       := github.com/agentizen/agent-sdk-go/pkg/...
TEST_PACKAGES  := $(shell go list ./... | grep -v '/examples')

# ---------- Aggregate targets ----------

.PHONY: all
all: tidy fmt vet lint build test

.PHONY: ci
ci: ci-setup all

.PHONY: quality
quality: lint test-coverage

# ---------- Dependencies ----------

.PHONY: tidy
tidy:
	@echo "--- go mod tidy ---"
	go mod tidy

# ---------- Formatting ----------

.PHONY: fmt
fmt:
	@echo "--- gofmt check ---"
	@BADFILES=$$(find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*' | xargs gofmt -l); \
	if [ -n "$$BADFILES" ]; then \
		echo "Files need gofmt:"; echo "$$BADFILES"; exit 1; \
	fi
	@echo "ok"

.PHONY: fmt-fix
fmt-fix:
	find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*' | xargs gofmt -s -w

# ---------- Imports ----------

.PHONY: imports
imports:
	@echo "--- goimports check ---"
	@which goimports > /dev/null 2>&1 || { echo "goimports not installed (run: make ci-setup)"; exit 1; }
	@BADFILES=$$(find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*' | xargs goimports -l); \
	if [ -n "$$BADFILES" ]; then \
		echo "Files need goimports:"; echo "$$BADFILES"; exit 1; \
	fi
	@echo "ok"

.PHONY: imports-fix
imports-fix:
	find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*' | xargs goimports -w

# ---------- Vet ----------

.PHONY: vet
vet:
	@echo "--- go vet ---"
	go vet ./...

# ---------- Lint ----------

.PHONY: lint
lint:
	@echo "--- golangci-lint ---"
	golangci-lint run --timeout=5m

# ---------- Security ----------

.PHONY: security
security:
	@echo "--- gosec ---"
	gosec -quiet ./...

# ---------- Build ----------

.PHONY: build
build:
	@echo "--- go build ---"
	go build -v ./...

# ---------- Test ----------

.PHONY: test
test:
	@echo "--- go test ---"
	go test --count=1 $(TEST_PACKAGES)

.PHONY: test-verbose
test-verbose:
	go test -v --count=1 $(TEST_PACKAGES)

.PHONY: test-race
test-race:
	@echo "--- go test -race ---"
	go test -race --count=1 $(TEST_PACKAGES)

.PHONY: test-coverage
test-coverage:
	@echo "--- go test -race -cover ---"
	go test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(TEST_PACKAGES)
	go tool cover -func=$(COVERAGE_FILE)

.PHONY: test-coverage-html
test-coverage-html: test-coverage
	go tool cover -html=$(COVERAGE_FILE)

# ---------- Examples ----------

.PHONY: examples
examples:
	@echo "--- Running examples ---"
	go run ./examples/mcp_live_server/
	@echo ""
	go run ./examples/tools_execution/
	@echo ""
	go run ./examples/skills_runtime/

# ---------- CI Setup ----------

.PHONY: ci-setup
ci-setup:
	@echo "--- Installing CI tools ---"
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest

# ---------- Clean ----------

.PHONY: clean
clean:
	rm -f $(COVERAGE_FILE)
