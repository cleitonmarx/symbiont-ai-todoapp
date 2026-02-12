package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ConversationSummary represents the current summarized chat state for one conversation.
type ConversationSummary struct {
	ID                      uuid.UUID
	ConversationID          string
	CurrentStateSummary     string
	LastSummarizedMessageID *uuid.UUID
	UpdatedAt               time.Time
}

// ConversationSummaryRepository defines the interface for storing and retrieving conversation summaries.
type ConversationSummaryRepository interface {
	// GetConversationSummary retrieves the current summary for the given conversation.
	GetConversationSummary(ctx context.Context, conversationID string) (ConversationSummary, bool, error)
	// StoreConversationSummary stores the summary for a conversation.
	StoreConversationSummary(ctx context.Context, summary ConversationSummary) error
	// DeleteConversationSummary deletes the summary for a conversation (used for testing).
	DeleteConversationSummary(ctx context.Context, conversationID string) error
}
