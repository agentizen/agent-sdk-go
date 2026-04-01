package main

import (
	"fmt"
	"log"

	agentsdk "github.com/agentizen/agent-sdk-go"
)

// inlineSkillMarkdown is a skill defined as an inline markdown string
// with YAML frontmatter. In production, skills are typically loaded from
// files using agentsdk.LoadSkillFromFile.
const inlineSkillMarkdown = `---
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

const anotherSkillMarkdown = `---
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
	fmt.Println("=== Skill Loading Example ===")

	// --- 1. Load skills from inline strings ---
	codeReviewSkill, err := agentsdk.LoadSkillFromString(inlineSkillMarkdown)
	if err != nil {
		log.Fatalf("Failed to load code-review skill: %v", err)
	}

	testWriterSkill, err := agentsdk.LoadSkillFromString(anotherSkillMarkdown)
	if err != nil {
		log.Fatalf("Failed to load test-writer skill: %v", err)
	}

	// --- 2. Print loaded skill headers ---
	fmt.Println("Loaded skills:")
	for _, s := range []agentsdk.Skill{codeReviewSkill, testWriterSkill} {
		h := s.Header()
		fmt.Printf("  - %s (v%s): %s\n", h.Name, h.Version, h.Description)
	}

	// --- 3. Create a skill registry and register skills ---
	registry := agentsdk.NewSkillRegistry()

	if err := registry.Register(codeReviewSkill); err != nil {
		log.Fatalf("Failed to register skill: %v", err)
	}
	if err := registry.Register(testWriterSkill); err != nil {
		log.Fatalf("Failed to register skill: %v", err)
	}

	fmt.Println("\nSkill registry contents:")
	for _, name := range registry.Names() {
		fmt.Printf("  - %s\n", name)
	}

	// --- 4. Create an agent with skills attached ---
	assistant := agentsdk.NewAgent("Code Assistant")
	assistant.SetSystemInstructions("You are a senior software engineer who can review code and write tests.")
	assistant.WithSkills(codeReviewSkill, testWriterSkill)

	fmt.Println("\nAgent configured:")
	fmt.Printf("  Name:   %s\n", assistant.Name)
	fmt.Printf("  Skills: %d\n", len(assistant.Skills))
	for _, s := range assistant.Skills {
		h := s.Header()
		fmt.Printf("    - %s (v%s): %s\n", h.Name, h.Version, h.Description)
	}

	// --- 5. Explain the runtime behavior ---
	fmt.Println("\nRuntime behavior:")
	fmt.Println("  When the agent runs, skill headers are injected into the system prompt.")
	fmt.Println("  A 'load_skill' tool is automatically added, allowing the LLM to load")
	fmt.Println("  the full skill content on demand. This keeps the initial prompt small")
	fmt.Println("  while making all skills discoverable.")
}
