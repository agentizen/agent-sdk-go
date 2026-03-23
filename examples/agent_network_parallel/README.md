# Agent Network — Parallel Strategy

This example demonstrates the **parallel dispatch strategy**: three specialised agents
(Researcher, Analyst, Writer) work simultaneously on different aspects of the same
prompt. The built-in orchestrator decomposes the prompt into per-agent sub-tasks,
runs all agents concurrently, and synthesises a final consolidated answer.

## When to use this strategy

Use **parallel** when:
- Sub-tasks are independent (no agent needs another's output to proceed).
- You want maximum throughput — all agents run at the same time.
- You need multiple domain perspectives combined into one answer.

## Architecture

```
User Prompt
    │
    ▼
Built-in Orchestrator (DECOMPOSITION)
    ├──▶ Researcher ──┐
    ├──▶ Analyst   ──┤  (all run concurrently)
    └──▶ Writer    ──┤
                     ▼
         Built-in Orchestrator (SYNTHESIS)
                     │
                     ▼
              FinalOutput
```

## Setup

```bash
export OPENAI_API_KEY=sk-...
```

## Run

```bash
cd examples/agent_network_parallel
go run main.go
```

## Expected output structure

```
Running parallel agent network on: "..."

=== Per-Agent Results ===

[Researcher — Research specialist] (1234ms)
  <research findings>

[Analyst — Strategic analyst] (987ms)
  <strategic analysis>

[Writer — Content writer] (756ms)
  <written content>

=== Final Synthesised Answer ===
<consolidated answer from the orchestrator>
```

## Key code

```go
cfg := network.NewNetworkConfig().
    WithAgents(
        network.AgentSlot{Agent: researcher, Role: "Research specialist"},
        network.AgentSlot{Agent: analyst,    Role: "Strategic analyst"},
        network.AgentSlot{Agent: writer,     Role: "Content writer"},
    ).
    WithStrategy(network.StrategyParallel).
    WithMaxConcurrency(3)

nr := network.NewNetworkRunner(runner.NewRunner().WithDefaultProvider(provider))
result, err := nr.RunNetwork(ctx, cfg, opts)
```
