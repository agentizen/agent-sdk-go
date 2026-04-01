package network

import (
	"context"
	"fmt"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
)

// NetworkRunner orchestrates a configured set of agents via a NetworkConfig.
// It embeds *runner.Runner to reuse model resolution, tracing, and agent execution.
type NetworkRunner struct {
	*runner.Runner
}

// NewNetworkRunner returns a NetworkRunner backed by the provided base Runner.
func NewNetworkRunner(base *runner.Runner) *NetworkRunner {
	return &NetworkRunner{Runner: base}
}

// RunNetwork executes the network synchronously and returns the aggregated result.
//
// The execution follows three phases:
//  1. Decomposition — the orchestrator analyses the prompt and assigns sub-tasks.
//  2. Dispatch — sub-tasks are distributed to agents using cfg.Strategy.
//  3. Synthesis — the orchestrator consolidates all agent results into FinalOutput.
//
// opts.RunConfig (model, provider, tracing settings) is forwarded to every agent call.
// opts.MaxTurns applies to each individual agent run.
func (nr *NetworkRunner) RunNetwork(
	ctx context.Context,
	cfg NetworkConfig,
	opts *runner.RunOptions,
) (*NetworkResult, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &runner.RunOptions{}
	}

	// Phase 1: Resolve orchestrator.
	orch := cfg.Orchestrator
	if orch == nil {
		var runCfg *runner.RunConfig
		if opts.RunConfig != nil {
			runCfg = opts.RunConfig
		}
		orch = newBuiltInOrchestrator(cfg, runCfg)
	}

	// Phase 2 & 3: Decompose (when applicable) and dispatch according to strategy.
	// Competitive sends the raw prompt to all agents directly, so decomposition
	// would be wasted work (extra LLM call) — skip it for that strategy.
	var agentResults []AgentRunResult
	switch cfg.Strategy {
	case StrategyParallel:
		subTasks, err := decomposePrompt(ctx, orch, nr.Runner, opts, cfg.Agents)
		if err != nil {
			return nil, fmt.Errorf("network: decomposition phase failed: %w", err)
		}
		agentResults = dispatchParallel(ctx, nr.Runner, cfg, subTasks, opts)
	case StrategySequential:
		subTasks, err := decomposePrompt(ctx, orch, nr.Runner, opts, cfg.Agents)
		if err != nil {
			return nil, fmt.Errorf("network: decomposition phase failed: %w", err)
		}
		agentResults = dispatchSequential(ctx, nr.Runner, cfg, subTasks, opts)
		// If the pipeline stopped due to an agent error, surface it immediately.
		if len(agentResults) > 0 {
			if last := agentResults[len(agentResults)-1]; last.Error != nil {
				return nil, fmt.Errorf("network: sequential agent %q failed: %w", last.AgentName, last.Error)
			}
		}
	case StrategyCompetitive:
		agentResults = dispatchCompetitive(ctx, nr.Runner, cfg, opts)
		// For competitive, return the winning result directly — synthesis is not needed
		// because only one agent's output is the final answer.
		var winner *AgentRunResult
		for i := range agentResults {
			if agentResults[i].Error == nil {
				winner = &agentResults[i]
				break
			}
		}
		if winner == nil && len(agentResults) > 0 {
			// All agents failed — surface the error.
			return nil, fmt.Errorf("network: all competitive agents failed; last error: %w", agentResults[len(agentResults)-1].Error)
		}
		var finalOutput interface{}
		var lastAgent *agent.Agent
		if winner != nil {
			if winner.RunResult != nil {
				finalOutput = winner.RunResult.FinalOutput
			}
			// Resolve the agent pointer from the roster by name.
			for _, slot := range cfg.Agents {
				if slot.Agent.Name == winner.AgentName {
					lastAgent = slot.Agent
					break
				}
			}
		}
		return &NetworkResult{
			AgentResults:       agentResults,
			FinalOutput:        finalOutput,
			OrchestratorResult: nil,
			Strategy:           cfg.Strategy,
			LastAgent:          lastAgent,
		}, nil
	default:
		return nil, fmt.Errorf("network: unsupported strategy %q", cfg.Strategy)
	}

	// Phase 4: Synthesize final answer.
	synthResult, err := synthesizeResults(ctx, orch, nr.Runner, opts, agentResults)
	if err != nil {
		return nil, fmt.Errorf("network: synthesis phase failed: %w", err)
	}

	var finalOutput interface{}
	if synthResult != nil {
		finalOutput = synthResult.FinalOutput
	}

	return &NetworkResult{
		AgentResults:       agentResults,
		FinalOutput:        finalOutput,
		OrchestratorResult: synthResult,
		Strategy:           cfg.Strategy,
		LastAgent:          orch,
	}, nil
}

// RunNetworkStreaming executes the network and returns a channel of typed
// NetworkStreamEvents emitted in real time:
//
//   - EventSubAgentStart / EventSubAgentEnd for each sub-agent (with content, usage, duration)
//   - EventOrchestratorContent for each token chunk during synthesis (per-token streaming)
//   - EventOrchestratorDone when synthesis is complete (with FinalOutput and usage)
//   - EventNetworkError on unrecoverable error
//
// The channel is closed after the final event or on error.
// The caller must drain the channel fully to avoid goroutine leaks.
func (nr *NetworkRunner) RunNetworkStreaming(
	ctx context.Context,
	cfg NetworkConfig,
	opts *runner.RunOptions,
) (<-chan NetworkStreamEvent, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &runner.RunOptions{}
	}

	// Small fixed buffer: the consumer must drain events concurrently.
	// Back-pressure from the channel naturally throttles the producer.
	// 2 events per agent (start+end) + a small margin for orchestrator events.
	ch := make(chan NetworkStreamEvent, len(cfg.Agents)*2+8)

	go func() {
		defer close(ch)

		// Phase 1: Resolve orchestrator.
		orch := cfg.Orchestrator
		if orch == nil {
			var runCfg *runner.RunConfig
			if opts.RunConfig != nil {
				runCfg = opts.RunConfig
			}
			orch = newBuiltInOrchestrator(cfg, runCfg)
		}

		// Phase 2 & 3: Decompose (when applicable) and dispatch.
		// Competitive sends the raw prompt directly, so skip decomposition for it.
		var agentResults []AgentRunResult
		switch cfg.Strategy {
		case StrategyParallel:
			subTasks, err := decomposePrompt(ctx, orch, nr.Runner, opts, cfg.Agents)
			if err != nil {
				ch <- NetworkStreamEvent{
					Type:  EventNetworkError,
					Error: fmt.Errorf("decomposition phase failed: %w", err),
				}
				return
			}
			agentResults = dispatchParallelWithEvents(ctx, nr.Runner, cfg, subTasks, opts, ch)
		case StrategySequential:
			subTasks, err := decomposePrompt(ctx, orch, nr.Runner, opts, cfg.Agents)
			if err != nil {
				ch <- NetworkStreamEvent{
					Type:  EventNetworkError,
					Error: fmt.Errorf("decomposition phase failed: %w", err),
				}
				return
			}
			agentResults = dispatchSequentialWithEvents(ctx, nr.Runner, cfg, subTasks, opts, ch)
			// Surface sequential pipeline errors immediately.
			if len(agentResults) > 0 {
				if last := agentResults[len(agentResults)-1]; last.Error != nil {
					ch <- NetworkStreamEvent{
						Type:      EventNetworkError,
						AgentName: last.AgentName,
						Error:     fmt.Errorf("sequential agent %q failed: %w", last.AgentName, last.Error),
					}
					return
				}
			}
		case StrategyCompetitive:
			agentResults = dispatchCompetitiveWithEvents(ctx, nr.Runner, cfg, opts, ch)
			// Short-circuit: return the winning agent result without synthesis.
			var winner *AgentRunResult
			for i := range agentResults {
				if agentResults[i].Error == nil {
					winner = &agentResults[i]
					break
				}
			}
			if winner == nil && len(agentResults) > 0 {
				ch <- NetworkStreamEvent{
					Type:  EventNetworkError,
					Error: fmt.Errorf("all competitive agents failed; last error: %w", agentResults[len(agentResults)-1].Error),
				}
				return
			}
			var finalOutput interface{}
			if winner != nil && winner.RunResult != nil {
				finalOutput = winner.RunResult.FinalOutput
			}
			ch <- NetworkStreamEvent{
				Type:        EventOrchestratorDone,
				FinalOutput: finalOutput,
			}
			return
		default:
			ch <- NetworkStreamEvent{
				Type:  EventNetworkError,
				Error: fmt.Errorf("unsupported strategy %q", cfg.Strategy),
			}
			return
		}

		// Phase 4: Synthesis with per-token streaming from the orchestrator.
		streamResult, err := synthesizeResultsStreaming(ctx, orch, nr.Runner, opts, agentResults)
		if err != nil {
			ch <- NetworkStreamEvent{
				Type:  EventNetworkError,
				Error: fmt.Errorf("synthesis phase failed: %w", err),
			}
			return
		}

		// Forward orchestrator tokens as EventOrchestratorContent events.
		var finalContent string
		var synthUsage *model.Usage
		for evt := range streamResult.Stream {
			switch evt.Type {
			case model.StreamEventTypeContent:
				finalContent += evt.Content
				ch <- NetworkStreamEvent{
					Type:    EventOrchestratorContent,
					Content: evt.Content,
				}
			case model.StreamEventTypeDone:
				if evt.Response != nil && evt.Response.Usage != nil {
					synthUsage = evt.Response.Usage
				}
			case model.StreamEventTypeError:
				if evt.Error != nil {
					ch <- NetworkStreamEvent{
						Type:  EventNetworkError,
						Error: fmt.Errorf("orchestrator streaming error: %w", evt.Error),
					}
					// Drain the remainder of the stream to unblock the producer
					// goroutine, then exit without emitting EventOrchestratorDone.
					for range streamResult.Stream { //nolint:revive
					}
					return
				}
			}
		}

		// Emit final orchestrator done event.
		var finalOutput interface{} = finalContent
		if streamResult.RunResult != nil && streamResult.FinalOutput != nil {
			finalOutput = streamResult.FinalOutput
		}

		ch <- NetworkStreamEvent{
			Type:        EventOrchestratorDone,
			FinalOutput: finalOutput,
			Usage:       synthUsage,
		}
	}()

	return ch, nil
}
