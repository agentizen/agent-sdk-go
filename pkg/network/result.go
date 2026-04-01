package network

import (
	"time"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/result"
)

// NetworkStreamEventType identifies the kind of streaming event emitted during a
// network run.
type NetworkStreamEventType string

const (
	// EventSubAgentStart is emitted just before a sub-agent begins execution.
	EventSubAgentStart NetworkStreamEventType = "sub_agent_start"

	// EventSubAgentEnd is emitted after a sub-agent finishes execution.
	EventSubAgentEnd NetworkStreamEventType = "sub_agent_end"

	// EventOrchestratorContent is emitted for each content token streamed by the
	// orchestrator during the synthesis phase.
	EventOrchestratorContent NetworkStreamEventType = "orchestrator_content"

	// EventOrchestratorDone is emitted once the orchestrator synthesis is complete.
	EventOrchestratorDone NetworkStreamEventType = "orchestrator_done"

	// EventNetworkError is emitted when the network encounters an unrecoverable error.
	EventNetworkError NetworkStreamEventType = "network_error"
)

// NetworkStreamEvent is a typed streaming event emitted during a network run.
// Depending on the Type field, different combinations of fields are populated:
//
//   - EventSubAgentStart:      AgentName, SubTaskID
//   - EventSubAgentEnd:        AgentName, SubTaskID, Content, Usage, Duration
//   - EventOrchestratorContent: Content (per-token chunk)
//   - EventOrchestratorDone:   FinalOutput, Usage
//   - EventNetworkError:       Error, AgentName (optional — empty for global errors)
type NetworkStreamEvent struct {
	// Type identifies the event kind.
	Type NetworkStreamEventType

	// AgentName identifies the source agent.
	AgentName string

	// SubTaskID correlates start/end events for the same sub-task.
	SubTaskID string

	// Content carries the text payload: a per-token chunk for orchestrator_content,
	// or the full agent output for sub_agent_end.
	Content string

	// Usage holds token consumption info, populated on sub_agent_end and orchestrator_done.
	Usage *model.Usage

	// FinalOutput is the orchestrator's synthesized output, populated only on orchestrator_done.
	FinalOutput interface{}

	// Duration is the wall-clock time for the sub-agent run, populated on sub_agent_end.
	Duration time.Duration

	// Error is the error that caused a network_error event.
	Error error
}

// AgentRunResult holds the outcome of running a single agent within the network.
type AgentRunResult struct {
	// AgentName is the name of the agent that was run.
	AgentName string

	// Role is the role this agent played in the network (from AgentSlot.Role).
	Role string

	// RunResult is the result returned by the runner for this agent.
	// May be nil if the run failed before producing a result.
	RunResult *result.RunResult

	// Error holds any error returned by the runner for this agent.
	Error error

	// Duration is the wall-clock time taken by this agent's run.
	Duration time.Duration
}

// NetworkResult is the outcome of a full network run.
type NetworkResult struct {
	// AgentResults holds one entry per dispatched agent.
	// For StrategyParallel the order matches the roster order.
	// For StrategySequential the order matches the execution chain.
	// For StrategyCompetitive there is exactly one entry (the winner).
	AgentResults []AgentRunResult

	// FinalOutput is the orchestrator's synthesized answer, or the winning
	// agent's FinalOutput in StrategyCompetitive mode when synthesis is skipped.
	FinalOutput interface{}

	// OrchestratorResult holds the orchestrator's own RunResult from the
	// synthesis step. May be nil if synthesis was bypassed.
	OrchestratorResult *result.RunResult

	// Strategy is the dispatch strategy that was used for this run.
	Strategy DispatchStrategy

	// LastAgent is the agent that produced FinalOutput.
	LastAgent *agent.Agent
}
