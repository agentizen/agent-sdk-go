package network

import (
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/agent"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
)

// ---------------------------------------------------------------------------
// extractJSON — brace-balanced scanner
// ---------------------------------------------------------------------------

func TestExtractJSON_SimpleObject(t *testing.T) {
	got := extractJSON(`{"Agent1": "task1", "Agent2": "task2"}`)
	want := `{"Agent1": "task1", "Agent2": "task2"}`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractJSON_SurroundingProse(t *testing.T) {
	got := extractJSON(`Here is the JSON: {"A": "x"} hope this helps!`)
	want := `{"A": "x"}`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractJSON_MarkdownCodeFence(t *testing.T) {
	got := extractJSON("```json\n{\"A\": \"task\"}\n```")
	want := `{"A": "task"}`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractJSON_NestedBraces(t *testing.T) {
	input := `{"A": {"nested": "value"}, "B": "task"}`
	got := extractJSON(input)
	if got != input {
		t.Errorf("got %q; want %q", got, input)
	}
}

func TestExtractJSON_BracesInProseBeforeJSON(t *testing.T) {
	input := `The format is {key: value}. Here's the real output: {"A": "task"}`
	got := extractJSON(input)
	// First balanced object is {key: value}
	want := `{key: value}`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractJSON_EscapedQuotesInStrings(t *testing.T) {
	input := `{"A": "task with \"braces {}\""}`
	got := extractJSON(input)
	if got != input {
		t.Errorf("got %q; want %q", got, input)
	}
}

func TestExtractJSON_NoBraces(t *testing.T) {
	input := "no json here"
	got := extractJSON(input)
	if got != input {
		t.Errorf("got %q; want %q", got, input)
	}
}

func TestExtractJSON_UnbalancedBraces(t *testing.T) {
	input := `prefix {"A": "task"`
	got := extractJSON(input)
	want := `{"A": "task"`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractJSON_EmptyObject(t *testing.T) {
	got := extractJSON(`some text {} more text`)
	want := `{}`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractJSON_WhitespaceOnly(t *testing.T) {
	got := extractJSON("   ")
	if got != "" {
		t.Errorf("got %q; want empty", got)
	}
}

func TestExtractJSON_BackslashOutsideString(t *testing.T) {
	input := `text\{"A": "B"}`
	got := extractJSON(input)
	want := `{"A": "B"}`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractJSON_StringWithBracesAndEscape(t *testing.T) {
	// JSON where a value contains escaped quotes and braces.
	input := `{"key": "val with \" and {inner}"}`
	got := extractJSON(input)
	if got != input {
		t.Errorf("got %q; want %q", got, input)
	}
}

// ---------------------------------------------------------------------------
// buildRosterDescription
// ---------------------------------------------------------------------------

func TestBuildRosterDescription_WithHint(t *testing.T) {
	slots := []AgentSlot{
		{Agent: agent.NewAgent("A"), Role: "researcher", SubTaskHint: "find data"},
	}
	got := buildRosterDescription(slots)
	want := `1. A — researcher (hint: find data)`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestBuildRosterDescription_WithoutHint(t *testing.T) {
	slots := []AgentSlot{
		{Agent: agent.NewAgent("B"), Role: "writer"},
	}
	got := buildRosterDescription(slots)
	want := `1. B — writer`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestBuildRosterDescription_EmptyRole(t *testing.T) {
	slots := []AgentSlot{
		{Agent: agent.NewAgent("C"), Role: ""},
	}
	got := buildRosterDescription(slots)
	want := `1. C — (no role specified)`
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestBuildRosterDescription_MultipleAgents(t *testing.T) {
	slots := []AgentSlot{
		{Agent: agent.NewAgent("A"), Role: "r1", SubTaskHint: "h1"},
		{Agent: agent.NewAgent("B"), Role: "r2"},
		{Agent: agent.NewAgent("C"), Role: ""},
	}
	got := buildRosterDescription(slots)
	want := "1. A — r1 (hint: h1)\n2. B — r2\n3. C — (no role specified)"
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// concurrencyLimit
// ---------------------------------------------------------------------------

func TestConcurrencyLimit_ZeroMeansAgentCount(t *testing.T) {
	cfg := NetworkConfig{Agents: make([]AgentSlot, 5), MaxConcurrency: 0}
	if got := concurrencyLimit(cfg); got != 5 {
		t.Errorf("got %d; want 5", got)
	}
}

func TestConcurrencyLimit_NegativeMeansAgentCount(t *testing.T) {
	cfg := NetworkConfig{Agents: make([]AgentSlot, 3), MaxConcurrency: -1}
	if got := concurrencyLimit(cfg); got != 3 {
		t.Errorf("got %d; want 3", got)
	}
}

func TestConcurrencyLimit_ExceedsAgentCount(t *testing.T) {
	cfg := NetworkConfig{Agents: make([]AgentSlot, 2), MaxConcurrency: 10}
	if got := concurrencyLimit(cfg); got != 2 {
		t.Errorf("got %d; want 2", got)
	}
}

func TestConcurrencyLimit_WithinRange(t *testing.T) {
	cfg := NetworkConfig{Agents: make([]AgentSlot, 5), MaxConcurrency: 3}
	if got := concurrencyLimit(cfg); got != 3 {
		t.Errorf("got %d; want 3", got)
	}
}

// ---------------------------------------------------------------------------
// agentRunOpts
// ---------------------------------------------------------------------------

func TestAgentRunOpts_NilRunConfig(t *testing.T) {
	parent := &runner.RunOptions{
		Input:    "original",
		MaxTurns: 7,
	}
	got := agentRunOpts(parent, "override")
	if got.Input != "override" {
		t.Errorf("Input = %v; want override", got.Input)
	}
	if got.MaxTurns != 7 {
		t.Errorf("MaxTurns = %d; want 7", got.MaxTurns)
	}
	if got.RunConfig != nil {
		t.Errorf("RunConfig should be nil when parent has nil RunConfig")
	}
}

func TestAgentRunOpts_WithRunConfig(t *testing.T) {
	parent := &runner.RunOptions{
		Input:    "original",
		MaxTurns: 3,
		RunConfig: &runner.RunConfig{
			Model: "some-model",
		},
	}
	got := agentRunOpts(parent, "new-input")
	if got.RunConfig == nil {
		t.Fatal("RunConfig should not be nil")
	}
	// Must be a copy, not the same pointer.
	if got.RunConfig == parent.RunConfig {
		t.Error("RunConfig should be a shallow copy, not the same pointer")
	}
	if got.RunConfig.Model != "some-model" {
		t.Errorf("RunConfig.Model = %v; want some-model", got.RunConfig.Model)
	}
}

// ---------------------------------------------------------------------------
// newBuiltInOrchestrator
// ---------------------------------------------------------------------------

func TestNewBuiltInOrchestrator_NilRunConfig(t *testing.T) {
	cfg := NetworkConfig{
		Agents: []AgentSlot{
			{Agent: agent.NewAgent("A"), Role: "r1"},
		},
	}
	orch := newBuiltInOrchestrator(cfg, nil)
	if orch == nil {
		t.Fatal("orchestrator should not be nil")
	}
	if orch.Name != "NetworkOrchestrator" {
		t.Errorf("Name = %q; want NetworkOrchestrator", orch.Name)
	}
	// Model should be nil (not empty string) when RunConfig is nil.
	if orch.Model != nil {
		t.Errorf("Model = %v; want nil", orch.Model)
	}
}

func TestNewBuiltInOrchestrator_WithModel(t *testing.T) {
	cfg := NetworkConfig{
		Agents: []AgentSlot{
			{Agent: agent.NewAgent("A"), Role: "r1"},
		},
	}
	rc := &runner.RunConfig{Model: "gpt-4.1"}
	orch := newBuiltInOrchestrator(cfg, rc)
	if orch.Model != "gpt-4.1" {
		t.Errorf("Model = %v; want gpt-4.1", orch.Model)
	}
}

// ---------------------------------------------------------------------------
// buildDecompositionSchema
// ---------------------------------------------------------------------------

func TestBuildDecompositionSchema(t *testing.T) {
	slots := []AgentSlot{
		{Agent: agent.NewAgent("Researcher"), Role: "Research"},
		{Agent: agent.NewAgent("Writer"), Role: "Writing"},
	}
	schema := buildDecompositionSchema(slots)

	// Verify type
	if schema["type"] != "object" {
		t.Errorf("schema type = %v; want object", schema["type"])
	}

	// Verify additionalProperties
	if schema["additionalProperties"] != false {
		t.Errorf("additionalProperties = %v; want false", schema["additionalProperties"])
	}

	// Verify properties
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties is not a map")
	}
	if len(props) != 2 {
		t.Errorf("properties count = %d; want 2", len(props))
	}
	for _, name := range []string{"Researcher", "Writer"} {
		p, exists := props[name]
		if !exists {
			t.Errorf("missing property %q", name)
			continue
		}
		pm, _ := p.(map[string]interface{})
		if pm["type"] != "string" {
			t.Errorf("property %q type = %v; want string", name, pm["type"])
		}
	}

	// Verify required
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required is not a string slice")
	}
	if len(required) != 2 {
		t.Errorf("required count = %d; want 2", len(required))
	}
}

func TestNewBuiltInOrchestrator_NilModel(t *testing.T) {
	cfg := NetworkConfig{
		Agents: []AgentSlot{
			{Agent: agent.NewAgent("A"), Role: "r1"},
		},
	}
	rc := &runner.RunConfig{Model: nil}
	orch := newBuiltInOrchestrator(cfg, rc)
	if orch.Model != nil {
		t.Errorf("Model = %v; want nil when RunConfig.Model is nil", orch.Model)
	}
}
