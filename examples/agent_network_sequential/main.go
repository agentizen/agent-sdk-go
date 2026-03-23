package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
	"github.com/citizenofai/agent-sdk-go/pkg/model/providers/openai"
	"github.com/citizenofai/agent-sdk-go/pkg/network"
	"github.com/citizenofai/agent-sdk-go/pkg/runner"
)

// Sequential Network example
//
// Three agents form a processing pipeline: Planner → Coder → Reviewer.
// Each agent receives the previous agent's output as its input, creating
// a chain where each step builds on the previous work.

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	provider := openai.NewProvider(apiKey)
	provider.SetDefaultModel("gpt-4.1-mini")

	// Each agent in the pipeline has a focused, single-step responsibility.
	planner := agent.NewAgent("Planner").
		WithModel("gpt-4.1-mini").
		SetSystemInstructions("You are a software architect. Given a feature request, produce a concise implementation plan with clear steps.")

	coder := agent.NewAgent("Coder").
		WithModel("gpt-4.1-mini").
		SetSystemInstructions("You are a Go developer. Given an implementation plan, write the Go code that implements it. Return only the code and brief inline comments.")

	reviewer := agent.NewAgent("Reviewer").
		WithModel("gpt-4.1-mini").
		SetSystemInstructions("You are a senior code reviewer. Given Go code, review it for correctness, idiomatic style, performance, and security. Provide specific actionable feedback.")

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: planner, Role: "Architect"},
			network.AgentSlot{Agent: coder, Role: "Developer"},
			network.AgentSlot{Agent: reviewer, Role: "Reviewer"},
		).
		WithStrategy(network.StrategySequential) // Each agent feeds the next.

	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	prompt := "Implement a thread-safe in-memory cache with TTL expiration in Go"

	fmt.Printf("Running sequential pipeline on: %q\n\n", prompt)

	opts := &runner.RunOptions{
		Input:    prompt,
		MaxTurns: 5,
		RunConfig: &runner.RunConfig{
			Model:         "gpt-4.1-mini",
			ModelProvider: provider,
		},
	}

	result, err := nr.RunNetwork(context.Background(), cfg, opts)
	if err != nil {
		log.Fatalf("Network run failed: %v", err)
	}

	// Print each stage of the pipeline.
	fmt.Println("=== Pipeline Stages ===")
	for i, ar := range result.AgentResults {
		fmt.Printf("\n--- Stage %d: %s (%s) ---\n", i+1, ar.AgentName, ar.Role)
		if ar.Error != nil {
			fmt.Printf("ERROR: %v\n", ar.Error)
		} else if ar.RunResult != nil {
			fmt.Println(ar.RunResult.FinalOutput)
		}
	}

	// The synthesised answer incorporates all pipeline stages.
	fmt.Println("\n=== Final Synthesis ===")
	fmt.Println(result.FinalOutput)
}
