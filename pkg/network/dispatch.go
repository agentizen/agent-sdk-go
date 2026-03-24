package network

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
	"github.com/citizenofai/agent-sdk-go/pkg/model"
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

	// Inject the decomposition JSON Schema so providers that support structured
	// output will enforce a valid response format (constrained decoding).
	schema := buildDecompositionSchema(slots)
	if orchOpts.RunConfig == nil {
		orchOpts.RunConfig = &runner.RunConfig{}
	}
	orchOpts.RunConfig.OutputSchema = schema

	// Use temperature 0 for maximum determinism during decomposition.
	// Clone ModelSettings before mutating to avoid side-effects on shared state.
	if orchOpts.RunConfig.ModelSettings == nil {
		orchOpts.RunConfig.ModelSettings = &model.Settings{}
	} else {
		cloned := *orchOpts.RunConfig.ModelSettings
		orchOpts.RunConfig.ModelSettings = &cloned
	}
	zero := 0.0
	orchOpts.RunConfig.ModelSettings.Temperature = &zero

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

// extractJSON extracts the first balanced JSON object from a string that may
// contain surrounding prose or markdown code fences. It uses brace-depth
// tracking to find the matching closing brace, ignoring braces inside
// JSON string literals.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	// No balanced closing brace found — return from start to end as best effort.
	return s[start:]
}

// synthesizeResults asks the orchestrator to produce a final consolidated answer
// from all per-agent results.
func synthesizeResults(
	ctx context.Context,
	orch *agent.Agent,
	base *runner.Runner,
	opts *runner.RunOptions,
	agentResults []AgentRunResult,
) (*result.RunResult, error) {
	var sb strings.Builder
	sb.WriteString("RESULTS:\n\n")
	for _, ar := range agentResults {
		fmt.Fprintf(&sb, "--- %s (%s) ---\n", ar.AgentName, ar.Role)
		if ar.Error != nil {
			fmt.Fprintf(&sb, "ERROR: %v\n\n", ar.Error)
			continue
		}
		if ar.RunResult != nil && ar.RunResult.FinalOutput != nil {
			fmt.Fprintf(&sb, "%v\n\n", ar.RunResult.FinalOutput)
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

// synthesizeResultsStreaming asks the orchestrator to produce a final consolidated
// answer with per-token streaming via RunStreaming.
func synthesizeResultsStreaming(
	ctx context.Context,
	orch *agent.Agent,
	base *runner.Runner,
	opts *runner.RunOptions,
	agentResults []AgentRunResult,
) (*result.StreamedRunResult, error) {
	var sb strings.Builder
	sb.WriteString("RESULTS:\n\n")
	for _, ar := range agentResults {
		fmt.Fprintf(&sb, "--- %s (%s) ---\n", ar.AgentName, ar.Role)
		if ar.Error != nil {
			fmt.Fprintf(&sb, "ERROR: %v\n\n", ar.Error)
			continue
		}
		if ar.RunResult != nil && ar.RunResult.FinalOutput != nil {
			fmt.Fprintf(&sb, "%v\n\n", ar.RunResult.FinalOutput)
		} else {
			sb.WriteString("(no output)\n\n")
		}
	}

	synthOpts := agentRunOpts(opts, sb.String())
	streamResult, err := base.RunStreaming(ctx, orch, synthOpts)
	if err != nil {
		return nil, fmt.Errorf("network: orchestrator streaming synthesis failed: %w", err)
	}
	return streamResult, nil
}

// extractUsage extracts usage info from a RunResult, returning nil if unavailable.
func extractUsage(rr *result.RunResult) *model.Usage {
	if rr == nil {
		return nil
	}
	for i := range rr.RawResponses {
		if rr.RawResponses[i].Usage != nil {
			return rr.RawResponses[i].Usage
		}
	}
	return nil
}

// emitSubAgentEvent sends a sub-agent start or end event to the channel if non-nil.
func emitSubAgentEvent(ch chan<- NetworkStreamEvent, evt NetworkStreamEvent) {
	if ch != nil {
		ch <- evt
	}
}

// runAgentWithEvents runs a single agent and emits start/end events.
func runAgentWithEvents(
	ctx context.Context,
	base *runner.Runner,
	slot AgentSlot,
	input interface{},
	opts *runner.RunOptions,
	subTaskID string,
	eventCh chan<- NetworkStreamEvent,
) AgentRunResult {
	emitSubAgentEvent(eventCh, NetworkStreamEvent{
		Type:      EventSubAgentStart,
		AgentName: slot.Agent.Name,
		SubTaskID: subTaskID,
	})

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

	var content string
	if runResult != nil && runResult.FinalOutput != nil {
		content = fmt.Sprintf("%v", runResult.FinalOutput)
	}

	emitSubAgentEvent(eventCh, NetworkStreamEvent{
		Type:      EventSubAgentEnd,
		AgentName: slot.Agent.Name,
		SubTaskID: subTaskID,
		Content:   content,
		Usage:     extractUsage(runResult),
		Duration:  dur,
	})

	return ar
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
	return dispatchParallelWithEvents(ctx, base, cfg, subTasks, opts, nil)
}

// dispatchParallelWithEvents is like dispatchParallel but emits sub-agent start/end
// events to the provided channel.
func dispatchParallelWithEvents(
	ctx context.Context,
	base *runner.Runner,
	cfg NetworkConfig,
	subTasks map[string]string,
	opts *runner.RunOptions,
	eventCh chan<- NetworkStreamEvent,
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

			sem <- struct{}{}
			defer func() { <-sem }()

			subInput := subTasks[s.Agent.Name]
			subTaskID := fmt.Sprintf("parallel-%d-%s", idx, s.Agent.Name)

			ar := runAgentWithEvents(ctx, base, s, subInput, opts, subTaskID, eventCh)

			mu.Lock()
			results[idx] = ar
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
	return dispatchSequentialWithEvents(ctx, base, cfg, subTasks, opts, nil)
}

// dispatchSequentialWithEvents is like dispatchSequential but emits sub-agent start/end
// events to the provided channel.
func dispatchSequentialWithEvents(
	ctx context.Context,
	base *runner.Runner,
	cfg NetworkConfig,
	subTasks map[string]string,
	opts *runner.RunOptions,
	eventCh chan<- NetworkStreamEvent,
) []AgentRunResult {
	results := make([]AgentRunResult, 0, len(cfg.Agents))
	prevOutput := ""

	for i, slot := range cfg.Agents {
		subTask := subTasks[slot.Agent.Name]
		var input interface{}
		if prevOutput != "" {
			input = fmt.Sprintf("Sub-task: %s\n\nPrevious agent output:\n%s", subTask, prevOutput)
		} else {
			input = subTask
		}

		subTaskID := fmt.Sprintf("sequential-%d-%s", i, slot.Agent.Name)
		ar := runAgentWithEvents(ctx, base, slot, input, opts, subTaskID, eventCh)
		results = append(results, ar)

		if ar.Error != nil {
			break
		}

		if ar.RunResult != nil && ar.RunResult.FinalOutput != nil {
			prevOutput = fmt.Sprintf("%v", ar.RunResult.FinalOutput)
		}
	}

	return results
}

// dispatchCompetitive sends the original prompt to all agents concurrently.
// The first agent to complete successfully wins; remaining agents are canceled.
// Returns a slice with a single winning AgentRunResult.
// If all agents fail, returns a slice with the last observed error.
func dispatchCompetitive(
	ctx context.Context,
	base *runner.Runner,
	cfg NetworkConfig,
	opts *runner.RunOptions,
) []AgentRunResult {
	return dispatchCompetitiveWithEvents(ctx, base, cfg, opts, nil)
}

// dispatchCompetitiveWithEvents is like dispatchCompetitive but emits sub-agent
// start/end events to the provided channel.
//
// The first agent to respond without error wins; remaining agents are canceled.
// All goroutines are guaranteed to finish before this function returns, so no
// channel is closed while writers are still active.
func dispatchCompetitiveWithEvents(
	ctx context.Context,
	base *runner.Runner,
	cfg NetworkConfig,
	opts *runner.RunOptions,
	eventCh chan<- NetworkStreamEvent,
) []AgentRunResult {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Collect all results in a fixed-size slice protected by a mutex.
	results := make([]AgentRunResult, len(cfg.Agents))
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winOnce sync.Once
		winIdx  = -1
	)

	for i, slot := range cfg.Agents {
		wg.Add(1)
		go func(idx int, s AgentSlot) {
			defer wg.Done()

			subTaskID := fmt.Sprintf("competitive-%d-%s", idx, s.Agent.Name)
			ar := runAgentWithEvents(ctx, base, s, opts.Input, opts, subTaskID, eventCh)

			mu.Lock()
			results[idx] = ar
			mu.Unlock()

			if ar.Error == nil {
				winOnce.Do(func() {
					winIdx = idx
					cancel()
				})
			}
		}(i, slot)
	}

	wg.Wait()

	// Return the winner if one exists.
	if winIdx >= 0 {
		return []AgentRunResult{results[winIdx]}
	}

	// All agents failed — return the last one in roster order that has an error.
	for i := len(results) - 1; i >= 0; i-- {
		if results[i].Error != nil {
			return []AgentRunResult{results[i]}
		}
	}
	return []AgentRunResult{results[len(results)-1]}
}
