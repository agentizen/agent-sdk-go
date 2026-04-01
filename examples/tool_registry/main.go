package main

import (
	"context"
	"fmt"
	"log"
	"time"

	agentsdk "github.com/agentizen/agent-sdk-go"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

// loggingMiddleware returns a middleware that logs tool invocations.
// It uses tool.WrapExecute to replace Execute while preserving the
// original tool's identity (name, description, schema).
func loggingMiddleware() agentsdk.ToolMiddleware {
	return func(next agentsdk.Tool) agentsdk.Tool {
		return tool.WrapExecute(next, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			fmt.Printf("[LOG] Calling tool %q with params: %v\n", next.GetName(), params)
			start := time.Now()
			result, err := next.Execute(ctx, params)
			fmt.Printf("[LOG] Tool %q completed in %v\n", next.GetName(), time.Since(start))
			return result, err
		})
	}
}

func main() {
	// --- 1. Create a tool registry ---
	registry := agentsdk.NewToolRegistry()
	fmt.Println("=== Tool Registry Example ===")

	// --- 2. Register individual function tools ---
	addTool := agentsdk.NewFunctionTool(
		"add",
		"Add two numbers together",
		func(_ context.Context, params map[string]interface{}) (interface{}, error) {
			a, _ := params["a"].(float64)
			b, _ := params["b"].(float64)
			return a + b, nil
		},
	).WithSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number", "description": "First operand"},
			"b": map[string]interface{}{"type": "number", "description": "Second operand"},
		},
		"required": []string{"a", "b"},
	})

	multiplyTool := agentsdk.NewFunctionTool(
		"multiply",
		"Multiply two numbers",
		func(_ context.Context, params map[string]interface{}) (interface{}, error) {
			a, _ := params["a"].(float64)
			b, _ := params["b"].(float64)
			return a * b, nil
		},
	).WithSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number", "description": "First operand"},
			"b": map[string]interface{}{"type": "number", "description": "Second operand"},
		},
		"required": []string{"a", "b"},
	})

	if err := registry.Register(addTool); err != nil {
		log.Fatalf("Failed to register add tool: %v", err)
	}
	if err := registry.Register(multiplyTool); err != nil {
		log.Fatalf("Failed to register multiply tool: %v", err)
	}

	fmt.Println("Registered individual tools:")
	for _, name := range registry.Names() {
		fmt.Printf("  - %s\n", name)
	}

	// --- 3. Register a group of tools ---
	uppercaseTool := agentsdk.NewFunctionTool(
		"uppercase",
		"Convert text to uppercase",
		func(_ context.Context, params map[string]interface{}) (interface{}, error) {
			text, _ := params["text"].(string)
			return fmt.Sprintf("%s (uppercased)", text), nil
		},
	)

	lowercaseTool := agentsdk.NewFunctionTool(
		"lowercase",
		"Convert text to lowercase",
		func(_ context.Context, params map[string]interface{}) (interface{}, error) {
			text, _ := params["text"].(string)
			return fmt.Sprintf("%s (lowercased)", text), nil
		},
	)

	if err := registry.RegisterGroup("text_tools", uppercaseTool, lowercaseTool); err != nil {
		log.Fatalf("Failed to register text_tools group: %v", err)
	}

	fmt.Println("\nRegistered group 'text_tools':")
	for _, t := range registry.GetGroup("text_tools") {
		fmt.Printf("  - %s: %s\n", t.GetName(), t.GetDescription())
	}

	// --- 4. Create an ExecutableTool (runs an external command) ---
	echoTool := agentsdk.NewExecutableTool(
		"echo",
		"Echo a message using the system echo command",
		"echo",
		[]string{"hello from executable tool"},
	).WithTimeout(5 * time.Second)

	fmt.Printf("\nExecutableTool created: %s (%s)\n", echoTool.GetName(), echoTool.GetDescription())

	// --- 5. Apply logging middleware to a tool ---
	wrappedAdd := agentsdk.WithToolMiddleware(addTool, loggingMiddleware())
	fmt.Printf("\nWrapped tool with logging middleware: %s\n", wrappedAdd.GetName())

	// Demonstrate the middleware by executing the wrapped tool
	result, err := wrappedAdd.Execute(context.Background(), map[string]interface{}{
		"a": float64(3),
		"b": float64(7),
	})
	if err != nil {
		log.Fatalf("Tool execution failed: %v", err)
	}
	fmt.Printf("Result: %v\n", result)

	// --- 6. Create an agent with tools from the registry group ---
	assistant := agentsdk.NewAgent("Math & Text Assistant")
	assistant.SetSystemInstructions("You are a helpful assistant with math and text tools.")

	// Add all tools from the "text_tools" group
	assistant.WithTools(registry.GetGroup("text_tools")...)

	// Add individually registered tools
	assistant.WithTools(addTool, multiplyTool)

	// Add the executable tool
	assistant.WithTools(echoTool)

	fmt.Println("\nAgent configured:")
	fmt.Printf("  Name: %s\n", assistant.Name)
	fmt.Printf("  Tools: %d\n", len(assistant.Tools))
	for _, t := range assistant.Tools {
		fmt.Printf("    - %s: %s\n", t.GetName(), t.GetDescription())
	}

	fmt.Println("\nAll registered tools in registry:", registry.Names())
}
