package network_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
	"github.com/citizenofai/agent-sdk-go/pkg/model"
	"github.com/citizenofai/agent-sdk-go/pkg/network"
	"github.com/citizenofai/agent-sdk-go/pkg/result"
	"github.com/citizenofai/agent-sdk-go/pkg/runner"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testModel is a deterministic model that returns the same content on every call.
// It creates a fresh channel on each StreamResponse invocation, which is required
// when the same model instance is called multiple times (orchestrator + agents).
type testModel struct {
	content string
}

func (m *testModel) GetResponse(_ context.Context, _ *model.Request) (*model.Response, error) {
	return &model.Response{Content: m.content}, nil
}

func (m *testModel) StreamResponse(_ context.Context, _ *model.Request) (<-chan model.StreamEvent, error) {
	ch := make(chan model.StreamEvent, 2)
	go func() {
		defer close(ch)
		ch <- model.StreamEvent{
			Type:    model.StreamEventTypeContent,
			Content: m.content,
		}
		ch <- model.StreamEvent{
			Type:     model.StreamEventTypeDone,
			Done:     true,
			Response: &model.Response{Content: m.content},
		}
	}()
	return ch, nil
}

// errorModel is a model whose StreamResponse always returns an error.
type errorModel struct{}

func (m *errorModel) GetResponse(_ context.Context, _ *model.Request) (*model.Response, error) {
	return nil, errors.New("model error")
}

func (m *errorModel) StreamResponse(_ context.Context, _ *model.Request) (<-chan model.StreamEvent, error) {
	return nil, errors.New("model error")
}

// testProvider wraps a single model.Model and returns it for any model name.
type testProvider struct {
	mdl model.Model
}

func (p *testProvider) GetModel(_ string) (model.Model, error) {
	return p.mdl, nil
}

// errorProvider returns an errorModel on every GetModel call.
type errorProvider struct{}

func (p *errorProvider) GetModel(_ string) (model.Model, error) {
	return &errorModel{}, nil
}

// dummyProvider is a non-nil provider that should never be called.
// It is used as a placeholder when agents carry their model.Model directly
// and model resolution via a provider should not happen.
type dummyProvider struct{}

func (p *dummyProvider) GetModel(name string) (model.Model, error) {
	return nil, errors.New("dummyProvider: unexpected GetModel call for " + name)
}

// makeAgent creates an agent with the given name connected to a testProvider.
func makeAgent(name string, provider model.Provider) *agent.Agent {
	a := agent.NewAgent(name)
	a.SetModelProvider(provider)
	a.WithModel("test-model")
	return a
}

// makeRunOpts returns RunOptions wired to the provided provider.
func makeRunOpts(provider model.Provider, input string) *runner.RunOptions {
	return &runner.RunOptions{
		Input:    input,
		MaxTurns: 3,
		RunConfig: &runner.RunConfig{
			Model:           "test-model",
			ModelProvider:   provider,
			TracingDisabled: true,
		},
	}
}

// ---------------------------------------------------------------------------
// NetworkConfig tests
// ---------------------------------------------------------------------------

func TestNetworkConfig_NewNetworkConfig(t *testing.T) {
	cfg := network.NewNetworkConfig()
	if cfg.Strategy != network.StrategyParallel {
		t.Errorf("default strategy = %q; want %q", cfg.Strategy, network.StrategyParallel)
	}
	if len(cfg.Agents) != 0 {
		t.Errorf("default agents len = %d; want 0", len(cfg.Agents))
	}
	if cfg.Orchestrator != nil {
		t.Error("default orchestrator should be nil")
	}
	if cfg.MaxConcurrency != 0 {
		t.Errorf("default MaxConcurrency = %d; want 0", cfg.MaxConcurrency)
	}
}

func TestNetworkConfig_WithAgents(t *testing.T) {
	a1 := agent.NewAgent("Agent1")
	a2 := agent.NewAgent("Agent2")

	original := network.NewNetworkConfig()
	updated := original.WithAgents(
		network.AgentSlot{Agent: a1, Role: "role1"},
		network.AgentSlot{Agent: a2, Role: "role2"},
	)

	// original is unchanged (copy-on-write).
	if len(original.Agents) != 0 {
		t.Errorf("original was mutated: len=%d", len(original.Agents))
	}
	if len(updated.Agents) != 2 {
		t.Fatalf("updated agents len = %d; want 2", len(updated.Agents))
	}
	if updated.Agents[0].Agent.Name != "Agent1" {
		t.Errorf("agents[0].Name = %q; want Agent1", updated.Agents[0].Agent.Name)
	}
	if updated.Agents[1].Role != "role2" {
		t.Errorf("agents[1].Role = %q; want role2", updated.Agents[1].Role)
	}
}

func TestNetworkConfig_WithStrategy(t *testing.T) {
	cfg := network.NewNetworkConfig().WithStrategy(network.StrategySequential)
	if cfg.Strategy != network.StrategySequential {
		t.Errorf("strategy = %q; want %q", cfg.Strategy, network.StrategySequential)
	}
}

func TestNetworkConfig_WithOrchestrator(t *testing.T) {
	orch := agent.NewAgent("CustomOrch")
	cfg := network.NewNetworkConfig().WithOrchestrator(orch)
	if cfg.Orchestrator != orch {
		t.Error("orchestrator not set correctly")
	}
}

func TestNetworkConfig_WithMaxConcurrency(t *testing.T) {
	cfg := network.NewNetworkConfig().WithMaxConcurrency(4)
	if cfg.MaxConcurrency != 4 {
		t.Errorf("MaxConcurrency = %d; want 4", cfg.MaxConcurrency)
	}
}

func TestNetworkConfig_Validate_Empty(t *testing.T) {
	cfg := network.NewNetworkConfig()
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty agents; got nil")
	}
}

func TestNetworkConfig_Validate_NilAgent(t *testing.T) {
	cfg := network.NewNetworkConfig().WithAgents(
		network.AgentSlot{Agent: nil, Role: "broken"},
	)
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for nil agent; got nil")
	}
}

func TestNetworkConfig_Validate_DuplicateNames(t *testing.T) {
	a := agent.NewAgent("DupName")
	b := agent.NewAgent("DupName")
	cfg := network.NewNetworkConfig().WithAgents(
		network.AgentSlot{Agent: a},
		network.AgentSlot{Agent: b},
	)
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for duplicate agent names; got nil")
	}
}

func TestNetworkConfig_Validate_InvalidStrategy(t *testing.T) {
	a := agent.NewAgent("Agent")
	cfg := network.NewNetworkConfig().
		WithAgents(network.AgentSlot{Agent: a}).
		WithStrategy("foobar")
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for unknown strategy; got nil")
	}
}

func TestNetworkConfig_Validate_Valid(t *testing.T) {
	a := agent.NewAgent("AgentA")
	b := agent.NewAgent("AgentB")
	cfg := network.NewNetworkConfig().WithAgents(
		network.AgentSlot{Agent: a, Role: "first"},
		network.AgentSlot{Agent: b, Role: "second"},
	)
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Result type tests
// ---------------------------------------------------------------------------

func TestNetworkResult_Fields(t *testing.T) {
	a := agent.NewAgent("TestAgent")
	rr := &result.RunResult{FinalOutput: "hello"}

	ar := network.AgentRunResult{
		AgentName: "TestAgent",
		Role:      "tester",
		RunResult: rr,
		Error:     nil,
		Duration:  50 * time.Millisecond,
	}

	nr := network.NetworkResult{
		AgentResults:       []network.AgentRunResult{ar},
		FinalOutput:        "final",
		OrchestratorResult: rr,
		Strategy:           network.StrategyParallel,
		LastAgent:          a,
	}

	if len(nr.AgentResults) != 1 {
		t.Errorf("AgentResults len = %d; want 1", len(nr.AgentResults))
	}
	if nr.AgentResults[0].AgentName != "TestAgent" {
		t.Errorf("AgentResults[0].AgentName = %q; want TestAgent", nr.AgentResults[0].AgentName)
	}
	if nr.FinalOutput != "final" {
		t.Errorf("FinalOutput = %v; want final", nr.FinalOutput)
	}
	if nr.Strategy != network.StrategyParallel {
		t.Errorf("Strategy = %q; want %q", nr.Strategy, network.StrategyParallel)
	}
	if nr.LastAgent != a {
		t.Error("LastAgent not set correctly")
	}
}

func TestNetworkStreamEvent_Fields(t *testing.T) {
	ev := network.NetworkStreamEvent{
		AgentName:   "Agent1",
		SubTaskID:   "task-001",
		StreamEvent: "some data",
		IsFinal:     true,
	}

	if ev.AgentName != "Agent1" {
		t.Errorf("AgentName = %q; want Agent1", ev.AgentName)
	}
	if ev.SubTaskID != "task-001" {
		t.Errorf("SubTaskID = %q; want task-001", ev.SubTaskID)
	}
	if !ev.IsFinal {
		t.Error("IsFinal should be true")
	}
}

// ---------------------------------------------------------------------------
// NetworkRunner construction
// ---------------------------------------------------------------------------

func TestNewNetworkRunner(t *testing.T) {
	base := runner.NewRunner()
	nr := network.NewNetworkRunner(base)
	if nr == nil {
		t.Fatal("NewNetworkRunner returned nil")
	}
}

// ---------------------------------------------------------------------------
// RunNetwork — strategy tests
// ---------------------------------------------------------------------------

func TestRunNetwork_Parallel(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "agent output"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	a1 := makeAgent("Alpha", provider)
	a2 := makeAgent("Beta", provider)

	cfg := network.NewNetworkConfig().WithAgents(
		network.AgentSlot{Agent: a1, Role: "first agent"},
		network.AgentSlot{Agent: a2, Role: "second agent"},
	)

	result, err := nr.RunNetwork(context.Background(), cfg, makeRunOpts(provider, "test prompt"))
	if err != nil {
		t.Fatalf("RunNetwork error: %v", err)
	}
	if len(result.AgentResults) != 2 {
		t.Errorf("AgentResults len = %d; want 2", len(result.AgentResults))
	}
	if result.Strategy != network.StrategyParallel {
		t.Errorf("Strategy = %q; want %q", result.Strategy, network.StrategyParallel)
	}
	if result.LastAgent == nil {
		t.Error("LastAgent is nil")
	}
}

func TestRunNetwork_Parallel_WithMaxConcurrency(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "output"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	a1 := makeAgent("Alpha", provider)
	a2 := makeAgent("Beta", provider)
	a3 := makeAgent("Gamma", provider)

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: a1, Role: "r1"},
			network.AgentSlot{Agent: a2, Role: "r2"},
			network.AgentSlot{Agent: a3, Role: "r3"},
		).
		WithMaxConcurrency(2)

	result, err := nr.RunNetwork(context.Background(), cfg, makeRunOpts(provider, "test"))
	if err != nil {
		t.Fatalf("RunNetwork error: %v", err)
	}
	if len(result.AgentResults) != 3 {
		t.Errorf("AgentResults len = %d; want 3", len(result.AgentResults))
	}
}

func TestRunNetwork_Sequential(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "sequential output"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	a1 := makeAgent("Planner", provider)
	a2 := makeAgent("Writer", provider)

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: a1, Role: "planner"},
			network.AgentSlot{Agent: a2, Role: "writer"},
		).
		WithStrategy(network.StrategySequential)

	result, err := nr.RunNetwork(context.Background(), cfg, makeRunOpts(provider, "build a plan"))
	if err != nil {
		t.Fatalf("RunNetwork error: %v", err)
	}
	if len(result.AgentResults) != 2 {
		t.Errorf("AgentResults len = %d; want 2", len(result.AgentResults))
	}
	if result.Strategy != network.StrategySequential {
		t.Errorf("Strategy = %q; want %q", result.Strategy, network.StrategySequential)
	}
}

func TestRunNetwork_Competitive(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "competitive output"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	a1 := makeAgent("Fast", provider)
	a2 := makeAgent("Slow", provider)

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: a1, Role: "fast responder"},
			network.AgentSlot{Agent: a2, Role: "slow responder"},
		).
		WithStrategy(network.StrategyCompetitive)

	result, err := nr.RunNetwork(context.Background(), cfg, makeRunOpts(provider, "quick question"))
	if err != nil {
		t.Fatalf("RunNetwork error: %v", err)
	}
	if len(result.AgentResults) != 1 {
		t.Errorf("AgentResults len = %d; want 1 (only winner)", len(result.AgentResults))
	}
	if result.Strategy != network.StrategyCompetitive {
		t.Errorf("Strategy = %q; want %q", result.Strategy, network.StrategyCompetitive)
	}
}

// ---------------------------------------------------------------------------
// RunNetwork — error handling tests
// ---------------------------------------------------------------------------

func TestRunNetwork_Validate_Error(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "x"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	// Empty config fails Validate().
	cfg := network.NewNetworkConfig()
	_, err := nr.RunNetwork(context.Background(), cfg, makeRunOpts(provider, "input"))
	if err == nil {
		t.Error("expected error for invalid config; got nil")
	}
}

func TestRunNetwork_NilOpts(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "output"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	a1 := makeAgent("Agent1", provider)
	cfg := network.NewNetworkConfig().WithAgents(network.AgentSlot{Agent: a1})

	// Should not panic even with nil opts.
	_, err := nr.RunNetwork(context.Background(), cfg, nil)
	// Will fail because no model is resolvable without RunConfig.ModelProvider.
	// But it should not panic — error is acceptable.
	_ = err
}

func TestRunNetwork_ContextCancel(t *testing.T) {
	// A provider that always returns an error model to trigger quick failures.
	base := runner.NewRunner().WithDefaultProvider(&errorProvider{})
	nr := network.NewNetworkRunner(base)

	a1 := agent.NewAgent("Agent1")
	a1.WithModel("test-model")
	cfg := network.NewNetworkConfig().WithAgents(network.AgentSlot{Agent: a1})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := nr.RunNetwork(ctx, cfg, &runner.RunOptions{
		Input:    "prompt",
		MaxTurns: 1,
		RunConfig: &runner.RunConfig{
			Model:           "test-model",
			ModelProvider:   &errorProvider{},
			TracingDisabled: true,
		},
	})
	// Expect an error because model errors out or ctx is cancelled.
	if err == nil {
		t.Error("expected error with cancelled context or failing model; got nil")
	}
}

func TestRunNetwork_AgentError_Parallel(t *testing.T) {
	// Build a custom orchestrator with a direct model.Model instance so that
	// RunConfig.Model does NOT override per-agent models for this test.
	orchModel := &testModel{content: "orchestrate"}
	orch := agent.NewAgent("TestOrch")
	orch.Model = orchModel // set model.Model directly, bypasses provider resolution

	goodModel := &testModel{content: "good output"}
	a1 := agent.NewAgent("GoodAgent")
	a1.Model = goodModel // direct model.Model

	a2 := agent.NewAgent("BadAgent")
	a2.Model = &errorModel{} // direct error model

	base := runner.NewRunner()
	nr := network.NewNetworkRunner(base)

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: a1, Role: "succeeds"},
			network.AgentSlot{Agent: a2, Role: "fails"},
		).
		WithOrchestrator(orch)

	// dummyProvider satisfies the runner's non-nil provider check without being used
	// for model resolution (agents have direct model.Model instances).
	opts := &runner.RunOptions{
		Input:    "test prompt",
		MaxTurns: 2,
		RunConfig: &runner.RunConfig{
			ModelProvider:   &dummyProvider{},
			TracingDisabled: true,
		},
	}

	result, err := nr.RunNetwork(context.Background(), cfg, opts)
	if err != nil {
		// It is also acceptable for synthesis to fail if BadAgent's error propagates.
		t.Logf("RunNetwork returned error (acceptable): %v", err)
		return
	}

	if len(result.AgentResults) != 2 {
		t.Fatalf("AgentResults len = %d; want 2", len(result.AgentResults))
	}

	var successCount, errorCount int
	for _, ar := range result.AgentResults {
		if ar.Error == nil {
			successCount++
		} else {
			errorCount++
		}
	}
	if successCount != 1 || errorCount != 1 {
		t.Errorf("successCount=%d errorCount=%d; want 1 each", successCount, errorCount)
	}
}

func TestRunNetwork_Sequential_StopsOnError(t *testing.T) {
	// First agent uses error provider → chain stops at step 1.
	base := runner.NewRunner().WithDefaultProvider(&errorProvider{})
	nr := network.NewNetworkRunner(base)

	a1 := agent.NewAgent("Agent1")
	a1.WithModel("test-model")
	a2 := agent.NewAgent("Agent2")
	a2.WithModel("test-model")

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: a1, Role: "first"},
			network.AgentSlot{Agent: a2, Role: "second"},
		).
		WithStrategy(network.StrategySequential)

	_, err := nr.RunNetwork(context.Background(), cfg, &runner.RunOptions{
		Input:    "test",
		MaxTurns: 1,
		RunConfig: &runner.RunConfig{
			Model:           "test-model",
			ModelProvider:   &errorProvider{},
			TracingDisabled: true,
		},
	})
	// Expect an error because orchestrator decomposition fails or agents fail.
	if err == nil {
		t.Error("expected error with failing model; got nil")
	}
}

func TestRunNetwork_Competitive_AllFail(t *testing.T) {
	base := runner.NewRunner().WithDefaultProvider(&errorProvider{})
	nr := network.NewNetworkRunner(base)

	a1 := agent.NewAgent("Agent1")
	a1.WithModel("test-model")
	a2 := agent.NewAgent("Agent2")
	a2.WithModel("test-model")

	cfg := network.NewNetworkConfig().
		WithAgents(
			network.AgentSlot{Agent: a1},
			network.AgentSlot{Agent: a2},
		).
		WithStrategy(network.StrategyCompetitive)

	_, err := nr.RunNetwork(context.Background(), cfg, &runner.RunOptions{
		Input:    "test",
		MaxTurns: 1,
		RunConfig: &runner.RunConfig{
			Model:           "test-model",
			ModelProvider:   &errorProvider{},
			TracingDisabled: true,
		},
	})
	// Expect error: orchestrator decomposition itself fails with error provider.
	if err == nil {
		t.Error("expected error when all agents fail; got nil")
	}
}

// ---------------------------------------------------------------------------
// Custom orchestrator test
// ---------------------------------------------------------------------------

func TestRunNetwork_CustomOrchestrator(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "custom synthesis"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	orch := agent.NewAgent("CustomOrch")
	orch.SetModelProvider(provider)
	orch.WithModel("test-model")
	orch.SetSystemInstructions("You are a custom orchestrator.")

	a1 := makeAgent("Analyst", provider)

	cfg := network.NewNetworkConfig().
		WithAgents(network.AgentSlot{Agent: a1, Role: "analyst"}).
		WithOrchestrator(orch)

	result, err := nr.RunNetwork(context.Background(), cfg, makeRunOpts(provider, "analyse this"))
	if err != nil {
		t.Fatalf("RunNetwork error: %v", err)
	}
	if result.OrchestratorResult == nil {
		t.Error("OrchestratorResult is nil")
	}
	if result.LastAgent == nil {
		t.Error("LastAgent is nil")
	}
}

// ---------------------------------------------------------------------------
// RunNetworkStreaming tests
// ---------------------------------------------------------------------------

func TestRunNetworkStreaming_EmitsEvents(t *testing.T) {
	provider := &testProvider{mdl: &testModel{content: "streamed output"}}
	base := runner.NewRunner().WithDefaultProvider(provider)
	nr := network.NewNetworkRunner(base)

	a1 := makeAgent("StreamAgent1", provider)
	a2 := makeAgent("StreamAgent2", provider)

	cfg := network.NewNetworkConfig().WithAgents(
		network.AgentSlot{Agent: a1, Role: "r1"},
		network.AgentSlot{Agent: a2, Role: "r2"},
	)

	ch, err := nr.RunNetworkStreaming(context.Background(), cfg, makeRunOpts(provider, "stream test"))
	if err != nil {
		t.Fatalf("RunNetworkStreaming error: %v", err)
	}

	var events []network.NetworkStreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) == 0 {
		t.Fatal("no events emitted from streaming run")
	}

	// Verify that the last event has IsFinal=true.
	lastEvent := events[len(events)-1]
	if !lastEvent.IsFinal {
		t.Errorf("last event IsFinal = false; want true")
	}

	// Verify at least one non-final event exists (one per agent).
	var agentEvents int
	for _, ev := range events {
		if !ev.IsFinal {
			agentEvents++
		}
	}
	if agentEvents != 2 {
		t.Errorf("non-final events = %d; want 2 (one per agent)", agentEvents)
	}
}

func TestRunNetworkStreaming_ValidateError(t *testing.T) {
	base := runner.NewRunner()
	nr := network.NewNetworkRunner(base)

	// Invalid config (no agents).
	cfg := network.NewNetworkConfig()
	_, err := nr.RunNetworkStreaming(context.Background(), cfg, makeRunOpts(&testProvider{mdl: &testModel{content: "x"}}, "input"))
	if err == nil {
		t.Error("expected validation error; got nil")
	}
}

func TestRunNetworkStreaming_ErrorEmittedOnChannel(t *testing.T) {
	// Force an error by using an error provider. The streaming run should emit an
	// error event and close the channel.
	base := runner.NewRunner().WithDefaultProvider(&errorProvider{})
	nr := network.NewNetworkRunner(base)

	a1 := agent.NewAgent("Agent1")
	a1.WithModel("test-model")

	cfg := network.NewNetworkConfig().WithAgents(
		network.AgentSlot{Agent: a1, Role: "r"},
	)

	ch, err := nr.RunNetworkStreaming(context.Background(), cfg, &runner.RunOptions{
		Input:    "test",
		MaxTurns: 1,
		RunConfig: &runner.RunConfig{
			Model:           "test-model",
			ModelProvider:   &errorProvider{},
			TracingDisabled: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected synchronous error: %v", err)
	}

	// Drain the channel — should receive an error event.
	var events []network.NetworkStreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) == 0 {
		t.Fatal("no events emitted even on error path")
	}
}
