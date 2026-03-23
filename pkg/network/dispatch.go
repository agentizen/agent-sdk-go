package network

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
	"github.com/citizenofai/agent-sdk-go/pkg/result"
	"github.com/citizenofai/agent-sdk-go/pkg/runner"
)

// concurrencyLimit returns the effective concurrency cap for parallel dispatch.
// When cfg.MaxConcurrency is zero or exceeds the agent count, the agent count is used.
func concurrencyLimit(cfg NetworkConfig) int {
	if cfg.MaxConcurrency <= 0 || cfg.MaxConcurrency > len(cfg.Agents) {
		return len(cfg.Agents)
	}
	return cfg.MaxConcurrency
}

// agentRunOpts builds a RunOptions for a single agent call, inheriting RunConfig and
// MaxTurns from the parent opts but overriding Input with the provided input value.
// A shallow copy of RunConfig is made so the runner cannot mutate the shared config
// across concurrent goroutines.
func agentRunOpts(parent *runner.RunOptions, input interface{}) *runner.RunOptions {
	var rc *runner.RunConfig
	if parent.RunConfig != nil {
		copy := *parent.RunConfig
		rc = &copy
	}
	return &runner.RunOptions{
		Input:     input,
		MaxTurns:  parent.MaxTurns,
		RunConfig: rc,
	}
}

// decomposePrompt asks the orchestrator to break the user prompt into per-agent sub-tasks.
// It returns a map from agent name to sub-task description.
// For any agent whose name is not returned by the orchestrator, the original input is used.
func decomposePrompt(
	ctx context.Context,
	orch *agent.Agent,
	base *runner.Runner,
	opts *runner.RunOptions,
	slots []AgentSlot,
) (map[string]string, error) {
	orchOpts := agentRunOpts(opts, opts.Input)

	runResult, err := base.Run(ctx, orch, orchOpts)
	if err != nil {
		return nil, fmt.Errorf("network: orchestrator decomposition failed: %w", err)
	}

	// Extract the JSON string from FinalOutput.
	var raw string
	switch v := runResult.FinalOutput.(type) {
	case string:
		raw = v
	case fmt.Stringer:
		raw = v.String()
	default:
		raw = fmt.Sprintf("%v", runResult.FinalOutput)
	}

	// Try to extract JSON object from the response (the model may wrap it in prose).
	raw = extractJSON(raw)

	subTasks := make(map[string]string, len(slots))
	if err := json.Unmarshal([]byte(raw), &subTasks); err != nil {
		// If parsing fails, fall back to using the original input for all agents.
		originalInput := fmt.Sprintf("%v", opts.Input)
		for _, slot := range slots {
			subTasks[slot.Agent.Name] = originalInput
		}
		return subTasks, nil
	}

	// Ensure every agent gets an entry — fall back to original input if missing.
	originalInput := fmt.Sprintf("%v", opts.Input)
	for _, slot := range slots {
		if _, ok := subTasks[slot.Agent.Name]; !ok {
			subTasks[slot.Agent.Name] = originalInput
		}
	}
	return subTasks, nil
}

// extractJSON attempts to extract the first JSON object from a string that may
// contain surrounding prose or markdown code fences.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip possible markdown code fences.
	if idx := strings.Index(s, "{"); idx >= 0 {
		s = s[idx:]
	}
	if idx := strings.LastIndex(s, "}"); idx >= 0 {
		s = s[:idx+1]
	}
	return s
}

// synthesiseResults asks the orchestrator to produce a final consolidated answer
// from all per-agent results.
func synthesiseResults(
	ctx context.Context,
	orch *agent.Agent,
	base *runner.Runner,
	opts *runner.RunOptions,
	agentResults []AgentRunResult,
) (*result.RunResult, error) {
	var sb strings.Builder
	sb.WriteString("RESULTS:\n\n")
	for _, ar := range agentResults {
		sb.WriteString(fmt.Sprintf("--- %s (%s) ---\n", ar.AgentName, ar.Role))
		if ar.Error != nil {
			sb.WriteString(fmt.Sprintf("ERROR: %v\n\n", ar.Error))
			continue
		}
		if ar.RunResult != nil && ar.RunResult.FinalOutput != nil {
			sb.WriteString(fmt.Sprintf("%v\n\n", ar.RunResult.FinalOutput))
		} else {
			sb.WriteString("(no output)\n\n")
		}
	}

	synthOpts := agentRunOpts(opts, sb.String())
	runResult, err := base.Run(ctx, orch, synthOpts)
	if err != nil {
		return nil, fmt.Errorf("network: orchestrator synthesis failed: %w", err)
	}
	return runResult, nil
}

// dispatchParallel runs all agents concurrently on their assigned sub-tasks.
// Results are collected once all goroutines complete, in roster order.
func dispatchParallel(
	ctx context.Context,
	base *runner.Runner,
	cfg NetworkConfig,
	subTasks map[string]string,
	opts *runner.RunOptions,
) []AgentRunResult {
	results := make([]AgentRunResult, len(cfg.Agents))
	var wg sync.WaitGroup
	var mu sync.Mutex

	limit := concurrencyLimit(cfg)
	sem := make(chan struct{}, limit)

	for i, slot := range cfg.Agents {
		wg.Add(1)
		go func(idx int, s AgentSlot) {
			defer wg.Done()

			// Acquire semaphore slot.
			sem <- struct{}{}
			defer func() { <-sem }()

			subInput := subTasks[s.Agent.Name]
			start := time.Now()
			runResult, runErr := base.Run(ctx, s.Agent, agentRunOpts(opts, subInput))
			dur := time.Since(start)

			mu.Lock()
			results[idx] = AgentRunResult{
				AgentName: s.Agent.Name,
				Role:      s.Role,
				RunResult: runResult,
				Error:     runErr,
				Duration:  dur,
			}
			mu.Unlock()
		}(i, slot)
	}

	wg.Wait()
	return results
}

// dispatchSequential runs agents one after another.
// The first agent receives its sub-task; each subsequent agent receives the previous
// agent's FinalOutput (falling back to its own sub-task if the previous output is empty).
// The chain stops on the first agent error.
func dispatchSequential(
	ctx context.Context,
	base *runner.Runner,
	cfg NetworkConfig,
	subTasks map[string]string,
	opts *runner.RunOptions,
) []AgentRunResult {
	results := make([]AgentRunResult, 0, len(cfg.Agents))
	prevOutput := ""

	for _, slot := range cfg.Agents {
		var input interface{}
		if prevOutput != "" {
			input = prevOutput
		} else {
			input = subTasks[slot.Agent.Name]
		}

		start := time.Now()
		runResult, runErr := base.Run(ctx, slot.Agent, agentRunOpts(opts, input))
		dur := time.Since(start)

		ar := AgentRunResult{
			AgentName: slot.Agent.Name,
			Role:      slot.Role,
			RunResult: runResult,
			Error:     runErr,
			Duration:  dur,
		}
		results = append(results, ar)

		if runErr != nil {
			// Stop the chain on error.
			break
		}

		if runResult != nil && runResult.FinalOutput != nil {
			prevOutput = fmt.Sprintf("%v", runResult.FinalOutput)
		}
	}

	return results
}

// dispatchCompetitive sends the original prompt to all agents concurrently.
// The first agent to complete successfully wins; remaining agents are cancelled.
// Returns a slice with a single winning AgentRunResult.
// If all agents fail, returns a slice with the last observed error.
func dispatchCompetitive(
	ctx context.Context,
	base *runner.Runner,
	cfg NetworkConfig,
	opts *runner.RunOptions,
) []AgentRunResult {
	type raceResult struct {
		ar  AgentRunResult
		won bool
	}

	winCh := make(chan AgentRunResult, 1)
	errCh := make(chan AgentRunResult, len(cfg.Agents))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for _, slot := range cfg.Agents {
		wg.Add(1)
		go func(s AgentSlot) {
			defer wg.Done()
			start := time.Now()
			runResult, runErr := base.Run(ctx, s.Agent, agentRunOpts(opts, opts.Input))
			dur := time.Since(start)
			ar := AgentRunResult{
				AgentName: s.Agent.Name,
				Role:      s.Role,
				RunResult: runResult,
				Error:     runErr,
				Duration:  dur,
			}
			if runErr == nil {
				select {
				case winCh <- ar:
					cancel() // Cancel remaining agents.
				default:
					// Another agent already won.
				}
			} else {
				errCh <- ar
			}
		}(slot)
	}

	// Collect all goroutines in background so channels don't leak.
	go func() {
		wg.Wait()
		close(winCh)
		close(errCh)
	}()

	// Return the first winner, or last error if all failed.
	if winner, ok := <-winCh; ok {
		return []AgentRunResult{winner}
	}

	// All agents failed — return the last error result.
	var lastErr AgentRunResult
	for ar := range errCh {
		lastErr = ar
	}
	return []AgentRunResult{lastErr}
}
