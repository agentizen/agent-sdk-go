package network

import (
	"errors"
	"fmt"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
)

// DispatchStrategy defines how sub-tasks are distributed across agents in a network.
type DispatchStrategy string

const (
	// StrategyParallel runs all agents concurrently on their assigned sub-tasks
	// and aggregates results once all complete.
	StrategyParallel DispatchStrategy = "parallel"

	// StrategySequential runs agents one after another; each agent receives the
	// previous agent's final output as its input.
	StrategySequential DispatchStrategy = "sequential"

	// StrategyCompetitive sends the same prompt to all agents simultaneously;
	// the first agent to respond without error wins and the others are cancelled.
	StrategyCompetitive DispatchStrategy = "competitive"
)

// AgentSlot binds an *agent.Agent to a named role within the network.
type AgentSlot struct {
	// Agent is the agent to include in the network. Must be non-nil.
	Agent *agent.Agent

	// Role is a human-readable description of the agent's purpose in the network.
	Role string

	// SubTaskHint is an optional hint provided to the orchestrator when assigning
	// sub-tasks to this agent. Leave empty to let the orchestrator decide.
	SubTaskHint string
}

// NetworkConfig declares the agent roster, dispatch strategy, and optional orchestrator.
// It is value-typed and immutable after construction — all builder methods return a copy.
type NetworkConfig struct {
	// Agents is the roster of agents in the network.
	Agents []AgentSlot

	// Strategy determines how sub-tasks are dispatched. Defaults to StrategyParallel.
	Strategy DispatchStrategy

	// Orchestrator is the agent responsible for decomposing the prompt and synthesising
	// results. When nil, a built-in orchestrator is created automatically using the
	// model and provider from RunConfig.
	Orchestrator *agent.Agent

	// MaxConcurrency limits the number of agents that may run simultaneously.
	// Only applies to StrategyParallel. Zero means no limit.
	MaxConcurrency int
}

// NewNetworkConfig returns a default NetworkConfig with StrategyParallel and no agents.
func NewNetworkConfig() NetworkConfig {
	return NetworkConfig{
		Strategy: StrategyParallel,
		Agents:   []AgentSlot{},
	}
}

// WithAgents returns a new NetworkConfig with the given slots appended to the roster.
func (c NetworkConfig) WithAgents(slots ...AgentSlot) NetworkConfig {
	newAgents := make([]AgentSlot, len(c.Agents)+len(slots))
	copy(newAgents, c.Agents)
	copy(newAgents[len(c.Agents):], slots)
	c.Agents = newAgents
	return c
}

// WithStrategy returns a new NetworkConfig with the given dispatch strategy.
func (c NetworkConfig) WithStrategy(s DispatchStrategy) NetworkConfig {
	c.Strategy = s
	return c
}

// WithOrchestrator returns a new NetworkConfig with a custom orchestrator agent.
// Set to nil to revert to the built-in orchestrator.
func (c NetworkConfig) WithOrchestrator(o *agent.Agent) NetworkConfig {
	c.Orchestrator = o
	return c
}

// WithMaxConcurrency returns a new NetworkConfig with the given concurrency limit.
// Only meaningful for StrategyParallel. Pass 0 to disable the limit.
func (c NetworkConfig) WithMaxConcurrency(n int) NetworkConfig {
	c.MaxConcurrency = n
	return c
}

// Validate checks the NetworkConfig for correctness.
// Returns the first error found, or nil if the config is valid.
func (c NetworkConfig) Validate() error {
	if len(c.Agents) == 0 {
		return errors.New("network: at least one AgentSlot is required")
	}

	seen := make(map[string]struct{}, len(c.Agents))
	for i, slot := range c.Agents {
		if slot.Agent == nil {
			return fmt.Errorf("network: AgentSlot[%d] has a nil Agent", i)
		}
		name := slot.Agent.Name
		if _, exists := seen[name]; exists {
			return fmt.Errorf("network: duplicate agent name %q in roster", name)
		}
		seen[name] = struct{}{}
	}

	switch c.Strategy {
	case StrategyParallel, StrategySequential, StrategyCompetitive:
		// valid
	default:
		return fmt.Errorf("network: unknown dispatch strategy %q", c.Strategy)
	}

	return nil
}
