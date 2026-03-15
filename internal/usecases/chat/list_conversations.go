package chat

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// ListConversations defines the interface for the ListConversations use case
type ListConversations interface {
	// Query returns a paginated list of conversations for the user ordered by last message time descending.
	Query(ctx context.Context, page int, pageSize int) ([]assistant.Conversation, map[uuid.UUID]int64, bool, error)
}

// ListConversationsImpl is the implementation of the ListConversations use case
type ListConversationsImpl struct {
	conversationRepo assistant.ConversationRepository
}

// NewListConversationsImpl creates a new instance of ListConversationsImpl
func NewListConversationsImpl(conversationRepo assistant.ConversationRepository) *ListConversationsImpl {
	return &ListConversationsImpl{
		conversationRepo: conversationRepo,
	}
}

// Query returns a paginated list of conversations for the user ordered by last message time descending.
func (uc *ListConversationsImpl) Query(ctx context.Context, page int, pageSize int) ([]assistant.Conversation, map[uuid.UUID]int64, bool, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	conversations, hasMore, err := uc.conversationRepo.ListConversations(spanCtx, page, pageSize)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, nil, false, err
	}

	conversationIDs := make([]uuid.UUID, 0, len(conversations))
	for _, conversation := range conversations {
		conversationIDs = append(conversationIDs, conversation.ID)
	}

	usageByConversationID, err := uc.conversationRepo.GetConversationContextTokenUsage(spanCtx, conversationIDs)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, nil, false, err
	}

	return conversations, usageByConversationID, hasMore, nil
}
