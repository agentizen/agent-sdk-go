package result_test

import (
	"errors"
	"testing"
	"time"

	"github.com/citizenofai/agent-sdk-go/pkg/agent"
	"github.com/citizenofai/agent-sdk-go/pkg/model"
	"github.com/citizenofai/agent-sdk-go/pkg/result"
	"github.com/stretchr/testify/assert"
)

// ---- RunItem types ----

func TestMessageItem(t *testing.T) {
	item := &result.MessageItem{
		Role:    "user",
		Content: "Hello, world!",
	}

	assert.Equal(t, "message", item.GetType())

	inputItem := item.ToInputItem()
	m, ok := inputItem.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "message", m["type"])
	assert.Equal(t, "user", m["role"])
	assert.Equal(t, "Hello, world!", m["content"])
}

func TestToolCallItem(t *testing.T) {
	item := &result.ToolCallItem{
		Name:       "my_tool",
		Parameters: map[string]interface{}{"param1": "value1"},
	}

	assert.Equal(t, "tool_call", item.GetType())

	inputItem := item.ToInputItem()
	m, ok := inputItem.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "tool_call", m["type"])
	assert.Equal(t, "my_tool", m["name"])
	params, ok := m["parameters"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "value1", params["param1"])
}

func TestToolResultItem(t *testing.T) {
	item := &result.ToolResultItem{
		Name:   "my_tool",
		Result: "tool output",
	}

	assert.Equal(t, "tool_result", item.GetType())

	inputItem := item.ToInputItem()
	m, ok := inputItem.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "tool_result", m["type"])
	assert.Equal(t, "my_tool", m["name"])
	assert.Equal(t, "tool output", m["result"])
}

func TestHandoffItem(t *testing.T) {
	item := &result.HandoffItem{
		AgentName: "other_agent",
		Input:     "some input",
	}

	assert.Equal(t, "handoff", item.GetType())

	inputItem := item.ToInputItem()
	m, ok := inputItem.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "handoff", m["type"])
	assert.Equal(t, "other_agent", m["agent_name"])
	assert.Equal(t, "some input", m["input"])
}

// ---- RunResult ----

func TestRunResult_ToInputList_StringInput(t *testing.T) {
	r := &result.RunResult{
		Input: "hello",
		NewItems: []result.RunItem{
			&result.MessageItem{Role: "assistant", Content: "hi there"},
		},
	}

	list := r.ToInputList()
	assert.Len(t, list, 2)

	first, ok := list[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "message", first["type"])
	assert.Equal(t, "user", first["role"])
	assert.Equal(t, "hello", first["content"])

	second, ok := list[1].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "message", second["type"])
	assert.Equal(t, "assistant", second["role"])
}

func TestRunResult_ToInputList_ListInput(t *testing.T) {
	existingItems := []interface{}{
		map[string]interface{}{"type": "message", "role": "user", "content": "prev message"},
	}
	r := &result.RunResult{
		Input:    existingItems,
		NewItems: []result.RunItem{},
	}

	list := r.ToInputList()
	assert.Len(t, list, 1)
	first, ok := list[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "prev message", first["content"])
}

func TestRunResult_ToInputList_NoInput(t *testing.T) {
	r := &result.RunResult{
		Input:    nil,
		NewItems: []result.RunItem{},
	}

	list := r.ToInputList()
	assert.Empty(t, list)
}

func TestRunResult_WithRawResponses(t *testing.T) {
	ag := agent.NewAgent()
	rawResp := model.Response{Content: "raw"}
	r := &result.RunResult{
		Input:        "test",
		RawResponses: []model.Response{rawResp},
		FinalOutput:  "final answer",
		LastAgent:    ag,
	}

	assert.Len(t, r.RawResponses, 1)
	assert.Equal(t, "raw", r.RawResponses[0].Content)
	assert.Equal(t, "final answer", r.FinalOutput)
	assert.Equal(t, ag, r.LastAgent)
}

func TestRunResult_GuardrailResults(t *testing.T) {
	r := &result.RunResult{
		InputGuardrailResults: []result.GuardrailResult{
			{Name: "input_check", Passed: true, Message: "ok"},
		},
		OutputGuardrailResults: []result.GuardrailResult{
			{Name: "output_check", Passed: false, Message: "failed", Error: errors.New("blocked")},
		},
	}

	assert.Len(t, r.InputGuardrailResults, 1)
	assert.True(t, r.InputGuardrailResults[0].Passed)

	assert.Len(t, r.OutputGuardrailResults, 1)
	assert.False(t, r.OutputGuardrailResults[0].Passed)
	assert.NotNil(t, r.OutputGuardrailResults[0].Error)
}

// ---- StreamEvent helpers ----

func TestContentEvent(t *testing.T) {
	e := result.ContentEvent("hello")
	assert.Equal(t, "content", e.Type)
	assert.Equal(t, "hello", e.Content)
	assert.Nil(t, e.Error)
}

func TestItemEvent(t *testing.T) {
	item := &result.MessageItem{Role: "user", Content: "msg"}
	e := result.ItemEvent(item)
	assert.Equal(t, "item", e.Type)
	assert.Equal(t, item, e.Item)
}

func TestAgentEvent(t *testing.T) {
	ag := agent.NewAgent()
	e := result.AgentEvent(ag)
	assert.Equal(t, "agent", e.Type)
	assert.Equal(t, ag, e.Agent)
}

func TestTurnEvent(t *testing.T) {
	e := result.TurnEvent(3)
	assert.Equal(t, "turn", e.Type)
	assert.Equal(t, 3, e.Turn)
}

func TestDoneEvent(t *testing.T) {
	e := result.DoneEvent()
	assert.Equal(t, "done", e.Type)
	assert.True(t, e.Done)
}

func TestErrorEvent(t *testing.T) {
	err := errors.New("something went wrong")
	e := result.ErrorEvent(err)
	assert.Equal(t, "error", e.Type)
	assert.Equal(t, err, e.Error)
}

// ---- StreamedRunResult ----

func TestStreamedRunResult_Fields(t *testing.T) {
	base := &result.RunResult{Input: "test"}
	ch := make(chan model.StreamEvent)
	recvCh := (<-chan model.StreamEvent)(ch)
	ag := agent.NewAgent()

	s := &result.StreamedRunResult{
		RunResult:    base,
		Stream:       recvCh,
		IsComplete:   false,
		CurrentAgent: ag,
		CurrentInput: "current",
		ContinueLoop: true,
		CurrentTurn:  2,
		ActiveTasks:  map[string]*result.TaskContext{},
		DelegationHistory: map[string][]string{
			"agent1": {"agent2"},
		},
	}

	assert.Equal(t, base, s.RunResult)
	assert.Equal(t, recvCh, s.Stream)
	assert.False(t, s.IsComplete)
	assert.Equal(t, ag, s.CurrentAgent)
	assert.Equal(t, "current", s.CurrentInput)
	assert.True(t, s.ContinueLoop)
	assert.Equal(t, 2, s.CurrentTurn)
	assert.NotNil(t, s.ActiveTasks)
	assert.Len(t, s.DelegationHistory["agent1"], 1)
}

// ---- TaskContext ----

func TestTaskContext_Constants(t *testing.T) {
	assert.Equal(t, result.TaskStatus("pending"), result.TaskStatusPending)
	assert.Equal(t, result.TaskStatus("complete"), result.TaskStatusComplete)
	assert.Equal(t, result.TaskStatus("failed"), result.TaskStatusFailed)
}

func TestTaskContext_Fields(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Second)

	tc := &result.TaskContext{
		TaskID:          "task-1",
		ParentAgentName: "parent",
		ChildAgentName:  "child",
		Status:          result.TaskStatusPending,
		CreatedAt:       now,
		CompletedAt:     &completed,
		Result:          "done",
	}

	assert.Equal(t, "task-1", tc.TaskID)
	assert.Equal(t, "parent", tc.ParentAgentName)
	assert.Equal(t, "child", tc.ChildAgentName)
	assert.Equal(t, result.TaskStatusPending, tc.Status)
	assert.Equal(t, now, tc.CreatedAt)
	assert.Equal(t, &completed, tc.CompletedAt)
	assert.Equal(t, "done", tc.Result)
}

func TestTaskContext_NilCompletedAt(t *testing.T) {
	tc := &result.TaskContext{
		TaskID:    "task-2",
		Status:    result.TaskStatusPending,
		CreatedAt: time.Now(),
	}

	assert.Nil(t, tc.CompletedAt)
}
