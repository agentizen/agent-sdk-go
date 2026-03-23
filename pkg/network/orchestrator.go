package network

import (
	"fmt"
	"strings"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
	"github.com/citizenofai/agent-sdk-go/pkg/runner"
)

// buildRosterDescription formats the agent roster into a human-readable list
// suitable for embedding in the orchestrator's system instructions.
func buildRosterDescription(slots []AgentSlot) string {
	var sb strings.Builder
	for i, slot := range slots {
		name := slot.Agent.Name
		role := slot.Role
		if role == "" {
			role = "(no role specified)"
		}
		hint := slot.SubTaskHint
		if hint != "" {
			fmt.Fprintf(&sb, "%d. %s — %s (hint: %s)\n", i+1, name, role, hint)
		} else {
			fmt.Fprintf(&sb, "%d. %s — %s\n", i+1, name, role)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// newBuiltInOrchestrator creates a ready-to-use orchestrator *agent.Agent that knows
// how to decompose a user prompt into sub-tasks (DECOMPOSITION MODE) and synthesise
// the collected results into a final answer (SYNTHESIS MODE).
//
// The orchestrator's model is set from runCfg.Model when non-nil;
// otherwise an empty string is used, which causes the runner to call
// provider.GetModel("") — resolving to the provider's default model.
// The model provider is NOT set on the agent itself; it is inherited from
// opts.RunConfig.ModelProvider (or the runner's default provider) at execution time.
func newBuiltInOrchestrator(cfg NetworkConfig, runCfg *runner.RunConfig) *agent.Agent {
	roster := buildRosterDescription(cfg.Agents)

	// Build agent name list for the decomposition JSON keys.
	agentNames := make([]string, len(cfg.Agents))
	for i, s := range cfg.Agents {
		agentNames[i] = fmt.Sprintf("%q", s.Agent.Name)
	}
	namesJoined := strings.Join(agentNames, ", ")

	instructions := fmt.Sprintf(`You are a network orchestrator coordinating %d specialised agents.

Agent roster:
%s

You operate in two modes depending on the input you receive:

DECOMPOSITION MODE — triggered when the input does NOT start with "RESULTS:".
Analyse the user prompt and assign a focused sub-task to each agent.
Respond ONLY with a valid JSON object (no markdown, no explanation) where every key
is an agent name from the roster and every value is a specific, self-contained sub-task string.
All %d agent names must appear as keys: %s.
Example: {"AgentA": "Research topic X thoroughly", "AgentB": "Write a 200-word summary of X"}

SYNTHESIS MODE — triggered when the input starts with "RESULTS:".
Read all the sub-results provided and synthesise a single, coherent, comprehensive final answer
that incorporates all agent findings. Write a clear, well-structured response.`,
		len(cfg.Agents), roster, len(cfg.Agents), namesJoined)

	orch := agent.NewAgent("NetworkOrchestrator", instructions)

	// Determine which model the orchestrator should use.
	// Using an empty string causes the runner to call provider.GetModel("") which
	// resolves to the provider's configured default model.
	var orchModel interface{} = ""
	if runCfg != nil && runCfg.Model != nil {
		orchModel = runCfg.Model
	}
	orch.WithModel(orchModel)

	return orch
}
