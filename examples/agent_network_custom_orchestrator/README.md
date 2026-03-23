# Agent Network — Custom Orchestrator

This example demonstrates how to supply a **custom orchestrator agent** to fully
control prompt decomposition and result synthesis. The custom orchestrator uses
domain-specific instructions tailored to a market analysis workflow.

## When to use a custom orchestrator

Use a custom orchestrator when:
- The built-in orchestrator's generic instructions are not precise enough for your domain.
- You need deterministic decomposition (e.g., always split into exactly two tasks).
- You want the synthesis to follow a specific format (e.g., an executive report template).
- You need the orchestrator to use a different model or provider than the agents.

## Architecture

```
User Prompt
    │
    ▼
CustomOrchestrator (domain-specific DECOMPOSITION)
    ├──▶ Researcher ──┐
    └──▶ Strategist ──┘  (parallel)
                     │
                     ▼
CustomOrchestrator (domain-specific SYNTHESIS)
                     │
                     ▼
     Executive Market Analysis Report
```

## Setup

```bash
export OPENAI_API_KEY=sk-...
```

## Run

```bash
cd examples/agent_network_custom_orchestrator
go run main.go
```

## Expected output structure

```
Running custom-orchestrated network on: "..."

=== Agent Contributions ===

[Researcher]
  <market data and statistics>

[Strategist]
  <strategic analysis>

=== Executive Market Analysis Report ===
## Executive Summary
...
## Key Findings
...
## Recommendations
...
```

## Key code

```go
customOrchestrator := agent.NewAgent("MarketingOrchestrator").
    WithModel("gpt-4.1-mini").
    SetSystemInstructions(`...DECOMPOSITION MODE ... SYNTHESIS MODE ...`)

cfg := network.NewNetworkConfig().
    WithAgents(
        network.AgentSlot{Agent: researcher, Role: "Market data researcher"},
        network.AgentSlot{Agent: strategist, Role: "Strategic analyst"},
    ).
    WithStrategy(network.StrategyParallel).
    WithOrchestrator(customOrchestrator) // Override built-in orchestrator.

result, err := nr.RunNetwork(ctx, cfg, opts)
// result.OrchestratorResult holds the custom orchestrator's synthesis RunResult.
```

## Important: custom orchestrator prompt contract

Your custom orchestrator's system instructions must handle two input modes:

1. **DECOMPOSITION MODE** — input does NOT start with `"RESULTS:"`. The orchestrator
   must return a valid JSON object mapping each agent name to a sub-task string:
   ```json
   {"Researcher": "task A", "Strategist": "task B"}
   ```

2. **SYNTHESIS MODE** — input starts with `"RESULTS:"`. The orchestrator receives all
   agent outputs and must return the final consolidated answer.

If the JSON from DECOMPOSITION MODE is not parseable, the network falls back to
providing the original user prompt as input to each agent.
