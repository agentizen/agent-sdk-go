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

// Competitive Network example
//
// Two agents — one using a powerful model and one using a faster model — race
// to answer the same prompt. The first agent to respond without error wins,
// and the other is cancelled. This is useful for latency-sensitive workloads
// where you want the fastest acceptable answer.

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	provider := openai.NewProvider(apiKey)

	// Two agents with the same instructions but different model preferences.
	// Both receive the same prompt; the faster one wins.
	powerAgent := agent.NewAgent("PowerAgent").
		WithModel("gpt-4.1").
		SetSystemInstructions("You are a highly accurate assistant. Provide a thorough, well-reasoned answer.")

	speedAgent := agent.NewAgent("SpeedAgent").
		WithModel("gpt-4.1-mini").
		SetSystemInstructions("You are a fast, concise assistant. Provide a direct, accurate answer quickly.")

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: powerAgent, Role: "High-accuracy model"},
			network.AgentSlot{Agent: speedAgent, Role: "Low-latency model"},
		).
		WithStrategy(network.StrategyCompetitive) // First to respond wins.

	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	prompt := "What are the key differences between PostgreSQL and MySQL for a high-traffic web application?"

	fmt.Printf("Running competitive race on: %q\n\n", prompt)
	fmt.Println("Two agents are competing — fastest non-error response wins...")

	opts := &runner.RunOptions{
		Input:    prompt,
		MaxTurns: 3,
		RunConfig: &runner.RunConfig{
			ModelProvider: provider,
		},
	}

	result, err := nr.RunNetwork(context.Background(), cfg, opts)
	if err != nil {
		log.Fatalf("Network run failed: %v", err)
	}

	// Only one agent result in competitive mode.
	if len(result.AgentResults) > 0 {
		winner := result.AgentResults[0]
		fmt.Printf("Winner: %s (%s) — responded in %dms\n\n",
			winner.AgentName, winner.Role, winner.Duration.Milliseconds())
	}

	fmt.Println("=== Winning Response ===")
	fmt.Println(result.FinalOutput)
}
