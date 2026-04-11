package ai

// Role constants for AI message roles.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// ContentType describes the kind of content in a MessagePart.
const (
	ContentTypeText  = "text"
	ContentTypeImage = "image" // base64 encoded screenshot
)

// MessagePart is a single content block within a Message.
// Text messages have Type=ContentTypeText and Text set.
// Image messages have Type=ContentTypeImage and ImageBase64 set.
type MessagePart struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
	MimeType    string `json:"mime_type,omitempty"` // e.g. "image/png"
}

// Message is a single turn in the AI conversation history.
type Message struct {
	Role  string        `json:"role"`
	Parts []MessagePart `json:"parts"`
	// ToolCallID is set when Role == RoleTool, linking this result to its call.
	ToolCallID string `json:"tool_call_id,omitempty"`
	// ToolCalls is set when Role == RoleAssistant and the assistant requested tools.
	// Providers must serialize these back so the next turn is valid.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation requested by the AI.
type ToolCall struct {
	ID        string            `json:"id"`
	ToolName  string            `json:"tool_name"`
	Arguments map[string]string `json:"arguments"`
}

// ToolResult carries the output of an executed tool back to the AI.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Output     string `json:"output"`
	IsError    bool   `json:"is_error"`
}

// Response is what the AI provider returns after one round-trip.
type Response struct {
	// Text is the final text reply (set when the AI is done).
	Text string
	// ToolCalls is non-empty when the AI wants to invoke tools.
	ToolCalls []ToolCall
	// Done signals the AI has finished the task (no more tool calls).
	Done bool
}

// ToolDefinition describes a tool the AI may call.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  []ToolParameter
}

// ToolParameter describes a single parameter of a tool.
type ToolParameter struct {
	Name        string
	Type        string // "string" | "integer" | "number" | "boolean"
	Description string
	Required    bool
}
