package domain

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

const DefaultConversationStateSummary = "No current state."

// ConversationSummaryGenerationReason describes why a summary generation run should happen.
type ConversationSummaryGenerationReason string

const (
	// ConversationSummaryGenerationReason_None indicates no summary should be generated.
	ConversationSummaryGenerationReason_None ConversationSummaryGenerationReason = "none"
	// ConversationSummaryGenerationReason_StateChangingToolSuccess indicates summary generation was triggered
	// by a successful state-changing tool call.
	ConversationSummaryGenerationReason_StateChangingToolSuccess ConversationSummaryGenerationReason = "state_changing_tool_success"
	// ConversationSummaryGenerationReason_MessageCountThreshold indicates summary generation was triggered
	// by accumulated unsummarized message count.
	ConversationSummaryGenerationReason_MessageCountThreshold ConversationSummaryGenerationReason = "message_count_threshold"
	// ConversationSummaryGenerationReason_TokenCountThreshold indicates summary generation was triggered
	// by accumulated unsummarized token count.
	ConversationSummaryGenerationReason_TokenCountThreshold ConversationSummaryGenerationReason = "token_count_threshold"
)

// ConversationSummaryGenerationPolicy controls summary generation thresholds.
type ConversationSummaryGenerationPolicy struct {
	TriggerMessageCount int
	TriggerTokenCount   int
}

// ConversationSummaryGenerationDecision is the output of summary generation policy evaluation.
type ConversationSummaryGenerationDecision struct {
	ShouldGenerate bool
	Reason         ConversationSummaryGenerationReason
	MessageCount   int
	TotalTokens    int
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
