package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	sourcePath       = "scripts/sources/model_registry.json"
	schemaPath       = "scripts/sources/model_registry.schema.json"
	capabilitiesPath = "pkg/model/capabilities.go"
	pricingPath      = "pkg/model/pricing.go"
	metadataPath     = "pkg/model/metadata.go"
	providersPath    = "pkg/model/provider.go"
)

func main() {
	writeChanges := flag.Bool("write", false, "Write changes to output files (default: report only)")
	verbose := flag.Bool("verbose", false, "Print discovered model updates per provider")
	flag.Parse()

	if err := run(*writeChanges, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(writeChanges, verbose bool) error {
	schemaVersion, err := loadRegistrySchemaVersion(schemaPath)
	if err != nil {
		return err
	}

	source, err := loadRegistrySource(sourcePath)
	if err != nil {
		return err
	}
	if source.Version != schemaVersion {
		return fmt.Errorf("registry source version %d does not match schema version %d", source.Version, schemaVersion)
	}

	if sourceModelCount(source) == 0 {
		fmt.Println("model registry source is empty")
		return nil
	}

	reportSourceSummary(source, verbose)

	if !writeChanges {
		fmt.Println("report-only mode: re-run with -write to generate registry files from source JSON")
		return nil
	}

	capabilitiesContent := renderCapabilitiesFromSource(source)
	if err := os.WriteFile(capabilitiesPath, []byte(capabilitiesContent), 0644); err != nil {
		return fmt.Errorf("write capabilities file: %w", err)
	}

	pricingContent := renderPricingFromSource(source)
	if err := os.WriteFile(pricingPath, []byte(pricingContent), 0644); err != nil {
		return fmt.Errorf("write pricing file: %w", err)
	}

	metadataContent := renderMetadataFromSource(source)
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		return fmt.Errorf("write metadata file: %w", err)
	}

	providerContent := renderProvidersFromSource(source)
	if err := os.WriteFile(providersPath, []byte(providerContent), 0644); err != nil {
		return fmt.Errorf("write providers file: %w", err)
	}

	fmt.Println("generated capabilities, pricing, metadata, and provider files from source JSON")
	return nil
}
