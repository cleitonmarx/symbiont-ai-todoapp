package chat

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// ListConversations defines the interface for the ListConversations use case
type ListConversations interface {
	// Query returns a paginated list of conversations for the user ordered by last message time descending.
	Query(ctx context.Context, page int, pageSize int) ([]assistant.Conversation, bool, error)
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
func (uc *ListConversationsImpl) Query(ctx context.Context, page int, pageSize int) ([]assistant.Conversation, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	conversations, hasMore, err := uc.conversationRepo.ListConversations(spanCtx, page, pageSize)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	return conversations, hasMore, nil
}
