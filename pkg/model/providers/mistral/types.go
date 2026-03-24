package mistral

// Mistral API types, constants, and defaults.
// These replace the abandoned github.com/gage-technologies/mistral-go SDK.

// Default Mistral API endpoint.
const defaultEndpoint = "https://api.mistral.ai"

// Message roles.
const (
	roleUser      = "user"
	roleAssistant = "assistant"
	roleSystem    = "system"
	roleTool      = "tool"
)

// Tool choice values.
const (
	toolChoiceAny  = "any"
	toolChoiceAuto = "auto"
	toolChoiceNone = "none"
)

// toolType identifies the kind of tool.
type toolType string

const toolTypeFunction toolType = "function"

// chatRequestParams holds the parameters for a Mistral chat completion request.
type chatRequestParams struct {
	Temperature    float64 `json:"temperature"`
	TopP           float64 `json:"top_p"`
	RandomSeed     *int    `json:"random_seed,omitempty"`
	MaxTokens      int     `json:"max_tokens"`
	SafePrompt     bool    `json:"safe_prompt"`
	Tools          []tool  `json:"tools"`
	ToolChoice     string  `json:"tool_choice"`
	ResponseFormat string  `json:"response_format"`
}

// defaultChatRequestParams provides sensible defaults matching the Mistral API.
// RandomSeed is intentionally omitted so that the API uses its own default
// (non-deterministic) behaviour. Set it explicitly in request.Settings if
// reproducible outputs are required.
var defaultChatRequestParams = chatRequestParams{
	Temperature: 1,
	TopP:        1,
	MaxTokens:   4000,
	SafePrompt:  false,
}

// tool defines a tool that the LLM can invoke.
type tool struct {
	Type     toolType     `json:"type"`
	Function toolFunction `json:"function"`
}

// toolFunction describes a callable function including its parameters schema.
type toolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// toolFunctionCall represents the LLM's request to call a specific function.
type toolFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// toolCall represents a single tool invocation returned by the LLM.
type toolCall struct {
	ID       string           `json:"id"`
	Type     toolType         `json:"type"`
	Function toolFunctionCall `json:"function"`
}
