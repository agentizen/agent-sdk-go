package agentsdk

import (
	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/runner"
)

// Extended type aliases for advanced use cases such as multi-agent workflows,
// guardrails, state management, and raw model interactions.
type (
	// WorkflowState tracks the current phase and artifacts of a running
	// multi-phase workflow.
	WorkflowState = runner.WorkflowState

	// ValidationRule defines a predicate applied to data at handoff boundaries.
	ValidationRule = runner.ValidationRule

	// ValidationSeverity indicates whether a failed rule blocks progress or
	// only emits a warning.
	ValidationSeverity = runner.ValidationSeverity

	// StateManagementConfig configures workflow-state persistence and
	// checkpoint frequency.
	StateManagementConfig = runner.StateManagementConfig

	// ValidationConfig holds the set of rules applied before and after
	// handoffs.
	ValidationConfig = runner.ValidationConfig

	// RecoveryConfig controls automatic recovery from panics and transient
	// failures.
	RecoveryConfig = runner.RecoveryConfig

	// WorkflowStateStore is the interface for persisting and restoring
	// workflow state across checkpoints.
	WorkflowStateStore = runner.WorkflowStateStore

	// HandoffInputFilter transforms the input payload before it is forwarded
	// to the receiving agent during a handoff.
	HandoffInputFilter = runner.HandoffInputFilter

	// InputGuardrail validates agent input before each model call.
	InputGuardrail = runner.InputGuardrail

	// OutputGuardrail validates agent output after each model call.
	OutputGuardrail = runner.OutputGuardrail

	// ModelResponse is the raw, structured response returned by a model
	// provider after a non-streaming call.
	ModelResponse = model.Response

	// ModelRequest is the structured request sent to a model provider.
	ModelRequest = model.Request

	// Usage holds token-consumption data reported by a model provider.
	Usage = model.Usage

	// HandoffCall describes the parameters of an agent-to-agent handoff or
	// return-to-delegator event.
	HandoffCall = model.HandoffCall
)

// Validation severity constants.
const (
	// ValidationError is a blocking validation failure that halts the workflow.
	ValidationError = runner.ValidationError

	// ValidationWarning is a non-blocking validation failure logged but not
	// halting.
	ValidationWarning = runner.ValidationWarning
)

// ContentPartType constants for multi-modal messages.
const (
	// ContentPartTypeText marks a plain-text segment.
	ContentPartTypeText = model.ContentPartTypeText

	// ContentPartTypeDocument marks a document segment (PDF, plain-text file).
	ContentPartTypeDocument = model.ContentPartTypeDocument

	// ContentPartTypeImage marks an image segment (PNG, JPEG, GIF, WEBP).
	ContentPartTypeImage = model.ContentPartTypeImage
)

// Handoff type constants.
const (
	// HandoffTypeDelegate indicates the current agent is delegating a task to
	// another agent.
	HandoffTypeDelegate = model.HandoffTypeDelegate

	// HandoffTypeReturn indicates the current agent is returning a completed
	// task result to its delegator.
	HandoffTypeReturn = model.HandoffTypeReturn
)
