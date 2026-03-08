#!/bin/bash

set -e

# CI Setup Script to ensure consistent Go version and toolchain settings
# This helps with VCS stamping issues and version mismatches in CI environments

echo "Setting up CI environment..."

# Check Go version
echo "Checking Go version..."
./scripts/check_go_version.sh || {
  echo "Go version check failed. Please install Go 1.25 or later."
  exit 1
}

# Check Go version
GO_VERSION=$(go version | awk '{print $3}')
echo "Detected Go version: $GO_VERSION"

echo "Installing required tools..."

# Define paths
GOBIN=$(go env GOPATH)/bin
export PATH=$GOBIN:$PATH

echo "Installing golangci-lint v1.54.2..."
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2

echo "Installing gosec latest..."
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

echo "Installing goimports..."
# Install goimports
go install golang.org/x/tools/cmd/goimports@latest

echo "Verifying tool installation..."
echo "golangci-lint version:"
which golangci-lint && golangci-lint --version || echo "golangci-lint not found in PATH"

echo "gosec version:"
which gosec && gosec --version || echo "gosec not found in PATH"

echo "goimports version:"
which goimports && echo "goimports installed" || echo "goimports not found in PATH"

# Set environment variables to avoid VCS stamping issues
export GOFLAGS="-buildvcs=false"

echo "Go environment:"
echo "$GO_VERSION"
echo "-buildvcs=false"

echo "CI setup complete!" 