package network

import (
	"time"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
	"github.com/citizenofai/agent-sdk-go/pkg/result"
)

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

	// FinalOutput is the orchestrator's synthesised answer, or the winning
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

// NetworkStreamEvent is a tagged streaming event originating from one agent in the network.
type NetworkStreamEvent struct {
	// AgentName identifies the source agent.
	AgentName string

	// SubTaskID is an optional identifier correlating this event to a specific sub-task.
	SubTaskID string

	// StreamEvent is the underlying event from the agent's stream.
	StreamEvent interface{}

	// IsFinal is true when this event carries the orchestrator's final synthesis result.
	IsFinal bool
}
