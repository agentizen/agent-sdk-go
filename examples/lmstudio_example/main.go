// Package main demonstrates the agent-sdk-go using a local LM Studio server.
//
// Prerequisites:
//   - LM Studio running on http://localhost:1234 (or set LMSTUDIO_BASE_URL)
//   - A model loaded in LM Studio (default: "gemma-3-4b-it")
//
// Run with:
//
//	go run ./examples/lmstudio_example
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/model/providers/lmstudio"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

func main() {
	// Allow overriding the base URL and model via environment variables.
	baseURL := os.Getenv("LMSTUDIO_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:1234/v1"
	}
	modelName := os.Getenv("LMSTUDIO_MODEL")
	if modelName == "" {
		modelName = "gemma-3-4b-it"
	}

	// Create and configure the LM Studio provider.
	provider := lmstudio.NewProvider()
	provider.SetBaseURL(baseURL)
	provider.SetDefaultModel(modelName)

	// Create the time-assistant agent.
	timeAgent := agent.NewAgent("Time Assistant")
	timeAgent.SetModelProvider(provider)
	timeAgent.WithModel(modelName)
	timeAgent.SetSystemInstructions(`You are a helpful time assistant that can provide the current time in various formats.
When a user asks for the time, use the get_current_time tool to get accurate information.
After using tools, ALWAYS provide a complete response to the user's question in natural language.
Make your responses helpful and to the point.`)

	// Add a tool that returns the current time in a requested format.
	timeAgent.WithTools(tool.NewFunctionTool(
		"get_current_time",
		"Get the current time in a specified format",
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			format := time.RFC3339

			if formatParam, ok := params["format"].(string); ok && formatParam != "" {
				switch formatParam {
				case "rfc3339":
					format = time.RFC3339
				case "kitchen":
					format = time.Kitchen
				case "date":
					format = "2006-01-02"
				case "datetime":
					format = "2006-01-02 15:04:05"
				case "unix":
					return time.Now().Unix(), nil
				}
			}

			return time.Now().Format(format), nil
		},
	).WithSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"format": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"rfc3339", "kitchen", "date", "datetime", "unix"},
				"description": "The format to return the time in. Options: rfc3339, kitchen, date, datetime, unix",
			},
		},
		"required": []string{},
	}))

	r := runner.NewRunner()
	r.WithDefaultProvider(provider)

	ctx := context.Background()

	// --- Basic query ---
	fmt.Println("Running agent with a basic query...")
	result, err := r.Run(ctx, timeAgent, &runner.RunOptions{
		Input: "Hi! Could you please tell me what time it is right now?",
	})
	if err != nil {
		log.Fatalf("Error running agent: %v", err)
	}
	fmt.Println("\nAgent response:")
	fmt.Println(result.FinalOutput)

	// --- Format-specific query ---
	fmt.Println("\nRunning agent with a specific format request...")
	result, err = r.Run(ctx, timeAgent, &runner.RunOptions{
		Input: "What time is it in kitchen format?",
	})
	if err != nil {
		log.Fatalf("Error running agent: %v", err)
	}
	fmt.Println("\nAgent response:")
	fmt.Println(result.FinalOutput)

	// --- Streaming ---
	fmt.Println("\nRunning agent with streaming...")
	streamResult, err := r.RunStreaming(ctx, timeAgent, &runner.RunOptions{
		Input: "What time is it right now?",
	})
	if err != nil {
		log.Fatalf("Error running agent with streaming: %v", err)
	}

	fmt.Println("\nStreaming response:")
	for event := range streamResult.Stream {
		switch event.Type {
		case model.StreamEventTypeContent:
			fmt.Print(event.Content)
		case model.StreamEventTypeToolCall:
			fmt.Printf("\n[Tool Call: %s]\n", event.ToolCall.Name)
		case model.StreamEventTypeDone:
			fmt.Println("\n[Done]")
		case model.StreamEventTypeError:
			fmt.Printf("\n[Error: %v]\n", event.Error)
			os.Exit(1)
		}
	}

	fmt.Println("\nFinal output:")
	fmt.Println(streamResult.FinalOutput)
}
