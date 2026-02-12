package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const GlobalConversationID = "global"

// ChatRole represents the role of a chat message
type ChatRole string

const (
	ChatRole_User      ChatRole = "user"
	ChatRole_Assistant ChatRole = "assistant"
	ChatRole_System    ChatRole = "system"
	ChatRole_Developer ChatRole = "developer"
	ChatRole_Tool      ChatRole = "tool"
)

// ChatMessageState represents the persistence state of a chat message.
type ChatMessageState string

const (
	// ChatMessageState_Completed indicates the message was fully generated and persisted.
	ChatMessageState_Completed ChatMessageState = "COMPLETED"
	// ChatMessageState_Failed indicates message generation failed.
	ChatMessageState_Failed ChatMessageState = "FAILED"
)

// ChatMessage represents an AI chat message in a conversation
type ChatMessage struct {
	ID               uuid.UUID
	ConversationID   string
	TurnID           uuid.UUID
	TurnSequence     int64
	ChatRole         ChatRole
	Content          string
	ToolCallID       *string
	ToolCalls        []LLMStreamEventToolCall
	Model            string
	MessageState     ChatMessageState
	ErrorMessage     *string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ListChatMessagesOptions defines optional filters for listing chat messages.
type ListChatMessagesOptions struct {
	ConversationID string
	AfterMessageID *uuid.UUID
}

// ListChatMessagesOption configures optional filters for listing chat messages.
type ListChatMessagesOption func(*ListChatMessagesOptions)

// ListChatMessagesByConversationID filters the query by conversation ID.
func ListChatMessagesByConversationID(conversationID string) ListChatMessagesOption {
	return func(options *ListChatMessagesOptions) {
		options.ConversationID = conversationID
	}
}

// ListChatMessagesAfterMessageID filters the query to return messages after a checkpoint message ID.
func ListChatMessagesAfterMessageID(messageID uuid.UUID) ListChatMessagesOption {
	return func(options *ListChatMessagesOptions) {
		options.AfterMessageID = &messageID
	}
}

// ChatMessageRepository defines the interface for chat message persistence
type ChatMessageRepository interface {
	// CreateChatMessages persists chat messages for the global conversation
	CreateChatMessages(ctx context.Context, messages []ChatMessage) error

	// ListChatMessages retrieves messages for the global conversation ordered by creation time.
	// If limit is greater than 0, only the last N messages are returned.
	// Returns messages and a boolean indicating if there are more messages.
	ListChatMessages(ctx context.Context, limit int, options ...ListChatMessagesOption) ([]ChatMessage, bool, error)

	// DeleteConversation removes all messages for the global conversation
	DeleteConversation(ctx context.Context) error
}
