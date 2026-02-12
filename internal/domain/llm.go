package domain

import (
	"context"

	"github.com/google/uuid"
)

// LLMStreamEventType represents the type of event in an LLM stream
type LLMStreamEventType string

const (
	LLMStreamEventType_Meta     LLMStreamEventType = "meta"
	LLMStreamEventType_Delta    LLMStreamEventType = "delta"
	LLMStreamEventType_ToolCall LLMStreamEventType = "tool_call"
	LLMStreamEventType_Done     LLMStreamEventType = "done"
)

// LLMTool represents a tool that can be executed by the LLM
type LLMTool interface {
	// Tool returns the LLMTool definition
	Definition() LLMToolDefinition
	// StatusMessage returns a user-friendly status line for this tool
	StatusMessage() string
	// Call executes the tool with the given tool call and chat messages
	Call(context.Context, LLMStreamEventToolCall, []LLMChatMessage) LLMChatMessage
}

// LLMToolRegistry defines the interface for calling registered LLM tools.
type LLMToolRegistry interface {
	// Call executes the tool with the given tool call and chat messages
	Call(context.Context, LLMStreamEventToolCall, []LLMChatMessage) LLMChatMessage
	// StatusMessage returns a friendly status message for the given tool name.
	StatusMessage(toolName string) string
	// List returns all registered LLM tools.
	List() []LLMToolDefinition
}

// LLMChatMessage represents a message in a chat request to the LLM API
type LLMChatMessage struct {
	Role       ChatRole
	Content    string
	ToolCallID *string
	ToolCalls  []LLMStreamEventToolCall
}

// LLMChatRequest represents a request to the LLM API
type LLMChatRequest struct {
	Model    string
	Messages []LLMChatMessage
	Stream   bool
	// Optional parameters
	Temperature *float64
	TopP        *float64
	MaxTokens   *int
	Tools       []LLMToolDefinition
}

// LLMToolDefinition represents a tool that can be used by the LLM
type LLMToolDefinition struct {
	Type     string
	Function LLMToolFunction
}

// LLMToolFunction represents a function tool for the LLM
type LLMToolFunction struct {
	Description string
	Name        string
	Parameters  LLMToolFunctionParameters
}

// LLMToolFunctionParameters represents the parameters schema for a function tool
type LLMToolFunctionParameters struct {
	Type       string
	Properties map[string]LLMToolFunctionParameterDetail
}

// LLMToolFunctionParameterDetail represents a single parameter in the function tool schema
type LLMToolFunctionParameterDetail struct {
	Type        string
	Description string
	Required    bool
}

// LLMStreamEventMeta contains metadata for a streaming chat session
type LLMStreamEventMeta struct {
	UserMessageID      uuid.UUID
	AssistantMessageID uuid.UUID
}

// LLMStreamEventDelta contains a text delta from the stream
type LLMStreamEventDelta struct {
	Text string
}

// LLMStreamEventToolCall contains a function call delta from the stream
type LLMStreamEventToolCall struct {
	ID        string
	Function  string
	Arguments string
	Text      string
}

// LLMStreamEventDone contains completion metadata and token usage
type LLMStreamEventDone struct {
	Usage              LLMUsage
	AssistantMessageID string
	CompletedAt        string
}

// LLMUsage contains token usage information
type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMChatResponse represents the response from a chat request to the LLM API
type LLMChatResponse struct {
	Content string
	Usage   LLMUsage
}

// EmbedResponse represents the response from an embedding request to the LLM API
type EmbedResponse struct {
	Embedding   []float64
	TotalTokens int
}

type LLMModelType string

const (
	LLMModelType_Chat      LLMModelType = "chat"
	LLMModelType_Embedding LLMModelType = "embedding"
)

// LLMModelInfo represents information about an available LLM model
type LLMModelInfo struct {
	Name string
	Type LLMModelType
}

// LLMStreamEventCallback is called for each event in the stream
type LLMStreamEventCallback func(eventType LLMStreamEventType, data any) error

// LLMClient defines the interface for interacting with an LLM API
type LLMClient interface {
	// ChatStream streams assistant output as events from an LLM server
	// It calls onEvent with each event (meta, delta, done) and returns any error
	ChatStream(ctx context.Context, req LLMChatRequest, onEvent LLMStreamEventCallback) error

	// Chat sends a chat request to the LLM and returns the full assistant response
	Chat(ctx context.Context, req LLMChatRequest) (LLMChatResponse, error)

	// Embed generates an embedding vector for the given input text
	Embed(ctx context.Context, model, input string) (EmbedResponse, error)

	// AvailableModels retrieves the list of available models
	AvailableModels(ctx context.Context) ([]LLMModelInfo, error)
}
