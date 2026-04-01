package main

import (
	"context"
	"fmt"
	"log"
	"time"

	agentsdk "github.com/agentizen/agent-sdk-go"
)

// analyticsPlugin is a custom plugin that bundles tools, skills, and MCP
// servers into a single unit. It embeds BasePlugin for default implementations.
type analyticsPlugin struct {
	agentsdk.BasePlugin
}

// newAnalyticsPlugin builds the plugin with all its components.
func newAnalyticsPlugin() *analyticsPlugin {
	// Create tools for the plugin
	queryTool := agentsdk.NewFunctionTool(
		"run_query",
		"Run an analytics SQL query",
		func(_ context.Context, params map[string]interface{}) (interface{}, error) {
			query, _ := params["sql"].(string)
			return fmt.Sprintf("Query executed: %s (mock result: 42 rows)", query), nil
		},
	).WithSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sql": map[string]interface{}{
				"type":        "string",
				"description": "SQL query to execute",
			},
		},
		"required": []string{"sql"},
	})

	chartTool := agentsdk.NewFunctionTool(
		"generate_chart",
		"Generate a chart from query results",
		func(_ context.Context, params map[string]interface{}) (interface{}, error) {
			chartType, _ := params["type"].(string)
			return fmt.Sprintf("Chart generated: %s (mock PNG)", chartType), nil
		},
	)

	// Load a skill for the plugin
	skillContent := `---
name: data-analysis
description: Guides the agent through data analysis best practices
version: "1.0.0"
---
# Data Analysis Skill

When analyzing data:
1. Understand the question being asked.
2. Identify relevant data sources.
3. Write efficient queries.
4. Visualize results clearly.
5. Summarize findings in plain language.
`
	dataSkill, err := agentsdk.LoadSkillFromString(skillContent)
	if err != nil {
		log.Fatalf("Failed to load skill: %v", err)
	}

	// Create an MCP HTTP client for the plugin's MCP server
	mcpClient := agentsdk.NewMCPHTTPClient(agentsdk.MCPClientOptions{
		AllowHTTP:      true,
		DefaultTimeout: 10 * time.Second,
	})

	return &analyticsPlugin{
		BasePlugin: agentsdk.BasePlugin{
			PluginName:        "analytics",
			PluginDescription: "Analytics plugin with query, charting, and data analysis capabilities",
			PluginVersion:     "1.2.0",
			PluginTools:       []agentsdk.Tool{queryTool, chartTool},
			PluginSkills:      []agentsdk.Skill{dataSkill},
			PluginMCPServers: []agentsdk.MCPServerConfig{
				{
					Handle:      "analytics-warehouse",
					URL:         "https://mcp.warehouse.example.com",
					Description: "Data warehouse MCP server",
					Client:      mcpClient,
				},
			},
		},
	}
}

// Init overrides BasePlugin.Init to perform custom setup.
func (p *analyticsPlugin) Init(_ context.Context) error {
	fmt.Printf("  [plugin:%s] Initializing v%s...\n", p.Name(), p.Version())
	// In production this might validate API keys, warm caches, etc.
	return nil
}

func main() {
	fmt.Println("=== Plugin Bundle Example ===")

	// --- 1. Create the plugin ---
	analytics := newAnalyticsPlugin()
	fmt.Printf("Plugin created: %s (v%s)\n", analytics.Name(), analytics.Version())
	fmt.Printf("  Tools:       %d\n", len(analytics.Tools()))
	fmt.Printf("  Skills:      %d\n", len(analytics.Skills()))
	fmt.Printf("  MCP Servers: %d\n", len(analytics.MCPServers()))

	// --- 2. Register the plugin to an agent via WithPlugins ---
	// WithPlugins calls Init, then merges the plugin's tools, skills, and
	// MCP servers into the agent's own configuration.
	fmt.Println("\nRegistering plugin to agent:")
	assistant := agentsdk.NewAgent("Data Analyst")
	assistant.SetSystemInstructions("You are a data analyst with access to analytics tools.")
	assistant.WithPlugins(analytics)

	// --- 3. Print the agent's merged configuration ---
	fmt.Println("\nAgent after plugin registration:")
	fmt.Printf("  Name:        %s\n", assistant.Name)
	fmt.Printf("  Plugins:     %d\n", len(assistant.Plugins))

	fmt.Printf("\n  Tools (%d):\n", len(assistant.Tools))
	for _, t := range assistant.Tools {
		fmt.Printf("    - %s: %s\n", t.GetName(), t.GetDescription())
	}

	fmt.Printf("\n  Skills (%d):\n", len(assistant.Skills))
	for _, s := range assistant.Skills {
		h := s.Header()
		fmt.Printf("    - %s (v%s): %s\n", h.Name, h.Version, h.Description)
	}

	fmt.Printf("\n  MCP Servers (%d):\n", len(assistant.MCPServers))
	for _, srv := range assistant.MCPServers {
		fmt.Printf("    - %s: %s (%s)\n", srv.Handle, srv.Description, srv.URL)
	}

	// --- 4. Optionally use a plugin registry for centralized management ---
	pluginRegistry := agentsdk.NewPluginRegistry()
	if err := pluginRegistry.Register(context.Background(), analytics); err != nil {
		log.Fatalf("Failed to register plugin: %v", err)
	}

	fmt.Println("\nPlugin registry contents:")
	for _, p := range pluginRegistry.All() {
		fmt.Printf("  - %s (v%s): %s\n", p.Name(), p.Version(), p.Description())
	}
}
