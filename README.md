<div align="center">
  <p><strong>Build and orchestrate AI agents in Go — multi-provider, multi-agent, production-ready</strong></p>
</div>

<p align="center">
  Agent SDK Go is an open-source framework for building powerful AI agents with Go. It supports multiple LLM providers, function calling, agent handoffs, structured output, streaming, and tracing.
</p>

<p align="center">
  <em>This is a maintained fork of an unmaintained upstream project, reworked under <a href="https://github.com/citizenofai">citizenofai</a> to meet our production needs.</em>
</p>

<p align="center">
    <a href="https://github.com/citizenofai/agent-sdk-go/actions/workflows/code-quality.yml"><img src="https://github.com/citizenofai/agent-sdk-go/actions/workflows/code-quality.yml/badge.svg" alt="Code Quality"></a>
    <a href="https://goreportcard.com/report/github.com/citizenofai/agent-sdk-go"><img src="https://goreportcard.com/badge/github.com/citizenofai/agent-sdk-go" alt="Go Report Card"></a>
    <a href="https://github.com/citizenofai/agent-sdk-go/blob/main/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/citizenofai/agent-sdk-go" alt="Go Version"></a>
    <a href="https://pkg.go.dev/github.com/citizenofai/agent-sdk-go"><img src="https://pkg.go.dev/badge/github.com/citizenofai/agent-sdk-go.svg" alt="PkgGoDev"></a><br>
    <a href="https://github.com/citizenofai/agent-sdk-go/actions/workflows/codeql-analysis.yml"><img src="https://github.com/citizenofai/agent-sdk-go/actions/workflows/codeql-analysis.yml/badge.svg" alt="CodeQL"></a>
    <a href="https://github.com/citizenofai/agent-sdk-go/blob/main/LICENSE"><img src="https://img.shields.io/github/license/citizenofai/agent-sdk-go" alt="License"></a>
    <a href="https://github.com/citizenofai/agent-sdk-go/stargazers"><img src="https://img.shields.io/github/stars/citizenofai/agent-sdk-go" alt="Stars"></a>
    <a href="https://github.com/citizenofai/agent-sdk-go/graphs/contributors"><img src="https://img.shields.io/github/contributors/citizenofai/agent-sdk-go" alt="Contributors"></a>
    <a href="https://github.com/citizenofai/agent-sdk-go/commits/main"><img src="https://img.shields.io/github/last-commit/citizenofai/agent-sdk-go" alt="Last Commit"></a>
</p>

---

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Provider Setup](#-provider-setup)
- [Key Components](#-key-components)
  - [Agent](#agent)
  - [Runner](#runner)
  - [Tools](#tools)
  - [Model Providers](#model-providers)
- [Advanced Features](#-advanced-features)
  - [Multi-Agent Workflows](#multi-agent-workflows)
  - [Tracing](#tracing)
  - [Structured Output](#structured-output)
  - [Streaming](#streaming)
  - [Bidirectional Agent Flow](#bidirectional-agent-flow)
- [Examples](#-examples)
- [Development](#-development)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🔍 Overview

Agent SDK Go provides a comprehensive framework for building AI agents in Go. It lets you create agents that use tools, hand off to specialized sub-agents, produce structured output, and stream responses — all while supporting a wide range of LLM providers (cloud and local).

This repository is a maintained fork of an unmaintained project, actively reworked by [citizenofai](https://github.com/citizenofai) to add new providers (Gemini, Mistral, Azure OpenAI), fix upstream issues, and keep dependencies up-to-date.

## 🌟 Features

- ✅ **Multiple LLM Providers** — OpenAI, Anthropic Claude, Google Gemini, Mistral, LM Studio (local), Azure OpenAI
- ✅ **Function Calling / Tools** — Call any Go function directly from your LLM
- ✅ **Agent Handoffs** — Build complex multi-agent pipelines with specialized agents
- ✅ **Bidirectional Flow** — Agents can delegate tasks and return results back to the caller
- ✅ **Structured Output Ready** — Design agents to produce structured/JSON responses; automatic Go struct parsing is planned
- ✅ **Streaming** — Real-time token streaming with event-based API
- ✅ **Tracing & Monitoring** — Built-in tracing for debugging agent flows
- ✅ **OpenAI Compatibility** — Compatible with OpenAI tool definitions and API format

## 📦 Installation

### Using `go get` (Recommended)

```bash
go get github.com/citizenofai/agent-sdk-go
```

### Add to imports and run `go mod tidy`

```go
import (
    "github.com/citizenofai/agent-sdk-go/pkg/agent"
    "github.com/citizenofai/agent-sdk-go/pkg/runner"
    "github.com/citizenofai/agent-sdk-go/pkg/tool"
    // add providers as needed
)
```

```bash
go mod tidy
```

> **Note:** Requires Go 1.23 or later.

## 🚀 Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/citizenofai/agent-sdk-go/pkg/agent"
    "github.com/citizenofai/agent-sdk-go/pkg/model/providers/openai"
    "github.com/citizenofai/agent-sdk-go/pkg/runner"
    "github.com/citizenofai/agent-sdk-go/pkg/tool"
)

func main() {
    // Create a provider (or use openai.NewProvider(os.Getenv("OPENAI_API_KEY")))
    provider := openai.NewProvider("your-openai-api-key")
    provider.SetDefaultModel("gpt-4o-mini")

    // Create a function tool
    getWeather := tool.NewFunctionTool(
        "get_weather",
        "Get the weather for a city",
        func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
            city := params["city"].(string)
            return fmt.Sprintf("The weather in %s is sunny.", city), nil
        },
    ).WithSchema(map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "city": map[string]interface{}{
                "type":        "string",
                "description": "The city to get weather for",
            },
        },
        "required": []string{"city"},
    })

    // Create an agent
    assistant := agent.NewAgent("Assistant")
    assistant.SetModelProvider(provider)
    assistant.WithModel("gpt-4o-mini")
    assistant.SetSystemInstructions("You are a helpful assistant.")
    assistant.WithTools(getWeather)

    // Create a runner and execute
    r := runner.NewRunner()
    r.WithDefaultProvider(provider)

    result, err := r.Run(context.Background(), assistant, &runner.RunOptions{
        Input: "What's the weather in Tokyo?",
    })
    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    fmt.Println(result.FinalOutput)
}
```

## 🖥️ Provider Setup

### OpenAI

```go
import "github.com/citizenofai/agent-sdk-go/pkg/model/providers/openai"

provider := openai.NewProvider("your-openai-api-key")
provider.SetDefaultModel("gpt-4o-mini")
```

### Anthropic Claude

```go
import "github.com/citizenofai/agent-sdk-go/pkg/model/providers/anthropic"

provider := anthropic.NewProvider("your-anthropic-api-key")
provider.SetDefaultModel("claude-3-haiku-20240307")
```

### Google Gemini

```go
import "github.com/citizenofai/agent-sdk-go/pkg/model/providers/gemini"

// Pass your Gemini API key explicitly
provider := gemini.NewProvider("your-gemini-api-key")
provider.SetDefaultModel("gemini-2.0-flash")
```

### Mistral

```go
import "github.com/citizenofai/agent-sdk-go/pkg/model/providers/mistral"

provider := mistral.NewProvider("your-mistral-api-key")
provider.SetDefaultModel("mistral-small-latest")
```

### LM Studio (local)

```go
import "github.com/citizenofai/agent-sdk-go/pkg/model/providers/lmstudio"

provider := lmstudio.NewProvider()
provider.SetBaseURL("http://127.0.0.1:1234/v1")
provider.SetDefaultModel("gemma-3-4b-it") // replace with your loaded model
```

### Azure OpenAI

See [examples/azure_openai_example](./examples/azure_openai_example) for a full configuration example using Azure-hosted OpenAI endpoints.

## 🧩 Key Components

### Agent

The `Agent` encapsulates an LLM with system instructions, tools, and optional handoff targets.

```go
a := agent.NewAgent("Assistant")
a.SetModelProvider(provider)
a.WithModel("gpt-4o-mini")
a.SetSystemInstructions("You are a helpful assistant.")
a.WithTools(tool1, tool2)
a.WithHandoffs(subAgent1, subAgent2) // optional: enable agent delegation
```

### Runner

The `Runner` drives the agent loop — executing turns, dispatching tool calls, and handling handoffs.

```go
r := runner.NewRunner()
r.WithDefaultProvider(provider)

// Blocking run
result, err := r.Run(ctx, agent, &runner.RunOptions{
    Input:    "Hello!",
    MaxTurns: 10,
})

// Streaming run
streamResult, err := r.RunStreaming(ctx, agent, &runner.RunOptions{Input: "Hello!"})
for event := range streamResult.Stream {
    // handle model.StreamEventTypeContent, ToolCall, Done, Error ...
}
```

### Tools

Tools expose Go functions to the LLM.

```go
myTool := tool.NewFunctionTool(
    "tool_name",
    "Tool description",
    func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
        // implementation
        return result, nil
    },
).WithSchema(map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "param": map[string]interface{}{
            "type":        "string",
            "description": "Parameter description",
        },
    },
    "required": []string{"param"},
})
```

### Model Providers

All providers implement the same `model.Provider` interface, making them interchangeable:

```go
// Swap providers without changing any agent or runner code
r.WithDefaultProvider(geminiProvider)
// or
r.WithDefaultProvider(mistralProvider)
```

## 🔧 Advanced Features

### Multi-Agent Workflows

<details>
<summary>Create specialized agents that collaborate on complex tasks</summary>

```go
mathAgent := agent.NewAgent("Math Agent")
mathAgent.SetModelProvider(provider)
mathAgent.SetSystemInstructions("You are a specialized math agent. Solve the problem and return the result.")
mathAgent.WithTools(calculatorTool)

weatherAgent := agent.NewAgent("Weather Agent")
weatherAgent.SetModelProvider(provider)
weatherAgent.SetSystemInstructions("You provide weather information.")
weatherAgent.WithTools(weatherTool)

// Frontend agent delegates to specialists
frontendAgent := agent.NewAgent("Frontend Agent")
frontendAgent.SetModelProvider(provider)
frontendAgent.SetSystemInstructions("Route user requests to the appropriate specialist agent.")
frontendAgent.WithHandoffs(mathAgent, weatherAgent)

r := runner.NewRunner()
r.WithDefaultProvider(provider)
result, err := r.Run(ctx, frontendAgent, &runner.RunOptions{
    Input: "What is 42 * 17?",
})
```

See [examples/multi_agent_example](./examples/multi_agent_example) and [examples/gemini_multi_agent_example](./examples/gemini_multi_agent_example).

</details>

### Tracing

<details>
<summary>Debug and monitor your agent flows</summary>

```go
import "github.com/citizenofai/agent-sdk-go/pkg/tracing"

tracer := tracing.NewTracer()
tracer.OnEvent(func(event tracing.Event) {
    fmt.Printf("[%s] %s\n", event.Type, event.Message)
})

r := runner.NewRunner()
r.WithTracer(tracer)
```

</details>

### Structured Output

<details>
<summary>Guide the LLM to produce JSON and parse the response</summary>

`WithOutputType` communicates a JSON schema to the LLM so it returns structured JSON. Automatic Go struct parsing is not yet implemented — `result.FinalOutput` is currently the raw response string. Unmarshal it yourself:

```go
import "encoding/json"

type WeatherReport struct {
    City        string  `json:"city"`
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
}

a := agent.NewAgent("Weather Reporter")
a.WithOutputType(&WeatherReport{}) // hints to the LLM to return JSON matching this shape
a.SetSystemInstructions("Return your response as valid JSON only.")

result, err := r.Run(ctx, a, &runner.RunOptions{
    Input: "Give me a weather report for Paris.",
})
if err != nil {
    log.Fatal(err)
}

var report WeatherReport
if err := json.Unmarshal([]byte(result.FinalOutput.(string)), &report); err != nil {
    log.Fatal(err)
}
fmt.Println(report.City, report.Temperature)
```

> **Note:** Automatic unmarshaling into the registered output type is planned but not yet implemented.

</details>

### Streaming

<details>
<summary>Receive real-time token-by-token responses</summary>

```go
import "github.com/citizenofai/agent-sdk-go/pkg/model"

streamResult, err := r.RunStreaming(ctx, agent, &runner.RunOptions{
    Input: "Tell me a story.",
})
if err != nil {
    log.Fatal(err)
}

for event := range streamResult.Stream {
    switch event.Type {
    case model.StreamEventTypeContent:
        fmt.Print(event.Content)
    case model.StreamEventTypeToolCall:
        fmt.Printf("\n[Tool: %s]\n", event.ToolCall.Name)
    case model.StreamEventTypeDone:
        fmt.Println("\n[Done]")
    case model.StreamEventTypeError:
        fmt.Printf("\n[Error: %v]\n", event.Error)
    }
}
```

</details>

### Bidirectional Agent Flow

<details>
<summary>Delegate tasks to sub-agents and receive results back</summary>

Agents can hand off control to a specialist and, when the specialist finishes, hand back to the original caller — enabling complex pipelines where a coordinator agent orchestrates multiple specialists sequentially.

See [examples/bidirectional_flow_example](./examples/bidirectional_flow_example) for a full working example.

</details>

## 📚 Examples

| Example | Description |
|---------|-------------|
| [multi_agent_example](./examples/multi_agent_example) | Multi-agent system using LM Studio (local LLM) |
| [openai_example](./examples/openai_example) | OpenAI provider with function calling |
| [openai_multi_agent_example](./examples/openai_multi_agent_example) | Multi-agent with OpenAI, tool calling and streaming |
| [openai_advanced_workflow](./examples/openai_advanced_workflow) | Advanced OpenAI workflow with hooks and custom config |
| [anthropic_example](./examples/anthropic_example) | Anthropic Claude with tool calling |
| [anthropic_handoff_example](./examples/anthropic_handoff_example) | Agent handoffs with Anthropic Claude |
| [gemini_example](./examples/gemini_example) | Google Gemini single-agent with function calling |
| [gemini_multi_agent_example](./examples/gemini_multi_agent_example) | Multi-agent handoffs using Gemini |
| [mistral_example](./examples/mistral_example) | Mistral provider with tool calling |
| [azure_openai_example](./examples/azure_openai_example) | Azure-hosted OpenAI endpoint |
| [bidirectional_flow_example](./examples/bidirectional_flow_example) | Bidirectional agent delegation and return |
| [typescript_code_review_example](./examples/typescript_code_review_example) | Collaborative code review with specialised agents |

### Running an Example

**With a cloud provider:**

```bash
# OpenAI
export OPENAI_API_KEY=your-api-key
cd examples/openai_example && go run .

# Anthropic
export ANTHROPIC_API_KEY=your-api-key
cd examples/anthropic_example && go run .

# Gemini
export GOOGLE_API_KEY=your-api-key
cd examples/gemini_example && go run .

# Mistral
export MISTRAL_API_KEY=your-api-key
cd examples/mistral_example && go run .
```

**With a local LLM via LM Studio:**

```bash
# 1. Start LM Studio with a loaded model and the local server running on http://127.0.0.1:1234
cd examples/multi_agent_example && go run .
```

### Debug Flags

```bash
DEBUG=1 go run examples/bidirectional_flow_example/main.go

# Provider-specific
OPENAI_DEBUG=1    go run examples/openai_multi_agent_example/main.go
ANTHROPIC_DEBUG=1 go run examples/anthropic_example/main.go
LMSTUDIO_DEBUG=1  go run examples/multi_agent_example/main.go
```

## 🛠️ Development

### Requirements

- Go 1.23 or later

### Setup

```bash
git clone https://github.com/citizenofai/agent-sdk-go.git
cd agent-sdk-go
./scripts/ci_setup.sh
```

### Scripts

| Script | Purpose |
|--------|---------|
| `./scripts/lint.sh` | Formatting and linting |
| `./scripts/security_check.sh` | Security scan (gosec) |
| `./scripts/check_all.sh` | All checks including tests |
| `./scripts/version.sh bump` | Bump version |

### Running Tests

```bash
cd test && make test
# or
./scripts/check_all.sh
```

### CI/CD

GitHub Actions workflows are in `.github/workflows/`. The main pipelines are:

- `code-quality.yml` — lint, vet, security scan
- `codeql-analysis.yml` — static security analysis
- `release.yml` — automated releases via GoReleaser

## 👥 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](./LICENSE) file for details.
