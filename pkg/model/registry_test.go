package model

import "testing"

func TestRegistrySpecsIncludesOfficialDocsAndGeminiAliases(t *testing.T) {
	specs := RegistrySpecs()
	if len(specs) == 0 {
		t.Fatalf("RegistrySpecs returned no specs")
	}

	foundGeminiFlash := false
	foundGeminiPro := false
	for _, spec := range specs {
		if spec.Provider == "gemini" {
			if spec.OfficialDocsURL == "" {
				t.Fatalf("gemini spec missing OfficialDocsURL: %+v", spec)
			}
			if spec.Prefix == "gemini-flash" {
				foundGeminiFlash = true
			}
			if spec.Prefix == "gemini-pro" {
				foundGeminiPro = true
			}
		}
	}

	if !foundGeminiFlash {
		t.Fatalf("RegistrySpecs does not include gemini-flash")
	}
	if !foundGeminiPro {
		t.Fatalf("RegistrySpecs does not include gemini-pro")
	}
}
