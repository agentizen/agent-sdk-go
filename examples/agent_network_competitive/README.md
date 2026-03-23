# Agent Network — Competitive Strategy

This example demonstrates the **competitive dispatch strategy**: multiple agents
receive the same prompt simultaneously and race to answer. The first agent to
respond without error wins; all remaining agents are cancelled.

## When to use this strategy

Use **competitive** when:
- Latency is critical and you want the fastest acceptable answer.
- You want to hedge across model tiers (powerful vs fast model).
- You need fault tolerance — if the primary model is slow or fails, a backup wins.

## Architecture

```
User Prompt
    ├──▶ PowerAgent (gpt-4.1)      ──┐
    └──▶ SpeedAgent (gpt-4.1-mini) ──┤  (all run concurrently)
                                      │
                              First to finish wins
                                      │
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
cd examples/agent_network_competitive
go run main.go
```

## Expected output structure

```
Running competitive race on: "..."

Two agents are competing — fastest non-error response wins...

Winner: SpeedAgent (Low-latency model) — responded in 812ms

=== Winning Response ===
<the winning agent's answer>
```

## Notes

- In competitive mode, `result.AgentResults` contains exactly one entry (the winner).
- The losing agents are cancelled via `context.WithCancel` when the winner is found.
- If ALL agents fail, the last observed error result is returned.

## Key code

```go
cfg := network.NewNetworkConfig().
    WithAgents(
        network.AgentSlot{Agent: powerAgent, Role: "High-accuracy model"},
        network.AgentSlot{Agent: speedAgent, Role: "Low-latency model"},
    ).
    WithStrategy(network.StrategyCompetitive)

result, err := nr.RunNetwork(ctx, cfg, opts)
// len(result.AgentResults) == 1 (winner only)
winner := result.AgentResults[0]
fmt.Printf("Winner: %s in %dms\n", winner.AgentName, winner.Duration.Milliseconds())
```
