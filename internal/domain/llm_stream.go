package domain

import "github.com/google/uuid"

// LLMStreamEventType represents the type of event in an LLM stream.
type LLMStreamEventType string

const (
	LLMStreamEventType_Meta          LLMStreamEventType = "meta"
	LLMStreamEventType_Delta         LLMStreamEventType = "delta"
	LLMStreamEventType_ToolCall      LLMStreamEventType = "tool_call"
	LLMStreamEventType_ToolStarted   LLMStreamEventType = "tool_call_started"
	LLMStreamEventType_ToolCompleted LLMStreamEventType = "tool_call_finished"
	LLMStreamEventType_Done          LLMStreamEventType = "done"
)

// LLMStreamEventMeta contains metadata for a streaming chat session.
type LLMStreamEventMeta struct {
	ConversationID      uuid.UUID `json:"conversation_id"`
	UserMessageID       uuid.UUID `json:"user_message_id"`
	AssistantMessageID  uuid.UUID `json:"assistant_message_id"`
	ConversationCreated bool      `json:"conversation_created"`
}

// LLMStreamEventDelta contains a text delta from the stream.
type LLMStreamEventDelta struct {
	Text string `json:"text"`
}

// LLMStreamEventToolCall contains a function call delta from the stream.
type LLMStreamEventToolCall struct {
	ID        string `json:"id"`
	Function  string `json:"function"`
	Arguments string `json:"arguments"`
	Text      string `json:"text"`
}

// LLMStreamEventToolCallCompleted indicates a tool invocation has finished.
type LLMStreamEventToolCallCompleted struct {
	ID            string  `json:"id"`
	Function      string  `json:"function"`
	Success       bool    `json:"success"`
	Error         *string `json:"error,omitempty"`
	ShouldRefetch bool    `json:"should_refetch"`
}

// LLMUsage contains token usage information.
type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMStreamEventDone contains completion metadata and token usage.
type LLMStreamEventDone struct {
	Usage              LLMUsage `json:"usage"`
	AssistantMessageID string   `json:"assistant_message_id"`
	CompletedAt        string   `json:"completed_at"`
}

// LLMStreamEventCallback is called for each event in the stream.
type LLMStreamEventCallback func(eventType LLMStreamEventType, data any) error
