package assistant

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConversationSummary represents the current summarized chat state for one conversation.
type ConversationSummary struct {
	ID                      uuid.UUID
	ConversationID          uuid.UUID
	CurrentStateSummary     string
	LastSummarizedMessageID *uuid.UUID
	UpdatedAt               time.Time
}

// DefaultConversationStateSummary is used when no persisted summary exists.
const DefaultConversationStateSummary = "No current state."

// CompactionPolicy controls compaction thresholds.
type CompactionPolicy struct {
	TriggerTokenCount int
}

// CompactionDecision is the output of compaction policy evaluation.
type CompactionDecision struct {
	ShouldCompact bool
	Reason        ContextCompactionReason
	MessageCount  int
	TotalTokens   int
}

// CurrentStateOrDefault returns the persisted current state summary, or a default string when empty.
func (cs ConversationSummary) CurrentStateOrDefault() string {
	summary := strings.TrimSpace(cs.CurrentStateSummary)
	if summary == "" {
		return DefaultConversationStateSummary
	}
	return summary
}

// ConversationSummaryRepository defines the interface for storing and retrieving conversation summaries.
type ConversationSummaryRepository interface {
	// GetConversationSummary retrieves the current summary for the given conversation.
	GetConversationSummary(ctx context.Context, conversationID uuid.UUID) (ConversationSummary, bool, error)
	// StoreConversationSummary stores the summary for a conversation.
	StoreConversationSummary(ctx context.Context, summary ConversationSummary) error
	// DeleteConversationSummary deletes the summary for a conversation (used for testing).
	DeleteConversationSummary(ctx context.Context, conversationID uuid.UUID) error
}
