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
)

// ChatMessage represents an AI chat message in a conversation
type ChatMessage struct {
	ID               uuid.UUID
	ConversationID   string
	ChatRole         ChatRole
	Content          string
	Model            string
	PromptTokens     int
	CompletionTokens int
	CreatedAt        time.Time
}

// ChatMessageRepository defines the interface for chat message persistence
type ChatMessageRepository interface {
	// CreateChatMessage persists a chat message for the global conversation
	CreateChatMessage(ctx context.Context, message ChatMessage) error

	// ListChatMessages retrieves messages for the global conversation ordered by creation time.
	// If limit is greater than 0, only the last N messages are returned.
	// Returns messages and a boolean indicating if there are more messages.
	ListChatMessages(ctx context.Context, limit int) ([]ChatMessage, bool, error)

	// DeleteConversation removes all messages for the global conversation
	DeleteConversation(ctx context.Context) error
}
