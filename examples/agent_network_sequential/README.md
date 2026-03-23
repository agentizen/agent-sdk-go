# Agent Network — Sequential Strategy

This example demonstrates the **sequential dispatch strategy**: agents form a
processing pipeline where each agent receives the previous agent's output as its
input. The pipeline here is: Planner → Coder → Reviewer.

## When to use this strategy

Use **sequential** when:
- Each stage depends on the output of the previous stage.
- You want a refinement pipeline (plan → implement → review).
- Strict ordering of processing steps is required.

## Architecture

```
User Prompt
    │
    ▼
Built-in Orchestrator (DECOMPOSITION)
    │
    ▼
Planner ──▶ output
               │
               ▼
           Coder ──▶ output
                         │
                         ▼
                     Reviewer ──▶ output
                                      │
                                      ▼
                         Orchestrator (SYNTHESIS)
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
cd examples/agent_network_sequential
go run main.go
```

## Expected output structure

```
Running sequential pipeline on: "..."

=== Pipeline Stages ===

--- Stage 1: Planner (Architect) ---
<implementation plan>

--- Stage 2: Coder (Developer) ---
<Go code>

--- Stage 3: Reviewer (Reviewer) ---
<code review feedback>

=== Final Synthesis ===
<consolidated report across all stages>
```

## Key code

```go
cfg := network.NewNetworkConfig().
    WithAgents(
        network.AgentSlot{Agent: planner,  Role: "Architect"},
        network.AgentSlot{Agent: coder,    Role: "Developer"},
        network.AgentSlot{Agent: reviewer, Role: "Reviewer"},
    ).
    WithStrategy(network.StrategySequential)

result, err := nr.RunNetwork(ctx, cfg, opts)
// result.AgentResults[0] = planner output
// result.AgentResults[1] = coder output (received planner's output as input)
// result.AgentResults[2] = reviewer output (received coder's output as input)
```
