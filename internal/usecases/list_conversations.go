package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

// ListConversations defines the interface for the ListConversations use case
type ListConversations interface {
	// Query returns a paginated list of conversations for the user ordered by last message time descending.
	Query(ctx context.Context, page int, pageSize int) ([]domain.Conversation, bool, error)
}

// ListConversationsImpl is the implementation of the ListConversations use case
type ListConversationsImpl struct {
	conversationRepo domain.ConversationRepository
}

// NewListConversationsImpl creates a new instance of ListConversationsImpl
func NewListConversationsImpl(conversationRepo domain.ConversationRepository) *ListConversationsImpl {
	return &ListConversationsImpl{
		conversationRepo: conversationRepo,
	}
}

// Query returns a paginated list of conversations for the user ordered by last message time descending.
func (uc *ListConversationsImpl) Query(ctx context.Context, page int, pageSize int) ([]domain.Conversation, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	conversations, hasMore, err := uc.conversationRepo.ListConversations(spanCtx, page, pageSize)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	return conversations, hasMore, nil
}

// InitListConversations initializes the ListConversations use case and registers it in the dependency container.
type InitListConversations struct {
	ConversationRepo domain.ConversationRepository `resolve:""`
}

// Initialize initializes the ListConversationsImpl use case.
func (init InitListConversations) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListConversations](NewListConversationsImpl(init.ConversationRepo))
	return ctx, nil
}
