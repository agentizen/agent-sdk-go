package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/model/providers/openai"
	"github.com/agentizen/agent-sdk-go/pkg/network"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
)

// Parallel Network example
//
// Three specialised agents — Researcher, Analyst, and Writer — work simultaneously
// on different aspects of the same prompt. A built-in orchestrator decomposes the
// prompt into per-agent sub-tasks, runs all agents concurrently, and synthesises
// a final consolidated answer.

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create a shared provider.
	provider := openai.NewProvider(apiKey)
	provider.SetDefaultModel("gpt-4.1-mini")

	// Define the three specialised agents.
	researcher := agent.NewAgent("Researcher").
		WithModel("gpt-4.1-mini").
		SetSystemInstructions("You are a research specialist. When given a sub-task, conduct focused research and return concise, factual findings.")

	analyst := agent.NewAgent("Analyst").
		WithModel("gpt-4.1-mini").
		SetSystemInstructions("You are a data analyst. When given a sub-task, analyse the topic from a quantitative and strategic perspective and return key insights.")

	writer := agent.NewAgent("Writer").
		WithModel("gpt-4.1-mini").
		SetSystemInstructions("You are a professional writer. When given a sub-task, craft clear, engaging prose that communicates the topic to a business audience.")

	// Build the network configuration.
	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: researcher, Role: "Research specialist"},
			network.AgentSlot{Agent: analyst, Role: "Strategic analyst"},
			network.AgentSlot{Agent: writer, Role: "Content writer"},
		).
		WithStrategy(network.StrategyParallel). // All agents run concurrently.
		WithMaxConcurrency(3)                   // Allow all 3 to run simultaneously.

	// Create the network runner backed by a standard runner.
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	prompt := "The impact of artificial intelligence on the future of work"

	fmt.Printf("Running parallel agent network on: %q\n\n", prompt)

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

	// Print per-agent results.
	fmt.Println("=== Per-Agent Results ===")
	for _, ar := range result.AgentResults {
		fmt.Printf("\n[%s — %s] (%.0fms)\n", ar.AgentName, ar.Role, float64(ar.Duration.Milliseconds()))
		if ar.Error != nil {
			fmt.Printf("  ERROR: %v\n", ar.Error)
		} else if ar.RunResult != nil {
			fmt.Printf("  %v\n", ar.RunResult.FinalOutput)
		}
	}

	// Print synthesised final answer.
	fmt.Println("\n=== Final Synthesised Answer ===")
	fmt.Println(result.FinalOutput)
	fmt.Printf("\nLast agent: %s | Strategy: %s\n", result.LastAgent.Name, result.Strategy)
}
