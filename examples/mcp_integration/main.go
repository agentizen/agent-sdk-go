package main

import (
	"fmt"
	"time"

	agentsdk "github.com/agentizen/agent-sdk-go"
)

func main() {
	fmt.Println("=== MCP Integration Example ===")

	// --- 1. Create an MCP HTTP client ---
	// The client handles transport for communicating with MCP servers.
	// AllowHTTP is set to true here for illustration (development only).
	mcpClient := agentsdk.NewMCPHTTPClient(agentsdk.MCPClientOptions{
		AllowHTTP:        true,
		MaxResponseBytes: 5 * 1024 * 1024, // 5MB
		DefaultTimeout:   15 * time.Second,
	})

	fmt.Println("MCP HTTP client created:")
	fmt.Println("  AllowHTTP:        true (dev only)")
	fmt.Println("  MaxResponseBytes: 5MB")
	fmt.Println("  DefaultTimeout:   15s")

	// --- 2. Define MCP server configurations ---
	// Each ServerConfig describes one MCP server. The Client field carries the
	// transport implementation, so agents can mix HTTP, stdio, and other transports.
	githubServer := agentsdk.MCPServerConfig{
		Handle:      "github",
		URL:         "https://mcp.github.example.com",
		Description: "GitHub MCP server for repository operations",
		Headers: map[string]string{
			"Authorization": "Bearer gh_mock_token",
		},
		Timeout: 30 * time.Second,
		Client:  mcpClient,
	}

	stripeServer := agentsdk.MCPServerConfig{
		Handle:      "stripe",
		URL:         "https://mcp.stripe.example.com",
		Description: "Stripe MCP server for billing operations",
		Headers: map[string]string{
			"Authorization": "Bearer sk_mock_token",
		},
		Timeout: 20 * time.Second,
		Client:  mcpClient,
	}

	fmt.Println("\nMCP server configurations:")
	for _, cfg := range []agentsdk.MCPServerConfig{githubServer, stripeServer} {
		fmt.Printf("  - %s: %s (%s)\n", cfg.Handle, cfg.Description, cfg.URL)
	}

	// --- 3. Optionally register servers in an MCP registry ---
	mcpRegistry := agentsdk.NewMCPRegistry()
	if err := mcpRegistry.Register(githubServer); err != nil {
		fmt.Printf("Failed to register github server: %v\n", err)
	}
	if err := mcpRegistry.Register(stripeServer); err != nil {
		fmt.Printf("Failed to register stripe server: %v\n", err)
	}

	fmt.Println("\nMCP registry contents:")
	for _, cfg := range mcpRegistry.All() {
		fmt.Printf("  - %s (%s)\n", cfg.Handle, cfg.URL)
	}

	// --- 4. Create an agent with MCP servers attached ---
	assistant := agentsdk.NewAgent("DevOps Assistant")
	assistant.SetSystemInstructions("You are a DevOps assistant with access to GitHub and Stripe.")
	assistant.WithMCPServers(githubServer, stripeServer)

	fmt.Println("\nAgent configured:")
	fmt.Printf("  Name:        %s\n", assistant.Name)
	fmt.Printf("  MCP Servers: %d\n", len(assistant.MCPServers))
	for _, srv := range assistant.MCPServers {
		fmt.Printf("    - %s: %s\n", srv.Handle, srv.Description)
	}

	// --- 5. Explain the runtime behavior ---
	fmt.Println("\nRuntime behavior:")
	fmt.Println("  When the agent runs, the Runner discovers tools from each MCP server")
	fmt.Println("  by calling ListTools. Each remote tool is adapted into a standard")
	fmt.Println("  tool.Tool via ToolAdapter, so the LLM can invoke them like any other tool.")
	fmt.Println("  Headers (including auth tokens) are sent with every request automatically.")
}
