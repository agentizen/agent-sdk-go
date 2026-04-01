# Tools, Skills, MCP, and Plugins

This document covers the four extensibility mechanisms in the Agent SDK: **Tools**, **Skills**, **MCP (Model Context Protocol)**, and **Plugins**. Each mechanism solves a different problem, and they compose together through the `Agent` type.

## Overview

| Mechanism | Purpose | Granularity | Lifecycle |
|-----------|---------|-------------|-----------|
| **Tool** | Give an agent executable capabilities | Single function or process | Synchronous call during turn |
| **Skill** | Give an agent domain knowledge | Markdown document with frontmatter | Lazy-loaded on demand by LLM |
| **MCP** | Connect an agent to remote tool servers | Server config + HTTP transport | Tools discovered at runtime |
| **Plugin** | Bundle tools, skills, and MCP configs | Composite unit | Initialized once on registration |

## Tools

A tool is anything that implements the `tool.Tool` interface:

```go
type Tool interface {
    GetName() string
    GetDescription() string
    GetParametersSchema() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}
```

The SDK provides three concrete tool types:

### FunctionTool

Wraps an arbitrary Go function:

```go
t := agentsdk.NewFunctionTool(
    "get_weather",
    "Get current weather for a city",
    func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
        city, _ := params["city"].(string)
        return fmt.Sprintf("Sunny in %s", city), nil
    },
).WithSchema(map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "city": map[string]interface{}{"type": "string"},
    },
    "required": []string{"city"},
})
```

### ExecutableTool

Runs an external process. Parameters are JSON-serialized to stdin; stdout is the result:

```go
t := agentsdk.NewExecutableTool(
    "lint",
    "Run golangci-lint on a file",
    "golangci-lint",
    []string{"run", "--fix"},
).WithTimeout(30 * time.Second).
  WithWorkDir("/path/to/project")
```

### Tool Registry

A thread-safe container for managing tools and tool groups:

```go
registry := agentsdk.NewToolRegistry()
registry.Register(addTool)
registry.RegisterGroup("math", addTool, multiplyTool)

// Retrieve a group and attach to an agent
agent.WithTools(registry.GetGroup("math")...)
```

### Tool Middleware

Middleware wraps a tool to add cross-cutting concerns (logging, validation, retries, rate limiting). Use `tool.WrapExecute` (from `github.com/agentizen/agent-sdk-go/pkg/tool`) to wrap a tool's `Execute` while preserving its original identity (name, description, schema):

```go
import "github.com/agentizen/agent-sdk-go/pkg/tool"

func loggingMiddleware() agentsdk.ToolMiddleware {
    return func(next agentsdk.Tool) agentsdk.Tool {
        return tool.WrapExecute(next, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
            log.Printf("Calling %s", next.GetName())
            return next.Execute(ctx, params)
        })
    }
}

wrapped := agentsdk.WithToolMiddleware(myTool, loggingMiddleware())
```

`WrapExecute` returns a new `Tool` that delegates `GetName`, `GetDescription`, and `GetParametersSchema` to the inner tool while replacing `Execute` with the provided function. This is the recommended approach over manually constructing a `FunctionTool`, which would discard the original schema.

Middlewares are applied in order: the first in the list is the outermost wrapper.

## Skills

Skills are markdown documents with YAML frontmatter. They provide domain knowledge to agents without bloating the initial system prompt.

### Skill Format

```markdown
---
name: code-review
description: Performs a thorough code review
version: "1.0.0"
---
# Code Review Skill

When reviewing code, check for:
1. Correctness
2. Style adherence
3. Performance issues
4. Security vulnerabilities
```

### Loading Skills

```go
// From a file
s, err := agentsdk.LoadSkillFromFile("skills/code-review.md")

// From a string
s, err := agentsdk.LoadSkillFromString(markdownContent)

// From an io.Reader
s, err := agentsdk.LoadSkillFromReader(reader)
```

### Skill Registry

```go
registry := agentsdk.NewSkillRegistry()
registry.Register(codeReviewSkill)
registry.Register(testWriterSkill)

// List all registered skill names
names := registry.Names()
```

### How Skills Integrate with the Agent

When an agent has skills attached, the Runner injects skill headers (name, description, version) into the system prompt so the LLM knows what knowledge is available. A `load_skill` tool is automatically added to the agent. The LLM can invoke this tool to load the full content of any skill on demand.

This two-phase approach (headers first, full content on demand) keeps the initial prompt small while making all skills discoverable.

```go
agent := agentsdk.NewAgent("Assistant")
agent.WithSkills(codeReviewSkill, testWriterSkill)
// At runtime:
//   1. System prompt includes skill headers
//   2. LLM sees "load_skill" tool with enum of available names
//   3. LLM calls load_skill("code-review") to get full content
```

## MCP (Model Context Protocol)

MCP connects agents to remote tool servers over HTTP. Each MCP server exposes a set of tools that the agent can discover and invoke at runtime.

### Architecture

```
Agent
  |
  +-- MCPServerConfig (handle: "github")
  |     |-- URL: https://mcp.github.example.com/mcp
  |     |-- Headers: {Authorization: Bearer ...}
  |     +-- Client: MCPHTTPClient
  |           |-- POST {URL} with {"tool":"...", "params":{...}}  (Execute)
  |           +-- POST {URL} with {"action":"list_tools"}         (ListTools)
  |
  +-- MCPServerConfig (handle: "stripe")
        |-- URL: https://mcp.stripe.example.com/api/mcp
        +-- Client: MCPHTTPClient
```

The URL in `ServerConfig` is used as-is — the client does not append any path segments. Configure the full endpoint URL including any path in `ServerConfig.URL`.

Each `ServerConfig` carries its own `Client` implementation, so agents can mix HTTP, stdio, and other transports within the same agent.

> **Note:** `ServerConfig.Client` must not be nil. If `Execute` or `ListTools` is called on a server with a nil `Client`, the SDK returns an error immediately.

### Creating an MCP Client

```go
client := agentsdk.NewMCPHTTPClient(agentsdk.MCPClientOptions{
    AllowHTTP:        false,          // reject http:// in production
    MaxResponseBytes: 10 * 1024 * 1024, // 10MB
    DefaultTimeout:   30 * time.Second,
})
```

### Configuring MCP Servers

```go
githubServer := agentsdk.MCPServerConfig{
    Handle:      "github",
    URL:         "https://mcp.github.example.com",
    Description: "GitHub operations",
    Headers:     map[string]string{"Authorization": "Bearer " + token},
    Timeout:     30 * time.Second,
    Client:      client,
}

agent := agentsdk.NewAgent("DevOps Assistant")
agent.WithMCPServers(githubServer)
```

### MCP Registry

For centralized server management:

```go
registry := agentsdk.NewMCPRegistry()
registry.Register(githubServer)
registry.Register(stripeServer)

// Discover all tools from a specific server
tools, err := registry.ToolsFor(ctx, "github")

// Discover tools from all registered servers
allTools, err := registry.AllTools(ctx)
```

### How MCP Integrates with the Agent

At runtime, the Runner calls `ListTools` on each MCP server to discover available tools. Each remote tool is wrapped via `ToolAdapter` into a standard `tool.Tool`. When the LLM invokes an MCP tool, the Runner calls `Execute` on the appropriate server, forwarding parameters as JSON and returning the response.

Context-level headers and user IDs can be injected via `mcp.WithHeaders(ctx, headers)` and `mcp.WithUserID(ctx, userID)`.

## Plugins

A plugin is a bundle of tools, skills, and MCP server configurations packaged as a single unit. Plugins implement the `Plugin` interface:

```go
type Plugin interface {
    Name() string
    Description() string
    Version() string
    Tools() []tool.Tool
    Skills() []skill.Skill
    MCPServers() []mcp.ServerConfig
    Init(ctx context.Context) error
}
```

### Creating a Plugin

Embed `BasePlugin` and override fields or methods as needed:

```go
type myPlugin struct {
    agentsdk.BasePlugin
}

func newMyPlugin() *myPlugin {
    return &myPlugin{
        BasePlugin: agentsdk.BasePlugin{
            PluginName:        "my-plugin",
            PluginDescription: "Does useful things",
            PluginVersion:     "1.0.0",
            PluginTools:       []agentsdk.Tool{myTool},
            PluginSkills:      []agentsdk.Skill{mySkill},
            PluginMCPServers:  []agentsdk.MCPServerConfig{myServer},
        },
    }
}

func (p *myPlugin) Init(ctx context.Context) error {
    // Validate API keys, warm caches, etc.
    return nil
}
```

### Registering Plugins to an Agent

`WithPlugins` calls `Init(context.Background())` on each plugin directly and fires the `OnPluginInit` hook. It then merges the plugin's components into the agent. Use `WithPluginsContext` if you need to pass a custom context (for example, one carrying deadlines or tracing spans):

```go
agent := agentsdk.NewAgent("Assistant")
agent.WithPlugins(myPlugin)
// 1. myPlugin.Init(context.Background()) is called
// 2. OnPluginInit hook fires
// 3. myPlugin.Tools()      -> merged into agent.Tools
//    myPlugin.Skills()     -> merged into agent.Skills
//    myPlugin.MCPServers() -> merged into agent.MCPServers
```

### Plugin Registry

For managing plugins across multiple agents:

```go
registry := agentsdk.NewPluginRegistry()
err := registry.Register(ctx, myPlugin)  // calls Init
plugin, ok := registry.Get("my-plugin")
all := registry.All()
```

## Composition

All four mechanisms compose naturally on the `Agent` type:

```go
agent := agentsdk.NewAgent("Full-Featured Assistant")
agent.SetSystemInstructions("You are a versatile assistant.")

// Direct tools
agent.WithTools(calculatorTool, weatherTool)

// Skills for domain knowledge
agent.WithSkills(codeReviewSkill, dataAnalysisSkill)

// Remote MCP servers
agent.WithMCPServers(githubServer, stripeServer)

// Plugins that bundle all three
agent.WithPlugins(analyticsPlugin, monitoringPlugin)
```

## Hooks and Tracing Integration

Lifecycle hooks (`agent.Hooks`) fire at key points during execution. Tools, skills, MCP calls, and plugins all participate in the hook and tracing lifecycle:

- `OnToolStart` / `OnToolEnd` fire for every tool call, including MCP-adapted tools and the `load_skill` tool.
- `OnAgentStart` / `OnAgentEnd` fire at the agent level, after all tools and skills are wired up.
- `OnPluginInit` fires when a plugin is initialized via `WithPlugins` or `PluginRegistry.Register`.
- `OnSkillLoad` fires when a skill's full content is loaded on demand by the LLM.
- `OnBeforeMCPCall` / `OnAfterMCPCall` fire around each MCP tool invocation, allowing inspection or modification of requests and responses.
- Tracing configuration (`runner.TracingConfig`) propagates trace IDs through tool calls and MCP requests via context.

## Examples

Working examples for each mechanism are in the `examples/` directory:

| Example | Path |
|---------|------|
| Tool Registry | `examples/tool_registry/main.go` |
| Skill Loading | `examples/skill_loading/main.go` |
| MCP Integration | `examples/mcp_integration/main.go` |
| Plugin Bundle | `examples/plugin_bundle/main.go` |

Run any example with:

```bash
go run ./examples/tool_registry/
```
