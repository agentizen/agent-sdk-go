// Package main demonstrates end-to-end skill loading and runtime behavior.
//
// It tests the full skill lifecycle:
//   - Loading skills from inline markdown strings
//   - Skill header parsing (name, description, version)
//   - Skill registry (registration, retrieval, listing)
//   - The load_skill tool (same mechanism used by the Runner at runtime)
//   - Agent integration
//
// Run: go run ./examples/skills_runtime
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	agentsdk "github.com/agentizen/agent-sdk-go"
)

const codeReviewSkill = `---
name: code-review
description: Performs a thorough code review with best practices
version: "1.0.0"
---
# Code Review Skill

When asked to review code, follow these steps:

1. **Correctness** - Check for bugs, logic errors, and edge cases.
2. **Style** - Verify adherence to the project style guide.
3. **Performance** - Identify potential bottlenecks or inefficiencies.
4. **Security** - Look for injection vulnerabilities, data leaks, etc.

Always provide actionable, constructive feedback.
`

const testWriterSkill = `---
name: test-writer
description: Writes comprehensive unit tests for Go code
version: "2.1.0"
---
# Test Writer Skill

When asked to write tests, follow these guidelines:

- Use table-driven tests where appropriate.
- Cover happy paths, error paths, and edge cases.
- Use the testify/suite pattern for complex test setups.
- Target >80% coverage for new code.
`

func main() {
	fmt.Println("=== Skills Runtime Example ===")
	fmt.Println("This example tests the full skill lifecycle end-to-end.")
	fmt.Println()

	ctx := context.Background()
	passed := 0
	failed := 0

	check := func(name string, err error) {
		if err != nil {
			fmt.Printf("  [FAIL] %s: %v\n", name, err)
			failed++
		} else {
			fmt.Printf("  [ok]   %s\n", name)
			passed++
		}
	}

	// --- 1. Load skills from inline strings ---
	fmt.Println("--- Skill Loading ---")

	review, err := agentsdk.LoadSkillFromString(codeReviewSkill)
	check("Load code-review skill", err)

	writer, err := agentsdk.LoadSkillFromString(testWriterSkill)
	check("Load test-writer skill", err)

	// --- 2. Verify headers ---
	fmt.Println("\n--- Header Parsing ---")

	h := review.Header()
	if h.Name != "code-review" {
		check("Header name", fmt.Errorf("expected 'code-review', got %q", h.Name))
	} else {
		check("Header name = 'code-review'", nil)
	}
	if h.Description != "Performs a thorough code review with best practices" {
		check("Header description", fmt.Errorf("unexpected: %q", h.Description))
	} else {
		check("Header description", nil)
	}
	if h.Version != "1.0.0" {
		check("Header version", fmt.Errorf("expected '1.0.0', got %q", h.Version))
	} else {
		check("Header version = '1.0.0'", nil)
	}

	// --- 3. Skill content loading ---
	fmt.Println("\n--- Content Loading ---")

	content, err := review.Load(ctx)
	check("Load skill content", err)
	if err == nil {
		if !strings.Contains(content, "Code Review Skill") {
			check("Content contains title", fmt.Errorf("missing title in content"))
		} else {
			check("Content contains title", nil)
		}
		if !strings.Contains(content, "Correctness") {
			check("Content contains steps", fmt.Errorf("missing steps in content"))
		} else {
			check("Content contains steps", nil)
		}
		fmt.Printf("           Content length: %d bytes\n", len(content))
	}

	// --- 4. Skill registry ---
	fmt.Println("\n--- Skill Registry ---")

	registry := agentsdk.NewSkillRegistry()

	err = registry.Register(review)
	check("Register code-review", err)

	err = registry.Register(writer)
	check("Register test-writer", err)

	// Duplicate should fail
	err = registry.Register(review)
	if err == nil {
		check("Rejects duplicate", fmt.Errorf("expected error for duplicate"))
	} else {
		check("Rejects duplicate", nil)
	}

	// Retrieve by name
	got, ok := registry.Get("code-review")
	if !ok {
		check("Get('code-review')", fmt.Errorf("not found"))
	} else if got.Header().Name != "code-review" {
		check("Get('code-review')", fmt.Errorf("wrong skill returned"))
	} else {
		check("Get('code-review')", nil)
	}

	// List names
	names := registry.Names()
	if len(names) != 2 {
		check("Names() count", fmt.Errorf("expected 2, got %d", len(names)))
	} else {
		check("Names() count = 2", nil)
	}
	fmt.Printf("           Registered: %v\n", names)

	// --- 5. load_skill tool (runtime mechanism) ---
	fmt.Println("\n--- load_skill Tool ---")

	// This is exactly what the Runner creates when an agent has skills.
	// The LLM can call this tool to load a skill's full content on demand.
	loadTool := agentsdk.NewLoadSkillTool([]agentsdk.Skill{review, writer})

	if loadTool.GetName() != "load_skill" {
		check("Tool name", fmt.Errorf("expected 'load_skill', got %q", loadTool.GetName()))
	} else {
		check("Tool name = 'load_skill'", nil)
	}

	desc := loadTool.GetDescription()
	if !strings.Contains(desc, "code-review") || !strings.Contains(desc, "test-writer") {
		check("Tool description lists skills", fmt.Errorf("missing skill names in: %q", desc))
	} else {
		check("Tool description lists available skills", nil)
	}

	// Verify schema has enum constraint
	schema := loadTool.GetParametersSchema()
	props, _ := schema["properties"].(map[string]interface{})
	nameProp, _ := props["name"].(map[string]interface{})
	enumVals, _ := nameProp["enum"].([]interface{})
	if len(enumVals) != 2 {
		check("Schema enum constraint", fmt.Errorf("expected 2 enum values, got %d", len(enumVals)))
	} else {
		check("Schema enum constraint = 2 values", nil)
	}

	// Execute: load code-review skill
	result, err := loadTool.Execute(ctx, map[string]interface{}{"name": "code-review"})
	check("Execute load_skill('code-review')", err)
	if err == nil {
		resultStr, ok := result.(string)
		if !ok {
			check("Result is string", fmt.Errorf("result type: %T", result))
		} else if !strings.Contains(resultStr, "Code Review Skill") {
			check("Result contains skill content", fmt.Errorf("missing content"))
		} else {
			check("Result contains skill content", nil)
			fmt.Printf("           Loaded %d bytes of skill content\n", len(resultStr))
		}
	}

	// Execute: load test-writer skill
	result, err = loadTool.Execute(ctx, map[string]interface{}{"name": "test-writer"})
	check("Execute load_skill('test-writer')", err)

	// Execute: unknown skill should fail
	_, err = loadTool.Execute(ctx, map[string]interface{}{"name": "nonexistent"})
	if err == nil {
		check("Rejects unknown skill", fmt.Errorf("expected error"))
	} else {
		check("Rejects unknown skill name", nil)
	}

	// Execute: missing name param should fail
	_, err = loadTool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		check("Rejects missing name param", fmt.Errorf("expected error"))
	} else {
		check("Rejects missing 'name' parameter", nil)
	}

	// --- 6. Agent integration ---
	fmt.Println("\n--- Agent Integration ---")

	assistant := agentsdk.NewAgent("Skill Test Agent")
	assistant.SetSystemInstructions("You are a code assistant.")
	assistant.WithSkills(review, writer)

	if len(assistant.Skills) != 2 {
		check("Agent skill count", fmt.Errorf("expected 2, got %d", len(assistant.Skills)))
	} else {
		check("Agent skill count = 2", nil)
	}

	// --- Summary ---
	fmt.Printf("\n=== Results: %d passed, %d failed ===\n", passed, failed)
	if failed > 0 {
		log.Fatalf("%d tests failed", failed)
	}
}
