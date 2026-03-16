#!/bin/bash

set -euo pipefail

echo "=== Syncing model registry ==="
go run ./scripts/sync-model-registry/ -write

echo "=== Formatting generated files ==="
gofmt -w \
  pkg/model/capabilities.go \
  pkg/model/pricing.go \
  pkg/model/metadata.go \
  pkg/model/provider.go

echo "=== Running model tests ==="
go test ./test/model/... -run TestProviderSupports -count=1
go test ./pkg/model/... -run TestPricing -count=1

echo "Model registry sync complete! ✅"
