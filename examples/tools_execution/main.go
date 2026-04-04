// Package main demonstrates end-to-end tool execution using the SDK.
//
// It tests all tool types supported by the SDK:
//   - FunctionTool: in-process Go function
//   - ExecutableTool: external command subprocess
//   - Tool middleware: wrapping tools with cross-cutting concerns
//   - Tool registry: registration, grouping, and retrieval
//
// Run: go run ./examples/tools_execution
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	agentsdk "github.com/agentizen/agent-sdk-go"
	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

func main() {
	fmt.Println("=== Tools Execution Example ===")
	fmt.Println("This example executes all tool types end-to-end.")
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

	// --- 1. FunctionTool: basic execution ---
	fmt.Println("--- FunctionTool ---")

	addTool := agentsdk.NewFunctionTool(
		"add",
		"Add two numbers",
		func(_ context.Context, params map[string]interface{}) (interface{}, error) {
			a, _ := params["a"].(float64)
			b, _ := params["b"].(float64)
			return a + b, nil
		},
	).WithSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	})

	result, err := addTool.Execute(ctx, map[string]interface{}{"a": 3.0, "b": 7.0})
	if err == nil && result.(float64) != 10.0 {
		err = fmt.Errorf("expected 10, got %v", result)
	}
	check("FunctionTool: add(3, 7) = 10", err)
	fmt.Printf("           Result: %v\n", result)

	// Verify metadata
	if addTool.GetName() != "add" {
		check("FunctionTool: GetName()", fmt.Errorf("expected 'add', got %q", addTool.GetName()))
	} else {
		check("FunctionTool: GetName()", nil)
	}
	if addTool.GetDescription() != "Add two numbers" {
		check("FunctionTool: GetDescription()", fmt.Errorf("unexpected description"))
	} else {
		check("FunctionTool: GetDescription()", nil)
	}
	if addTool.GetParametersSchema() == nil {
		check("FunctionTool: GetParametersSchema()", fmt.Errorf("schema is nil"))
	} else {
		check("FunctionTool: GetParametersSchema()", nil)
	}

	// --- 2. FunctionTool: error handling ---
	failTool := agentsdk.NewFunctionTool(
		"fail",
		"Always fails",
		func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("intentional error")
		},
	)

	_, err = failTool.Execute(ctx, nil)
	if err == nil {
		check("FunctionTool: error propagation", fmt.Errorf("expected error, got nil"))
	} else if !strings.Contains(err.Error(), "intentional error") {
		check("FunctionTool: error propagation", fmt.Errorf("unexpected error: %v", err))
	} else {
		check("FunctionTool: error propagation", nil)
	}

	// --- 3. ExecutableTool: run external command ---
	fmt.Println("\n--- ExecutableTool ---")

	echoTool := agentsdk.NewExecutableTool(
		"echo_test",
		"Echo a test message",
		"echo",
		[]string{"hello-from-executable"},
	).WithTimeout(5 * time.Second)

	result, err = echoTool.Execute(ctx, nil)
	if err == nil {
		output := strings.TrimSpace(fmt.Sprintf("%v", result))
		if !strings.Contains(output, "hello-from-executable") {
			err = fmt.Errorf("expected 'hello-from-executable' in output, got %q", output)
		}
	}
	check("ExecutableTool: echo", err)
	fmt.Printf("           Result: %v\n", strings.TrimSpace(fmt.Sprintf("%v", result)))

	// --- 4. Tool middleware ---
	fmt.Println("\n--- Tool Middleware ---")

	var middlewareCalled bool
	loggingMiddleware := func(next agentsdk.Tool) agentsdk.Tool {
		return tool.WrapExecute(next, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			middlewareCalled = true
			return next.Execute(ctx, params)
		})
	}

	wrappedAdd := agentsdk.WithToolMiddleware(addTool, loggingMiddleware)
	result, err = wrappedAdd.Execute(ctx, map[string]interface{}{"a": 5.0, "b": 5.0})
	if err == nil && !middlewareCalled {
		err = fmt.Errorf("middleware was not called")
	}
	if err == nil && result.(float64) != 10.0 {
		err = fmt.Errorf("expected 10, got %v", result)
	}
	check("Middleware: intercepts execution", err)

	// Verify middleware preserves identity
	if wrappedAdd.GetName() != "add" {
		check("Middleware: preserves tool name", fmt.Errorf("expected 'add', got %q", wrappedAdd.GetName()))
	} else {
		check("Middleware: preserves tool name", nil)
	}

	// --- 5. Tool registry ---
	fmt.Println("\n--- Tool Registry ---")

	registry := agentsdk.NewToolRegistry()

	err = registry.Register(addTool)
	check("Registry: register 'add'", err)

	multiplyTool := agentsdk.NewFunctionTool("multiply", "Multiply", func(_ context.Context, p map[string]interface{}) (interface{}, error) {
		return p["a"].(float64) * p["b"].(float64), nil
	})
	err = registry.Register(multiplyTool)
	check("Registry: register 'multiply'", err)

	// Duplicate registration should fail
	err = registry.Register(addTool)
	if err == nil {
		check("Registry: rejects duplicate", fmt.Errorf("expected error for duplicate"))
	} else {
		check("Registry: rejects duplicate", nil)
	}

	// Retrieve by name
	got, ok := registry.Get("add")
	if !ok || got.GetName() != "add" {
		check("Registry: Get('add')", fmt.Errorf("not found"))
	} else {
		check("Registry: Get('add')", nil)
	}

	// List names
	names := registry.Names()
	if len(names) != 2 {
		check("Registry: Names()", fmt.Errorf("expected 2 names, got %d", len(names)))
	} else {
		check("Registry: Names()", nil)
	}

	// Group registration
	upperTool := agentsdk.NewFunctionTool("upper", "Uppercase", func(_ context.Context, p map[string]interface{}) (interface{}, error) {
		return strings.ToUpper(p["text"].(string)), nil
	})
	lowerTool := agentsdk.NewFunctionTool("lower", "Lowercase", func(_ context.Context, p map[string]interface{}) (interface{}, error) {
		return strings.ToLower(p["text"].(string)), nil
	})

	err = registry.RegisterGroup("text", upperTool, lowerTool)
	check("Registry: RegisterGroup('text')", err)

	group := registry.GetGroup("text")
	if len(group) != 2 {
		check("Registry: GetGroup('text')", fmt.Errorf("expected 2 tools, got %d", len(group)))
	} else {
		check("Registry: GetGroup('text')", nil)
	}

	// Execute from group
	result, err = group[0].Execute(ctx, map[string]interface{}{"text": "hello"})
	check("Registry: execute grouped tool", err)
	fmt.Printf("           Result: %v\n", result)

	// --- 6. Agent integration ---
	fmt.Println("\n--- Agent Integration ---")

	assistant := agentsdk.NewAgent("Tool Test Agent")
	assistant.WithTools(addTool, multiplyTool, echoTool)
	assistant.WithTools(registry.GetGroup("text")...)

	if len(assistant.Tools) != 5 {
		check("Agent: tool count", fmt.Errorf("expected 5 tools, got %d", len(assistant.Tools)))
	} else {
		check("Agent: tool count = 5", nil)
	}

	// --- Summary ---
	fmt.Printf("\n=== Results: %d passed, %d failed ===\n", passed, failed)
	if failed > 0 {
		log.Fatalf("%d tests failed", failed)
	}
}
