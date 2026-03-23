package network

import (
	"context"
	"fmt"

	"github.com/citizenofai/agent-sdk-go/pkg/model"
	"github.com/citizenofai/agent-sdk-go/pkg/runner"
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

	// Phase 2: Decompose prompt into per-agent sub-tasks.
	subTasks, err := decomposePrompt(ctx, orch, nr.Runner, opts, cfg.Agents)
	if err != nil {
		return nil, fmt.Errorf("network: decomposition phase failed: %w", err)
	}

	// Phase 3: Dispatch sub-tasks according to strategy.
	var agentResults []AgentRunResult
	switch cfg.Strategy {
	case StrategyParallel:
		agentResults = dispatchParallel(ctx, nr.Runner, cfg, subTasks, opts)
	case StrategySequential:
		agentResults = dispatchSequential(ctx, nr.Runner, cfg, subTasks, opts)
	case StrategyCompetitive:
		agentResults = dispatchCompetitive(ctx, nr.Runner, cfg, opts)
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

		// Phase 2: Decompose prompt.
		subTasks, err := decomposePrompt(ctx, orch, nr.Runner, opts, cfg.Agents)
		if err != nil {
			ch <- NetworkStreamEvent{
				Type:  EventNetworkError,
				Error: fmt.Errorf("decomposition phase failed: %w", err),
			}
			return
		}

		// Phase 3: Dispatch sub-tasks with event emission.
		var agentResults []AgentRunResult
		switch cfg.Strategy {
		case StrategyParallel:
			agentResults = dispatchParallelWithEvents(ctx, nr.Runner, cfg, subTasks, opts, ch)
		case StrategySequential:
			agentResults = dispatchSequentialWithEvents(ctx, nr.Runner, cfg, subTasks, opts, ch)
		case StrategyCompetitive:
			agentResults = dispatchCompetitiveWithEvents(ctx, nr.Runner, cfg, opts, ch)
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
