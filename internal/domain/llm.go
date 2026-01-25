package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LLMStreamEventType represents the type of event in an LLM stream
type LLMStreamEventType string

const (
	LLMStreamEventType_Meta  LLMStreamEventType = "meta"
	LLMStreamEventType_Delta LLMStreamEventType = "delta"
	LLMStreamEventType_Done  LLMStreamEventType = "done"
)

// LLMChatMessage represents a message in a chat request to the LLM API
type LLMChatMessage struct {
	Role    ChatRole
	Content string
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
}

// LLMUsage represents token usage information from the LLM
type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMStreamEventMeta contains metadata for a streaming chat session
type LLMStreamEventMeta struct {
	ConversationID     string
	UserMessageID      uuid.UUID
	AssistantMessageID uuid.UUID
	StartedAt          time.Time
}

// LLMStreamEventDelta contains a text delta from the stream
type LLMStreamEventDelta struct {
	Text string `json:"text"`
}

// LLMStreamEventDone contains completion metadata and token usage
type LLMStreamEventDone struct {
	AssistantMessageID string
	CompletedAt        string
	Usage              *LLMUsage
}

// LLMStreamEventCallback is called for each event in the stream
type LLMStreamEventCallback func(eventType LLMStreamEventType, data any) error

// LLMClient defines the interface for interacting with an LLM API
type LLMClient interface {
	// ChatStream streams assistant output as events from an LLM server
	// It calls onEvent with each event (meta, delta, done) and returns any error
	ChatStream(ctx context.Context, req LLMChatRequest, onEvent LLMStreamEventCallback) error

	// Chat sends a chat request to the LLM and returns the full assistant response
	Chat(ctx context.Context, req LLMChatRequest) (string, error)
}
