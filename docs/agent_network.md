# Agent Network — Developer Guide

The `pkg/network` package adds a **multi-agent coordination layer** on top of the core runner.
Instead of calling a single agent, you configure a *network* of specialised agents that collaborate
on the same user prompt using one of three built-in dispatch strategies.

---

## Table of Contents

- [Concepts](#concepts)
- [Quick Start](#quick-start)
- [NetworkConfig API](#networkconfig-api)
- [Dispatch Strategies](#dispatch-strategies)
  - [Parallel](#parallel)
  - [Sequential](#sequential)
  - [Competitive](#competitive)
- [Orchestrator](#orchestrator)
- [Streaming](#streaming)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Examples](#examples)

---

## Concepts

| Term | Description |
|---|---|
| `NetworkConfig` | Immutable, copy-on-write configuration value describing the network |
| `AgentSlot` | An agent paired with a human-readable role and an optional sub-task hint |
| `DispatchStrategy` | How the network dispatches work to agents (`parallel`, `sequential`, `competitive`) |
| `NetworkRunner` | Thin wrapper around `*runner.Runner` that drives network execution |
| `NetworkResult` | Collected per-agent results plus the synthesised final output |
| Built-in Orchestrator | Auto-created agent: decomposes the prompt into sub-tasks → synthesises the final answer |

### Execution flow

```
User Prompt
    │
    ▼
[Orchestrator — DECOMPOSITION]
    │  returns JSON map {agentName: subTask}
    ▼
[Agents dispatched by strategy]
    │  each runs its sub-task
    ▼
[Orchestrator — SYNTHESIS]
    │  consolidates all agent results
    ▼
NetworkResult.FinalOutput
```

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/citizenofai/agent-sdk-go/pkg/agent"
    "github.com/citizenofai/agent-sdk-go/pkg/model/providers/openai"
    "github.com/citizenofai/agent-sdk-go/pkg/network"
    "github.com/citizenofai/agent-sdk-go/pkg/runner"
)

func main() {
    provider := openai.NewProvider(os.Getenv("OPENAI_API_KEY"))

    researcher := agent.NewAgent("Researcher", "You are a research specialist.")
    writer := agent.NewAgent("Writer", "You are a professional writer.")

    cfg := network.NewNetworkConfig().
        WithAgents(
            network.AgentSlot{Agent: researcher, Role: "Research specialist"},
            network.AgentSlot{Agent: writer, Role: "Content writer"},
        ).
        WithStrategy(network.StrategyParallel)

    nr := network.NewNetworkRunner(runner.NewRunner().WithDefaultProvider(provider))

    opts := &runner.RunOptions{Input: "Explain the impact of AI on healthcare"}
    result, err := nr.RunNetwork(context.Background(), cfg, opts)
    if err != nil {
        panic(err)
    }

    fmt.Println(result.FinalOutput)
}
```

---

## NetworkConfig API

`NetworkConfig` is a **value type** — every builder method returns a new copy, leaving the
original unchanged (copy-on-write). This makes it safe to share and reuse partial configurations.

```go
cfg := network.NewNetworkConfig().           // zero value with StrategyParallel default
    WithAgents(slots ...AgentSlot).          // set the agent roster
    WithStrategy(strategy DispatchStrategy). // override the dispatch strategy
    WithOrchestrator(orch *agent.Agent).     // use a custom orchestrator instead of the built-in one
    WithMaxConcurrency(n int)                // cap goroutines in parallel mode (0 = unlimited)
```

### AgentSlot

```go
type AgentSlot struct {
    Agent       *agent.Agent // required
    Role        string       // shown to the orchestrator ("Research specialist")
    SubTaskHint string       // optional extra hint for task decomposition
}
```

### Validation

`cfg.Validate()` is called automatically by `RunNetwork`. It checks:

- At least one agent slot.
- No nil agents.
- No duplicate agent names.
- Strategy is one of `parallel`, `sequential`, `competitive`.

---

## Dispatch Strategies

### Parallel

All agents run **concurrently**. The orchestrator decomposes the prompt into per-agent sub-tasks
first; then all agents execute simultaneously. Use when sub-tasks are independent and you want
minimum wall-clock time.

```
User Prompt ──▶ Orchestrator (DECOMP)
                    ├──▶ Agent A ──▶ result A ──┐
                    ├──▶ Agent B ──▶ result B ──┤──▶ Orchestrator (SYNTH) ──▶ FinalOutput
                    └──▶ Agent C ──▶ result C ──┘
```

```go
cfg := network.NewNetworkConfig().
    WithAgents(slotA, slotB, slotC).
    WithStrategy(network.StrategyParallel).
    WithMaxConcurrency(3) // optional goroutine cap
```

`result.AgentResults` contains one entry per agent, in the same order as configured via `WithAgents`.

---

### Sequential

Agents form a **pipeline**: each agent receives the previous agent's output as its input.
The orchestrator decomposes into per-agent sub-tasks; then agents run one-by-one.
Use when each stage depends on the previous stage's output (plan → code → review).

```
User Prompt ──▶ Orchestrator (DECOMP)
                    ▼
               Agent A ──▶ output A
                               ▼
                          Agent B ──▶ output B
                                          ▼
                                     Agent C ──▶ output C
                                                     ▼
                                          Orchestrator (SYNTH) ──▶ FinalOutput
```

```go
cfg := network.NewNetworkConfig().
    WithAgents(planner, coder, reviewer).
    WithStrategy(network.StrategySequential)
```

`result.AgentResults[i]` = output of slot i; `result.LastAgent` = last agent in the chain.

---

### Competitive

All agents receive the **same prompt** and race to answer. The first agent that responds without
error wins; all remaining agents are cancelled via `context.WithCancel`. Use when latency is
critical or you want to hedge across model tiers.

```
User Prompt ──▶ Agent A (fast model) ──┐
            └──▶ Agent B (big model)  ──┤  First non-error response wins
                                         ▼
                                    FinalOutput (winner's output)
```

```go
cfg := network.NewNetworkConfig().
    WithAgents(
        network.AgentSlot{Agent: fastAgent, Role: "Low-latency model"},
        network.AgentSlot{Agent: powerAgent, Role: "High-accuracy model"},
    ).
    WithStrategy(network.StrategyCompetitive)
```

`result.AgentResults` contains exactly **one entry** (the winner).
`result.OrchestratorResult` is `nil` — synthesis is bypassed; the winner's output is the final answer.
If all agents fail, `RunNetwork` returns an error.

---

## Orchestrator

The built-in orchestrator is created automatically if you do not provide one. It operates in
two modes, selected by the input prefix:

| Mode | Trigger | Output |
|---|---|---|
| DECOMPOSITION | Input does **not** start with `"RESULTS:"` | JSON `{"AgentName": "sub-task string", ...}` |
| SYNTHESIS | Input starts with `"RESULTS:"` | Free-form final answer |

### Custom orchestrator

Pass your own agent via `WithOrchestrator`:

```go
orch := agent.NewAgent("MyOrch", `You are a domain expert orchestrator. ...`)
orch.WithModel("gpt-4.1")

cfg := network.NewNetworkConfig().
    WithAgents(slotA, slotB).
    WithOrchestrator(orch).
    WithStrategy(network.StrategyParallel)
```

The custom orchestrator must respect the same JSON output contract for DECOMPOSITION mode
(one key per agent name in the roster) or the network runner will fall back gracefully to
assigning the raw user prompt to every agent.

### Orchestrator model

The built-in orchestrator inherits the model from `opts.RunConfig.Model` when set; otherwise
it falls back to the provider's default model (`provider.GetModel("")`). To target a specific
model for orchestration:

```go
opts := &runner.RunOptions{
    Input: "...",
    RunConfig: &runner.RunConfig{
        ModelProvider: provider,
        Model:         "gpt-4.1",
    },
}
```

---

## Streaming

`RunNetworkStreaming` returns a channel of `NetworkStreamEvent`. Each event has a `Type` field
(`EventSubAgentStart`, `EventSubAgentEnd`, `EventOrchestratorContent`, `EventOrchestratorDone`,
or `EventNetworkError`) and additional fields populated according to the type.

```go
ch, err := nr.RunNetworkStreaming(ctx, cfg, opts)
if err != nil {
    panic(err)
}
for evt := range ch {
    switch evt.Type {
    case network.EventSubAgentEnd:
        fmt.Printf("[%s] finished: %s\n", evt.AgentName, evt.Content)
    case network.EventOrchestratorContent:
        fmt.Print(evt.Content) // per-token chunk
    case network.EventOrchestratorDone:
        fmt.Printf("\n=== Final answer ===\n%v\n", evt.FinalOutput)
    case network.EventNetworkError:
        fmt.Printf("error: %v\n", evt.Error)
    }
}
```

The channel is closed after the final event (or after an error event).

---

## Error Handling

| Situation | Behaviour |
|---|---|
| `Validate()` fails | `RunNetwork` returns error immediately, no LLM calls made |
| One parallel agent fails | Error recorded in `AgentRunResult.Error`; other agents continue |
| Sequential agent fails | Pipeline stops; `RunNetwork` / `RunNetworkStreaming` returns an error |
| All competitive agents fail | `RunNetwork` / `RunNetworkStreaming` returns an error |
| Orchestrator (DECOMP) returns invalid JSON | Runner falls back: assigns the raw user prompt to every agent |
| Context cancelled | In-flight agents are cancelled; `RunNetwork` returns `context.Canceled` |

---

## Testing

The network package is tested in `test/network/network_test.go` (coverage ≥ 90 %).

For the exact, up-to-date method signatures, test helpers, and mock implementations, see:
[`test/network/network_test.go`](../test/network/network_test.go)

Run the test suite:

```bash
go test -race ./test/network/... -v
```

---

## Examples

| Example | Strategy | What it demonstrates |
|---|---|---|
| [`agent_network_parallel`](../examples/agent_network_parallel) | Parallel | Three specialised agents (Researcher, Analyst, Writer) working simultaneously |
| [`agent_network_sequential`](../examples/agent_network_sequential) | Sequential | Planner → Coder → Reviewer pipeline |
| [`agent_network_competitive`](../examples/agent_network_competitive) | Competitive | Fast model vs powerful model race |
| [`agent_network_custom_orchestrator`](../examples/agent_network_custom_orchestrator) | Parallel | Domain-specific orchestrator for market analysis |

Each example folder contains its own `README.md` with architecture diagram, setup, and expected
output.
