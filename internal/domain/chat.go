package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

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
	ConversationID   uuid.UUID
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

// IsToolCall returns true if the chat message is a tool call, based on its role.
func (m ChatMessage) IsToolCallSuccess() bool {
	return m.ChatRole == ChatRole_Tool &&
		m.ToolCallID != nil &&
		m.MessageState == ChatMessageState_Completed
}

// ListChatMessagesParams defines optional filters for listing chat messages.
type ListChatMessagesParams struct {
	AfterMessageID *uuid.UUID
}

// ListChatMessagesOption configures optional filters for listing chat messages.
type ListChatMessagesOption func(*ListChatMessagesParams)

// WithChatMessagesAfterMessageID filters the query to return messages after a checkpoint message ID.
func WithChatMessagesAfterMessageID(messageID uuid.UUID) ListChatMessagesOption {
	return func(options *ListChatMessagesParams) {
		options.AfterMessageID = &messageID
	}
}

// ChatMessageRepository defines the interface for chat message persistence
type ChatMessageRepository interface {
	// CreateChatMessages persists chat messages for a conversation
	CreateChatMessages(ctx context.Context, messages []ChatMessage) error

	// ListChatMessages retrieves paginated chat messages for a conversation, with optional filters.
	ListChatMessages(ctx context.Context, conversationID uuid.UUID, page int, pageSize int, options ...ListChatMessagesOption) ([]ChatMessage, bool, error)

	// DeleteConversationMessages removes all messages for a conversation.
	DeleteConversationMessages(ctx context.Context, conversationID uuid.UUID) error
}
