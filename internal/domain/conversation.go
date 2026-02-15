package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ConversationTitleSource string

const (
	ConversationTitleSource_User ConversationTitleSource = "user"
	ConversationTitleSource_LLM  ConversationTitleSource = "llm"
	ConversationTitleSource_Auto ConversationTitleSource = "auto"
)

// Conversation represents a chat conversation, which can have multiple messages and a title.
type Conversation struct {
	ID            uuid.UUID
	Title         string
	TitleSource   ConversationTitleSource
	LastMessageAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Validate checks if the conversation has valid data.
func (c Conversation) Validate() error {
	if c.Title == "" {
		return NewValidationErr("conversation title cannot be empty")
	}
	if c.TitleSource != ConversationTitleSource_User &&
		c.TitleSource != ConversationTitleSource_LLM &&
		c.TitleSource != ConversationTitleSource_Auto {
		return NewValidationErr(fmt.Sprintf("invalid conversation title source: %s", c.TitleSource))
	}
	return nil
}

// ConversationRepository defines the interface for managing conversations.
type ConversationRepository interface {
	// CreateConversation creates a new conversation with the given title and returns it.
	CreateConversation(context.Context, string, ConversationTitleSource) (Conversation, error)
	// GetConversation returns the conversation with the given ID, a boolean indicating if it was found, and an error if any.
	GetConversation(context.Context, uuid.UUID) (Conversation, bool, error)
	// UpdateConversation updates the conversation with the given ID.
	UpdateConversation(context.Context, Conversation) error
	// ListConversations returns a list of conversations with pagination support ordered by last message time descending.
	ListConversations(ctx context.Context, page int, pageSize int) ([]Conversation, bool, error)
	// DeleteConversation deletes the conversation with the given ID.
	DeleteConversation(context.Context, uuid.UUID) error
}

// GenerateAutoConversationTitle generates a conversation title based on the user's initial message.
func GenerateAutoConversationTitle(userMessage string) string {
	// Simple heuristic: use the first 5 words of the user's message as the title, or "New Conversation" if empty.
	words := strings.Fields(userMessage)
	if len(words) == 0 {
		return "New Conversation"
	}
	if len(words) <= 5 {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:5], " ") + "..."
}
