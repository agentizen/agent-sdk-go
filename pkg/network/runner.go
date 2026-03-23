package network

import (
	"context"
	"fmt"

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

	// Phase 4: Synthesise final answer.
	synthResult, err := synthesiseResults(ctx, orch, nr.Runner, opts, agentResults)
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

// RunNetworkStreaming executes the network asynchronously and returns a channel of
// tagged NetworkStreamEvents. Each dispatched agent emits one event when it completes;
// a final event with IsFinal=true carries the synthesised FinalOutput.
//
// The channel is closed after the final synthesis event or on error.
// The caller must drain the channel fully to avoid goroutine leaks.
func (nr *NetworkRunner) RunNetworkStreaming(
	ctx context.Context,
	cfg NetworkConfig,
	opts *runner.RunOptions,
) (<-chan NetworkStreamEvent, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Buffer one slot per agent plus the synthesis event.
	bufSize := len(cfg.Agents) + 1
	ch := make(chan NetworkStreamEvent, bufSize)

	go func() {
		defer close(ch)

		networkResult, err := nr.RunNetwork(ctx, cfg, opts)
		if err != nil {
			ch <- NetworkStreamEvent{
				AgentName:   "NetworkRunner",
				IsFinal:     true,
				StreamEvent: fmt.Errorf("network run failed: %w", err),
			}
			return
		}

		// Emit one event per agent result.
		for _, ar := range networkResult.AgentResults {
			ch <- NetworkStreamEvent{
				AgentName:   ar.AgentName,
				StreamEvent: ar,
				IsFinal:     false,
			}
		}

		// Emit final synthesis event.
		ch <- NetworkStreamEvent{
			AgentName:   "NetworkOrchestrator",
			StreamEvent: networkResult.FinalOutput,
			IsFinal:     true,
		}
	}()

	return ch, nil
}
